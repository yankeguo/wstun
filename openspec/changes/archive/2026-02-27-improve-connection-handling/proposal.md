## Why

The system intermittently produces `upstream connect error or disconnect/reset before headers. reset reason: connection termination` from proxies/load-balancers. This happens because WebSocket control connections silently die under proxy idle-timeout policies, write operations have no deadline on the server side (causing goroutine hangs), and dead control connections are never removed from the registry—leaving subsequent `RequestConnection` calls to silently drop their CONNECT messages into the void.

## What Changes

- Server SHALL send periodic ping frames on all control connections to keep them alive through proxies with idle-timeout policies.
- All WebSocket writes (control messages, CONNECT messages, ping frames) SHALL carry an explicit write deadline to prevent goroutines from blocking indefinitely.
- `sendToControlConnections` SHALL remove dead control connections immediately upon write failure, rather than leaving stale entries in the registry.
- `RequestConnection` SHALL skip retry attempts immediately when no healthy control connection exists for the target service, rather than sleeping through retry intervals in vain.
- The data-connection transfer loop write-deadline on read (`PongWait`) is misleading—data connections have their own full-duplex lifecycle independent of pong; the read deadline on data connections SHALL be removed or made configurable separately to prevent spurious 60-second timeouts on legitimately idle tunnels.
- Server goroutines started by `handleControlConnection` and `RequestConnection` SHALL not leak when the control connection shuts down mid-retry.

## Capabilities

### New Capabilities

- `connection-health`: Requirements for server-side keepalive pings on control connections, mandatory write deadlines on all WebSocket writes, and immediate removal of dead control connections from the registry.

### Modified Capabilities

- `websocket-protocol`: Add requirements for write deadlines on all WebSocket text/binary writes and for server-side keepalive pings on the control connection channel.

## Impact

- `server_protocol.go`: `sendToControlConnections`, `handleControlConnection`, `RequestConnection`, `NewServerProtocol`
- `client_protocol.go`: `transfer` (data-connection read-deadline handling)
- `openspec/specs/websocket-protocol/spec.md`: New scenarios for write deadlines and server-side keepalive
- `openspec/specs/connection-health/spec.md`: New spec (created by this change)
