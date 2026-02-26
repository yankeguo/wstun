# cli-server Specification

## Purpose
TBD - created by archiving change add-cli-commands. Update Purpose after archive.
## Requirements
### Requirement: Server Command Binary
The system SHALL provide a `wstun-server` command-line executable that runs a WebSocket tunnel server.

#### Scenario: Server binary exists
- **WHEN** the project is built
- **THEN** a `wstun-server` binary is produced in `cmd/wstun-server/`

#### Scenario: Server displays help text
- **WHEN** `wstun-server -h` or `wstun-server --help` is executed
- **THEN** usage information and available flags are displayed

### Requirement: Configuration File Flag
The `wstun-server` command SHALL accept a `-c` flag that specifies the path to a configuration file.

#### Scenario: Valid configuration file provided
- **WHEN** `wstun-server -c /path/to/config.yaml` is executed with a valid config file
- **THEN** the server loads configuration from the specified file
- **AND** the server starts successfully

#### Scenario: Configuration file not found
- **WHEN** `wstun-server -c /nonexistent/config.yaml` is executed
- **THEN** an error message is displayed indicating the file cannot be found
- **AND** the server exits with a non-zero status code

#### Scenario: No configuration file specified
- **WHEN** `wstun-server` is executed without the `-c` flag
- **THEN** an error message is displayed indicating configuration is required
- **AND** the server exits with a non-zero status code

### Requirement: Graceful Shutdown
The `wstun-server` command SHALL handle system signals for graceful shutdown.

#### Scenario: SIGINT received
- **WHEN** the server is running and receives SIGINT (Ctrl+C)
- **THEN** the server initiates graceful shutdown
- **AND** existing connections are closed cleanly
- **AND** the server exits with status code 0

#### Scenario: SIGTERM received
- **WHEN** the server is running and receives SIGTERM
- **THEN** the server initiates graceful shutdown
- **AND** existing connections are closed cleanly
- **AND** the server exits with status code 0

### Requirement: Server Runtime Operation
The server SHALL load service definitions, start TCP listeners for each service, and handle tunneling connections via WebSocket protocol.

#### Scenario: Server loads services on startup
- **WHEN** server starts with valid configuration
- **THEN** all service definition files are loaded from ServicesDir
- **AND** TCP listener is created for each service's ListenAddr
- **AND** HTTP/HTTPS server is started on server's ListenAddr

#### Scenario: Server accepts incoming connections
- **WHEN** TCP connection arrives on a service's ListenAddr
- **THEN** server generates ULID and sends CONNECT message via WebSocket
- **AND** waits for client to establish data connection with HANDLE message
- **AND** bridges TCP connection with WebSocket for bidirectional data transfer

#### Scenario: Server authenticates WebSocket connections
- **WHEN** WebSocket connection is established
- **THEN** server verifies BasicAuth credentials against loaded service definitions
- **AND** accepts connection if credentials match a known service
- **AND** rejects connection with 401 if credentials are invalid

#### Scenario: Server enforces max connections
- **WHEN** max_connections is configured and limit is reached
- **THEN** server rejects new TCP connections until existing connections close
- **AND** logs warning about connection limit reached

### Requirement: Server Graceful Shutdown
The server SHALL cleanly shut down all listeners and connections when receiving shutdown signal.

#### Scenario: Shutdown closes resources
- **WHEN** server receives SIGINT or SIGTERM
- **THEN** server closes all TCP listeners (stops accepting new connections)
- **AND** closes HTTP server gracefully
- **AND** waits for active connections to finish (with timeout)
- **AND** logs shutdown progress and completion

### Requirement: Server Logging
The server SHALL log operational events according to configured log level.

#### Scenario: Server logs at info level
- **WHEN** log_level is "info" or not specified
- **THEN** server logs startup, service loading, connection events, and shutdown
- **AND** does not log detailed protocol messages

#### Scenario: Server logs errors
- **WHEN** errors occur (service load failure, listener failure, connection errors)
- **THEN** server logs error messages with context
- **AND** continues operating when possible (e.g., one service fails, others continue)

### Requirement: Retry configuration via configuration file
The `wstun-server` command SHALL load retry configuration from the configuration file.

#### Scenario: Configuration file with retry settings
- **WHEN** configuration file contains `retry_attempts: 3`, `retry_interval: 5s`, `retry_max_interval: 60s`
- **THEN** server SHALL load these retry settings
- **AND** apply them to connection request handling

#### Scenario: Default retry behavior
- **WHEN** configuration file does not contain retry settings
- **THEN** `RetryAttempts` SHALL default to 0 (no retry)
- **AND** `RetryInterval` SHALL default to 5s
- **AND** `RetryMaxInterval` SHALL default to 60s

#### Scenario: Partial retry configuration
- **WHEN** configuration file contains only `retry_attempts: 2`
- **THEN** server SHALL use `retry_attempts: 2`
- **AND** `RetryInterval` SHALL default to 5s
- **AND** `RetryMaxInterval` SHALL default to 60s

