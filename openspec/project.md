# Project Context

## Purpose
`wstun` is a WebSocket tunneling library and command-line tools written in Go, providing a simple and efficient way to establish TCP tunnels over WebSocket connections. The project enables secure and reliable communication channels for exposing local services through WebSocket proxies, particularly useful for firewall traversal, development environments, and service exposure scenarios.

**Key Capabilities:**
- Tunnel TCP connections over WebSocket (ws:// and wss://)
- Service-based authentication and routing
- Multi-service support on a single server
- Automatic reconnection with configurable retry logic
- Graceful shutdown handling

## Tech Stack
- **Language:** Go 1.25.3
- **Dependencies (minimal by design):**
  - `github.com/gorilla/websocket` v1.5.3 - WebSocket protocol implementation
  - `github.com/oklog/ulid/v2` v2.1.1 - ULID generation for connection matching
  - `gopkg.in/yaml.v3` v3.0.1 - YAML configuration parsing
- **Build Tools:** Standard Go toolchain (`go build`, `go test`)
- **Deployment:** Standalone binaries (`wstun-server`, `wstun-client`)

## Project Conventions

### Code Style
- Follow standard Go conventions and idioms
- Use `gofmt` for formatting
- Use `golint` and `go vet` for static analysis
- Package-level documentation in doc comments
- Exported functions and types must have doc comments starting with the name
- Use camelCase for unexported names, PascalCase for exported names
- Prefer short, clear variable names in small scopes
- Use meaningful names for package-level declarations

### Architecture Patterns
- **Dual-mode design:** Library (reusable components) + CLI tools (standalone binaries)
- **Two-connection protocol model:**
  - Control connection: long-lived WebSocket for orchestration and commands
  - Data connections: ephemeral WebSocket connections for actual traffic tunneling
- **Service-based routing:** Each service has unique credentials and listen address
- **ULID-based connection matching:** ULIDs identify connection pairs (not for security)
- **BasicAuth authentication:** Username (service name) and password (service token)
- **Single responsibility principle** for types and functions
- **Prefer interfaces** for flexibility and testability (e.g., `io.ReadWriteCloser`)
- **Use `context.Context`** for cancellation and timeout handling
- **Error handling:** Return errors explicitly, use `fmt.Errorf` with `%w` for wrapping
- **Configuration:** YAML files with validation and defaults
- **Protocol design:** Text messages for control, binary messages for data transfer

### Testing Strategy
- Unit tests using Go's standard testing package
- Test files named `*_test.go`
- Use table-driven tests for multiple test cases
- Aim for high test coverage on public API
- Use `go test -race` to detect race conditions
- Benchmark critical paths using Go's benchmarking framework
- Integration tests when external dependencies are involved

### Git Workflow
- Use conventional commits with scope (e.g., `feat(tunnel): add connection pooling`, `fix(ws): handle connection timeout`)
- Scopes typically align with package or functional areas
- Branch naming: `feature/*`, `fix/*`, `refactor/*`
- Main branch: `main`
- Keep commits atomic and focused
- Write meaningful commit messages with imperative mood

## Domain Context

### Core Concepts
- **WebSocket Protocol (RFC 6455):** Bidirectional, full-duplex communication over HTTP
- **TCP Tunneling:** Encapsulating TCP connections within WebSocket frames for traversal
- **Service Exposure:** Making local services accessible through remote endpoints
- **Firewall Traversal:** Bypassing restrictive firewalls using outbound WebSocket connections

### Protocol Flow
1. Client establishes control connection with BasicAuth (`username:password`)
2. Client sends `WAITING` message to indicate readiness
3. When server receives incoming TCP connection, it generates ULID and sends `CONNECT:[ULID]`
4. Client opens new data connection with BasicAuth
5. Client sends `HANDLE:[ULID]` to claim the connection
6. Server matches ULID and bridges the connections for bidirectional data transfer

### Key Terminology
- **Control Connection:** Long-lived WebSocket for orchestration (text messages)
- **Data Connection:** Short-lived WebSocket for traffic tunneling (binary messages)
- **Service:** Logical endpoint with authentication credentials and listen address
- **ULID:** Universally Unique Lexicographically Sortable Identifier for connection pairing
- **Target Address:** Local service address that client tunnels to
- **Listen Address:** Public address where server exposes the service

### Connection Lifecycle
- **Establishment:** BasicAuth validation → connection type detection → pairing/orchestration
- **Operation:** Bidirectional binary data streaming with buffer management
- **Termination:** Graceful close on either side, cleanup of paired resources
- **Reconnection:** Client automatically reconnects control connection with backoff

## Important Constraints

### Compatibility
- Must be compatible with Go 1.25.3+
- Should work on Linux, macOS, and Windows
- Must support both ws:// and wss:// protocols

### Dependencies
- Minimize external dependencies to reduce maintenance burden
- Only add dependencies for core functionality (WebSocket, YAML, ULID)
- Carefully evaluate new dependencies for security, maintenance, and licensing

### Reliability
- Library must be thread-safe and suitable for concurrent use
- Must handle network failures gracefully (reconnection, timeouts)
- Must handle malformed protocol messages without crashing
- Graceful shutdown must allow in-flight connections to complete

### Security
- BasicAuth credentials sent only over secure connections in production
- ULID used for connection matching, NOT for security or authentication
- TLS certificate validation enabled by default (can be disabled for testing only)
- No credential logging or exposure in logs

### Performance
- Aim for low latency in data transfer (minimize buffering overhead)
- Efficient memory usage (stream data, avoid loading entire payloads)
- Handle high connection concurrency without degradation
- Connection setup should be fast (< 100ms in typical network conditions)

### Configuration
- All configuration via YAML files (no environment variables or flags for config values)
- Configuration must be validated on load with clear error messages
- Support for configuration hot-reload not required (restart expected)

## External Dependencies

### Current Dependencies
- **`github.com/gorilla/websocket` v1.5.3**
  - Purpose: RFC 6455 compliant WebSocket implementation
  - License: BSD-2-Clause
  - Rationale: Industry standard, well-maintained, high performance
  
- **`github.com/oklog/ulid/v2` v2.1.1**
  - Purpose: ULID generation for connection matching
  - License: Apache-2.0
  - Rationale: Lexicographically sortable, collision-resistant identifiers

- **`gopkg.in/yaml.v3` v3.0.1**
  - Purpose: YAML configuration file parsing
  - License: Apache-2.0 and MIT
  - Rationale: Standard Go YAML library, human-friendly configuration format

### Dependency Evaluation Criteria
Future dependencies should be carefully evaluated for:
- **Maintenance status:** Active development and community support
- **License compatibility:** Permissive licenses (MIT, BSD, Apache-2.0)
- **Security track record:** No history of critical vulnerabilities
- **Performance impact:** Must not introduce significant overhead
- **Size and scope:** Prefer focused libraries over large frameworks
- **Stability:** Prefer stable APIs over rapidly changing interfaces

## Project Structure

### Directory Layout
```
wstun/
├── cmd/                      # Command-line applications
│   ├── wstun-server/        # Server binary
│   │   └── main.go          # Server entry point
│   └── wstun-client/        # Client binary
│       └── main.go          # Client entry point
├── examples/                 # Example configuration files
│   ├── server.yaml          # Server config example
│   ├── client.yaml          # Client config example
│   └── services/            # Service definition examples
│       └── my-service.yaml  # Service config example
├── openspec/                 # OpenSpec documentation
│   ├── project.md           # This file
│   ├── AGENTS.md            # AI assistant instructions
│   ├── specs/               # Current specifications
│   └── changes/             # Proposed changes and archives
├── *.go                      # Library implementation
├── *_test.go                # Unit tests
├── go.mod                    # Go module definition
├── go.sum                    # Dependency checksums
└── README.md                 # User documentation
```

### File Naming Conventions
- **Library files:** Descriptive names (`config.go`, `protocol.go`, `wstun.go`)
- **Protocol files:** Role-specific (`client_protocol.go`, `server_protocol.go`)
- **Test files:** Mirror source files with `_test.go` suffix
- **Integration tests:** `cmd_integration_test.go` for end-to-end testing
- **Commands:** Binary names with `wstun-` prefix for clarity

### Module Organization
- **Root package (`wstun`):** Public API, configuration, protocol utilities
- **`cmd/wstun-server`:** Server application with flag parsing and lifecycle
- **`cmd/wstun-client`:** Client application with flag parsing and lifecycle
- No internal packages yet (keep simple until proven necessary)

## Common Patterns and Idioms

### Configuration Pattern
```go
// 1. Define struct with yaml tags
type ServerOptions struct {
    Field string `yaml:"field_name"`
}

// 2. Validation method
func (o *ServerOptions) Validate() error { ... }

// 3. Set defaults method
func (o *ServerOptions) SetDefaults() { ... }

// 4. Load function
func LoadServerConfig(path string) (*ServerOptions, error) {
    // Read file → Parse YAML → SetDefaults → Validate
}
```

### Protocol Message Pattern
```go
// Constants for message types
const (
    MessageTypeWaiting = "WAITING"
    MessageTypePrefixConnect = "CONNECT:"
)

// Format functions
func FormatConnectMessage(connID string) string

// Parse functions  
func ParseConnectMessage(message string) (string, bool)
```

### Error Handling Pattern
```go
// Always wrap errors with context
return fmt.Errorf("failed to parse config: %w", err)

// Validation errors are clear and actionable
return fmt.Errorf("listen_addr is required")
```

### CLI Application Pattern
```go
// 1. Flag definition with flag.String()
// 2. Custom Usage function
// 3. flag.Parse()
// 4. Validate required flags
// 5. Load configuration
// 6. Set up signal handling for graceful shutdown
// 7. Start application logic
// 8. Wait for shutdown signal
```

## Naming Conventions

### Types and Functions
- **Options structs:** `ServerOptions`, `ClientOptions` (not `Config` or `Settings`)
- **Load functions:** `LoadServerConfig`, `LoadClientConfig` (verb + noun)
- **Validation methods:** `Validate() error` (receiver method)
- **Protocol functions:** `FormatConnectMessage`, `ParseHandleMessage` (verb + noun)

### Configuration Fields
- **YAML fields:** snake_case (`listen_addr`, `service_token`, `max_connections`)
- **Go fields:** PascalCase (`ListenAddr`, `ServiceToken`, `MaxConnections`)
- **Boolean flags:** Positive naming (`TLSInsecure` not `TLSSecure`)

### Variables and Parameters
- **Short-lived:** Single letter or abbreviation (`err`, `opts`, `cfg`)
- **Package-level:** Full descriptive names (`MessageTypePrefixConnect`)
- **Interface parameters:** Descriptive (`incoming io.ReadWriteCloser`)

## Development Workflow

### Making Changes
1. Read `openspec/project.md` (this file) for conventions
2. Check for existing proposals in `openspec/changes/`
3. Create change proposal if needed (see `openspec/AGENTS.md`)
4. Implement changes following conventions
5. Write/update tests
6. Run `go test -race ./...` to verify
7. Commit with conventional commit format: `feat(scope): description`

### Testing Approach
- **Unit tests:** Test individual functions and methods (`*_test.go`)
- **Table-driven tests:** Multiple test cases in single test function
- **Integration tests:** Test full client-server interaction (`cmd_integration_test.go`)
- **Race detection:** Always run `go test -race` before committing
- **Coverage:** Aim for high coverage on public API

### Build and Run
```bash
# Build both binaries
go build ./cmd/wstun-server ./cmd/wstun-client

# Run tests
go test -v ./...
go test -race ./...

# Run server
./wstun-server -c examples/server.yaml

# Run client
./wstun-client -c examples/client.yaml
```
