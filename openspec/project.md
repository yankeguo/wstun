# Project Context

## Purpose
`wstun` is a WebSocket tunneling library written in Go, providing a simple and efficient way to establish tunneled connections over WebSocket protocols. The library aims to enable secure and reliable communication channels for various networking scenarios.

## Tech Stack
- Go 1.25.3
- Standard library focused (minimal external dependencies)
- Pure Go implementation

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
- Library design pattern: provide composable, reusable components
- Single responsibility principle for types and functions
- Prefer interfaces for flexibility and testability
- Use context.Context for cancellation and timeout handling
- Error handling: return errors explicitly, use fmt.Errorf with %w for wrapping
- Configuration through functional options pattern when appropriate

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
- WebSocket protocol (RFC 6455) for bidirectional communication
- Tunneling: encapsulating one protocol within another for routing, security, or firewall traversal
- Connection management: handling lifecycle, timeouts, reconnection logic
- Binary and text message handling
- Performance considerations for network I/O operations

## Important Constraints
- Must be compatible with Go 1.25.3+
- Should minimize external dependencies to reduce maintenance burden
- Library should be thread-safe and suitable for concurrent use
- Must handle network failures gracefully
- Performance: aim for low latency and efficient memory usage

## External Dependencies
- Currently using only Go standard library
- Future dependencies should be carefully evaluated for:
  - Maintenance status and community support
  - License compatibility
  - Security track record
  - Performance impact
