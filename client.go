package wstun

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	defaultPingInterval     = 30 * time.Second
	defaultPongWait         = 60 * time.Second
	defaultWriteWait        = 10 * time.Second
	connectHandlerTimeout   = 30 * time.Second
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
	Logger         *slog.Logger
	Dialer         *websocket.Dialer

	// PingInterval is how often to send WebSocket pings, on both the control
	// connection and every data connection (default: 30s).
	// Setting it to <= 0 disables keepalive pings entirely; data connections
	// then carry no read deadline, which leaves dead-peer detection to TCP
	// keepalive alone.
	PingInterval time.Duration

	// PongWait is the grace period for a pong reply before a connection is
	// considered dead (default: 60s). On data connections the effective read
	// deadline is PingInterval+PongWait, renewed by every pong and every
	// message received, so an idle-but-alive tunnel (e.g. SSE with long gaps
	// between events) is never closed.
	PongWait time.Duration

	// WriteWait is the write deadline applied to all WebSocket writes (default: 10s).
	WriteWait time.Duration

	controlConn  *websocket.Conn
	controlMu    sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	activeConns  atomic.Int64
	shutdownMode atomic.Bool
}

// NewClientProtocol creates a new client protocol handler.
func NewClientProtocol(serverURL, serviceName, serviceToken string) *ClientProtocol {
	ctx, cancel := context.WithCancel(context.Background())
	return &ClientProtocol{
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
}

// logger returns the configured logger (with "service" attribute) or slog.Default().
func (cp *ClientProtocol) logger() *slog.Logger {
	if cp.Logger != nil {
		return cp.Logger.With("service", cp.ServiceName)
	}
	return slog.Default().With("service", cp.ServiceName)
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

	cp.logger().Info("control connection established, WAITING message sent")

	cp.wg.Add(1)
	go cp.pingLoop(conn)

	err = cp.listenControl(conn)

	cp.cancel()
	cp.wg.Wait()

	return err
}

func (cp *ClientProtocol) pingLoop(conn *websocket.Conn) {
	defer cp.wg.Done()

	// PingInterval <= 0 disables keepalive; guarding here also prevents
	// time.NewTicker from panicking on a non-positive interval.
	if cp.PingInterval <= 0 {
		return
	}

	ticker := time.NewTicker(cp.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cp.ctx.Done():
			return
		case <-ticker.C:
			// WriteControl is safe for concurrent use with all other methods,
			// so no controlMu is needed here.
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(cp.WriteWait)); err != nil {
				cp.logger().Warn("ping failed", "error", err)
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
		cp.logger().Warn("graceful shutdown timeout, forcing close")
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
		if cp.activeConns.Load() == 0 {
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
				cp.logger().Info("control connection closed normally")
				return nil
			}
			if errors.Is(err, context.Canceled) || cp.ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("failed to read message: %w", err)
		}

		if err := conn.SetReadDeadline(time.Now().Add(cp.PongWait)); err != nil {
			cp.logger().Warn("failed to update read deadline", "error", err)
		}

		if messageType != websocket.TextMessage {
			cp.logger().Debug("received non-text message on control connection, ignoring")
			continue
		}

		messageStr := string(message)

		if connID, ok := ParseConnectMessage(messageStr); ok {
			cp.logger().Info("received CONNECT request", "conn_id", connID)
			go cp.handleConnectRequest(connID)
		} else {
			cp.logger().Warn("received unrecognized message", "message", messageStr)
		}
	}
}

