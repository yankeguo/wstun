package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"sync"
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
	protocolMu sync.RWMutex
	logger     *slog.Logger
	shutdown   chan struct{}
}

// NewClient creates a new Client instance.
func NewClient(config *wstun.ClientOptions, logger *slog.Logger) *Client {
	return &Client{
		config:   config,
		logger:   logger,
		shutdown: make(chan struct{}),
	}
}

// Run starts the client with reconnection logic.
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

		if attempt == 0 {
			c.logger.Info("connecting to server")
		} else {
			c.logger.Info("reconnecting to server", "attempt", attempt+1)
		}

		err := c.protocol.Start()

		select {
		case <-c.shutdown:
			return nil
		default:
		}

		if err != nil {
			c.logger.Error("connection error", "error", err)
		} else {
			c.logger.Info("connection closed")
		}

		attempt++

		if c.config.MaxReconnectAttempts > 0 && attempt >= c.config.MaxReconnectAttempts {
			return fmt.Errorf("max reconnection attempts (%d) reached", c.config.MaxReconnectAttempts)
		}

		actualDelay := addJitter(delay, jitterPercent)
		c.logger.Info("reconnecting after delay", "delay", actualDelay)

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

// Shutdown gracefully shuts down the client.
func (c *Client) Shutdown() error {
	close(c.shutdown)

	c.protocolMu.RLock()
	protocol := c.protocol
	c.protocolMu.RUnlock()

	if protocol != nil {
		if err := protocol.Stop(); err != nil {
			c.logger.Error("protocol stop error", "error", err)
		}

		c.logger.Info("waiting for active connections to close")
		if protocol.WaitForActiveConnections(5 * time.Second) {
			c.logger.Info("all connections closed")
		} else {
			c.logger.Warn("timeout waiting for connections, some may be forcefully closed")
		}
	}

	return nil
}

// handleConnect handles CONNECT requests by establishing TCP connection to target.
func (c *Client) handleConnect(ctx context.Context, connID string) (io.ReadWriteCloser, error) {
	c.logger.Debug("handling connection request", "conn_id", connID)

	dialer := &net.Dialer{
		Timeout: 10 * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", c.config.TargetAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to target %s: %w", c.config.TargetAddr, err)
	}

	c.logger.Info("connected to target", "conn_id", connID)

	return conn, nil
}

// addJitter adds random jitter to a duration.
func addJitter(d time.Duration, jitter float64) time.Duration {
	if d <= 0 || jitter <= 0 {
		return d
	}
	jitterRange := float64(d) * jitter
	jitterValue := rand.Float64()*jitterRange*2 - jitterRange
	return time.Duration(float64(d) + jitterValue)
}
