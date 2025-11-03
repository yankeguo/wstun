## ADDED Requirements

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
- **THEN** client logs attempt number and delay
- **AND** logs successful reconnection

### Requirement: Client Graceful Shutdown
The client SHALL cleanly shut down connections when receiving shutdown signal.

#### Scenario: Shutdown closes resources
- **WHEN** client receives SIGINT or SIGTERM
- **THEN** client closes control connection gracefully
- **AND** allows in-flight data transfers to complete
- **AND** logs shutdown completion
- **AND** exits with status code 0

### Requirement: Client Logging
The client SHALL log operational events according to configured log level.

#### Scenario: Client logs at info level
- **WHEN** log_level is "info" or not specified
- **THEN** client logs startup, connection events, reconnection, and shutdown
- **AND** does not log detailed protocol messages

#### Scenario: Client logs connection failures
- **WHEN** connection to server fails or target connection fails
- **THEN** client logs error with context (server URL, target address, error details)
- **AND** continues with reconnection logic if applicable

