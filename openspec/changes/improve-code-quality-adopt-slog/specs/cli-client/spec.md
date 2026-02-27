## MODIFIED Requirements

### Requirement: Client Logging

The client SHALL log operational events using `*slog.Logger` with structured attributes according to configured log level.

#### Scenario: Client logs at info level

- **WHEN** log_level is "info" or not specified
- **THEN** client logs startup, connection events, reconnection, and shutdown via `slog` at info level
- **AND** each log entry SHALL include a `"service"` attribute with the configured `ServiceName`
- **AND** does not log detailed protocol messages

#### Scenario: Client logs connection failures

- **WHEN** connection to server fails or target connection fails
- **THEN** client logs error via `slog` at error level with `"service"` attribute and `"error"` attribute containing the error value
- **AND** continues with reconnection logic if applicable

#### Scenario: Client logger initialisation

- **WHEN** client starts up
- **THEN** client SHALL construct a `*slog.Logger` using `slog.NewTextHandler` with level derived from `ParseLogLevel(config.LogLevel)`
- **AND** pass the logger to `ClientProtocol`

### Requirement: Client Reconnection

The client SHALL automatically reconnect to server when connection is lost, with exponential backoff.

#### Scenario: Client reconnects on disconnect

- **WHEN** control connection is lost or fails
- **THEN** client waits ReconnectInterval before reconnecting
- **AND** uses exponential backoff for subsequent attempts (2x each time, capped at 60s)
- **AND** continues reconnecting until successful or max attempts reached

#### Scenario: Client respects max reconnect attempts

- **WHEN** MaxReconnectAttempts is configured (> 0)
- **THEN** client attempts reconnection up to that many times
- **AND** exits with error if all attempts fail
- **WHEN** MaxReconnectAttempts is 0
- **THEN** client reconnects indefinitely

#### Scenario: Client logs reconnection attempts

- **WHEN** reconnection is attempted
- **THEN** client logs attempt number and delay via `slog` at warn level with `"service"`, `"attempt"`, and `"delay"` attributes
- **AND** logs successful reconnection via `slog` at info level

## REMOVED Requirements

### Requirement: shouldLog helper function

**Reason**: Replaced by `slog` built-in level filtering via `slog.HandlerOptions.Level`. The `shouldLog` function is no longer needed.

**Migration**: Remove `shouldLog` from `cmd/wstun-client/main.go`. Use `wstun.ParseLogLevel` to obtain the `slog.Level` and pass it to `slog.HandlerOptions{Level: level}` when constructing the handler.
