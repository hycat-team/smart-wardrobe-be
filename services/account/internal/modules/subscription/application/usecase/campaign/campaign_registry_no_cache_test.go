package campaign

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// 026 T076: the two performance/portability rules that are PROPERTIES rather than
// behaviour — FR-080 (no cache in front of the registry) and FR-081 (no environment
// variable carrying campaign data).
//
// WHY A SOURCE GUARD AND NOT A BEHAVIOURAL TEST. Neither rule can be observed from
// outside: a cache whose entries expire in one millisecond behaves exactly like no
// cache at all for every test in this package, and a campaign read from an environment
// variable produces the same grants as one read from the table. What the two claims
// really forbid is a LAYER — a symbol, an import, a key — and layers are visible on the
// AST and in the text of a file, not in the values it returns. So the assertions below
// read the source rather than the results, in the same way
// `TestAdministratorExclusionIsDefinedOnceAndSharedByEveryPath` does for FR-080's other
// half.
//
// All three guards are STATIC, so they run in a package with no database and no Redis.
// That is deliberate: FR-080 and FR-081 are the two rules whose regression is cheapest
// to catch before the code is even run.

// campaignRepoRoot walks up from this package to the workspace root.
//
// The path is written out rather than searched for a marker file, because a search
// would succeed from anywhere and the guard's own failure mode — silently finding
// nothing and passing vacuously — is exactly what the existence checks below rule out.
func campaignRepoRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(
		"..", "..", "..", "..", "..", "..", "..", "..",
	)
	resolved, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("resolve repository root from %s: %v", root, err)
	}
	// The marker is `go.work`, which only the workspace root carries.
	if _, err := os.Stat(filepath.Join(resolved, "go.work")); err != nil {
		t.Fatalf("cannot locate the workspace root at %s (go.work missing): %v", resolved, err)
	}
	return resolved
}

// campaignServiceRoot is `services/account`, where every campaign symbol lives.
func campaignServiceRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join(campaignRepoRoot(t), "services", "account")
}

// walkGoFiles lists every non-test .go file under `root`.
//
// GENERATED AND TEST FILES ARE SKIPPED for two reasons, both of which matter here: a
// generated file is not where a layer would be introduced on purpose, and a test file
// is allowed to mention a cache symbol as the SUBJECT of an assertion — a guard that
// matched its own fixtures would fail the moment somebody wrote a test for it.
func walkGoFiles(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			name := entry.Name()
			// `api/swagger` holds generated OpenAPI documents; `vendor` is not part
			// of this repository's layout but would be noise if it appeared.
			if name == "swagger" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(body[:minInt(len(body), 200)]), "Code generated") {
			return nil
		}
		out = append(out, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	if len(out) == 0 {
		t.Fatalf("no Go source found under %s; the guard would pass vacuously", root)
	}
	return out
}

// minInt is the two-argument minimum.
//
// IT IS NOT NAMED `min`. Go has a builtin of that name since 1.21, and a package-level
// function called `min` in a _test file would shadow the builtin for EVERY test file in
// this package — so a later test using the builtin's variadic form would stop compiling
// with a message pointing at this file.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// renderNode prints one AST node back as source.
//
// Go's printer rather than a substring search of the file, for the reason
// `renderExpr` already gives: the question these guards ask is "does this DECLARATION
// refer to the campaign registry", and a substring search cannot tell a declaration
// from a sentence explaining why there isn't one.
func renderNode(t *testing.T, node ast.Node) string {
	t.Helper()
	var out bytes.Buffer
	if err := format.Node(&out, token.NewFileSet(), node); err != nil {
		t.Fatalf("render node: %v", err)
	}
	return out.String()
}

// campaignDecl is one top-level declaration of a file.
type campaignDecl struct {
	name string
	body string
	kind string
}

