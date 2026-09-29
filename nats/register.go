// kvx/nats - 驱动注册
package nats

import (
	"github.com/gospacex/kvx"
	"github.com/gospacex/kvx/kvxbase"
)

func init() {
	kvx.RegisterCreator("nats", func(cfg *kvxbase.Config) (kvxbase.KVStore, error) {
		return NewNats(cfg)
	})
}
