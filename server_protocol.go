package wstun

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	// ErrInvalidCredentials indicates authentication failure.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrInvalidFirstMessage indicates the first message is not a valid text message.
	ErrInvalidFirstMessage = errors.New("invalid first message")

	// ErrUnknownConnectionID indicates HANDLE message references unknown connection ID.
	ErrUnknownConnectionID = errors.New("unknown connection ID")

	// ErrConnectionTimeout indicates connection handling timed out.
	ErrConnectionTimeout = errors.New("connection timeout")
)

// ServiceAuthFunc is a function type for authenticating service credentials.
// Returns the service name if authentication succeeds, empty string otherwise.
type ServiceAuthFunc func(serviceName, serviceToken string) bool

// ControlConnectionHandler is called when a control connection (WAITING) is established.
// The handler should read from the connection and handle CONNECT messages or other control logic.
type ControlConnectionHandler func(conn *websocket.Conn, serviceName string)

// DataConnectionHandler is called when a data connection (HANDLE) is established and matched.
// The handler should perform bidirectional data transfer.
// incoming is the connection from the original request, outgoing is the WebSocket to the client.
type DataConnectionHandler func(incoming io.ReadWriteCloser, outgoing *websocket.Conn)

// ServerProtocol manages WebSocket protocol handling on the server side.
type ServerProtocol struct {
	// AuthFunc authenticates service credentials
	AuthFunc ServiceAuthFunc

	// ControlHandler handles control connections
	ControlHandler ControlConnectionHandler

	// DataHandler handles data connections
	DataHandler DataConnectionHandler

	// PendingTimeout is the maximum time to wait for HANDLE after sending CONNECT
	PendingTimeout time.Duration

	// Logger for protocol events (optional)
	Logger *log.Logger

	upgrader websocket.Upgrader

	// pendingConns maps connection ID to pending connection info
	pendingConns sync.Map

	// controlConns maps service name to list of control connections
	controlConns sync.Map
}

type pendingConnection struct {
	serviceName string
	incoming    io.ReadWriteCloser
	created     time.Time
	cancel      context.CancelFunc
}

// NewServerProtocol creates a new server protocol handler.
func NewServerProtocol(authFunc ServiceAuthFunc) *ServerProtocol {
	return &ServerProtocol{
		AuthFunc:       authFunc,
		PendingTimeout: 30 * time.Second,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Accept all origins; security is via BasicAuth
			},
		},
	}
}

// HandleWebSocket handles incoming WebSocket connections.
// This is the single, path-agnostic endpoint for all WebSocket connections.
func (sp *ServerProtocol) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract and verify BasicAuth credentials
	serviceName, serviceToken, ok := r.BasicAuth()
	if !ok {
		http.Error(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	if sp.AuthFunc == nil || !sp.AuthFunc(serviceName, serviceToken) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		sp.logf("Authentication failed for service: %s", serviceName)
		return
	}

	// Upgrade to WebSocket
	conn, err := sp.upgrader.Upgrade(w, r, nil)
	if err != nil {
		sp.logf("WebSocket upgrade failed: %v", err)
		return
	}

	// Read first text message to determine connection type
	messageType, message, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		sp.logf("Failed to read first message: %v", err)
		return
	}

	if messageType != websocket.TextMessage {
		conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseUnsupportedData, "First message must be text"),
			time.Now().Add(time.Second))
		conn.Close()
		sp.logf("Invalid first message type from service %s", serviceName)
		return
	}

	messageStr := string(message)

	// Route based on message content
	if messageStr == MessageTypeWaiting {
		sp.handleControlConnection(conn, serviceName)
	} else if connID, ok := ParseHandleMessage(messageStr); ok {
		sp.handleDataConnection(conn, serviceName, connID)
	} else {
		conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Unrecognized first message"),
			time.Now().Add(time.Second))
		conn.Close()
		sp.logf("Unrecognized first message from service %s: %s", serviceName, messageStr)
	}
}

