// kvx/keydb - 驱动注册
package keydb

import (
	"github.com/gospacex/kvx"
	"github.com/gospacex/kvx/kvxbase"
)

func init() {
	kvx.RegisterCreator("keydb", func(cfg *kvxbase.Config) (kvxbase.KVStore, error) {
		return NewKeyDB(cfg)
	})
}
