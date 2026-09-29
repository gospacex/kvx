# Pebble 驱动功能矩阵

> 代码依据：`pebble.go`、`config.go`、`register.go`（kvx/pebble）
> 更新日期：2026-09-09 | 驱动注册名：`pebble` | 客户端库：`github.com/cockroachdb/pebble`

## 1. 基础命令矩阵（KVStore 接口）

| 接口方法 | 底层实现 | 状态 | 行为说明 |
|---|---|---|---|
| `Get(ctx, key)` | `db.Get(key)` | ✅ 支持 | Miss 返回 `("", nil)` |
| `Set(ctx, key, val, ttl)` | `db.Set(key, val, nil)` | ✅ 支持 | TTL 通过 value 元数据包装实现 |
| `Delete(ctx, key)` | `db.Delete(key)` | ✅ 支持 | — |
| `Exists(ctx, key)` | `db.Has(key)` | ✅ 支持 | — |
| `Keys(ctx, pattern)` | `db.NewIter()` + 过滤 | ✅ 支持 | ⚠️ 全库扫描 |
| `MGet(ctx, keys...)` | 顺序调用 `Get` | ⚠️ 部分 | 非原子批量 |
| `MSet(ctx, pairs)` | 批量写入（batch） | ✅ 支持 | 原子批量 |
| `TTL(ctx, key)` | 读取 value 中 TTL 元数据 | ✅ 支持 | 需业务侧配合封装 |
| `Incr(ctx, key)` | — | ❌ 不支持 | Pebble 无原子自增 |
| `Decr(ctx, key)` | — | ❌ 不支持 | — |
| `HealthCheck()` | `db.Metrics()` | ✅ 支持 | — |
| `Close()` | `db.Close()` | ✅ 支持 | — |

### OTel 追踪属性

| 属性 | 值 |
|---|---|
| span name | `kv.pebble.{COMMAND}` |
| `db.system` | `pebble` |

## 2. 部署拓扑矩阵

| 模式 | 状态 | 说明 |
|---|---|---|
| 嵌入式单节点 | ✅ 支持 | Pebble 内嵌于进程 |
| 网络服务 | ❌ 不支持 | 无服务端模式 |
| 持久化 | ✅ 支持 | SStable 落盘，LSM Tree |
| 高可用 | ❌ 不支持 | 无复制/集群能力 |

## 3. 配置字段生效状态（kvx.Config）

| 字段 | 状态 | 说明 |
|---|---|---|
| `Addr` | ⚠️ 部分 | 映射为本地目录路径 |
| `Password` | ❌ 不适用 | — |
| `DB` | ❌ 不适用 | Pebble 单实例即一个 DB |
| `PoolSize` | ❌ 不适用 | 无连接池 |
| `Timeout` | ✅ 生效 | 操作超时 |
| `TLSEnabled` | ❌ 不适用 | — |
| `EnableTrace` | ⚠️ 部分 | span 始终上报 |
| `Name` / `Driver` | ✅ 生效 | — |

## 4. 扩展命令矩阵

| 命令 | 状态 | 说明 |
|---|---|---|
| Prefix Scan | ✅ 支持 | `iter.SeekGE(prefix)` + 遍历 |
| 批量写入 | ✅ 支持 | `pebble.Batch` 原生支持 |
| 压缩 | ✅ 支持 | Snappy/ZSTD 原生支持 |
| Checkpoint | ✅ 支持 | `db.Checkpoint()` 原生支持 |

## 5. 企业级业务场景矩阵

| 业务场景 | 支持状态 | 实现方式与约束 |
|---|---|---|
| 嵌入式 KV 存储 | ✅ 优势场景 | CockroachDB/PlanetScale 同款引擎 |
| 本地配置缓存 | ✅ 完整支持 | 持久化 + 快速读 |
| 审计日志存储 | ✅ 支持 | 追加写友好 |
| 会话存储 | ⚠️ 部分 | TTL 非原生 |
| 限流计数 | ❌ 不支持 | 无原子自增 |
| 热点数据缓存 | ⚠️ 部分 | 磁盘 IO 比内存慢 |
| Checkpoint 快照 | ✅ 支持 | 原生支持，可用于备份 |
| 健康探针 | ✅ 支持 | Metrics 检查 |
| 链路追踪 | ✅ 支持 | `kv.pebble.*` span |

## 6. 已知限制与风险

1. **Incr/Decr 不支持**：Pebble 无原子计数原语。
2. **TTL 非原生**：通过 value 元数据模拟，读取时需解析。
3. **无网络协议**：嵌入式引擎，仅单进程使用。
4. **Compaction 写放大**：高写负载时 compaction 线程会占用 CPU/IO。
5. **Keys 全量扫描**：大数据集性能差。
6. **与 Badger/BBolt 差异**：Pebble 是 LSM Tree，Badger 也是 LSM 但实现不同，跨驱动迁移需注意语义差异。

## 7. 快速接入示例

```go
import (
    "github.com/gospacex/kvx"
    _ "github.com/gospacex/kvx/pebble" // 注册驱动
)

stores, _ := kvx.New([]*kvx.Config{{
    Name: "default", Driver: "pebble",
    Addr: "/tmp/pebble-data", // 本地数据目录
    Timeout: 5 * time.Second,
}})
s := stores["default"] // kvx.KVStore
```
