// kvx/dragonfly - Dragonfly DB 驱动配置（Standalone + Cluster）
package dragonfly

import (
	"time"

	"github.com/gospacex/kvx/kvxbase"
)

// StandaloneConfig Dragonfly 单实例配置
type StandaloneConfig struct {
	Addr         string        `yaml:"addr" json:"addr"`
	Password     string        `yaml:"password" json:"password"`
	DB           int           `yaml:"db" json:"db"`
	PoolSize     int           `yaml:"pool_size" json:"pool_size"`
	MinIdleConn  int           `yaml:"min_idle_conn" json:"min_idle_conn"`
	TLSEnabled   bool          `yaml:"tls_enabled" json:"tls_enabled"`
	Timeout      time.Duration `yaml:"timeout" json:"timeout"`
	RetryTimes   int           `yaml:"retry_times" json:"retry_times"`
	MaxRetries   int           `yaml:"max_retries" json:"max_retries"`
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
}

// ClusterConfig Dragonfly 集群配置
type ClusterConfig struct {
	Addrs        []string      `yaml:"addrs" json:"addrs"`
	Password     string        `yaml:"password" json:"password"`
	PoolSize     int           `yaml:"pool_size" json:"pool_size"`
	MinIdleConn  int           `yaml:"min_idle_conn" json:"min_idle_conn"`
	TLSEnabled   bool          `yaml:"tls_enabled" json:"tls_enabled"`
	Timeout      time.Duration `yaml:"timeout" json:"timeout"`
	RetryTimes   int           `yaml:"retry_times" json:"retry_times"`
	MaxRetries   int           `yaml:"max_retries" json:"max_retries"`
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
	ClusterMode  bool          `yaml:"cluster_mode" json:"cluster_mode"`
	Shards       int           `yaml:"shards" json:"shards"`
	Replicas     int           `yaml:"replicas" json:"replicas"`
}

// ToStandaloneConfig 转换为 kvxbase.Config
func (s *StandaloneConfig) ToStandaloneConfig(name string) *kvxbase.Config {
	return &kvxbase.Config{
		Name:        name,
		Driver:      "dragonfly",
		Addr:        s.Addr,
		Password:    s.Password,
		DB:          s.DB,
		PoolSize:    s.PoolSize,
		MinIdleConn: s.MinIdleConn,
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
	cfg := &kvxbase.Config{
		Name:        name,
		Driver:      "dragonfly",
		Addrs:       addrs,
		Password:    c.Password,
		PoolSize:    c.PoolSize,
		MinIdleConn: c.MinIdleConn,
		TLSEnabled:  c.TLSEnabled,
		EnableTrace: true,
		Timeout:     c.Timeout,
	}
	if c.ClusterMode {
		cfg.Cluster = "true"
	}
	return cfg
}
