// kvx/memcached - 驱动注册
package memcached

import (
	"github.com/gospacex/kvx"
	"github.com/gospacex/kvx/kvxbase"
)

func init() {
	kvx.RegisterCreator("memcached", func(cfg *kvxbase.Config) (kvxbase.KVStore, error) {
		return NewMemcached(cfg)
	})
}
