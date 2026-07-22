# AGENTS.md

本文档面向 AI 编码代理，介绍 wstun 项目的架构、开发约定与常用命令。

## 项目概述

wstun 是一个简单高效的 **TCP over WebSocket 隧道**库和命令行工具，使用 Go 语言编写。典型场景：将 NAT/防火墙后的本地 TCP 服务，通过反向 WebSocket 隧道暴露到公网服务器上。

- **模块路径**：`cnb.cool/yankeguo/wstun`（托管在 cnb.cool）
- **语言**：Go 1.25.3+
- **依赖**（仅 3 个外部依赖，保持最小化）：
  - `github.com/gorilla/websocket` v1.5.3 — WebSocket 实现
  - `github.com/oklog/ulid/v2` v2.1.1 — 连接 ID（ULID）生成
  - `gopkg.in/yaml.v3` v3.0.1 — YAML 配置解析
- **日志**：使用标准库 `log/slog` 结构化日志（无第三方日志库）

### 协议设计（双连接模型）

- **控制连接**：长连接，文本消息（TextMessage）。客户端发送 `WAITING` 表示就绪；服务器发送 `CONNECT:[ULID]` 通知有新连接。
- **数据连接**：短连接，二进制消息（BinaryMessage）。客户端发送 `HANDLE:[ULID]` 认领连接，之后进行双向数据转发。
- 所有连接均通过 **BasicAuth** 认证（username = service_name，password = service_token）。
- ULID 仅用于连接配对，**不是**安全机制。
- 服务器使用单一 WebSocket 端点（`/`），路径无关。

## 项目结构

```
wstun/
├── cmd/
│   ├── wstun-server/        # 服务器命令行（main.go 入口 + server.go 服务封装）
│   └── wstun-client/        # 客户端命令行（main.go 入口 + client.go 重连逻辑）
├── protocol.go              # 协议常量、消息格式化/解析、ULID 校验（包文档在此文件头部）
├── server.go                # ServerProtocol：服务器端协议处理（WebSocket 升级、认证、连接配对、重试）
├── client.go                # ClientProtocol：客户端协议处理（控制连接、ping/pong 心跳、数据转发）
├── config.go                # ServerOptions / ClientOptions / ServiceDefinition，YAML 加载与校验
├── log.go                   # ParseLogLevel：日志级别字符串到 slog.Level 的映射
├── *_test.go                # 单元测试与集成测试（均在根包 wstun 中）
├── examples/                # 配置示例（server.yaml、client-*.yaml、services/*.yaml）及 deploy/ 部署脚本
├── Dockerfile.server        # 服务器镜像（debian:13-slim + 预编译二进制）
├── Dockerfile.client        # 客户端镜像
├── wstun-client@.service    # systemd 模板单元（配置路径 /etc/wstun-client/%i.yml）
├── .cnb.yml                 # cnb.cool CI 流水线
└── README.md                # 完整中文用户文档（配置详解、示例、FAQ）
```

## 构建与测试命令

```bash
# 构建（产物输出到仓库根目录，已被 .gitignore 忽略）
go build -o wstun-server ./cmd/wstun-server
go build -o wstun-client ./cmd/wstun-client

# 运行全部测试
go test ./...

# 竞态检测（提交前必须通过）
go test -race ./...

# 运行特定测试
go test -v -run TestProtocolFlow

# 覆盖率
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 代码检查（提交前必须通过）
go fmt ./...
go vet ./...
```

没有 Makefile、lint 配置或 Taskfile；构建/测试直接依赖 Go 工具链。

## 代码组织与约定

