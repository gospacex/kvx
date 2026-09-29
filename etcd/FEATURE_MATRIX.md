# etcd 驱动功能矩阵

> 代码依据：`etcd.go`、`config.go`、`register.go`（kvx/etcd）
> 更新日期：2026-09-09 | 驱动注册名：`etcd` | 客户端库：`go.etcd.io/etcd/client/v3`

## 1. 基础命令矩阵（KVStore 接口）

| 接口方法 | 底层实现 | 状态 | 行为说明 |
|---|---|---|---|
| `Get(ctx, key)` | `client.KV.Get(ctx, key)` | ✅ 支持 | 未找到返回 `("", nil)`（`etcdserver.ErrKeyNotFound` 吞掉） |
| `Set(ctx, key, val, ttl)` | `client.KV.Put(ctx, key, val)` + Lease | ✅ 支持 | TTL 通过 Lease grant 实现，租约到期自动删除 |
| `Delete(ctx, key)` | `client.KV.Delete(ctx, key)` | ✅ 支持 | 删除不存在的 key 返回 success=true（无错误） |
| `Exists(ctx, key)` | `client.KV.Get(ctx, key)` + 检查 count | ✅ 支持 | — |
| `Keys(ctx, pattern)` | `client.KV.Get(ctx, prefix, withPrefix)` | ✅ 支持 | 前缀扫描，返回匹配 key 列表 |
| `MGet(ctx, keys...)` | 顺序调用 `Get` | ⚠️ 部分 | 非原子批量；可用 `withSerializable` 串行读 |
| `MSet(ctx, pairs)` | `client.Txn()` 事务 | ✅ 支持 | 通过 Txn If/Then 实现原子批量写入 |
| `TTL(ctx, key)` | `client.TimeToLive(ctx, leaseID, true)` | ✅ 支持 | 读取租约剩余 TTL |
| `Incr(ctx, key)` | — | ❌ 不支持 | etcd 无原子自增，需 Compare-and-Swap 模拟 |
| `Decr(ctx, key)` | — | ❌ 不支持 | — |
| `HealthCheck()` | `client.MemberList(ctx)` | ✅ 支持 | 检查集群可达性 |
| `Close()` | 客户端关闭 | ✅ 支持 | — |

### OTel 追踪属性

| 属性 | 值 |
|---|---|
| span name | `kv.etcd.{COMMAND}` |
| `db.system` | `etcd` |

## 2. 部署拓扑矩阵

| 模式 | 状态 | 说明 |
|---|---|---|
| 单机 etcd | ✅ 支持 | `cfg.Addr` 单地址 |
| 多节点 Cluster | ✅ 支持 | `cfg.Addrs` 多地址，客户端自动负载均衡 |
| TLS/mTLS | ✅ 支持 | `cfg.TLSEnabled` → ETCDCTL_CLIENT_CERT/KEY |
| Auth 认证 | ✅ 支持 | `cfg.Username`/`cfg.Password` 传递 |
| 连接重试 | ✅ 支持 | etcd 客户端内置自动重连 |
| Lease 自动续期 | ✅ 支持 | Put 时绑定 Lease，客户端自动 keepalive |

## 3. 配置字段生效状态（kvx.Config）

| 字段 | 状态 | 说明 |
|---|---|---|
| `Addr` | ✅ 生效 | 单节点地址 |
| `Addrs` | ✅ 生效 | 多节点地址（Cluster 模式） |
| `Password` | ✅ 生效 | Username/Password 认证 |
| `DB` | ❌ 不适用 | etcd 无 DB 概念，用 prefix 隔离 |
| `PoolSize` | ❌ 不适用 | gRPC 连接池由客户端管理 |
| `Timeout` | ✅ 生效 | 操作超时，默认 5s |
| `TLSEnabled` | ✅ 生效 | — |
| `EnableTrace` | ⚠️ 部分 | span 始终上报 |
| `Name` / `Driver` | ✅ 生效 | — |

## 4. 扩展命令矩阵

| 命令 | 状态 | 说明 |
|---|---|---|
| Watch 监听 | ✅ 支持 | etcd 原生 Watch，KVStore 未封装但可直连 |
| Lease Grant/Revoke | ✅ 支持 | 原生 Lease 管理 |
| Txn 事务 | ✅ 支持 | Compare-and-Swap 原子操作 |
| Range 扫描 | ✅ 支持 | 前缀扫描 + 分页 |
| KeepAlive | ✅ 支持 | Lease 续期 |

## 5. 企业级业务场景矩阵

| 业务场景 | 支持状态 | 实现方式与约束 |
|---|---|---|
| KV 缓存层 | ✅ 完整支持 | 持久化 + Watch 监听变更 |
| 会话存储 | ✅ 完整支持 | Lease TTL 自动过期 |
| 分布式锁 | ✅ 完整支持 | Lease + Compare 原生支持 |
| 配置中心 | ✅ 优势场景 | Watch 实时监听配置变更 |
| 服务发现 | ✅ 支持 | Lease + Watch 天然适合 |
| 限流计数 | ⚠️ 可间接支持 | 通过 Txn CAS 模拟，非原子 |
| 验证码 / OTP | ✅ 完整支持 | Lease TTL 自动失效 |
| 热点数据缓存 | ✅ 完整支持 | — |
| 排行榜 | ❌ 不支持 | — |
| 持久化需求 | ✅ 优势场景 | etcd 持久化到 WAL，重启不丢 |
| 多租户隔离 | ✅ 支持 | Prefix 隔离命名空间 |
| 健康探针 | ✅ 支持 | MemberList 检查 |
| 链路追踪 | ✅ 支持 | `kv.etcd.*` span |

## 6. 已知限制与风险

1. **Incr/Decr 不支持**：etcd 无原子计数原语，CAS 模拟非原子。
2. **MGet 非原子**：顺序 Get，高并发下可能读到不一致结果。
3. **Lease 开销**：每个 Set 操作绑定 Lease，高频写入时 Lease 续约有额外 gRPC 流量。
4. **Keys 性能**：前缀扫描返回全量匹配 key，大 key 空间下性能受限。
5. **值大小限制**：etcd 单值最大 1.5MB，超出需分片存储。
6. **集群依赖**：etcd 本身是强一致集群，部署运维复杂度高于 Redis。

## 7. 快速接入示例

```go
import (
    "github.com/gospacex/kvx"
    _ "github.com/gospacex/kvx/etcd" // 注册驱动
)

stores, _ := kvx.New([]*kvx.Config{{
    Name: "default", Driver: "etcd",
    Addrs: []string{"127.0.0.1:2379"},
    Password: "etcd123456",
    Timeout: 5 * time.Second,
}})
s := stores["default"] // kvx.KVStore
```
