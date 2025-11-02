# WebSocket Protocol Design

## Context
The wstun tunneling system requires a lightweight WebSocket-based protocol to:
- Authenticate clients using service credentials
- Manage connection lifecycle with minimal overhead
- Route incoming traffic to the appropriate client connections
- Support efficient bidirectional data transfer

This design focuses on simplicity and testability, keeping the protocol layer independent from command-line tools.

## Goals / Non-Goals

### Goals
- Define clear message protocol for client-server communication
- Use ULID for globally unique connection identifiers
- Implement BasicAuth for initial authentication
- Support bidirectional binary message streaming
- Keep protocol layer independent and testable
- Handle connection lifecycle explicitly (WAITING, CONNECT, HANDLE states)

### Non-Goals
- Integration with CLI commands (deferred to future change)
- Connection pooling or multiplexing (may add later if needed)
- Message compression or encryption (handled at WebSocket/TLS layer)
- Automatic reconnection logic (client responsibility, not protocol layer)

## Decisions

### Decision: Use text-based control messages, binary for data
**Rationale**: Control messages (WAITING, CONNECT:[ID], HANDLE:[ID]) are infrequent and benefit from human readability during debugging. Data transfer uses binary WebSocket frames for efficiency.

**Alternatives considered**:
- All binary with length prefixes: More complex, no debugging benefit
- All text with base64 data: Inefficient for large payloads

### Decision: Use ULID for connection IDs
**Rationale**: 
- Lexicographically sortable timestamp component
- Cryptographically strong randomness
- 26-character string representation (URL-safe)
- Single server instance friendly (no coordination needed)
- **ULID is for connection matching only, NOT for security** - every connection requires BasicAuth

**Alternatives considered**:
- UUID v4: Not time-sortable, slightly longer string representation
- Sequential integers: Requires coordination, predictable

### Decision: BasicAuth for every connection
**Rationale**:
- **Every WebSocket connection (control and data) requires BasicAuth verification**
- Standard HTTP authentication mechanism
- Well-supported by WebSocket libraries
- Simple to implement and test
- ServiceName as username, ServiceToken as password
- Ensures security even if ULID is intercepted or guessed

**Alternatives considered**:
- Custom headers: Less standard, more implementation work
- JWT tokens: Overkill for this use case, requires token management
- ULID-based authentication: Insecure, ULID is for matching not security

### Decision: Two-connection model (control + data)
**Rationale**:
- Control connection remains open for multiplexing future requests
- Data connections are dedicated, simplifying routing logic
- Clear separation of concerns
- Easy to implement and reason about

**Alternatives considered**:
- Single connection with message framing: More complex state machine
- Connection pooling: Premature optimization

### Decision: Client initiates data connections
**Rationale**:
- Works through NAT and firewalls
- Server cannot initiate connections to client
- Follows WebSocket's client-initiated model

### Decision: Single WebSocket endpoint with first-message routing
**Rationale**:
- Server uses single WebSocket endpoint for both control and data connections
- **Server does not distinguish by URL path** - client can connect to any path
- First text message determines connection type: "WAITING" = control, "HANDLE:[ULID]" = data
- Simplifies server implementation: one upgrade handler, read first message, route accordingly
- Reduces configuration complexity (no need for separate endpoints)
- Client always sends text message as first message on any connection
- **Path-agnostic design enables gateway-level routing** - gateways can route to different server instances without protocol awareness

**Alternatives considered**:
- Separate WebSocket endpoints for control vs data: More configuration, harder to manage
- Custom headers for connection type: Less flexible, harder to test
- Query parameters: Exposes connection ID in URL (less clean)
- Path-based routing: Requires gateway awareness of protocol details

## Protocol Flow

