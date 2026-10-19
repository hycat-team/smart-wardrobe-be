package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

// configWithoutEmailBlock copies a config file with the top-level `email:`
// block (header line plus its indented body) removed, so `email.enabled` is
// genuinely ABSENT rather than absent-and-defaulted. That distinction is the
// whole point of these tests: 025 T113 asks for a default of true, and the only
// way to prove it is to load a file that does not mention the key.
func configWithoutEmailBlock(t *testing.T, source string) string {
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
		if strings.HasPrefix(line, "email:") {
			skipping = true
			continue
		}
		kept = append(kept, line)
	}

	return writeTempConfig(t, strings.Join(kept, "\n"))
}

// configWithEmailEnabled copies a config file whose email block carries exactly
// one key, so the assertion is about the loader reading `enabled` and nothing
// else.
func configWithEmailEnabled(t *testing.T, source, value string) string {
	t.Helper()
	stripped := configWithoutEmailBlock(t, source)
	raw, err := os.ReadFile(stripped)
	if err != nil {
		t.Fatal(err)
	}
	body := strings.TrimRight(string(raw), "\n") + "\nemail:\n    enabled: " + value + "\n"
	return writeTempConfig(t, body)
}

func writeTempConfig(t *testing.T, body string) string {
	t.Helper()
	target := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(target, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return target
}

// TestEmailEnabledDefaultsToTrueWhenTheKeyIsAbsent is the production-safety
// property of 025 T113, asserted against the real files rather than a
// hand-built struct.
//
// configs/config.production.yaml is the decisive case: it was written before
// this key existed, carries the live launch-2026-10 campaign, and this task is
// forbidden from editing it. If the default were Go's zero value, upgrading to
// a build that reads this key would stop every production email without a single
// error anywhere — a silent loss of mail, not a boot failure.
func TestEmailEnabledDefaultsToTrueWhenTheKeyIsAbsent(t *testing.T) {
	local, prod := repoConfigs(t)

	cases := []struct {
		name   string
		config string
	}{
		{"production config file with the whole email block removed", configWithoutEmailBlock(t, prod)},
		{"local config file with the whole email block removed", configWithoutEmailBlock(t, local)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			baseEnv(t)
			t.Setenv("CONFIG_FILE", tc.config)
			t.Setenv("SERVER_ENV", "local")

			cfg, err := loadConfig()
			if err != nil {
				t.Fatalf("a config without email.enabled must still load: %v", err)
			}
			if !cfg.Email.Enabled {
				t.Error("email.enabled must default to true when the key is absent; " +
					"the zero value of a Go bool means every config file written before " +
					"this key existed would silently stop sending email")
			}
		})
	}
}

// TestEmailEnabledReadsTheValueInTheYaml proves the other direction: the default
// must not be sticky. An explicit `enabled: false` is a value somebody wrote,
// and it has to win — otherwise the switch would be impossible to use.
func TestEmailEnabledReadsTheValueInTheYaml(t *testing.T) {
	local, _ := repoConfigs(t)

	cases := []struct {
		value string
		want  bool
	}{
		{"false", false},
		{"true", true},
	}

	for _, tc := range cases {
		t.Run("enabled: "+tc.value, func(t *testing.T) {
			baseEnv(t)
			t.Setenv("CONFIG_FILE", configWithEmailEnabled(t, local, tc.value))
			t.Setenv("SERVER_ENV", "local")

			cfg, err := loadConfig()
			if err != nil {
				t.Fatalf("must load: %v", err)
			}
			if cfg.Email.Enabled != tc.want {
				t.Errorf("email.enabled = %v, want %v", cfg.Email.Enabled, tc.want)
			}
		})
	}
}

// TestEmailEnabledEnvOverride keeps the switch operable where the production
// config file must not be edited (constitution §VIII): EMAIL_ENABLED wins over
// the YAML, and an unset variable leaves the loaded value alone.
func TestEmailEnabledEnvOverride(t *testing.T) {
	local, _ := repoConfigs(t)
	disabled := configWithEmailEnabled(t, local, "true")

	t.Run("false disables sending", func(t *testing.T) {
		baseEnv(t)
		t.Setenv("CONFIG_FILE", disabled)
		t.Setenv("SERVER_ENV", "local")
		t.Setenv("EMAIL_ENABLED", "false")

		cfg, err := loadConfig()
		if err != nil {
			t.Fatalf("must load: %v", err)
		}
		if cfg.Email.Enabled {
			t.Error("EMAIL_ENABLED=false must win over `enabled: true`")
		}
	})

	t.Run("unset leaves the yaml value alone", func(t *testing.T) {
		baseEnv(t)
		t.Setenv("CONFIG_FILE", disabled)
		t.Setenv("SERVER_ENV", "local")
		t.Setenv("EMAIL_ENABLED", "")

		cfg, err := loadConfig()
		if err != nil {
			t.Fatalf("must load: %v", err)
		}
		if !cfg.Email.Enabled {
			t.Error("an empty EMAIL_ENABLED must not override the yaml value")
		}
	})
}

// TestShippedLocalConfigKeepsEmailEnabled writes the development path out
// explicitly (T113 allows editing configs/config.yaml, and forbids editing the
// production one), so a bulk verification run has an obvious line to flip and
// the development path provably behaves as it did before the key existed.
func TestShippedLocalConfigKeepsEmailEnabled(t *testing.T) {
	local, _ := repoConfigs(t)
	baseEnv(t)
	t.Setenv("CONFIG_FILE", local)
	t.Setenv("SERVER_ENV", "local")

	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("local profile must load: %v", err)
	}
	if !cfg.Email.Enabled {
		t.Error("configs/config.yaml must ship email.enabled: true")
	}
}

// TestProductionConfigStatesTheEmailSwitchExplicitly is 026 D-05's guard. The
// production file used to inherit the loader default (missing key = send), which
// is correct behaviour but invisible configuration: an operator reading the file
// could not see that production sends real email by default, and the production
// template did not mention the switch at all. Both are now stated.
func TestProductionConfigStatesTheEmailSwitchExplicitly(t *testing.T) {
	_, prod := repoConfigs(t)

	raw, err := os.ReadFile(prod)
	if err != nil {
		t.Fatal(err)
	}
	v := newYamlReader(t, raw)
	if !v.IsSet("email.enabled") {
		t.Fatal("configs/config.production.yaml must declare `email.enabled` explicitly; " +
			"a missing key means send, which is the answer a reader cannot see")
	}
	if !v.GetBool("email.enabled") {
		t.Fatal("configs/config.production.yaml must ship `email.enabled: true`")
	}
}

// TestEveryEnvExampleMentionsTheEmailSwitch keeps the switch discoverable from
// the templates an operator actually copies. .env.production.example is the one
// that matters: without the line, a bulk verification run on production has no
// visible way to turn sending off.
func TestEveryEnvExampleMentionsTheEmailSwitch(t *testing.T) {
	root := repoRoot(t)

	templates, err := filepath.Glob(filepath.Join(root, ".env*.example"))
	if err != nil {
		t.Fatal(err)
	}
	if len(templates) == 0 {
		t.Fatal("no .env*.example templates found; the guard would pass vacuously")
	}

	for _, path := range templates {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), "EMAIL_ENABLED") {
			t.Errorf("%s must mention EMAIL_ENABLED: the production default is `send`, "+
				"so the switch has to be visible without reading source", filepath.Base(path))
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func newYamlReader(t *testing.T, raw []byte) *viper.Viper {
	t.Helper()
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewBuffer(raw)); err != nil {
		t.Fatalf("must parse as YAML: %v", err)
	}
	return v
}
