## MODIFIED Requirements

### Requirement: Connection Lifecycle Management

The system SHALL properly manage WebSocket connection lifecycle including closure and error handling.

#### Scenario: Normal connection closure from client

- **WHEN** client closes data connection normally
- **THEN** server receives close frame
- **AND** server closes corresponding tunneled connection
- **AND** removes connection from active connections

#### Scenario: Normal connection closure from server

- **WHEN** server closes data connection normally
- **THEN** client receives close frame
- **AND** client stops data transfer on that connection
- **AND** removes connection from active connections

#### Scenario: Control connection closed

- **WHEN** control connection is closed or lost
- **THEN** all associated data connections are identified
- **AND** client or server cleans up associated resources
- **AND** client may attempt to re-establish control connection

#### Scenario: WebSocket error handling

- **WHEN** WebSocket error occurs on any connection
- **THEN** error is logged as a structured `slog` entry at error level with `"service"` and `"conn_id"` attributes where applicable
- **AND** connection is closed gracefully if possible
- **AND** resources are cleaned up

### Requirement: Graceful Error Handling

The system SHALL handle errors gracefully without crashing or leaking resources.

#### Scenario: Nil handler protection

- **WHEN** ConnectHandler or DataHandler is nil
- **THEN** system logs warning via `slog` at warn level and gracefully skips operation
- **AND** does not panic with nil pointer dereference

#### Scenario: Panic recovery in goroutines

- **WHEN** unexpected panic occurs in data transfer goroutine
- **THEN** system recovers from panic using defer/recover
- **AND** logs panic details via `slog` at error level with `"conn_id"` attribute
- **AND** closes associated connections

#### Scenario: Write to closed connection

- **WHEN** system attempts to write to already-closed WebSocket
- **THEN** error is caught and logged via `slog` at error level
- **AND** system does not crash
- **AND** associated resources are cleaned up

### Requirement: Server WebSocket Endpoint and Connection Routing

The server SHALL provide a single path-agnostic WebSocket endpoint that handles both control and data connections, routing them based on the first text message received.

#### Scenario: Single endpoint for all connections

- **WHEN** server is configured
- **THEN** server provides one WebSocket upgrade endpoint
- **AND** both control and data connections use the same endpoint URL

#### Scenario: Path-agnostic connection handling

- **WHEN** client connects to server WebSocket endpoint
- **THEN** server accepts connections on any URL path
- **AND** URL path is ignored during connection handling
- **AND** routing is based solely on first message content

#### Scenario: First message determines connection type

- **WHEN** server accepts a WebSocket connection
- **THEN** server waits for and reads the first text message
- **AND** routes to control handler if message is "WAITING"
- **AND** routes to data handler if message starts with "HANDLE:"

#### Scenario: Simplified server logic

- **WHEN** implementing WebSocket handler
- **THEN** server uses unified upgrade handler for all connections
- **AND** single authentication check applies to all connections
- **AND** first message routing eliminates need for separate endpoints

#### Scenario: Invalid first message type

- **WHEN** server receives first message that is not a text frame
- **THEN** server closes the WebSocket connection with error
- **AND** logs the invalid message type via `slog` at error level with `"service"` attribute

#### Scenario: Unrecognized first message content

- **WHEN** server reads first text message that is neither "WAITING" nor starts with "HANDLE:"
- **THEN** server closes the WebSocket connection with error
- **AND** logs the unrecognized message content via `slog` at error level with `"service"` attribute
