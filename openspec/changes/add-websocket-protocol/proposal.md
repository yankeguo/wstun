# WebSocket Protocol Encapsulation Proposal

## Why
The wstun project needs a WebSocket protocol layer to enable tunneled connections between clients and servers. This protocol will handle authentication, connection lifecycle management, and bidirectional data transfer while remaining decoupled from the CLI commands for independent testing and reusability.

## What Changes
- Add WebSocket protocol message types and constants (WAITING, CONNECT, HANDLE)
- Implement unified server WebSocket endpoint with first-message routing (WAITING = control, HANDLE:[ULID] = data)
- Implement client-side WebSocket protocol handler for establishing and managing connections
- Implement server-side WebSocket protocol handler for accepting and routing connections
- Use ULID for unique connection identification
- Support BasicAuth for initial service authentication
- Enable bidirectional binary message exchange on established connections
- Create protocol layer that is independent and can be tested in isolation

**Key Design Principles**:
1. **Single endpoint, path-agnostic**: Server accepts WebSocket connections on any URL path; first text message routes to control or data handler
2. **Security**: Every connection (control and data) requires BasicAuth; ULID is for connection matching only, NOT for security
3. **Gateway-friendly**: Path-agnostic design allows gateways to route to different server instances without protocol awareness

## Impact
- Affected specs: new capability `websocket-protocol`
- Affected code: new files in wstun package (protocol.go, client_protocol.go, server_protocol.go)
- No changes to existing CLI commands or configuration (integration deferred)
- Foundation for future connection handling in wstun-server and wstun-client

