## 1. Update Configuration Structures
- [x] 1.1 Add `ServicesDir` field to `ServerOptions` struct with yaml tag `services_dir`
- [x] 1.2 Add `ServiceName` field to `ClientOptions` with yaml tag `service_name`
- [x] 1.3 Add `ServiceToken` field to `ClientOptions` with yaml tag `service_token`
- [x] 1.4 Rename `LocalAddr` to `TargetAddr` in `ClientOptions` (yaml tag `target_addr`)
- [x] 1.5 Create new `ServiceDefinition` struct with `ServiceToken` and `ListenAddr` fields

## 2. Update Validation Logic
- [x] 2.1 Add `services_dir` validation to `ServerOptions.Validate()` (required field)
- [x] 2.2 Add `service_name` validation to `ClientOptions.Validate()` (required, non-empty)
- [x] 2.3 Add `service_token` validation to `ClientOptions.Validate()` (required, non-empty)
- [x] 2.4 Update `target_addr` validation in `ClientOptions.Validate()` (required, valid address format)
- [x] 2.5 Add validation for `ServiceDefinition` (service_token and listen_addr required)
- [x] 2.6 Add service name format validation (alphanumeric, hyphens, underscores only)

## 3. Add Service Loading Functions
- [x] 3.1 Implement `LoadServiceDefinition(path string) (*ServiceDefinition, error)` to load individual service YAML
- [x] 3.2 Implement `LoadAllServices(servicesDir string) (map[string]*ServiceDefinition, error)` to load all services from directory
- [x] 3.3 Add error handling for missing/invalid service files

## 4. Update Example Configuration Files
- [x] 4.1 Update `examples/server.yaml` to include `services_dir` field
- [x] 4.2 Update `examples/client.yaml` with `service_name`, `service_token`, and `target_addr` (remove `local_addr`)
- [x] 4.3 Create `examples/services/example-service.yaml` as service definition template
- [x] 4.4 Add comments explaining the new architecture

## 5. Update Tests
- [x] 5.1 Update existing `config_test.go` tests for new field names
- [x] 5.2 Add tests for `ServiceDefinition` loading and validation
- [x] 5.3 Add tests for `LoadAllServices` with valid and invalid directories
- [x] 5.4 Add tests for service name format validation
- [x] 5.5 Test error cases (missing fields, invalid formats)

## 6. Update Documentation
- [x] 6.1 Add migration notes in example files
- [ ] 6.2 Update README.md with new configuration structure
- [ ] 6.3 Document service definition file format

