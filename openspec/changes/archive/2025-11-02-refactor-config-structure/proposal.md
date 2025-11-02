## Why
The current configuration structure assumes the client listens locally and the server proxies to it, but the intended architecture is reversed: clients expose local services to the server via WebSocket tunnels, and the server makes those services available on specified ports. The configuration structures need to reflect this service-oriented tunnel architecture with proper authentication.

## What Changes
- **ServerOptions**: Add `services_dir` field to specify directory containing service definitions
- **Service Definition Files**: New YAML format for per-service configuration (service_token, listen_addr), where filename = service_name
- **ClientOptions**: Add `service_name` and `service_token` fields for service identification and authentication
- **ClientOptions**: Replace `local_addr` with `target_addr` to represent the local service being tunneled
- Update validation logic for all configuration changes
- Update example YAML files to reflect new structure

## Impact
- Affected specs: `configuration`
- Affected code: 
  - `config.go` (ServerOptions, ClientOptions structs and validation)
  - `examples/server.yaml`
  - `examples/client.yaml`
  - `config_test.go` (configuration tests)
- **BREAKING**: Existing configuration files will need migration to new field names and structure

