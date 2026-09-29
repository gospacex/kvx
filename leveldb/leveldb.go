// kvx/leveldb - LevelDB KV 存储驱动
package leveldb

import (
	"context"
	"fmt"
	"time"

	"github.com/gospacex/kvx/kvxbase"
	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type Store struct {
	db     *leveldb.DB
	name   string
	logger logx.Logger
}

func NewLevelDB(cfg *kvxbase.Config) (*Store, error) {
	cfg.FillDefaults()

	path := cfg.Addr
	if path == "" {
		path = "./data/leveldb/" + cfg.Name
	}

	db, err := leveldb.OpenFile(path, &opt.Options{
		BlockCacheCapacity: 32 * opt.MiB,
		WriteBuffer:        4 * opt.MiB,
	})
	if err != nil {
		return nil, fmt.Errorf("kvx/leveldb: open failed: %w", err)
	}

	return &Store{db: db, name: cfg.Name, logger: logx.GetGlobal()}, nil
}

func (s *Store) getBucket() []byte { return []byte("default") }

func (s *Store) Get(ctx context.Context, key string) (string, error) {
	val, err := s.db.Get(append(s.getBucket(), key...), nil)
	if err == leveldb.ErrNotFound {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("kvx/leveldb get: %w", err)
	}
	return string(val), nil
}

func (s *Store) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	err := s.db.Put(append(s.getBucket(), key...), []byte(value), nil)
	if err != nil {
		return fmt.Errorf("kvx/leveldb set: %w", err)
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	err := s.db.Delete(append(s.getBucket(), key...), nil)
	if err == leveldb.ErrNotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("kvx/leveldb delete: %w", err)
	}
	return nil
}

func (s *Store) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.db.Get(append(s.getBucket(), key...), nil)
	if err == leveldb.ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) Keys(ctx context.Context, prefix string) ([]string, error) {
	prefixBytes := append(s.getBucket(), prefix...)
	iter := s.db.NewIterator(&util.Range{Start: prefixBytes}, nil)
	defer iter.Release()

	var keys []string
	for iter.Next() {
		k := iter.Key()
		if len(k) >= len(prefixBytes) && string(k[:len(prefixBytes)]) == prefix {
			keys = append(keys, string(k[len(s.getBucket()):]))
		} else if len(k) > len(prefixBytes) {
			break
		}
	}
	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("kvx/leveldb keys: %w", err)
	}
	return keys, nil
}

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

func (s *Store) MSet(ctx context.Context, pairs map[string]string) error {
	batch := new(leveldb.Batch)
	for key, value := range pairs {
		batch.Put(append(s.getBucket(), key...), []byte(value))
	}
	err := s.db.Write(batch, nil)
	if err != nil {
		return fmt.Errorf("kvx/leveldb mset: %w", err)
	}
	return nil
}

func (s *Store) TTL(ctx context.Context, key string) (time.Duration, error) {
	return 0, nil
}

func (s *Store) Incr(ctx context.Context, key string) (int64, error) {
	keyBytes := append(s.getBucket(), key...)
	var result int64
	tr, err := s.db.OpenTransaction()
	if err != nil {
		return 0, err
	}
	v, err := tr.Get(keyBytes, nil)
	if err != leveldb.ErrNotFound && err != nil {
		tr.Discard()
		return 0, err
	}
	var current int64
	if err == nil {
		fmt.Sscanf(string(v), "%d", &current)
	}
	newVal := []byte(fmt.Sprintf("%d", current+1))
	if err := tr.Put(keyBytes, newVal, nil); err != nil {
		tr.Discard()
		return 0, err
	}
	result = current + 1
	if err := tr.Commit(); err != nil {
		return 0, err
	}
	return result, nil
}

func (s *Store) Decr(ctx context.Context, key string) (int64, error) {
	keyBytes := append(s.getBucket(), key...)
	var result int64
	tr, err := s.db.OpenTransaction()
	if err != nil {
		return 0, err
	}
	v, err := tr.Get(keyBytes, nil)
	if err != leveldb.ErrNotFound && err != nil {
		tr.Discard()
		return 0, err
	}
	var current int64
	if err == nil {
		fmt.Sscanf(string(v), "%d", &current)
	}
	newVal := []byte(fmt.Sprintf("%d", current-1))
	if err := tr.Put(keyBytes, newVal, nil); err != nil {
		tr.Discard()
		return 0, err
	}
	result = current - 1
	if err := tr.Commit(); err != nil {
		return 0, err
	}
	return result, nil
}

func (s *Store) HealthCheck() error {
	keyBytes := append(s.getBucket(), []byte("_health_check_kvx_")...)
	return s.db.Put(keyBytes, []byte("ok"), nil)
}

func (s *Store) Close() error {
	return s.db.Close()
}
