package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"cnb.cool/yankeguo/wstun"
	"github.com/gorilla/websocket"
)

// Server encapsulates the server state and operations.
type Server struct {
	config      *wstun.ServerOptions
	services    map[string]*wstun.ServiceDefinition
	protocol    *wstun.ServerProtocol
	listeners   map[string]net.Listener
	httpServer  *http.Server
	logger      *log.Logger
	activeConns int32
	shutdown    chan struct{}
	wg          sync.WaitGroup
}

func main() {
	// Define flags
	configPath := flag.String("c", "", "path to configuration file (required)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -c <config-file>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "WebSocket tunnel server\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -c /etc/wstun/server.yaml\n", os.Args[0])
	}
	flag.Parse()

	// Validate required flag
	if *configPath == "" {
		fmt.Fprintf(os.Stderr, "Error: configuration file is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Load configuration
	config, err := wstun.LoadServerConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Configure logger based on log level
	logPrefix := "[wstun-server] "
	logger := log.New(os.Stdout, logPrefix, log.LstdFlags)
	logger.Printf("Starting server with config from: %s", *configPath)
	logger.Printf("Listen address: %s", config.ListenAddr)
	logger.Printf("Log level: %s", config.LogLevel)

	if config.TLSCert != "" && config.TLSKey != "" {
		logger.Printf("TLS enabled (cert: %s, key: %s)", config.TLSCert, config.TLSKey)
	}

	if config.MaxConnections > 0 {
		logger.Printf("Max connections: %d", config.MaxConnections)
	} else {
		logger.Printf("Max connections: unlimited")
	}

	if config.RetryAttempts > 0 {
		logger.Printf("Retry attempts: %d", config.RetryAttempts)
		logger.Printf("Retry interval: %v", config.RetryInterval)
		logger.Printf("Retry max interval: %v", config.RetryMaxInterval)
	}

	// Create and start server
	server, err := NewServer(config, logger)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal
	sig := <-sigChan
	logger.Printf("Received signal %v, initiating graceful shutdown...", sig)

	// Graceful shutdown
	if err := server.Shutdown(); err != nil {
		log.Fatalf("Shutdown error: %v", err)
	}

	logger.Printf("Shutdown complete")
}

// NewServer creates a new Server instance.
func NewServer(config *wstun.ServerOptions, logger *log.Logger) (*Server, error) {
	// Load services
	services, err := wstun.LoadAllServices(config.ServicesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load services: %w", err)
	}

	if len(services) == 0 {
		return nil, fmt.Errorf("no services loaded from %s", config.ServicesDir)
	}

	logger.Printf("Loaded %d service(s): %v", len(services), getServiceNames(services))

	server := &Server{
		config:    config,
		services:  services,
		listeners: make(map[string]net.Listener),
		logger:    logger,
		shutdown:  make(chan struct{}),
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
		s.logger.Printf("Service %s listening on %s", serviceName, service.ListenAddr)

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
			s.logger.Printf("WebSocket server starting with TLS on %s", s.config.ListenAddr)
			err = s.httpServer.ListenAndServeTLS(s.config.TLSCert, s.config.TLSKey)
		} else {
			s.logger.Printf("WebSocket server starting on %s", s.config.ListenAddr)
			err = s.httpServer.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			s.logger.Printf("HTTP server error: %v", err)
		}
	}()

	s.logger.Printf("Server started successfully")
	return nil
}

// acceptLoop accepts incoming TCP connections for a service.
func (s *Server) acceptLoop(serviceName string, listener net.Listener) {
	defer s.wg.Done()

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
				// Shutdown in progress, expected error
				return
			default:
				s.logger.Printf("Accept error for service %s: %v", serviceName, err)
				continue
			}
		}

		// Check max connections
		if s.config.MaxConnections > 0 {
			current := atomic.LoadInt32(&s.activeConns)
			if current >= int32(s.config.MaxConnections) {
				s.logger.Printf("Max connections (%d) reached, rejecting connection", s.config.MaxConnections)
				conn.Close()
				continue
			}
		}

		// Request connection via protocol
		connID, err := s.protocol.RequestConnection(serviceName, conn)
		if err != nil {
			s.logger.Printf("Failed to request connection for service %s: %v", serviceName, err)
			conn.Close()
			continue
		}

		if shouldLog(s.config.LogLevel, "info") {
			s.logger.Printf("Connection requested for service %s: %s", serviceName, connID)
		}
	}
}

// handleDataConnection handles bidirectional data transfer.
func (s *Server) handleDataConnection(incoming io.ReadWriteCloser, outgoing *websocket.Conn) {
	atomic.AddInt32(&s.activeConns, 1)
	defer atomic.AddInt32(&s.activeConns, -1)
	defer incoming.Close()
	defer outgoing.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	// Incoming -> Outgoing (TCP -> WebSocket)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				s.logger.Printf("Panic in incoming->outgoing transfer: %v", r)
			}
		}()

		buf := make([]byte, 32*1024)
		for {
			n, err := incoming.Read(buf)
			if err != nil {
				if err != io.EOF && shouldLog(s.config.LogLevel, "debug") {
					s.logger.Printf("Incoming read error: %v", err)
				}
				return
			}

			// Handle zero-length reads
			if n <= 0 {
				continue
			}

			if err := outgoing.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				if shouldLog(s.config.LogLevel, "debug") {
					s.logger.Printf("Outgoing write error: %v", err)
				}
				return
			}
		}
	}()

	// Outgoing -> Incoming (WebSocket -> TCP)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				s.logger.Printf("Panic in outgoing->incoming transfer: %v", r)
			}
		}()

		for {
			messageType, data, err := outgoing.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) && shouldLog(s.config.LogLevel, "debug") {
					s.logger.Printf("Outgoing read error: %v", err)
				}
				return
			}
			if messageType != websocket.BinaryMessage {
				if shouldLog(s.config.LogLevel, "debug") {
					s.logger.Printf("Received non-binary message, ignoring")
				}
				continue
			}

			// Handle zero-length data
			if len(data) == 0 {
				continue
			}

			if _, err := incoming.Write(data); err != nil {
				if shouldLog(s.config.LogLevel, "debug") {
					s.logger.Printf("Incoming write error: %v", err)
				}
				return
			}
		}
	}()

	wg.Wait()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown() error {
	close(s.shutdown)

	// Close all TCP listeners
	s.logger.Printf("Closing TCP listeners...")
	for serviceName, listener := range s.listeners {
		if err := listener.Close(); err != nil {
			s.logger.Printf("Error closing listener for service %s: %v", serviceName, err)
		}
	}

	// Shutdown HTTP server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Printf("HTTP server shutdown error: %v", err)
	}

	// Wait for all goroutines with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Printf("All connections closed cleanly")
	case <-time.After(10 * time.Second):
		s.logger.Printf("Shutdown timeout, forcing exit")
	}

	return nil
}

// Helper functions

func getServiceNames(services map[string]*wstun.ServiceDefinition) []string {
	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	return names
}

func shouldLog(logLevel, requiredLevel string) bool {
	levels := map[string]int{
		"debug": 0,
		"info":  1,
		"warn":  2,
		"error": 3,
	}

	configLevel, ok := levels[logLevel]
	if !ok {
		configLevel = 1 // default to info
	}

	required, ok := levels[requiredLevel]
	if !ok {
		required = 1
	}

	return configLevel <= required
}
