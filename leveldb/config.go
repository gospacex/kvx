// kvx/leveldb - LevelDB 驱动配置（仅 Standalone，本地文件存储）
package leveldb

import (
	"time"

	"github.com/gospacex/kvx/kvxbase"
)

// StandaloneConfig LevelDB 本地存储配置
type StandaloneConfig struct {
	Dir         string        `yaml:"dir" json:"dir"`
	BlockCache  int           `yaml:"block_cache_mb" json:"block_cache_mb"`
	WriteBuffer int           `yaml:"write_buffer_mb" json:"write_buffer_mb"`
	Compression bool          `yaml:"compression" json:"compression"`
	Timeout     time.Duration `yaml:"timeout" json:"timeout"`
}

// ToStandaloneConfig 转换为 kvxbase.Config
func (s *StandaloneConfig) ToStandaloneConfig(name string) *kvxbase.Config {
	dir := s.Dir
	if dir == "" {
		dir = "./data/leveldb/" + name
	}
	return &kvxbase.Config{
		Name:        name,
		Driver:      "leveldb",
		Addr:        dir,
		EnableTrace: true,
		Timeout:     s.Timeout,
	}
}
