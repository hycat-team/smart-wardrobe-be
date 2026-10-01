// Command nginx-routes generates the Nginx route map from each service's
// generated Swagger so a new API prefix never has to be hand-maintained in
// three places (FR-028).
//
// It computes the minimal set of longest-prefix locations that route every
// documented path to its owning service, marking `/stream` and `/sse` paths so
// they get proxy_buffering off + long timeouts (FR-036), and marking the
// long-running endpoints listed in longTimeoutPrefixes so they get their own
// wider proxy_send_timeout/proxy_read_timeout instead of inheriting the
// server-level default, then emits the region between GENERATED markers in
//
//	deployments/nginx/templates/services/vps-3service.conf.template (VPS api.closy upstreams)
//	deployments/nginx/templates/services-redirect/vps-3service.conf.template (VPS + api-v2 redirect grace)
//	deployments/nginx/templates/aws/api-v2.conf.template (AWS box api-v2, same "vps" target set)
//	deployments/nginx/local.conf (local compose)
//
// Run `make nginx-routes`; `make nginx-check` verifies coverage and CI fails on
// drift (`make nginx-routes && git diff --exit-code -- deployments/nginx`).
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	beginMarker = "# BEGIN GENERATED ROUTES (make nginx-routes) - do not edit by hand"
	endMarker   = "# END GENERATED ROUTES"
	// minPrefixSegments anchors generated prefixes at the API base (/api/v1);
	// see computeLocations.
	minPrefixSegments = 3
	// longTimeoutSeconds is the proxy budget given to a long-running endpoint.
	// It reuses the value already used by the generated SSE locations so there
	// is exactly one long-timeout number in the ingress.
	longTimeoutSeconds = 300
)

// longTimeoutPrefixes lists the API prefixes whose request budget outlives the
// server-level proxy_*_timeout (60s) inherited from the `server` block.
//
// The admin batch premium grant is the case that forced this list (FR-076): the
// use case enforces a 60s processing budget opened before any I/O, and the app
// write deadline is 120s (`server.write_timeout_seconds`). A proxy that gives up
// at 60s therefore cuts a request the app is still legitimately working on and
// the operator sees an infrastructure error for a grant that succeeded. The
// proxy must outlast the app write deadline, not merely the app budget.
var longTimeoutPrefixes = []string{
	"/api/v1/admin/subscription-grants",
}

var swaggerFiles = map[string]string{
	"account":  "services/account/api/swagger/swagger.json",
	"commerce": "services/commerce/api/swagger/swagger.json",
	"ai":       "services/ai/api/swagger/swagger.json",
}

// serviceOrder keeps generated files/regions stable across runs.
var serviceOrder = []string{"account", "commerce", "ai"}

// target renders the proxy target for a service in the given mode.
var targetTemplates = map[string]map[string]string{
	"vps": {
		"account":  "$account_upstream",
		"commerce": "$commerce_upstream",
		"ai":       "$ai_upstream",
	},
	"local": {
		"account":  "http://account:8080",
		"commerce": "http://commerce:8080",
		"ai":       "http://ai:8080",
	},
}

type location struct {
	prefix string
	exact  bool
	owner  string
	// sse marks a streaming endpoint (path ends in /stream or /sse) that needs
	// proxy_buffering off + long timeouts (FR-036).
	sse bool
	// long marks an endpoint whose request budget outlives the server-level
	// proxy timeouts, so the location must carry its own longer
	// proxy_send_timeout/proxy_read_timeout. Buffering stays on: the response is
	// a single buffered document, not a stream.
	long bool
}

type markerFile struct {
	path string // relative to root
	mode string // vps | local
}

var markerFiles = []markerFile{
	{path: "deployments/nginx/templates/services/vps-3service.conf.template", mode: "vps"},
	{path: "deployments/nginx/templates/services-redirect/vps-3service.conf.template", mode: "vps"},
	{path: "deployments/nginx/templates/aws/api-v2.conf.template", mode: "vps"},
	{path: "deployments/nginx/local.conf", mode: "local"},
}

