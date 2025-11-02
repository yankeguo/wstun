## Context
The wstun architecture follows a "reverse tunnel" pattern where:
1. Client connects to server via WebSocket
2. Client exposes a local service (target_addr) through the tunnel
3. Server authenticates the client using service credentials
4. Server listens on a configured port and forwards traffic through the tunnel to the client's local service

This requires service-oriented configuration where the server manages multiple service definitions, each with its own authentication token and listening address.

## Goals / Non-Goals

### Goals
- Enable server to manage multiple service definitions via filesystem
- Provide secure per-service authentication using tokens
- Clearly distinguish between "target" (client-side local service) and "listen" (server-side exposed port)
- Support dynamic service configuration without code changes

### Non-Goals
- Implement actual tunnel functionality (configuration only)
- Add service discovery or registration API
- Support hot-reloading of service definitions (future enhancement)

## Decisions

### Service Definition Storage
**Decision**: Use individual YAML files in `services_dir`, where filename = service_name

**Rationale**:
- Simple filesystem-based service management
- Easy to add/remove services (just add/delete files)
- Clear service identity (filename)
- Follows standard configuration pattern

**Format**:
```yaml
# services_dir/my-service.yaml
service_token: "secret-token-here"
listen_addr: ":9090"
```

**Alternatives considered**:
- Single file with all services: harder to manage, merge conflicts
- Database storage: unnecessary complexity for this use case

### Client Configuration Changes
**Decision**: 
- `local_addr` → `target_addr` (semantic clarity)
- Add `service_name` for service identification
- Add `service_token` for authentication

**Rationale**:
- `target_addr` better expresses "the local service I want to tunnel"
- `local_addr` was misleading (sounded like "where client listens")
- Service name and token enable server-side authentication

### Validation Rules
- Server: `services_dir` must be a valid directory path (existence checked at runtime)
- Service files: Must contain `service_token` (non-empty) and `listen_addr` (valid address)
- Client: All of `service_name`, `service_token`, `target_addr` are required
- Service token should be non-empty string (actual token format validation is future work)

## Risks / Trade-offs

### Risk: Service Token Security
- Tokens stored in plaintext in YAML files
- **Mitigation**: Document filesystem permission requirements (0600), consider encryption in future

### Risk: Breaking Change
- All existing configurations break
- **Mitigation**: Provide clear migration guide, bump version if applicable

### Trade-off: Filesystem vs API
- Using filesystem means no remote service management
- **Acceptable**: Matches simplicity goals, can add API layer later

## Migration Plan

### For Server Configurations
1. Create `services_dir` directory
2. For each service, create `<service-name>.yaml` with:
   - `service_token`: generate unique token
   - `listen_addr`: port where service should be exposed

### For Client Configurations
1. Add `service_name`: name matching server-side service file
2. Add `service_token`: matching token from server
3. Rename `local_addr` → `target_addr`

### Example Migration
**Before (client.yaml)**:
```yaml
server_url: "ws://localhost:8080"
local_addr: ":9090"
```

**After (client.yaml)**:
```yaml
server_url: "ws://localhost:8080"
service_name: "my-service"
service_token: "secret-token"
target_addr: "127.0.0.1:3000"
```

**Server-side (services_dir/my-service.yaml)**:
```yaml
service_token: "secret-token"
listen_addr: ":9090"
```

## Open Questions
- Should service_token support multiple formats (JWT, plain, etc.)? → Start with plain strings
- Should we validate service_name format (e.g., DNS-compatible)? → Yes, add basic validation
- Should services_dir be optional with fallback? → No, required for new architecture

