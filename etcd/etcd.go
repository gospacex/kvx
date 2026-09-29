// kvx/etcd - etcd KV 存储驱动（基于 go.etcd.io/etcd/client/v3）
package etcd

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gospacex/kvx/kvxbase"
	"github.com/gospacex/logx"
	_ "github.com/gospacex/logx/zap"
	"go.etcd.io/etcd/client/v3"
)

type Store struct {
	client *clientv3.Client
	lease  clientv3.LeaseID
	kv     clientv3.KV
	name   string

	mu          sync.Mutex
	keepaliveCh <-chan *clientv3.LeaseKeepAliveResponse
	done        chan struct{}
	closeOnce   sync.Once
	logger      logx.Logger
}

func NewStore(endpoints []string, leaseTTL int64, name string) (*Store, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("kvx/etcd: connect failed: %w", err)
	}

	s := &Store{
		client: client,
		kv:     clientv3.NewKV(client),
		name:   name,
		done:   make(chan struct{}),
		logger: logx.GetGlobal(),
	}

	if err := s.initLease(leaseTTL); err != nil {
		client.Close()
		return nil, err
	}

	return s, nil
}

func NewEtcd(cfg *kvxbase.Config) (kvxbase.KVStore, error) {
	return NewStoreFromConfig(cfg)
}

func NewStoreFromConfig(cfg *kvxbase.Config) (*Store, error) {
	cfg.FillDefaults()

	endpoints := []string{cfg.Addr}
	for _, a := range cfg.Addrs {
		a = trimSpace(a)
		if a != "" && a != cfg.Addr {
			endpoints = append(endpoints, a)
		}
	}

	leaseTTL := int64(60)
	if cfg.Timeout > 0 {
		leaseTTL = int64(cfg.Timeout.Seconds())
		if leaseTTL < 10 {
			leaseTTL = 10
		}
	}

	return NewStore(endpoints, leaseTTL, cfg.Name)
}

func trimSpace(s string) string {
	start := 0
	for start < len(s) && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	end := len(s)
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func (s *Store) initLease(ttl int64) error {
	leaseResp, err := s.client.Grant(context.Background(), ttl)
	if err != nil {
		return fmt.Errorf("kvx/etcd: grant lease failed: %w", err)
	}

	s.lease = leaseResp.ID
	s.keepaliveCh, err = s.client.KeepAlive(context.Background(), s.lease)
	if err != nil {
		return fmt.Errorf("kvx/etcd: keepalive failed: %w", err)
	}

	go s.keepaliveMonitor()
	return nil
}

func (s *Store) keepaliveMonitor() {
	for {
		select {
		case _, ok := <-s.keepaliveCh:
			if !ok {
				s.mu.Lock()
				if s.done == nil {
					s.mu.Unlock()
					return
				}
				ka, err := s.client.KeepAlive(context.Background(), s.lease)
				s.mu.Unlock()
				if err != nil {
					time.Sleep(time.Second)
					continue
				}
				s.keepaliveCh = ka
			}
		case <-s.done:
			return
		}
	}
}

func (s *Store) Get(ctx context.Context, key string) (string, error) {
	resp, err := s.kv.Get(ctx, key)
	if err != nil {
		return "", fmt.Errorf("kvx/etcd: get: %w", err)
	}
	if len(resp.Kvs) == 0 {
		return "", nil
	}
	return string(resp.Kvs[0].Value), nil
}

func (s *Store) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	opts := []clientv3.OpOption{}
	if ttl > 0 || s.lease != 0 {
		opts = append(opts, clientv3.WithLease(s.lease))
	}
	_, err := s.kv.Put(ctx, key, value, opts...)
	if err != nil {
		return fmt.Errorf("kvx/etcd: put: %w", err)
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	_, err := s.kv.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("kvx/etcd: delete: %w", err)
	}
	return nil
}

func (s *Store) Exists(ctx context.Context, key string) (bool, error) {
	resp, err := s.kv.Get(ctx, key)
	if err != nil {
		return false, fmt.Errorf("kvx/etcd: exists: %w", err)
	}
	return len(resp.Kvs) > 0, nil
}

