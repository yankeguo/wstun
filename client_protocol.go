package wstun

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ConnectRequestHandler is called when the client receives a CONNECT message from the server.
// The handler should establish a connection to the target service and return a ReadWriteCloser.
type ConnectRequestHandler func(ctx context.Context, connID string) (io.ReadWriteCloser, error)

// ClientProtocol manages WebSocket protocol handling on the client side.
type ClientProtocol struct {
	// ServerURL is the WebSocket server URL
	ServerURL string

	// ServiceName is the service identifier
	ServiceName string

	// ServiceToken is the authentication token
	ServiceToken string

	// ConnectHandler is called when CONNECT message is received
	ConnectHandler ConnectRequestHandler

	// Logger for protocol events (optional)
	Logger *log.Logger

	// Dialer for WebSocket connections
	Dialer *websocket.Dialer

	controlConn *websocket.Conn
	controlMu   sync.Mutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewClientProtocol creates a new client protocol handler.
func NewClientProtocol(serverURL, serviceName, serviceToken string) *ClientProtocol {
	ctx, cancel := context.WithCancel(context.Background())
	return &ClientProtocol{
		ServerURL:    serverURL,
		ServiceName:  serviceName,
		ServiceToken: serviceToken,
		Dialer:       websocket.DefaultDialer,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start establishes the control connection and starts listening for CONNECT messages.
// This method blocks until the connection is closed or an error occurs.
func (cp *ClientProtocol) Start() error {
	// Establish control connection
	conn, err := cp.dial()
	if err != nil {
		return fmt.Errorf("failed to establish control connection: %w", err)
	}

	cp.controlMu.Lock()
	cp.controlConn = conn
	cp.controlMu.Unlock()

	defer func() {
		cp.controlMu.Lock()
		if cp.controlConn != nil {
			cp.controlConn.Close()
			cp.controlConn = nil
		}
		cp.controlMu.Unlock()
	}()

	// Send WAITING message
	if err := conn.WriteMessage(websocket.TextMessage, []byte(MessageTypeWaiting)); err != nil {
		return fmt.Errorf("failed to send WAITING message: %w", err)
	}

	cp.logf("Control connection established, WAITING message sent")

	// Listen for messages
	return cp.listenControl(conn)
}

// Stop closes the control connection and stops the client.
func (cp *ClientProtocol) Stop() error {
	cp.cancel()

	cp.controlMu.Lock()
	defer cp.controlMu.Unlock()

	if cp.controlConn != nil {
		err := cp.controlConn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		)
		cp.controlConn.Close()
		cp.controlConn = nil
		return err
	}
	return nil
}

func (cp *ClientProtocol) dial() (*websocket.Conn, error) {
	// Parse server URL and add BasicAuth
	u, err := url.Parse(cp.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}

	// Create request header with BasicAuth
	header := http.Header{}
	header.Set("Authorization", basicAuth(cp.ServiceName, cp.ServiceToken))

	// Dial WebSocket (path-agnostic, server accepts any path)
	conn, resp, err := cp.Dialer.Dial(u.String(), header)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("dial failed with status %d: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("dial failed: %w", err)
	}

	return conn, nil
}

func (cp *ClientProtocol) listenControl(conn *websocket.Conn) error {
	for {
		select {
		case <-cp.ctx.Done():
			return cp.ctx.Err()
		default:
		}

		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				cp.logf("Control connection closed normally")
				return nil
			}
			return fmt.Errorf("failed to read message: %w", err)
		}

		if messageType != websocket.TextMessage {
			cp.logf("Received non-text message on control connection, ignoring")
			continue
		}

		messageStr := string(message)

		// Parse CONNECT message
		if connID, ok := ParseConnectMessage(messageStr); ok {
			cp.logf("Received CONNECT request: %s", connID)
			go cp.handleConnectRequest(connID)
		} else {
			cp.logf("Received unrecognized message: %s", messageStr)
		}
	}
}