// topLevelDecls parses one file and returns every top-level declaration in it.
//
// PACKAGE-LEVEL `var` AND `const` GROUPS are SPLIT into their individual names rather
// than reported as one block, so a file that declares `campaignCacheTTL` inside a
// `const (...)` block cannot hide behind the block's own name.
func topLevelDecls(t *testing.T, path, body string) []campaignDecl {
	t.Helper()
	parsed, err := parser.ParseFile(token.NewFileSet(), path, body, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	var out []campaignDecl
	for _, decl := range parsed.Decls {
		switch typed := decl.(type) {
		case *ast.FuncDecl:
			out = append(out, campaignDecl{name: typed.Name.Name, body: renderNode(t, typed), kind: "func"})
		case *ast.GenDecl:
			if typed.Tok == token.IMPORT {
				continue
			}
			rendered := renderNode(t, typed)
			for _, spec := range typed.Specs {
				switch inner := spec.(type) {
				case *ast.TypeSpec:
					out = append(out, campaignDecl{name: inner.Name.Name, body: rendered, kind: "type"})
				case *ast.ValueSpec:
					for _, ident := range inner.Names {
						out = append(out, campaignDecl{name: ident.Name, body: rendered, kind: typed.Tok.String()})
					}
				}
			}
		}
	}
	return out
}

// campaignRegistryIdentifier matches the words that name the campaign registry itself.
//
// It is deliberately BROAD — `entities.Campaign`, `ICampaignRepository`,
// `CampaignSummaryRes` and a bare `campaign` field all appear in the answer to "does
// this cache hold the registry?" — and it is case-insensitive on the first letter so
// that a lower-camel helper cannot slip past a capital-C pattern.
var campaignRegistryIdentifier = regexp.MustCompile(`(?i)campaign`)

// cacheSymbolName matches a declaration whose NAME says it caches something.
var cacheSymbolName = regexp.MustCompile(`(?i)cache`)

// cachingImport matches an import of a caching layer, by path.
var cachingImport = regexp.MustCompile(`(?i)(infrastructure/caching|redis|go-redis)`)

// TestCampaignRegistryHasNoCacheLayer is FR-080's first half.
//
// FR-080 asks for ONE indexed query per signup message and forbids a cache in front of
// the registry, and the reason is a failure mode rather than a number: research.md R7
// — a cached list lets one replica keep granting from a campaign another replica has
// just closed, which is precisely the failure the "close it early" goal exists to
// prevent. So the rule is not "keep the cache fresh", it is "there is no cache".
//
// THE ASSERTION IS ON DECLARATION NAMES. Every symbol in `services/account` whose name
// contains "cache" is rendered, and any that mentions the campaign registry is
// reported. The set that exists today is the per-user SUBSCRIPTION OVERVIEW cache of
// 024 — which is a different fact about a different row, and which this guard must not
// forbid, so it passes only because it never mentions a campaign. A
// `CampaignRegistryCache`, a `campaignCacheTTL`, or a `GetCachedCampaign` would all be
// caught here, and each would be caught by NAME even if its body were a comment saying
// it was unused.
func TestCampaignRegistryHasNoCacheLayer(t *testing.T) {
	files := walkGoFiles(t, campaignServiceRoot(t))

	var offenders []string
	var cacheSymbols []string

	for _, path := range files {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(body)

		for _, decl := range topLevelDecls(t, path, text) {
			if !cacheSymbolName.MatchString(decl.name) {
				continue
			}
			cacheSymbols = append(cacheSymbols, decl.name)
			// The body's COMMENTS are stripped before the campaign test, because
			// `RedisSubscriptionOverviewCache`'s own header explains at length why the
			// CAMPAIGN registry is not cached beside it — and a guard that matched its
			// own justification would forbid writing down the reason.
			stripped := stripLineComments(decl.body)
			if campaignRegistryIdentifier.MatchString(stripped) {
				offenders = append(offenders, path+" -> "+decl.kind+" "+decl.name)
			}
		}
	}

	// The guard must be RUNNING over something. Without this, deleting the package
	// would turn the test green.
	if len(cacheSymbols) == 0 {
		t.Fatal("no cache symbol found anywhere in services/account; this guard is not reading the " +
			"module it claims to guard and would pass vacuously")
	}

	if len(offenders) > 0 {
		t.Errorf("FR-080: the campaign registry must have NO cache layer - one indexed query per signup "+
			"message, and no cached list, because a cached list would let one replica keep granting from a "+
			"campaign another replica has just closed (research.md R7). These declarations mention the "+
			"campaign registry and are named like a cache:\n  %s", strings.Join(offenders, "\n  "))
	}
}

// TestCampaignPackagesImportNoCachingLayer is the import half of FR-080.
//
// IT IS A SEPARATE TEST rather than a branch of the one above, because the two fail
// for different reasons and a reader who sees one green and one red must be able to
// tell which rule was broken. The declaration test catches a cache NAMED as one; this
// one catches a cache reached through a generic service handle, whose package-level
// names may be perfectly innocent.
//
// The scope is the two campaign directories that could hold a read or a write of the
// registry: the use-case package (where the grant reads it) and the presentation
// package (where the operator endpoints read and write it).
func TestCampaignPackagesImportNoCachingLayer(t *testing.T) {
	service := campaignServiceRoot(t)
	targets := []string{
		filepath.Join(service, "internal", "modules", "subscription", "application", "usecase", "campaign"),
		filepath.Join(service, "internal", "modules", "subscription", "presentation"),
		filepath.Join(service, "internal", "modules", "subscription", "domain", "entities"),
	}

	// The whole service is scanned so that a cache for the registry cannot be declared
	// one package away from the campaign packages and wired in from there.
	files := walkGoFiles(t, service)

	var offenders []string
	for _, path := range files {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, body, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, imp := range parsed.Imports {
			value, err := strconvUnquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("read import path in %s: %v", path, err)
			}
			if !cachingImport.MatchString(value) {
				continue
			}
			// The 024 overview cache is allowed to import Redis; the campaign PATH is
			// not. The discriminator is the FILE, not the module: a file inside a
			// campaign directory is the campaign path, and a file anywhere else that
			// mentions a campaign type is not.
			for _, dir := range targets {
				if strings.HasPrefix(path, dir+string(filepath.Separator)) {
					offenders = append(offenders, path+" imports "+value)
				}
			}
		}
	}

	if len(offenders) > 0 {
		t.Errorf("FR-080: no campaign package may import a caching layer; the registry is read from the "+
			"table on every signup message:\n  %s", strings.Join(offenders, "\n  "))
	}
}