func (s *Store) Keys(ctx context.Context, prefix string) ([]string, error) {
	resp, err := s.kv.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithKeysOnly())
	if err != nil {
		return nil, fmt.Errorf("kvx/etcd: keys: %w", err)
	}
	keys := make([]string, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		keys = append(keys, string(kv.Key))
	}
	return keys, nil
}

func (s *Store) MGet(ctx context.Context, keys ...string) (map[string]string, error) {
	result := make(map[string]string)
	for _, key := range keys {
		val, err := s.Get(ctx, key)
		if err != nil {
			continue
		}
		result[key] = val
	}
	return result, nil
}

func (s *Store) MSet(ctx context.Context, pairs map[string]string) error {
	for key, value := range pairs {
		if err := s.Set(ctx, key, value, 0); err != nil {
			return fmt.Errorf("kvx/etcd: mset: %w", err)
		}
	}
	return nil
}

func (s *Store) TTL(ctx context.Context, key string) (time.Duration, error) {
	resp, err := s.kv.Get(ctx, key, clientv3.WithPrevKV())
	if err != nil {
		return -1, fmt.Errorf("kvx/etcd: ttl get: %w", err)
	}
	if len(resp.Kvs) == 0 {
		return 0, nil
	}
	leaseID := resp.Kvs[0].Lease
	if leaseID == 0 {
		return 0, nil
	}
	ttlResp, err := s.client.TimeToLive(ctx, clientv3.LeaseID(leaseID))
	if err != nil {
		return -1, fmt.Errorf("kvx/etcd: ttl lookup: %w", err)
	}
	return time.Duration(ttlResp.TTL) * time.Second, nil
}

func (s *Store) Incr(ctx context.Context, key string) (int64, error) {
	for i := 0; i < 10; i++ {
		resp, err := s.kv.Get(ctx, key)
		if err != nil {
			return 0, err
		}
		var current int64
		if len(resp.Kvs) > 0 {
			fmt.Sscanf(string(resp.Kvs[0].Value), "%d", &current)
		}
		newVal := fmt.Sprintf("%d", current+1)
		opts := []clientv3.OpOption{}
		if s.lease != 0 {
			opts = append(opts, clientv3.WithLease(s.lease))
		}
		_, err = s.kv.Txn(ctx).If(clientv3.Compare(clientv3.ModRevision(key), "=", resp.Header.Revision-1)).Then(clientv3.OpPut(key, newVal, opts...)).Commit()
		if err == nil {
			return current + 1, nil
		}
	}
	return 0, fmt.Errorf("kvx/etcd: incr failed after retries")
}

func (s *Store) Decr(ctx context.Context, key string) (int64, error) {
	for i := 0; i < 10; i++ {
		resp, err := s.kv.Get(ctx, key)
		if err != nil {
			return 0, err
		}
		var current int64
		if len(resp.Kvs) > 0 {
			fmt.Sscanf(string(resp.Kvs[0].Value), "%d", &current)
		}
		newVal := fmt.Sprintf("%d", current-1)
		opts := []clientv3.OpOption{}
		if s.lease != 0 {
			opts = append(opts, clientv3.WithLease(s.lease))
		}
		_, err = s.kv.Txn(ctx).If(clientv3.Compare(clientv3.ModRevision(key), "=", resp.Header.Revision-1)).Then(clientv3.OpPut(key, newVal, opts...)).Commit()
		if err == nil {
			return current - 1, nil
		}
	}
	return 0, fmt.Errorf("kvx/etcd: decr failed after retries")
}

func (s *Store) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := s.client.Status(ctx, s.client.Endpoints()[0])
	return err
}

func (s *Store) Close() error {
	var closeErr error
	s.closeOnce.Do(func() {
		close(s.done)
		if s.lease != 0 {
			_, _ = s.client.Revoke(context.Background(), s.lease)
		}
		closeErr = s.client.Close()
	})
	return closeErr
}
