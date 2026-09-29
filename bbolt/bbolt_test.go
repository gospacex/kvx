// kvx/bbolt - BBolt KV 存储驱动测试
package bbolt

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gospacex/kvx/kvxbase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) (*Store, func()) {
	t.Helper()
	dir := t.TempDir()
	cfg := &kvxbase.Config{
		Name: "bbolt_test",
		Addr: filepath.Join(dir, "test.db"),
	}
	s, err := NewBBolt(cfg)
	require.NoError(t, err)
	return s, func() { s.Close() }
}

func TestNew(t *testing.T) {
	s, close := newTestStore(t)
	defer close()
	assert.NotNil(t, s)
}

func TestGetSet(t *testing.T) {
	s, close := newTestStore(t)
	defer close()

	require.NoError(t, s.Set(context.Background(), "k", "v", 0))
	val, err := s.Get(context.Background(), "k")
	require.NoError(t, err)
	assert.Equal(t, "v", val)
}

func TestGetNotFound(t *testing.T) {
	s, close := newTestStore(t)
	defer close()

	val, err := s.Get(context.Background(), "missing")
	require.NoError(t, err)
	assert.Equal(t, "", val)
}

func TestDelete(t *testing.T) {
	s, close := newTestStore(t)
	defer close()

	require.NoError(t, s.Set(context.Background(), "k", "v", 0))
	require.NoError(t, s.Delete(context.Background(), "k"))

	val, err := s.Get(context.Background(), "k")
	require.NoError(t, err)
	assert.Equal(t, "", val)
}

func TestExists(t *testing.T) {
	s, close := newTestStore(t)
	defer close()

	found, err := s.Exists(context.Background(), "nope")
	require.NoError(t, err)
	assert.False(t, found)

	require.NoError(t, s.Set(context.Background(), "yes", "v", 0))
	found, err = s.Exists(context.Background(), "yes")
	require.NoError(t, err)
	assert.True(t, found)
}

func TestKeys(t *testing.T) {
	s, close := newTestStore(t)
	defer close()

	require.NoError(t, s.Set(context.Background(), "a:1", "v1", 0))
	require.NoError(t, s.Set(context.Background(), "a:2", "v2", 0))
	require.NoError(t, s.Set(context.Background(), "b:1", "v3", 0))

	keys, err := s.Keys(context.Background(), "a:")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"a:1", "a:2"}, keys)
}

func TestMGetMSet(t *testing.T) {
	s, close := newTestStore(t)
	defer close()

	require.NoError(t, s.MSet(context.Background(), map[string]string{"x": "1", "y": "2"}))
	m, err := s.MGet(context.Background(), "x", "y", "z")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"x": "1", "y": "2"}, m)
}

func TestIncrDecr(t *testing.T) {
	s, close := newTestStore(t)
	defer close()

	v, err := s.Incr(context.Background(), "cnt")
	require.NoError(t, err)
	assert.Equal(t, int64(1), v)

	v, err = s.Incr(context.Background(), "cnt")
	require.NoError(t, err)
	assert.Equal(t, int64(2), v)

	v, err = s.Decr(context.Background(), "cnt")
	require.NoError(t, err)
	assert.Equal(t, int64(1), v)
}

func TestDecrUnderflow(t *testing.T) {
	s, close := newTestStore(t)
	defer close()

	_, err := s.Decr(context.Background(), "zero")
	require.Error(t, err)
}

func TestHealthCheck(t *testing.T) {
	s, close := newTestStore(t)
	defer close()
	require.NoError(t, s.HealthCheck())
}

func TestOverwrite(t *testing.T) {
	s, close := newTestStore(t)
	defer close()

	require.NoError(t, s.Set(context.Background(), "k", "old", 0))
	require.NoError(t, s.Set(context.Background(), "k", "new", 0))

	val, err := s.Get(context.Background(), "k")
	require.NoError(t, err)
	assert.Equal(t, "new", val)
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
