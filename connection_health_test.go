package wstun

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestSendToControlConnections_PrunesDeadConn verifies that sendToControlConnections
// removes a control connection from the registry when a write to it fails.
// This prevents CONNECT messages from being silently dropped into a dead connection.
func TestSendToControlConnections_PrunesDeadConn(t *testing.T) {
	sp := NewServerProtocol(func(_, _ string) bool { return true })
	sp.Logger = log.New(io.Discard, "", 0)
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
	sp.Logger = log.New(io.Discard, "", 0)
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
	cp.Logger = log.New(io.Discard, "", 0)
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
