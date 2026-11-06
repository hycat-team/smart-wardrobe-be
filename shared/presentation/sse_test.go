package presentation_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	shared_pres "smart-wardrobe-be/shared/presentation"

	"github.com/gin-gonic/gin"
)

const (
	sseChunks       = 8
	sseChunkDelay   = 40 * time.Millisecond
	sseWriteTimeout = 100 * time.Millisecond // far shorter than the total stream duration
)

// startSSEServer spins a Gin server whose WriteTimeout is shorter than the
// total stream duration. extendDeadline controls whether the handler pushes
// the write deadline per chunk (the mechanism under test, spec 019 FR-003).
func startSSEServer(t *testing.T, extendDeadline bool) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/stream", func(c *gin.Context) {
		flusher, err := shared_pres.InitSSE(c)
		if err != nil {
			return
		}
		for i := 0; i < sseChunks; i++ {
			if extendDeadline {
				shared_pres.ExtendSSEWriteDeadline(c)
			}
			if _, err := fmt.Fprintf(c.Writer, "data: %d\n\n", i); err != nil {
				return
			}
			flusher.Flush()
			time.Sleep(sseChunkDelay)
		}
	})
	srv := httptest.NewUnstartedServer(r)
	srv.Config.WriteTimeout = sseWriteTimeout
	srv.Start()
	return srv
}

func readStreamEvents(t *testing.T, url string) int {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("stream request failed: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return strings.Count(string(body), "data:")
}

// FR-003 positive: with per-chunk deadline extension the whole stream is
// delivered even though the server WriteTimeout is far shorter.
func TestSSEStreamSurvivesWriteTimeoutWithDeadlineExtension(t *testing.T) {
	srv := startSSEServer(t, true)
	defer srv.Close()

	if got := readStreamEvents(t, srv.URL+"/stream"); got != sseChunks {
		t.Fatalf("expected all %d chunks to survive the write timeout, got %d", sseChunks, got)
	}
}

// heartbeat_seconds must actually drive the ping ticker: a config carried from
// before the wiring existed (zero) has to keep the previous 15s behaviour.
func TestConfigureSSEHeartbeatFallsBackToDefault(t *testing.T) {
	t.Cleanup(shared_pres.ResetSSEDeadlines)

	shared_pres.ConfigureSSEHeartbeat(0)
	if got := shared_pres.SSEHeartbeat(); got != shared_pres.DefaultSSEHeartbeatInterval {
		t.Fatalf("heartbeat = %v, want the default %v", got, shared_pres.DefaultSSEHeartbeatInterval)
	}

	shared_pres.ConfigureSSEHeartbeat(-3 * time.Second)
	if got := shared_pres.SSEHeartbeat(); got != shared_pres.DefaultSSEHeartbeatInterval {
		t.Fatalf("heartbeat = %v, want the default %v", got, shared_pres.DefaultSSEHeartbeatInterval)
	}
}

func TestConfigureSSEHeartbeatAppliesConfiguredValue(t *testing.T) {
	t.Cleanup(shared_pres.ResetSSEDeadlines)

	shared_pres.ConfigureSSEHeartbeat(5 * time.Second)
	if got := shared_pres.SSEHeartbeat(); got != 5*time.Second {
		t.Fatalf("heartbeat = %v, want 5s", got)
	}

	shared_pres.ResetSSEDeadlines()
	if got := shared_pres.SSEHeartbeat(); got != shared_pres.DefaultSSEHeartbeatInterval {
		t.Fatalf("reset heartbeat = %v, want the default", got)
	}
}