// TestEveryGrantReadsTheRegistryThroughTheRepository is FR-080's second half: the
// positive claim, that the grant path really does read the registry, and does so
// through `ICampaignRepository` rather than through configuration.
//
// THREE DOORS, and each is closed independently:
//
//  1. `campaign_uc.go` asks the REPOSITORY for the open campaigns and hands the result
//     to the evaluator. Asserting the call site rather than the behaviour is what makes
//     this meaningful: a fake repository in a test answers just as well as a real one,
//     so no behavioural test in this package can tell the two apart.
//  2. NO production file in the campaign package mentions `cfg.Campaigns` or
//     `CampaignConfig`. Those were 025's source, and they were removed from the
//     configuration package; a surviving mention would mean one of the two was
//     reintroduced under a name this package still resolves.
//  3. The grant's re-read INSIDE the transaction asks for `GetByCodeForShare` (FR-045),
//     which is what stops a campaign closed in the gap between the two reads from
//     still granting.
func TestEveryGrantReadsTheRegistryThroughTheRepository(t *testing.T) {
	sources := campaignSourceFiles(t)
	if len(sources) == 0 {
		t.Fatal("no campaign source files found; the guard would pass vacuously")
	}

	var configReaders []string
	var listOpenSites []string
	var forShareSites []string

	for _, file := range sources {
		// Doors 2 and 1/3, as text over the STRIPPED source so that a paragraph
		// explaining the removal cannot be mistaken for the removal failing.
		stripped := stripLineComments(file.body)

		if strings.Contains(stripped, "cfg.Campaigns") ||
			strings.Contains(stripped, ".Campaigns") ||
			strings.Contains(stripped, "CampaignConfig") {
			configReaders = append(configReaders, file.name)
		}
		if strings.Contains(stripped, "campaignRepo.ListOpen(") {
			listOpenSites = append(listOpenSites, file.name)
		}
		if strings.Contains(stripped, "GetByCodeForShare(") {
			forShareSites = append(forShareSites, file.name)
		}
	}

	if len(listOpenSites) == 0 {
		t.Errorf("no campaign source calls campaignRepo.ListOpen; every signup message must read the open " +
			"campaigns from the registry (FR-040, FR-080). A grant path that assembled its list from " +
			"anything else would be the second source of truth this feature exists to end")
	}
	if len(configReaders) > 0 {
		t.Errorf("FR-001/FR-080: no campaign source may read a campaign list from configuration; "+
			"`subscription.campaigns` is the only source of which campaigns exist. Found the configuration "+
			"vocabulary in: %s", strings.Join(configReaders, ", "))
	}
	if len(forShareSites) == 0 {
		t.Errorf("no campaign source calls GetByCodeForShare; the grant must re-read the campaign under a " +
			"shared lock inside its own transaction, or an operator who closed the campaign in the gap " +
			"between the two reads would still see one grant go out (FR-045)")
	}
}

