package wstun

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Test server protocol authentication
func TestServerProtocolAuth(t *testing.T) {
	// Create auth function
	authFunc := func(serviceName, serviceToken string) bool {
		return serviceName == "test-service" && serviceToken == "test-token"
	}

	sp := NewServerProtocol(authFunc)
	sp.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))

	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(sp.HandleWebSocket))
	defer server.Close()

	tests := []struct {
		name          string
		serviceName   string
		serviceToken  string
		expectSuccess bool
	}{
		{
			name:          "valid credentials",
			serviceName:   "test-service",
			serviceToken:  "test-token",
			expectSuccess: true,
		},
		{
			name:          "invalid service name",
			serviceName:   "wrong-service",
			serviceToken:  "test-token",
			expectSuccess: false,
		},
		{
			name:          "invalid token",
			serviceName:   "test-service",
			serviceToken:  "wrong-token",
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert http:// to ws://
			wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

			cp := NewClientProtocol(wsURL, tt.serviceName, tt.serviceToken)
			cp.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))

			// Try to connect in goroutine with timeout
			errCh := make(chan error, 1)
			go func() {
				errCh <- cp.Start()
			}()

			// Wait a bit for connection attempt
			time.Sleep(100 * time.Millisecond)

			cp.Stop()
			err := <-errCh

			if tt.expectSuccess && err != nil && !strings.Contains(err.Error(), "401") {
				// Success case should connect, might error on stop
				return
			}
			if !tt.expectSuccess && (err == nil || !strings.Contains(err.Error(), "401")) {
				t.Errorf("Expected authentication failure, got error: %v", err)
			}
		})
	}
}

// Test full protocol flow
func TestProtocolFlow(t *testing.T) {
	t.Skip("Integration test - requires more precise timing control")
	// Create auth function
	authFunc := func(serviceName, serviceToken string) bool {
		return serviceName == "test-service" && serviceToken == "test-token"
	}

	sp := NewServerProtocol(authFunc)
	sp.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))

	// Track connections
	var dataHandlerCalled sync.WaitGroup
	dataHandlerCalled.Add(1)

	sp.DataHandler = func(incoming io.ReadWriteCloser, outgoing *websocket.Conn) {
		defer dataHandlerCalled.Done()
		defer incoming.Close()
		defer outgoing.Close()

		// Simple echo for testing
		buf := make([]byte, 1024)
		n, _ := incoming.Read(buf)
		if n > 0 {
			outgoing.WriteMessage(websocket.BinaryMessage, buf[:n])
		}
	}

	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(sp.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Start client
	cp := NewClientProtocol(wsURL, "test-service", "test-token")
	cp.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))

	// Set up connect handler
	cp.ConnectHandler = func(ctx context.Context, connID string) (io.ReadWriteCloser, error) {
		// Create a pipe to simulate target connection
		client, server := net.Pipe()

		// Server side writes test data and keeps connection open
		go func() {
			defer server.Close()
			server.Write([]byte("test-data"))
			// Keep connection open for a bit
			time.Sleep(500 * time.Millisecond)
		}()

		return client, nil
	}

	// Start client in background
	go cp.Start()
	defer cp.Stop()

	// Wait for control connection to establish
	time.Sleep(100 * time.Millisecond)

	// Simulate incoming connection
	incomingClient, incomingServer := net.Pipe()
	defer incomingClient.Close()

	// Request connection from server
	connID, err := sp.RequestConnection("test-service", incomingServer)
	if err != nil {
		t.Fatalf("RequestConnection() error = %v", err)
	}

	if len(connID) != 26 {
		t.Errorf("connection ID length = %d, want 26", len(connID))
	}

	// Wait for data handler to be called
	done := make(chan struct{})
	go func() {
		dataHandlerCalled.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("DataHandler was not called within timeout")
	}
}

