package keydb

import (
	"context"
	"testing"

	"github.com/gospacex/kvx/kvxbase"
	"github.com/stretchr/testify/assert"
)

func TestKeyDBIntegration(t *testing.T) {
	s, err := NewKeyDB(&kvxbase.Config{
		Name:     "test",
		Driver:   "keydb",
		Addr:     "localhost:6380",
		Password: "keydb123456",
	})
	if err != nil {
		t.Skipf("keydb not available: %v", err)
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

func TestKeyDBHealthCheck(t *testing.T) {
	s, err := NewKeyDB(&kvxbase.Config{
		Name:     "test",
		Driver:   "keydb",
		Addr:     "localhost:6380",
		Password: "keydb123456",
	})
	if err != nil {
		t.Skipf("keydb not available: %v", err)
	}
	defer s.Close()

	assert.NoError(t, s.HealthCheck())
}
