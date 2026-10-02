package config

import (
	"bytes"
	"os"
	"sort"
	"testing"

	"github.com/spf13/viper"
)

// 025 T106 (SC-014, T089): BOTH shipped config files declare a `campaigns` block,
// both parse, and they are deliberately NOT required to be identical.
//
// WHY NOT IDENTICAL — the part that matters, and the reason this is a test rather
// than a comment:
//
//   configs/config.yaml            carries TWO campaigns (launch + a sample)
//   configs/config.production.yaml carries ONE (launch only)
//
// FR-011 gives the account to the LATEST-STARTING candidate campaign. So a test
// that demanded the two files match would be a mechanism for pushing a dev
// fixture into production: the moment somebody added the sample to production to
// make such a test pass, the sample would start WONNING on FR-011 and take real
// signups away from `launch-2026-10` — with every check still green, because the
// sample is a perfectly valid campaign. That is the exact failure quickstart.md's
// safety gate is written about, and quickstart.md 1.2 still says "add the block
// to BOTH files", which is right for the block and wrong for its contents.
//
// So this test asserts presence and validity on both sides, and asserts the
// DIFFERENCE on purpose. It never compares the two lists for equality, and if a
// future edit makes them equal this test must be re-read, not silently updated.
//
// It must also NOT be "weakened" into only checking config.yaml: a file missing
// the block is the failure mode the requirement exists for, because viper's exact
// unmarshal treats an absent key as an empty list rather than an error, so the
// service would boot and silently grant nothing.

// shippedCampaignCodes loads one shipped config file and returns its campaign
// codes, sorted, so a failure message is stable.
func shippedCampaignCodes(t *testing.T, path string) []string {
	t.Helper()

	baseEnv(t)
	t.Setenv("CONFIG_FILE", path)
	t.Setenv("SERVER_ENV", "local")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("%s must load: %v", path, err)
	}
	if len(cfg.Campaigns) == 0 {
		t.Fatalf("%s declares no campaign. The loader treats an ABSENT `campaigns` key as an "+
			"empty list rather than an error, so the service would boot, every grant would be "+
			"skipped, and nothing anywhere would report it (SC-014)", path)
	}

	codes := make([]string, 0, len(cfg.Campaigns))
	for i, campaign := range cfg.Campaigns {
		if campaign.Code == "" {
			t.Errorf("%s campaigns[%d] has no code", path, i)
		}
		if campaign.StartsAtTime.IsZero() {
			t.Errorf("%s campaigns[%d] (%s): starts_at was not parsed into an instant (025 T007)", path, i, campaign.Code)
		}
		// An empty `seen`: the duplicate-code rule is a property of the WHOLE list,
		// which validateCampaigns already covers over every shipped file below.
		if err := validateCampaign(campaign, map[string]struct{}{}); err != nil {
			t.Errorf("%s campaigns[%d] (%s) is not a shape FR-010 accepts: %v", path, i, campaign.Code, err)
		}
		codes = append(codes, campaign.Code)
	}
	if err := validateCampaignSweep(cfg.CampaignSweep); err != nil {
		t.Errorf("%s: %v", path, err)
	}

	sort.Strings(codes)
	return codes
}

// yamlHasTopLevelKey reports whether the raw file carries the key at all, which
// is a different question from "the list is non-empty" and the one SC-014 asks.
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

func TestBothShippedConfigsDeclareAndParseTheCampaignsBlock(t *testing.T) {
	local, prod := repoConfigs(t)

	for _, path := range []string{local, prod} {
		if !yamlHasTopLevelKey(t, path, "campaigns") {
			t.Errorf("%s has no `campaigns` key (SC-014)", path)
		}
		if !yamlHasTopLevelKey(t, path, "campaign_sweep") {
			t.Errorf("%s has no `campaign_sweep` key; one sweep worker serves every campaign (FR-083)", path)
		}
		codes := shippedCampaignCodes(t, path)
		if len(codes) == 0 {
			t.Errorf("%s yielded no campaign", path)
		}
	}
}

// TestTheTwoConfigsDifferOnPurpose records WHY they differ, so the difference is
// never "fixed" by copying the dev list into production.
//
// The launch campaign must be in both files — that is the campaign customers are
// being sold right now. The dev-only sample must be in config.yaml and absent from
// production: FR-011 picks the latest STARTING campaign, so a sample declared in
// production would out-rank the launch campaign the moment its start date passed
// and would take real signups away from it, silently.
func TestTheTwoConfigsDifferOnPurpose(t *testing.T) {
	local, prod := repoConfigs(t)

	localCodes := shippedCampaignCodes(t, local)
	prodCodes := shippedCampaignCodes(t, prod)

	if len(localCodes) != 2 {
		t.Errorf("configs/config.yaml declares %d campaigns (%v), want 2: the launch campaign plus the "+
			"dev-only sample that is the whole of US5 (T089)", len(localCodes), localCodes)
	}
	if len(prodCodes) != 1 {
		t.Errorf("configs/config.production.yaml declares %d campaigns (%v), want exactly 1. "+
			"FR-011 gives the account to the LATEST-STARTING campaign, so a second campaign here "+
			"steals real signups from launch-2026-10 with every check still green "+
			"(see quickstart.md's safety gate)",
			len(prodCodes), prodCodes)
	}

	if !contains(prodCodes, "launch-2026-10") {
		t.Errorf("configs/config.production.yaml must declare launch-2026-10 — that is the campaign "+
			"customers were sold; got %v", prodCodes)
	}
	if !contains(localCodes, "launch-2026-10") {
		t.Errorf("configs/config.yaml must declare launch-2026-10 too; got %v", localCodes)
	}

	for _, code := range prodCodes {
		if code != "launch-2026-10" {
			t.Errorf("configs/config.production.yaml declares %q. Only the launch campaign belongs "+
				"there: a campaign declared in production that starts later wins FR-011 and takes "+
				"real signups away from launch-2026-10. Move the sample to configs/config.yaml "+
				"(T089, quickstart.md's safety gate)", code)
		}
	}
}

func contains(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}
