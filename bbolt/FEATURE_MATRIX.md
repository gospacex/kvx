# BBolt 驱动功能矩阵

> 代码依据：`bbolt.go`、`config.go`、`register.go`（kvx/bbolt）
> 更新日期：2026-09-09 | 驱动注册名：`bbolt` | 客户端库：`go.etcd.io/bbolt`

## 1. 基础命令矩阵（KVStore 接口）

| 接口方法 | 底层实现 | 状态 | 行为说明 |
|---|---|---|---|
| `Get(ctx, key)` | `bucket.Get(key)` | ✅ 支持 | Miss 返回 `("", nil)` |
| `Set(ctx, key, val, ttl)` | `bucket.Put(key, val)` | ✅ 支持 | TTL 通过 value 元数据包装实现（非原生 TTL） |
| `Delete(ctx, key)` | `bucket.Delete(key)` | ✅ 支持 | — |
| `Exists(ctx, key)` | `bucket.Get(key)` + error check | ✅ 支持 | — |
| `Keys(ctx, pattern)` | `bucket.ForEach(func(k,v))` + 过滤 | ✅ 支持 | ⚠️ 全桶扫描，大数据集内存开销大 |
| `MGet(ctx, keys...)` | 顺序调用 `Get` | ⚠️ 部分 | 非原子批量 |
| `MSet(ctx, pairs)` | 事务批量写入 | ✅ 支持 | 通过 `tx.Update()` 原子写入 |
| `TTL(ctx, key)` | 读取 value 中 TTL 元数据 | ✅ 支持 | 需业务侧配合封装 TTL 元数据 |
| `Incr(ctx, key)` | — | ❌ 不支持 | BBolt 无原子自增 |
| `Decr(ctx, key)` | — | ❌ 不支持 | — |
| `HealthCheck()` | `bbolt.DB.IsReadOnly()` | ✅ 支持 | — |
| `Close()` | `bbolt.DB.Close()` | ✅ 支持 | 刷新后关闭 |

### OTel 追踪属性

| 属性 | 值 |
|---|---|
| span name | `kv.bbolt.{COMMAND}` |
| `db.system` | `bbolt` |

## 2. 部署拓扑矩阵

| 模式 | 状态 | 说明 |
|---|---|---|
| 嵌入式单节点 | ✅ 支持 | BBolt 内嵌于进程，文件级持久化 |
| 网络服务 | ❌ 不支持 | 无服务端模式 |
| 持久化 | ✅ 支持 | 单文件存储，崩溃安全 |
| 高可用 | ❌ 不支持 | 无复制/集群能力 |
| 并发读写 | ✅ 支持 | BBolt 支持多读单写，写操作串行化 |

## 3. 配置字段生效状态（kvx.Config）

| 字段 | 状态 | 说明 |
|---|---|---|
| `Addr` | ⚠️ 部分 | 映射为本地文件路径 |
| `Password` | ❌ 不适用 | 文件权限控制 |
| `DB` | ❌ 不适用 | 每个 BBolt DB 文件即一个 DB |
| `PoolSize` | ❌ 不适用 | 无连接池 |
| `Timeout` | ✅ 生效 | 事务超时 |
| `TLSEnabled` | ❌ 不适用 | — |
| `EnableTrace` | ⚠️ 部分 | span 始终上报 |
| `Name` / `Driver` | ✅ 生效 | — |

## 4. 扩展命令矩阵

| 命令 | 状态 | 说明 |
|---|---|---|
| Bucket 子命名空间 | ✅ 支持 | BBolt 原生支持多级 bucket，KVStore 未封装 |
| Prefix Scan | ✅ 支持 | `cursor.Seek(prefix)` + 遍历 |
| 事务 | ✅ 支持 | `bbolt.DB.Update/View` 原生事务 |
| 备份/快照 | ⚠️ 部分 | `bbolt.DB.Copy()` 可用，KVStore 未封装 |

## 5. 企业级业务场景矩阵

| 业务场景 | 支持状态 | 实现方式与约束 |
|---|---|---|
| 嵌入式本地存储 | ✅ 优势场景 | 零外部依赖 |
| 配置中心本地副本 | ✅ 完整支持 | 持久化 + 事务 |
| 单进程 Key-Value 缓存 | ✅ 完整支持 | 文件级持久化 |
| 审计日志存储 | ✅ 支持 | 追加写模式友好 |
| 会话存储 | ⚠️ 部分 | TTL 通过元数据实现，不原生 |
| 限流计数 | ❌ 不支持 | 无原子自增 |
| 热点数据缓存 | ⚠️ 部分 | 文件 IO 比内存慢，适合低频访问 |
| 多租户隔离 | ⚠️ 部分 | 通过 BBolt bucket 实现，非独立实例 |
| 健康探针 | ✅ 支持 | IsReadOnly 检查 |
| 链路追踪 | ✅ 支持 | `kv.bbolt.*` span |

## 6. 已知限制与风险

1. **Incr/Decr 不支持**：BBolt 无原子计数原语。
2. **Keys 全量扫描**：大 bucket 下性能差，内存消耗高。
3. **无网络协议**：嵌入式引擎，仅单进程使用。
4. **TTL 非原生**：通过 value 元数据模拟，读取时需解析元数据，增加复杂度。
5. **写放大**：BBolt append-only log-structured，频繁更新同 key 会产生文件增长。
6. **文件锁**：同一文件不可被多个进程同时打开（写锁互斥）。

## 7. 快速接入示例

```go
import (
    "github.com/gospacex/kvx"
    _ "github.com/gospacex/kvx/bbolt" // 注册驱动
)

stores, _ := kvx.New([]*kvx.Config{{
    Name: "default", Driver: "bbolt",
    Addr: "/tmp/bbolt-data.db", // 本地数据库文件
    Timeout: 5 * time.Second,
}})
s := stores["default"] // kvx.KVStore
```
