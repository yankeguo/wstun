# Implement Server and Client CLI with Protocol

## Why
The WebSocket protocol implementation is complete, but the server and client CLI binaries only have skeletal implementations with TODOs. The CLIs need to be wired up to use the protocol implementation to provide functional tunneling capabilities.

## What Changes
- Implement server CLI to load services, start HTTP/HTTPS server, and handle tunneling using ServerProtocol
- Implement client CLI to connect to server and tunnel connections using ClientProtocol
- Add connection lifecycle management including reconnection with exponential backoff for client
- Add proper graceful shutdown handling for both server and client
- Add comprehensive logging based on configured log levels
- Add connection counting and max connection limits for server
- Add bidirectional data transfer handling between TCP listeners and WebSocket connections

## Impact
- Affected specs: `cli-server`, `cli-client` (completing their requirements)
- Affected code: 
  - `cmd/wstun-server/main.go` - replace TODOs with full server implementation
  - `cmd/wstun-client/main.go` - replace TODOs with full client implementation
- No changes to protocol or configuration code needed (already complete)

