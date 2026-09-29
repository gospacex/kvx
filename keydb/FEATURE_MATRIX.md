# KeyDB 驱动功能矩阵

> 代码依据：`keydb.go`、`config.go`、`register.go`（kvx/keydb）
> 更新日期：2026-09-09 | 驱动注册名：`keydb` | 客户端库：`github.com/go-redis/redis/v8`（KeyDB 兼容 Redis 协议）

## 1. 基础命令矩阵（KVStore 接口）

| 接口方法 | 底层命令 | 状态 | 行为说明 |
|---|---|---|---|
| `Get(ctx, key)` | `GET` | ✅ 支持 | Miss 返回 `("", nil)` |
| `Set(ctx, key, val, ttl)` | `SET` | ✅ 支持 | `ttl <= 0` 归零为**永不过期** |
| `Delete(ctx, key)` | `DEL` | ✅ 支持 | — |
| `Exists(ctx, key)` | `EXISTS` | ✅ 支持 | — |
| `Keys(ctx, pattern)` | `KEYS` | ✅ 支持 | ⚠️ 多线程缓解但大库仍需谨慎 |
| `MGet(ctx, keys...)` | `MGET` | ✅ 支持 | — |
| `MSet(ctx, pairs)` | `MSET` | ✅ 支持 | — |
| `TTL(ctx, key)` | `TTL` | ✅ 支持 | — |
| `Incr(ctx, key)` | `INCR` | ✅ 支持 | — |
| `Decr(ctx, key)` | `DECR` | ✅ 支持 | — |
| `HealthCheck()` | `PING` | ✅ 支持 | — |
| `Close()` | 连接池关闭 | ✅ 支持 | — |

## 2. 部署拓扑矩阵

| 模式 | 状态 | 说明 |
|---|---|---|
| 单机 Standalone | ✅ 支持 | 唯一接线模式 |
| 多线程活性 | ✅ 服务端能力 | `--server-threads` 驱动无需感知 |
| Active-Active 复制 | ❌ 未接线 | KeyDB 多主复制需服务端配置，驱动仅单地址 |
| Cluster 分片 | ❌ 接线未生效 | `ClusterMode` 等字段存在但驱动不消费，集群配置是**摆设** |
| TLS | ✅ 支持 | `TLSEnabled` → `MinVersion: TLS1.2` |
| 连接重试 | ❌ 无 | Ping 失败立即报错 |

## 3. 配置字段生效状态（kvx.Config）

| 字段 | 状态 | 说明 |
|---|---|---|
| `Addr` | ✅ 生效 | 单机地址 |
| `Password` | ✅ 生效 | — |
| `DB` | ✅ 生效 | — |
| `PoolSize` | ✅ 生效 | 默认 10 |
| `Timeout` | ✅ 生效 | 默认 5s |
| `TLSEnabled` | ✅ 生效 | — |
| `Cluster` | ❌ 不生效 | `New()` 无集群分支 |
| `EnableTrace` | ⚠️ 部分 | 字段存在但未读取 |
| `Name` / `Driver` | ✅ 生效 | — |

## 4. 扩展命令矩阵

与 kvx/redis 一致（共享 go-redis 底层）。

## 5. 企业级业务场景矩阵

| 业务场景 | 支持状态 | 实现方式与约束 |
|---|---|---|
| 高性能 KV 缓存 | ✅ 优势场景 | 多线程无单线程瓶颈 |
| 会话存储 | ✅ 完整支持 | — |
| 限流计数 | ✅ 完整支持 | Incr/Decr |
| 热点数据缓存 | ✅ 完整支持 | — |
| 排行榜 | ❌ 不支持 | — |
| 持久化 | ⚠️ 部分 | KeyDB 支持 RDB 快照，驱动未封装持久化命令 |
| 多主容灾 | ⚠️ 部分 | 服务端支持，驱动单地址无法故障转移 |
| 健康探针 | ✅ 支持 | PING |
| 链路追踪 | ✅ 支持 | `kv.keydb.*` span |

## 6. 已知限制与风险

1. **集群配置是摆设**：与 Dragonfly 驱动同病。
2. **初始化无重试**：Ping 失败立即报错。
3. **`Keys` 阻塞风险**：多线程缓解但仍需慎用。
4. **`EnableTrace` 开关失效**。
5. **TTL=0 永久语义**：与 kvx/memcached（300s）不同。
6. **与 Dragonfly 驱动高度同构**：修复 Redis 协议系问题需四处同步。

## 7. 快速接入示例

```go
import (
    "github.com/gospacex/kvx"
    _ "github.com/gospacex/kvx/keydb" // 注册驱动
)

stores, _ := kvx.New([]*kvx.Config{{
    Name: "default", Driver: "keydb",
    Addr: "127.0.0.1:6380", Password: "keydb123456",
    PoolSize: 10, Timeout: 5 * time.Second,
}})
s := stores["default"] // kvx.KVStore
```