// TestNoEnvironmentVariableCarriesCampaignData is FR-081, plus the §VIII portability
// rule it exists to protect.
//
// FR-081 is "no new environment variable", and the reason it matters is not tidiness:
// constitution §VIII requires switching environments to be a configuration change and
// never a code change, and a campaign value in an environment variable is BOTH a second
// source of truth for the registry (FR-001) and a value that differs per environment for
// no reason a promotion can explain.
//
// THE ASSERTION IS ON ASSIGNMENTS, NOT ON MENTIONS. Both example files deliberately
// DISCUSS campaigns — in prose, to say that no variable exists and why — and a guard
// that matched the word would forbid documenting the rule. So only non-comment lines
// count, and the files that mention campaigns in prose are additionally required to
// still say so, which is what keeps the explanation from being quietly deleted.
func TestNoEnvironmentVariableCarriesCampaignData(t *testing.T) {
	root := campaignRepoRoot(t)

	examples := []string{".env.example", ".env.production.example", ".env.aws.example"}
	var inspected int
	var offenders []string
	var documented []string

	envVar := regexp.MustCompile(`^\s*(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=`)

	for _, name := range examples {
		path := filepath.Join(root, name)
		body, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			t.Fatalf("read %s: %v", name, err)
		}
		inspected++
		mentionsCampaign := false

		for i, line := range strings.Split(string(body), "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			if strings.HasPrefix(trimmed, "#") {
				// Comments are English by constitution §Code Style and are where the
				// "there is no variable here" explanation lives.
				if campaignRegistryIdentifier.MatchString(trimmed) {
					mentionsCampaign = true
				}
				continue
			}
			m := envVar.FindStringSubmatch(trimmed)
			if m == nil {
				continue
			}
			key := m[1]
			if !strings.Contains(strings.ToUpper(key), "CAMPAIGN") {
				continue
			}
			offenders = append(offenders, fmt.Sprintf("%s:%d assigns %s", name, i+1, key))
		}

		if mentionsCampaign {
			documented = append(documented, name)
		}
	}

	if inspected == 0 {
		t.Fatalf("no .env example file found at the workspace root %s; the guard would pass vacuously", root)
	}
	if len(offenders) > 0 {
		t.Errorf("FR-081: no environment variable may carry campaign data - the registry is the table and "+
			"the YAML `campaigns` block is gone, so a variable would be a THIRD source of truth for one "+
			"fact:\n  %s", strings.Join(offenders, "\n  "))
	}
	// At least one example file must still carry the explanation, or the rule became
	// invisible the moment somebody added the key this test forbids.
	if len(documented) == 0 {
		t.Errorf("no .env example file mentions campaigns at all; the note explaining that there is " +
			"deliberately no such variable (FR-081) has gone missing")
	}
}

// TestConfigurationHasNoCampaignsBlock is the YAML half of the same rule.
//
// The variable guard above would pass even if somebody re-added the `campaigns:` block
// to a config file, which is the FIRST failure mode of 025 and the one this feature
// closed. `campaign_sweep:` must survive — the sweep cadence belongs to the deployment
// and keeping it in YAML is what keeps configuration loading free of database access
// (FR-003/FR-004) — so the assertion is that `campaigns` is GONE while `campaign_sweep`
// REMAINS, in both environment files.
func TestConfigurationHasNoCampaignsBlock(t *testing.T) {
	root := campaignRepoRoot(t)
	configs := []string{
		filepath.Join("configs", "config.yaml"),
		filepath.Join("configs", "config.production.yaml"),
	}

	// A YAML key at the start of a line, so `campaign_sweep` does not match `campaigns`.
	campaignsKey := regexp.MustCompile(`(?m)^\s*campaigns\s*:`)
	sweepKey := regexp.MustCompile(`(?m)^\s*campaign_sweep\s*:`)

	var offenders []string
	var inspected int

	for _, rel := range configs {
		body, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		inspected++
		if campaignsKey.Match(body) {
			offenders = append(offenders, rel+" still declares a `campaigns` block")
		}
		if !sweepKey.Match(body) {
			offenders = append(offenders, rel+" no longer declares `campaign_sweep`")
		}
	}

	if inspected != len(configs) {
		t.Fatalf("expected to inspect %d config files, inspected %d", len(configs), inspected)
	}
	if len(offenders) > 0 {
		t.Errorf("FR-001/FR-002/FR-004: the YAML `campaigns` block must be gone from both environment files "+
			"while `campaign_sweep` stays (its cadence belongs to the deployment, and keeping it here is "+
			"what keeps configuration loading free of database access, FR-003):\n  %s",
			strings.Join(offenders, "\n  "))
	}
}

// stripLineComments removes `//` comments so a guard cannot match its own explanation.
func stripLineComments(body string) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "//"); idx >= 0 {
			lines[i] = line[:idx]
		}
	}
	return strings.Join(lines, "\n")
}

// strconvUnquote reads an import path literal.
//
// Spelled as a helper rather than importing `strconv` at the top because the file's only
// other parser need is `go/parser`, and one more import for one call would be noise.
func strconvUnquote(literal string) (string, error) {
	if len(literal) < 2 || literal[0] != '"' || literal[len(literal)-1] != '"' {
		return "", errUnquotedImportPath
	}
	return literal[1 : len(literal)-1], nil
}

type importPathError struct{}

func (importPathError) Error() string { return "import path is not a quoted string" }

var errUnquotedImportPath = importPathError{}
