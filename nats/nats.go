// kvx/nats - NATS JetStream KV 存储驱动
package nats

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gospacex/kvx/kvxbase"
	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"github.com/gospacex/traceflag"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Store struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	kv     jetstream.KeyValue
	name   string
	tracer *traceflag.Tracer
	logger logx.Logger
}

func NewNats(cfg *kvxbase.Config) (*Store, error) {
	cfg.FillDefaults()

	servers := []string{cfg.Addr}
	for _, a := range cfg.Addrs {
		a = strings.TrimSpace(a)
		if a != "" && a != cfg.Addr {
			servers = append(servers, a)
		}
	}

	opts := []nats.Option{nats.Timeout(cfg.Timeout)}
	if cfg.Password != "" {
		opts = append(opts, nats.UserInfo("user", cfg.Password))
	}

	nc, err := nats.Connect(strings.Join(servers, ","), opts...)
	if err != nil {
		return nil, fmt.Errorf("kvx/nats: connect failed: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("kvx/nats: jetstream failed: %w", err)
	}

	ctx := context.Background()
	kv, err := js.KeyValue(ctx, cfg.Name)
	if err != nil {
		kv, err = js.CreateKeyValue(ctx, jetstream.KeyValueConfig{
			Bucket:      cfg.Name,
			Description: "kvx store",
			Storage:     jetstream.FileStorage,
		})
		if err != nil {
			nc.Close()
			return nil, fmt.Errorf("kvx/nats: create kv bucket failed: %w", err)
		}
	}

	return &Store{
		nc:     nc,
		js:     js,
		kv:     kv,
		name:   cfg.Name,
		tracer: traceflag.NewTracer(cfg.EnableTrace, "github.com/gospacex/kvx/nats", "kvx"),
		logger: logx.GetGlobal(),
	}, nil
}

func (s *Store) Get(ctx context.Context, key string) (string, error) {
	entry, err := s.kv.Get(ctx, key)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return "", nil
		}
		return "", fmt.Errorf("kvx/nats get(%q): %w", key, err)
	}
	return string(entry.Value()), nil
}

func (s *Store) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	_, err := s.kv.Put(ctx, key, []byte(value))
	if err != nil {
		return fmt.Errorf("kvx/nats set(%q): %w", key, err)
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	if err := s.kv.Delete(ctx, key); err != nil {
		return fmt.Errorf("kvx/nats delete(%q): %w", key, err)
	}
	return nil
}

func (s *Store) Exists(ctx context.Context, key string) (bool, error) {
	_, err := s.kv.Get(ctx, key)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return false, nil
		}
		return false, fmt.Errorf("kvx/nats exists(%q): %w", key, err)
	}
	return true, nil
}

func (s *Store) Keys(ctx context.Context, pattern string) ([]string, error) {
	keys, err := s.kv.Keys(ctx)
	if err != nil {
		return nil, fmt.Errorf("kvx/nats keys: %w", err)
	}
	if pattern == "" || pattern == "*" {
		return keys, nil
	}
	var result []string
	for _, k := range keys {
		if strings.HasPrefix(k, pattern) {
			result = append(result, k)
		}
	}
	return result, nil
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
	_, err := s.kv.Get(ctx, key)
	if err != nil {
		if err == jetstream.ErrKeyNotFound {
			return 0, nil
		}
		return 0, fmt.Errorf("kvx/nats ttl(%q): %w", key, err)
	}
	return 0, nil
}

func (s *Store) Incr(ctx context.Context, key string) (int64, error) {
	entry, err := s.kv.Get(ctx, key)
	var current int64
	if err == nil {
		fmt.Sscanf(string(entry.Value()), "%d", &current)
	}
	current++
	_, err = s.kv.Put(ctx, key, []byte(fmt.Sprintf("%d", current)))
	if err != nil {
		return 0, fmt.Errorf("kvx/nats incr(%q): %w", key, err)
	}
	return current, nil
}

func (s *Store) Decr(ctx context.Context, key string) (int64, error) {
	entry, err := s.kv.Get(ctx, key)
	var current int64
	if err == nil {
		fmt.Sscanf(string(entry.Value()), "%d", &current)
	}
	current--
	_, err = s.kv.Put(ctx, key, []byte(fmt.Sprintf("%d", current)))
	if err != nil {
		return 0, fmt.Errorf("kvx/nats decr(%q): %w", key, err)
	}
	return current, nil
}

func (s *Store) HealthCheck() error {
	if s.nc.Status() == nats.CONNECTED {
		return nil
	}
	return fmt.Errorf("kvx/nats: not connected")
}

func (s *Store) Close() error {
	s.nc.Close()
	return nil
}
