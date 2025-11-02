# cli-client Specification

## Purpose
TBD - created by archiving change add-cli-commands. Update Purpose after archive.
## Requirements
### Requirement: Client Command Binary
The system SHALL provide a `wstun-client` command-line executable that runs a WebSocket tunnel client.

#### Scenario: Client binary exists
- **WHEN** the project is built
- **THEN** a `wstun-client` binary is produced in `cmd/wstun-client/`

#### Scenario: Client displays help text
- **WHEN** `wstun-client -h` or `wstun-client --help` is executed
- **THEN** usage information and available flags are displayed

### Requirement: Configuration File Flag
The `wstun-client` command SHALL accept a `-c` flag that specifies the path to a configuration file.

#### Scenario: Valid configuration file provided
- **WHEN** `wstun-client -c /path/to/config.yaml` is executed with a valid config file
- **THEN** the client loads configuration from the specified file
- **AND** the client starts successfully

#### Scenario: Configuration file not found
- **WHEN** `wstun-client -c /nonexistent/config.yaml` is executed
- **THEN** an error message is displayed indicating the file cannot be found
- **AND** the client exits with a non-zero status code

#### Scenario: No configuration file specified
- **WHEN** `wstun-client` is executed without the `-c` flag
- **THEN** an error message is displayed indicating configuration is required
- **AND** the client exits with a non-zero status code

### Requirement: Graceful Shutdown
The `wstun-client` command SHALL handle system signals for graceful shutdown.

#### Scenario: SIGINT received
- **WHEN** the client is running and receives SIGINT (Ctrl+C)
- **THEN** the client initiates graceful shutdown
- **AND** existing connections are closed cleanly
- **AND** the client exits with status code 0

#### Scenario: SIGTERM received
- **WHEN** the client is running and receives SIGTERM
- **THEN** the client initiates graceful shutdown
- **AND** existing connections are closed cleanly
- **AND** the client exits with status code 0

