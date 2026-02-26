package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"cnb.cool/yankeguo/wstun"
	"github.com/gorilla/websocket"
)

const (
	maxReconnectDelay = 60 * time.Second
	jitterPercent     = 0.3
)

// Client encapsulates the client state and operations.
type Client struct {
	config     *wstun.ClientOptions
	protocol   *wstun.ClientProtocol
	protocolMu sync.RWMutex // Protects protocol access
	logger     *log.Logger
	shutdown   chan struct{}
}

func main() {
	// Define flags
	configPath := flag.String("c", "", "path to configuration file (required)")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -c <config-file>\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "WebSocket tunnel client\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -c /etc/wstun/client.yaml\n", os.Args[0])
	}
	flag.Parse()

	// Validate required flag
	if *configPath == "" {
		fmt.Fprintf(os.Stderr, "Error: configuration file is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Load configuration
	config, err := wstun.LoadClientConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Configure logger based on log level
	logPrefix := fmt.Sprintf("[wstun-client] ")
	logger := log.New(os.Stdout, logPrefix, log.LstdFlags)
	logger.Printf("Starting client with config from: %s", *configPath)
	logger.Printf("Server URL: %s", config.ServerURL)
	logger.Printf("Service name: %s", config.ServiceName)
	logger.Printf("Target address: %s", config.TargetAddr)
	logger.Printf("Log level: %s", config.LogLevel)

	if config.TLSInsecure {
		logger.Printf("TLS verification: disabled (insecure)")
	} else {
		logger.Printf("TLS verification: enabled")
	}

	logger.Printf("Reconnect interval: %v", config.ReconnectInterval)

	if config.MaxReconnectAttempts > 0 {
		logger.Printf("Max reconnect attempts: %d", config.MaxReconnectAttempts)
	} else {
		logger.Printf("Max reconnect attempts: unlimited")
	}

	// Create and start client
	client := NewClient(config, logger)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run client with reconnection logic in a goroutine
	clientErr := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				clientErr <- fmt.Errorf("panic in client.Run(): %v", r)
			}
		}()
		clientErr <- client.Run()
	}()

	// Wait for either shutdown signal or fatal error
	select {
	case sig := <-sigChan:
		logger.Printf("Received signal %v, initiating graceful shutdown...", sig)
		if err := client.Shutdown(); err != nil {
			logger.Printf("Shutdown error: %v", err)
		}
	case err := <-clientErr:
		if err != nil {
			log.Fatalf("Client error: %v", err)
		}
	}

	logger.Printf("Shutdown complete")
}

// NewClient creates a new Client instance.
func NewClient(config *wstun.ClientOptions, logger *log.Logger) *Client {
	return &Client{
		config:   config,
		logger:   logger,
		shutdown: make(chan struct{}),
	}
}

func (c *Client) Run() error {
	attempt := 0
	delay := c.config.ReconnectInterval

	for {
		select {
		case <-c.shutdown:
			return nil
		default:
		}

		c.protocolMu.Lock()
		c.protocol = wstun.NewClientProtocol(
			c.config.ServerURL,
			c.config.ServiceName,
			c.config.ServiceToken,
		)
		c.protocol.Logger = c.logger
		c.protocol.PingInterval = c.config.PingInterval
		c.protocol.PongWait = c.config.PongTimeout
		c.protocol.WriteWait = c.config.WriteTimeout

		if c.config.TLSInsecure {
			c.protocol.Dialer = &websocket.Dialer{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			}
		}

		c.protocol.ConnectHandler = c.handleConnect
		c.protocolMu.Unlock()

		if shouldLog(c.config.LogLevel, "info") {
			if attempt == 0 {
				c.logger.Printf("Connecting to server...")
			} else {
				c.logger.Printf("Reconnecting to server (attempt %d)...", attempt+1)
			}
		}

		err := c.protocol.Start()

		select {
		case <-c.shutdown:
			return nil
		default:
		}

		if err != nil {
			c.logger.Printf("Connection error: %v", err)
		} else {
			c.logger.Printf("Connection closed")
		}

		attempt++

		if c.config.MaxReconnectAttempts > 0 && attempt >= c.config.MaxReconnectAttempts {
			return fmt.Errorf("max reconnection attempts (%d) reached", c.config.MaxReconnectAttempts)
		}

		actualDelay := addJitter(delay, jitterPercent)
		if shouldLog(c.config.LogLevel, "info") {
			c.logger.Printf("Reconnecting in %v...", actualDelay)
		}

		select {
		case <-c.shutdown:
			return nil
		case <-time.After(actualDelay):
		}

		newDelay := delay * 2
		if newDelay < delay || newDelay > maxReconnectDelay {
			delay = maxReconnectDelay
		} else {
			delay = newDelay
		}
	}
}

func addJitter(d time.Duration, jitter float64) time.Duration {
	if d <= 0 || jitter <= 0 {
		return d
	}
	jitterRange := float64(d) * jitter
	jitterValue := rand.Float64()*jitterRange*2 - jitterRange
	return time.Duration(float64(d) + jitterValue)
}

// handleConnect handles CONNECT requests by establishing TCP connection to target.
func (c *Client) handleConnect(ctx context.Context, connID string) (io.ReadWriteCloser, error) {
	if shouldLog(c.config.LogLevel, "debug") {
		c.logger.Printf("Handling connection request: %s", connID)
	}

	// Dial target with context timeout
	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", c.config.TargetAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to target %s: %w", c.config.TargetAddr, err)
	}

	if shouldLog(c.config.LogLevel, "info") {
		c.logger.Printf("Connected to target for connection: %s", connID)
	}

	return conn, nil
}

func (c *Client) Shutdown() error {
	close(c.shutdown)

	c.protocolMu.RLock()
	protocol := c.protocol
	c.protocolMu.RUnlock()

	if protocol != nil {
		if err := protocol.Stop(); err != nil {
			c.logger.Printf("Protocol stop error: %v", err)
		}

		c.logger.Printf("Waiting for active connections to close...")
		if protocol.WaitForActiveConnections(5 * time.Second) {
			c.logger.Printf("All connections closed")
		} else {
			c.logger.Printf("Timeout waiting for connections, some may be forcefully closed")
		}
	}

	return nil
}

// Helper function

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
