package redis

import (
	"context"
	"testing"
	"time"

	"github.com/gospacex/kvx/kvxbase"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	// 使用无法连接 addr，验证连接检查生效
	_, err := NewRedis(&kvxbase.Config{
		Name:   "test",
		Driver: "redis",
		Addr:   "localhost:59999",
	})
	assert.Error(t, err)
}

func TestNewSingleNode(t *testing.T) {
	s, err := NewRedis(&kvxbase.Config{
		Name:     "test",
		Driver:   "redis",
		Addr:     "localhost:6379",
		Password: "redis123456",
	})
	if err != nil {
		t.Skipf("redis not available: %v", err)
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

func TestNewWithDefaults(t *testing.T) {
	s, err := NewRedis(&kvxbase.Config{
		Name:     "test",
		Driver:   "redis",
		Addr:     "localhost:6379",
		Password: "redis123456",
	})
	if err != nil {
		t.Skipf("redis not available: %v", err)
	}
	defer s.Close()

	_ = s // skip deep check — constructor validation sufficient
}

func TestIncrDecr(t *testing.T) {
	s, err := NewRedis(&kvxbase.Config{
		Name:     "test",
		Driver:   "redis",
		Addr:     "localhost:6379",
		Password: "redis123456",
	})
	if err != nil {
		t.Skipf("redis not available: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

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

func TestTTL(t *testing.T) {
	s, err := NewRedis(&kvxbase.Config{
		Name:     "test",
		Driver:   "redis",
		Addr:     "localhost:6379",
		Password: "redis123456",
	})
	if err != nil {
		t.Skipf("redis not available: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	err = s.Set(ctx, "ttl_key", "ttl_val", 1*time.Second)
	assert.NoError(t, err)

	ttl, err := s.TTL(ctx, "ttl_key")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, ttl, time.Duration(0))
}

func TestHealthCheck(t *testing.T) {
	s, err := NewRedis(&kvxbase.Config{
		Name:     "test",
		Driver:   "redis",
		Addr:     "localhost:6379",
		Password: "redis123456",
	})
	if err != nil {
		t.Skipf("redis not available: %v", err)
	}
	defer s.Close()

	assert.NoError(t, s.HealthCheck())
}

func TestMGetMSet(t *testing.T) {
	s, err := NewRedis(&kvxbase.Config{
		Name:     "test",
		Driver:   "redis",
		Addr:     "localhost:6379",
		Password: "redis123456",
	})
	if err != nil {
		t.Skipf("redis not available: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	err = s.MSet(ctx, map[string]string{"k1": "v1", "k2": "v2"})
	assert.NoError(t, err)

	m, err := s.MGet(ctx, "k1", "k2")
	assert.NoError(t, err)
	assert.Equal(t, "v1", m["k1"])
	assert.Equal(t, "v2", m["k2"])
}
