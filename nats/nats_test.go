package nats

import (
	"context"
	"testing"

	"github.com/gospacex/kvx/kvxbase"
	"github.com/stretchr/testify/assert"
)

func TestNatsIntegration(t *testing.T) {
	s, err := NewNats(&kvxbase.Config{
		Name:   "test-nats",
		Driver: "nats",
		Addr:   "localhost:4222",
	})
	if err != nil {
		t.Skipf("nats not available: %v", err)
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

func TestNatsHealthCheck(t *testing.T) {
	s, err := NewNats(&kvxbase.Config{
		Name:   "test-nats",
		Driver: "nats",
		Addr:   "localhost:4222",
	})
	if err != nil {
		t.Skipf("nats not available: %v", err)
	}
	defer s.Close()

	assert.NoError(t, s.HealthCheck())
}
