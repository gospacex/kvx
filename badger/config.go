// kvx/badger - Badger 驱动配置（仅 Standalone，本地文件存储）
package badger

import (
	"time"

	"github.com/gospacex/kvx/kvxbase"
)

// StandaloneConfig Badger 本地存储配置
type StandaloneConfig struct {
	Dir           string        `yaml:"dir" json:"dir"`
	ValueLogGC    bool          `yaml:"value_log_gc" json:"value_log_gc"`
	MemTableSize  int           `yaml:"mem_table_size" json:"mem_table_size"`
	BlockCache    bool          `yaml:"block_cache" json:"block_cache"`
	Compression   bool          `yaml:"compression" json:"compression"`
	NumCompactors int           `yaml:"num_compactors" json:"num_compactors"`
	Timeout       time.Duration `yaml:"timeout" json:"timeout"`
}

// ToStandaloneConfig 转换为 kvxbase.Config
func (s *StandaloneConfig) ToStandaloneConfig(name string) *kvxbase.Config {
	dir := s.Dir
	if dir == "" {
		dir = "./data/badger/" + name
	}
	return &kvxbase.Config{
		Name:        name,
		Driver:      "badger",
		Addr:        dir,
		EnableTrace: true,
		Timeout:     s.Timeout,
	}
}
