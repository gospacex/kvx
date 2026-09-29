// kvx/etcd - etcd 驱动配置（Standalone + Cluster 多节点）
package etcd

import (
	"time"

	"github.com/gospacex/kvx/kvxbase"
)

// StandaloneConfig etcd 单实例配置
type StandaloneConfig struct {
	Addr       string        `yaml:"addr" json:"addr"`
	Username   string        `yaml:"username" json:"username"`
	Password   string        `yaml:"password" json:"password"`
	TLSEnabled bool          `yaml:"tls_enabled" json:"tls_enabled"`
	Timeout    time.Duration `yaml:"timeout" json:"timeout"`
	LeaseTTL   int64         `yaml:"lease_ttl" json:"lease_ttl"`
}

// ClusterConfig etcd 集群配置（多节点）
type ClusterConfig struct {
	Addrs        []string      `yaml:"addrs" json:"addrs"`
	Username     string        `yaml:"username" json:"username"`
	Password     string        `yaml:"password" json:"password"`
	TLSEnabled   bool          `yaml:"tls_enabled" json:"tls_enabled"`
	Timeout      time.Duration `yaml:"timeout" json:"timeout"`
	LeaseTTL     int64         `yaml:"lease_ttl" json:"lease_ttl"`
	Namespace    string        `yaml:"namespace" json:"namespace"`
	WatchEnabled bool          `yaml:"watch_enabled" json:"watch_enabled"`
}

// ToStandaloneConfig 转换为 kvxbase.Config
func (s *StandaloneConfig) ToStandaloneConfig(name string) *kvxbase.Config {
	return &kvxbase.Config{
		Name:        name,
		Driver:      "etcd",
		Addr:        s.Addr,
		Password:    s.Password,
		TLSEnabled:  s.TLSEnabled,
		EnableTrace: true,
		Timeout:     s.Timeout,
	}
}

// ToClusterConfig 转换为 kvxbase.Config
func (c *ClusterConfig) ToClusterConfig(name string) *kvxbase.Config {
	addrs := make([]string, 0, len(c.Addrs))
	for _, a := range c.Addrs {
		addrs = append(addrs, a)
	}
	return &kvxbase.Config{
		Name:        name,
		Driver:      "etcd",
		Addrs:       addrs,
		Password:    c.Password,
		TLSEnabled:  c.TLSEnabled,
		EnableTrace: true,
		Timeout:     c.Timeout,
	}
}
