package wstun

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

var (
	ErrClientStopped = errors.New("client stopped")
)

const (
	defaultPingInterval     = 30 * time.Second
	defaultPongWait         = 60 * time.Second
	defaultWriteWait        = 10 * time.Second
	defaultReadTimeout      = 0
	dataConnectMaxRetries   = 3
	dataConnectRetryDelay   = 500 * time.Millisecond
	gracefulShutdownTimeout = 5 * time.Second
)

type ConnectRequestHandler func(ctx context.Context, connID string) (io.ReadWriteCloser, error)

type ClientProtocol struct {
	ServerURL      string
	ServiceName    string
	ServiceToken   string
	ConnectHandler ConnectRequestHandler
	Logger         *log.Logger
	Dialer         *websocket.Dialer

	PingInterval time.Duration
	PongWait     time.Duration
	WriteWait    time.Duration

	controlConn  *websocket.Conn
	controlMu    sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	activeConns  int64
	shutdownMode atomic.Bool
}

// NewClientProtocol creates a new client protocol handler.
func NewClientProtocol(serverURL, serviceName, serviceToken string) *ClientProtocol {
	ctx, cancel := context.WithCancel(context.Background())
	cp := &ClientProtocol{
		ServerURL:    serverURL,
		ServiceName:  serviceName,
		ServiceToken: serviceToken,
		Dialer:       websocket.DefaultDialer,
		PingInterval: defaultPingInterval,
		PongWait:     defaultPongWait,
		WriteWait:    defaultWriteWait,
		ctx:          ctx,
		cancel:       cancel,
	}
	return cp
}

// Start establishes the control connection and starts listening for CONNECT messages.
// This method blocks until the connection is closed or an error occurs.
func (cp *ClientProtocol) Start() error {
	conn, err := cp.dial()
	if err != nil {
		return fmt.Errorf("failed to establish control connection: %w", err)
	}

	cp.controlMu.Lock()
	cp.controlConn = conn
	cp.shutdownMode.Store(false)
	cp.controlMu.Unlock()

	defer func() {
		cp.closeControlConn()
	}()

	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(cp.PongWait))
	})

	if err := conn.SetReadDeadline(time.Now().Add(cp.PongWait)); err != nil {
		return fmt.Errorf("failed to set read deadline: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, []byte(MessageTypeWaiting)); err != nil {
		return fmt.Errorf("failed to send WAITING message: %w", err)
	}

	cp.logf("Control connection established, WAITING message sent")

	cp.wg.Add(1)
	go cp.pingLoop(conn)

	err = cp.listenControl(conn)

	cp.cancel()
	cp.wg.Wait()

	return err
}

func (cp *ClientProtocol) pingLoop(conn *websocket.Conn) {
	defer cp.wg.Done()

	ticker := time.NewTicker(cp.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cp.ctx.Done():
			return
		case <-ticker.C:
			cp.controlMu.Lock()
			if cp.controlConn == nil {
				cp.controlMu.Unlock()
				return
			}
			err := cp.controlConn.WriteControl(websocket.PingMessage, nil, time.Now().Add(cp.WriteWait))
			cp.controlMu.Unlock()
			if err != nil {
				cp.logf("Ping failed: %v", err)
				return
			}
		}
	}
}

func (cp *ClientProtocol) closeControlConn() {
	cp.controlMu.Lock()
	defer cp.controlMu.Unlock()
	if cp.controlConn != nil {
		cp.controlConn.Close()
		cp.controlConn = nil
	}
}