func (cp *ClientProtocol) handleConnectRequest(connID string) {
	if cp.ConnectHandler == nil {
		cp.logf("No ConnectHandler configured, ignoring request")
		return
	}

	// Create context with timeout for establishing connection
	ctx, cancel := context.WithTimeout(cp.ctx, 30*time.Second)
	defer cancel()

	// Get connection to target service
	target, err := cp.ConnectHandler(ctx, connID)
	if err != nil {
		cp.logf("Failed to connect to target for %s: %v", connID, err)
		return
	}
	defer target.Close()

	// Establish data connection to server
	dataConn, err := cp.dial()
	if err != nil {
		cp.logf("Failed to establish data connection for %s: %v", connID, err)
		return
	}
	defer dataConn.Close()

	// Send HANDLE message
	if err := dataConn.WriteMessage(websocket.TextMessage, []byte(FormatHandleMessage(connID))); err != nil {
		cp.logf("Failed to send HANDLE message for %s: %v", connID, err)
		return
	}

	cp.logf("Data connection established for: %s", connID)

	// Perform bidirectional data transfer
	cp.transfer(dataConn, target, connID)
}

// transfer performs bidirectional data transfer between WebSocket and target connection.
func (cp *ClientProtocol) transfer(ws *websocket.Conn, target io.ReadWriteCloser, connID string) {
	var wg sync.WaitGroup
	wg.Add(2)

	// WebSocket -> Target
	go func() {
		defer wg.Done()
		defer target.Close()
		for {
			messageType, data, err := ws.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					cp.logf("[%s] WebSocket read error: %v", connID, err)
				}
				return
			}

			if messageType != websocket.BinaryMessage {
				cp.logf("[%s] Received non-binary message, ignoring", connID)
				continue
			}

			if _, err := target.Write(data); err != nil {
				cp.logf("[%s] Target write error: %v", connID, err)
				return
			}
		}
	}()

	// Target -> WebSocket
	go func() {
		defer wg.Done()
		defer ws.Close()
		buf := make([]byte, 32*1024) // 32KB buffer
		for {
			n, err := target.Read(buf)
			if err != nil {
				if err != io.EOF {
					cp.logf("[%s] Target read error: %v", connID, err)
				}
				return
			}

			if err := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				cp.logf("[%s] WebSocket write error: %v", connID, err)
				return
			}
		}
	}()

	wg.Wait()
	cp.logf("Data transfer completed for: %s", connID)
}

func (cp *ClientProtocol) logf(format string, args ...interface{}) {
	if cp.Logger != nil {
		cp.Logger.Printf("[ClientProtocol] "+format, args...)
	}
}

// basicAuth creates a BasicAuth header value.
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return "Basic " + base64Encode(auth)
}

// base64Encode encodes a string to base64 (standard encoding).
func base64Encode(s string) string {
	const base64Table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	data := []byte(s)
	n := len(data)
	if n == 0 {
		return ""
	}

	// Calculate output length
	outLen := (n + 2) / 3 * 4
	out := make([]byte, outLen)

	si := 0
	oi := 0
	for si < n-2 {
		val := uint32(data[si])<<16 | uint32(data[si+1])<<8 | uint32(data[si+2])
		out[oi] = base64Table[(val>>18)&0x3F]
		out[oi+1] = base64Table[(val>>12)&0x3F]
		out[oi+2] = base64Table[(val>>6)&0x3F]
		out[oi+3] = base64Table[val&0x3F]
		si += 3
		oi += 4
	}

	remain := n - si
	if remain == 1 {
		val := uint32(data[si]) << 16
		out[oi] = base64Table[(val>>18)&0x3F]
		out[oi+1] = base64Table[(val>>12)&0x3F]
		out[oi+2] = '='
		out[oi+3] = '='
	} else if remain == 2 {
		val := uint32(data[si])<<16 | uint32(data[si+1])<<8
		out[oi] = base64Table[(val>>18)&0x3F]
		out[oi+1] = base64Table[(val>>12)&0x3F]
		out[oi+2] = base64Table[(val>>6)&0x3F]
		out[oi+3] = '='
	}

	return string(out)
}
