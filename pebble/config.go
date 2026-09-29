// kvx/pebble - Pebble 驱动配置（仅 Standalone，本地文件存储）
package pebble

import (
	"time"

	"github.com/gospacex/kvx/kvxbase"
)

// StandaloneConfig Pebble 本地存储配置
type StandaloneConfig struct {
	Dir           string        `yaml:"dir" json:"dir"`
	LBaseMaxBytes int           `yaml:"l_base_max_bytes" json:"l_base_max_bytes"`
	BlockCache    int           `yaml:"block_cache_mb" json:"block_cache_mb"`
	Compression   bool          `yaml:"compression" json:"compression"`
	Timeout       time.Duration `yaml:"timeout" json:"timeout"`
}

// ToStandaloneConfig 转换为 kvxbase.Config
func (s *StandaloneConfig) ToStandaloneConfig(name string) *kvxbase.Config {
	dir := s.Dir
	if dir == "" {
		dir = "./data/pebble/" + name
	}
	return &kvxbase.Config{
		Name:        name,
		Driver:      "pebble",
		Addr:        dir,
		EnableTrace: true,
		Timeout:     s.Timeout,
	}
}
