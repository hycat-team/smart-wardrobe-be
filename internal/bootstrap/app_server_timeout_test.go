package bootstrap

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestSlowlorisConnectionClosedAfterReadTimeout proves spec 019 SC-001: a
// connection that completes the TCP handshake but never sends request data is
// closed by the server once ReadTimeout elapses, instead of being held
// indefinitely (slowloris-style resource exhaustion).
func TestSlowlorisConnectionClosedAfterReadTimeout(t *testing.T) {
	const readTimeout = 150 * time.Millisecond

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	// Mirror the wiring in App.Run — ReadTimeout is what must terminate the
	// silent connection.
	srv.Config.ReadTimeout = readTimeout
	srv.Start()
	defer srv.Close()

	conn, err := net.Dial("tcp", srv.Listener.Addr().String())
	if err != nil {
		t.Fatalf("dial server: %v", err)
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	start := time.Now()
	buf := make([]byte, 1)
	if _, err := conn.Read(buf); err == nil {
		t.Fatal("server held the silent connection open; expected it to be closed after ReadTimeout")
	}
	if elapsed := time.Since(start); elapsed > readTimeout+2*time.Second {
		t.Fatalf("silent connection closed after %v; expected within ReadTimeout (%v) plus slack", elapsed, readTimeout)
	}
}
