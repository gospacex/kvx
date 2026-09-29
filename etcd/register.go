// kvx/etcd - 驱动注册
package etcd

import (
	"github.com/gospacex/kvx"
	"github.com/gospacex/kvx/kvxbase"
)

func init() {
	kvx.RegisterCreator("etcd", func(cfg *kvxbase.Config) (kvxbase.KVStore, error) {
		return NewEtcd(cfg)
	})
}
