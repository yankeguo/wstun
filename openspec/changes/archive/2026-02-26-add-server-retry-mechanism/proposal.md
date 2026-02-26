## Why

当前 server 在 `RequestConnection` 方法中发送 `CONNECT:[ULID]` 消息后，只会等待一次 client 建立数据连接（默认 30s 超时）。如果 client 由于瞬时网络问题或负载过高而未能及时响应，连接会直接失败，导致外部 TCP 连接被关闭。增加重试机制可以提高隧道连接的可靠性和容错性，特别是在不稳定的网络环境下。

## What Changes

- 在 server 协议层的 `RequestConnection` 方法中增加重试逻辑
- 当 client 未在超时时间内建立数据连接时，重新发送 `CONNECT` 消息
- 支持配置重试次数和重试间隔（指数退避）
- 在重试耗尽后，才关闭外部 TCP 连接并返回错误

## Capabilities

### New Capabilities
- `server-connection-retry`: Server 端连接请求重试机制，包括可配置的重试次数、重试间隔和退避策略

### Modified Capabilities
- `cli-server`: 支持从配置文件加载 server 重试相关配置

## Impact

- `server_protocol.go`: 修改 `RequestConnection` 方法，添加重试逻辑
- `cmd/wstun-server/main.go`: 加载重试配置并传递给 protocol
- 配置文件：新增 `RetryAttempts`, `RetryInterval`, `RetryBackoff` 等配置项
