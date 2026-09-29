// kvx/bbolt - BBolt KV 存储驱动
// 使用 go.etcd.io/bbolt，本地文件存储
package bbolt

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/gospacex/kvx/kvxbase"
	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"go.etcd.io/bbolt"
)

// Store BBolt KVStore 实现
type Store struct {
	db     *bbolt.DB
	name   string
	logger logx.Logger
}

func NewBBolt(cfg *kvxbase.Config) (*Store, error) {
	cfg.FillDefaults()

	path := cfg.Addr
	if path == "" {
		path = "./data/bbolt/" + cfg.Name + ".db"
	}

	db, err := bbolt.Open(path, 0600, &bbolt.Options{Timeout: cfg.Timeout})
	if err != nil {
		return nil, fmt.Errorf("kvx/bbolt: open failed: %w", err)
	}

	// 创建默认 bucket
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("default"))
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("kvx/bbolt: create bucket failed: %w", err)
	}

	return &Store{db: db, name: cfg.Name, logger: logx.GetGlobal()}, nil
}

func (s *Store) getBucket() []byte { return []byte("default") }

// Get 获取单个值
func (s *Store) Get(ctx context.Context, key string) (string, error) {
	var val string
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(s.getBucket())
		v := b.Get([]byte(key))
		if v == nil {
			return nil
		}
		val = string(v)
		return nil
	})
	return val, err
}

// Set 设置单个值
func (s *Store) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(s.getBucket())
		return b.Put([]byte(key), []byte(value))
	})
}

// Delete 删除单个值
func (s *Store) Delete(ctx context.Context, key string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(s.getBucket())
		return b.Delete([]byte(key))
	})
}

// Exists 检查 key 是否存在
func (s *Store) Exists(ctx context.Context, key string) (bool, error) {
	var found bool
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(s.getBucket())
		if b.Get([]byte(key)) != nil {
			found = true
		}
		return nil
	})
	return found, err
}

// Keys 获取匹配前缀的 keys
func (s *Store) Keys(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(s.getBucket())
		return b.ForEach(func(k, _ []byte) error {
			if prefix == "" || len(k) >= len(prefix) && string(k[:len(prefix)]) == prefix {
				keys = append(keys, string(k))
			}
			return nil
		})
	})
	return keys, err
}

// MGet 批量获取
func (s *Store) MGet(ctx context.Context, keys ...string) (map[string]string, error) {
	result := make(map[string]string)
	for _, key := range keys {
		val, err := s.Get(ctx, key)
		if err != nil {
			continue
		}
		if val != "" {
			result[key] = val
		}
	}
	return result, nil
}

// MSet 批量设置
func (s *Store) MSet(ctx context.Context, pairs map[string]string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(s.getBucket())
		for key, value := range pairs {
			if err := b.Put([]byte(key), []byte(value)); err != nil {
				return err
			}
		}
		return nil
	})
}

// TTL BBolt 不直接支持 TTL，返回 0
func (s *Store) TTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}

// Incr 自增
func (s *Store) Incr(ctx context.Context, key string) (int64, error) {
	var result int64
	err := s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(s.getBucket())
		v := b.Get([]byte(key))
		var current int64
		if v != nil {
			current = int64(binary.BigEndian.Uint64(v))
		}
		newVal := make([]byte, 8)
		binary.BigEndian.PutUint64(newVal, uint64(current+1))
		if err := b.Put([]byte(key), newVal); err != nil {
			return err
		}
		result = current + 1
		return nil
	})
	return result, err
}

// Decr 自减
func (s *Store) Decr(ctx context.Context, key string) (int64, error) {
	var result int64
	err := s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(s.getBucket())
		v := b.Get([]byte(key))
		var current int64
		if v != nil {
			current = int64(binary.BigEndian.Uint64(v))
		}
		if current == 0 {
			return fmt.Errorf("kvx/bbolt: decr underflow on key %q", key)
		}
		newVal := make([]byte, 8)
		binary.BigEndian.PutUint64(newVal, uint64(current-1))
		if err := b.Put([]byte(key), newVal); err != nil {
			return err
		}
		result = current - 1
		return nil
	})
	return result, err
}

// HealthCheck 健康检查
func (s *Store) HealthCheck() error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(s.getBucket())
		return b.Put([]byte("_health_check_kvx_"), []byte("ok"))
	})
}

// Close 关闭连接
func (s *Store) Close() error {
	return s.db.Close()
}