func main() {
	root := flag.String("root", ".", "repository root")
	flag.Parse()

	services, err := loadSwagger(*root)
	if err != nil {
		fail(err)
	}
	locs, err := computeLocations(services)
	if err != nil {
		fail(err)
	}

	for _, mf := range markerFiles {
		path := filepath.Join(*root, mf.path)
		block := renderRegion(locs, mf.mode)
		if err := replaceRegion(path, block); err != nil {
			fail(err)
		}
	}

	fmt.Printf("nginx-routes: generated %d locations for %d services\n", len(locs), len(serviceOrder))
}

// computeLocations derives the minimal longest-prefix locations. A candidate
// prefix is accepted for a path when no path of another service sits under it;
// when the accepted prefix equals the full path it becomes an exact location.
func computeLocations(services map[string][]string) ([]location, error) {
	seen := map[string]string{}
	for svc, paths := range services {
		for _, p := range paths {
			if other, ok := seen[p]; ok {
				return nil, fmt.Errorf("path %s is documented by both %s and %s", p, other, svc)
			}
			seen[p] = svc
		}
	}

	byKey := map[string]*location{}
	for _, svc := range serviceOrder {
		for _, p := range services[svc] {
			prefix := p
			segs := splitPath(p)
			// Never emit a prefix broader than the API base (/api/v1) so a
			// single-service sparse dataset cannot capture unrelated paths.
			for i := minPrefixSegments; i <= len(segs); i++ {
				cand := "/" + strings.Join(segs[:i], "/")
				if !sharedWithOther(cand, svc, services) {
					prefix = cand
					break
				}
			}
			exact := prefix == p && deeperInOther(prefix, svc, services)
			key := prefix + "\x00" + svc
			if l, ok := byKey[key]; ok {
				l.exact = l.exact || exact
			} else {
				byKey[key] = &location{prefix: prefix, exact: exact, owner: svc}
			}
		}
	}

	locs := make([]location, 0, len(byKey))
	for _, l := range byKey {
		locs = append(locs, *l)
	}
	locs = minimize(locs)
	applyLongTimeout(locs)
	locs = mergeSSE(locs, computeSSELocations(services))
	sort.Slice(locs, func(i, j int) bool {
		if locs[i].owner != locs[j].owner {
			return ownerRank(locs[i].owner) < ownerRank(locs[j].owner)
		}
		return locs[i].prefix < locs[j].prefix
	})
	return locs, nil
}

// computeSSELocations derives one streaming location per static prefix of a
// path ending in /stream or /sse. The static prefix stops before the first
// `{param}`, and being longer than the generic service prefix it wins Nginx
// longest-prefix matching, so buffering is disabled for streams only.
func computeSSELocations(services map[string][]string) []location {
	byKey := map[string]*location{}
	for _, svc := range serviceOrder {
		for _, p := range services[svc] {
			if !isSSEPath(p) {
				continue
			}
			prefix := staticPrefix(p)
			if prefix == "" {
				continue
			}
			key := prefix + "\x00" + svc
			if _, ok := byKey[key]; !ok {
				byKey[key] = &location{prefix: prefix, owner: svc, sse: true}
			}
		}
	}
	out := make([]location, 0, len(byKey))
	for _, l := range byKey {
		out = append(out, *l)
	}
	return out
}

// mergeSSE appends streaming locations and upgrades a matching generic location
// in place (same prefix+owner) so it carries the buffering-off directives.
func mergeSSE(locs, sse []location) []location {
	index := make(map[string]int, len(locs))
	for i, l := range locs {
		index[l.prefix+"\x00"+l.owner] = i
	}
	for _, s := range sse {
		if i, ok := index[s.prefix+"\x00"+s.owner]; ok {
			locs[i].sse = true
			continue
		}
		locs = append(locs, s)
	}
	return locs
}

func isSSEPath(p string) bool {
	return strings.HasSuffix(p, "/stream") || strings.HasSuffix(p, "/sse")
}

// applyLongTimeout flags every location at or under a longTimeoutPrefix so it
// gets a proxy budget wider than the server-level 60s default.
//
// It runs after minimize, because that is when the surviving prefixes are
// known: the shortest prefix that still covers a slow endpoint is the one that
// needs the override.
func applyLongTimeout(locs []location) {
	for i := range locs {
		for _, p := range longTimeoutPrefixes {
			if locs[i].prefix == p || strings.HasPrefix(locs[i].prefix, p+"/") {
				locs[i].long = true
				break
			}
		}
	}
}

