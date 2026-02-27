## ADDED Requirements

### Requirement: Server-side keepalive pings on control connections

The server SHALL send periodic WebSocket ping frames on every active control connection to prevent proxy idle-timeout from silently resetting the connection.

#### Scenario: Ping sent at configured interval

- **WHEN** a control connection is active
- **THEN** server SHALL send a `websocket.PingMessage` frame at least once every `ControlPingInterval` duration
- **AND** `ControlPingInterval` SHALL default to 15 seconds

#### Scenario: Ping goroutine exits on control connection close

- **WHEN** a control connection is closed or errors
- **THEN** the associated ping goroutine SHALL exit without leaking
- **AND** no further ping attempts SHALL be made on that connection

#### Scenario: Ping write uses mutex

- **WHEN** server sends a ping frame on a control connection
- **THEN** the write SHALL be serialised using the same mutex used by `sendToControlConnections`
- **AND** no concurrent write panic SHALL occur

#### Scenario: Configurable ping interval

- **WHEN** `ControlPingInterval` is set to a positive duration
- **THEN** server SHALL use that value as the ping interval
- **WHEN** `ControlPingInterval` is zero or negative
- **THEN** server SHALL use the default value of 15 seconds

### Requirement: Write deadline on all server-side WebSocket writes

The server SHALL set an explicit write deadline before every WebSocket write to prevent goroutines from blocking indefinitely on dead connections.

#### Scenario: Write deadline on CONNECT message

- **WHEN** server sends a CONNECT message via `sendToControlConnections`
- **THEN** server SHALL set a write deadline of `WriteWait` on the connection before writing
- **AND** `WriteWait` SHALL default to 10 seconds

#### Scenario: Write deadline on ping frame

- **WHEN** server sends a ping frame on a control connection
- **THEN** server SHALL set a write deadline of `WriteWait` before the write

#### Scenario: Configurable write wait

- **WHEN** `WriteWait` is set to a positive duration on `ServerProtocol`
- **THEN** server SHALL use that value for all write deadlines

### Requirement: Dead control connection pruning on write failure

The server SHALL immediately remove a control connection from the registry when a write to that connection fails.

#### Scenario: Stale connection pruned after failed CONNECT send

- **WHEN** `sendToControlConnections` fails to write to a control connection
- **THEN** server SHALL delete that control connection from the registry
- **AND** subsequent calls SHALL NOT attempt to write to the deleted connection

#### Scenario: Concurrent prune is safe

- **WHEN** both the write-failure path and the read-loop close path attempt to remove the same control connection concurrently
- **THEN** no panic or data race SHALL occur
- **AND** the connection SHALL be removed exactly from the registry (idempotent delete)

### Requirement: Fast-fail RequestConnection when no control connection is available

The server SHALL not sleep through retry intervals when no control connection is registered for the target service.

#### Scenario: Empty registry causes immediate attempt failure

- **WHEN** `RequestConnection` is called for a service with no registered control connections
- **THEN** that attempt SHALL be counted as exhausted immediately without sleeping
- **AND** if `RetryAttempts` is 0 the incoming connection SHALL be closed and an error returned immediately

#### Scenario: Non-empty registry proceeds normally

- **WHEN** at least one control connection is registered for the service
- **THEN** `RequestConnection` SHALL proceed with the existing retry/timeout logic unchanged