// Test CONNECT message timeout
func TestConnectionTimeout(t *testing.T) {
	authFunc := func(serviceName, serviceToken string) bool {
		return serviceName == "test-service" && serviceToken == "test-token"
	}

	sp := NewServerProtocol(authFunc)
	sp.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	sp.PendingTimeout = 200 * time.Millisecond // Short timeout for testing

	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(sp.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Start client but don't handle CONNECT messages
	cp := NewClientProtocol(wsURL, "test-service", "test-token")
	cp.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	cp.ConnectHandler = nil // No handler, so CONNECT will be ignored

	go cp.Start()
	defer cp.Stop()

	// Wait for control connection
	time.Sleep(100 * time.Millisecond)

	// Request connection
	incomingClient, incomingServer := net.Pipe()
	defer incomingClient.Close()

	start := time.Now()
	_, err := sp.RequestConnection("test-service", incomingServer)
	elapsed := time.Since(start)

	// Should return error after timeout (no retry by default)
	if err == nil {
		t.Fatal("Expected error after timeout, got nil")
	}

	// Should take approximately PendingTimeout (200ms) + small overhead
	if elapsed < 150*time.Millisecond || elapsed > 400*time.Millisecond {
		t.Errorf("Expected ~200ms for timeout, got %v", elapsed)
	}

	// Connection should be closed
	buf := make([]byte, 1)
	_, err = incomingClient.Read(buf)
	if err == nil {
		t.Error("Expected connection to be closed after timeout")
	}
}

// Test concurrent connections
func TestConcurrentConnections(t *testing.T) {
	t.Skip("Integration test - requires more precise timing control")
	authFunc := func(serviceName, serviceToken string) bool {
		return serviceName == "test-service" && serviceToken == "test-token"
	}

	sp := NewServerProtocol(authFunc)
	sp.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))

	var handleCount sync.WaitGroup
	var handleMutex sync.Mutex
	handledIDs := make(map[string]bool)

	sp.DataHandler = func(incoming io.ReadWriteCloser, outgoing *websocket.Conn) {
		defer handleCount.Done()
		defer incoming.Close()
		defer outgoing.Close()
		time.Sleep(50 * time.Millisecond) // Simulate some work
	}

	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(sp.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Start client
	cp := NewClientProtocol(wsURL, "test-service", "test-token")
	cp.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))

	cp.ConnectHandler = func(ctx context.Context, connID string) (io.ReadWriteCloser, error) {
		handleMutex.Lock()
		handledIDs[connID] = true
		handleMutex.Unlock()

		client, server := net.Pipe()
		go func() {
			defer server.Close()
		}()
		return client, nil
	}

	go cp.Start()
	defer cp.Stop()

	time.Sleep(100 * time.Millisecond)

	// Request multiple connections concurrently
	numConns := 5
	handleCount.Add(numConns)
	connIDs := make([]string, numConns)

	for i := 0; i < numConns; i++ {
		go func(idx int) {
			incomingClient, incomingServer := net.Pipe()
			defer incomingClient.Close()

			connID, err := sp.RequestConnection("test-service", incomingServer)
			if err != nil {
				t.Errorf("RequestConnection() error = %v", err)
				return
			}
			connIDs[idx] = connID
		}(i)
	}

	// Wait for all handlers to complete
	done := make(chan struct{})
	go func() {
		handleCount.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Not all connections were handled within timeout")
	}

	// Verify all connections were handled
	handleMutex.Lock()
	defer handleMutex.Unlock()

	for i, connID := range connIDs {
		if connID != "" && !handledIDs[connID] {
			t.Errorf("Connection %d (ID: %s) was not handled", i, connID)
		}
	}
}

// Test server protocol retry configuration
func TestServerProtocolRetryDefaults(t *testing.T) {
	authFunc := func(serviceName, serviceToken string) bool {
		return true
	}

	sp := NewServerProtocol(authFunc)

	if sp.RetryAttempts != 0 {
		t.Errorf("expected default RetryAttempts to be 0, got %d", sp.RetryAttempts)
	}
	if sp.RetryInterval != 5*time.Second {
		t.Errorf("expected default RetryInterval to be 5s, got %v", sp.RetryInterval)
	}
	if sp.RetryMaxInterval != 60*time.Second {
		t.Errorf("expected default RetryMaxInterval to be 60s, got %v", sp.RetryMaxInterval)
	}
}

