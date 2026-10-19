// Command migration-ownership-check enforces the DDL ownership contract in
// specs/030-ddl-ownership-handover/contracts/ownership-contract.md (FR-009).
//
// It reads the schema lists from the contract itself (single source of truth:
// the §1 ownership table plus the G-ACCT/G-COMM/G-AI bullets in §2) and scans
// services/*/database/migrations/*.sql for cross-owner DDL. Every violation
// is reported as RULE file:line and the command exits non-zero.
//
// Scope notes (see the contract for the full text):
//   - Grandfathered versions: the 13 exact filenames rewritten or created by
//     the handover itself are skipped by (service, tree, version). See
//     contracts/ownership-contract.md §3 (transition clause) and §4
//     (contract versioning).
//   - Scanning is line-based: a line counts when it holds a DDL keyword
//     (ALTER/CREATE/DROP) next to a forbidden "<schema>." reference. SQL
//     "--" comments are stripped first (naive cut at the first "--"; string
//     literals containing "--" are not expected in migration files). A line
//     holding "REFERENCES identity.users(" is exempt where the rule allows
//     it (G-ACCT, G-COMM; G-AI has no exception).
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	backtickRe = regexp.MustCompile("`([^`]*)`")
	schemaRe   = regexp.MustCompile(`^[a-z][a-z0-9_]*\.$`)
	ddlRe      = regexp.MustCompile(`(?i)\b(ALTER|CREATE|DROP)\b`)
)

// Grandfathered versions, matched by exact filename — i.e. skip ONLY these
// (service, tree, version) triples, never whole patterns. The 11 rewritten
// Account contents plus the AI/Commerce baselines are exempted by
// contracts/ownership-contract.md §3 (transition clause: their new content
// already removed the offending DDL); §4 requires any later map change to go
// through its own spec with an ADR instead of silently editing this list.
var grandfathered = map[string]bool{
	// Account tree: 11 rewritten contents (filenames + versions kept).
	"20260622131631_add_ai_cost_policy.sql":                    true,
	"20260627143730_add_token_estimation_audit.sql":            true,
	"20260627225700_update_ai_cost_quota_limits.sql":           true,
	"20260629072235_lowercase_constants_and_constraints.sql":   true,
	"20260704062159_init_ai_executor_module.sql":               true,
	"20260723000001_create_dashboard_snapshots_tables.sql":     true,
	"20260724173038_fix_total_tokens_and_check_constraint.sql": true,
	"20260801000000_separate_module_schemas.sql":               true,
	"20260930120000_admin_premium_grant_schema.sql":            true,
	"20260930150000_free_default_outfit_attempt_cap.sql":       true,
	"20261011090002_add_payos_status_counts.sql":               true,
	// AI baseline (m=1) and Commerce baseline (k=1).
	"20261016213000_adopt_aiexecutor_ownership.sql": true,
	"20261016213001_adopt_dashboard_ownership.sql":  true,
}

type rule struct {
	id         string // G-ACCT, G-COMM, G-AI
	service    string // account, commerce, ai
	tree       string // repo-relative migration dir, parsed from contract §1
	forbidden  []string
	exceptions []string
}

type violation struct {
	ruleID string
	path   string
	line   int
	text   string
}

func main() {
	root := flag.String("root", ".", "repository root to scan")
	contract := flag.String("contract", "", "ownership contract path (default <root>/specs/030-ddl-ownership-handover/contracts/ownership-contract.md)")
	flag.Parse()

	contractPath := *contract
	if contractPath == "" {
		contractPath = filepath.Join(*root, "specs", "030-ddl-ownership-handover", "contracts", "ownership-contract.md")
	}

	rules, err := parseContract(contractPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "migration-ownership-check:", err)
		os.Exit(2)
	}

	var violations []violation
	scanned, skipped := 0, 0
	for _, r := range rules {
		dir := filepath.Join(*root, filepath.FromSlash(r.tree))
		entries, err := os.ReadDir(dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "migration-ownership-check: read", dir+":", err)
			os.Exit(2)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
				continue
			}
			rel := filepath.ToSlash(filepath.Join(r.tree, e.Name()))
			if grandfathered[e.Name()] {
				skipped++
				continue
			}
			scanned++
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				fmt.Fprintln(os.Stderr, "migration-ownership-check: read", rel+":", err)
				os.Exit(2)
			}
			for i, raw := range strings.Split(string(data), "\n") {
				if _, ok := lineViolates(stripComment(raw), r); ok {
					violations = append(violations, violation{
						ruleID: r.id,
						path:   rel,
						line:   i + 1,
						text:   truncate(strings.TrimSpace(raw), 160),
					})
				}
			}
		}
	}

	if len(violations) > 0 {
		for _, v := range violations {
			fmt.Printf("%s %s:%d: %s\n", v.ruleID, v.path, v.line, v.text)
		}
		fmt.Printf("migration-ownership-check: FAIL — %d violation(s) in %d scanned file(s) (%d grandfathered skipped)\n", len(violations), scanned, skipped)
		os.Exit(1)
	}
	fmt.Printf("migration-ownership-check: OK — %d migration file(s) scanned, %d grandfathered version(s) skipped, 0 violations\n", scanned, skipped)
}

