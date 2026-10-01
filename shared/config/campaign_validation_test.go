package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// validCampaign is the shape data-model.md §4.1 accepts: a code, an existing plan
// slug, a main budget of 50 with a reserve of 5, and a zone-qualified start.
func validCampaign() CampaignConfig {
	return CampaignConfig{
		Code:     "launch-2026-10",
		PlanSlug: "premium-monthly",
		Quota:    50,
		Reserve:  5,
		StartsAt: "2026-10-02T00:00:00+07:00",
		EndsAt:   "2026-11-01T00:00:00+07:00",
	}
}

// 025 T014 (FR-010/FR-021, data-model.md §4.1): the seven invalid shapes must be
// refused at STARTUP.
//
// A campaign that boots but is wrong is worse than one that does not boot: the
// campaign mechanism is data-driven, so a typo in code, quota or the window is
// invisible until real accounts are silently granted nothing — or granted from a
// budget nobody intended.
func TestValidateCampaignsRejectsEveryInvalidShape(t *testing.T) {
	cases := []struct {
		name      string
		campaigns func() []CampaignConfig
	}{
		{
			name: "code empty",
			campaigns: func() []CampaignConfig {
				c := validCampaign()
				c.Code = "  "
				return []CampaignConfig{c}
			},
		},
		{
			name: "code outside the allowed alphabet",
			campaigns: func() []CampaignConfig {
				c := validCampaign()
				c.Code = "Launch_2026"
				return []CampaignConfig{c}
			},
		},
		{
			name: "code shorter than three characters",
			campaigns: func() []CampaignConfig {
				c := validCampaign()
				c.Code = "ab"
				return []CampaignConfig{c}
			},
		},
		{
			name: "code duplicated in the list",
			campaigns: func() []CampaignConfig {
				first := validCampaign()
				second := validCampaign()
				second.Quota = 10
				return []CampaignConfig{first, second}
			},
		},
		{
			name: "plan slug empty",
			campaigns: func() []CampaignConfig {
				c := validCampaign()
				c.PlanSlug = ""
				return []CampaignConfig{c}
			},
		},
		{
			name: "quota below one",
			campaigns: func() []CampaignConfig {
				c := validCampaign()
				c.Quota = 0
				return []CampaignConfig{c}
			},
		},
		{
			name: "reserve negative",
			campaigns: func() []CampaignConfig {
				c := validCampaign()
				c.Reserve = -1
				return []CampaignConfig{c}
			},
		},
		{
			name: "starts_at without an explicit time zone",
			campaigns: func() []CampaignConfig {
				c := validCampaign()
				c.StartsAt = "2026-10-02T00:00:00"
				c.EndsAt = ""
				return []CampaignConfig{c}
			},
		},
		{
			name: "ends_at not after starts_at",
			campaigns: func() []CampaignConfig {
				c := validCampaign()
				c.EndsAt = "2026-10-01T00:00:00+07:00"
				return []CampaignConfig{c}
			},
		},
		{
			name: "ends_at equal to starts_at",
			campaigns: func() []CampaignConfig {
				c := validCampaign()
				c.EndsAt = c.StartsAt
				return []CampaignConfig{c}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateCampaigns(tc.campaigns()); err == nil {
				t.Fatal("invalid campaign configuration must fail at startup")
			}
		})
	}
}

// TestValidateCampaignsAcceptsValidShapes keeps the guard from becoming a wall:
// the checks above must not reject a legitimate campaign.
func TestValidateCampaignsAcceptsValidShapes(t *testing.T) {
	t.Run("empty list means no campaign is configured", func(t *testing.T) {
		if err := validateCampaigns(nil); err != nil {
			t.Fatalf("an empty campaign list is a deployment choice, not an error: %v", err)
		}
	})

	t.Run("open ended campaign may omit ends_at", func(t *testing.T) {
		c := validCampaign()
		c.EndsAt = ""
		if err := validateCampaigns([]CampaignConfig{c}); err != nil {
			t.Fatalf("ends_at is optional (FR-009): %v", err)
		}
	})

	t.Run("zero reserve is a hard ceiling equal to quota", func(t *testing.T) {
		c := validCampaign()
		c.Reserve = 0
		if err := validateCampaigns([]CampaignConfig{c}); err != nil {
			t.Fatalf("reserve 0 means the ceiling is exactly quota (FR-021): %v", err)
		}
	})

	t.Run("several campaigns with distinct codes", func(t *testing.T) {
		first := validCampaign()
		second := validCampaign()
		second.Code = "launch-2026-11"
		second.Quota = 100
		if err := validateCampaigns([]CampaignConfig{first, second}); err != nil {
			t.Fatalf("two valid campaigns must load: %v", err)
		}
	})
}

// 025 T014 (FR-083, SC-013/SC-014): an explicit interval below the floor is
// refused; an ABSENT one falls back to the floor.
//
// The two are different situations even though both arrive as the same number in
// Go: 0 written down is a misconfiguration to reject, 0 because the key was
// missing is a default to apply. validateCampaignSweep only ever sees the former
// — the loader installs the default while the key is still absent.
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
// with both campaign blocks removed, so `campaign_sweep` is genuinely absent.
func TestCampaignSweepDefaultsWhenTheBlockIsAbsent(t *testing.T) {
	local, _ := repoConfigs(t)
	baseEnv(t)
	t.Setenv("CONFIG_FILE", configWithoutCampaignBlocks(t, local))
	t.Setenv("SERVER_ENV", "local")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("a config without campaign blocks must still load: %v", err)
	}
	if len(cfg.Campaigns) != 0 {
		t.Errorf("campaigns = %+v, want empty", cfg.Campaigns)
	}
	if cfg.CampaignSweep.IntervalSeconds != DefaultCampaignSweepIntervalSeconds {
		t.Errorf("campaign_sweep.interval_seconds = %d, want the default %d",
			cfg.CampaignSweep.IntervalSeconds, DefaultCampaignSweepIntervalSeconds)
	}
	if !cfg.CampaignSweep.RunOnStart {
		t.Error("campaign_sweep.run_on_start must default to true (FR-079, SC-016)")
	}
}

// TestShippedConfigCarriesTheCampaignBlocks guards T009/T010 at the same level:
// both config files must parse, and the sweep cadence must be readable.
func TestShippedConfigCarriesTheCampaignBlocks(t *testing.T) {
	local, prod := repoConfigs(t)
	for _, path := range []string{local, prod} {
		baseEnv(t)
		t.Setenv("CONFIG_FILE", path)
		t.Setenv("SERVER_ENV", "local")

		cfg, err := loadConfig()
		if err != nil {
			t.Fatalf("%s must load: %v", path, err)
		}
		if len(cfg.Campaigns) == 0 {
			t.Errorf("%s declares no campaign; the block is mandatory, not optional", path)
		}
		for i, campaign := range cfg.Campaigns {
			if campaign.StartsAtTime.IsZero() {
				t.Errorf("%s campaigns[%d]: starts_at was not parsed (025 T007)", path, i)
			}
		}
		if err := validateCampaignSweep(cfg.CampaignSweep); err != nil {
			t.Errorf("%s: %v", path, err)
		}
	}
}

// configWithoutCampaignBlocks copies a config file with the top-level
// `campaigns:` and `campaign_sweep:` blocks (header lines plus their indented
// bodies) removed, so the "key absent" code path can be exercised against a
// configuration that is otherwise valid.
func configWithoutCampaignBlocks(t *testing.T, source string) string {
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
		if strings.HasPrefix(line, "campaigns:") || strings.HasPrefix(line, "campaign_sweep:") {
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
