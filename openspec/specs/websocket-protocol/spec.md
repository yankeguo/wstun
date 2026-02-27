# websocket-protocol Specification

## Purpose

TBD - created by archiving change add-websocket-protocol. Update Purpose after archive.

## Requirements

### Requirement: Protocol Message Types

The system SHALL define message type constants for WebSocket protocol communication.

#### Scenario: Message constants exist

- **WHEN** examining protocol constants
- **THEN** `MessageTypeWaiting` constant exists with value "WAITING"
- **AND** `MessageTypePrefixConnect` constant exists with value "CONNECT:"
- **AND** `MessageTypePrefixHandle` constant exists with value "HANDLE:"

#### Scenario: Message format validation

- **WHEN** validating protocol messages
- **THEN** CONNECT messages have format "CONNECT:[ULID]"
- **AND** HANDLE messages have format "HANDLE:[ULID]"
- **AND** ULID is exactly 26 characters

### Requirement: Authentication and Security Model

The system SHALL require BasicAuth authentication for every WebSocket connection, with ULID used only for connection matching, not security.

#### Scenario: Every connection requires BasicAuth

- **WHEN** any WebSocket connection is established (control or data)
- **THEN** BasicAuth with ServiceName and ServiceToken MUST be provided
- **AND** server verifies credentials before accepting connection
- **AND** connection is rejected if authentication fails

#### Scenario: ULID is for matching not security

- **WHEN** client sends HANDLE message with ULID
- **THEN** ULID is used only to match connection to pending request
- **AND** ULID does NOT provide authentication or authorization
- **AND** BasicAuth is the sole authentication mechanism

#### Scenario: Data connection authentication

- **WHEN** client establishes data connection with HANDLE:[ULID]
- **THEN** server verifies BasicAuth credentials independently
- **AND** server does not rely on ULID for security validation
- **AND** malicious actor cannot use intercepted ULID without valid credentials

### Requirement: Client Initial Connection Establishment

The client SHALL establish an initial WebSocket connection to the server using BasicAuth credentials.

#### Scenario: Successful initial connection

- **WHEN** client connects with valid ServiceName and ServiceToken via BasicAuth
- **THEN** WebSocket connection is established successfully
- **AND** client can send and receive messages

#### Scenario: Authentication failure

- **WHEN** client connects with invalid ServiceName or ServiceToken
- **THEN** WebSocket connection is rejected with 401 Unauthorized
- **AND** no connection is established

#### Scenario: WAITING message sent

- **WHEN** initial connection is established successfully
- **THEN** client sends text message "WAITING" as the first message
- **AND** message is sent as WebSocket text frame

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

### Requirement: Server Initial Connection Handling

The server SHALL accept initial WebSocket connections and verify BasicAuth credentials against service definitions.

#### Scenario: Successful authentication on any connection

- **WHEN** server receives connection with valid BasicAuth for a known service
- **THEN** connection is accepted and upgraded to WebSocket
- **AND** server waits for first text message to determine connection type
- **AND** authentication is required for both control and data connections

#### Scenario: Authentication with unknown service

- **WHEN** server receives connection with ServiceName not in service definitions
- **THEN** connection is rejected with 401 Unauthorized
- **AND** connection is not upgraded to WebSocket

#### Scenario: Authentication with invalid token

- **WHEN** server receives connection with valid ServiceName but wrong ServiceToken
- **THEN** connection is rejected with 401 Unauthorized
- **AND** connection is not upgraded to WebSocket

#### Scenario: WAITING message received as first message

- **WHEN** server reads first text message and it equals "WAITING"
- **THEN** server routes connection to control connection handler
- **AND** marks the connection as ready to handle requests
- **AND** connection remains open for future CONNECT messages

### Requirement: Server Connection Request Notification

The server SHALL notify clients of incoming connection requests by sending CONNECT messages with ULID identifiers.

#### Scenario: Generate ULID for new connection

- **WHEN** an incoming request needs to be routed to a client
- **THEN** server generates a new ULID as connection identifier
- **AND** ULID is unique across all active connections

#### Scenario: Send CONNECT message

- **WHEN** server assigns ULID to an incoming connection request
- **THEN** server sends text message "CONNECT:[ULID]" on the control connection
- **AND** message uses the ULID generated for this specific connection
- **AND** message is sent as WebSocket text frame

#### Scenario: Multiple concurrent connections

- **WHEN** multiple incoming requests arrive for same service
- **THEN** server generates unique ULID for each request
- **AND** sends separate CONNECT messages for each request

### Requirement: Client Data Connection Initiation

The client SHALL establish new WebSocket connections to handle specific connection requests identified by ULID.

#### Scenario: Receive CONNECT message

- **WHEN** client receives "CONNECT:[ULID]" message on control connection
- **THEN** client extracts ULID from the message
- **AND** client initiates new WebSocket connection to server

#### Scenario: Send HANDLE message with authentication

- **WHEN** client establishes new connection for handling a request
- **THEN** client MUST provide same BasicAuth credentials (ServiceName, ServiceToken)
- **AND** client sends text message "HANDLE:[ULID]" as first message on new connection
- **AND** message uses the ULID received in CONNECT message
- **AND** message is sent as WebSocket text frame

#### Scenario: HANDLE connection requires authentication

- **WHEN** client opens new connection for HANDLE
- **THEN** server verifies BasicAuth credentials before upgrade
- **AND** server rejects connection if credentials are invalid
- **AND** ULID alone is NOT sufficient for authentication

