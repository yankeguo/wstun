## Context

`wstun` uses a two-tier WebSocket architecture: a persistent _control connection_ (WAITING) and short-lived _data connections_ (HANDLE). In production the control connection typically passes through an L7 proxy (Envoy, nginx, AWS ALB) that enforces an idle-connection timeout. When no traffic flows the proxy silently resets the TCP stream, producing the `upstream connect error … connection termination` message observed downstream.

Current pain-points identified in the code:

- **No server-side ping on control connections.** The client already sends pings (`pingLoop`), but the server never does. If a proxy sits between client and server, the idle-detector on the server side of the proxy fires first.
- **No write deadline on `sendToControlConnections`.** A hanging write goroutine blocks `RequestConnection` indefinitely even though the connection is already dead.
- **Dead control connections not pruned on write error.** `sendToControlConnections` logs the failure but leaves the stale `*controlConnection` in the `sync.Map`, so subsequent CONNECT messages are sent into the void.
- **Data-connection read deadline re-uses `PongWait`.** The `transfer` goroutine calls `ws.SetReadDeadline(time.Now().Add(cp.PongWait))` on every iteration. For legitimate long-lived idle tunnels (e.g. keep-alive HTTP) this causes a spurious 60-second timeout.
- **`RequestConnection` retry-sleep ignores connection health.** The code sleeps for exponential-backoff between retries even when no control connection is registered at all.

## Goals / Non-Goals

**Goals:**

- Prevent control connections from being silently killed by proxy idle-timeout by adding server-side periodic pings.
- Prevent goroutine hangs by applying write deadlines to every server-side WebSocket write.
- Keep the control-connection registry clean by removing dead connections on first write error.
- Make data-connection read behaviour correct for idle-but-valid tunnels.
- Fast-fail `RequestConnection` when no control connection exists.

**Non-Goals:**

- Changing the wire protocol (message format, auth scheme, endpoint URL).
- Adding TLS or proxy-protocol passthrough.
- Changing the retry/backoff configuration API exposed to callers.

## Decisions

### D1 — Server-side ping on control connections via `handleControlConnection`

**Decision**: Add a background goroutine inside `handleControlConnection` that sends `websocket.PingMessage` every N seconds (default 15 s, configurable via `ControlPingInterval`). Use the existing `controlConnection.writeMu` mutex for every write—including the ping—to serialise writes.

**Rationale**: Proxy idle timeouts are commonly set to 60–300 s. A 15 s ping comfortably keeps the connection alive across all common defaults without significant bandwidth overhead. The client already tracks server pings via its `PongWait`-based read deadline, so this requires no client-side change.

**Alternative considered**: Have the _client_ increase its ping frequency further. Rejected because the idle-detection asymmetry lives on the server side of the proxy; a client ping travels _through_ the proxy whereas a server ping originates _from_ behind it, directly activating the server-proxy leg.

### D2 — Write deadline on all server-side WebSocket writes

**Decision**: Before every `WriteMessage` / `WriteControl` call in `sendToControlConnections`, set a write deadline of `WriteWait` (default 10 s) via `cc.conn.SetWriteDeadline`. The `WriteWait` constant is already compiled into `ClientProtocol`; add an exported `WriteWait` field to `ServerProtocol` with the same default.

**Rationale**: Without a write deadline a dead TCP connection causes `WriteMessage` to block until the OS-level TCP timeout fires (minutes to hours). This blocks `RequestConnection` and leaks goroutines.

### D3 — Prune dead control connections on write failure

**Decision**: When `sendToControlConnections` fails to write to a `controlConnection`, call `conns.Delete(cc)` immediately after logging. Move the connection cleanup logic into a dedicated helper `removeControlConnection(serviceName, cc)`.

**Rationale**: Leaving stale entries means CONNECT messages are delivered to connections that will never respond, causing silent retry exhaustion. Removing eagerly is safe because a concurrent read in `handleControlConnection` will independently detect the close and call `unregisterControlConnection`.

### D4 — Remove per-read deadline from data-connection transfer loops

**Decision**: Remove the `ws.SetReadDeadline(time.Now().Add(cp.PongWait))` call inside the `ws→target` goroutine in `transfer`. The connection is already kept alive by the OS TCP stack; if the tunnel peer silently dies the eventual TCP reset surfaces as a read error naturally.

**Alternative considered**: Keep the read deadline but reset it on every successful read. Rejected because there is no symmetric ping mechanism on data connections, so the deadline would fire on any idle but live tunnel.

**Trade-off**: We lose automatic detection of half-open data connections. Acceptable because data connections are short-lived by design (one HTTP request) and the proxy enforces its own response timeout independently.

### D5 — Fast-fail RequestConnection when no control connection exists

**Decision**: Before sending the CONNECT message in `RequestConnection`, check whether any control connections are registered for `serviceName`. If the map is empty, skip the sleep and fail the attempt immediately (counting it as an exhausted attempt).

**Rationale**: Sleeping for `RetryInterval` when no client is connected wastes time and keeps the caller's goroutine parked. The incoming connection still gets closed after `RetryAttempts` exhausted, preserving existing behaviour.

## Risks / Trade-offs

- **[Risk] Ping goroutine leak if ping fails silently.** If a ping write error occurs but `handleControlConnection` doesn't notice (e.g. the read loop hasn't exited yet), the ping goroutine must exit cleanly. → Mitigation: Ping goroutine checks a `done` channel closed by `handleControlConnection`'s deferred cleanup; on write error the goroutine exits and relies on the read loop to trigger `unregisterControlConnection`.
- **[Risk] Write-deadline interferes with slow clients on slow links.** 10 s write deadline may be too tight for high-latency links. → Mitigation: `WriteWait` is already configurable; document recommended tuning in README.
- **[Risk] Race between write-failure pruning (D3) and `unregisterControlConnection` (called from read loop).** Both sides may delete the same key concurrently. → Mitigation: `sync.Map.Delete` is idempotent; deleting an already-deleted key is a no-op.

## Migration Plan

1. Add `ControlPingInterval` and `WriteWait` fields to `ServerProtocol`; set defaults in `NewServerProtocol`.
2. Modify `handleControlConnection` to launch ping goroutine.
3. Modify `sendToControlConnections` to set write deadline and prune on failure.
4. Modify `transfer` in `client_protocol.go` to remove per-read deadline.
5. Modify `RequestConnection` to fast-fail when control registry is empty.
6. Update `openspec/specs/` accordingly.
7. All changes are backward-compatible—no config file or API contract changes.