func (sp *ServerProtocol) handleControlConnection(conn *websocket.Conn, serviceName string) {
	sp.logf("Control connection established for service: %s", serviceName)

	// Register control connection
	sp.registerControlConnection(serviceName, conn)
	defer sp.unregisterControlConnection(serviceName, conn)

	// Delegate to handler if provided
	if sp.ControlHandler != nil {
		sp.ControlHandler(conn, serviceName)
	} else {
		// Default: keep connection open, read until close
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}

	sp.logf("Control connection closed for service: %s", serviceName)
}

func (sp *ServerProtocol) handleDataConnection(conn *websocket.Conn, serviceName string, connID string) {
	sp.logf("Data connection received for service %s, connection ID: %s", serviceName, connID)

	// Look up pending connection
	value, ok := sp.pendingConns.Load(connID)
	if !ok {
		conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Unknown connection ID"),
			time.Now().Add(time.Second))
		conn.Close()
		sp.logf("Unknown connection ID: %s", connID)
		return
	}

	pending := value.(*pendingConnection)

	// Verify connection belongs to the authenticated service
	if pending.serviceName != serviceName {
		conn.WriteControl(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Connection ID mismatch"),
			time.Now().Add(time.Second))
		conn.Close()
		sp.logf("Connection ID %s does not belong to service %s", connID, serviceName)
		return
	}

	// Remove from pending
	sp.pendingConns.Delete(connID)
	pending.cancel() // Cancel timeout

	sp.logf("Data connection matched for service %s, connection ID: %s", serviceName, connID)

	// Delegate to handler
	if sp.DataHandler != nil {
		sp.DataHandler(pending.incoming, conn)
	}
}

// RequestConnection notifies a client about an incoming connection request.
// Returns the connection ID and a channel to wait for the data connection.
// The incoming connection should be provided and will be paired with the client's HANDLE connection.
func (sp *ServerProtocol) RequestConnection(serviceName string, incoming io.ReadWriteCloser) (string, error) {
	// Generate connection ID
	connID, err := GenerateConnectionID()
	if err != nil {
		return "", fmt.Errorf("failed to generate connection ID: %w", err)
	}

	// Create pending connection with timeout context
	ctx, cancel := context.WithTimeout(context.Background(), sp.PendingTimeout)
	pending := &pendingConnection{
		serviceName: serviceName,
		incoming:    incoming,
		created:     time.Now(),
		cancel:      cancel,
	}

	sp.pendingConns.Store(connID, pending)

	// Start timeout handler
	go func() {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			// Timeout occurred, remove pending connection
			if _, ok := sp.pendingConns.Load(connID); ok {
				sp.pendingConns.Delete(connID)
				incoming.Close()
				sp.logf("Connection request timeout for ID: %s", connID)
			}
		}
	}()

	// Send CONNECT message to control connections for this service
	message := FormatConnectMessage(connID)
	sp.sendToControlConnections(serviceName, message)

	sp.logf("Connection request sent to service %s with ID: %s", serviceName, connID)
	return connID, nil
}

func (sp *ServerProtocol) registerControlConnection(serviceName string, conn *websocket.Conn) {
	value, _ := sp.controlConns.LoadOrStore(serviceName, &sync.Map{})
	conns := value.(*sync.Map)
	conns.Store(conn, struct{}{})
}

func (sp *ServerProtocol) unregisterControlConnection(serviceName string, conn *websocket.Conn) {
	value, ok := sp.controlConns.Load(serviceName)
	if !ok {
		return
	}
	conns := value.(*sync.Map)
	conns.Delete(conn)
}

func (sp *ServerProtocol) sendToControlConnections(serviceName string, message string) {
	value, ok := sp.controlConns.Load(serviceName)
	if !ok {
		sp.logf("No control connections for service: %s", serviceName)
		return
	}

	conns := value.(*sync.Map)
	conns.Range(func(key, _ interface{}) bool {
		conn := key.(*websocket.Conn)
		// Write synchronously to avoid concurrent write issues
		if err := conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
			sp.logf("Failed to send message to control connection: %v", err)
		}
		return true
	})
}

func (sp *ServerProtocol) logf(format string, args ...interface{}) {
	if sp.Logger != nil {
		sp.Logger.Printf("[ServerProtocol] "+format, args...)
	}
}
