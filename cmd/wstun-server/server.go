package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"cnb.cool/yankeguo/wstun"
	"github.com/gorilla/websocket"
)

// shutdownTimeout bounds graceful shutdown phases (HTTP server and goroutines).
const shutdownTimeout = 10 * time.Second

// Server encapsulates the server state and operations.
type Server struct {
	config      *wstun.ServerOptions
	services    map[string]*wstun.ServiceDefinition
	protocol    *wstun.ServerProtocol
	listeners   map[string]net.Listener
	httpServer  *http.Server
	logger      *slog.Logger
	activeConns atomic.Int32
	shutdown    chan struct{}
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewServer creates a new Server instance.
func NewServer(config *wstun.ServerOptions, logger *slog.Logger) (*Server, error) {
	// Load services
	services, err := wstun.LoadAllServices(config.ServicesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load services: %w", err)
	}

	if len(services) == 0 {
		return nil, fmt.Errorf("no services loaded from %s", config.ServicesDir)
	}

	logger.Info("services loaded", "count", len(services), "names", getServiceNames(services))

	ctx, cancel := context.WithCancel(context.Background())

	server := &Server{
		config:    config,
		services:  services,
		listeners: make(map[string]net.Listener),
		logger:    logger,
		shutdown:  make(chan struct{}),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Create authentication function
	authFunc := func(serviceName, serviceToken string) bool {
		service, ok := services[serviceName]
		if !ok {
			return false
		}
		return service.ServiceToken == serviceToken
	}

	// Create ServerProtocol
	server.protocol = wstun.NewServerProtocol(authFunc)
	server.protocol.Logger = logger
	server.protocol.DataHandler = server.handleDataConnection
	server.protocol.RetryAttempts = config.RetryAttempts
	server.protocol.RetryInterval = config.RetryInterval
	server.protocol.RetryMaxInterval = config.RetryMaxInterval
	server.protocol.ControlPingInterval = config.ControlPingInterval
	server.protocol.WriteWait = config.WriteWait

	return server, nil
}

// Start starts the server.
func (s *Server) Start() error {
	// Start TCP listeners for each service
	for serviceName, service := range s.services {
		listener, err := net.Listen("tcp", service.ListenAddr)
		if err != nil {
			return fmt.Errorf("failed to start listener for service %s on %s: %w", serviceName, service.ListenAddr, err)
		}
		s.listeners[serviceName] = listener
		s.logger.Info("service listening", "service", serviceName, "addr", service.ListenAddr)

		// Start accept loop for this service
		s.wg.Add(1)
		go s.acceptLoop(serviceName, listener)
	}

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.protocol.HandleWebSocket)

	s.httpServer = &http.Server{
		Addr:    s.config.ListenAddr,
		Handler: mux,
	}

	// Start HTTP/HTTPS server
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		var err error
		if s.config.TLSCert != "" && s.config.TLSKey != "" {
			// Configure TLS
			tlsConfig := &tls.Config{
				MinVersion: tls.VersionTLS12,
			}
			s.httpServer.TLSConfig = tlsConfig
			s.logger.Info("websocket server starting with TLS", "addr", s.config.ListenAddr)
			err = s.httpServer.ListenAndServeTLS(s.config.TLSCert, s.config.TLSKey)
		} else {
			s.logger.Info("websocket server starting", "addr", s.config.ListenAddr)
			err = s.httpServer.ListenAndServe()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("http server error", "error", err)
		}
	}()

	s.logger.Info("server started successfully")
	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() error {
	close(s.shutdown)
	// Abort any in-flight connection requests waiting for a client HANDLE.
	s.cancel()

	// Close all TCP listeners
	s.logger.Info("closing TCP listeners")
	for serviceName, listener := range s.listeners {
		if err := listener.Close(); err != nil {
			s.logger.Error("error closing listener", "service", serviceName, "error", err)
		}
	}

	// Shutdown HTTP server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("http server shutdown error", "error", err)
	}

	// Wait for all goroutines with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("all connections closed cleanly")
	case <-time.After(shutdownTimeout):
		s.logger.Warn("shutdown timeout, forcing exit")
	}

	return nil
}

