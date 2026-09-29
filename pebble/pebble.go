// kvx/pebble - Pebble KV 存储驱动
// 使用 cockroachdb/pebble，本地文件存储（RocksDB 替代）
package pebble

import (
	"context"
	"fmt"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/gospacex/kvx/kvxbase"
	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
)

// Store Pebble KVStore 实现
type Store struct {
	db     *pebble.DB
	name   string
	logger logx.Logger
}

func NewPebble(cfg *kvxbase.Config) (*Store, error) {
	cfg.FillDefaults()

	path := cfg.Addr
	if path == "" {
		path = "./data/pebble/" + cfg.Name
	}

	db, err := pebble.Open(path, &pebble.Options{})
	if err != nil {
		return nil, fmt.Errorf("kvx/pebble: open failed: %w", err)
	}

	return &Store{db: db, name: cfg.Name, logger: logx.GetGlobal()}, nil
}

func (s *Store) getBucket() []byte { return []byte("default") }

// Get 获取单个值
func (s *Store) Get(ctx context.Context, key string) (string, error) {
	keyBytes := append(s.getBucket(), key...)
	val, closer, err := s.db.Get(keyBytes)
	if err == pebble.ErrNotFound {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("kvx/pebble get: %w", err)
	}
	defer closer.Close()
	return string(val), nil
}

// Set 设置单个值
func (s *Store) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	keyBytes := append(s.getBucket(), key...)
	opts := &pebble.WriteOptions{}
	err := s.db.Set(keyBytes, []byte(value), opts)
	if err != nil {
		return fmt.Errorf("kvx/pebble set: %w", err)
	}
	return nil
}

// Delete 删除单个值
func (s *Store) Delete(ctx context.Context, key string) error {
	keyBytes := append(s.getBucket(), key...)
	err := s.db.Delete(keyBytes, nil)
	if err == pebble.ErrNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("kvx/pebble delete: %w", err)
	}
	return nil
}

// Exists 检查 key 是否存在
func (s *Store) Exists(ctx context.Context, key string) (bool, error) {
	keyBytes := append(s.getBucket(), key...)
	_, closer, err := s.db.Get(keyBytes)
	if err == pebble.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	closer.Close()
	return true, nil
}

// Keys 获取匹配前缀的 keys
func (s *Store) Keys(ctx context.Context, prefix string) ([]string, error) {
	prefixBytes := append(s.getBucket(), prefix...)
	iter, err := s.db.NewIter(nil)
	if err != nil {
		return nil, fmt.Errorf("kvx/pebble keys: %w", err)
	}
	defer iter.Close()

	iter.SeekGE(prefixBytes)
	var keys []string
	for ; iter.Valid(); iter.Next() {
		k := iter.Key()
		if len(k) >= len(prefixBytes) && string(k[:len(prefixBytes)]) == prefix {
			keys = append(keys, string(k[len(s.getBucket()):]))
		} else if len(k) > len(prefixBytes) {
			break
		}
	}
	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("kvx/pebble keys: %w", err)
	}
	return keys, nil
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
	w := s.db.NewBatch()
	defer w.Close()
	for key, value := range pairs {
		keyBytes := append(s.getBucket(), key...)
		if err := w.Set(keyBytes, []byte(value), nil); err != nil {
			return fmt.Errorf("kvx/pebble mset: %w", err)
		}
	}
	if err := w.Commit(pebble.Sync); err != nil {
		return fmt.Errorf("kvx/pebble mset: %w", err)
	}
	return nil
}

// TTL Pebble 不直接支持 TTL，返回 0
func (s *Store) TTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}

// Incr 自增
func (s *Store) Incr(ctx context.Context, key string) (int64, error) {
	keyBytes := append(s.getBucket(), key...)
	val, closer, err := s.db.Get(keyBytes)
	var current int64
	if err == nil {
		defer closer.Close()
		fmt.Sscanf(string(val), "%d", &current)
	}
	b := s.db.NewBatch()
	newVal := []byte(fmt.Sprintf("%d", current+1))
	if err := b.Set(keyBytes, newVal, nil); err != nil {
		b.Close()
		return 0, err
	}
	if err := s.db.Apply(b, nil); err != nil {
		b.Close()
		return 0, err
	}
	b.Close()
	return current + 1, nil
}

// Decr 自减
func (s *Store) Decr(ctx context.Context, key string) (int64, error) {
	keyBytes := append(s.getBucket(), key...)
	val, closer, err := s.db.Get(keyBytes)
	var current int64
	if err == nil {
		defer closer.Close()
		fmt.Sscanf(string(val), "%d", &current)
	}
	b := s.db.NewBatch()
	newVal := []byte(fmt.Sprintf("%d", current-1))
	if err := b.Set(keyBytes, newVal, nil); err != nil {
		b.Close()
		return 0, err
	}
	if err := s.db.Apply(b, nil); err != nil {
		b.Close()
		return 0, err
	}
	b.Close()
	return current - 1, nil
}

// HealthCheck 健康检查
func (s *Store) HealthCheck() error {
	keyBytes := append(s.getBucket(), []byte("_health_check_kvx_")...)
	return s.db.Set(keyBytes, []byte("ok"), nil)
}

// Close 关闭连接
func (s *Store) Close() error {
	return s.db.Close()
}
