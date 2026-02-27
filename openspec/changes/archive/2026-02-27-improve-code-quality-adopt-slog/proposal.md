## Why

The codebase currently uses the stdlib `log.Logger` for all logging, which produces flat, unstructured text and makes it hard to filter or correlate log lines by service. Code organisation also has room for improvement: some files are large, helper logic is mixed with core logic, and naming conventions are inconsistent across packages.

## What Changes

- **BREAKING** Replace `*log.Logger` field on `ServerProtocol` and `ClientProtocol` with `*slog.Logger` to enable structured, levelled logging throughout the library.
- Server-side log entries include a `service` attribute so each line is traceable to the originating service name.
- Client-side log entries include a `service` attribute mirroring the `ServiceName` configuration field.
- Remove the custom `logf` helper wrappers; call `slog.Logger` methods directly with structured key-value pairs.
- Split `cmd/wstun-server/main.go` and `cmd/wstun-client/main.go` into focused files (`server.go`, `listener.go`, `client.go`, `connector.go`, etc.) following single-responsibility principles.
- Rename internal variables and unexported identifiers to follow idiomatic Go naming (`cc` → `ctrlConn`, `sp` field names, etc.).
- Consolidate duplicated `logPrefix` / `validLogLevels` / `SetDefaults` patterns so shared logic lives in one place.
- Update `LogLevel` wiring to initialise `slog` at the correct level via `slog.LevelVar` or `slog.HandlerOptions`.

## Capabilities

### New Capabilities

- `structured-logging`: Covers the slog adoption contract — how `slog.Logger` is accepted by the library types, how log levels are mapped, and the required structured attributes (`service`, `conn_id`, etc.).

### Modified Capabilities

- `websocket-protocol`: `ServerProtocol.Logger` field type changes from `*log.Logger` to `*slog.Logger` (**BREAKING**); log call-sites gain structured key-value pairs.
- `cli-client`: Logger construction changes from `log.New(…)` to `slog.New(…)`; `LogLevel` string now drives `slog.LevelVar`.
- `cli-server`: Same logger construction change as cli-client; service-name attribute attached at handler level.

## Impact

- **Library API**: `ServerProtocol.Logger` and `ClientProtocol.Logger` field types change — callers must update to pass `*slog.Logger`.
- **cmd packages**: Both `cmd/wstun-client` and `cmd/wstun-server` require updated logger initialisation.
- **Dependencies**: No new external dependencies; `log/slog` is part of the Go standard library (Go 1.21+).
- **Tests**: Integration and unit tests that construct `ServerProtocol` or `ClientProtocol` with a logger must be updated.
- **go.mod**: Minimum Go version must be ≥ 1.21 to use `log/slog`; verify and update `go.mod` if needed.
