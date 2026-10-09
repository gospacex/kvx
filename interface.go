// kvx/interface.go
// kvx：统一 KV 存储接口，支持多种后端实现
package kvx

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/gospacex/configx"
	"github.com/gospacex/kvx/kvxbase"
	_ "github.com/gospacex/logx/zap"
	"github.com/gospacex/traceflag"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/yaml.v3"
)

// Type aliases to kvxbase — root package re-exports for convenience.
type KVStore = kvxbase.KVStore
type Stores = kvxbase.Stores
type Config = kvxbase.Config

// ======================= KVStore 接口 =======================
// (re-exported via type alias above; kept for documentation clarity)

// ======================= Config 配置 =======================
// (re-exported via type alias above)

// ======================= New 工厂函数（switch-based） =======================
func New(cfgs []*Config) (Stores, error) {
	stores := make(Stores)
	for _, cfg := range cfgs {
		cfg.FillDefaults()
		// 接入 hubx 统一调用链开关：组件自身配置（enable_trace）为主开关，
		// 全局总闸 traceflag.SetEnabled(false) 可一键熔断。放在 FillDefaults
		// 之后，保证默认值已就位再参与判定。
		cfg.EnableTrace = traceflag.Apply("kvx", cfg.EnableTrace)
		store, err := newByDriver(cfg)
		if err != nil {
			return nil, fmt.Errorf("kvx: failed to create store %q (driver=%s): %w", cfg.Name, cfg.Driver, err)
		}
		if cfg.EnableTrace {
			store = NewTraceStore(cfg.Name, store)
		}
		stores[cfg.Name] = store
	}
	return stores, nil
}

// ======================= 驱动注册表 =======================
// 核心包不直接 import 任何驱动；各驱动在自己的包内通过 init() 调用
// RegisterCreator 注册（使用方 blank import 对应驱动包即可启用）。

var (
	registryMu sync.RWMutex
	creators   = map[string]func(*Config) (KVStore, error){}
)

// RegisterCreator 注册 KV 驱动创建函数
func RegisterCreator(name string, creator func(*Config) (KVStore, error)) {
	registryMu.Lock()
	defer registryMu.Unlock()
	creators[name] = creator
}

// newByDriver 按注册表创建驱动实例
func newByDriver(cfg *Config) (KVStore, error) {
	registryMu.RLock()
	creator, ok := creators[cfg.Driver]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("kvx: unsupported driver %q (register via driver package, e.g. _ \"github.com/gospacex/kvx/redis\")", cfg.Driver)
	}
	return creator(cfg)
}

// ======================= ConfigX 集成 =======================

// KVSection 对应 configx.yaml 中的 kvx 配置节
type KVSection struct {
	// mapstructure 标签不可省：configx 的 Unmarshal 走 viper → mapstructure，
	// 只用 yaml 标签时 mapstructure 无法把 "stores" 匹配到字段 KVStores，
	// 结果是静默零值 —— 表现为 ReadAndCreate 恒报 "no stores configured"。
	KVStores map[string]*Config `yaml:"stores" json:"stores" mapstructure:"stores"`
}

// ReadConfig 加载配置（兼容老接口）
func ReadConfig(opts ...configx.ConfigOptions) error {
	_, err := configx.ReadConfig(opts...)
	return err
}

// ReadAndCreate 从 configx 读取 kvx 配置并创建所有存储实例
func ReadAndCreate(opts ...configx.ConfigOptions) (Stores, error) {
	c, err := configx.ReadConfig(opts...)
	if err != nil {
		return nil, fmt.Errorf("kvx: failed to read config: %w", err)
	}

	section := &KVSection{}
	if err := c.Unmarshal("kvx", section); err != nil {
		return nil, fmt.Errorf("kvx: failed to unmarshal kvx section: %w", err)
	}

	if section.KVStores == nil {
		return nil, fmt.Errorf("kvx: no stores configured")
	}

	cfgs := make([]*Config, 0, len(section.KVStores))
	for name, cfg := range section.KVStores {
		cfg.Name = name
		cfgs = append(cfgs, cfg)
	}

	return New(cfgs)
}

// MustReadAndCreate 类似 ReadAndCreate，失败时 panic
func MustReadAndCreate(opts ...configx.ConfigOptions) Stores {
	stores, err := ReadAndCreate(opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kvx: failed to initialize stores: %v\n", err)
		os.Exit(1)
	}
	return stores
}

// ======================= YAML 配置辅助 =======================

// ConfigFromYAML 从 YAML 字节解析 Config 列表
func ConfigFromYAML(data []byte) ([]*Config, error) {
	var section KVSection
	if err := yaml.Unmarshal(data, &section); err != nil {
		return nil, fmt.Errorf("kvx: failed to parse yaml: %w", err)
	}
	cfgs := make([]*Config, 0, len(section.KVStores))
	for name, cfg := range section.KVStores {
		cfg.Name = name
		cfgs = append(cfgs, cfg)
	}
	return cfgs, nil
}

