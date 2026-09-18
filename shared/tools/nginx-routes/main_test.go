package main

import (
	"os"
	"sort"
	"strings"
	"testing"
)

func keysOf(locs []location) []string {
	out := make([]string, 0, len(locs))
	for _, l := range locs {
		mod := "~"
		if l.exact {
			mod = "="
		}
		out = append(out, l.owner+" "+mod+" "+l.prefix)
	}
	sort.Strings(out)
	return out
}

func TestComputeLocationsAnchorsAtAPIV1(t *testing.T) {
	services := map[string][]string{
		"account": {"/api/v1/auth/login", "/api/v1/auth/logout"},
	}
	locs, err := computeLocations(services)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"account ~ /api/v1/auth"}
	if got := keysOf(locs); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestComputeLocationsSharedBoundary(t *testing.T) {
	services := map[string][]string{
		"account":  {"/api/v1/me", "/api/v1/me/avatar"},
		"commerce": {"/api/v1/me/wardrobe-items"},
	}
	locs, err := computeLocations(services)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"account = /api/v1/me",
		"account ~ /api/v1/me/avatar",
		"commerce ~ /api/v1/me/wardrobe-items",
	}
	if got := keysOf(locs); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestComputeLocationsDuplicatePathFails(t *testing.T) {
	services := map[string][]string{
		"account":  {"/api/v1/foo"},
		"commerce": {"/api/v1/foo"},
	}
	if _, err := computeLocations(services); err == nil {
		t.Fatal("expected error for a path owned by two services")
	}
}

func TestComputeLocationsEmitsSSELocations(t *testing.T) {
	services := map[string][]string{
		"commerce": {
			"/api/v1/ai/chat/sessions/{contextID}/messages/stream",
			"/api/v1/wardrobe-items/tasks/{taskId}/sse",
		},
	}
	locs, err := computeLocations(services)
	if err != nil {
		t.Fatal(err)
	}

	sse := map[string]string{}
	for _, l := range locs {
		if l.sse {
			sse[l.prefix] = l.owner
		}
	}
	want := map[string]string{
		"/api/v1/ai/chat/sessions":     "commerce",
		"/api/v1/wardrobe-items/tasks": "commerce",
	}
	if len(sse) != len(want) {
		t.Fatalf("sse locations = %v, want %v", sse, want)
	}
	for prefix, owner := range want {
		if sse[prefix] != owner {
			t.Errorf("missing sse location %s -> %s (got %v)", prefix, owner, sse)
		}
	}

	// The generic prefixes must also survive for non-streaming endpoints.
	keys := keysOf(locs)
	for _, wantKey := range []string{"commerce ~ /api/v1/ai", "commerce ~ /api/v1/wardrobe-items"} {
		found := false
		for _, k := range keys {
			if k == wantKey {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing generic location %q (got %v)", wantKey, keys)
		}
	}
}

func TestRenderAPIV2SSEDisablesBuffering(t *testing.T) {
	out := renderAPIV2("commerce", []location{
		{prefix: "/api/v1/ai/chat/sessions", owner: "commerce", sse: true},
	})
	for _, want := range []string{
		"location ^~ /api/v1/ai/chat/sessions {",
		"proxy_buffering off;",
		"proxy_read_timeout 300s;",
		"proxy_pass ${COMMERCE_RUN_URL};",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("api-v2 SSE output missing %q", want)
		}
	}
}

func TestRenderRegionSSEDisablesBuffering(t *testing.T) {
	out := renderRegion([]location{
		{prefix: "/api/v1/wardrobe-items/tasks", owner: "commerce", sse: true},
	}, "local")
	if !strings.Contains(out, "proxy_buffering off;") || !strings.Contains(out, "http://commerce:8080") {
		t.Fatalf("local SSE region wrong:\n%s", out)
	}
}

func TestMinimizeDropsCoveredPrefixes(t *testing.T) {
	locs := []location{
		{prefix: "/api/v1/brand", owner: "commerce"},
		{prefix: "/api/v1/brand-benefits", owner: "commerce"},
		{prefix: "/api/v1/brand-items", owner: "commerce"},
		{prefix: "/api/v1/brands", owner: "ai"},
	}
	got := keysOf(minimize(locs))
	want := []string{
		"ai ~ /api/v1/brands",
		"commerce ~ /api/v1/brand",
	}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestRenderAPIV2IncludesAuthAndTarget(t *testing.T) {
	locs := []location{
		{prefix: "/api/v1/auth", owner: "account"},
		{prefix: "/api/v1/me", exact: true, owner: "account"},
	}
	out := renderAPIV2("account", locs)
	for _, want := range []string{
		"location ^~ /api/v1/auth {",
		"location = /api/v1/me {",
		"auth_request /_token_helper_account;",
		"proxy_pass ${ACCOUNT_RUN_URL};",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("generated output missing %q", want)
		}
	}
}

func TestReplaceRegion(t *testing.T) {
	dir := t.TempDir()
	file := dir + "/local.conf"
	content := "server {\n" +
		"    " + beginMarker + "\n" +
		"    location ^~ /old { proxy_pass http://old:8080; }\n" +
		"    " + endMarker + "\n" +
		"}\n"
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	block := renderRegion([]location{{prefix: "/api/v1/auth", owner: "account"}}, "local")
	if err := replaceRegion(file, block); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	updated := string(raw)
	if strings.Contains(updated, "/old") || !strings.Contains(updated, "/api/v1/auth") {
		t.Fatalf("region not replaced:\n%s", updated)
	}
}
