## 1. Configuration and Data Model

- [x] 1.1 Add retry configuration fields to server config struct (`RetryAttempts`, `RetryInterval`, `RetryMaxInterval`)
- [x] 1.2 Add default values for retry configuration (RetryAttempts=0, RetryInterval=5s, RetryMaxInterval=60s)
- [x] 1.3 Update config validation to ensure non-negative retry values

## 2. Configuration Loading

- [x] 2.1 Ensure configuration file can parse retry settings (retry_attempts, retry_interval, retry_max_interval)

## 3. Server Protocol Retry Implementation

- [x] 3.1 Modify `RequestConnection` method signature to accept retry configuration
- [x] 3.2 Implement retry loop with exponential backoff in `RequestConnection`
- [x] 3.3 Ensure same ULID is used across all retry attempts
- [x] 3.4 Reset pending connection timeout on each retry attempt
- [x] 3.5 Stop retrying and proceed when data connection is established
- [x] 3.6 Close external TCP connection and return error when all retries exhausted
- [x] 3.7 Add logging for retry attempts (info level: "Retrying connection X/N after Ys")

## 4. Testing

- [x] 4.1 Write unit tests for exponential backoff calculation function
- [x] 4.2 Write unit tests for retry loop logic with mock pending connection
- [x] 4.3 Write integration test: retry succeeds on second attempt
- [x] 4.4 Write integration test: all retries exhausted, connection fails
- [x] 4.5 Write integration test: retry disabled (RetryAttempts=0) maintains existing behavior
- [x] 4.6 Test configuration file parsing with all retry settings

## 5. Documentation

- [x] 5.1 Update README.md with new server configuration options
- [x] 5.2 Document retry behavior and exponential backoff algorithm
- [x] 5.3 Add example configuration file with retry settings