// parseContract derives the three gate rules from the contract file: the
// migration tree per owner service comes from the §1 table, and each rule's
// forbidden schema list (plus its REFERENCES exception, if any) comes from
// that rule's §2 bullet. No schema list is hardcoded here. It fails closed:
// an unparsable contract or an empty rule is an operational error, never a
// silent pass.
func parseContract(path string) ([]rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read contract %s: %w", path, err)
	}

	trees := map[string]string{}
	universe := map[string]bool{}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		cols := strings.Split(trimmed, "|")
		if len(cols) < 4 {
			continue
		}
		for _, s := range schemasIn(cols[1]) {
			universe[s] = true
		}
		owner := strings.TrimSpace(cols[2])
		if owner != "account" && owner != "commerce" && owner != "ai" {
			continue
		}
		var tree string
		if m := backtickRe.FindStringSubmatch(cols[3]); m != nil {
			tree = strings.TrimSuffix(strings.TrimSpace(m[1]), "/")
		}
		if tree == "" {
			return nil, fmt.Errorf("contract §1: no migration tree for owner %q", owner)
		}
		trees[owner] = tree
	}

	services := []struct {
		id  string
		svc string
	}{
		{"G-ACCT", "account"},
		{"G-COMM", "commerce"},
		{"G-AI", "ai"},
	}
	bullets := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "- ") {
			continue
		}
		for _, s := range services {
			if _, done := bullets[s.id]; done {
				continue
			}
			if strings.Contains(trimmed, s.id) {
				bullets[s.id] = trimmed
			}
		}
	}

	var rules []rule
	for _, s := range services {
		tree, ok := trees[s.svc]
		if !ok {
			return nil, fmt.Errorf("contract §1: no migration tree for owner %q", s.svc)
		}
		bullet, ok := bullets[s.id]
		if !ok {
			return nil, fmt.Errorf("contract §2: no %s bullet found", s.id)
		}
		var listed, exceptions []string
		for _, m := range backtickRe.FindAllStringSubmatch(bullet, -1) {
			inner := strings.TrimSpace(m[1])
			if strings.Contains(inner, "REFERENCES") {
				exceptions = append(exceptions, inner)
				continue
			}
			for _, part := range strings.Split(inner, "|") {
				part = strings.TrimSpace(part)
				if schemaRe.MatchString(part) {
					listed = append(listed, strings.TrimSuffix(part, "."))
				}
			}
		}
		r := rule{id: s.id, service: s.svc, tree: tree, exceptions: exceptions}
		if s.id == "G-AI" {
			// "no change outside aiexecutor": everything in the §1 universe
			// except the owned schema is forbidden. No exception.
			for schema := range universe {
				if schema != "aiexecutor" {
					r.forbidden = append(r.forbidden, schema)
				}
			}
		} else {
			r.forbidden = listed
		}
		if len(r.forbidden) == 0 {
			return nil, fmt.Errorf("contract §2: %s parsed to an empty forbidden list", s.id)
		}
		for _, f := range r.forbidden {
			if !universe[f] {
				return nil, fmt.Errorf("contract: %s forbids unknown schema %q (not in §1 map)", s.id, f)
			}
		}
		rules = append(rules, r)
	}
	return rules, nil
}

// schemasIn extracts `schema` tokens from a §1 table cell.
func schemasIn(cell string) []string {
	var out []string
	for _, m := range backtickRe.FindAllStringSubmatch(cell, -1) {
		s := strings.TrimSpace(m[1])
		if schemaRe.MatchString(s + ".") {
			out = append(out, strings.TrimSuffix(s, "."))
		}
	}
	return out
}

// stripComment cuts a SQL "--" line comment. Naive by design (see package
// doc): migration files here do not embed "--" inside string literals.
func stripComment(line string) string {
	if i := strings.Index(line, "--"); i >= 0 {
		return line[:i]
	}
	return line
}

// lineViolates reports whether one (comment-stripped) line holds a DDL
// keyword next to a schema the rule forbids, honouring the rule's REFERENCES
// exception. It returns the offending schema for diagnostics.
func lineViolates(code string, r rule) (string, bool) {
	if !ddlRe.MatchString(code) {
		return "", false
	}
	low := strings.ToLower(code)
	for _, e := range r.exceptions {
		if strings.Contains(low, strings.ToLower(e)) {
			return "", false
		}
	}
	for _, f := range r.forbidden {
		if strings.Contains(low, f+".") {
			return f, true
		}
	}
	return "", false
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
