# Dragonfly 驱动功能矩阵

> 代码依据：`dragonfly.go`、`config.go`、`register.go`（kvx/dragonfly）
> 更新日期：2026-09-09 | 驱动注册名：`dragonfly` | 客户端库：`github.com/go-redis/redis/v8`（DragonflyDB 兼容 Redis 协议）

## 1. 基础命令矩阵（KVStore 接口）

| 接口方法 | 底层命令 | 状态 | 行为说明 |
|---|---|---|---|
| `Get(ctx, key)` | `GET` | ✅ 支持 | Miss 返回 `("", nil)` |
| `Set(ctx, key, val, ttl)` | `SET` | ✅ 支持 | `ttl <= 0` 归零为**永不过期** |
| `Delete(ctx, key)` | `DEL` | ✅ 支持 | — |
| `Exists(ctx, key)` | `EXISTS` | ✅ 支持 | 返回 `count > 0` |
| `Keys(ctx, pattern)` | `KEYS` | ✅ 支持 | Dragonfly 对 KEYS 有快照优化，仍需谨慎 |
| `MGet(ctx, keys...)` | `MGET` | ✅ 支持 | 返回 map，不存在 key 不进 map |
| `MSet(ctx, pairs)` | `MSET` | ✅ 支持 | 原子批量写入 |
| `TTL(ctx, key)` | `TTL` | ✅ 支持 | 不存在返回 `-2`，永不过期返回 `-1` |
| `Incr(ctx, key)` | `INCR` | ✅ 支持 | — |
| `Decr(ctx, key)` | `DECR` | ✅ 支持 | — |
| `HealthCheck()` | `PING` | ✅ 支持 | — |
| `Close()` | 连接池关闭 | ✅ 支持 | — |

## 2. 部署拓扑矩阵

| 模式 | 状态 | 说明 |
|---|---|---|
| 单机 Standalone | ✅ 支持 | `cfg.Addr` 直连 |
| Cluster 集群 | ❌ 接线未生效 | `config.go` 有 `ClusterMode/Shards/Replicas` 字段，但 `New()` 仅 `redis.NewClient` 单连接，`cfg.Cluster` 被忽略 |
| TLS | ✅ 支持 | `TLSEnabled` → `MinVersion: TLS1.2` |
| 连接重试 | ❌ 无 | Ping 失败立即报错（对比 Redis 驱动 3 次退避） |

> ⚠️ `ToClusterConfig` 设置的 `cfg.Cluster="true"` 无人消费，集群配置是**摆设**。

## 3. 配置字段生效状态（kvx.Config）

| 字段 | 状态 | 说明 |
|---|---|---|
| `Addr` | ✅ 生效 | 单机地址 |
| `Password` | ✅ 生效 | — |
| `DB` | ✅ 生效 | — |
| `PoolSize` | ✅ 生效 | 默认 10 |
| `Timeout` | ✅ 生效 | 三超时合一，默认 5s |
| `TLSEnabled` | ✅ 生效 | — |
| `Cluster` | ❌ 不生效 | `New()` 无集群分支 |
| `EnableTrace` | ⚠️ 部分 | 字段存在但未读取，span 始终上报 |
| `Name` / `Driver` | ✅ 生效 | — |

## 4. 扩展命令矩阵

与 kvx/redis 一致（共享 go-redis 底层），Hash/List/ZSet/PubSub 均未在 KVStore 接口封装。

## 5. 企业级业务场景矩阵

| 业务场景 | 支持状态 | 实现方式与约束 |
|---|---|---|
| 高性能 KV 缓存 | ✅ 优势场景 | Dragonfly 多线程架构，吞吐优于 Redis |
| 会话存储 | ✅ 完整支持 | — |
| 限流计数 | ✅ 完整支持 | Incr/Decr |
| 验证码 / OTP | ✅ 完整支持 | — |
| 热点数据缓存 | ✅ 完整支持 | — |
| 排行榜 | ❌ 不支持 | 接口无 Sorted Set |
| 持久化 | ⚠️ 部分 | Dragonfly 支持 AOF/RDB，但驱动未封装相关命令 |
| 健康探针 | ✅ 支持 | PING |
| 链路追踪 | ✅ 支持 | `kv.dragonfly.*` span |

## 6. 已知限制与风险

1. **集群配置是摆设**：与 cachex/dragonfly 同病——`cfg.Cluster` 无人消费。
2. **初始化无重试**：Ping 失败立即报错，容器启动顺序敏感。
3. **`Keys` 虽优化仍需谨慎**：建议业务侧封装 SCAN。
4. **`EnableTrace` 开关失效**。
5. **TTL=0 永久语义**：与 kvx/memcached（300s）不同。

## 7. 快速接入示例

```go
import (
    "github.com/gospacex/kvx"
    _ "github.com/gospacex/kvx/dragonfly" // 注册驱动
)

stores, _ := kvx.New([]*kvx.Config{{
    Name: "default", Driver: "dragonfly",
    Addr: "127.0.0.1:6381", Password: "dragonfly123456",
    PoolSize: 10, Timeout: 5 * time.Second,
}})
s := stores["default"] // kvx.KVStore
```
