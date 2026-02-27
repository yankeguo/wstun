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

	// RetryAttempts is the maximum number of retry attempts for connection requests (0 = no retry)
	RetryAttempts int

	// RetryInterval is the initial retry interval
	RetryInterval time.Duration

	// RetryMaxInterval is the maximum retry interval
	RetryMaxInterval time.Duration

	// ControlPingInterval is the interval for server-side keepalive pings on control connections.
	// Defaults to 15 seconds.
	ControlPingInterval time.Duration

	// WriteWait is the write deadline applied to all server-side WebSocket writes.
	// Defaults to 10 seconds.
	WriteWait time.Duration

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

type controlConnection struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
}

// NewServerProtocol creates a new server protocol handler.
func NewServerProtocol(authFunc ServiceAuthFunc) *ServerProtocol {
	return &ServerProtocol{
		AuthFunc:            authFunc,
		PendingTimeout:      30 * time.Second,
		RetryAttempts:       0,
		RetryInterval:       5 * time.Second,
		RetryMaxInterval:    60 * time.Second,
		ControlPingInterval: 15 * time.Second,
		WriteWait:           10 * time.Second,
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

	// Register control connection and retain the wrapper for the ping goroutine.
	cc := sp.registerControlConnection(serviceName, conn)
	defer sp.unregisterControlConnection(serviceName, conn)

	// Start keepalive ping goroutine; close done when this function exits.
	doneCh := make(chan struct{})
	defer close(doneCh)
	go sp.pingControlConnection(doneCh, serviceName, cc)

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

	// Atomically load and delete pending connection
	value, loaded := sp.pendingConns.LoadAndDelete(connID)
	if !loaded {
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
		pending.incoming.Close()
		sp.logf("Connection ID %s does not belong to service %s", connID, serviceName)
		return
	}

	// Cancel timeout
	pending.cancel()

	sp.logf("Data connection matched for service %s, connection ID: %s", serviceName, connID)

	// Delegate to handler
	if sp.DataHandler != nil {
		// Use defer/recover to catch panics in handler
		defer func() {
			if r := recover(); r != nil {
				sp.logf("Panic in DataHandler for connection %s: %v", connID, r)
			}
			conn.Close()
			pending.incoming.Close()
		}()
		sp.DataHandler(pending.incoming, conn)
	} else {
		sp.logf("No DataHandler configured, closing connection: %s", connID)
		conn.Close()
		pending.incoming.Close()
	}
}

// RequestConnection notifies a client about an incoming connection request.
// It sends a CONNECT:[ULID] message over the control connection and waits for the client
// to establish a matching data connection. Supports exponential-backoff retry when
// RetryAttempts > 0. Closes incoming and returns an error when all attempts are exhausted.
func (sp *ServerProtocol) RequestConnection(serviceName string, incoming io.ReadWriteCloser) (string, error) {
	// Generate connection ID (same ULID is reused across all retry attempts)
	connID, err := GenerateConnectionID()
	if err != nil {
		return "", fmt.Errorf("failed to generate connection ID: %w", err)
	}

	for attempt := 0; ; attempt++ {
		// Fast-fail if no control connections are registered: avoid spending PendingTimeout
		// waiting for a HANDLE that will never come.
		if !sp.hasControlConnections(serviceName) {
			sp.logf("No control connections available for service %s (attempt %d)", serviceName, attempt+1)
			if attempt >= sp.RetryAttempts {
				incoming.Close()
				return "", fmt.Errorf("connection request failed after %d attempt(s): no control connections", attempt+1)
			}
			// Count as a failed attempt without sleeping; move on immediately.
			continue
		}

		// Create a fresh timeout context for each attempt
		ctx, cancel := context.WithTimeout(context.Background(), sp.PendingTimeout)
		pending := &pendingConnection{
			serviceName: serviceName,
			incoming:    incoming,
			created:     time.Now(),
			cancel:      cancel,
		}

		sp.pendingConns.Store(connID, pending)

		// Timeout goroutine: only removes the pending entry; incoming is NOT closed
		// here so that retry attempts can reuse the same connection.
		timeoutChan := make(chan bool, 1)
		go func() {
			<-ctx.Done()
			timedOut := ctx.Err() == context.DeadlineExceeded
			if timedOut {
				sp.pendingConns.Delete(connID)
				sp.logf("Connection timeout for pending connection: %s", connID)
			}
			timeoutChan <- timedOut
		}()

		// Send CONNECT message to all control connections for this service
		sp.sendToControlConnections(serviceName, FormatConnectMessage(connID))

		if attempt == 0 {
			sp.logf("Connection request sent to service %s with ID: %s", serviceName, connID)
		} else {
			sp.logf("Retrying connection request for service %s with ID: %s (attempt %d/%d)", serviceName, connID, attempt, sp.RetryAttempts)
		}

		// Wait for the data connection handler to cancel the context (success)
		// or for the deadline to fire (timeout).
		timedOut := <-timeoutChan

		if !timedOut {
			// Data connection was successfully matched
			return connID, nil
		}

		// Timeout – all retry attempts used up; close the incoming connection.
		if attempt >= sp.RetryAttempts {
			incoming.Close()
			return "", fmt.Errorf("connection request failed after %d attempt(s)", attempt+1)
		}

		// Compute exponential backoff: RetryInterval * 2^attempt, capped at RetryMaxInterval
		retryDelay := sp.RetryInterval
		for i := 0; i < attempt; i++ {
			retryDelay *= 2
			if retryDelay > sp.RetryMaxInterval {
				retryDelay = sp.RetryMaxInterval
				break
			}
		}
		sp.logf("Waiting %v before retry %d/%d for connection %s", retryDelay, attempt+1, sp.RetryAttempts, connID)
		time.Sleep(retryDelay)
	}
}

func (sp *ServerProtocol) registerControlConnection(serviceName string, conn *websocket.Conn) *controlConnection {
	value, _ := sp.controlConns.LoadOrStore(serviceName, &sync.Map{})
	conns := value.(*sync.Map)
	cc := &controlConnection{conn: conn}
	conns.Store(cc, struct{}{})
	return cc
}

// removeControlConnection removes a specific control connection wrapper from the registry.
// It is safe to call concurrently; deleting an already-deleted key is a no-op.
func (sp *ServerProtocol) removeControlConnection(serviceName string, cc *controlConnection) {
	value, ok := sp.controlConns.Load(serviceName)
	if !ok {
		return
	}
	conns := value.(*sync.Map)
	conns.Delete(cc)
}

// hasControlConnections reports whether at least one control connection is
// currently registered for the given service.
func (sp *ServerProtocol) hasControlConnections(serviceName string) bool {
	value, ok := sp.controlConns.Load(serviceName)
	if !ok {
		return false
	}
	conns := value.(*sync.Map)
	found := false
	conns.Range(func(_, _ interface{}) bool {
		found = true
		return false // stop at first entry
	})
	return found
}

// pingControlConnection sends periodic WebSocket ping frames on a control connection
// to prevent proxy idle-timeout from silently resetting it.
// It exits when done is closed or a write fails.
func (sp *ServerProtocol) pingControlConnection(done <-chan struct{}, serviceName string, cc *controlConnection) {
	interval := sp.ControlPingInterval
	if interval <= 0 {
		interval = 15 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			cc.writeMu.Lock()
			err := cc.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(sp.WriteWait))
			cc.writeMu.Unlock()
			if err != nil {
				sp.logf("Keepalive ping failed for service %s: %v", serviceName, err)
				return
			}
		}
	}
}

