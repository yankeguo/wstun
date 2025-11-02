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

**Alternatives considered**:
- UUID v4: Not time-sortable, slightly longer string representation
- Sequential integers: Requires coordination, predictable

### Decision: BasicAuth for initial authentication
**Rationale**:
- Standard HTTP authentication mechanism
- Well-supported by WebSocket libraries
- Simple to implement and test
- ServiceName as username, ServiceToken as password

**Alternatives considered**:
- Custom headers: Less standard, more implementation work
- JWT tokens: Overkill for this use case, requires token management

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

## Protocol Flow

```
Client                          Server
  |                               |
  | 1. Connect (BasicAuth)        |
  |------------------------------>|
  |                               | (Verify credentials)
  |                               |
  | 2. Send "WAITING"             |
  |------------------------------>|
  |                               | (Mark client ready)
  |                               |
  |                     [Incoming request arrives]
  |                               |
  | 3. Receive "CONNECT:[ULID]"   |
  |<------------------------------|
  |                               |
  | 4. New connection (BasicAuth) |
  |------------------------------>|
  |                               |
  | 5. Send "HANDLE:[ULID]"       |
  |------------------------------>|
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

## Risks / Trade-offs

### Risk: Connection ID collision
**Mitigation**: ULID provides 128-bit randomness component, collision probability is negligible for expected scale

### Risk: Orphaned connections if client disappears
**Mitigation**: WebSocket close events and timeouts (implementation detail for server)

### Risk: Control connection failure
**Mitigation**: Client must detect and reconnect (outside protocol scope)

### Trade-off: Two connections per tunneled connection
**Impact**: More WebSocket overhead, but simpler implementation and better separation of concerns. For expected use cases (dozens of concurrent connections), overhead is acceptable.

## Migration Plan

N/A - New capability, no migration required.

## Open Questions

None - requirements are clear and scoped.

