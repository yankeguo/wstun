## ADDED Requirements

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

### Requirement: Server Initial Connection Handling
The server SHALL accept initial WebSocket connections and verify BasicAuth credentials against service definitions.

#### Scenario: Successful authentication
- **WHEN** server receives connection with valid BasicAuth for a known service
- **THEN** connection is accepted and upgraded to WebSocket
- **AND** server stores the connection as available for the service

#### Scenario: Authentication with unknown service
- **WHEN** server receives connection with ServiceName not in service definitions
- **THEN** connection is rejected with 401 Unauthorized

#### Scenario: Authentication with invalid token
- **WHEN** server receives connection with valid ServiceName but wrong ServiceToken
- **THEN** connection is rejected with 401 Unauthorized

#### Scenario: WAITING message received
- **WHEN** server receives "WAITING" text message on initial connection
- **THEN** server marks the connection as ready to handle requests
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

#### Scenario: Send HANDLE message
- **WHEN** client establishes new connection for handling a request
- **THEN** client sends text message "HANDLE:[ULID]" as first message on new connection
- **AND** message uses the ULID received in CONNECT message
- **AND** message is sent as WebSocket text frame
- **AND** uses same BasicAuth credentials as control connection

#### Scenario: HANDLE authentication
- **WHEN** client opens new connection for HANDLE
- **THEN** client uses same ServiceName and ServiceToken via BasicAuth
- **AND** server verifies credentials before accepting HANDLE

### Requirement: Server Data Connection Matching
The server SHALL match incoming HANDLE connections to pending connection requests by ULID.

#### Scenario: Successful HANDLE match
- **WHEN** server receives "HANDLE:[ULID]" on new connection
- **THEN** server looks up pending connection request with matching ULID
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
- **THEN** error is logged with connection context
- **AND** connection is closed gracefully if possible
- **AND** resources are cleaned up

