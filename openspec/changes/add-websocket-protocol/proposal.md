# WebSocket Protocol Encapsulation Proposal

## Why
The wstun project needs a WebSocket protocol layer to enable tunneled connections between clients and servers. This protocol will handle authentication, connection lifecycle management, and bidirectional data transfer while remaining decoupled from the CLI commands for independent testing and reusability.

## What Changes
- Add WebSocket protocol message types and constants (WAITING, CONNECT, HANDLE)
- Implement client-side WebSocket protocol handler for establishing and managing connections
- Implement server-side WebSocket protocol handler for accepting and routing connections
- Use ULID for unique connection identification
- Support BasicAuth for initial service authentication
- Enable bidirectional binary message exchange on established connections
- Create protocol layer that is independent and can be tested in isolation

## Impact
- Affected specs: new capability `websocket-protocol`
- Affected code: new files in wstun package (protocol.go, client_protocol.go, server_protocol.go)
- No changes to existing CLI commands or configuration (integration deferred)
- Foundation for future connection handling in wstun-server and wstun-client

