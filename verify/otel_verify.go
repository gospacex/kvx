// Package verify 提供 OTel 链路追踪集成验证工具
package verify

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gospacex/kvx"
	kotel "github.com/gospacex/kvx/otel"
	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// Run 执行完整的 OTel 集成验证，返回错误或 nil
func Run() error {
	logger := logx.GetGlobal()

	// 1. 初始化 OTel（stdout 双通道，debug 模式确保 span 可见）
	shutdown := kotel.InitOTel(
		kotel.WithServiceName("kvx-verify"),
		kotel.WithBackend(kotel.BackendStdout),
		kotel.WithDebug(true),
	)
	defer shutdown(context.Background())
	logger.Info("otel: tracer provider initialized")

	// 2. 创建 KV Store（Redis 为例，确保 docker-compose 中 redis 服务已启动）
	stores, err := kvx.New([]*kvx.Config{{
		Name:        "default",
		Driver:      "redis",
		Addr:        "127.0.0.1:6379",
		Password:    "redis123456",
		PoolSize:    5,
		Timeout:     3 * time.Second,
		EnableTrace: true,
	}})
	if err != nil {
		return fmt.Errorf("kvx init failed: %w", err)
	}
	defer stores["default"].Close()
	logger.Info("kvx store initialized")

	// 3. 在业务 span 上下文中执行 KV 操作，验证链路完整性
	tracer := otel.Tracer("github.com/gospacex/kvx/verify")
	ctx, span := tracer.Start(context.Background(), "verify-kv-operation")
	defer span.End()

	span.SetAttributes(
		attribute.String("verify.step", "initial"),
		attribute.Int("verify.user_id", 1000),
	)

	// 4. Set 操作（自动携带 span 上下文传播）
	key := "verify:user:1000"
	if err := stores["default"].Set(ctx, key, `{"name":"alice"}`, 60*time.Second); err != nil {
		return fmt.Errorf("set failed: %w", err)
	}
	span.SetAttributes(attribute.String("verify.action", "set"), attribute.String("verify.key", key))

	// 5. Get 操作（应携带同一 trace）
	val, err := stores["default"].Get(ctx, key)
	if err != nil {
		return fmt.Errorf("get failed: %w", err)
	}
	span.SetAttributes(attribute.String("verify.action", "get"), attribute.String("verify.value", val))

	// 6. 检查 span 上下文是否有效
	if !span.SpanContext().IsValid() {
		return fmt.Errorf("span context is invalid — tracing is not working")
	}

	fmt.Printf("✅ OTel verification passed\n")
	fmt.Printf("   TraceID : %s\n", span.SpanContext().TraceID())
	fmt.Printf("   SpanID  : %s\n", span.SpanContext().SpanID())
	fmt.Printf("   Operation: Set → Get\n")
	fmt.Printf("   Result   : value=%s\n", val)

	// 7. 清理测试数据
	stores["default"].Delete(context.Background(), key)

	return nil
}

func main() {
	if err := Run(); err != nil {
		fmt.Fprintf(os.Stderr, "verify failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ALL CHECKS PASSED")
}
