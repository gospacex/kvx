// kvx/pebble - 驱动注册
package pebble

import (
	"github.com/gospacex/kvx"
	"github.com/gospacex/kvx/kvxbase"
)

func init() {
	kvx.RegisterCreator("pebble", func(cfg *kvxbase.Config) (kvxbase.KVStore, error) {
		return NewPebble(cfg)
	})
}
