## MODIFIED Requirements

### Requirement: Server Logging

The server SHALL log operational events using `*slog.Logger` with structured attributes according to configured log level.

#### Scenario: Server logs at info level

- **WHEN** log_level is "info" or not specified
- **THEN** server logs startup, service loading, connection events, and shutdown via `slog` at info level
- **AND** does not log detailed protocol messages

#### Scenario: Server logs errors

- **WHEN** errors occur (service load failure, listener failure, connection errors)
- **THEN** server logs error messages via `slog` at error level with `"error"` attribute containing the error value
- **AND** continues operating when possible (e.g., one service fails, others continue)

#### Scenario: Server logger initialisation

- **WHEN** server starts up
- **THEN** server SHALL construct a `*slog.Logger` using `slog.NewTextHandler` with level derived from `ParseLogLevel(config.LogLevel)`
- **AND** pass the logger to `ServerProtocol`

#### Scenario: Server per-service log attribute

- **WHEN** server handles a connection for a specific service
- **THEN** server SHALL derive a child logger via `logger.With("service", serviceName)`
- **AND** use the child logger for all log entries related to that service

## REMOVED Requirements

### Requirement: shouldLog helper function

**Reason**: Replaced by `slog` built-in level filtering via `slog.HandlerOptions.Level`. The `shouldLog` function is no longer needed.

**Migration**: Remove `shouldLog` from `cmd/wstun-server/main.go`. Use `wstun.ParseLogLevel` to obtain the `slog.Level` and pass it to `slog.HandlerOptions{Level: level}` when constructing the handler.
