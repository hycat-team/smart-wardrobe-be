package main

import (
	"os"
	"path/filepath"
	"testing"
)

const testContract = "# Ownership Contract (test fixture)\n" +
	"\n" +
	"## 1. Canonical ownership map\n" +
	"\n" +
	"| Schema | Owner service | Migration tree |\n" +
	"|---|---|---|\n" +
	"| `identity`, `subscription` | account | `services/account/database/migrations/` |\n" +
	"| `wardrobe`, `fashion`, `brand`, `dashboard`, `community` | commerce | `services/commerce/database/migrations/` |\n" +
	"| `aiexecutor` | ai | `services/ai/database/migrations/` |\n" +
	"| `shared`, `public` | infra chung (account giữ tạm) | `services/account/database/migrations/` |\n" +
	"\n" +
	"## 2. Gate rules (FR-009)\n" +
	"\n" +
	"- **G-ACCT**: cây Account MUST NOT chứa `ALTER/CREATE/DROP` trên `aiexecutor.|wardrobe.|fashion.|brand.|dashboard.|community.`. Ngoại lệ: `REFERENCES identity.users(` (FK user xuyên schema được phép).\n" +
	"- **G-COMM**: cây Commerce MUST NOT chứa thay đổi trên `identity.|subscription.|aiexecutor.`. Ngoại lệ: `REFERENCES identity.users(`.\n" +
	"- **G-AI**: cây AI MUST NOT chứa thay đổi ngoài `aiexecutor.`. Không ngoại lệ.\n"

func writeTestContract(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ownership-contract.md")
	if err := os.WriteFile(path, []byte(testContract), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParseContractDerivesRulesFromFile(t *testing.T) {
	rules, err := parseContract(writeTestContract(t))
	if err != nil {
		t.Fatalf("parseContract: %v", err)
	}
	if len(rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(rules))
	}
	byID := map[string]rule{}
	for _, r := range rules {
		byID[r.id] = r
	}

	if got := len(byID["G-ACCT"].forbidden); got != 6 {
		t.Errorf("G-ACCT: expected 6 forbidden schemas, got %d (%v)", got, byID["G-ACCT"].forbidden)
	}
	if byID["G-ACCT"].tree != "services/account/database/migrations" {
		t.Errorf("G-ACCT: wrong tree %q", byID["G-ACCT"].tree)
	}
	if len(byID["G-ACCT"].exceptions) == 0 {
		t.Error("G-ACCT: expected the REFERENCES identity.users exception")
	}

	if got := len(byID["G-COMM"].forbidden); got != 3 {
		t.Errorf("G-COMM: expected 3 forbidden schemas, got %d (%v)", got, byID["G-COMM"].forbidden)
	}
	if len(byID["G-COMM"].exceptions) == 0 {
		t.Error("G-COMM: expected the REFERENCES identity.users exception")
	}

	ai := byID["G-AI"]
	if len(ai.exceptions) != 0 {
		t.Errorf("G-AI: expected no exception, got %v", ai.exceptions)
	}
	for _, f := range ai.forbidden {
		if f == "aiexecutor" {
			t.Error("G-AI: own schema must not be forbidden")
		}
	}
	// Universe is 10 schemas (identity, subscription, wardrobe, fashion,
	// brand, dashboard, community, aiexecutor, shared, public).
	if got := len(ai.forbidden); got != 9 {
		t.Errorf("G-AI: expected 9 forbidden schemas, got %d (%v)", got, ai.forbidden)
	}

	if got := len(grandfathered); got != 13 {
		t.Errorf("expected 13 grandfathered versions (11 account + 2 baselines), got %d", got)
	}
}

func TestParseContractIgnoresLaterMentions(t *testing.T) {
	// A later prose line mentioning a rule id (like the §3 transition
	// clause) must not clobber the §2 bullet the rule was parsed from.
	path := filepath.Join(t.TempDir(), "ownership-contract.md")
	body := testContract + "\n## 3. Transition\n\n- 11 file Account được miễn G-ACCT cho đúng các version đó.\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	rules, err := parseContract(path)
	if err != nil {
		t.Fatalf("parseContract: %v", err)
	}
	for _, r := range rules {
		if r.id == "G-ACCT" && len(r.forbidden) != 6 {
			t.Fatalf("G-ACCT clobbered by later mention: %v", r.forbidden)
		}
	}
}

func TestLineViolates(t *testing.T) {
	rules, err := parseContract(writeTestContract(t))
	if err != nil {
		t.Fatalf("parseContract: %v", err)
	}
	byID := map[string]rule{}
	for _, r := range rules {
		byID[r.id] = r
	}

	cases := []struct {
		name string
		rule rule
		code string
		want bool
	}{
		{"acct flags aiexecutor DDL", byID["G-ACCT"], "ALTER TABLE aiexecutor.ai_policy_grants ADD COLUMN x int;", true},
		{"acct flags wardrobe move", byID["G-ACCT"], "ALTER TABLE wardrobe.wardrobe_items SET SCHEMA public;", true},
		{"acct ignores own schema", byID["G-ACCT"], "ALTER TABLE subscription.subscription_plans ADD COLUMN x int;", false},
		{"acct strips line comments", byID["G-ACCT"], "-- (4) New table aiexecutor.ai_cost_policy_audit_logs.", false},
		{"acct needs a DDL keyword", byID["G-ACCT"], "SELECT * FROM aiexecutor.ai_policy_grants;", false},
		{"comm flags subscription DDL", byID["G-COMM"], "ALTER TABLE subscription.subscription_plans ADD COLUMN x int;", true},
		{"comm honours the user FK exception", byID["G-COMM"], "CREATE TABLE dashboard.s (user_id uuid REFERENCES identity.users(id));", false},
		{"comm ignores own schema", byID["G-COMM"], "CREATE TABLE dashboard.admin_daily (id uuid PRIMARY KEY);", false},
		{"ai flags public DDL", byID["G-AI"], "CREATE TABLE public.probe (id serial PRIMARY KEY);", true},
		{"ai flags identity DDL without exception", byID["G-AI"], "ALTER TABLE identity.users ADD COLUMN x int REFERENCES identity.users(id);", true},
		{"ai allows own schema", byID["G-AI"], "CREATE TABLE aiexecutor.t (id uuid PRIMARY KEY);", false},
	}
	for _, c := range cases {
		if _, ok := lineViolates(c.code, c.rule); ok != c.want {
			t.Errorf("%s: got violation=%v, want %v", c.name, ok, c.want)
		}
	}
}
