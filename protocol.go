// Package wstun provides WebSocket tunneling protocol implementation.
//
// The protocol enables secure tunneling of TCP connections over WebSocket,
// with the following key features:
//
//   - Single WebSocket endpoint, path-agnostic server design
//   - BasicAuth authentication for all connections (control and data)
//   - ULID-based connection matching (not for security)
//   - Text-based control messages, binary data transfer
//   - Two-connection model: control connection for orchestration, data connections for actual traffic
//
// Protocol Flow:
//
// 1. Client establishes control connection with BasicAuth
// 2. Client sends "WAITING" message to indicate readiness
// 3. When server receives an incoming connection request, it generates a ULID and sends "CONNECT:[ULID]"
// 4. Client opens a new data connection with BasicAuth
// 5. Client sends "HANDLE:[ULID]" to claim the connection
// 6. Server matches the ULID and enables bidirectional binary data transfer
//
// Example Server Usage:
//
//	authFunc := func(serviceName, serviceToken string) bool {
//	    // Verify credentials against service definitions
//	    return checkCredentials(serviceName, serviceToken)
//	}
//
//	sp := NewServerProtocol(authFunc)
//	sp.DataHandler = func(incoming io.ReadWriteCloser, outgoing *websocket.Conn) {
//	    // Handle bidirectional data transfer
//	    // incoming: original TCP connection
//	    // outgoing: WebSocket to client
//	}
//
//	http.HandleFunc("/", sp.HandleWebSocket)
//	http.ListenAndServe(":8080", nil)
//
//	// When connection arrives:
//	conn, _ := listener.Accept()
//	sp.RequestConnection("service-name", conn)
//
// Example Client Usage:
//
//	cp := NewClientProtocol("ws://server:8080", "service-name", "service-token")
//	cp.ConnectHandler = func(ctx context.Context, connID string) (io.ReadWriteCloser, error) {
//	    // Connect to local target service
//	    return net.Dial("tcp", "localhost:8080")
//	}
//
//	if err := cp.Start(); err != nil {
//	    log.Fatal(err)
//	}
package wstun

import (
	"crypto/rand"
	"fmt"
	"strings"

	"github.com/oklog/ulid/v2"
)

// Protocol message type constants for WebSocket communication.
const (
	// MessageTypeWaiting is sent by client on control connection to indicate readiness.
	MessageTypeWaiting = "WAITING"

	// MessageTypePrefixConnect is the prefix for server messages notifying client of new connection.
	// Format: "CONNECT:[ULID]"
	MessageTypePrefixConnect = "CONNECT:"

	// MessageTypePrefixHandle is the prefix for client messages claiming a connection.
	// Format: "HANDLE:[ULID]"
	MessageTypePrefixHandle = "HANDLE:"
)

// ULID length in string representation.
const ulidLength = 26

// GenerateConnectionID generates a new ULID for connection identification.
// ULID is used for connection matching only, NOT for security.
func GenerateConnectionID() (string, error) {
	id, err := ulid.New(ulid.Now(), rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to generate ULID: %w", err)
	}
	return id.String(), nil
}

// FormatConnectMessage formats a CONNECT message with the given connection ID.
func FormatConnectMessage(connID string) string {
	return MessageTypePrefixConnect + connID
}

// FormatHandleMessage formats a HANDLE message with the given connection ID.
func FormatHandleMessage(connID string) string {
	return MessageTypePrefixHandle + connID
}

// ParseConnectMessage extracts the connection ID from a CONNECT message.
// Returns the connection ID and true if valid, empty string and false otherwise.
func ParseConnectMessage(message string) (string, bool) {
	if !strings.HasPrefix(message, MessageTypePrefixConnect) {
		return "", false
	}
	connID := message[len(MessageTypePrefixConnect):]
	if len(connID) != ulidLength {
		return "", false
	}
	return connID, true
}

// ParseHandleMessage extracts the connection ID from a HANDLE message.
// Returns the connection ID and true if valid, empty string and false otherwise.
func ParseHandleMessage(message string) (string, bool) {
	if !strings.HasPrefix(message, MessageTypePrefixHandle) {
		return "", false
	}
	connID := message[len(MessageTypePrefixHandle):]
	if len(connID) != ulidLength {
		return "", false
	}
	return connID, true
}

// ValidateConnectionID checks if a string is a valid ULID.
func ValidateConnectionID(connID string) bool {
	if len(connID) != ulidLength {
		return false
	}
	_, err := ulid.Parse(connID)
	return err == nil
}
