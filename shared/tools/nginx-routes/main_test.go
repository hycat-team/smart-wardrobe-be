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

func TestRenderRegionSSEDisablesBuffering(t *testing.T) {
	out := renderRegion([]location{
		{prefix: "/api/v1/wardrobe-items/tasks", owner: "commerce", sse: true},
	}, "local")
	if !strings.Contains(out, "proxy_buffering off;") || !strings.Contains(out, "http://commerce:8080") {
		t.Fatalf("local SSE region wrong:\n%s", out)
	}
}

func TestComputeLocationsWidenProxyTimeoutForLongRunningEndpoint(t *testing.T) {
	services := map[string][]string{
		"account": {
			"/api/v1/admin/subscription-grants",
			"/api/v1/admin/subscription-grants/restore",
			"/api/v1/admin/users",
		},
		// Commerce also owns /api/v1/admin/*, which is what keeps the grants
		// prefix from minimizing up to /api/v1/admin in the real route map.
		"commerce": {"/api/v1/admin/dashboard"},
	}
	locs, err := computeLocations(services)
	if err != nil {
		t.Fatal(err)
	}
	long := map[string]bool{}
	for _, l := range locs {
		long[l.prefix] = l.long
	}
	want := map[string]bool{
		"/api/v1/admin/subscription-grants": true,
		"/api/v1/admin/users":               false,
	}
	for prefix, expected := range want {
		got, ok := long[prefix]
		if !ok {
			t.Errorf("missing location %s (got %v)", prefix, long)
			continue
		}
		if got != expected {
			t.Errorf("location %s long = %v, want %v", prefix, got, expected)
		}
	}
}

func TestRenderRegionLongTimeoutKeepsBuffering(t *testing.T) {
	out := renderRegion([]location{
		{prefix: "/api/v1/admin/subscription-grants", owner: "account", long: true},
	}, "vps")
	if !strings.Contains(out, "proxy_send_timeout 300s;") || !strings.Contains(out, "proxy_read_timeout 300s;") {
		t.Fatalf("long-timeout location did not widen the proxy budget:\n%s", out)
	}
	// A single buffered document must still be buffered; only the budget changes.
	if strings.Contains(out, "proxy_buffering off;") {
		t.Fatalf("long-timeout location must not disable buffering:\n%s", out)
	}
	if !strings.Contains(out, "$account_upstream") {
		t.Fatalf("vps target wrong:\n%s", out)
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

// `make generate-check` runs this tool twice against an unchanged tree and
// fails on any diff, so `replaceRegion` being idempotent is a load-bearing
// property, not a nicety: a non-idempotent replacement stacks a second copy of
// the GENERATED ROUTES region on every run, and the diff shows up before anyone
// reads what changed.
//
// Nothing tested that until now - the idempotency claim lived only in the docs -
// so the same input is written twice here and the file must come out byte for
// byte identical.
func TestReplaceRegionIsIdempotent(t *testing.T) {
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
	first, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	// Same input, second run.
	if err := replaceRegion(file, block); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}

	if string(first) != string(second) {
		t.Fatalf("replaceRegion is not idempotent, the region was rewritten\nfirst:\n%s\nsecond:\n%s", first, second)
	}
	if got := strings.Count(string(second), beginMarker); got != 1 {
		t.Fatalf("expected exactly 1 BEGIN marker after two runs, got %d:\n%s", got, second)
	}
	if got := strings.Count(string(second), endMarker); got != 1 {
		t.Fatalf("expected exactly 1 END marker after two runs, got %d:\n%s", got, second)
	}
}
