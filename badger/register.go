// kvx/badger - 驱动注册
package badger

import (
	"github.com/gospacex/kvx"
	"github.com/gospacex/kvx/kvxbase"
)

func init() {
	kvx.RegisterCreator("badger", func(cfg *kvxbase.Config) (kvxbase.KVStore, error) {
		return NewBadger(cfg)
	})
}