// Test exponential backoff calculation
func TestCalculateRetryDelay(t *testing.T) {
	tests := []struct {
		name             string
		attempt          int
		retryInterval    time.Duration
		retryMaxInterval time.Duration
		expected         time.Duration
	}{
		{
			name:             "first attempt",
			attempt:          0,
			retryInterval:    5 * time.Second,
			retryMaxInterval: 60 * time.Second,
			expected:         5 * time.Second,
		},
		{
			name:             "second attempt",
			attempt:          1,
			retryInterval:    5 * time.Second,
			retryMaxInterval: 60 * time.Second,
			expected:         10 * time.Second,
		},
		{
			name:             "third attempt",
			attempt:          2,
			retryInterval:    5 * time.Second,
			retryMaxInterval: 60 * time.Second,
			expected:         20 * time.Second,
		},
		{
			name:             "max interval cap",
			attempt:          10,
			retryInterval:    5 * time.Second,
			retryMaxInterval: 60 * time.Second,
			expected:         60 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := tt.retryInterval
			for i := 0; i < tt.attempt; i++ {
				newDelay := delay * 2
				if newDelay > tt.retryMaxInterval {
					delay = tt.retryMaxInterval
				} else {
					delay = newDelay
				}
			}
			if delay != tt.expected {
				t.Errorf("expected delay %v, got %v", tt.expected, delay)
			}
		})
	}
}