// staticPrefix returns the path up to the first `{param}` with no trailing slash.
func staticPrefix(p string) string {
	if i := strings.IndexByte(p, '{'); i >= 0 {
		p = p[:i]
	}
	return strings.TrimSuffix(p, "/")
}

// minimize drops a location when a shorter non-exact prefix of the same owner
// already covers it (Nginx longest-prefix would still route those paths). Exact
// locations are preserved because they guard a shared boundary.
func minimize(locs []location) []location {
	sort.Slice(locs, func(i, j int) bool { return len(locs[i].prefix) < len(locs[j].prefix) })
	kept := make([]location, 0, len(locs))
	for _, l := range locs {
		if l.exact {
			kept = append(kept, l)
			continue
		}
		covered := false
		for _, k := range kept {
			if k.owner == l.owner && !k.exact && strings.HasPrefix(l.prefix, k.prefix) {
				covered = true
				break
			}
		}
		if !covered {
			kept = append(kept, l)
		}
	}
	return kept
}

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

// sharedWithOther reports whether a path of another service sits at or under cand.
func sharedWithOther(cand, owner string, services map[string][]string) bool {
	for svc, paths := range services {
		if svc == owner {
			continue
		}
		for _, q := range paths {
			if atOrUnder(q, cand) {
				return true
			}
		}
	}
	return false
}

// deeperInOther reports whether another service has a path strictly under cand.
func deeperInOther(cand, owner string, services map[string][]string) bool {
	for svc, paths := range services {
		if svc == owner {
			continue
		}
		for _, q := range paths {
			if strings.HasPrefix(q, cand+"/") {
				return true
			}
		}
	}
	return false
}

func atOrUnder(q, cand string) bool {
	return q == cand || strings.HasPrefix(q, cand+"/")
}

func ownerRank(owner string) int {
	for i, svc := range serviceOrder {
		if svc == owner {
			return i
		}
	}
	return len(serviceOrder)
}

func renderRegion(locs []location, mode string) string {
	var b strings.Builder
	b.WriteString("    " + beginMarker + "\n")
	for _, l := range locs {
		mod := "^~"
		if l.exact {
			mod = "="
		}
		if l.sse {
			fmt.Fprintf(&b, "    location %s %s { proxy_pass %s; proxy_buffering off; proxy_cache off; proxy_send_timeout 300s; proxy_read_timeout 300s; }\n",
				mod, l.prefix, targetTemplates[mode][l.owner])
			continue
		}
		if l.long {
			fmt.Fprintf(&b, "    location %s %s { proxy_pass %s; proxy_send_timeout %ds; proxy_read_timeout %ds; }\n",
				mod, l.prefix, targetTemplates[mode][l.owner], longTimeoutSeconds, longTimeoutSeconds)
			continue
		}
		fmt.Fprintf(&b, "    location %s %s { proxy_pass %s; }\n", mod, l.prefix, targetTemplates[mode][l.owner])
	}
	b.WriteString("    " + endMarker)
	return b.String()
}

func replaceRegion(path, block string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	lines := strings.Split(string(raw), "\n")
	start, end := -1, -1
	for i, line := range lines {
		switch strings.TrimSpace(line) {
		case beginMarker:
			start = i
		case endMarker:
			end = i
		}
	}
	if start < 0 || end < 0 || end < start {
		return fmt.Errorf("%s: missing GENERATED ROUTES markers", path)
	}

	replaced := make([]string, 0, len(lines)-(end-start-1))
	replaced = append(replaced, lines[:start]...)
	replaced = append(replaced, strings.Split(block, "\n")...)
	replaced = append(replaced, lines[end+1:]...)
	if err := os.WriteFile(path, []byte(strings.Join(replaced, "\n")), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func loadSwagger(root string) (map[string][]string, error) {
	out := make(map[string][]string, len(swaggerFiles))
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
		paths := make([]string, 0, len(doc.Paths))
		for p := range doc.Paths {
			paths = append(paths, p)
		}
		sort.Strings(paths)
		out[svc] = paths
	}
	return out, nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "nginx-routes:", err)
	os.Exit(1)
}
