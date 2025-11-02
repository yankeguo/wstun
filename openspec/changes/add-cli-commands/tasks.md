# Implementation Tasks

## 1. Configuration Structures
- [ ] 1.1 Define `ServerOptions` struct with all required fields
- [ ] 1.2 Define `ClientOptions` struct with all required fields
- [ ] 1.3 Add validation methods for both structs
- [ ] 1.4 Add YAML struct tags to all fields

## 2. Configuration Loading
- [ ] 2.1 Create configuration loading function that reads and parses YAML
- [ ] 2.2 Add error handling for missing or malformed config files
- [ ] 2.3 Implement default value handling for optional fields
- [ ] 2.4 Write unit tests for configuration loading

## 3. Server CLI Command
- [ ] 3.1 Create `cmd/wstun-server/` directory
- [ ] 3.2 Implement main.go with flag parsing for `-c` option
- [ ] 3.3 Load ServerOptions from configuration file
- [ ] 3.4 Validate loaded configuration
- [ ] 3.5 Add usage documentation and help text
- [ ] 3.6 Handle graceful shutdown on signals

## 4. Client CLI Command
- [ ] 4.1 Create `cmd/wstun-client/` directory
- [ ] 4.2 Implement main.go with flag parsing for `-c` option
- [ ] 4.3 Load ClientOptions from configuration file
- [ ] 4.4 Validate loaded configuration
- [ ] 4.5 Add usage documentation and help text
- [ ] 4.6 Handle graceful shutdown on signals

## 5. Documentation and Testing
- [ ] 5.1 Create example configuration files for both server and client
- [ ] 5.2 Document configuration file format and all available fields
- [ ] 5.3 Add integration tests for both CLI commands
- [ ] 5.4 Update README with usage examples