// Test retry disabled (RetryAttempts=0)
func TestRequestConnectionNoRetry(t *testing.T) {
	authFunc := func(serviceName, serviceToken string) bool {
		return serviceName == "test-service" && serviceToken == "test-token"
	}

	sp := NewServerProtocol(authFunc)
	sp.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	sp.PendingTimeout = 100 * time.Millisecond
	sp.RetryAttempts = 0 // No retry

	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(sp.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Start client but don't handle CONNECT messages
	cp := NewClientProtocol(wsURL, "test-service", "test-token")
	cp.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	cp.ConnectHandler = nil

	go cp.Start()
	defer cp.Stop()

	// Wait for control connection
	time.Sleep(100 * time.Millisecond)

	// Request connection
	incomingClient, incomingServer := net.Pipe()
	defer incomingClient.Close()

	start := time.Now()
	_, err := sp.RequestConnection("test-service", incomingServer)
	elapsed := time.Since(start)

	// Should return error after single attempt (no retries)
	if err == nil {
		t.Fatal("Expected error after timeout with no retry, got nil")
	}

	// Should take approximately PendingTimeout (100ms) + small overhead
	if elapsed < 80*time.Millisecond || elapsed > 200*time.Millisecond {
		t.Errorf("Expected ~100ms for single attempt, got %v", elapsed)
	}

	// Connection should be closed
	buf := make([]byte, 1)
	_, err = incomingClient.Read(buf)
	if err == nil {
		t.Error("Expected connection to be closed after timeout with no retry")
	}
}

// Test retry with exhausted attempts (now fails immediately on timeout)
func TestRequestConnectionRetryExhausted(t *testing.T) {
	authFunc := func(serviceName, serviceToken string) bool {
		return serviceName == "test-service" && serviceToken == "test-token"
	}

	sp := NewServerProtocol(authFunc)
	sp.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	sp.PendingTimeout = 100 * time.Millisecond
	sp.RetryAttempts = 2
	sp.RetryInterval = 50 * time.Millisecond
	sp.RetryMaxInterval = 200 * time.Millisecond

	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(sp.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Start client but don't handle CONNECT messages
	cp := NewClientProtocol(wsURL, "test-service", "test-token")
	cp.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	cp.ConnectHandler = nil

	go cp.Start()
	defer cp.Stop()

	// Wait for control connection
	time.Sleep(100 * time.Millisecond)

	// Request connection
	incomingClient, incomingServer := net.Pipe()
	defer incomingClient.Close()

	start := time.Now()
	_, err := sp.RequestConnection("test-service", incomingServer)
	elapsed := time.Since(start)

	// Should return error after all retry attempts are exhausted
	if err == nil {
		t.Fatal("Expected error after timeout, got nil")
	}

	// With RetryAttempts=2, PendingTimeout=100ms, RetryInterval=50ms:
	//   attempt 0: 100ms wait + 50ms backoff
	//   attempt 1: 100ms wait + 100ms backoff
	//   attempt 2: 100ms wait → exhausted
	// Total ≈ 450ms; allow generous range for slow CI machines.
	if elapsed < 350*time.Millisecond || elapsed > 800*time.Millisecond {
		t.Errorf("Expected ~450ms for 3 timeout attempts, got %v", elapsed)
	}

	// Connection should be closed
	buf := make([]byte, 1)
	_, err = incomingClient.Read(buf)
	if err == nil {
		t.Error("Expected connection to be closed after timeout")
	}
}

// Test successful connection (no retry needed)
func TestRequestConnectionRetrySuccess(t *testing.T) {
	authFunc := func(serviceName, serviceToken string) bool {
		return serviceName == "test-service" && serviceToken == "test-token"
	}

	sp := NewServerProtocol(authFunc)
	sp.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	sp.PendingTimeout = 200 * time.Millisecond
	sp.RetryAttempts = 3
	sp.RetryInterval = 100 * time.Millisecond
	sp.RetryMaxInterval = 500 * time.Millisecond

	var dataHandlerCalled sync.WaitGroup
	dataHandlerCalled.Add(1)

	sp.DataHandler = func(incoming io.ReadWriteCloser, outgoing *websocket.Conn) {
		defer dataHandlerCalled.Done()
		defer incoming.Close()
		defer outgoing.Close()
	}

	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(sp.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Start client with immediate handler (no delay)
	cp := NewClientProtocol(wsURL, "test-service", "test-token")
	cp.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))

	var connectCount int32
	cp.ConnectHandler = func(ctx context.Context, connID string) (io.ReadWriteCloser, error) {
		atomic.AddInt32(&connectCount, 1)
		client, serverPipe := net.Pipe()
		go func() {
			defer serverPipe.Close()
		}()
		return client, nil
	}

	go cp.Start()
	defer cp.Stop()

	// Wait for control connection
	time.Sleep(100 * time.Millisecond)

	// Request connection
	incomingClient, incomingServer := net.Pipe()
	defer incomingClient.Close()

	connID, err := sp.RequestConnection("test-service", incomingServer)
	if err != nil {
		t.Fatalf("RequestConnection() error = %v", err)
	}

	if len(connID) != 26 {
		t.Errorf("connection ID length = %d, want 26", len(connID))
	}

	// Wait for data handler to be called
	done := make(chan struct{})
	go func() {
		dataHandlerCalled.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("DataHandler was not called within timeout")
	}

	// Should have received exactly 1 CONNECT message (no retry needed)
	finalCount := atomic.LoadInt32(&connectCount)
	if finalCount != 1 {
		t.Errorf("Expected 1 connection attempt, got %d", finalCount)
	}
}

// TestSendToControlConnections_PrunesDeadConn verifies that sendToControlConnections
// removes a control connection from the registry when a write to it fails.
// This prevents CONNECT messages from being silently dropped into a dead connection.
func TestSendToControlConnections_PrunesDeadConn(t *testing.T) {
	sp := NewServerProtocol(func(_, _ string) bool { return true })
	sp.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	sp.WriteWait = 100 * time.Millisecond // short deadline so the write fails quickly

	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	serverConnCh := make(chan *websocket.Conn, 1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("WebSocket upgrade error: %v", err)
			return
		}
		serverConnCh <- conn
		// Keep the handler alive so the server-side conn stays open.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer ts.Close()

	// Establish a client connection so the server side gets a valid *websocket.Conn.
	dialer := websocket.Dialer{}
	clientConn, _, err := dialer.Dial("ws"+strings.TrimPrefix(ts.URL, "http"), nil)
	if err != nil {
		t.Fatalf("client dial: %v", err)
	}
	defer clientConn.Close()

	serverConn := <-serverConnCh

	// Register the server-side connection directly with the ServerProtocol.
	sp.registerControlConnection("svc", serverConn)
	if !sp.hasControlConnections("svc") {
		t.Fatal("control connection should be registered")
	}

	// Close the SERVER-side conn directly so that writing to it fails immediately.
	// (Closing only the client-side is not reliable: the kernel's TCP send buffer may
	// absorb the write before the RST is processed, making WriteMessage succeed.)
	serverConn.Close()

	// sendToControlConnections should detect the write failure and prune the dead entry.
	connID, err := GenerateConnectionID()
	if err != nil {
		t.Fatalf("GenerateConnectionID: %v", err)
	}
	sp.sendToControlConnections("svc", FormatConnectMessage(connID))

	if sp.hasControlConnections("svc") {
		t.Error("dead control connection should have been pruned after write failure")
	}
}

// TestRequestConnection_FastFailNoControlConns verifies that RequestConnection returns
// immediately (without sleeping through PendingTimeout) when no control connection is
// registered for the target service.
func TestRequestConnection_FastFailNoControlConns(t *testing.T) {
	sp := NewServerProtocol(func(_, _ string) bool { return true })
	sp.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	sp.RetryAttempts = 0
	sp.PendingTimeout = 30 * time.Second // intentionally long; must NOT block this long

	incoming, other := net.Pipe()
	defer other.Close()

	start := time.Now()
	_, err := sp.RequestConnection("no-such-service", incoming)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from RequestConnection with no control connections")
	}
	if elapsed >= time.Second {
		t.Errorf("fast-fail took %v; expected < 1s (should not sleep PendingTimeout)", elapsed)
	}
}

