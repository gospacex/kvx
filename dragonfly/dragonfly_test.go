package dragonfly

import (
	"context"
	"testing"
	"time"

	"github.com/gospacex/kvx/kvxbase"
	"github.com/stretchr/testify/assert"
)

func TestDragonflyIntegration(t *testing.T) {
	s, err := NewDragonfly(&kvxbase.Config{
		Name:   "test",
		Driver: "dragonfly",
		Addr:   "localhost:6381",
	})
	if err != nil {
		t.Skipf("dragonfly not available: %v", err)
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

func TestDragonflyHealthCheck(t *testing.T) {
	s, err := NewDragonfly(&kvxbase.Config{
		Name:   "test",
		Driver: "dragonfly",
		Addr:   "localhost:6381",
	})
	if err != nil {
		t.Skipf("dragonfly not available: %v", err)
	}
	defer s.Close()

	assert.NoError(t, s.HealthCheck())
}

func TestDragonflyIncrDecr(t *testing.T) {
	s, err := NewDragonfly(&kvxbase.Config{
		Name:   "test",
		Driver: "dragonfly",
		Addr:   "localhost:6381",
	})
	if err != nil {
		t.Skipf("dragonfly not available: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	n, err := s.Incr(ctx, "cnt")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), n)

	n, err = s.Decr(ctx, "cnt")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), n)
}

func TestDragonflyTTL(t *testing.T) {
	s, err := NewDragonfly(&kvxbase.Config{
		Name:   "test",
		Driver: "dragonfly",
		Addr:   "localhost:6381",
	})
	if err != nil {
		t.Skipf("dragonfly not available: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	err = s.Set(ctx, "ttl_k", "ttl_v", 1*time.Second)
	assert.NoError(t, err)

	ttl, err := s.TTL(ctx, "ttl_k")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, ttl, time.Duration(0))
}
