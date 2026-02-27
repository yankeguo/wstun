## 1. ServerProtocol configuration fields

- [x] 1.1 Add `ControlPingInterval time.Duration` field to `ServerProtocol` struct in `server_protocol.go`
- [x] 1.2 Add `WriteWait time.Duration` field to `ServerProtocol` struct in `server_protocol.go`
- [x] 1.3 Set `ControlPingInterval: 15 * time.Second` and `WriteWait: 10 * time.Second` defaults in `NewServerProtocol`

## 2. Server-side keepalive ping on control connections (D1)

- [x] 2.1 Add a `pingControlConnection` goroutine function to `server_protocol.go` that accepts a `done <-chan struct{}` and a `*controlConnection`, ticks at `ControlPingInterval`, and sends `websocket.PingMessage` using the `controlConnection.writeMu` mutex with a `WriteWait` write deadline
- [x] 2.2 In `handleControlConnection`, create a `done` channel, start `pingControlConnection` as a goroutine, and close `done` in the deferred cleanup so the ping goroutine exits when the control connection closes
- [x] 2.3 Verify the ping goroutine exits cleanly when the control connection's read loop exits (no goroutine leak)

## 3. Write deadline on CONNECT messages (D2)

- [x] 3.1 In `sendToControlConnections`, call `cc.conn.SetWriteDeadline(time.Now().Add(sp.WriteWait))` inside the `cc.writeMu` lock, immediately before `cc.conn.WriteMessage`

## 4. Dead control connection pruning on write failure (D3)

- [x] 4.1 Add helper method `removeControlConnection(serviceName string, cc *controlConnection)` to `ServerProtocol` that loads the service's `sync.Map` and deletes `cc`
- [x] 4.2 In `sendToControlConnections`, call `sp.removeControlConnection(serviceName, cc)` when `WriteMessage` returns a non-nil error

## 5. Fast-fail RequestConnection when no control connection exists (D5)

- [x] 5.1 Add helper method `hasControlConnections(serviceName string) bool` to `ServerProtocol` that returns true if the service's control connection map is non-empty
- [x] 5.2 In the `RequestConnection` retry loop, call `hasControlConnections` before sending the CONNECT message; if it returns false, skip `sendToControlConnections`, count the attempt as exhausted immediately (do not sleep), and continue to the retry-exceeded check

## 6. Remove per-read deadline on data connections (D4)

- [x] 6.1 In `client_protocol.go` `transfer` method, remove the `ws.SetReadDeadline(time.Now().Add(cp.PongWait))` call from inside the WebSocket→target read goroutine loop

## 7. Tests

- [x] 7.1 Add a test in `edge_cases_test.go` (or a new file) verifying that `sendToControlConnections` prunes a dead control connection after a write failure
- [x] 7.2 Add a test verifying `RequestConnection` returns immediately (without sleeping) when no control connection is registered
- [x] 7.3 Add a test verifying a data connection transfer does not error due to read deadline when no data is exchanged for longer than `PongWait`
- [x] 7.4 Run `go test ./...` and confirm all tests pass

## 8. Spec sync

- [x] 8.1 Run `openspec sync --change "improve-connection-handling"` (or prepare for archive) to merge delta specs into main specs once implementation is verified