func (sp *ServerProtocol) unregisterControlConnection(serviceName string, conn *websocket.Conn) {
	value, ok := sp.controlConns.Load(serviceName)
	if !ok {
		return
	}
	conns := value.(*sync.Map)
	// Find and remove the control connection wrapper
	conns.Range(func(key, _ interface{}) bool {
		cc := key.(*controlConnection)
		if cc.conn == conn {
			conns.Delete(key)
			return false // stop iteration
		}
		return true
	})
}

func (sp *ServerProtocol) sendToControlConnections(serviceName string, message string) {
	value, ok := sp.controlConns.Load(serviceName)
	if !ok {
		sp.logf("No control connections for service: %s", serviceName)
		return
	}

	conns := value.(*sync.Map)
	conns.Range(func(key, _ interface{}) bool {
		cc := key.(*controlConnection)
		// Use mutex to protect concurrent writes; set write deadline to prevent hangs.
		cc.writeMu.Lock()
		cc.conn.SetWriteDeadline(time.Now().Add(sp.WriteWait))
		err := cc.conn.WriteMessage(websocket.TextMessage, []byte(message))
		cc.writeMu.Unlock()

		if err != nil {
			sp.logf("Failed to send message to control connection, removing stale entry: %v", err)
			sp.removeControlConnection(serviceName, cc)
		}
		return true
	})
}

func (sp *ServerProtocol) logf(format string, args ...interface{}) {
	if sp.Logger != nil {
		sp.Logger.Printf("[ServerProtocol] "+format, args...)
	}
}
