# cli-server Specification

## Purpose
TBD - created by archiving change add-cli-commands. Update Purpose after archive.
## Requirements
### Requirement: Server Command Binary
The system SHALL provide a `wstun-server` command-line executable that runs a WebSocket tunnel server.

#### Scenario: Server binary exists
- **WHEN** the project is built
- **THEN** a `wstun-server` binary is produced in `cmd/wstun-server/`

#### Scenario: Server displays help text
- **WHEN** `wstun-server -h` or `wstun-server --help` is executed
- **THEN** usage information and available flags are displayed

### Requirement: Configuration File Flag
The `wstun-server` command SHALL accept a `-c` flag that specifies the path to a configuration file.

#### Scenario: Valid configuration file provided
- **WHEN** `wstun-server -c /path/to/config.yaml` is executed with a valid config file
- **THEN** the server loads configuration from the specified file
- **AND** the server starts successfully

#### Scenario: Configuration file not found
- **WHEN** `wstun-server -c /nonexistent/config.yaml` is executed
- **THEN** an error message is displayed indicating the file cannot be found
- **AND** the server exits with a non-zero status code

#### Scenario: No configuration file specified
- **WHEN** `wstun-server` is executed without the `-c` flag
- **THEN** an error message is displayed indicating configuration is required
- **AND** the server exits with a non-zero status code

### Requirement: Graceful Shutdown
The `wstun-server` command SHALL handle system signals for graceful shutdown.

#### Scenario: SIGINT received
- **WHEN** the server is running and receives SIGINT (Ctrl+C)
- **THEN** the server initiates graceful shutdown
- **AND** existing connections are closed cleanly
- **AND** the server exits with status code 0

#### Scenario: SIGTERM received
- **WHEN** the server is running and receives SIGTERM
- **THEN** the server initiates graceful shutdown
- **AND** existing connections are closed cleanly
- **AND** the server exits with status code 0

