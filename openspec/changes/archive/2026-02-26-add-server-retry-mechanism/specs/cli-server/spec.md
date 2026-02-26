## REMOVED Requirements

### Requirement: Retry configuration flags
**Reason**: Retry configuration should be managed through configuration file only for consistency and simplicity.
**Migration**: Use the configuration file to set `retry_attempts`, `retry_interval`, and `retry_max_interval` options.

## ADDED Requirements

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
