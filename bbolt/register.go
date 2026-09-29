// kvx/bbolt - 驱动注册
package bbolt

import (
	"github.com/gospacex/kvx"
	"github.com/gospacex/kvx/kvxbase"
)

func init() {
	kvx.RegisterCreator("bbolt", func(cfg *kvxbase.Config) (kvxbase.KVStore, error) {
		return NewBBolt(cfg)
	})
}
