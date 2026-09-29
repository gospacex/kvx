// Package otel provides OpenTelemetry initialization for the kvx SDK.
//
// 使用方式：
//   - 在 main() 最开头调用 InitOTel，注册 TracerProvider
//   - 关闭程序时调用 shutdown()
//   - 驱动通过 configx enable_trace: true 自动开启链路上报
//
// 支持后端：Jaeger（HTTP OTLP）、Kafka MQ、stdout（调试）
package otel

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/gospacex/configx"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// ---- 全局状态（单例 + 懒加载）----

var (
	setupOnce     sync.Once
	shutdownFns   []func(context.Context) error
	globalTp      *sdktrace.TracerProvider
	globalStarted bool
)

// Config 封装 OTel 初始化参数
type Config struct {
	ServiceName string
	Backend     Backend // "jaeger" | "mq" | "stdout" | "all"
	Endpoint    string  // Jaeger HTTP 端点
	KafkaBroker string  // Kafka broker 地址（逗号分隔）
	KafkaTopic  string  // Kafka topic
	SamplerRate float64 // 采样率 0.0~1.0，默认 1.0
	Debug       bool    // 调试模式，强制 stdout 输出
}

// Backend 表示追踪后端类型
type Backend string

const (
	BackendJaeger Backend = "jaeger"
	BackendMQ     Backend = "mq"
	BackendStdout Backend = "stdout"
	BackendAll    Backend = "all"
)

// defaultConfig 返回默认配置
func defaultConfig() *Config {
	return &Config{
		ServiceName: "kvx-service",
		Backend:     BackendJaeger,
		Endpoint:    "http://localhost:14268/api/traces",
		KafkaBroker: "localhost:9092",
		KafkaTopic:  "kvx-spans",
		SamplerRate: 1.0,
		Debug:       false,
	}
}

// ---- 公共初始化入口 ----

// InitOTel 全局单次初始化，返回 shutdown 函数供 defer 调用
//
// 示例：
//
//	func main() {
//	    shutdown := otel.InitOTel(
//	        otel.WithServiceName("my-app"),
//	        otel.WithBackend(otel.BackendJaeger),
//	    )
//	    defer shutdown(context.Background())
//	    // ... kvx.New(...) 等业务代码
//	}
func InitOTel(opts ...Option) func(context.Context) error {
	setupOnce.Do(func() {
		cfg := defaultConfig()
		for _, fn := range opts {
			fn(cfg)
		}

		if err := initTracerProvider(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "otel: init failed: %v\n", err)
			return
		}
		globalStarted = true
		registerConfigxHook()
	})
	return func(ctx context.Context) error {
		var lastErr error
		for _, fn := range shutdownFns {
			if err := fn(ctx); err != nil {
				lastErr = err
			}
		}
		return lastErr
	}
}

// GetTracerProvider 获取已初始化的全局 TracerProvider（线程安全）
func GetTracerProvider() *sdktrace.TracerProvider {
	return globalTp
}

// Started 返回 OTel 是否已完成初始化
func Started() bool {
	return globalStarted
}

// Option 初始化选项
type Option func(*Config)

// WithServiceName 设置服务名
func WithServiceName(name string) Option {
	return func(c *Config) { c.ServiceName = name }
}

// WithBackend 设置后端（jaeger / mq / stdout / all）
func WithBackend(b Backend) Option {
	return func(c *Config) { c.Backend = b }
}

// WithEndpoint 设置 Jaeger HTTP 端点
func WithEndpoint(ep string) Option {
	return func(c *Config) { c.Endpoint = ep }
}

// WithKafka 设置 Kafka broker 和 topic
func WithKafka(broker, topic string) Option {
	return func(c *Config) { c.KafkaBroker = broker; c.KafkaTopic = topic }
}

// WithSamplerRate 设置采样率（0.0 ~ 1.0）
func WithSamplerRate(rate float64) Option {
	return func(c *Config) { c.SamplerRate = rate }
}

// WithDebug 开启调试模式（强制 stdout 输出所有 span）
func WithDebug(on bool) Option {
	return func(c *Config) { c.Debug = on }
}

// ---- 内部实现 ----

func initTracerProvider(cfg *Config) error {
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.DeploymentEnvironment(os.Getenv("ENV")),
		),
	)
	if err != nil {
		return fmt.Errorf("otel: resource: %w", err)
	}

	backends := resolveBackends(cfg)
	processors := make([]sdktrace.SpanProcessor, 0, len(backends))

	for _, b := range backends {
		switch b {
		case BackendJaeger:
			exp, err := jaeger.New(jaeger.WithCollectorEndpoint(
				jaeger.WithEndpoint(cfg.Endpoint),
			))
			if err != nil {
				return fmt.Errorf("otel: jaeger exporter: %w", err)
			}
			processors = append(processors, sdktrace.NewBatchSpanProcessor(exp))
			shutdownFns = append(shutdownFns, exp.Shutdown)

		case BackendMQ:
			exp, err := newMQExporter(cfg.KafkaBroker, cfg.KafkaTopic)
			if err != nil {
				return fmt.Errorf("otel: mq exporter: %w", err)
			}
			processors = append(processors, sdktrace.NewBatchSpanProcessor(exp))
			shutdownFns = append(shutdownFns, exp.Shutdown)

		case BackendStdout:
			exp, err := stdouttrace.New()
			if err != nil {
				return fmt.Errorf("otel: stdout exporter: %w", err)
			}
			processors = append(processors, sdktrace.NewBatchSpanProcessor(exp))
			shutdownFns = append(shutdownFns, exp.Shutdown)
		}
	}

	sampler := sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SamplerRate))
	opts := []sdktrace.TracerProviderOption{
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	}
	for _, sp := range processors {
		opts = append(opts, sdktrace.WithSpanProcessor(sp))
	}
	globalTp = sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(globalTp)
	return nil
}

// resolveBackends 根据配置展开后端列表
func resolveBackends(cfg *Config) []Backend {
	if cfg.Debug {
		return []Backend{BackendStdout}
	}
	switch cfg.Backend {
	case BackendAll:
		return []Backend{BackendJaeger, BackendMQ}
	case BackendMQ:
		return []Backend{BackendMQ}
	case BackendStdout:
		return []Backend{BackendStdout}
	default:
		return []Backend{BackendJaeger}
	}
}

// registerConfigxHook 从 configx 读取 kvx.enable_trace 配置并写入日志
func registerConfigxHook() {
	c, err := configx.ReadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "otel: configx read failed: %v\n", err)
		return
	}
	var traceEnabled bool
	if err := c.Unmarshal("kvx.enable_trace", &traceEnabled); err != nil {
		return
	}
	if !traceEnabled {
		fmt.Printf("otel: kvx.enable_trace=false, tracing disabled\n")
		return
	}
	fmt.Printf("otel: kvx.enable_trace=true, tracing active\n")
}
