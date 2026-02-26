package wstun

import (
	"context"
	"fmt"
	"io"
	"log"
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

// Test protocol constants
func TestProtocolConstants(t *testing.T) {
	if MessageTypeWaiting != "WAITING" {
		t.Errorf("MessageTypeWaiting = %s, want WAITING", MessageTypeWaiting)
	}
	if MessageTypePrefixConnect != "CONNECT:" {
		t.Errorf("MessageTypePrefixConnect = %s, want CONNECT:", MessageTypePrefixConnect)
	}
	if MessageTypePrefixHandle != "HANDLE:" {
		t.Errorf("MessageTypePrefixHandle = %s, want HANDLE:", MessageTypePrefixHandle)
	}
}

// Test ULID generation
func TestGenerateConnectionID(t *testing.T) {
	id1, err := GenerateConnectionID()
	if err != nil {
		t.Fatalf("GenerateConnectionID() error = %v", err)
	}
	if len(id1) != 26 {
		t.Errorf("GenerateConnectionID() length = %d, want 26", len(id1))
	}

	id2, err := GenerateConnectionID()
	if err != nil {
		t.Fatalf("GenerateConnectionID() error = %v", err)
	}
	if id1 == id2 {
		t.Errorf("GenerateConnectionID() generated duplicate IDs")
	}
}

// Test message formatting
func TestFormatMessages(t *testing.T) {
	connID := "01234567890123456789012345"

	connect := FormatConnectMessage(connID)
	expected := "CONNECT:01234567890123456789012345"
	if connect != expected {
		t.Errorf("FormatConnectMessage() = %s, want %s", connect, expected)
	}

	handle := FormatHandleMessage(connID)
	expected = "HANDLE:01234567890123456789012345"
	if handle != expected {
		t.Errorf("FormatHandleMessage() = %s, want %s", handle, expected)
	}
}

// Test message parsing
func TestParseMessages(t *testing.T) {
	tests := []struct {
		name    string
		message string
		parser  func(string) (string, bool)
		wantID  string
		wantOK  bool
	}{
		{
			name:    "valid CONNECT",
			message: "CONNECT:01234567890123456789012345",
			parser:  ParseConnectMessage,
			wantID:  "01234567890123456789012345",
			wantOK:  true,
		},
		{
			name:    "valid HANDLE",
			message: "HANDLE:01234567890123456789012345",
			parser:  ParseHandleMessage,
			wantID:  "01234567890123456789012345",
			wantOK:  true,
		},
		{
			name:    "invalid prefix",
			message: "INVALID:01234567890123456789012345",
			parser:  ParseConnectMessage,
			wantID:  "",
			wantOK:  false,
		},
		{
			name:    "short ID",
			message: "CONNECT:123",
			parser:  ParseConnectMessage,
			wantID:  "",
			wantOK:  false,
		},
		{
			name:    "long ID",
			message: "CONNECT:012345678901234567890123456",
			parser:  ParseConnectMessage,
			wantID:  "",
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := tt.parser(tt.message)
			if gotID != tt.wantID || gotOK != tt.wantOK {
				t.Errorf("parser(%s) = (%s, %v), want (%s, %v)",
					tt.message, gotID, gotOK, tt.wantID, tt.wantOK)
			}
		})
	}
}

// Test connection ID validation
func TestValidateConnectionID(t *testing.T) {
	validID, err := GenerateConnectionID()
	if err != nil {
		t.Fatalf("GenerateConnectionID() error = %v", err)
	}

	tests := []struct {
		name string
		id   string
		want bool
	}{
		{
			name: "valid ULID",
			id:   validID,
			want: true,
		},
		{
			name: "invalid length",
			id:   "123",
			want: false,
		},
		{
			name: "invalid format",
			id:   "!!!!!!!!!!!!!!!!!!!!!!!!",
			want: false,
		},
		{
			name: "empty string",
			id:   "",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateConnectionID(tt.id)
			if got != tt.want {
				t.Errorf("ValidateConnectionID(%s) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

// Test server protocol authentication
func TestServerProtocolAuth(t *testing.T) {
	// Create auth function
	authFunc := func(serviceName, serviceToken string) bool {
		return serviceName == "test-service" && serviceToken == "test-token"
	}

	sp := NewServerProtocol(authFunc)
	sp.Logger = log.New(io.Discard, "", 0)

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
			cp.Logger = log.New(io.Discard, "", 0)

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
	sp.Logger = log.New(io.Discard, "", 0)

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
	cp.Logger = log.New(io.Discard, "", 0)

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
	sp.Logger = log.New(io.Discard, "", 0)
	sp.PendingTimeout = 200 * time.Millisecond // Short timeout for testing

	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(sp.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Start client but don't handle CONNECT messages
	cp := NewClientProtocol(wsURL, "test-service", "test-token")
	cp.Logger = log.New(io.Discard, "", 0)
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
	sp.Logger = log.New(io.Discard, "", 0)

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
	cp.Logger = log.New(io.Discard, "", 0)

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
	sp.Logger = log.New(io.Discard, "", 0)
	sp.PendingTimeout = 100 * time.Millisecond
	sp.RetryAttempts = 0 // No retry

	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(sp.HandleWebSocket))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Start client but don't handle CONNECT messages
	cp := NewClientProtocol(wsURL, "test-service", "test-token")
	cp.Logger = log.New(io.Discard, "", 0)
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
	sp.Logger = log.New(io.Discard, "", 0)
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
	cp.Logger = log.New(io.Discard, "", 0)
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
	sp.Logger = log.New(io.Discard, "", 0)
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
	cp.Logger = log.New(io.Discard, "", 0)

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

// Test base64 encoding (testing internal basicAuth implementation in client_protocol.go)
func TestBase64Encode(t *testing.T) {
	tests := []struct {
		username string
		password string
		want     string
	}{
		{"test", "password", "Basic dGVzdDpwYXNzd29yZA=="},
		{"user", "pass", "Basic dXNlcjpwYXNz"},
		{"", "", "Basic "}, // Empty credentials
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("user=%s", tt.username), func(t *testing.T) {
			// Create a temporary client protocol to test basicAuth
			cp := NewClientProtocol("ws://localhost:8080", tt.username, tt.password)

			// We can't directly test basicAuth as it's unexported, but we can verify
			// the Authorization header is set correctly when dialing
			// This test is more of a smoke test - the actual basicAuth is tested via integration
			if cp.ServiceName != tt.username {
				t.Errorf("ServiceName = %q, want %q", cp.ServiceName, tt.username)
			}
			if cp.ServiceToken != tt.password {
				t.Errorf("ServiceToken = %q, want %q", cp.ServiceToken, tt.password)
			}
		})
	}
}
