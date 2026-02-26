## Context

### 当前架构

wstun server 采用双连接模型：

1. **控制连接（Control Connection）**：WebSocket 长连接，用于协调
   - Client 发送 `WAITING` 消息表示就绪
   - Server 发送 `CONNECT:[ULID]` 通知 client 有新连接

2. **数据连接（Data Connection）**：WebSocket 短连接，用于实际数据传输
   - Client 收到 `CONNECT` 后建立新 WebSocket 连接
   - Client 发送 `HANDLE:[ULID]` 认领连接
   - 双向二进制数据传输

### 问题场景

`RequestConnection` 方法（`server_protocol.go:226`）的工作流程：
1. 生成 ULID 作为连接 ID
2. 创建 pending connection 记录（30s 超时）
3. 向该 service 的所有 control connection 发送 `CONNECT:[ULID]`
4. 等待 client 建立 data connection 或超时

**问题**：如果 client 因瞬时网络抖动、高负载或 GC 暂停未能及时响应，连接直接失败，外部 TCP 连接被关闭。

### 对比

Client 端已有完善的 reconnection 机制（指数退避，最大 60s），但 Server 端缺乏类似的 retry 机制。

## Goals / Non-Goals

**Goals:**
- 在 `RequestConnection` 中实现可配置的重试机制
- 支持指数退避策略，避免雪崩效应
- 保持向后兼容，默认行为不变
- 提供清晰的配置接口（配置文件）

**Non-Goals:**
- 修改 client 端行为（client 已有 reconnection）
- 修改 WebSocket 协议或消息格式
- 支持跨多个 server 实例的分布式重试
- 持久化 pending connection 状态（单机内存模型保持简单）

## Decisions

### 1. 重试触发条件

**Decision**: 仅在 pending connection 超时时重试，其他错误（如 context cancelled）不重试。

**Rationale**:
- 超时是 client 未及时响应的最常见原因
- context cancelled 通常是外部请求取消，不应重试

**Alternatives considered**:
- 所有错误都重试 → 拒绝，可能放大错误（如 client 已下线）

### 2. 重试策略

**Decision**: 指数退避 + 最大间隔限制，类似 client 端实现。

**Algorithm**:
```
delay = RetryInterval * (2 ^ attempt)
if delay > RetryMaxInterval: delay = RetryMaxInterval
```

**Rationale**:
- 与 client 端保持一致，降低心智负担
- 指数退避避免在故障时对 client 造成 thundering herd
- 最大间隔防止等待时间过长

**Alternatives considered**:
- 固定间隔 → 简单但不够灵活
- 线性退避 → 在 client 恢复时连接建立较慢

### 3. 重试时是否重新生成 ULID

**Decision**: 重试时保留相同的 ULID。

**Rationale**:
- 简化 pending connection 管理（无需清理旧记录）
- client 收到重复的 `CONNECT:[ULID]` 可以识别为同一连接
- 避免创建多个 pending connection 记录

**Alternatives considered**:
- 每次重试生成新 ULID → 复杂化，需要清理旧记录，client 会创建多个连接

### 4. 配置选项

**Decision**: 新增三个配置项：
- `RetryAttempts` (int): 最大重试次数，默认 0（保持当前行为）
- `RetryInterval` (duration): 初始重试间隔，默认 5s
- `RetryMaxInterval` (duration): 最大重试间隔，默认 60s

**Rationale**:
- 默认关闭保持向后兼容
- 三个参数提供足够的灵活性
- 命名与 client 端的 reconnection 配置区分（这是 server 的 retry）

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| 重试导致外部 TCP 连接等待时间过长 | 配置合理的 `RetryMaxInterval`；外部可设置自己的超时 |
| Client 收到重复的 `CONNECT` 消息 | Client 已支持处理重复消息（根据 ULID 匹配） |
| 内存中 pending connection 堆积 | 每次重试重置超时时间，不会创建额外记录 |
| 与旧版本 client 不兼容 | 协议不变，只是重复发送相同的 `CONNECT` 消息 |

## Migration Plan

1. **Phase 1**: 添加配置选项，默认 `RetryAttempts=0`，行为不变
2. **Phase 2**: 实现 `RequestConnection` 中的重试逻辑
3. **Phase 3**: 测试不同重试配置下的行为
4. **Phase 4**: 文档更新，说明新配置选项

**Rollback**: 将 `RetryAttempts` 设为 0 即可恢复原有行为。

## Open Questions

1. 是否需要支持 jitter 来避免多个 server 实例同时重试？
   - **当前决定**: 暂不支持，单机场景下不需要
   
2. 是否需要在 metrics 中暴露重试次数统计？
   - **当前决定**: 暂不实现，保持简单
