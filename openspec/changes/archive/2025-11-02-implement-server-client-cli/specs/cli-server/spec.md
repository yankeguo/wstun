## ADDED Requirements

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

