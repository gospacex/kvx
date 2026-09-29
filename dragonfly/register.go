// kvx/dragonfly - 驱动注册
package dragonfly

import (
	"github.com/gospacex/kvx"
	"github.com/gospacex/kvx/kvxbase"
)

func init() {
	kvx.RegisterCreator("dragonfly", func(cfg *kvxbase.Config) (kvxbase.KVStore, error) {
		return NewDragonfly(cfg)
	})
}
