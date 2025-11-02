# Design: CLI Commands

## Context
The wstun library provides core WebSocket tunneling functionality but lacks executable binaries for end users. This design covers the creation of two standalone CLI tools (`wstun-server` and `wstun-client`) that provide a simple interface for running tunnels with file-based configuration.

## Goals / Non-Goals

**Goals:**
- Provide simple, single-purpose CLI binaries for server and client roles
- Support file-based configuration for easy deployment and automation
- Follow Go standard library patterns where possible
- Keep dependencies minimal and well-maintained

**Non-Goals:**
- Complex CLI frameworks or subcommand structures (keep it simple: single command + config flag)
- Interactive configuration wizards (file-based only)
- Built-in configuration file generation (users write YAML manually)
- Advanced CLI features like shell completion (can be added later if needed)

## Decisions

### Decision: YAML for Configuration Format
**Choice:** Use YAML format for configuration files

**Rationale:**
- Human-readable and widely adopted in operations/DevOps contexts
- Native Go library support via `gopkg.in/yaml.v3` (stable, well-maintained)
- Better readability than JSON for config files with comments support
- Industry standard for configuration files in similar tools

**Alternatives considered:**
- JSON: Less human-friendly, no comment support
- TOML: Less common in Go ecosystem, would require additional dependency
- INI: Too simplistic for nested configuration structures
- Custom format: Unnecessary complexity

### Decision: Separate Binaries for Client and Server
**Choice:** Two distinct executables (`wstun-server` and `wstun-client`)

**Rationale:**
- Clear separation of concerns and roles
- Simpler deployment (deploy only what you need)
- Reduces binary size for single-role deployments
- Follows Unix philosophy of single-purpose tools

**Alternatives considered:**
- Single binary with subcommands: Adds complexity, larger binary for single-role use
- Single binary with mode flag: Less discoverable, more error-prone

### Decision: Configuration File Required
**Choice:** Make `-c` flag mandatory; no default configuration locations

**Rationale:**
- Explicit over implicit: users know exactly which config is being used
- Prevents confusion from multiple config search paths
- Simpler implementation and testing
- Better for containerized/orchestrated deployments

**Alternatives considered:**
- Search multiple default paths: Adds complexity and potential confusion
- Allow both file and CLI flags: Increases complexity and precedence rules

### Decision: Configuration Struct Field Design
**Choice:** Define comprehensive struct fields covering common use cases, with sensible defaults for optional fields

**Fields to determine (TBD during implementation):**

**ServerOptions (preliminary):**
- `ListenAddr` (required): Address to bind server (e.g., `:8080`)
- `TLSCert` (optional): Path to TLS certificate file
- `TLSKey` (optional): Path to TLS private key file
- `LogLevel` (optional): Logging verbosity (default: `info`)
- `MaxConnections` (optional): Maximum concurrent connections (default: unlimited)

**ClientOptions (preliminary):**
- `ServerURL` (required): WebSocket server URL to connect to
- `LocalAddr` (required): Local address to listen on for incoming connections
- `TLSInsecure` (optional): Skip TLS verification (default: `false`)
- `ReconnectInterval` (optional): Time between reconnection attempts (default: `5s`)
- `MaxReconnectAttempts` (optional): Max reconnection attempts (default: unlimited)
- `LogLevel` (optional): Logging verbosity (default: `info`)

**Rationale:**
- Required fields ensure minimal viable configuration
- Optional fields with defaults provide flexibility without overwhelming users
- Field names follow Go conventions (PascalCase for exported)
- Types chosen for clarity and ease of validation

## Risks / Trade-offs

### Risk: External YAML Dependency
**Mitigation:** Use `gopkg.in/yaml.v3` which is stable, widely used, and actively maintained. This is an acceptable exception to the "minimal dependencies" constraint as YAML is the de-facto standard for configuration.

### Risk: Configuration Field Evolution
**Trade-off:** Once we define the configuration structs, changing them becomes a breaking change.

**Mitigation:**
- Start with essential fields only
- Use optional fields with defaults where appropriate
- Version configuration format if major changes needed later
- Document all fields clearly

### Risk: Validation Complexity
**Trade-off:** Comprehensive validation adds code but prevents runtime errors.

**Mitigation:**
- Implement validation methods on both structs
- Fail fast with clear error messages
- Validate at configuration load time, before starting server/client

## Migration Plan

**Initial Implementation:**
1. Add `gopkg.in/yaml.v3` to `go.mod`
2. Define `ServerOptions` and `ClientOptions` structs in a new `config` package
3. Implement configuration loading and validation functions
4. Create `cmd/wstun-server/main.go` with flag parsing and server startup
5. Create `cmd/wstun-client/main.go` with flag parsing and client startup
6. Add signal handling for graceful shutdown in both commands
7. Create example configuration files in `examples/` directory
8. Document usage in README

**Testing Strategy:**
- Unit tests for configuration parsing and validation
- Integration tests running actual binaries with test configs
- Test error cases (missing files, invalid YAML, validation failures)

**Rollout:**
- No existing users to migrate (new functionality)
- Document configuration format in README and example files

## Open Questions

1. **Specific ServerOptions and ClientOptions fields:** Need to finalize exact fields based on core library capabilities. The preliminary list above should be refined during implementation.

2. **Logging implementation:** Should we use standard `log` package or introduce a structured logging library? Recommendation: Start with `log.Logger` for simplicity.

3. **Configuration file examples location:** Should example configs live in `examples/` or `cmd/wstun-{server,client}/` directories? Recommendation: `examples/` at project root.

4. **Environment variable overrides:** Should configuration support environment variable overrides? Recommendation: Not in initial version; add later if needed.

