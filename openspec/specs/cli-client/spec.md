# cli-client Specification

## Purpose

TBD - created by archiving change add-cli-commands. Update Purpose after archive.

## Requirements

### Requirement: Client Command Binary

The system SHALL provide a `wstun-client` command-line executable that runs a WebSocket tunnel client.

#### Scenario: Client binary exists

- **WHEN** the project is built
- **THEN** a `wstun-client` binary is produced in `cmd/wstun-client/`

#### Scenario: Client displays help text

- **WHEN** `wstun-client -h` or `wstun-client --help` is executed
- **THEN** usage information and available flags are displayed

### Requirement: Configuration File Flag

The `wstun-client` command SHALL accept a `-c` flag that specifies the path to a configuration file.

#### Scenario: Valid configuration file provided

- **WHEN** `wstun-client -c /path/to/config.yaml` is executed with a valid config file
- **THEN** the client loads configuration from the specified file
- **AND** the client starts successfully

#### Scenario: Configuration file not found

- **WHEN** `wstun-client -c /nonexistent/config.yaml` is executed
- **THEN** an error message is displayed indicating the file cannot be found
- **AND** the client exits with a non-zero status code

#### Scenario: No configuration file specified

- **WHEN** `wstun-client` is executed without the `-c` flag
- **THEN** an error message is displayed indicating configuration is required
- **AND** the client exits with a non-zero status code

### Requirement: Graceful Shutdown

The `wstun-client` command SHALL handle system signals for graceful shutdown.

#### Scenario: SIGINT received

- **WHEN** the client is running and receives SIGINT (Ctrl+C)
- **THEN** the client initiates graceful shutdown
- **AND** existing connections are closed cleanly
- **AND** the client exits with status code 0

#### Scenario: SIGTERM received

- **WHEN** the client is running and receives SIGTERM
- **THEN** the client initiates graceful shutdown
- **AND** existing connections are closed cleanly
- **AND** the client exits with status code 0

### Requirement: Client Runtime Operation

The client SHALL establish WebSocket control connection to server, handle CONNECT requests, and tunnel traffic to target service.

#### Scenario: Client establishes control connection

- **WHEN** client starts with valid configuration
- **THEN** client connects to ServerURL via WebSocket with BasicAuth
- **AND** sends WAITING message to indicate readiness
- **AND** waits for CONNECT messages from server

#### Scenario: Client handles connection requests

- **WHEN** client receives CONNECT message with ULID
- **THEN** client establishes TCP connection to TargetAddr
- **AND** opens new WebSocket data connection with BasicAuth
- **AND** sends HANDLE message with ULID
- **AND** performs bidirectional data transfer between TCP and WebSocket

#### Scenario: Client respects TLS settings

- **WHEN** TLSInsecure is false (default)
- **THEN** client verifies server TLS certificate
- **WHEN** TLSInsecure is true
- **THEN** client skips TLS certificate verification

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

### Requirement: Client Graceful Shutdown

The client SHALL cleanly shut down connections when receiving shutdown signal.

#### Scenario: Shutdown closes resources

- **WHEN** client receives SIGINT or SIGTERM
- **THEN** client closes control connection gracefully
- **AND** allows in-flight data transfers to complete
- **AND** logs shutdown completion
- **AND** exits with status code 0

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
