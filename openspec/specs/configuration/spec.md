# configuration Specification

## Purpose
TBD - created by archiving change add-cli-commands. Update Purpose after archive.
## Requirements
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

### Requirement: YAML Configuration File Format
The system SHALL support loading configuration from YAML files.

#### Scenario: Valid YAML file parsed
- **WHEN** a valid YAML configuration file is provided
- **THEN** the file is successfully parsed into the appropriate Options struct
- **AND** all fields are populated with values from the YAML

#### Scenario: Invalid YAML syntax
- **WHEN** a YAML file with syntax errors is provided
- **THEN** parsing fails with an error describing the syntax issue
- **AND** the error includes the line number where the problem occurs

#### Scenario: YAML file with missing required fields
- **WHEN** a YAML file is missing required configuration fields
- **THEN** validation fails after parsing
- **AND** an error lists all missing required fields

#### Scenario: YAML file with invalid field types
- **WHEN** a YAML file contains fields with incorrect data types
- **THEN** parsing or validation fails with type mismatch errors

### Requirement: Default Configuration Values
The system SHALL provide sensible default values for optional configuration fields.

#### Scenario: Optional fields use defaults
- **WHEN** a configuration file omits optional fields
- **THEN** those fields are populated with safe default values
- **AND** the configuration is still valid

#### Scenario: Explicit values override defaults
- **WHEN** a configuration file provides explicit values for optional fields
- **THEN** the explicit values are used instead of defaults

### Requirement: Service Definition Structure
The system SHALL define a `ServiceDefinition` struct that represents per-service configuration on the server.

#### Scenario: ServiceDefinition contains required fields
- **WHEN** examining the ServiceDefinition struct
- **THEN** it includes a `service_token` field for client authentication
- **AND** it includes a `listen_addr` field for the server-side listening address

#### Scenario: ServiceDefinition validates successfully
- **WHEN** a ServiceDefinition instance has valid `service_token` and `listen_addr`
- **THEN** validation succeeds and returns no errors

#### Scenario: ServiceDefinition validation fails
- **WHEN** a ServiceDefinition instance has empty `service_token` or `listen_addr`
- **THEN** validation fails with descriptive error messages

### Requirement: Service Definition File Loading
The system SHALL support loading individual service definitions from YAML files.

#### Scenario: Load valid service definition file
- **WHEN** `LoadServiceDefinition` is called with path to valid service YAML
- **THEN** the file is successfully parsed into ServiceDefinition struct
- **AND** all fields are populated with values from the YAML

#### Scenario: Load invalid service definition file
- **WHEN** `LoadServiceDefinition` is called with path to invalid YAML or missing file
- **THEN** loading fails with descriptive error message

### Requirement: Bulk Service Loading
The system SHALL support loading all service definitions from a directory.

#### Scenario: Load all services from directory
- **WHEN** `LoadAllServices` is called with valid `services_dir` path
- **THEN** all `.yaml` files in the directory are loaded
- **AND** a map of service_name (from filename) to ServiceDefinition is returned
- **AND** service_name is derived from filename without `.yaml` extension

#### Scenario: Load services from empty directory
- **WHEN** `LoadAllServices` is called with path to empty directory
- **THEN** an empty map is returned without error

#### Scenario: Load services with invalid files
- **WHEN** `LoadAllServices` is called and directory contains invalid YAML files
- **THEN** loading continues for valid files
- **AND** errors are collected and reported for invalid files

#### Scenario: Load services from non-existent directory
- **WHEN** `LoadAllServices` is called with non-existent directory path
- **THEN** loading fails with directory not found error

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

