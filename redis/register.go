// kvx/redis - 驱动注册
package redis

import (
	"github.com/gospacex/kvx"
	"github.com/gospacex/kvx/kvxbase"
)

func init() {
	kvx.RegisterCreator("redis", func(cfg *kvxbase.Config) (kvxbase.KVStore, error) {
		return NewRedis(cfg)
	})
}
