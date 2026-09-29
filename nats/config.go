// kvx/nats - NATS JetStream 驱动配置（Standalone + Cluster）
package nats

import (
	"time"

	"github.com/gospacex/kvx/kvxbase"
)

// StandaloneConfig NATS 单实例配置
type StandaloneConfig struct {
	Addr           string        `yaml:"addr" json:"addr"`
	Username       string        `yaml:"username" json:"username"`
	Password       string        `yaml:"password" json:"password"`
	AllowReconnect bool          `yaml:"allow_reconnect" json:"allow_reconnect"`
	MaxReconnect   int           `yaml:"max_reconnect" json:"max_reconnect"`
	Timeout        time.Duration `yaml:"timeout" json:"timeout"`
	JSBucket       string        `yaml:"js_bucket" json:"js_bucket"`
}

// ClusterConfig NATS 集群配置（多节点）
type ClusterConfig struct {
	Servers        []string      `yaml:"servers" json:"servers"`
	Username       string        `yaml:"username" json:"username"`
	Password       string        `yaml:"password" json:"password"`
	AllowReconnect bool          `yaml:"allow_reconnect" json:"allow_reconnect"`
	MaxReconnect   int           `yaml:"max_reconnect" json:"max_reconnect"`
	Timeout        time.Duration `yaml:"timeout" json:"timeout"`
	JSBucket       string        `yaml:"js_bucket" json:"js_bucket"`
	ClusterName    string        `yaml:"cluster_name" json:"cluster_name"`
}

// ToStandaloneConfig 转换为 kvxbase.Config
func (s *StandaloneConfig) ToStandaloneConfig(name string) *kvxbase.Config {
	return &kvxbase.Config{
		Name:        name,
		Driver:      "nats",
		Addr:        s.Addr,
		Password:    s.Password,
		EnableTrace: true,
		Timeout:     s.Timeout,
	}
}

// ToClusterConfig 转换为 kvxbase.Config
func (c *ClusterConfig) ToClusterConfig(name string) *kvxbase.Config {
	servers := make([]string, 0, len(c.Servers))
	for _, s := range c.Servers {
		servers = append(servers, s)
	}
	return &kvxbase.Config{
		Name:        name,
		Driver:      "nats",
		Addrs:       servers,
		Password:    c.Password,
		EnableTrace: true,
		Timeout:     c.Timeout,
	}
}
