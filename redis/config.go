// kvx/redis - Redis KV 驱动配置（Standalone + Cluster）
package redis

import (
	"time"

	"github.com/gospacex/kvx/kvxbase"
)

// StandaloneConfig Redis 单实例配置
type StandaloneConfig struct {
	Addr           string        `yaml:"addr" json:"addr"`
	Password       string        `yaml:"password" json:"password"`
	DB             int           `yaml:"db" json:"db"`
	PoolSize       int           `yaml:"pool_size" json:"pool_size"`
	MinIdleConn    int           `yaml:"min_idle_conn" json:"min_idle_conn"`
	SentinelMaster string        `yaml:"sentinel_master" json:"sentinel_master"`
	SentinelAddrs  string        `yaml:"sentinel_addrs" json:"sentinel_addrs"`
	TLSEnabled     bool          `yaml:"tls_enabled" json:"tls_enabled"`
	Timeout        time.Duration `yaml:"timeout" json:"timeout"`
	RetryTimes     int           `yaml:"retry_times" json:"retry_times"`
	MaxRetries     int           `yaml:"max_retries" json:"max_retries"`
	// Enterprise
	RouteByLatency bool          `yaml:"route_by_latency" json:"route_by_latency"`
	ReadTimeout    time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout" json:"write_timeout"`
	EnableTrace    bool          `yaml:"enable_trace" json:"enable_trace"`
}

// ClusterConfig Redis 集群配置（Cluster/Sentinel）
type ClusterConfig struct {
	Addrs          []string      `yaml:"addrs" json:"addrs"`
	Password       string        `yaml:"password" json:"password"`
	PoolSize       int           `yaml:"pool_size" json:"pool_size"`
	MinIdleConn    int           `yaml:"min_idle_conn" json:"min_idle_conn"`
	TLSEnabled     bool          `yaml:"tls_enabled" json:"tls_enabled"`
	Timeout        time.Duration `yaml:"timeout" json:"timeout"`
	RetryTimes     int           `yaml:"retry_times" json:"retry_times"`
	MaxRetries     int           `yaml:"max_retries" json:"max_retries"`
	SentinelMaster string        `yaml:"sentinel_master" json:"sentinel_master"`
	SentinelAddrs  []string      `yaml:"sentinel_addrs" json:"sentinel_addrs"`
	// Enterprise
	RouteByLatency bool          `yaml:"route_by_latency" json:"route_by_latency"`
	ReadTimeout    time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout" json:"write_timeout"`
	MaxRedirects   int           `yaml:"max_redirects" json:"max_redirects"`
}

// ToStandaloneConfig 转换为 kvxbase.Config
func (s *StandaloneConfig) ToStandaloneConfig(name string) *kvxbase.Config {
	return &kvxbase.Config{
		Name:        name,
		Driver:      "redis",
		Addr:        s.Addr,
		Password:    s.Password,
		DB:          s.DB,
		PoolSize:    s.PoolSize,
		MinIdleConn: s.MinIdleConn,
		TLSEnabled:  s.TLSEnabled,
		EnableTrace: s.EnableTrace,
		Timeout:     s.Timeout,
	}
}

// ToClusterConfig 转换为 kvxbase.Config
func (c *ClusterConfig) ToClusterConfig(name string) *kvxbase.Config {
	addrs := make([]string, 0, len(c.Addrs))
	for _, a := range c.Addrs {
		addrs = append(addrs, a)
	}
	sentAddrs := make([]string, 0, len(c.SentinelAddrs))
	for _, a := range c.SentinelAddrs {
		sentAddrs = append(sentAddrs, a)
	}
	cfg := &kvxbase.Config{
		Name:        name,
		Driver:      "redis",
		Addr:        "",
		Cluster:     "true",
		Addrs:       addrs,
		Password:    c.Password,
		PoolSize:    c.PoolSize,
		MinIdleConn: c.MinIdleConn,
		TLSEnabled:  c.TLSEnabled,
		EnableTrace: false,
		Timeout:     c.Timeout,
	}
	if c.MaxRedirects > 0 {
		cfg.Cluster = "true"
	}
	return cfg
}

func encodeAddrs(addrs []string) string {
	s := ""
	for i, a := range addrs {
		if i > 0 {
			s += ","
		}
		s += a
	}
	return s
}
