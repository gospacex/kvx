// kvx/badger - Badger KV 存储驱动
package badger

import (
	"context"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/gospacex/kvx/kvxbase"
	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"github.com/gospacex/traceflag"
)

type Store struct {
	db     *badger.DB
	name   string
	tracer *traceflag.Tracer
	logger logx.Logger
}

func NewBadger(cfg *kvxbase.Config) (*Store, error) {
	cfg.FillDefaults()

	dir := cfg.Addr
	if dir == "" {
		dir = "./data/badger/" + cfg.Name
	}

	opts := badger.DefaultOptions(dir).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("kvx/badger: open failed: %w", err)
	}

	return &Store{
		db:     db,
		name:   cfg.Name,
		tracer: traceflag.NewTracer(cfg.EnableTrace, "github.com/gospacex/kvx/badger", "kvx"),
		logger: logx.GetGlobal(),
	}, nil
}

func (s *Store) Get(ctx context.Context, key string) (string, error) {
	var val string
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		b, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		val = string(b)
		return nil
	})
	if err == badger.ErrKeyNotFound {
		return "", nil
	}
	return val, err
}

func (s *Store) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	entry := badger.NewEntry([]byte(key), []byte(value))
	if ttl > 0 {
		entry = entry.WithTTL(ttl)
	}
	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.SetEntry(entry)
	})
	if err != nil {
		return fmt.Errorf("kvx/badger set: %w", err)
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
	if err != nil {
		return fmt.Errorf("kvx/badger delete: %w", err)
	}
	return nil
}

func (s *Store) Exists(ctx context.Context, key string) (bool, error) {
	val, err := s.Get(ctx, key)
	if err != nil {
		return false, err
	}
	return val != "", nil
}

func (s *Store) Keys(ctx context.Context, prefix string) ([]string, error) {
	var keys []string
	prefixBytes := []byte(prefix)
	err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek([]byte(prefix)); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			if prefix == "" || len(k) >= len(prefixBytes) && string(k[:len(prefixBytes)]) == prefix {
				keys = append(keys, string(k))
			} else if len(k) > len(prefixBytes) {
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("kvx/badger keys: %w", err)
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
	return s.db.Update(func(txn *badger.Txn) error {
		for key, value := range pairs {
			if err := txn.Set([]byte(key), []byte(value)); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) TTL(ctx context.Context, key string) (time.Duration, error) {
	var ttl time.Duration
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		expiresAt := item.ExpiresAt()
		if expiresAt == 0 {
			ttl = 0
		} else {
			ttl = time.Until(time.Unix(int64(expiresAt), 0))
			if ttl < 0 {
				ttl = 0
			}
		}
		return nil
	})
	if err != nil {
		return -1, err
	}
	return ttl, nil
}

func (s *Store) Incr(ctx context.Context, key string) (int64, error) {
	var result int64
	err := s.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		var current int64
		if err == nil {
			val, _ := item.ValueCopy(nil)
			fmt.Sscanf(string(val), "%d", &current)
		}
		newVal := fmt.Sprintf("%d", current+1)
		if err := txn.Set([]byte(key), []byte(newVal)); err != nil {
			return err
		}
		result = current + 1
		return nil
	})
	return result, err
}

func (s *Store) Decr(ctx context.Context, key string) (int64, error) {
	var result int64
	err := s.db.Update(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		var current int64
		if err == nil {
			val, _ := item.ValueCopy(nil)
			fmt.Sscanf(string(val), "%d", &current)
		}
		newVal := fmt.Sprintf("%d", current-1)
		if err := txn.Set([]byte(key), []byte(newVal)); err != nil {
			return err
		}
		result = current - 1
		return nil
	})
	return result, err
}

func (s *Store) HealthCheck() error {
	return s.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte("_health_check_kvx_"), []byte("ok"))
	})
}

func (s *Store) Close() error {
	return s.db.Close()
}
