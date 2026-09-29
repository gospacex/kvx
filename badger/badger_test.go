// kvx/badger - Badger KV 存储驱动测试
package badger

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gospacex/kvx/kvxbase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) (*Store, func()) {
	t.Helper()
	dir := t.TempDir()
	cfg := &kvxbase.Config{
		Name: "badger_test",
		Addr: filepath.Join(dir, "badger"),
	}
	s, err := NewBadger(cfg)
	require.NoError(t, err)
	return s, func() { s.Close() }
}

func TestNew(t *testing.T) {
	s, close := newTestStore(t)
	defer close()
	assert.NotNil(t, s)
	assert.Equal(t, "badger_test", s.name)
}

func TestGetSet(t *testing.T) {
	s, close := newTestStore(t)
	defer close()

	err := s.Set(context.Background(), "key1", "value1", 0)
	require.NoError(t, err)

	val, err := s.Get(context.Background(), "key1")
	require.NoError(t, err)
	assert.Equal(t, "value1", val)
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

	err := s.Set(context.Background(), "k", "v", 0)
	require.NoError(t, err)

	err = s.Delete(context.Background(), "k")
	require.NoError(t, err)

	val, err := s.Get(context.Background(), "k")
	require.NoError(t, err)
	assert.Equal(t, "", val)
}

func TestExists(t *testing.T) {
	s, close := newTestStore(t)
	defer close()

	exists, err := s.Exists(context.Background(), "nope")
	require.NoError(t, err)
	assert.False(t, exists)

	err = s.Set(context.Background(), "yes", "v", 0)
	require.NoError(t, err)

	exists, err = s.Exists(context.Background(), "yes")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestKeys(t *testing.T) {
	s, close := newTestStore(t)
	defer close()

	pairs := map[string]string{
		"user:1:name": "alice",
		"user:2:name": "bob",
		"order:1:id":  "o1",
	}
	for k, v := range pairs {
		require.NoError(t, s.Set(context.Background(), k, v, 0))
	}

	keys, err := s.Keys(context.Background(), "user:")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"user:1:name", "user:2:name"}, keys)
}

func TestMGetMSet(t *testing.T) {
	s, close := newTestStore(t)
	defer close()

	err := s.MSet(context.Background(), map[string]string{
		"a": "1", "b": "2", "c": "3",
	})
	require.NoError(t, err)

	m, err := s.MGet(context.Background(), "a", "b", "c", "missing")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"a": "1", "b": "2", "c": "3"}, m)
}

func TestTTL(t *testing.T) {
	s, close := newTestStore(t)
	defer close()

	err := s.Set(context.Background(), "k", "v", 5*time.Second)
	require.NoError(t, err)

	// Badger uses internal TTL but KVStore TTL is best-effort
	_, err = s.TTL(context.Background(), "k")
	require.NoError(t, err)
}

func TestIncrDecr(t *testing.T) {
	s, close := newTestStore(t)
	defer close()

	v, err := s.Incr(context.Background(), "counter")
	require.NoError(t, err)
	assert.Equal(t, int64(1), v)

	v, err = s.Incr(context.Background(), "counter")
	require.NoError(t, err)
	assert.Equal(t, int64(2), v)

	v, err = s.Decr(context.Background(), "counter")
	require.NoError(t, err)
	assert.Equal(t, int64(1), v)
}

func TestHealthCheck(t *testing.T) {
	s, close := newTestStore(t)
	defer close()

	err := s.HealthCheck()
	require.NoError(t, err)
}

func TestClose(t *testing.T) {
	s, close := newTestStore(t)
	close()
	// double-close should not panic
	assert.NoError(t, s.Close())
}

func TestSetAndGetLargeValue(t *testing.T) {
	s, close := newTestStore(t)
	defer close()

	large := string(make([]byte, 1024*1024)) // 1MB
	err := s.Set(context.Background(), "big", large, 0)
	require.NoError(t, err)

	val, err := s.Get(context.Background(), "big")
	require.NoError(t, err)
	assert.Equal(t, large, val)
}

func TestOverwriteValue(t *testing.T) {
	s, close := newTestStore(t)
	defer close()

	require.NoError(t, s.Set(context.Background(), "k", "v1", 0))
	require.NoError(t, s.Set(context.Background(), "k", "v2", 0))

	val, err := s.Get(context.Background(), "k")
	require.NoError(t, err)
	assert.Equal(t, "v2", val)
}

func TestDeleteNonExistent(t *testing.T) {
	s, close := newTestStore(t)
	defer close()

	// should not error
	err := s.Delete(context.Background(), "missing")
	require.NoError(t, err)
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