// acceptLoop accepts incoming TCP connections for a service.
func (s *Server) acceptLoop(serviceName string, listener net.Listener) {
	defer s.wg.Done()

	log := s.logger.With("service", serviceName)

	for {
		select {
		case <-s.shutdown:
			return
		default:
		}

		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-s.shutdown:
				return
			default:
				log.Error("accept error", "error", err)
				continue
			}
		}

		// Check max connections
		if s.config.MaxConnections > 0 {
			current := s.activeConns.Load()
			if current >= int32(s.config.MaxConnections) {
				log.Warn("max connections reached, rejecting connection", "max", s.config.MaxConnections)
				conn.Close()
				continue
			}
		}

		// Handle each connection concurrently: RequestConnection blocks until the
		// client claims the connection (up to PendingTimeout, plus retries), so a
		// serialized accept loop would stall all other incoming connections and
		// the listeners behind it.
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()

			// RequestConnectionContext closes conn itself on failure.
			connID, err := s.protocol.RequestConnectionContext(s.ctx, serviceName, conn)
			if err != nil {
				if s.ctx.Err() == nil {
					log.Error("failed to request connection", "error", err)
				}
				return
			}

			log.Info("connection requested", "conn_id", connID)
		}()
	}
}

// handleDataConnection handles bidirectional data transfer.
func (s *Server) handleDataConnection(incoming io.ReadWriteCloser, outgoing *websocket.Conn) {
	s.activeConns.Add(1)
	defer s.activeConns.Add(-1)

	var wg sync.WaitGroup
	var closeIncomingOnce sync.Once
	var closeOutgoingOnce sync.Once

	closeIncoming := func() {
		closeIncomingOnce.Do(func() {
			incoming.Close()
		})
	}

	closeOutgoing := func() {
		closeOutgoingOnce.Do(func() {
			// Best-effort close frame so the tunnel client sees a clean shutdown.
			outgoing.WriteControl(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
				time.Now().Add(s.config.WriteWait))
			outgoing.Close()
		})
	}

	wg.Add(2)

	// Incoming -> Outgoing (TCP -> WebSocket)
	go func() {
		defer wg.Done()
		// When either direction ends, close both sides so the other goroutine
		// unblocks instead of leaking on a silent peer (e.g. an SSE consumer
		// that never sends data would otherwise keep incoming.Read blocked
		// forever after the tunnel client disappeared).
		defer closeIncoming()
		defer closeOutgoing()
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("panic in incoming->outgoing transfer", "panic", r)
			}
		}()

		buf := make([]byte, 32*1024)
		for {
			n, err := incoming.Read(buf)
			// Process n bytes BEFORE checking err: Read() can return n>0 and io.EOF
			// simultaneously on the final chunk; discarding those bytes causes data loss.
			if n > 0 {
				// Per-write deadline: only a peer that stops draining for longer
				// than WriteWait is dropped; an idle-but-alive connection is not
				// affected because the deadline is re-armed on every write.
				if writeErr := outgoing.SetWriteDeadline(time.Now().Add(s.config.WriteWait)); writeErr != nil {
					s.logger.Debug("failed to set write deadline", "error", writeErr)
					return
				}
				if writeErr := outgoing.WriteMessage(websocket.BinaryMessage, buf[:n]); writeErr != nil {
					s.logger.Debug("outgoing write error", "error", writeErr)
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					s.logger.Debug("incoming read error", "error", err)
				}
				return
			}
		}
	}()

	// Outgoing -> Incoming (WebSocket -> TCP)
	go func() {
		defer wg.Done()
		defer closeIncoming()
		defer closeOutgoing()
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("panic in outgoing->incoming transfer", "panic", r)
			}
		}()

		for {
			messageType, data, err := outgoing.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					s.logger.Debug("outgoing read error", "error", err)
				}
				return
			}
			if messageType != websocket.BinaryMessage {
				s.logger.Debug("received non-binary message, ignoring")
				continue
			}

			if len(data) == 0 {
				continue
			}

			// Bound writes to the external client so a stuck consumer is
			// dropped instead of leaking the transfer goroutines forever.
			if nc, ok := incoming.(net.Conn); ok {
				if err := nc.SetWriteDeadline(time.Now().Add(s.config.WriteWait)); err != nil {
					s.logger.Debug("failed to set write deadline on incoming connection", "error", err)
					return
				}
			}
			if _, err := incoming.Write(data); err != nil {
				s.logger.Debug("incoming write error", "error", err)
				return
			}
		}
	}()

	wg.Wait()
}

// getServiceNames returns a list of service names from a map.
func getServiceNames(services map[string]*wstun.ServiceDefinition) []string {
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	return names
}
