package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClassify(t *testing.T) {
	cases := map[string]string{
		"${ACCOUNT_RUN_URL}":   "account",
		"http://account:8080":  "account",
		"$account_upstream":    "account",
		"${COMMERCE_RUN_URL}":  "commerce",
		"http://commerce:8080": "commerce",
		"$commerce_upstream":   "commerce",
		"${AI_RUN_URL}":        "ai",
		"$ai_upstream":         "ai",
		"http://ai:8080":       "ai",
		"":                     "",
	}
	for target, want := range cases {
		if got := classify(target); got != want {
			t.Errorf("classify(%q) = %q, want %q", target, got, want)
		}
	}
}

func TestWinnerLongestPrefix(t *testing.T) {
	locs := []location{
		{pattern: "/api/v1/admin", owner: "account"},
		{pattern: "/api/v1/admin/ai", owner: "ai"},
		{pattern: "/api/v1/admin/brands", owner: "commerce"},
	}
	cases := map[string]string{
		"/api/v1/admin/users":       "account",
		"/api/v1/admin/ai/grants":   "ai",
		"/api/v1/admin/brands/{id}": "commerce",
		"/api/v1/unrelated":         "",
	}
	for path, want := range cases {
		if got := winner(locs, path); got != want {
			t.Errorf("winner(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestWinnerExactBeatsPrefix(t *testing.T) {
	locs := []location{
		{pattern: "/api/v1/me", exact: true, owner: "account"},
		{pattern: "/api/v1/me/wardrobe-items", owner: "commerce"},
	}
	if got := winner(locs, "/api/v1/me"); got != "account" {
		t.Errorf("exact /api/v1/me = %q, want account", got)
	}
	if got := winner(locs, "/api/v1/me/wardrobe-items"); got != "commerce" {
		t.Errorf("/api/v1/me/wardrobe-items = %q, want commerce", got)
	}
}

func TestWinnerIgnoresRegexAndUnknownOwner(t *testing.T) {
	locs := []location{
		{pattern: "/api/v1/ai/.*/stream", regex: true, owner: "commerce"},
		{pattern: "/api/v1/foo", owner: ""},
	}
	if got := winner(locs, "/api/v1/ai/x/stream"); got != "" {
		t.Errorf("regex location must be ignored, got %q", got)
	}
	if got := winner(locs, "/api/v1/foo/bar"); got != "" {
		t.Errorf("ownerless location must be ignored, got %q", got)
	}
}

func TestParseRouteFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "routes.conf")
	content := `# comment location /not/a/match
location ^~ /api/v1/brands {
    proxy_pass http://commerce:8080;
}
location = /api/v1/me { proxy_pass $account_upstream; }
location ^~ /api/v1/admin/ai { proxy_pass $ai_upstream; }
location ~ ^/api/v1/ai/.*/stream { proxy_pass http://commerce:8080; }
`
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	locs, err := parseRouteFile(file, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(locs) != 4 {
		t.Fatalf("got %d locations, want 4: %+v", len(locs), locs)
	}

	want := []location{
		{pattern: "/api/v1/brands", owner: "commerce"},
		{pattern: "/api/v1/me", exact: true, owner: "account"},
		{pattern: "/api/v1/admin/ai", owner: "ai"},
		{pattern: "^/api/v1/ai/.*/stream", regex: true, owner: "commerce"},
	}
	for i := range want {
		if locs[i] != want[i] {
			t.Errorf("location[%d] = %+v, want %+v", i, locs[i], want[i])
		}
	}
}

func TestParseRouteFileForcedOwner(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "account.conf")
	content := `location ^~ /api/v1/auth {
    proxy_pass ${ACCOUNT_RUN_URL};
}
`
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	locs, err := parseRouteFile(file, "account")
	if err != nil {
		t.Fatal(err)
	}
	if len(locs) != 1 || locs[0].owner != "account" {
		t.Fatalf("forced owner not applied: %+v", locs)
	}
}
