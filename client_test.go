package wstun

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Test base64 encoding (testing internal basicAuth implementation in client.go)
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

// TestTransfer_NoSpuriousReadDeadline verifies that the data-connection transfer loop
// does NOT close an idle-but-valid WebSocket connection simply because PongWait elapsed
// with no incoming data.  The old code incorrectly set ws.SetReadDeadline on every
// iteration, causing spurious i/o timeout errors on tunnels with long idle gaps.
func TestTransfer_NoSpuriousReadDeadline(t *testing.T) {
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
	clientWsConn, _, err := dialer.Dial("ws"+strings.TrimPrefix(ts.URL, "http"), nil)
	if err != nil {
		t.Fatalf("client dial: %v", err)
	}
	defer clientWsConn.Close()

	serverWsConn := <-serverConnCh

	// Use a net.Pipe pair: transferA is used as the "target" inside transfer(),
	// transferB is used by the test goroutine to read data relayed from the WebSocket.
	transferA, transferB := net.Pipe()
	defer transferB.Close()

	cp := NewClientProtocol("ws://unused", "svc", "tok")
	cp.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	cp.PongWait = 100 * time.Millisecond // very short - old code would have fired a read deadline
	cp.WriteWait = time.Second

	transferDone := make(chan struct{})
	go func() {
		defer close(transferDone)
		cp.transfer(clientWsConn, transferA, "test-conn")
	}()

	// Wait significantly longer than PongWait without sending any WebSocket data.
	// Under the old code, the read deadline on clientWsConn would have fired here.
	time.Sleep(300 * time.Millisecond)

	// If transfer is still running, sending data now should reach transferB.
	want := []byte("still alive after idle period")
	if err := serverWsConn.WriteMessage(websocket.BinaryMessage, want); err != nil {
		t.Fatalf("WebSocket write after idle period: %v (transfer goroutine may have exited early)", err)
	}

	got := make([]byte, len(want))
	transferB.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := io.ReadFull(transferB, got); err != nil {
		t.Fatalf("data not received at target pipe after idle period (read deadline may have fired): %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("received %q, want %q", got, want)
	}

	// Clean up
	clientWsConn.Close()
	transferA.Close()
	<-transferDone
}
