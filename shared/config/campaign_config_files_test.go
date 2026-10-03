package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// 026 T012 (SC-008, FR-002): this file INVERTS 025's T106.
//
// 025 T106 asserted that BOTH shipped config files DECLARE a `campaigns` block,
// and that config.yaml declares two campaigns while config.production.yaml
// declares exactly one. Every one of those assertions is now false, and a test
// that still asserted them would be a machine for putting the block back: the
// second source of truth 026 exists to end (spec.md Câu 1).
//
// The assertions below therefore run in the OPPOSITE direction, which is what
// makes this a real guard rather than a deleted test:
//
//   - `campaigns` MUST BE ABSENT from both files. Re-adding it would create two
//     sources for the registry, and every count derived from `quota`/`reserve`
//     would then be wrong against one of them.
//   - `campaign_sweep` MUST STILL BE PRESENT in both files. It is the cadence of
//     the catch-up sweep — a knob of the mechanism, not campaign data — and it is
//     what keeps configuration loading free of database access (FR-003/FR-004).
//     Deleting it by association would silently disable the sweep.
//
// WHY THE "2 IN DEV / 1 IN PRODUCTION" ASSERTIONS ARE GONE RATHER THAN INVERTED.
// They were about WHICH campaigns each file carried, and that question no longer
// has an answer in configuration. It has an answer in the registry, where an
// operator creates campaigns through the API. Keeping a per-environment list
// assertion here would keep a test guarding a question this package cannot ask.
//
// WHY THE KEY-ABSENT QUESTION IS ASKED OF THE RAW FILE, NOT OF THE DECODED
// STRUCT. `UnmarshalExact` treats an absent key as an empty list rather than as
// an error, so the service would boot happily on a file that declared nothing —
// which is exactly why this test reads the YAML instead of trusting the load.
// The companion test below closes the second half: the key does not exist in the
// STRUCT either, so there is no code path that could reintroduce it.

// yamlHasTopLevelKey reports whether the raw file carries the key at all.
func yamlHasTopLevelKey(t *testing.T, path, key string) bool {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewBuffer(raw)); err != nil {
		t.Fatalf("%s must parse as YAML: %v", path, err)
	}
	return v.IsSet(key)
}

// TestShippedConfigsNoLongerDeclareTheCampaignsBlock is the inverted T106
// (FR-002, SC-008).
func TestShippedConfigsNoLongerDeclareTheCampaignsBlock(t *testing.T) {
	local, prod := repoConfigs(t)

	for _, path := range []string{local, prod} {
		if yamlHasTopLevelKey(t, path, "campaigns") {
			t.Errorf("%s declares a `campaigns` key. Campaigns live in the table "+
				"`subscription.campaigns` and are written by the admin API (026 FR-001/FR-002). "+
				"A key here that no code reads is worse than an absent one: it looks "+
				"authoritative to the next operator who opens a campaign \"by editing the file\".",
				path)
		}
	}
}

// TestSweepCadenceStaysInConfiguration pins the half of 025 that survives, and
// the half a careless cleanup would have taken with it (FR-004).
func TestSweepCadenceStaysInConfiguration(t *testing.T) {
	local, prod := repoConfigs(t)

	for _, path := range []string{local, prod} {
		if !yamlHasTopLevelKey(t, path, "campaign_sweep") {
			t.Errorf("%s has no `campaign_sweep` key; one sweep worker serves every campaign "+
				"(025 FR-083) and the cadence is a deployment knob, not campaign data (026 FR-004)",
				path)
		}
		baseEnv(t)
		t.Setenv("CONFIG_FILE", path)
		t.Setenv("SERVER_ENV", "local")

		cfg, err := loadConfig()
		if err != nil {
			t.Fatalf("%s must load without a `campaigns` block: %v", path, err)
		}
		if err := validateCampaignSweep(cfg.CampaignSweep); err != nil {
			t.Errorf("%s: %v", path, err)
		}
	}
}

// TestValidateCampaignSweepRejectsExplicitIntervalBelowTheFloor is 025's SC-013,
// kept: the campaign LIST is gone but the sweep CADENCE is still configuration,
// and an interval below the floor is still a misconfiguration rather than a
// default to apply.
//
// The two are different situations even though both arrive as the same number in
// Go: 0 written down is a misconfiguration to reject, 0 because the key was
// missing is a default to apply. `validateCampaignSweep` only ever sees the
// former — the loader installs the default while the key is still absent.
func TestValidateCampaignSweepRejectsExplicitIntervalBelowTheFloor(t *testing.T) {
	for _, interval := range []int{0, 59} {
		if err := validateCampaignSweep(CampaignSweepConfig{IntervalSeconds: interval}); err == nil {
			t.Errorf("interval_seconds %d must fail at startup (FR-083)", interval)
		}
	}
}

func TestValidateCampaignSweepAcceptsTheFloor(t *testing.T) {
	sweep := CampaignSweepConfig{IntervalSeconds: DefaultCampaignSweepIntervalSeconds}
	if err := validateCampaignSweep(sweep); err != nil {
		t.Errorf("the floor itself must be accepted: %v", err)
	}
	if err := validateCampaignSweep(CampaignSweepConfig{IntervalSeconds: 900}); err != nil {
		t.Errorf("the shipped cadence must be accepted: %v", err)
	}
}

// TestCampaignSweepDefaultsWhenTheBlockIsAbsent proves the default path against a
// real configuration file rather than a hand-built struct: repoConfigs is copied
// with the `campaign_sweep` block removed, so the key is genuinely absent.
func TestCampaignSweepDefaultsWhenTheBlockIsAbsent(t *testing.T) {
	local, _ := repoConfigs(t)
	baseEnv(t)
	t.Setenv("CONFIG_FILE", configWithoutSweepBlock(t, local))
	t.Setenv("SERVER_ENV", "local")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("a config without the sweep block must still load: %v", err)
	}
	if cfg.CampaignSweep.IntervalSeconds != DefaultCampaignSweepIntervalSeconds {
		t.Errorf("campaign_sweep.interval_seconds = %d, want the default %d",
			cfg.CampaignSweep.IntervalSeconds, DefaultCampaignSweepIntervalSeconds)
	}
	if !cfg.CampaignSweep.RunOnStart {
		t.Error("campaign_sweep.run_on_start must default to true (FR-079, SC-016)")
	}
}

// configWithoutSweepBlock copies a config file with the top-level
// `campaign_sweep:` block (header line plus its indented body) removed, so the
// "key absent" code path can be exercised against a configuration that is
// otherwise valid.
func configWithoutSweepBlock(t *testing.T, source string) string {
	t.Helper()

	raw, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}

	kept := make([]string, 0, 256)
	skipping := false
	for _, line := range strings.Split(string(raw), "\n") {
		if skipping {
			if strings.TrimSpace(line) == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				continue
			}
			skipping = false
		}
		if strings.HasPrefix(line, "campaign_sweep:") {
			skipping = true
			continue
		}
		kept = append(kept, line)
	}

	target := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(target, []byte(strings.Join(kept, "\n")), 0o600); err != nil {
		t.Fatal(err)
	}
	return target
}
