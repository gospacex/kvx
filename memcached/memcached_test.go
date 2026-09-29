package memcached

import (
	"context"
	"testing"
	"time"

	"github.com/gospacex/kvx/kvxbase"
	"github.com/stretchr/testify/assert"
)

func TestNewDefault(t *testing.T) {
	_, err := NewMemcached(&kvxbase.Config{
		Name:   "test",
		Driver: "memcached",
		Addr:   "localhost:59999",
	})
	assert.Error(t, err)
}

func TestMemcachedIntegration(t *testing.T) {
	s, err := NewMemcached(&kvxbase.Config{
		Name:   "test",
		Driver: "memcached",
		Addr:   "localhost:11211",
	})
	if err != nil {
		t.Skipf("memcached not available: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	err = s.Set(ctx, "k1", "v1", 0)
	assert.NoError(t, err)

	val, err := s.Get(ctx, "k1")
	assert.NoError(t, err)
	assert.Equal(t, "v1", val)

	exists, err := s.Exists(ctx, "k1")
	assert.NoError(t, err)
	assert.True(t, exists)

	err = s.Delete(ctx, "k1")
	assert.NoError(t, err)

	exists, err = s.Exists(ctx, "k1")
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestIncrDecr(t *testing.T) {
	s, err := NewMemcached(&kvxbase.Config{
		Name:   "test",
		Driver: "memcached",
		Addr:   "localhost:11211",
	})
	if err != nil {
		t.Skipf("memcached not available: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// 确保 key 不存在以便 Incr 正常工作
	_ = s.Delete(ctx, "counter")

	n, err := s.Incr(ctx, "counter")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), n)

	n, err = s.Incr(ctx, "counter")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), n)

	n, err = s.Decr(ctx, "counter")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), n)
}

func TestHealthCheck(t *testing.T) {
	s, err := NewMemcached(&kvxbase.Config{
		Name:   "test",
		Driver: "memcached",
		Addr:   "localhost:11211",
	})
	if err != nil {
		t.Skipf("memcached not available: %v", err)
	}
	defer s.Close()

	assert.NoError(t, s.HealthCheck())
}

func TestKeysNotSupported(t *testing.T) {
	s, err := NewMemcached(&kvxbase.Config{
		Name:   "test",
		Driver: "memcached",
		Addr:   "localhost:11211",
	})
	if err != nil {
		t.Skipf("memcached not available: %v", err)
	}
	defer s.Close()

	_, err = s.Keys(context.Background(), "*")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

func TestFillDefaults(t *testing.T) {
	cfg := &kvxbase.Config{
		Name:   "test",
		Driver: "memcached",
	}
	cfg.FillDefaults()
	assert.Equal(t, 5*time.Second, cfg.Timeout)
	assert.Equal(t, 10, cfg.PoolSize)
}
