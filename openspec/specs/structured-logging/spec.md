```markdown
# structured-logging Specification

## Purpose

TBD - created by syncing change improve-code-quality-adopt-slog.

## Requirements

### Requirement: slog Logger Acceptance

The `ServerProtocol` and `ClientProtocol` types SHALL accept a `*slog.Logger` via a public `Logger` field. When the field is `nil`, the types SHALL fall back to `slog.Default()`.

#### Scenario: Logger field set

- **WHEN** caller sets `ServerProtocol.Logger` or `ClientProtocol.Logger` to a non-nil `*slog.Logger`
- **THEN** all log output from that instance SHALL use the provided logger

#### Scenario: Logger field nil

- **WHEN** `Logger` field is `nil` (zero-value)
- **THEN** the instance SHALL use `slog.Default()` for all log output

### Requirement: Log Level Parsing

The `wstun` package SHALL provide a `ParseLogLevel(s string) slog.Level` function that maps configuration strings to `slog.Level` values.

#### Scenario: Valid level strings

- **WHEN** input is `"debug"`, `"info"`, `"warn"`, or `"error"` (case-insensitive)
- **THEN** the function SHALL return `slog.LevelDebug`, `slog.LevelInfo`, `slog.LevelWarn`, or `slog.LevelError` respectively

#### Scenario: Empty or unknown level string

- **WHEN** input is empty or an unrecognised string
- **THEN** the function SHALL return `slog.LevelInfo` as the default

### Requirement: Structured Log Attributes

All log entries emitted by the library SHALL include structured key-value attributes instead of interpolated format strings.

#### Scenario: Service attribute on server

- **WHEN** `ServerProtocol` logs an event related to a specific service
- **THEN** the log entry SHALL include an attribute with key `"service"` and value equal to the authenticated service name

#### Scenario: Connection ID attribute

- **WHEN** `ServerProtocol` or `ClientProtocol` logs an event related to a specific connection
- **THEN** the log entry SHALL include an attribute with key `"conn_id"` and value equal to the ULID connection identifier

#### Scenario: Client service attribute

- **WHEN** `ClientProtocol` logs any event
- **THEN** the log entry SHALL include an attribute with key `"service"` and value equal to the configured `ServiceName`

### Requirement: Log Level Usage

Log statements SHALL use the appropriate `slog` level method for their severity.

#### Scenario: Info-level events

- **WHEN** an expected operational event occurs (connection established, shutdown started, reconnection succeeded)
- **THEN** the event SHALL be logged at `slog.LevelInfo`

#### Scenario: Error-level events

- **WHEN** an unexpected failure occurs (authentication failure, write error, panic recovery)
- **THEN** the event SHALL be logged at `slog.LevelError`

#### Scenario: Warn-level events

- **WHEN** a degraded but recoverable situation occurs (keepalive ping failure, retry attempt, no handler configured)
- **THEN** the event SHALL be logged at `slog.LevelWarn`

#### Scenario: Debug-level events

- **WHEN** verbose diagnostic information is emitted (message routing details, data transfer byte counts)
- **THEN** the event SHALL be logged at `slog.LevelDebug`

### Requirement: No logf Helper

The `ServerProtocol` and `ClientProtocol` types SHALL NOT contain a `logf` helper method. All logging SHALL be performed by calling methods directly on the `*slog.Logger` field (or `slog.Default()`).

#### Scenario: logf method absent

- **WHEN** inspecting the exported and unexported methods of `ServerProtocol` or `ClientProtocol`
- **THEN** no method named `logf` SHALL exist
```
