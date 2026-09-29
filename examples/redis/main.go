// kvx/examples/redis/main.go: Redis 键值存储示例
// 功能：Set、Get、Delete、MSet、MGet、TTL、Keys、Incr、Decr、调用链追踪
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gospacex/configx"
	"github.com/gospacex/kvx"
	kotel "github.com/gospacex/kvx/otel"
	_ "github.com/gospacex/kvx/redis"
	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
)

func main() {
	ctx := context.Background()

	// 1. 初始化 logx
	if err := logx.ReadConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "logx init fallback: %v\n", err)
	}
	logger := logx.GetGlobal()

	// 2. 初始化 OTel 调用链（stdout 调试模式，确保 span 可见）
	shutdown := kotel.InitOTel(
		kotel.WithServiceName("kvx-redis-example"),
		kotel.WithBackend(kotel.BackendStdout),
		kotel.WithDebug(true),
	)
	defer shutdown(ctx)
	logger.InfoContext(ctx, "kvx redis example started", logx.String("driver", "redis"))

	// 3. 读取配置并创建存储
	stores, err := kvx.ReadAndCreate(configx.ConfigOptions{
		ConfigFile: "config.yaml",
		ConfigType: "yaml",
	})
	if err != nil {
		logger.FatalContext(ctx, "create kvx client failed", logx.Err(err))
	}
	defer stores["default"].Close()

	store := stores["default"]
	if store == nil {
		logger.FatalContext(ctx, "store instance not found")
	}
	logger.InfoContext(ctx, "driver name", logx.String("driver", "redis"))

	// 4. 健康检查
	if err := store.HealthCheck(); err != nil {
		logger.FatalContext(ctx, "health check failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "health check passed")

	// 5. Set 和 Get
	key := "test-key"
	value := "hello-kvx"
	logger.InfoContext(ctx, "setting value",
		logx.String("key", key),
		logx.String("value", value),
	)
	if err := store.Set(ctx, key, value, 10*time.Second); err != nil {
		logger.FatalContext(ctx, "set failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "value set", logx.String("key", key))

	got, err := store.Get(ctx, key)
	if err != nil {
		logger.FatalContext(ctx, "get failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "value retrieved",
		logx.String("key", key),
		logx.String("value", got),
	)

	// 6. MSet 和 MGet
	multiKey := map[string]string{
		"k1": "v1",
		"k2": "v2",
		"k3": "v3",
	}
	logger.InfoContext(ctx, "setting multiple values",
		logx.Int("count", len(multiKey)),
	)
	if err := store.MSet(ctx, multiKey); err != nil {
		logger.FatalContext(ctx, "mset failed", logx.Err(err))
	}
	keys := []string{"k1", "k2", "k3"}
	gots, err := store.MGet(ctx, keys...)
	if err != nil {
		logger.FatalContext(ctx, "mget failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "multiple values retrieved",
		logx.Int("count", len(gots)),
	)

	// 7. Keys pattern query
	pattern := "k*"
	kvs, err := store.Keys(ctx, pattern)
	if err != nil {
		logger.FatalContext(ctx, "keys failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "keys found",
		logx.Int("count", len(kvs)),
		logx.String("pattern", pattern),
	)
	for _, k := range kvs {
		logger.InfoContext(ctx, "key", logx.String("key", k))
	}

	// 8. Incr
	if err := store.Set(ctx, "counter", "0", 30*time.Second); err != nil {
		logger.FatalContext(ctx, "set counter failed", logx.Err(err))
	}
	val, err := store.Incr(ctx, "counter")
	if err != nil {
		logger.FatalContext(ctx, "incr failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "incr result", logx.Any("value", val))

	// 9. Decr
	val, err = store.Decr(ctx, "counter")
	if err != nil {
		logger.FatalContext(ctx, "decr failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "decr result", logx.Any("value", val))

	// 10. TTL
	ttl, err := store.TTL(ctx, key)
	if err != nil {
		logger.FatalContext(ctx, "ttl failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "key ttl", logx.Any("ttl", ttl))

	// 11. Exists
	exists, err := store.Exists(ctx, key)
	if err != nil {
		logger.FatalContext(ctx, "exists failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "key exists", logx.Any("exists", exists))

	// 12. Delete
	if err := store.Delete(ctx, key); err != nil {
		logger.FatalContext(ctx, "delete failed", logx.Err(err))
	}
	for _, k := range keys {
		if err := store.Delete(ctx, k); err != nil {
			logger.FatalContext(ctx, "delete failed", logx.Err(err))
		}
	}
	logger.InfoContext(ctx, "keys deleted")

	fmt.Println("\n✅ All operations completed successfully.")
}