func (cp *ClientProtocol) Stop() error {
	cp.shutdownMode.Store(true)
	cp.cancel()

	done := make(chan struct{})
	go func() {
		cp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(gracefulShutdownTimeout):
		cp.logf("Graceful shutdown timeout, forcing close")
	}

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

func (cp *ClientProtocol) WaitForActiveConnections(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if atomic.LoadInt64(&cp.activeConns) == 0 {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

func (cp *ClientProtocol) dial() (*websocket.Conn, error) {
	// Parse server URL
	u, err := url.Parse(cp.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %w", err)
	}

	// Create request header with BasicAuth and Origin
	header := http.Header{}
	header.Set("Authorization", basicAuth(cp.ServiceName, cp.ServiceToken))

	// Set Origin header inferred from WebSocket URL (ws->http, wss->https)
	originScheme := "http"
	if u.Scheme == "wss" {
		originScheme = "https"
	}
	header.Set("Origin", originScheme+"://"+u.Host)

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
			if errors.Is(err, context.Canceled) || cp.ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("failed to read message: %w", err)
		}

		if err := conn.SetReadDeadline(time.Now().Add(cp.PongWait)); err != nil {
			cp.logf("Failed to update read deadline: %v", err)
		}

		if messageType != websocket.TextMessage {
			cp.logf("Received non-text message on control connection, ignoring")
			continue
		}

		messageStr := string(message)

		if connID, ok := ParseConnectMessage(messageStr); ok {
			cp.logf("Received CONNECT request: %s", connID)
			go cp.handleConnectRequest(connID)
		} else {
			cp.logf("Received unrecognized message: %s", messageStr)
		}
	}
}

func (cp *ClientProtocol) handleConnectRequest(connID string) {
	defer func() {
		if r := recover(); r != nil {
			cp.logf("Panic in handleConnectRequest for %s: %v", connID, r)
		}
	}()

	if cp.ConnectHandler == nil {
		cp.logf("No ConnectHandler configured, ignoring request")
		return
	}

	if cp.shutdownMode.Load() {
		cp.logf("Client in shutdown mode, ignoring CONNECT request: %s", connID)
		return
	}

	atomic.AddInt64(&cp.activeConns, 1)
	defer atomic.AddInt64(&cp.activeConns, -1)

	ctx, cancel := context.WithTimeout(cp.ctx, 30*time.Second)
	defer cancel()

	target, err := cp.ConnectHandler(ctx, connID)
	if err != nil {
		cp.logf("Failed to connect to target for %s: %v", connID, err)
		return
	}
	if target == nil {
		cp.logf("ConnectHandler returned nil connection for %s", connID)
		return
	}
	defer target.Close()

	var dataConn *websocket.Conn
	var lastErr error

	for attempt := 0; attempt < dataConnectMaxRetries; attempt++ {
		if cp.shutdownMode.Load() || cp.ctx.Err() != nil {
			cp.logf("Client shutting down, aborting data connection for %s", connID)
			return
		}

		dataConn, lastErr = cp.dial()
		if lastErr == nil {
			break
		}

		cp.logf("Data connection attempt %d/%d failed for %s: %v", attempt+1, dataConnectMaxRetries, connID, lastErr)

		if attempt < dataConnectMaxRetries-1 {
			select {
			case <-cp.ctx.Done():
				return
			case <-time.After(dataConnectRetryDelay):
			}
		}
	}

	if lastErr != nil {
		cp.logf("Failed to establish data connection for %s after %d attempts: %v", connID, dataConnectMaxRetries, lastErr)
		return
	}
	defer dataConn.Close()

	if err := dataConn.WriteMessage(websocket.TextMessage, []byte(FormatHandleMessage(connID))); err != nil {
		cp.logf("Failed to send HANDLE message for %s: %v", connID, err)
		return
	}

	cp.logf("Data connection established for: %s", connID)

	cp.transfer(dataConn, target, connID)
}

func (cp *ClientProtocol) transfer(ws *websocket.Conn, target io.ReadWriteCloser, connID string) {
	var wg sync.WaitGroup
	var closeChOnce sync.Once
	var closeWSOnce sync.Once
	var closeTargetOnce sync.Once
	closeCh := make(chan struct{})

	signalClose := func() {
		closeChOnce.Do(func() {
			close(closeCh)
		})
	}

	closeWebSocket := func() {
		closeWSOnce.Do(func() {
			ws.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
				time.Now().Add(cp.WriteWait))
			ws.Close()
		})
	}

	closeTarget := func() {
		closeTargetOnce.Do(func() {
			target.Close()
		})
	}

	wg.Add(2)

	go func() {
		defer wg.Done()
		defer closeTarget()
		defer signalClose()
		defer func() {
			if r := recover(); r != nil {
				cp.logf("[%s] Panic in WebSocket->Target transfer: %v", connID, r)
			}
		}()

		for {
			messageType, data, err := ws.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
						// Suppress spurious error when closeCh fired and closeWebSocket() raced with ReadMessage
						select {
						case <-closeCh:
						default:
							cp.logf("[%s] WebSocket read error: %v", connID, err)
						}
					}
				}
				return
			}

			if messageType != websocket.BinaryMessage {
				cp.logf("[%s] Received non-binary message, ignoring", connID)
				continue
			}

			if len(data) == 0 {
				continue
			}

			if _, err := target.Write(data); err != nil {
				cp.logf("[%s] Target write error: %v", connID, err)
				return
			}
		}
	}()

	go func() {
		defer wg.Done()
		defer closeWebSocket()
		defer signalClose()
		defer func() {
			if r := recover(); r != nil {
				cp.logf("[%s] Panic in Target->WebSocket transfer: %v", connID, r)
			}
		}()

		buf := make([]byte, 32*1024)
		for {
			select {
			case <-closeCh:
				return
			default:
			}

			n, err := target.Read(buf)
			// Process n bytes BEFORE checking err: Read() can return n>0 and io.EOF
			// simultaneously on the final chunk; discarding those bytes causes data loss.
			if n > 0 {
				if writeErr := ws.SetWriteDeadline(time.Now().Add(cp.WriteWait)); writeErr != nil {
					cp.logf("[%s] Failed to set write deadline: %v", connID, writeErr)
					return
				}
				if writeErr := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
					cp.logf("[%s] WebSocket write error: %v", connID, writeErr)
					return
				}
			}
			if err != nil {
				if err != io.EOF && !errors.Is(err, context.Canceled) {
					cp.logf("[%s] Target read error: %v", connID, err)
				}
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
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}
