## MODIFIED Requirements
### Requirement: Server Configuration Structure
The system SHALL define a `ServerOptions` struct that represents server configuration for managing tunneled services with comprehensive validation.

#### Scenario: ServerOptions contains required fields
- **WHEN** examining the ServerOptions struct
- **THEN** it includes a `services_dir` field for service definitions directory
- **AND** it includes fields for security settings (TLS)
- **AND** it includes fields for logging configuration
- **AND** it includes fields for connection limits

#### Scenario: ServerOptions validates successfully
- **WHEN** a ServerOptions instance has all required fields populated with valid values
- **THEN** validation succeeds and returns no errors
- **AND** `services_dir` is a non-empty string
- **AND** `listen_addr` is a valid address format

#### Scenario: ServerOptions validation fails for invalid data
- **WHEN** a ServerOptions instance has invalid or missing required fields
- **THEN** validation fails with descriptive error messages
- **AND** empty `services_dir` causes validation error
- **AND** invalid `listen_addr` format causes validation error

#### Scenario: Listen address format validation
- **WHEN** ServerOptions has invalid listen address format
- **THEN** validation fails with clear error message
- **AND** provides examples of valid formats

### Requirement: Client Configuration Structure
The system SHALL define a `ClientOptions` struct that represents client configuration for exposing local services through tunnels with enhanced validation.

#### Scenario: ClientOptions contains required fields
- **WHEN** examining the ClientOptions struct
- **THEN** it includes a `service_name` field identifying the service
- **AND** it includes a `service_token` field for authentication
- **AND** it includes a `target_addr` field for the local service address to tunnel
- **AND** it includes a `server_url` field for WebSocket server connection
- **AND** it includes fields for security settings
- **AND** it includes fields for reconnection behavior
- **AND** it includes fields for logging configuration

#### Scenario: ClientOptions validates successfully
- **WHEN** a ClientOptions instance has all required fields populated with valid values
- **THEN** validation succeeds and returns no errors
- **AND** `service_name`, `service_token`, and `target_addr` are all non-empty
- **AND** `server_url` has valid WebSocket scheme

#### Scenario: ClientOptions validation fails for invalid data
- **WHEN** a ClientOptions instance has invalid or missing required fields
- **THEN** validation fails with descriptive error messages
- **AND** missing `service_name`, `service_token`, or `target_addr` causes validation errors
- **AND** invalid `server_url` scheme causes validation error

#### Scenario: Service name format validation
- **WHEN** a ClientOptions has `service_name` with invalid characters
- **THEN** validation fails with format error
- **AND** only alphanumeric characters, hyphens, and underscores are allowed
- **AND** service name cannot start or end with hyphen

#### Scenario: Server URL scheme validation
- **WHEN** ClientOptions has server_url with invalid scheme
- **THEN** validation fails with error
- **AND** only "ws://" or "wss://" schemes are allowed
- **AND** error message suggests correct format

#### Scenario: Service name length validation
- **WHEN** ClientOptions has service_name exceeding maximum length
- **THEN** validation fails with error
- **AND** maximum length is 255 characters
- **AND** error message includes current length

## ADDED Requirements
### Requirement: Path Safety in Configuration
The system SHALL safely handle file paths and prevent directory traversal attacks.

#### Scenario: Safe path construction
- **WHEN** loading service definition files
- **THEN** system uses filepath.Join for path construction
- **AND** avoids string concatenation for paths

#### Scenario: Path traversal protection
- **WHEN** loading configuration files
- **THEN** system validates paths do not contain directory traversal patterns
- **AND** rejects suspicious patterns like "../"

#### Scenario: Symlink handling
- **WHEN** services directory contains symlinks
- **THEN** system follows symlinks but validates final target
- **AND** symlink targets must be within expected directory

### Requirement: Configuration File Error Handling
The system SHALL provide clear error messages for configuration file issues.

#### Scenario: Empty configuration file
- **WHEN** configuration file exists but is empty
- **THEN** parsing fails with clear error
- **AND** error message indicates file is empty

#### Scenario: Partial configuration file
- **WHEN** configuration file is incomplete or truncated
- **THEN** parsing fails with clear error
- **AND** error indicates which fields are missing

#### Scenario: UTF-8 validation
- **WHEN** configuration file contains invalid UTF-8
- **THEN** parsing fails with encoding error
- **AND** error message indicates file encoding issue

### Requirement: Service Directory Edge Cases
The system SHALL handle edge cases in service directory loading.

#### Scenario: Services directory with only invalid files
- **WHEN** LoadAllServices is called on directory with only invalid YAML files
- **THEN** function returns empty map with aggregated errors
- **AND** error message lists all files that failed to load

#### Scenario: Mixed valid and invalid files
- **WHEN** services directory contains both valid and invalid files
- **THEN** valid files are loaded successfully
- **AND** invalid files are reported in error
- **AND** operation continues despite partial failures

#### Scenario: Service name collision
- **WHEN** service filename conflicts after extension removal
- **THEN** later file overwrites earlier one
- **AND** warning is logged about collision

