# Design: Server and Client CLI Implementation

## Context
The protocol layer (WebSocket communication, message handling, authentication) is complete. The configuration layer (loading YAML, validation) is complete. The CLI binaries have skeleton implementations that load config but don't actually start the tunneling functionality. This design covers how to wire everything together.

## Goals / Non-Goals

### Goals
- Complete functional server that accepts WebSocket connections and tunnels traffic to services
- Complete functional client that maintains persistent connection and handles tunneling requests
- Proper lifecycle management (startup, shutdown, reconnection)
- Production-ready error handling and logging
- Respect all configuration options (TLS, logging, limits, etc.)

### Non-Goals
- Changing protocol behavior (already specified and implemented)
- Changing configuration format (already specified and implemented)
- Adding new features beyond what specs require
- Performance optimization (basic implementation first)

## Decisions

### Server Architecture
**Decision:** Use a Server struct that encapsulates protocol, listeners, and state management.

**Structure:**
```go
type Server struct {
    config    *ServerOptions
    services  map[string]*ServiceDefinition
    protocol  *ServerProtocol
    listeners map[string]net.Listener  // serviceName -> TCP listener
    httpServer *http.Server
    logger    *log.Logger
    activeConns int32  // atomic counter
    shutdown  chan struct{}
}
```

**Rationale:**
- Encapsulation makes shutdown and cleanup easier
- Clear ownership of resources
- Testable structure

### Server Lifecycle
**Flow:**
1. Load configuration and validate
2. Load all service definitions from ServicesDir
3. Create ServerProtocol with auth function
4. Start TCP listener for each service (on ServiceDefinition.ListenAddr)
5. Start goroutine per service to accept connections
6. Start HTTP server with WebSocket handler (on ServerOptions.ListenAddr)
7. Wait for shutdown signal
8. Close all TCP listeners
9. Close HTTP server gracefully
10. Wait for active connections to finish (with timeout)

**Alternatives considered:**
- Using HTTP server's lifecycle only: Rejected because we need separate TCP listeners per service
- Starting TCP listeners on-demand: Rejected for simplicity; fail-fast on startup is better

### Client Architecture
**Decision:** Use a Client struct with reconnection loop in main goroutine.

**Structure:**
```go
type Client struct {
    config   *ClientOptions
    protocol *ClientProtocol
    logger   *log.Logger
    shutdown chan struct{}
}
```

**Rationale:**
- Simple structure matches single-service client role
- Reconnection logic belongs in main loop, not protocol layer
- Protocol layer remains reusable

### Client Reconnection Strategy
**Decision:** Exponential backoff with configurable max attempts, starting from ReconnectInterval.

**Algorithm:**
```
attempt = 0
delay = config.ReconnectInterval

while running:
  err := protocol.Start()  // blocks until disconnect
  if shutdown requested:
    break
  
  attempt++
  if max_attempts > 0 && attempt >= max_attempts:
    log fatal and exit
  
  log reconnecting in delay
  sleep(delay)
  delay = min(delay * 2, 60s)  // cap at 60s
```

**Rationale:**
- Exponential backoff reduces load on server during outages
- Cap prevents excessive delays
- Max attempts provides failure escape hatch
- Simple to implement and understand

### Data Transfer Implementation
**Server DataHandler:**
- Receives: incoming TCP connection, outgoing WebSocket connection
- Creates two goroutines: TCP->WS and WS->TCP
- Uses io.Copy for efficient transfer
- Increments/decrements connection counter
- Handles close on either direction

**Client ConnectHandler:**
- Receives: context, connection ID
- Dials TargetAddr using net.Dial
- Returns io.ReadWriteCloser
- Let protocol handle bidirectional transfer (already implemented in client_protocol.go)

**Rationale:**
- Protocol layer already handles WebSocket specifics
- io.Copy is simple and efficient
- Goroutine-per-direction is standard Go pattern

### Logging Strategy
**Levels implemented:**
- `error`: Authentication failures, connection errors, load failures
- `warn`: Connection timeouts, reconnection attempts
- `info`: Startup, shutdown, connection established/closed
- `debug`: Protocol messages, detailed flow

**Decision:** Use standard log.Logger with prefixes, filter by LogLevel configuration.

**Alternatives considered:**
- Structured logging library: Rejected to minimize dependencies
- Log levels in logger: Standard library doesn't support levels, so manual filtering

## Risks / Trade-offs

### Risk: TCP listener errors after startup
**Mitigation:** Log error and continue serving other services. Operator can check logs and fix configuration.

### Risk: Connection leaks during shutdown
**Mitigation:** 
- Track active connections with atomic counter
- Close all listeners first (stops new connections)
- Context cancellation for in-flight operations
- Timeout for graceful shutdown (10 seconds)

### Risk: Client reconnection storm
**Mitigation:**
- Exponential backoff with cap
- Max reconnect attempts option
- Server can handle authentication quickly (reject invalid creds)

### Trade-off: Synchronous TCP accept vs connection pool
**Choice:** Synchronous accept with goroutine-per-connection
**Rationale:** Simpler implementation, adequate for expected scale, matches Go idioms

## Migration Plan
N/A - This is initial implementation completing skeletal code.

## Open Questions
None - Protocol and configuration are well-defined, implementation is straightforward wiring.

