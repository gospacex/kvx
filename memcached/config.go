// kvx/memcached - Memcached 驱动配置（Standalone + Cluster 多节点）
package memcached

import (
	"time"

	"github.com/gospacex/kvx/kvxbase"
)

// StandaloneConfig Memcached 单节点配置
type StandaloneConfig struct {
	Addr        string        `yaml:"addr" json:"addr"`
	Password    string        `yaml:"password" json:"password"`
	PoolSize    int           `yaml:"pool_size" json:"pool_size"`
	MaxIdleConn int           `yaml:"max_idle_conn" json:"max_idle_conn"`
	TLSEnabled  bool          `yaml:"tls_enabled" json:"tls_enabled"`
	Timeout     time.Duration `yaml:"timeout" json:"timeout"`
	RetryTimes  int           `yaml:"retry_times" json:"retry_times"`
	MaxRetries  int           `yaml:"max_retries" json:"max_retries"`
	MaxConn     int           `yaml:"max_conn" json:"max_conn"`
	KeyPrefix   string        `yaml:"key_prefix" json:"key_prefix"`
	UseSASL     bool          `yaml:"use_sasl" json:"use_sasl"`
}

// ClusterConfig Memcached 集群配置（多节点分布式）
type ClusterConfig struct {
	Addrs          []string      `yaml:"addrs" json:"addrs"`
	Password       string        `yaml:"password" json:"password"`
	PoolSize       int           `yaml:"pool_size" json:"pool_size"`
	MaxIdleConn    int           `yaml:"max_idle_conn" json:"max_idle_conn"`
	TLSEnabled     bool          `yaml:"tls_enabled" json:"tls_enabled"`
	Timeout        time.Duration `yaml:"timeout" json:"timeout"`
	RetryTimes     int           `yaml:"retry_times" json:"retry_times"`
	MaxRetries     int           `yaml:"max_retries" json:"max_retries"`
	MaxConn        int           `yaml:"max_conn" json:"max_conn"`
	KeyPrefix      string        `yaml:"key_prefix" json:"key_prefix"`
	UseSASL        bool          `yaml:"use_sasl" json:"use_sasl"`
	ConsistentHash bool          `yaml:"consistent_hash" json:"consistent_hash"`
}

// ToStandaloneConfig 转换为 kvxbase.Config
func (s *StandaloneConfig) ToStandaloneConfig(name string) *kvxbase.Config {
	cfg := &kvxbase.Config{
		Name:        name,
		Driver:      "memcached",
		Addr:        s.Addr,
		Password:    s.Password,
		PoolSize:    s.PoolSize,
		MinIdleConn: s.MaxIdleConn,
		TLSEnabled:  s.TLSEnabled,
		EnableTrace: true,
		Timeout:     s.Timeout,
	}
	return cfg
}

// ToClusterConfig 转换为 kvxbase.Config
func (c *ClusterConfig) ToClusterConfig(name string) *kvxbase.Config {
	addrs := make([]string, 0, len(c.Addrs))
	for _, a := range c.Addrs {
		addrs = append(addrs, a)
	}
	cfg := &kvxbase.Config{
		Name:        name,
		Driver:      "memcached",
		Addrs:       addrs,
		Password:    c.Password,
		PoolSize:    c.PoolSize,
		MinIdleConn: c.MaxIdleConn,
		TLSEnabled:  c.TLSEnabled,
		EnableTrace: true,
		Timeout:     c.Timeout,
	}
	if c.ConsistentHash {
		cfg.Cluster = "true"
	}
	return cfg
}
