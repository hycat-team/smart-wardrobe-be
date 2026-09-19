// Command nginx-check verifies that every HTTP path documented by each
// service's generated Swagger is reachable through the Nginx route map and is
// routed to the owning service (FR-028).
//
// The route map is maintained in four places (the services VPS template, the
// services-redirect VPS template for the api-v2 redirect grace period, the aws
// box template, and the local compose config), so a new API prefix is easy to
// forget. This command parses
// each service's swagger.json and
// every route-map file, then asserts each path resolves (longest-prefix, like
// Nginx) to a location whose proxy target belongs to the owning service.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// routeFile is one Nginx config to scan. Ownership is derived from each
// location's proxy_pass target.
type routeFile struct {
	path  string
	owner string
}

var routeFiles = []routeFile{
	{path: "deployments/nginx/templates/services/vps-3service.conf.template"},
	{path: "deployments/nginx/templates/services-redirect/vps-3service.conf.template"},
	{path: "deployments/nginx/templates/aws/api-v2.conf.template"},
	{path: "deployments/nginx/local.conf"},
}

var swaggerFiles = map[string]string{
	"account":  "services/account/api/swagger/swagger.json",
	"commerce": "services/commerce/api/swagger/swagger.json",
	"ai":       "services/ai/api/swagger/swagger.json",
}

var (
	locationRe  = regexp.MustCompile(`^location\s+(=|\^~|~\*|~)?\s*([^\s{]+)`)
	proxyPassRe = regexp.MustCompile(`proxy_pass\s+([^;\s]+)`)
)

type location struct {
	pattern string
	exact   bool
	regex   bool
	owner   string
}

func main() {
	root := flag.String("root", ".", "repository root")
	flag.Parse()

	swagger, err := loadSwagger(*root)
	if err != nil {
		fail(err)
	}

	var locations []location
	for _, rf := range routeFiles {
		locs, err := parseRouteFile(filepath.Join(*root, rf.path), rf.owner)
		if err != nil {
			fail(err)
		}
		locations = append(locations, locs...)
	}

	checked, failures := 0, 0
	for _, svc := range sortedKeys(swagger) {
		for _, p := range sortedKeys(swagger[svc]) {
			checked++
			owner := winner(locations, p)
			if owner == svc {
				continue
			}
			failures++
			if owner == "" {
				fmt.Printf("MISSING %-9s %-58s (no nginx location matches)\n", svc, p)
			} else {
				fmt.Printf("MISSING %-9s %-58s (longest-prefix match belongs to %q)\n", svc, p, owner)
			}
		}
	}

	if failures > 0 {
		fmt.Printf("\nnginx-check: FAIL — %d/%d swagger paths not routed to the owning service\n", failures, checked)
		fmt.Println("Add the missing prefix to deployments/nginx/templates/services/vps-3service.conf.template, deployments/nginx/templates/services-redirect/vps-3service.conf.template, deployments/nginx/templates/aws/api-v2.conf.template and deployments/nginx/local.conf (or regenerate with `make nginx-routes`)")
		os.Exit(1)
	}
	fmt.Printf("nginx-check: OK — all %d swagger paths routed to the owning service\n", checked)
}

// winner returns the owner selected by Nginx longest-prefix matching (exact
// match beats any prefix). Empty when nothing matches.
func winner(locs []location, path string) string {
	bestOwner, bestScore := "", -1
	for _, l := range locs {
		if l.regex || l.owner == "" {
			continue
		}
		score := len(l.pattern)
		matched := strings.HasPrefix(path, l.pattern)
		if l.exact {
			matched = l.pattern == path
			score += 1 << 30
		}
		if matched && score > bestScore {
			bestScore, bestOwner = score, l.owner
		}
	}
	return bestOwner
}

func parseRouteFile(path, forcedOwner string) ([]location, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var (
		locs []location
		cur  *location
	)
	flush := func() {
		if cur != nil {
			locs = append(locs, *cur)
			cur = nil
		}
	}

	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if m := locationRe.FindStringSubmatch(line); m != nil {
			flush()
			mod := m[1]
			cur = &location{
				pattern: m[2],
				exact:   mod == "=",
				regex:   mod == "~" || mod == "~*",
				owner:   forcedOwner,
			}
		}
		if cur == nil {
			continue
		}
		if cur.owner == "" {
			if m := proxyPassRe.FindStringSubmatch(line); m != nil {
				cur.owner = classify(m[1])
			}
		}
		// A line with `}` but no `{` closes the current block (locations here
		// never nest). Single-line blocks keep both braces and are not closed.
		if strings.Contains(line, "}") && !strings.Contains(line, "{") {
			flush()
		}
	}
	flush()
	return locs, nil
}

func classify(target string) string {
	t := strings.ToLower(target)
	switch {
	case strings.Contains(t, "account"):
		return "account"
	case strings.Contains(t, "commerce"):
		return "commerce"
	case strings.Contains(t, "ai_run_url"), strings.Contains(t, "ai_upstream"), strings.Contains(t, "ai:8080"), strings.Contains(t, "//ai"):
		return "ai"
	}
	return ""
}

func loadSwagger(root string) (map[string]map[string]json.RawMessage, error) {
	out := make(map[string]map[string]json.RawMessage, len(swaggerFiles))
	for svc, rel := range swaggerFiles {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", rel, err)
		}
		var doc struct {
			Paths map[string]json.RawMessage `json:"paths"`
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil, fmt.Errorf("parse %s: %w", rel, err)
		}
		out[svc] = doc.Paths
	}
	return out, nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "nginx-check:", err)
	os.Exit(2)
}
