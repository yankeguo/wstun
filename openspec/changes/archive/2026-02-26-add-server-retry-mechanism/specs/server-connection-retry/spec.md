## ADDED Requirements

### Requirement: Server connection request retry mechanism
The server SHALL implement a retry mechanism in `RequestConnection` that re-sends `CONNECT` messages when client fails to establish data connection within timeout.

#### Scenario: Retry on timeout
- **WHEN** client does not establish data connection within PendingTimeout
- **THEN** server SHALL retry sending `CONNECT:[ULID]` message
- **AND** the same ULID SHALL be used for all retry attempts
- **AND** server SHALL wait for the next retry attempt

#### Scenario: Retry exhausted
- **WHEN** all retry attempts are exhausted
- **THEN** server SHALL close the external TCP connection
- **AND** server SHALL return an error indicating connection failed

#### Scenario: Success before retry exhausted
- **WHEN** client establishes data connection before retry attempts are exhausted
- **THEN** server SHALL proceed with bidirectional data transfer
- **AND** no further retry attempts SHALL be made

### Requirement: Configurable retry attempts
The server SHALL support a configurable maximum number of retry attempts for connection requests.

#### Scenario: Retry enabled
- **WHEN** `RetryAttempts` is set to a positive integer N
- **THEN** server SHALL retry up to N times when connection times out

#### Scenario: Retry disabled (default)
- **WHEN** `RetryAttempts` is 0 or not configured
- **THEN** server SHALL maintain existing behavior (no retry, fail immediately on timeout)

### Requirement: Configurable retry interval with exponential backoff
The server SHALL implement exponential backoff for retry intervals.

#### Scenario: Exponential backoff calculation
- **WHEN** retry attempt number is N (0-indexed)
- **THEN** the retry delay SHALL be calculated as: `RetryInterval * (2^N)`
- **AND** if the calculated delay exceeds `RetryMaxInterval`, the delay SHALL be capped at `RetryMaxInterval`

#### Scenario: First retry
- **WHEN** first retry is triggered (attempt 0)
- **THEN** server SHALL wait for `RetryInterval` duration before retrying

#### Scenario: Subsequent retries
- **WHEN** second retry is triggered (attempt 1)
- **THEN** server SHALL wait for `RetryInterval * 2` duration before retrying

#### Scenario: Maximum interval cap
- **WHEN** calculated retry delay would exceed `RetryMaxInterval`
- **THEN** server SHALL use `RetryMaxInterval` as the delay
