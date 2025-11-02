## MODIFIED Requirements

### Requirement: Server Configuration Structure
The system SHALL define a `ServerOptions` struct that represents server configuration for managing tunneled services.

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

#### Scenario: ServerOptions validation fails for invalid data
- **WHEN** a ServerOptions instance has invalid or missing required fields
- **THEN** validation fails with descriptive error messages
- **AND** empty `services_dir` causes validation error

### Requirement: Client Configuration Structure
The system SHALL define a `ClientOptions` struct that represents client configuration for exposing local services through tunnels.

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

#### Scenario: ClientOptions validation fails for invalid data
- **WHEN** a ClientOptions instance has invalid or missing required fields
- **THEN** validation fails with descriptive error messages
- **AND** missing `service_name`, `service_token`, or `target_addr` causes validation errors

#### Scenario: Service name format validation
- **WHEN** a ClientOptions has `service_name` with invalid characters
- **THEN** validation fails with format error
- **AND** only alphanumeric characters, hyphens, and underscores are allowed

## ADDED Requirements

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

