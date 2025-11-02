# Implementation Tasks

## 1. Configuration Structures
- [x] 1.1 Define `ServerOptions` struct with all required fields
- [x] 1.2 Define `ClientOptions` struct with all required fields
- [x] 1.3 Add validation methods for both structs
- [x] 1.4 Add YAML struct tags to all fields

## 2. Configuration Loading
- [x] 2.1 Create configuration loading function that reads and parses YAML
- [x] 2.2 Add error handling for missing or malformed config files
- [x] 2.3 Implement default value handling for optional fields
- [x] 2.4 Write unit tests for configuration loading

## 3. Server CLI Command
- [x] 3.1 Create `cmd/wstun-server/` directory
- [x] 3.2 Implement main.go with flag parsing for `-c` option
- [x] 3.3 Load ServerOptions from configuration file
- [x] 3.4 Validate loaded configuration
- [x] 3.5 Add usage documentation and help text
- [x] 3.6 Handle graceful shutdown on signals

## 4. Client CLI Command
- [x] 4.1 Create `cmd/wstun-client/` directory
- [x] 4.2 Implement main.go with flag parsing for `-c` option
- [x] 4.3 Load ClientOptions from configuration file
- [x] 4.4 Validate loaded configuration
- [x] 4.5 Add usage documentation and help text
- [x] 4.6 Handle graceful shutdown on signals

## 5. Documentation and Testing
- [x] 5.1 Create example configuration files for both server and client
- [x] 5.2 Document configuration file format and all available fields
- [x] 5.3 Add integration tests for both CLI commands
- [x] 5.4 Update README with usage examples