- **库与 CLI 分离**：根包 `wstun` 是可复用的协议库（`ServerProtocol` / `ClientProtocol` 通过公开字段如 `DataHandler`、`ConnectHandler`、`Logger`、`PingInterval` 等注入行为）；`cmd/` 下的两个 `main` 包是薄封装，负责加载 YAML 配置、装配库组件、信号处理与优雅关闭。
- **配置模式**：每个配置结构体都有 `Validate()` 和 `SetDefaults()` 方法；`Load*Config` 函数按「读文件 → YAML 解析 → SetDefaults → Validate」的顺序加载。新增配置字段时须同步更新这四个位置及 README 的配置表格和 `examples/` 中的注释示例。
- **错误处理**：使用 `fmt.Errorf("...: %w", err)` 包装错误；哨兵错误（如 `ErrInvalidCredentials`）定义在 `server.go` 顶部。
- **并发安全**：共享状态用 `sync.Mutex`/`RWMutex` 或 `atomic` 保护；项目强调线程安全，所有改动须通过 `go test -race`。
- **健壮性约定**：关键 goroutine 中有 `recover()` panic 恢复；协议消息有长度上限（`maxMessageLength = 1024`）；服务名、连接 ID 等输入有严格校验。保持这些防御性检查，不要在重构中移除。
- **数据转发细节**：处理 `Read()` 返回值时必须先处理 `n > 0` 的字节再判断 `err`（`Read` 可能同时返回数据和 `io.EOF`），否则会丢数据。
- **注释语言**：代码注释使用英文；面向用户的文档（README）使用中文。代码注释遵循 Go doc 惯例（以符号名开头）。

## 测试策略

- 所有测试位于根包 `wstun`（`*_test.go`），`cmd/` 下没有测试文件，但 `integration_test.go` 会以 `go run` / 子进程方式对两个 CLI 做集成测试（测试二进制帮助信息、缺失配置、无效配置等路径）。
- 测试文件分工（每个测试文件对应同名被测源文件）：
  - `protocol_test.go` — 协议常量、ULID 生成/校验、消息格式化/解析（含超长消息、非法 ULID 等边界）
  - `server_test.go` — 服务器端：认证、完整连接流程、并发、超时、重试逻辑、控制连接健康检查与死连接清理
  - `client_test.go` — 客户端：BasicAuth 凭据、数据转发循环（空闲连接不误判超时）
  - `config_test.go` — 配置校验、默认值、YAML 加载（含服务名/服务器 URL 边界、路径安全）
  - `log_test.go` — 日志级别解析
  - `integration_test.go` — CLI 集成测试（原 cmd_integration_test.go）
- 新增功能须附带测试；提交前确保 `go test ./...` 与 `go test -race ./...` 全部通过、`go fmt`/`go vet` 无输出。

## CI 与部署

- CI 使用 cnb.cool 流水线（`.cnb.yml`），在 `main` 分支 push 和 tag push 时触发：用 `golang:1.25.3-trixie` 镜像构建两个二进制，然后构建并推送 Docker 镜像到 cnb 仓库（tag 为 `server-<版本>` / `client-<版本>`，main 分支为 `latest`）。
- Dockerfile 基于 `debian:13-slim`，直接将 CI 中预编译好的二进制 `ADD` 进镜像，默认启动命令为 `wstun-server -c config.yaml`（工作目录 `/wstun-server`，挂载配置到此处）。
- Linux 上可用 `wstun-client@.service` systemd 模板运行客户端实例。
- `examples/deploy/` 中有 Kubernetes 环境的部署示例脚本（jsonnet + shell）。

## 安全注意事项

- `service_token` 是唯一的认证凭证，生产环境必须使用强随机令牌（如 `openssl rand -hex 32`）并定期轮换。
- 生产环境使用 TLS（`wss://`）；`tls_insecure: true` 仅限测试，服务器端 TLS 最低版本为 TLS 1.2。
- ULID 连接 ID 不提供安全性，不要在协议改动中把它当作认证依据。
- 服务定义文件通过 `filepath.Join` 等安全路径处理加载，防止目录遍历；相关逻辑在 `LoadAllServices` 中，修改时保持该防护。

## 文档维护

修改配置项、协议行为或命令行用法时，同步更新：

- `README.md`（中文，含配置表格、示例）
- `examples/` 下相关 YAML 文件的注释
- 本文件 `AGENTS.md`（如结构或约定发生变化）
