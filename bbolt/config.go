// kvx/bbolt - BBolt 驱动配置（仅 Standalone，本地文件存储）
package bbolt

import (
	"time"

	"github.com/gospacex/kvx/kvxbase"
)

// StandaloneConfig BBolt 本地存储配置
type StandaloneConfig struct {
	Path     string        `yaml:"path" json:"path"`
	FileMode int           `yaml:"file_mode" json:"file_mode"`
	Timeout  time.Duration `yaml:"timeout" json:"timeout"`
}

// ToStandaloneConfig 转换为 kvxbase.Config
func (s *StandaloneConfig) ToStandaloneConfig(name string) *kvxbase.Config {
	path := s.Path
	if path == "" {
		path = "./data/bbolt/" + name + ".db"
	}
	return &kvxbase.Config{
		Name:        name,
		Driver:      "bbolt",
		Addr:        path,
		EnableTrace: true,
		Timeout:     s.Timeout,
	}
}
