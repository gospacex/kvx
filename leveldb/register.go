// kvx/leveldb - 驱动注册
package leveldb

import (
	"github.com/gospacex/kvx"
	"github.com/gospacex/kvx/kvxbase"
)

func init() {
	kvx.RegisterCreator("leveldb", func(cfg *kvxbase.Config) (kvxbase.KVStore, error) {
		return NewLevelDB(cfg)
	})
}
