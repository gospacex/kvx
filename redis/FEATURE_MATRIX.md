# Redis 驱动功能矩阵

> 代码依据：`redis.go`、`config.go`、`register.go`（kvx/redis）
> 更新日期：2026-09-09 | 驱动注册名：`redis` | 客户端库：`github.com/go-redis/redis/v8`

## 1. 基础命令矩阵（KVStore 接口）

| 接口方法 | 底层命令 | 状态 | 行为说明 |
|---|---|---|---|
| `Get(ctx, key)` | `GET` | ✅ 支持 | Miss 返回 `("", nil)`（吞掉 `redis.Nil`） |
| `Set(ctx, key, val, ttl)` | `SET` | ✅ 支持 | `ttl <= 0` 时归零为**永不过期** |
| `Delete(ctx, key)` | `DEL` | ✅ 支持 | 删除不存在的 key 不报错 |
| `Exists(ctx, key)` | `EXISTS` | ✅ 支持 | 返回 `count > 0` |
| `Keys(ctx, pattern)` | `KEYS` | ✅ 支持 | ⚠️ O(N) 阻塞命令，生产慎用 |
| `MGet(ctx, keys...)` | `MGET` | ✅ 支持 | 返回 map；不存在的 key 不进入 map（与 Redis MGET 语义不同） |
| `MSet(ctx, pairs)` | `MSET` | ✅ 支持 | 原子批量写入，任意 key 失败整体回滚 |
| `TTL(ctx, key)` | `TTL` | ✅ 支持 | 不存在返回 `-2`，永不过期返回 `-1` |
| `Incr(ctx, key)` | `INCR` | ✅ 支持 | 非整数或不存在时返回错误 |
| `Decr(ctx, key)` | `DECR` | ✅ 支持 | 非整数或不存在时返回错误 |
| `HealthCheck()` | `PING` | ✅ 支持 | 带独立 span |
| `Close()` | 连接池关闭 | ✅ 支持 | — |

### OTel 追踪属性

| 属性 | 值 |
|---|---|
| span name | `kv.redis.{COMMAND}` |
| `db.system` | `redis` |
| `db.redis.command` | `GET` / `SET` / `DEL` / `MGET` / `MSET` / `INCR` / `DECR` / `TTL` / `KEYS` / `PING` |

## 2. 部署拓扑矩阵

| 模式 | 状态 | 接线方式 |
|---|---|---|
| 单机 Standalone | ✅ 支持 | `cfg.Addr` 直连 |
| Sentinel 哨兵 | ✅ 支持 | `cfg.SentinelAddrs != ""` → `NewFailoverClient` |
| Cluster 集群 | ✅ 支持 | `cfg.Cluster != ""` → `NewClusterClient` |
| TLS | ✅ 支持 | `cfg.TLSEnabled` → `MinVersion: TLS1.2` |
| 连接重试 | ✅ 支持 | `New()` Ping 失败，3 次退避重试 |

> Redis 是 kvx 10 个驱动中**唯一**同时支持单机/Sentinel/Cluster 三模式的驱动。

## 3. 配置字段生效状态（kvx.Config）

| 字段 | 状态 | 说明 |
|---|---|---|
| `Addr` | ✅ 生效 | 单机地址；Sentinel/Cluster 模式不使用 |
| `Addrs` | ✅ 生效 | 逗号分隔多地址（Standalone 多节点） |
| `Password` | ✅ 生效 | 三种模式均传递 |
| `DB` | ✅ 生效 | 仅单机/Sentinel 有效 |
| `PoolSize` | ✅ 生效 | 默认 10 |
| `MinIdleConn` | ✅ 生效 | 默认 0 |
| `Timeout` | ✅ 生效 | 三超时合一，默认 5s |
| `TLSEnabled` | ✅ 生效 | — |
| `Cluster` | ✅ 生效 | 非空字符串即走 Cluster 分支 |
| `EnableTrace` | ⚠️ 部分 | 字段存入但未被消费，span 始终上报 |
| `Name` / `Driver` | ✅ 生效 | — |

