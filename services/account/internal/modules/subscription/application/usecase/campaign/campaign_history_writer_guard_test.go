package campaign

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// 025 T105 (FR-095, SC-008): ONE grant writes EXACTLY ONE plan-change history row.
//
// The behavioural half already exists — `TestCampaignGrantWritesExactlyOneHistoryRow`
// in `campaign_grant_test.go` counts the rows a single grant produces. That test
// answers "how many rows did THIS grant write". It cannot answer "could a second
// writer appear", which is the half that actually rots:
//
//   - a history row with no matching plan change is invisible in the admin
//     look-up, so an operator cannot answer "what changed this account";
//   - two rows for one grant means the AI service can apply the entitlement
//     TWICE, and a duplicated entitlement is a quota that costs money.
//
// Both are silent. Nothing errors, no queue dead-letters, and the campaign still
// reports success — the customer is either short an answer or over-entitled.
//
// So this file counts CALL SITES instead. It asserts that the campaign package
// reaches the history table through exactly one door — `PublishPlanChange`, the
// single publish point shared with every other plan change (FR-011) — and through
// exactly one plan application. A second writer added tomorrow, or a direct
// history write that bypasses the publish point, turns this red at the point of
// the edit.

// historyWriterCalls are the calls that can put a row into the plan-change
// history. Each must appear EXACTLY once across the package's production files.
var historyWriterCalls = map[string]int{
	// The apply workflow: writes the subscription, RETURNS the history row without
	// persisting it. Twice in one grant would mean two version bumps for one
	// change, and the term would be computed from a subscription that had already
	// moved on.
	"ApplySubscriptionPlanWithEvent": 1,
	// The single publish point: persists the history row and the outbox record in
	// the caller's transaction. Bypassing it is how three of the nine plan-change
	// paths ended up with no history at all (024).
	"PublishPlanChange": 1,
}

// directHistoryWrites are calls that would write a history row behind the publish
// point's back. None may appear in this package at all.
var directHistoryWrites = map[string]string{
	"CreateIfAbsent":                   "writes user_subscription_events directly",
	"CreateSubscriptionEvent":          "writes user_subscription_events directly",
	"WritePlanChangeHistory":           "writes user_subscription_events directly",
	"SavePlanChangeHistory":            "writes user_subscription_events directly",
	"InsertPlanChangeEvent":            "writes user_subscription_events directly",
	"IUserSubscriptionEventRepository": "the history repository has no business in the grant path",
}

// historySymbols are TYPES, not calls: the grant path must not even hold the
// history repository or the history entity. A struct field of that type is a
// dependency on writing history by hand, whether or not anything calls it yet.
//
// Note this is narrower than "the package imports no repository package" — it
// legitimately imports `ICampaignClaimRepository` and `IUserSubscriptionRepository`
// (the claim window, and the subscription row it re-reads). What it must not hold
// is the THIRD repository: the one whose only job is writing plan-change history.
var historySymbols = map[string]string{
	"IUserSubscriptionEventRepository": "the repository whose only job is writing plan-change history",
	"UserSubscriptionEventRepository":  "its implementation",
	"UserSubscriptionEvent":            "the history entity itself",
}

// countPlanHistoryWriters walks one production file and returns the call sites it
// finds, as "line: call" strings, plus every history type it merely NAMES.
// Comments are not parsed, so a file may explain itself in prose.
func countPlanHistoryWriters(path string) (found map[string][]string, named []string, err error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, nil, err
	}

	found = map[string][]string{}
	note := func(key, where, what string) {
		found[key] = append(found[key], where+": "+what)
	}

	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.CallExpr:
			var name string
			switch fn := n.Fun.(type) {
			case *ast.SelectorExpr:
				name = fn.Sel.Name
			case *ast.Ident:
				name = fn.Name
			}
			if name == "" {
				return true
			}
			if _, tracked := historyWriterCalls[name]; tracked {
				note(name, fset.Position(n.Pos()).String(), name)
			}
			if why, banned := directHistoryWrites[name]; banned {
				note("!"+name, fset.Position(n.Pos()).String(), name+" — "+why)
			}
		case *ast.Ident:
			if why, banned := historySymbols[n.Name]; banned {
				named = append(named, fset.Position(n.Pos()).String()+": "+n.Name+" — "+why)
			}
		}
		return true
	})
	return found, named, nil
}

