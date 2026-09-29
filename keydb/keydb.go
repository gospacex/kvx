// kvx/keydb - KeyDB KV 存储驱动
// KeyDB 与 Redis 协议完全兼容，使用 go-redis/v8 UniversalClient
package keydb

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gospacex/kvx/kvxbase"
	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"github.com/gospacex/traceflag"
)

// Store KeyDB KVStore 实现
type Store struct {
	client redis.UniversalClient
	name   string
	tracer *traceflag.Tracer
	logger logx.Logger
}

func NewKeyDB(cfg *kvxbase.Config) (*Store, error) {
	cfg.FillDefaults()

	var client redis.UniversalClient
	switch {
	case cfg.Cluster != "":
		client = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        splitComma(cfg.Cluster),
			Password:     cfg.Password,
			PoolSize:     cfg.PoolSize,
			MinIdleConns: cfg.MinIdleConn,
			DialTimeout:  cfg.Timeout,
			ReadTimeout:  cfg.Timeout,
			WriteTimeout: cfg.Timeout,
			TLSConfig:    tlsConfig(cfg.TLSEnabled),
		})
	default:
		addr := cfg.Addr
		if addr == "" {
			addr = "localhost:6380"
		}
		client = redis.NewClient(&redis.Options{
			Addr:         addr,
			Password:     cfg.Password,
			PoolSize:     cfg.PoolSize,
			MinIdleConns: cfg.MinIdleConn,
			DialTimeout:  cfg.Timeout,
			ReadTimeout:  cfg.Timeout,
			WriteTimeout: cfg.Timeout,
			TLSConfig:    tlsConfig(cfg.TLSEnabled),
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("kvx/keydb: connect failed: %w", err)
	}

	return &Store{
		client: client,
		name:   cfg.Name,
		tracer: traceflag.NewTracer(cfg.EnableTrace, "github.com/gospacex/kvx/keydb", "kvx"),
		logger: logx.GetGlobal(),
	}, nil
}

func (s *Store) Get(ctx context.Context, key string) (string, error) {
	val, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

func (s *Store) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return s.client.Set(ctx, key, value, ttl).Err()
}

func (s *Store) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}

func (s *Store) Exists(ctx context.Context, key string) (bool, error) {
	count, err := s.client.Exists(ctx, key).Result()
	return count > 0, err
}

func (s *Store) Keys(ctx context.Context, pattern string) ([]string, error) {
	return s.client.Keys(ctx, pattern).Result()
}

func (s *Store) MGet(ctx context.Context, keys ...string) (map[string]string, error) {
	if len(keys) == 0 {
		return map[string]string{}, nil
	}
	cmds, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(keys))
	for i, key := range keys {
		if cmds[i] != nil {
			if v, ok := cmds[i].(string); ok {
				result[key] = v
			}
		}
	}
	return result, nil
}

func (s *Store) MSet(ctx context.Context, pairs map[string]string) error {
	if len(pairs) == 0 {
		return nil
	}
	args := make([]interface{}, 0, len(pairs)*2)
	for k, v := range pairs {
		args = append(args, k, v)
	}
	return s.client.MSet(ctx, args...).Err()
}

func (s *Store) TTL(ctx context.Context, key string) (time.Duration, error) {
	return s.client.TTL(ctx, key).Result()
}

func (s *Store) Incr(ctx context.Context, key string) (int64, error) {
	return s.client.Incr(ctx, key).Result()
}

func (s *Store) Decr(ctx context.Context, key string) (int64, error) {
	return s.client.Decr(ctx, key).Result()
}

func (s *Store) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return s.client.Ping(ctx).Err()
}

func (s *Store) Close() error {
	return s.client.Close()
}

func splitComma(s string) []string {
	result := make([]string, 0)
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			if part := s[start:i]; part != "" {
				result = append(result, part)
			}
			start = i + 1
		}
	}
	return result
}

func tlsConfig(enabled bool) *tls.Config {
	if !enabled {
		return nil
	}
	return &tls.Config{MinVersion: tls.VersionTLS12}
}