func (cp *ClientProtocol) handleConnectRequest(connID string) {
	log := cp.logger().With("conn_id", connID)

	defer func() {
		if r := recover(); r != nil {
			log.Error("panic in handleConnectRequest", "panic", r)
		}
	}()

	if cp.ConnectHandler == nil {
		log.Warn("no ConnectHandler configured, ignoring request")
		return
	}

	if cp.shutdownMode.Load() {
		log.Info("client in shutdown mode, ignoring CONNECT request")
		return
	}

	cp.activeConns.Add(1)
	defer cp.activeConns.Add(-1)

	ctx, cancel := context.WithTimeout(cp.ctx, connectHandlerTimeout)
	defer cancel()

	target, err := cp.ConnectHandler(ctx, connID)
	if err != nil {
		log.Error("failed to connect to target", "error", err)
		return
	}
	if target == nil {
		log.Error("ConnectHandler returned nil connection")
		return
	}
	defer target.Close()

	var dataConn *websocket.Conn
	var lastErr error

	for attempt := 0; attempt < dataConnectMaxRetries; attempt++ {
		if cp.shutdownMode.Load() || cp.ctx.Err() != nil {
			log.Info("client shutting down, aborting data connection")
			return
		}

		dataConn, lastErr = cp.dial()
		if lastErr == nil {
			break
		}

		log.Warn("data connection attempt failed", "attempt", attempt+1, "max_attempts", dataConnectMaxRetries, "error", lastErr)

		if attempt < dataConnectMaxRetries-1 {
			select {
			case <-cp.ctx.Done():
				return
			case <-time.After(dataConnectRetryDelay):
			}
		}
	}

	if lastErr != nil {
		log.Error("failed to establish data connection after retries", "attempts", dataConnectMaxRetries, "error", lastErr)
		return
	}
	defer dataConn.Close()

	if err := dataConn.WriteMessage(websocket.TextMessage, []byte(FormatHandleMessage(connID))); err != nil {
		log.Error("failed to send HANDLE message", "error", err)
		return
	}

	log.Info("data connection established")

	cp.transfer(dataConn, target, connID)
}

func (cp *ClientProtocol) transfer(ws *websocket.Conn, target io.ReadWriteCloser, connID string) {
	log := cp.logger().With("conn_id", connID)

	// Data-connection keepalive: send periodic pings and bound the read side with
	// a deadline of PingInterval+PongWait, renewed by every pong and every message
	// received. This keeps the tunnel leg active through proxies with idle timeouts
	// and detects dead peers, while never closing an idle-but-alive connection
	// (e.g. SSE with long gaps between events). Disabled when PingInterval <= 0.
	keepaliveDone := make(chan struct{})
	var keepaliveWg sync.WaitGroup
	keepaliveEnabled := cp.PingInterval > 0
	if keepaliveEnabled {
		refreshReadDeadline := func() {
			if err := ws.SetReadDeadline(time.Now().Add(cp.PingInterval + cp.PongWait)); err != nil {
				log.Debug("failed to refresh read deadline", "error", err)
			}
		}
		refreshReadDeadline()
		ws.SetPongHandler(func(string) error {
			refreshReadDeadline()
			return nil
		})
		keepaliveWg.Add(1)
		go func() {
			defer keepaliveWg.Done()
			ticker := time.NewTicker(cp.PingInterval)
			defer ticker.Stop()
			for {
				select {
				case <-keepaliveDone:
					return
				case <-ticker.C:
					// WriteControl is safe for concurrent use with all other methods.
					// On failure the read deadline will fire and end the transfer.
					if err := ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(cp.WriteWait)); err != nil {
						return
					}
				}
			}
		}()
		defer func() {
			close(keepaliveDone)
			keepaliveWg.Wait()
		}()
	}

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
				log.Error("panic in WebSocket->Target transfer", "panic", r)
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
							log.Error("websocket read error", "error", err)
						}
					}
				}
				return
			}

			// Any message received also proves the peer is alive; renew the deadline.
			if keepaliveEnabled {
				if err := ws.SetReadDeadline(time.Now().Add(cp.PingInterval + cp.PongWait)); err != nil {
					log.Debug("failed to refresh read deadline", "error", err)
				}
			}

			if messageType != websocket.BinaryMessage {
				log.Debug("received non-binary message, ignoring")
				continue
			}

			if len(data) == 0 {
				continue
			}

			if _, err := target.Write(data); err != nil {
				log.Error("target write error", "error", err)
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
				log.Error("panic in Target->WebSocket transfer", "panic", r)
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
					log.Error("failed to set write deadline", "error", writeErr)
					return
				}
				if writeErr := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
					log.Error("websocket write error", "error", writeErr)
					return
				}
			}
			if err != nil {
				if err != io.EOF && !errors.Is(err, context.Canceled) {
					log.Error("target read error", "error", err)
				}
				return
			}
		}
	}()

	wg.Wait()
	log.Info("data transfer completed")
}

// basicAuth creates a BasicAuth header value.
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}
