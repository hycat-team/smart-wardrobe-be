package search

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"smart-wardrobe-be/shared/config"
)

// disabledCfg returns a config with the search engine switched off (FR-001).
// The logger is intentionally nil: every disabled code path must return before
// touching it (no network, no goroutine, no log spam).
func disabledCfg() *config.Config {
	return &config.Config{}
}

func TestDisabledClientNoOps(t *testing.T) {
	c := NewElasticsearchClient(disabledCfg(), nil)
	defer c.Stop()

	if c.Enabled() {
		t.Fatal("Enabled() = true, want false for disabled engine")
	}

	ctx := context.Background()
	if err := c.IndexDocument(ctx, "wardrobe_items", "id-1", map[string]any{"a": 1}); err != nil {
		t.Fatalf("IndexDocument must no-op when disabled, got %v", err)
	}
	if err := c.DeleteDocument(ctx, "wardrobe_items", "id-1"); err != nil {
		t.Fatalf("DeleteDocument must no-op when disabled, got %v", err)
	}
	if _, err := c.Search(ctx, "wardrobe_items", map[string]any{}); err == nil {
		t.Fatal("Search must fail when disabled so read paths fall back to database")
	}
}

func TestDisabledClientStartsNoHealthGoroutine(t *testing.T) {
	// A reachable ES proves the point: an enabled client would flip healthy,
	// a disabled one must stay unhealthy because no checker ever runs.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := disabledCfg()
	cfg.Elasticsearch.Addresses = []string{srv.URL}
	c := NewElasticsearchClient(cfg, nil)
	defer c.Stop()

	time.Sleep(200 * time.Millisecond)
	if c.IsHealthy() {
		t.Fatal("IsHealthy() = true for disabled engine — health goroutine must not start")
	}
}
