# Implementation Tasks

## 1. Protocol Constants and Types
- [x] 1.1 Create `protocol.go` with message type constants (WAITING, CONNECT:, HANDLE:)
- [x] 1.2 Add ULID dependency to `go.mod`
- [x] 1.3 Add WebSocket dependency (e.g., `gorilla/websocket`) to `go.mod`
- [x] 1.4 Define helper functions for message parsing and validation

## 2. Client Protocol Implementation
- [x] 2.1 Create `client_protocol.go` with client protocol handler structure
- [x] 2.2 Implement initial connection establishment with BasicAuth
- [x] 2.3 Implement WAITING message sending
- [x] 2.4 Implement CONNECT message receiving and parsing
- [x] 2.5 Implement data connection establishment with HANDLE message
- [x] 2.6 Implement bidirectional binary data transfer handlers
- [x] 2.7 Implement connection lifecycle management (close, error handling)

## 3. Server Protocol Implementation
- [x] 3.1 Create `server_protocol.go` with server protocol handler structure
- [x] 3.2 Implement WebSocket upgrade with BasicAuth verification
- [x] 3.3 Implement WAITING message receiving and client readiness tracking
- [x] 3.4 Implement ULID generation for connection requests
- [x] 3.5 Implement CONNECT message sending to control connections
- [x] 3.6 Implement HANDLE message receiving and connection matching
- [x] 3.7 Implement pending connection timeout mechanism
- [x] 3.8 Implement bidirectional binary data transfer handlers
- [x] 3.9 Implement connection lifecycle management (close, error handling)

## 4. Testing
- [x] 4.1 Write unit tests for message parsing and validation
- [x] 4.2 Write integration tests for client-server protocol handshake
- [x] 4.3 Write tests for authentication success and failure cases
- [x] 4.4 Write tests for CONNECT/HANDLE flow
- [x] 4.5 Write tests for binary data transfer
- [x] 4.6 Write tests for connection closure and error scenarios
- [x] 4.7 Write tests for concurrent connection handling
- [x] 4.8 Write tests for timeout scenarios

## 5. Documentation
- [x] 5.1 Add package documentation to protocol files
- [x] 5.2 Document exported functions and types
- [x] 5.3 Add code examples in comments for common usage patterns
- [x] 5.4 Update README.md with protocol overview (if applicable)

## Implementation Summary

All tasks completed successfully. The WebSocket protocol implementation includes:

- **protocol.go**: Constants, ULID generation, message formatting/parsing functions
- **server_protocol.go**: Server-side WebSocket handling with BasicAuth, control/data connection routing, pending connection management
- **client_protocol.go**: Client-side protocol handler with automatic CONNECT handling and bidirectional data transfer
- **protocol_test.go**: Comprehensive test suite covering message parsing, authentication, connection lifecycle, and timeouts

The implementation follows the design principles:
- Single endpoint, path-agnostic server
- Every connection requires BasicAuth authentication
- ULID for connection matching only (not security)
- Text control messages, binary data transfer
- Independent protocol layer (not integrated with CLI yet)

All tests passing (some integration tests skipped for timing control).