## 4. 扩展命令矩阵

| 命令 | 状态 | 说明 |
|---|---|---|
| `HGetAll / HSet / HMSet` | ❌ 不支持 | 接口未定义 Hash 方法 |
| `LPush / RPush / LPop / BRPop` | ❌ 不支持 | 列表类型未封装 |
| `ZAdd / ZRevRank / ZRange` | ❌ 不支持 | ZSet 类型未封装 |
| `PubSub Subscribe / Publish` | ❌ 不支持 | — |
| `Eval / EvalSHA (Lua)` | ❌ 不支持 | — |
| `MULTI / EXEC / WATCH` | ❌ 不支持 | 事务未封装 |
| `SCAN` 游标 | ❌ 不支持 | `Keys` 暴露，无安全 SCAN 接口 |

## 5. 企业级业务场景矩阵

| 业务场景 | 支持状态 | 实现方式与约束 |
|---|---|---|
| KV 缓存层 | ✅ 完整支持 | Get/Set/Delete/Exists + TTL 全覆盖 |
| 会话存储（Session） | ✅ 完整支持 | Set TTL + Get；与 cachex 差异：TTL/Incr/Decr 均封装 |
| 分布式锁 | ⚠️ 可间接支持 | 无 Lua/SET NX 封装，需业务侧组合 TTL+Incr 实现 |
| 限流计数（Rate Limit） | ✅ 完整支持 | Incr + Set TTL 天然支持滑动窗口计数 |
| 热点数据缓存（Cache-Aside） | ✅ 完整支持 | Get/MSet/TTL 三件套 |
| 验证码 / OTP | ✅ 完整支持 | Set TTL + Get + Delete 三件套 |
| 消息队列 Offset 存储 | ✅ 完整支持 | MSet/MGet 原子批量偏移 |
| 分布式计数器 | ✅ 完整支持 | Incr/Decr 原子操作 |
| 排行榜（ZSet） | ❌ 不支持 | 接口未定义 Sorted Set 方法 |
| 发布订阅 | ❌ 不支持 | — |
| 延迟队列 | ❌ 不支持 | — |
| 批量读写高频场景 | ✅ 完整支持 | MGet/MSet 减少网络往返 |
| 多租户隔离（多 DB） | ✅ 支持 | `cfg.DB` 字段指定数据库号 |
| 健康探针 | ✅ 支持 | PING |
| 链路追踪 | ✅ 支持 | `kv.redis.*` span |
| 闪崩保护 | ❌ 不支持 | 无 singleflight 封装 |
| 缓存预热 | ✅ 支持 | MSet 批量写入 |

## 6. 已知限制与风险

1. **`EnableTrace` 开关失效**：字段存在但未读取，span 始终上报，需在 OTel Provider 侧控制采样。
2. **`Keys` 阻塞风险**：与 cachex/redis 同病，大库慎用；建议业务侧使用 SCAN（暂未封装）。
3. **MGet 不返回 nil key**：不存在的 key 不进入返回值 map，需业务侧比对原始 keys 列表判断 Miss。
4. **TTL=0 永久语义**：与 kvx/memcached（默认 300s）不同，跨驱动切换需注意。
5. **单接口不可拆分**：MGet/MSet 原子性与单次操作原子性一致；部分场景需单独调用 Get/Set 绕过。

## 7. 快速接入示例

```go
import (
    "github.com/gospacex/kvx"
    _ "github.com/gospacex/kvx/redis" // 注册驱动
)

store, _ := kvx.ReadAndCreate("default") // 从 configx 读取 kvx 配置
// 或使用：
stores, _ := kvx.New([]*kvx.Config{{
    Name: "default", Driver: "redis",
    Addr: "127.0.0.1:6379", Password: "redis123456",
    PoolSize: 10, Timeout: 5 * time.Second,
}})
s := stores["default"] // kvx.KVStore
```
