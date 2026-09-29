# Memcached 驱动功能矩阵

> 代码依据：`memcached.go`、`config.go`、`register.go`（kvx/memcached）
> 更新日期：2026-09-09 | 驱动注册名：`memcached` | 客户端库：`github.com/bradfitz/gomemcache/memcache`

## 1. 基础命令矩阵（KVStore 接口）

| 接口方法 | 底层命令 | 状态 | 行为说明 |
|---|---|---|---|
| `Get(ctx, key)` | `get` | ✅ 支持 | Miss 返回 `("", nil)` |
| `Set(ctx, key, val, ttl)` | `set` | ✅ 支持 | ⚠️ `ttl <= 0` 时默认 **300 秒**（与 Redis 系"永不过期"语义**相反**） |
| `Delete(ctx, key)` | `delete` | ✅ 支持 | Miss 静默成功 |
| `Exists(ctx, key)` | `get`（模拟） | ✅ 支持 | GET 取值后丢弃，多一次反序列化 |
| `Keys(ctx, pattern)` | — | ❌ 不支持 | 直接返回错误："Keys not supported" |
| `MGet(ctx, keys...)` | — | ❌ 不支持 | gomemcache 无原生 MGET；模拟为顺序 GET，非原子 |
| `MSet(ctx, pairs)` | — | ❌ 不支持 | 无原生 MSET；模拟为顺序 SET，非原子 |
| `TTL(ctx, key)` | `get`（模拟） | ✅ 支持 | 读取元数据后 TTL；⚠️ 需额外协议扩展（当前通过自定义过期字段模拟） |
| `Incr(ctx, key)` | `incr` | ✅ 支持 | 整数 key 自增；不存在或非整数返回错误 |
| `Decr(ctx, key)` | `decr` | ✅ 支持 | 整数 key 自减；不存在或非整数返回错误 |
| `HealthCheck()` | `get _health_check_` | ✅ 支持 | 用 GET Miss 模拟心跳 |
| `Close()` | — | ✅ 支持 | 空实现 |

### OTel 追踪属性

| 属性 | 值 |
|---|---|
| span name | `kv.memcached.{COMMAND}` |
| `db.system` | `memcached` |

## 2. 部署拓扑矩阵

| 模式 | 状态 | 说明 |
|---|---|---|
| 单节点 | ✅ 支持 | `cfg.Addr` 单地址直连 |
| 多节点客户端分片 | ✅ 支持 | `cfg.Addr` 逗号分隔，gomemcache 内置一致性哈希 |
| TLS | ❌ 不支持 | gomemcache 本身不支持 TLS，需外挂 proxy（mcrouter/stunnel） |
| SASL 认证 | ❌ 未接线 | `Password` 字段存在但未传给客户端 |
| 连接池 | ✅ 支持 | `PoolSize` 映射 `MaxIdleConns` |

## 3. 配置字段生效状态（kvx.Config）

| 字段 | 状态 | 说明 |
|---|---|---|
| `Addr` | ✅ 生效 | 逗号分隔多节点 |
| `Addrs` | ✅ 生效 | 同上 |
| `Password` | ❌ 不生效 | SASL 未接线 |
| `DB` | ❌ 不适用 | Memcached 无 DB 概念 |
| `PoolSize` | ✅ 生效 | 映射 `MaxIdleConns` |
| `MinIdleConn` | ⚠️ 部分 | 字段存在，实际优先级低于 `PoolSize` |
| `Timeout` | ✅ 生效 | 默认 5s |
| `TLSEnabled` | ❌ 不生效 | — |
| `Cluster` | ❌ 不生效 | 字段被忽略（gomemcache 多地址已实现分片） |
| `EnableTrace` | ⚠️ 部分 | span 始终上报 |
| `Name` / `Driver` | ✅ 生效 | — |

## 4. 扩展命令矩阵

| 命令 | 状态 | 说明 |
|---|---|---|
| `HGetAll / HSet` | ❌ 不支持 | 协议无 Hash 类型 |
| `LPush / RPush / LPop` | ❌ 不支持 | 协议无 List 类型 |
| `ZAdd / ZRange` | ❌ 不支持 | 协议无 ZSet 类型 |
| `PubSub` | ❌ 不支持 | — |
| `Lua / Eval` | ❌ 不支持 | — |
| `Transactions` | ❌ 不支持 | — |

## 5. 企业级业务场景矩阵

| 业务场景 | 支持状态 | 实现方式与约束 |
|---|---|---|
| 高性能 KV 缓存 | ✅ 优势场景 | 多线程模型，无 Redis 单线程瓶颈 |
| 会话存储 | ✅ 支持 | Set TTL + Get；注意 TTL ≤ 0 变 300s |
| 限流计数 | ✅ 支持 | Incr/Decr 原子操作 |
| 验证码 / OTP | ✅ 支持 | 短 TTL 场景语义正确 |
| 批量读写 | ⚠️ 部分 | MGet/MSet 模拟非原子，高并发下可接受但不保证 |
| 分布式计数器 | ✅ 支持 | Incr/Decr |
| 热点数据缓存 | ✅ 完整支持 | Get/Set/Delete + TTL |
| 排行榜 / 集合 | ❌ 不支持 | 协议无结构类型 |
| 持久化需求 | ❌ 不支持 | 纯内存，重启丢数据 |
| 健康探针 | ✅ 支持 | GET Miss 模拟心跳 |
| 链路追踪 | ✅ 支持 | `kv.memcached.*` span |
| 云托管/带认证实例 | ❌ 不支持 | SASL/TLS 均未接线 |

## 6. 已知限制与风险

1. **TTL 语义跨驱动不一致（最关键）**：`ttl <= 0` 时 memcached=300s、redis/dragonfly/keydb/etcd=永久。通用封装层必须显式传 TTL。
2. **Keys 不可用**：接口契约在此驱动上"合法破坏"——KVStore.Keys 返回错误。
3. **MGet/MSet 非原子**：模拟实现为多次顺序操作，高并发下不能替代原子的 Redis MGET/MSET。
4. **SASL/TLS 缺失**：云托管 Memcached（如 AWS ElastiCache 加密传输）无法直接接入。
5. **健康检查产生 Miss 流量**：`_health_check_` 每次 GET 均有网络开销，可接受但需知悉。
6. **TTL 30 天上限**：超过 30 天需转为 unix 绝对时间戳，当前未做此转换。

## 7. 快速接入示例

```go
import (
    "github.com/gospacex/kvx"
    _ "github.com/gospacex/kvx/memcached" // 注册驱动
)

stores, _ := kvx.New([]*kvx.Config{{
    Name: "default", Driver: "memcached",
    Addr: "127.0.0.1:11211", // 多节点: "10.0.0.1:11211,10.0.0.2:11211"
    PoolSize: 10, Timeout: 5 * time.Second,
}})
s := stores["default"] // kvx.KVStore
// 注意：Set 必须显式传 TTL，ttl=0 → 300s 默认
```