// ConfigFromYAMLFile 从文件路径加载并解析
func ConfigFromYAMLFile(path string) ([]*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("kvx: failed to read config file %q: %w", path, err)
	}
	return ConfigFromYAML(data)
}

// ======================= OTel Trace 工具 =======================

// traceStore 包装 KVStore 并注入 OTel 属性
type traceStore struct {
	name   string
	store  KVStore
	tracer *traceflag.Tracer
}

// NewTraceStore 创建带追踪的 Store
//
// tracer 走 traceflag 统一开关：包装器本身只挂接在 EnableTrace 为 true 的
// 实例上（实例级闸门），component 为 "kvx" —— 组件级 SetComponent 与全局
// 总闸 traceflag.SetEnabled(false) 均可热熔断。关闭时 Start 返回 noop span
// 且不修改 ctx。
func NewTraceStore(name string, store KVStore) KVStore {
	return &traceStore{
		name:   name,
		store:  store,
		tracer: traceflag.NewTracer(true, "github.com/gospacex/kvx", "kvx"),
	}
}

func (t *traceStore) setAttrs(ctx context.Context, op, key string) {
	span := trace.SpanFromContext(ctx)
	if span.IsRecording() {
		attrs := []attribute.KeyValue{
			attribute.String("db.store", t.name),
			attribute.String("db.operation", op),
		}
		if key != "" {
			attrs = append(attrs, attribute.String("db.statement", key))
		}
		span.SetAttributes(attrs...)
	}
}

func (t *traceStore) startSpan(ctx context.Context, op string) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, "db.kvx."+op)
}

// Get 带追踪的 Get
func (t *traceStore) Get(ctx context.Context, key string) (string, error) {
	t.setAttrs(ctx, "GET", key)
	ctx, span := t.startSpan(ctx, "GET")
	defer span.End()
	return t.store.Get(ctx, key)
}

// Set 带追踪的 Set
func (t *traceStore) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	t.setAttrs(ctx, "SET", key)
	ctx, span := t.startSpan(ctx, "SET")
	defer span.End()
	return t.store.Set(ctx, key, value, ttl)
}

// Delete 带追踪的 Delete
func (t *traceStore) Delete(ctx context.Context, key string) error {
	t.setAttrs(ctx, "DELETE", key)
	ctx, span := t.startSpan(ctx, "DELETE")
	defer span.End()
	return t.store.Delete(ctx, key)
}

// Exists 带追踪的 Exists
func (t *traceStore) Exists(ctx context.Context, key string) (bool, error) {
	t.setAttrs(ctx, "EXISTS", key)
	ctx, span := t.startSpan(ctx, "EXISTS")
	defer span.End()
	return t.store.Exists(ctx, key)
}

// Keys 带追踪的 Keys
func (t *traceStore) Keys(ctx context.Context, pattern string) ([]string, error) {
	t.setAttrs(ctx, "KEYS", "")
	ctx, span := t.startSpan(ctx, "KEYS")
	defer span.End()
	return t.store.Keys(ctx, pattern)
}

// MGet 带追踪的 MGet
func (t *traceStore) MGet(ctx context.Context, keys ...string) (map[string]string, error) {
	t.setAttrs(ctx, "MGet", "")
	ctx, span := t.startSpan(ctx, "MGet")
	defer span.End()
	return t.store.MGet(ctx, keys...)
}

// MSet 带追踪的 MSet
func (t *traceStore) MSet(ctx context.Context, pairs map[string]string) error {
	ctx, span := t.startSpan(ctx, "MSet")
	defer span.End()
	return t.store.MSet(ctx, pairs)
}

// TTL 带追踪的 TTL
func (t *traceStore) TTL(ctx context.Context, key string) (time.Duration, error) {
	t.setAttrs(ctx, "TTL", key)
	ctx, span := t.startSpan(ctx, "TTL")
	defer span.End()
	return t.store.TTL(ctx, key)
}

// Incr 带追踪的 Incr
func (t *traceStore) Incr(ctx context.Context, key string) (int64, error) {
	t.setAttrs(ctx, "INCR", key)
	ctx, span := t.startSpan(ctx, "INCR")
	defer span.End()
	return t.store.Incr(ctx, key)
}

// Decr 带追踪的 Decr
func (t *traceStore) Decr(ctx context.Context, key string) (int64, error) {
	t.setAttrs(ctx, "DECR", key)
	ctx, span := t.startSpan(ctx, "DECR")
	defer span.End()
	return t.store.Decr(ctx, key)
}

// HealthCheck 委托给底层 store
func (t *traceStore) HealthCheck() error {
	return t.store.HealthCheck()
}

// Close 委托给底层 store
func (t *traceStore) Close() error {
	return t.store.Close()
}
