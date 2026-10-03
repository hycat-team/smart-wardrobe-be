package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// 026 T025 (FR-003, SC-010): loading configuration performs ZERO database queries
// and the package knows nothing that could perform one.
//
// WHY A SOURCE SCAN AND NOT ONLY A BEHAVIOURAL TEST. "Loading configuration opens
// no connection" is observable — load a config whose `DB_HOST` points nowhere and
// see whether it still succeeds. What is NOT observable is the reason it succeeds,
// and the reason is what erodes: a helper that opens a connection LAZILY, or a
// validation function that grows a "just check the catalogue" lookup, would leave
// the behavioural test green on a machine where the host happens to be reachable
// and green on a laptop, and would violate constitution §I/§VIII in production
// where the rule is absolute.
//
// The two halves therefore answer different questions and both are needed:
//
//	imports — can this package reach a database AT ALL?
//	behaviour — does loading a shipped config with an unroutable DB_HOST succeed?
//
// Before 026 the first half was FALSE in spirit rather than in fact: the package
// could not import a driver, but it held a `campaigns` list whose entries were
// validated at startup and whose `plan_slug` could not be checked — the one rule
// that needed the database had to be deferred to admission time precisely because
// of that. With the registry in the database there is nothing to validate here at
// all, and the boundary is worth pinning.

// forbiddenConfigImports are the packages that could open a database connection.
//
// The list names DRIVERS as well as the standard library, because "shared/config
// does not import gorm" is not the same claim as "shared/config cannot reach
// Postgres": a `lib/pq` or `pgx` import would satisfy the first sentence and break
// the second.
var forbiddenConfigImports = []string{
	"gorm.io/gorm",
	"database/sql",
	"github.com/jackc/pgx",
	"github.com/lib/pq",
	"github.com/prisma/prisma-client-go",
	"smart-wardrobe-be/shared/infrastructure/db",
	"smart-wardrobe-be/services",
}

// TestConfigPackageCannotReachADatabase reads the package's own source.
//
// Test files are included deliberately: a test that opens a connection would make
// the suite need a database, which is exactly the coupling 025's
// `campaign_validation_test.go` had and which T011 removed.
func TestConfigPackageCannotReachADatabase(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package directory: %v", err)
	}

	scanned := 0
	var hits []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" {
			continue
		}
		// This file is skipped because it NAMES the forbidden imports in order to
		// search for them, and a guard that fails on itself is a guard somebody
		// deletes. The exclusion is by exact file name rather than by "_test.go",
		// so every OTHER test file — including the ones 025 left behind — is still
		// covered, which is where a stray import would actually appear.
		if name == "config_no_database_test.go" {
			continue
		}
		scanned++
		body, readErr := os.ReadFile(name)
		if readErr != nil {
			t.Fatalf("read %s: %v", name, readErr)
		}
		for _, forbidden := range forbiddenConfigImports {
			if strings.Contains(string(body), `"`+forbidden+`"`) {
				hits = append(hits, name+": "+forbidden)
			}
		}
	}

	if scanned == 0 {
		t.Fatal("no Go file found; the scan would pass vacuously")
	}
	if len(hits) > 0 {
		t.Errorf("FR-003/SC-010: shared/config must not be able to reach a database. "+
			"These imports would give it that power: %v.\n"+
			"Loading configuration MUST NOT touch the database (constitution §I/§VIII); "+
			"campaign data is read at RUN time through the repository instead.",
			hits)
	}
}

// TestLoadingAShippedConfigOpensNoConnection is the behavioural half.
//
// `DB_HOST` points at an address in TEST-NET-1, which by RFC 5737 can never be
// dialled, so a connection attempt cannot succeed — it can only hang and time out,
// or be refused. The load completing at all is therefore evidence that nothing
// dialled anything.
func TestLoadingAShippedConfigOpensNoConnection(t *testing.T) {
	local, prod := repoConfigs(t)

	for _, path := range []string{local, prod} {
		baseEnv(t)
		t.Setenv("CONFIG_FILE", path)
		t.Setenv("SERVER_ENV", "local")
		// 192.0.2.0/24 — TEST-NET-1, reserved for documentation and unroutable.
		t.Setenv("DB_HOST", "192.0.2.1")
		t.Setenv("DB_LISTENER_HOST", "192.0.2.1")

		cfg, err := loadConfig()
		if err != nil {
			t.Fatalf("%s must load with an undiallable database host: %v", path, err)
		}
		if cfg.Database.Host != "192.0.2.1" {
			t.Fatalf("%s: DB_HOST = %q — the value must be READ and NOT dialled", path, cfg.Database.Host)
		}
		// The sweep cadence is the only campaign-shaped thing left in configuration,
		// and it must still be there and still be readable (FR-004).
		if cfg.CampaignSweep.IntervalSeconds < DefaultCampaignSweepIntervalSeconds {
			t.Errorf("%s: campaign_sweep.interval_seconds = %d, want at least the floor %d",
				path, cfg.CampaignSweep.IntervalSeconds, DefaultCampaignSweepIntervalSeconds)
		}
	}
}

// TestConfigStructHasNoCampaignList is the structural half of FR-001/FR-002.
//
// A field is what a second source of truth looks like before anybody writes code to
// populate it, and the population is the easy part — the decision not to read the
// registry from configuration is the hard one. Reflecting over the struct turns
// that decision into something a test can fail on, so re-adding `Campaigns
// []CampaignConfig` is a red build rather than a code-review opinion.
func TestConfigStructHasNoCampaignList(t *testing.T) {
	// A compile-time reference to the type that must no longer exist would be the
	// strongest form of this assertion, but referencing a deleted type does not
	// compile — so the reflection form is used instead, and it fails with a message
	// naming what came back.
	configType := reflect.TypeOf(Config{})
	if field, found := configType.FieldByName("Campaigns"); found {
		t.Errorf("FR-001/FR-002: Config.Campaigns is back as %s. The campaign registry lives in "+
			"`subscription.campaigns` and is read at RUN time; a list in configuration is the "+
			"second source of truth 026 exists to remove (spec.md Câu 1)",
			field.Type.String())
	}
	// And the sweep cadence MUST still be there, or the catch-up would stop being
	// scheduled and the removal would have cost more than it saved.
	if _, found := configType.FieldByName("CampaignSweep"); !found {
		t.Error("FR-004: Config.CampaignSweep must survive — the sweep cadence is configuration, " +
			"and dropping it would stop the catch-up from being scheduled")
	}
}

// TestConfigLoaderDoesNotMentionTheCampaignRegistry closes the last door.
//
// A campaign could be read from configuration without a struct field at all — by
// reading the raw YAML. So the loader is scanned for the key by name.
func TestConfigLoaderDoesNotMentionTheCampaignRegistry(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package directory: %v", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		body, readErr := os.ReadFile(name)
		if readErr != nil {
			t.Fatalf("read %s: %v", name, readErr)
		}
		// `"campaigns"` as a quoted literal is what a `mapstructure` tag or a
		// `v.SetDefault` call would contain. `campaign_sweep` contains the substring
		// too, so the match excludes the longer key.
		if strings.Contains(string(body), `"campaigns"`) {
			t.Errorf("FR-001: %s mentions the `campaigns` key. The registry is not configuration "+
				"any more; a mention here is either a stale tag or the start of a second source "+
				"of truth", name)
		}
	}
}