```
Client                          Server (Single WebSocket Endpoint)
  |                               |
  | 1. Connect (BasicAuth)        |
  |------------------------------>|
  |                               | (Verify credentials)
  |                               | (Wait for first text message)
  | 2. Send "WAITING" (text)      |
  |------------------------------>|
  |                               | (Read first message)
  |                               | (Route: WAITING → control handler)
  |                               | (Mark client ready)
  |                               |
  |                     [Incoming request arrives]
  |                               |
  | 3. Receive "CONNECT:[ULID]"   |
  |<------------------------------|
  |                               |
  | 4. New connection (BasicAuth) |
  |------------------------------>|
  |                               | (Verify credentials)
  |                               | (Wait for first text message)
  | 5. Send "HANDLE:[ULID]" (text)|
  |------------------------------>|
  |                               | (Read first message)
  |                               | (Route: HANDLE:[ULID] → data handler)
  |                               | (Match connection to request)
  |                               |
  | 6. Binary data exchange       |
  |<----------------------------->|
  |                               |
```

## Message Format

### Control Messages (Text)
- `WAITING` - Client signals readiness to accept connections
- `CONNECT:[ULID]` - Server assigns connection ID and requests handling
- `HANDLE:[ULID]` - Client claims connection ID on new WebSocket

### Data Messages (Binary)
- Raw binary frames after HANDLE handshake completes

## File Structure

```
wstun/
├── protocol.go           # Shared constants, message types
├── client_protocol.go    # Client protocol implementation
├── server_protocol.go    # Server protocol implementation
└── protocol_test.go      # Protocol tests
```

## Implementation Notes

### Server WebSocket Handler Pattern

The server implementation follows this pattern:

```go
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    // Note: URL path is ignored - any path is acceptable
    
    // 1. Verify BasicAuth (ServiceName, ServiceToken) - REQUIRED for every connection
    serviceName, serviceToken := extractBasicAuth(r)
    if !validateCredentials(serviceName, serviceToken) {
        http.Error(w, "Unauthorized", 401)
        return
    }
    
    // 2. Upgrade to WebSocket
    conn := upgrader.Upgrade(w, r)
    
    // 3. Read first text message (MUST be text frame)
    messageType, message := conn.ReadMessage()
    if messageType != websocket.TextMessage {
        conn.Close()
        return
    }
    
    // 4. Route based on message content
    if message == "WAITING" {
        handleControlConnection(conn, serviceName)
    } else if strings.HasPrefix(message, "HANDLE:") {
        ulid := extractULID(message)
        // Note: ULID is for matching only, security already verified by BasicAuth
        handleDataConnection(conn, serviceName, ulid)
    } else {
        conn.Close() // Invalid first message
    }
}
```

### Client Connection Pattern

Both control and data connections follow the same initialization:

```go
// Control connection - can connect to any path
conn := dialWithBasicAuth(serverURL, serviceName, serviceToken)
conn.WriteMessage(websocket.TextMessage, []byte("WAITING"))

// Data connection - can connect to any path, MUST use same BasicAuth
conn := dialWithBasicAuth(serverURL, serviceName, serviceToken)
conn.WriteMessage(websocket.TextMessage, []byte("HANDLE:"+ulid))

// Note: Both connections require BasicAuth for security
// ULID is only used for matching data connection to pending request
```

## Risks / Trade-offs

### Risk: Connection ID collision
**Mitigation**: ULID provides 128-bit randomness component, collision probability is negligible for expected scale

### Risk: ULID interception or prediction
**Mitigation**: **Every connection requires BasicAuth authentication**. Even if attacker intercepts or predicts ULID, they cannot establish HANDLE connection without valid ServiceName and ServiceToken. ULID is explicitly designed as a matching mechanism, not a security token.

### Risk: Orphaned connections if client disappears
**Mitigation**: WebSocket close events and timeouts (implementation detail for server)

### Risk: Control connection failure
**Mitigation**: Client must detect and reconnect (outside protocol scope)

### Risk: Path-agnostic endpoint exposed to public
**Mitigation**: BasicAuth protects all connections. Server can optionally implement rate limiting or IP filtering at HTTP layer before WebSocket upgrade.

### Trade-off: Two connections per tunneled connection
**Impact**: More WebSocket overhead, but simpler implementation and better separation of concerns. For expected use cases (dozens of concurrent connections), overhead is acceptable.

## Migration Plan

N/A - New capability, no migration required.

## Open Questions

None - requirements are clear and scoped.