### Requirement: Server Data Connection Matching

The server SHALL match incoming HANDLE connections to pending connection requests by ULID.

#### Scenario: HANDLE received as first message

- **WHEN** server reads first text message and it starts with "HANDLE:"
- **THEN** server routes connection to data connection handler
- **AND** extracts ULID from the message
- **AND** security is already verified by BasicAuth, ULID is for matching only

#### Scenario: Successful HANDLE match

- **WHEN** server receives "HANDLE:[ULID]" as first message on authenticated connection
- **THEN** server looks up pending connection request with matching ULID
- **AND** verifies ULID belongs to a service matching the authenticated ServiceName
- **AND** associates new WebSocket with the pending request
- **AND** connection is ready for bidirectional data transfer

#### Scenario: HANDLE with unknown ULID

- **WHEN** server receives "HANDLE:[ULID]" with ULID not matching any pending request
- **THEN** server closes the WebSocket connection with error code
- **AND** sends close reason indicating unknown connection ID

#### Scenario: HANDLE timeout

- **WHEN** CONNECT message is sent but no HANDLE is received within timeout period
- **THEN** server closes the pending request
- **AND** removes ULID from pending requests map
- **AND** may send error response to original incoming connection

### Requirement: Bidirectional Binary Data Transfer

The system SHALL support bidirectional binary message exchange on established data connections.

#### Scenario: Client sends binary data

- **WHEN** data connection is established via HANDLE handshake
- **THEN** client can send binary WebSocket frames
- **AND** server receives binary data from client

#### Scenario: Server sends binary data

- **WHEN** data connection is established via HANDLE handshake
- **THEN** server can send binary WebSocket frames
- **AND** client receives binary data from server

#### Scenario: Data transfer after control messages

- **WHEN** HANDLE handshake completes successfully
- **THEN** all subsequent messages on that connection are binary frames
- **AND** no more text control messages are expected on data connection

#### Scenario: Concurrent data transfer

- **WHEN** multiple data connections are active for a service
- **THEN** each connection transfers data independently
- **AND** data on one connection does not affect others

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

### Requirement: Protocol Message Validation

The system SHALL validate all protocol messages for correctness and reject malformed messages.

#### Scenario: ULID character validation

- **WHEN** parsing CONNECT or HANDLE messages
- **THEN** system validates ULID contains only valid base32 characters (0-9, A-Z)
- **AND** rejects ULIDs with invalid characters

#### Scenario: Message length validation

- **WHEN** receiving text messages on WebSocket
- **THEN** system validates message length is within reasonable bounds
- **AND** rejects excessively long messages (e.g., > 1KB for control messages)

#### Scenario: Empty ULID validation

- **WHEN** parsing CONNECT or HANDLE messages
- **THEN** system rejects messages with empty connection IDs
- **AND** returns false from Parse\* functions

### Requirement: WebSocket Write Synchronization

The system SHALL protect WebSocket connections from concurrent write operations.

#### Scenario: Concurrent write protection on server

- **WHEN** multiple goroutines attempt to write to same WebSocket connection
- **THEN** system uses mutex to serialize writes
- **AND** prevents concurrent write panics from gorilla/websocket

#### Scenario: Concurrent write protection on client

- **WHEN** client writes control and data messages concurrently
- **THEN** system uses mutex to serialize writes
- **AND** ensures thread-safe WebSocket operations

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

### Requirement: Pending Connection Cleanup

The system SHALL properly clean up pending connections on shutdown or timeout.

#### Scenario: Server shutdown cleanup

- **WHEN** server shuts down with pending connections
- **THEN** all pending connections are closed
- **AND** pending connections map is cleared
- **AND** timeout goroutines are cancelled

#### Scenario: Control connection cleanup

- **WHEN** control connection disconnects
- **THEN** associated pending requests are cancelled
- **AND** control connection is removed from registry
- **AND** no goroutine leaks occur

### Requirement: Resource Exhaustion Protection

The system SHALL protect against resource exhaustion scenarios.

#### Scenario: ULID generation failure

- **WHEN** ULID generation fails (e.g., entropy source issue)
- **THEN** error is returned to caller
- **AND** connection request is rejected gracefully
- **AND** incoming connection is closed with error message

#### Scenario: Pending connection map overflow

- **WHEN** large number of pending connections accumulate
- **THEN** system enforces timeout-based cleanup
- **AND** old pending connections are removed
- **AND** memory usage remains bounded

### Requirement: Data connection read deadline behaviour

The system SHALL NOT impose a recurring per-iteration read deadline on data WebSocket connections based on the pong-wait interval, to prevent spurious timeouts on legitimately idle tunnels.

#### Scenario: Idle data connection not timed out by read deadline

- **WHEN** a data connection (`HANDLE`) is open but neither side is actively sending data
- **THEN** the connection SHALL remain open until an explicit error or close frame is received
- **AND** the connection SHALL NOT be closed solely because `PongWait` duration has elapsed between reads

#### Scenario: Data connection closed on genuine error

- **WHEN** the underlying TCP stream for a data connection is reset or closed by the network
- **THEN** the next read or write operation SHALL return an error
- **AND** both sides of the data tunnel SHALL be closed and cleaned up

#### Scenario: Data connection closed on normal WebSocket close

- **WHEN** either side sends a WebSocket close frame on the data connection
- **THEN** the other side SHALL receive the close frame and close its end of the tunnel
- **AND** no read deadline timeout SHALL be required to detect this closure