func TestTheCampaignGrantWritesHistoryThroughExactlyOneDoor(t *testing.T) {
	perCall := map[string][]string{}
	var namedHistoryTypes []string

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Test files are excluded for the same reason as everywhere else in 025:
		// they name the forbidden calls in order to search for them.
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		found, named, err := countPlanHistoryWriters(path)
		if err != nil {
			return err
		}
		for name, sites := range found {
			perCall[name] = append(perCall[name], sites...)
		}
		namedHistoryTypes = append(namedHistoryTypes, named...)
		return nil
	})
	if err != nil {
		t.Fatalf("scanning the campaign package failed: %v", err)
	}

	for name, want := range historyWriterCalls {
		got := perCall[name]
		if len(got) != want {
			sort.Strings(got)
			t.Errorf("FR-095/SC-008: %s must have exactly %d call site in the campaign package, found %d: %v. "+
				"Two writes for one grant duplicate the AI entitlement; zero leave the grant invisible "+
				"in the admin look-up. Neither reports an error anywhere",
				name, want, len(got), got)
		}
	}

	for name := range directHistoryWrites {
		if sites := perCall["!"+name]; len(sites) > 0 {
			sort.Strings(sites)
			t.Errorf("FR-095: %s must not be called from the campaign package (%s): %v",
				name, directHistoryWrites[name], sites)
		}
	}

	if len(namedHistoryTypes) > 0 {
		sort.Strings(namedHistoryTypes)
		t.Errorf("FR-095: the grant path must reach the history only through PublishPlanChange, so it "+
			"must not even NAME the history repository or entity — a dependency on writing history by "+
			"hand is a dependency whether or not anything calls it yet: %v", namedHistoryTypes)
	}
}

// TestHistoryWriterScanActuallyCountsCallSites is the negative control: a guard
// that cannot count is indistinguishable from one that is looking in the wrong
// place.
//
// The first snippet is two apply calls and one publish; the second is two
// publishes and a hand-rolled history write. A text scan would also have matched
// the comments in the third snippet, which is why this counts call expressions.
func TestHistoryWriterScanActuallyCountsCallSites(t *testing.T) {
	dir := t.TempDir()

	twoApplies := filepath.Join(dir, "two_applies.go")
	if err := os.WriteFile(twoApplies, []byte(`package p
// ApplySubscriptionPlanWithEvent and PublishPlanChange are only names here.
func a(x int) {
	subscription.ApplySubscriptionPlanWithEvent(ctx, repo, id, plan, now)
	subscription.ApplySubscriptionPlanWithEvent(ctx, repo, id, plan, now)
	uc.PublishPlanChange(ctx, event, sub, plan)
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	found, _, err := countPlanHistoryWriters(twoApplies)
	if err != nil {
		t.Fatal(err)
	}
	if len(found["ApplySubscriptionPlanWithEvent"]) != 2 {
		t.Errorf("the scan found %d apply call sites in a snippet with 2; it is not counting call expressions",
			len(found["ApplySubscriptionPlanWithEvent"]))
	}
	if len(found["PublishPlanChange"]) != 1 {
		t.Errorf("expected the 1 publish call site to be counted, got %d", len(found["PublishPlanChange"]))
	}

	handRolled := filepath.Join(dir, "hand_rolled.go")
	if err := os.WriteFile(handRolled, []byte(`package p
func b() {
	uc.events.CreateIfAbsent(ctx, event)
	uc.PublishPlanChange(ctx, event, sub, plan)
	uc.PublishPlanChange(ctx, event, sub, plan)
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	found, _, err = countPlanHistoryWriters(handRolled)
	if err != nil {
		t.Fatal(err)
	}
	if len(found["!CreateIfAbsent"]) != 1 {
		t.Error("a hand-rolled history write must be reported as banned")
	}
	if len(found["PublishPlanChange"]) != 2 {
		t.Errorf("expected 2 publish call sites, got %d", len(found["PublishPlanChange"]))
	}

	commentOnly := filepath.Join(dir, "comment_only.go")
	if err := os.WriteFile(commentOnly, []byte(`package p
// ApplySubscriptionPlanWithEvent writes the plan; PublishPlanChange writes the history.
// Neither is called here.
func c() {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	found, _, err = countPlanHistoryWriters(commentOnly)
	if err != nil {
		t.Fatal(err)
	}
	if len(found["ApplySubscriptionPlanWithEvent"]) != 0 || len(found["PublishPlanChange"]) != 0 {
		t.Error("names in comments must not count as call sites, or every explanatory comment would fail the guard")
	}

	// Holding the history repository as a field is a dependency too, even with
	// nothing calling it yet.
	holdingRepo := filepath.Join(dir, "holding_repo.go")
	if err := os.WriteFile(holdingRepo, []byte(`package p
type UC struct {
	events repositories.IUserSubscriptionEventRepository
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, named, err := countPlanHistoryWriters(holdingRepo)
	if err != nil {
		t.Fatal(err)
	}
	if len(named) == 0 {
		t.Error("a held history repository must be reported even though nothing calls it")
	}
}
