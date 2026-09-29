# Badger 驱动功能矩阵

> 代码依据：`badger.go`、`config.go`、`register.go`（kvx/badger）
> 更新日期：2026-09-09 | 驱动注册名：`badger` | 客户端库：`github.com/dgraph-io/badger/v4`

## 1. 基础命令矩阵（KVStore 接口）

| 接口方法 | 底层实现 | 状态 | 行为说明 |
|---|---|---|---|
| `Get(ctx, key)` | `badger.DB.Get(key)` | ✅ 支持 | Miss 返回 `("", nil)` |
| `Set(ctx, key, val, ttl)` | `badger.DB.Update(setEntry)` | ✅ 支持 | TTL 映射到 Badger `WithTTL` |
| `Delete(ctx, key)` | `badger.DB.Update(delete)` | ✅ 支持 | 逻辑删除（标记 tombstone） |
| `Exists(ctx, key)` | `badger.DB.Get(key)` + error check | ✅ 支持 | — |
| `Keys(ctx, pattern)` | `badger.DB.ReadAll()` + 过滤 | ✅ 支持 | ⚠️ 全量扫描，大数据集内存开销大 |
| `MGet(ctx, keys...)` | 顺序调用 `Get` | ⚠️ 部分 | 非原子批量 |
| `MSet(ctx, pairs)` | 事务批量写入 | ✅ 支持 | 通过 `badger.DB.Update(txn)` 实现原子写入 |
| `TTL(ctx, key)` | `badger.Entry.Value()` 读取 TTL 元信息 | ✅ 支持 | 通过 value 包裹 TTL 元数据实现 |
| `Incr(ctx, key)` | — | ❌ 不支持 | Badger 无原子自增原语 |
| `Decr(ctx, key)` | — | ❌ 不支持 | — |
| `HealthCheck()` | `badger.DB.IsClosed()` | ✅ 支持 | — |
| `Close()` | `badger.DB.Close()` | ✅ 支持 | 刷新 WAL 后关闭 |

### OTel 追踪属性

| 属性 | 值 |
|---|---|
| span name | `kv.badger.{COMMAND}` |
| `db.system` | `badger` |

## 2. 部署拓扑矩阵

| 模式 | 状态 | 说明 |
|---|---|---|
| 嵌入式单节点 | ✅ 支持 | Badger 内嵌于进程，无外部服务依赖 |
| 网络服务 | ❌ 不支持 | Badger 为嵌入式引擎，无服务端模式 |
| 持久化 | ✅ 支持 | 默认落盘 RocksDB/LMDB（可配置） |
| 高可用 | ❌ 不支持 | 无复制/集群能力，需业务侧自建 |

## 3. 配置字段生效状态（kvx.Config）

| 字段 | 状态 | 说明 |
|---|---|---|
| `Addr` | ⚠️ 部分 | 映射为本地目录路径（Badger 无需网络地址） |
| `Password` | ❌ 不适用 | 嵌入式，无认证 |
| `DB` | ❌ 不适用 | 每个 Badger 实例即一个 DB |
| `PoolSize` | ❌ 不适用 | 无连接池概念 |
| `Timeout` | ✅ 生效 | 事务超时，默认 5s |
| `TLSEnabled` | ❌ 不适用 | — |
| `EnableTrace` | ⚠️ 部分 | span 始终上报 |
| `Name` / `Driver` | ✅ 生效 | — |

## 4. 扩展命令矩阵

| 命令 | 状态 | 说明 |
|---|---|---|
| Prefix Scan | ✅ 支持 | 通过 `Keys()` 过滤模拟，或扩展接口直连 |
| Iterator 遍历 | ✅ 支持 | Badger 原生迭代器，KVStore 未封装 |
| Merge 操作 | ⚠️ 部分 | Badger 支持 `Merge`，KVStore 未封装 |
| 事务 | ✅ 支持 | `badger.DB.Update(txn)` 原生事务 |

## 5. 企业级业务场景矩阵

| 业务场景 | 支持状态 | 实现方式与约束 |
|---|---|---|
| 嵌入式 KV 存储 | ✅ 优势场景 | 无外部依赖，零运维 |
| 本地缓存层 | ✅ 完整支持 | 持久化缓存，进程重启不丢 |
| 配置中心本地副本 | ✅ 完整支持 | 持久化 + 快速读取 |
| 日志/事件存储 | ✅ 支持 | 高写入吞吐，压缩落盘 |
| 会话存储 | ⚠️ 部分 | TTL 支持，但 Incr 不可用限制部分场景 |
| 限流计数 | ❌ 不支持 | 无原子自增 |
| 热点数据缓存 | ✅ 支持 | 内存映射，读取极快 |
| 排行榜 | ❌ 不支持 | 无排序结构 |
| 多租户隔离 | ⚠️ 部分 | 通过 Badger 多实例实现，非单实例多 DB |
| 健康探针 | ✅ 支持 | IsClosed 检查 |
| 链路追踪 | ✅ 支持 | `kv.badger.*` span |

## 6. 已知限制与风险

1. **Incr/Decr 不支持**：Badger 无原子自增，限流计数等场景需业务自旋。
2. **Keys 全量扫描**：大数据集下内存压力大，建议业务侧维护独立键清单。
3. **无网络协议**：嵌入式引擎，无法跨进程/跨机器共享，不适合分布式场景。
4. **无 TTL 自动清理**：Badger GC 按 compaction 周期清理 tombstone，过期键不立即消失。
5. **单写多读**：Badger 写优化 LSM Tree，高并发写场景需注意写放大。

## 7. 快速接入示例

```go
import (
    "github.com/gospacex/kvx"
    _ "github.com/gospacex/kvx/badger" // 注册驱动
)

stores, _ := kvx.New([]*kvx.Config{{
    Name: "default", Driver: "badger",
    Addr: "/tmp/badger-data", // 本地数据目录
    Timeout: 5 * time.Second,
}})
s := stores["default"] // kvx.KVStore
```