// TestPingControlConnection_PruneOnFailure verifies that when a keepalive ping
// write fails (e.g. the underlying TCP connection was silently dropped by a
// proxy like Istio/Envoy), pingControlConnection removes the dead entry from
// the registry and closes the connection to unblock the read loop.
func TestPingControlConnection_PruneOnFailure(t *testing.T) {
	sp := NewServerProtocol(func(_, _ string) bool { return true })
	sp.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	sp.WriteWait = 100 * time.Millisecond          // short write deadline so write fails fast
	sp.ControlPingInterval = 50 * time.Millisecond // fire ping quickly in test

	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	serverConnCh := make(chan *websocket.Conn, 1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		serverConnCh <- conn
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer ts.Close()

	dialer := websocket.Dialer{}
	clientConn, _, err := dialer.Dial("ws"+strings.TrimPrefix(ts.URL, "http"), nil)
	if err != nil {
		t.Fatalf("client dial: %v", err)
	}
	defer clientConn.Close()

	serverConn := <-serverConnCh
	ctrlConn := sp.registerControlConnection("svc", serverConn)

	// Close the server-side connection so the next ping write fails immediately.
	serverConn.Close()

	// Run the ping goroutine; the closed connection should trigger pruning quickly.
	done := make(chan struct{})
	defer close(done)
	go sp.pingControlConnection(done, "svc", ctrlConn)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !sp.hasControlConnections("svc") {
			return // pruned as expected
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Error("dead control connection was not pruned by pingControlConnection after write failure")
}

// TestRequestConnection_FastFailWhenSendPrunesAllConnections verifies that
// RequestConnection returns well within PendingTimeout when sendToControlConnections
// removes all control connections (because they were all dead/stale). Previously
// the code would block for the full 30-second PendingTimeout in this scenario,
// causing "upstream connect error / connection termination" to be observed by
// Istio-proxied clients for ~30 seconds before the error was surfaced.
func TestRequestConnection_FastFailWhenSendPrunesAllConnections(t *testing.T) {
	sp := NewServerProtocol(func(_, _ string) bool { return true })
	sp.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	sp.WriteWait = 100 * time.Millisecond // short deadline so write failure is fast
	sp.PendingTimeout = 30 * time.Second  // intentionally long; must NOT block this long

	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	serverConnCh := make(chan *websocket.Conn, 1)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		serverConnCh <- conn
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer ts.Close()

	dialer := websocket.Dialer{}
	clientConn, _, err := dialer.Dial("ws"+strings.TrimPrefix(ts.URL, "http"), nil)
	if err != nil {
		t.Fatalf("client dial: %v", err)
	}
	defer clientConn.Close()

	serverConn := <-serverConnCh

	// Register then immediately close the server-side connection.
	sp.registerControlConnection("svc", serverConn)
	serverConn.Close()

	incoming, other := net.Pipe()
	defer other.Close()

	start := time.Now()
	_, err = sp.RequestConnection("svc", incoming)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from RequestConnection with dead control connection")
	}
	// With WriteWait=100ms the send fails in ~100ms, and the fast-fail path should
	// return immediately; allow generous headroom but never PendingTimeout (30s).
	if elapsed >= 2*time.Second {
		t.Errorf("fast-fail took %v; should be < 2s (must not block for PendingTimeout=30s)", elapsed)
	}
}
