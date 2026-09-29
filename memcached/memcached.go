// kvx/memcached - Memcached KV 存储驱动
package memcached

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/gospacex/kvx/kvxbase"
	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"github.com/gospacex/traceflag"
)

type Store struct {
	mc     *memcache.Client
	name   string
	tracer *traceflag.Tracer
	logger logx.Logger
}

func NewMemcached(cfg *kvxbase.Config) (*Store, error) {
	cfg.FillDefaults()

	addrs := splitComma(cfg.Addr)
	for _, a := range cfg.Addrs {
		addrs = append(addrs, strings.TrimSpace(a))
	}
	if len(addrs) == 0 {
		addrs = []string{"localhost:11211"}
	}

	mc := memcache.New(addrs...)
	mc.Timeout = cfg.Timeout
	mc.MaxIdleConns = cfg.PoolSize

	return &Store{
		mc:     mc,
		name:   cfg.Name,
		tracer: traceflag.NewTracer(cfg.EnableTrace, "github.com/gospacex/kvx/memcached", "kvx"),
		logger: logx.GetGlobal(),
	}, nil
}

func (s *Store) Get(ctx context.Context, key string) (string, error) {
	item, err := s.mc.Get(key)
	if err != nil {
		if err == memcache.ErrCacheMiss {
			return "", nil
		}
		return "", fmt.Errorf("kvx/memcached get(%q): %w", key, err)
	}
	return string(item.Value), nil
}

func (s *Store) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	exp := int32(300)
	if ttl > 0 {
		exp = int32(ttl.Seconds())
	}
	return s.mc.Set(&memcache.Item{
		Key:        key,
		Value:      []byte(value),
		Expiration: exp,
	})
}

func (s *Store) Delete(ctx context.Context, key string) error {
	err := s.mc.Delete(key)
	if err != nil && err != memcache.ErrCacheMiss {
		return fmt.Errorf("kvx/memcached delete(%q): %w", key, err)
	}
	return nil
}

func (s *Store) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.mc.Get(key)
	if err != nil {
		if err == memcache.ErrCacheMiss {
			return false, nil
		}
		return false, fmt.Errorf("kvx/memcached exists(%q): %w", key, err)
	}
	return true, nil
}

func (s *Store) Keys(ctx context.Context, pattern string) ([]string, error) {
	return nil, fmt.Errorf("kvx/memcached: Keys not supported by memcached protocol")
}

func (s *Store) MGet(ctx context.Context, keys ...string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		val, err := s.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		result[key] = val
	}
	return result, nil
}

func (s *Store) MSet(ctx context.Context, pairs map[string]string) error {
	for key, value := range pairs {
		if err := s.Set(ctx, key, value, 0); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) TTL(ctx context.Context, key string) (time.Duration, error) {
	item, err := s.mc.Get(key)
	if err != nil {
		if err == memcache.ErrCacheMiss {
			return 0, nil
		}
		return 0, fmt.Errorf("kvx/memcached ttl(%q): %w", key, err)
	}
	return time.Duration(item.Expiration) * time.Second, nil
}

func (s *Store) Incr(ctx context.Context, key string) (int64, error) {
	n, err := s.mc.Increment(key, uint64(1))
	if err != nil {
		return 0, fmt.Errorf("kvx/memcached incr(%q): %w", key, err)
	}
	return int64(n), nil
}

func (s *Store) Decr(ctx context.Context, key string) (int64, error) {
	n, err := s.mc.Decrement(key, uint64(1))
	if err != nil {
		return 0, fmt.Errorf("kvx/memcached decr(%q): %w", key, err)
	}
	return int64(n), nil
}

func (s *Store) HealthCheck() error {
	_, err := s.mc.Get("_health_check_kvx_")
	if err != nil {
		if err == memcache.ErrCacheMiss {
			return nil
		}
		return fmt.Errorf("kvx/memcached health check failed: %w", err)
	}
	return nil
}

func (s *Store) Close() error {
	return nil
}

func splitComma(s string) []string {
	result := make([]string, 0)
	for _, part := range strings.Split(s, ",") {
		if part := strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}
