## ADDED Requirements

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
