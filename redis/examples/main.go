// kvx/redis/examples: Redis KV 存储示例
// 功能：Set、Get、Incr、TTL、Keys、HealthCheck
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gospacex/configx"
	"github.com/gospacex/kvx"
	_ "github.com/gospacex/kvx/redis"
	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
)

func main() {
	ctx := context.Background()

	if err := logx.ReadConfig(); err != nil {
		log.Printf("logx.ReadConfig fallback: %v", err)
	}
	logger := logx.GetGlobal()
	logger.InfoContext(ctx, "kvx redis example started", logx.String("driver", "redis"))

	instances, err := kvx.ReadAndCreate(configx.ConfigOptions{
		ConfigFile: "config.yaml",
		ConfigType: "yaml",
	})
	if err != nil {
		logger.Fatal("create kvx client failed", logx.Err(err))
	}
	defer func() {
		for name, st := range instances {
			if cerr := st.Close(); cerr != nil {
				logger.Error("close store failed", logx.String("name", name), logx.Err(cerr))
			}
		}
	}()

	store := instances["default"]

	// Set / Get
	if err := store.Set(ctx, "k1", "v1", 10*time.Second); err != nil {
		logger.Fatal("set failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "set key", logx.String("key", "k1"))

	val, err := store.Get(ctx, "k1")
	if err != nil {
		logger.Fatal("get failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "get value", logx.String("value", val))

	// Incr
	if _, err := store.Incr(ctx, "counter"); err != nil {
		logger.Fatal("incr failed", logx.Err(err))
	}
	counter, err := store.Get(ctx, "counter")
	if err != nil {
		logger.Fatal("get counter failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "counter value", logx.String("value", counter))

	// Keys
	keys, err := store.Keys(ctx, "k*")
	if err != nil {
		logger.Fatal("keys failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "keys found", logx.Int("count", len(keys)))

	// HealthCheck
	if err := store.HealthCheck(); err != nil {
		logger.Fatal("health check failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "health check passed")

	fmt.Println("\n✅ All operations completed successfully.")
}
