// kvx/etcd/examples: etcd KV 存储示例
// 功能：Set / Get / Exists / Keys / Delete / HealthCheck
//
// 运行：
//
//	cd kvx/etcd/examples && go run .
//
// 需先起中间件：cd kvx/etcd && ./scripts/start.sh
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gospacex/configx"
	"github.com/gospacex/kvx"
	_ "github.com/gospacex/kvx/etcd"
	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
)

func main() {
	ctx := context.Background()

	if err := logx.ReadConfig(); err != nil {
		log.Printf("logx.ReadConfig fallback: %v", err)
	}
	logger := logx.GetGlobal()
	logger.InfoContext(ctx, "kvx etcd example started", logx.String("driver", "etcd"))

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

	// Exists
	ok, err := store.Exists(ctx, "k1")
	if err != nil {
		logger.Fatal("exists failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "key exists", logx.Any("exists", ok))

	// Keys
	keys, err := store.Keys(ctx, "k*")
	if err != nil {
		logger.Fatal("keys failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "keys found", logx.Int("count", len(keys)))

	// Delete
	if err := store.Delete(ctx, "k1"); err != nil {
		logger.Fatal("delete failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "deleted key", logx.String("key", "k1"))

	// HealthCheck
	if err := store.HealthCheck(); err != nil {
		logger.Fatal("health check failed", logx.Err(err))
	}
	logger.InfoContext(ctx, "health check passed")

	fmt.Println("\n✅ All operations completed successfully.")
}
