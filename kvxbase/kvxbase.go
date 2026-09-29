// Package kvxbase defines core types shared by all KV drivers.
// Kept in a separate package to avoid import cycles between the root kvx package
// and its subpackage drivers.
package kvxbase

import "context"
import "time"

// KVStore is the unified key-value store interface.
type KVStore interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Keys(ctx context.Context, pattern string) ([]string, error)
	MGet(ctx context.Context, keys ...string) (map[string]string, error)
	MSet(ctx context.Context, pairs map[string]string) error
	TTL(ctx context.Context, key string) (time.Duration, error)
	Incr(ctx context.Context, key string) (int64, error)
	Decr(ctx context.Context, key string) (int64, error)
	HealthCheck() error
	Close() error
}

// Config is the configuration structure for a KV store.
//
// tag 说明：三套 tag 齐备，分别对应 configx 的三条读取通道——
// yaml 本地文件 / json 远程配置中心 / mapstructure 本地 viper 通道。
// 多词字段（pool_size、min_idle_conn、tls_enabled、enable_trace …）
// 缺 mapstructure tag 时经 viper 通道解码会静默为零值。
type Config struct {
	Name        string        `json:"name" yaml:"name" mapstructure:"name"`
	Driver      string        `json:"driver" yaml:"driver" mapstructure:"driver"`
	Addr        string        `json:"addr" yaml:"addr" mapstructure:"addr"`
	Addrs       []string      `json:"addrs" yaml:"addrs" mapstructure:"addrs"`
	Password    string        `json:"password" yaml:"password" mapstructure:"password"`
	DB          int           `json:"db" yaml:"db" mapstructure:"db"`
	PoolSize    int           `json:"pool_size" yaml:"pool_size" mapstructure:"pool_size"`
	MinIdleConn int           `json:"min_idle_conn" yaml:"min_idle_conn" mapstructure:"min_idle_conn"`
	Timeout     time.Duration `json:"timeout" yaml:"timeout" mapstructure:"timeout"`
	TLSEnabled  bool          `json:"tls_enabled" yaml:"tls.enabled" mapstructure:"tls_enabled"`
	Cluster     string        `json:"cluster" yaml:"cluster" mapstructure:"cluster"`
	// EnableTrace 统一调用链开关（configx 配置键 enable_trace）
	EnableTrace bool `json:"enable_trace" yaml:"enable_trace" mapstructure:"enable_trace"`
}

// Stores 多存储注册表
type Stores map[string]KVStore

// FillDefaults sets default values on Config.
func (c *Config) FillDefaults() {
	if c.Name == "" {
		c.Name = "default"
	}
	if c.Driver == "" {
		c.Driver = "redis"
	}
	if c.Addr == "" && len(c.Addrs) == 0 {
		c.Addr = "127.0.0.1:6379"
	}
	if c.Timeout == 0 {
		c.Timeout = 3 * time.Second
	}
	if c.PoolSize == 0 {
		c.PoolSize = 10
	}
	if c.MinIdleConn == 0 {
		c.MinIdleConn = 2
	}
}
