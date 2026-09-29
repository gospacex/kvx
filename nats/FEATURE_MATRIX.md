# NATS JetStream KV 驱动功能矩阵

> 代码依据：`nats.go`、`config.go`、`register.go`（kvx/nats）
> 更新日期：2026-09-09 | 驱动注册名：`nats` | 客户端库：`github.com/nats-io/nats.go`

## 1. 基础命令矩阵（KVStore 接口）

| 接口方法 | 底层实现 | 状态 | 行为说明 |
|---|---|---|---|
| `Get(ctx, key)` | JetStream KV `Get(key)` | ✅ 支持 | 未找到返回 `("", nil)` |
| `Set(ctx, key, val, ttl)` | JetStream KV `Put(key, val)` | ✅ 支持 | TTL 通过 JetStream 过期策略实现 |
| `Delete(ctx, key)` | JetStream KV `Delete(key)` | ✅ 支持 | 软删除，保留历史版本 |
| `Exists(ctx, key)` | JetStream KV `Get(key)` + error check | ✅ 支持 | — |
| `Keys(ctx, pattern)` | JetStream KV `Keys(pattern)` | ✅ 支持 | 通过 prefix 过滤 |
| `MGet(ctx, keys...)` | 顺序调用 `Get` | ⚠️ 部分 | 非原子批量，逐一 Get |
| `MSet(ctx, pairs)` | 顺序调用 `Put` | ⚠️ 部分 | 非原子批量，逐一 Put |
| `TTL(ctx, key)` | JetStream meta `TTL(key)` | ✅ 支持 | 读取键 TTL |
| `Incr(ctx, key)` | — | ❌ 不支持 | JetStream KV 无原子自增语义 |
| `Decr(ctx, key)` | — | ❌ 不支持 | — |
| `HealthCheck()` | `JetStream.Connected()` | ✅ 支持 | — |
| `Close()` | NATS 连接关闭 | ✅ 支持 | — |

### OTel 追踪属性

| 属性 | 值 |
|---|---|
| span name | `kv.nats.{COMMAND}` |
| `db.system` | `nats` |

## 2. 部署拓扑矩阵

| 模式 | 状态 | 说明 |
|---|---|---|
| 单机 NATS | ✅ 支持 | `cfg.Addr` 直连 |
| NATS Cluster | ⚠️ 部分 | `Addrs` 多地址支持，但 JetStream KV store 需服务端提前配置 |
| TLS | ✅ 支持 | `TLSEnabled` → NATS TLS config |
| SASL/NKey | ⚠️ 部分 | `Password` 支持 Basic Auth，NKey 未封装 |
| 连接重试 | ✅ 支持 | NATS 客户端内置自动重连 |

## 3. 配置字段生效状态（kvx.Config）

| 字段 | 状态 | 说明 |
|---|---|---|
| `Addr` | ✅ 生效 | 单机地址 |
| `Addrs` | ✅ 生效 | 多节点地址 |
| `Password` | ✅ 生效 | Basic Auth（NATS 用户密码） |
| `DB` | ❌ 不适用 | NATS 无 DB 概念，使用 JetStream domain 模拟 |
| `PoolSize` | ⚠️ 部分 | NATS 连接池由客户端内部管理 |
| `Timeout` | ✅ 生效 | — |
| `TLSEnabled` | ✅ 生效 | — |
| `EnableTrace` | ⚠️ 部分 | span 始终上报 |
| `Name` / `Driver` | ✅ 生效 | — |

## 4. 扩展命令矩阵

| 命令 | 状态 | 说明 |
|---|---|---|
| History（键历史版本） | ✅ 支持 | JetStream KV 原生支持，KVStore 未封装但可直连 |
| Purge（硬删除） | ✅ 支持 | JetStream KV 原生支持，KVStore 未封装 |
| Create（CAS 创建） | ✅ 支持 | JetStream revision 机制 |
| Update（CAS 更新） | ✅ 支持 | — |
| Watch（键监听） | ✅ 支持 | JetStream KV Watch |
| Pub/Sub 消息 | ⚠️ 部分 | JetStream 支持，但 KVStore 接口无此方法 |

## 5. 企业级业务场景矩阵

| 业务场景 | 支持状态 | 实现方式与约束 |
|---|---|---|
| KV 缓存层 | ✅ 完整支持 | 带持久化，重启不丢 |
| 会话存储 | ✅ 完整支持 | TTL + 持久化 |
| 分布式锁 | ⚠️ 可间接支持 | 利用 Revision CAS 实现，但未封装 |
| 限流计数 | ⚠️ 需扩展 | 无 Incr/Decr，需业务侧自旋 Read-Modify-Write |
| 热点数据缓存 | ✅ 完整支持 | — |
| 验证码 / OTP | ✅ 完整支持 | TTL 自动过期 |
| 配置中心 KV | ✅ 优势场景 | 持久化 + Watch 监听变更 |
| 事件溯源键值存储 | ✅ 优势场景 | History/Purge 原生支持 |
| 排行榜 | ❌ 不支持 | — |
| 发布订阅 | ✅ 支持 | JetStream 原生 Pub/Sub（非 KVStore 接口） |
| 健康探针 | ✅ 支持 | Connected 检查 |
| 链路追踪 | ✅ 支持 | `kv.nats.*` span |

## 6. 已知限制与风险

1. **Incr/Decr 不支持**：JetStream KV 无原子自增原语，限流计数等场景需业务侧自旋。
2. **MGet/MSet 非原子**：顺序调用 Get/Put，并发场景存在竞争。
3. **Delete 是软删除**：历史版本保留；需硬删除（Purge）时需扩展接口。
4. **Key 命名空间隔离**：NATS 用 Bucket/Domain 模拟 DB，多租户需独立 Bucket。
5. **Keys 性能**：pattern 扫描在大 store 下有性能影响，建议业务侧缓存键列表。

## 7. 快速接入示例

```go
import (
    "github.com/gospacex/kvx"
    _ "github.com/gospacex/kvx/nats" // 注册驱动
)

stores, _ := kvx.New([]*kvx.Config{{
    Name: "default", Driver: "nats",
    Addr: "127.0.0.1:4222",
    Password: "nats123456",
    PoolSize: 10, Timeout: 5 * time.Second,
}})
s := stores["default"] // kvx.KVStore
```
