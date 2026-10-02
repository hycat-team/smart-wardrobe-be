package campaign

import (
	"bytes"
	"context"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
	"testing"
	"time"

	identitydto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	sharedconfig "smart-wardrobe-be/shared/config"
)

// 025 T095-T096: the administrator exclusion on the RECEIVING side (US6).
//
// T094 already put the rule on the message path (`admission_uc.go`, condition 3,
// with its own reason code `created_by_admin`). What is left is everything the
// other side of the mechanism has to agree with: the catch-up sweep, which learns
// the same fact from a database column rather than from an event, and the proof
// that the two cannot drift apart.
//
// Its own file, and not appended to `campaign_sweep_test.go`, because these are
// about the EXCLUSION rather than about the sweep's ordering, budget or health.

// TestCampaignSweepExcludesAdministratorAccountsWithoutConsumingBudget is T095.
//
// WHAT THE SWEEP OWNS, precisely. The account's exclusion has TWO layers here, and
// this test covers the one the sweep is responsible for. `payloadFor` copies
// `created_by_admin` into the payload it rebuilds, so `ProcessUserCreated` would
// refuse the account even if the sweep forwarded it — deleting the sweep's own
// filter costs a transaction and an advisory lock per administrator account, not
// a customer's slot. What the sweep's OWN filter buys is that the account never
// reaches a transaction at all, and that the run's report says SKIPPED rather than
// FORWARDED, which is the difference between "this account does not apply" and
// "this account was handed on and something downstream decided".
//
// So the assertions are on the LEDGER and on the REPORT, and the budget is the
// regression net underneath both: quota = 1, reserve = 0, exactly one grantable
// slot in the whole campaign, and the administrator's account is the OLDER of the
// two so it is evaluated first. That is the arrangement in which any leak of the
// flag — in `payloadFor`, in the shared rule, or in the sweep's filter — costs a
// real customer the only slot they were ever going to get, and it costs them
// SILENTLY: the operator sees a full campaign and a healthy sweep.
//
// It is worth stating plainly that T102 already proves an administrator's account
// leaves the sweep with no claim and no plan. What is new here is the sweep's own
// filter: that the account is dropped before the shared path is asked, and that it
// is reported as skipped rather than forwarded.
func TestCampaignSweepExcludesAdministratorAccountsWithoutConsumingBudget(t *testing.T) {
	const code = "launch-2026-10"
	registeredAt := baseInstant().Add(-20 * time.Minute)
	startsAt := registeredAt.Add(-time.Hour)
	// ONE slot. This is what makes the arrangement unforgiving.
	campaigns := []sharedconfig.CampaignConfig{
		testCampaignConfig(code, "premium-monthly", 1, 0, &startsAt, nil),
	}

	// Oldest first, so the sweep meets the administrator's account before the
	// customer's.
	adminAccount := sweepAccount(registeredAt, true)
	customer := sweepAccount(registeredAt.Add(time.Minute), false)
	h, _, sweep := newTestSweep(t, campaigns, []*identitydto.UserBriefContractDTO{adminAccount, customer})

	if got := h.claims.total(); got != 0 {
		t.Fatalf("the ledger must start empty: %d claims present before the sweep", got)
	}

	report, err := sweep.Sweep(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The report comes first, because it is the SWEEP's own claim about what it
	// did, and every ledger assertion below is only meaningful once it agrees.
	// `Forwarded == 1` is the load-bearing one: it proves the administrator's
	// account never reached the shared path, so no transaction, no advisory lock
	// and no quota decision was spent on it.
	if report.Scanned != 2 {
		t.Errorf("expected the sweep to scan 2 accounts, got %d", report.Scanned)
	}
	if report.Skipped != 1 {
		t.Errorf("expected 1 account skipped (the administrator's), got %d — the sweep must apply the "+
			"exclusion ITSELF rather than hand the account to the shared path and let it decide", report.Skipped)
	}
	if report.Forwarded != 1 {
		t.Errorf("expected 1 account forwarded to the shared path, got %d", report.Forwarded)
	}
	if report.Failed != 0 {
		t.Errorf("expected 0 failures, got %d", report.Failed)
	}

	// The budget. One row, from the one account that was entitled to it.
	if got := h.claims.countBucket(code, entities.GrantBucketMain); got != 1 {
		t.Fatalf("main grants = %d, want exactly 1 — with quota 1, any leak of the administrator flag costs a "+
			"real customer the only slot in the campaign (FR-050)", got)
	}
	claim := h.claims.claimFor(customer.ID)
	if claim == nil {
		t.Fatalf("the real customer was not granted. A full campaign and a healthy sweep is all an operator " +
			"would have seen")
	}
	if claim.CampaignCode != code || claim.PlanSlug != "premium-monthly" {
		t.Errorf("the surviving claim names campaign %q plan %q, want %q / premium-monthly",
			claim.CampaignCode, claim.PlanSlug, code)
	}
	sub, err := h.subs.GetByUserID(context.Background(), customer.ID)
	if err != nil || sub == nil {
		t.Fatalf("read the customer's subscription: %v", err)
	}
	if sub.CurrentPlanCode != "premium-monthly" {
		t.Fatalf("expected the customer to be granted premium-monthly, got %q", sub.CurrentPlanCode)
	}

	// The administrator's account: nothing written, nothing applied, nothing counted
	// against the ledger.
	if got := h.claims.claimFor(adminAccount.ID); got != nil {
		t.Fatalf("an account created by an administrator must not be granted (FR-091), got claim %#v", got)
	}
	adminSub, err := h.subs.GetByUserID(context.Background(), adminAccount.ID)
	if err != nil {
		t.Fatalf("read the administrator's subscription: %v", err)
	}
	if adminSub != nil {
		t.Fatalf("the administrator's account must be left exactly as it was (FR-091), got plan %q",
			adminSub.CurrentPlanCode)
	}
	if got := h.claims.total(); got != 1 {
		t.Fatalf("expected exactly 1 claim in the whole campaign, got %d", got)
	}

	// One grant writes one plan-change event. Two would let the AI service apply the
	// entitlement twice, and the administrator's account must produce none at all.
	rows := h.events.published()
	if len(rows) != 1 {
		t.Fatalf("expected exactly 1 plan-change row, got %d — an administrator's account must publish none", len(rows))
	}
	if rows[0].Event.UserID != customer.ID {
		t.Fatalf("the plan-change row belongs to %s, want the customer %s", rows[0].Event.UserID, customer.ID)
	}
}

// TestAdministratorExclusionIsDefinedOnceAndSharedByEveryPath is T096 / FR-080.
//
// FR-080 is a structural requirement: one account, two doors — settled by the
// message or recovered by the sweep — and both must be decided by the SAME rule.
// Two copies of the condition would satisfy every behavioural test in this
// package. `TestCampaignAdmissionSkipsAdministratorCreatedAccount` exercises the
// message path and `TestCampaignSweepExcludesAdministratorAccountsWithoutConsumingBudget`
// exercises the sweep, and a drift edit — one path's `if` widened, or its reason
// code respelled — would leave BOTH green, because each still passes on the input
// its own test hands it.
//
// So the property is asserted on the SOURCE, through the AST rather than through
// substrings. Four claims, each of which a plausible edit can break on its own:
//
//  1. `adminSkipReason` is DECLARED exactly once. Two declarations would be two
//     rules wearing the same name.
//  2. The message path calls it, with the flag it has — the payload's.
//  3. The sweep calls it, with the flag IT has — the column's, through the
//     account contract. Asserting the ARGUMENT matters as much as the call: a
//     sweep that called the shared function with a hardcoded `false` would satisfy
//     "both paths use one function" and exclude nobody.
//  4. NO production `if` OUTSIDE the shared function tests the flag itself. This
//     is the claim that actually forbids drift, and it is the one a behavioural
//     test cannot make. Inline `if account.CreatedByAdmin { ... }` is precisely
//     the second rule FR-080 forbids, and it would compile, pass every test here,
//     and put the two doors one edit apart from disagreeing. The function's own
//     body is exempt — its `if createdByAdmin` IS the rule, not a copy of it — and
//     so is `if reason := adminSkipReason(flag); reason != ""`, which is the shape
//     the message path is written in and must stay writable in.
//
// A fifth, smaller claim pins the reason CODE to one place, because two spellings
// of `created_by_admin` would satisfy claims 1 to 4 and still leave an operator
// reading two different strings in two logs.
func TestAdministratorExclusionIsDefinedOnceAndSharedByEveryPath(t *testing.T) {
	sources := campaignSourceFiles(t)
	if len(sources) == 0 {
		t.Fatal("no campaign source files found; the guard would pass vacuously")
	}

	var declarations []string
	callsByFile := map[string][]string{}
	var inlineTests []string
	var codeWriters []string

	for _, file := range sources {
		parsed, err := parser.ParseFile(token.NewFileSet(), file.name, file.body, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", file.name, err)
		}

		// The reason code is written once, in `skip_reason.go`. Any other file
		// spelling it is a second vocabulary for one condition.
		if strings.Contains(file.body, `"created_by_admin"`) {
			codeWriters = append(codeWriters, file.name)
		}

		ast.Inspect(parsed, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.FuncDecl:
				if typed.Recv == nil && typed.Name.Name == "adminSkipReason" {
					declarations = append(declarations, file.name)
					// The rule's OWN body is the one place the flag is tested — that
					// test IS the rule — so the walk does not descend into it. Every
					// other `if createdByAdmin` in the package is a second rule.
					return false
				}
			case *ast.CallExpr:
				ident, ok := typed.Fun.(*ast.Ident)
				if !ok || ident.Name != "adminSkipReason" {
					return true
				}
				if len(typed.Args) != 1 {
					t.Errorf("%s calls adminSkipReason with %d arguments; the shared rule takes the one flag "+
						"both paths derive from their own source (FR-080)", file.name, len(typed.Args))
					return true
				}
				callsByFile[file.name] = append(callsByFile[file.name], renderExpr(t, typed.Args[0]))
			case *ast.IfStmt:
				if typed.Cond == nil {
					return true
				}
				rendered := renderExpr(t, typed.Cond)
				// `if reason := adminSkipReason(flag); reason != ""` IS the shared rule,
				// not a second copy of it — the message path is written that way and
				// must stay writable that way.
				if strings.Contains(rendered, "adminSkipReason(") {
					return true
				}
				if strings.Contains(rendered, "CreatedByAdmin") || strings.Contains(rendered, "createdByAdmin") {
					inlineTests = append(inlineTests, file.name+": if "+rendered)
				}
			}
			return true
		})
	}

	if len(declarations) != 1 {
		t.Errorf("FR-080: the administrator exclusion must be ONE function; found %d declarations of "+
			"adminSkipReason in %v. Two declarations are two rules wearing one name, and nothing but this "+
			"test can see the difference", len(declarations), declarations)
	}

	assertCallArgument(t, callsByFile, "admission_uc.go", "createdByAdmin",
		"the message path must decide through the shared function, using the flag the PAYLOAD carries")
	assertCallArgument(t, callsByFile, "sweep_uc.go", "account.CreatedByAdmin",
		"the catch-up sweep must decide through the shared function, using the created_by_admin COLUMN it "+
			"reads from the account contract (FR-080, FR-091)")

	if len(inlineTests) > 0 {
		t.Errorf("FR-080: no production `if` in the campaign package may test the administrator flag itself. "+
			"The two doors must reach the rule through adminSkipReason, or they can drift apart and nothing "+
			"will report it:\n  %s", strings.Join(inlineTests, "\n  "))
	}

	if len(codeWriters) != 1 || codeWriters[0] != "skip_reason.go" {
		t.Errorf("FR-061/FR-091: the reason code \"created_by_admin\" must be written in skip_reason.go only, "+
			"found it in %v. Two spellings of one condition leave an operator reading two strings in two logs",
			codeWriters)
	}
}

// assertCallArgument states that one path decides through the shared function,
// with the flag that path actually has.
func assertCallArgument(t *testing.T, callsByFile map[string][]string, file, want, why string) {
	t.Helper()
	arguments, called := callsByFile[file]
	if !called {
		t.Errorf("%s never calls adminSkipReason; %s. A path that decides on its own is the drift FR-080 "+
			"forbids, and it would still pass every behavioural test here", file, why)
		return
	}
	for _, argument := range arguments {
		if argument == want {
			return
		}
	}
	t.Errorf("%s calls adminSkipReason(%s), want adminSkipReason(%s); %s",
		file, strings.Join(arguments, "), adminSkipReason("), want, why)
}

// renderExpr prints one expression back as source, so a condition can be searched
// for the identifier it names without matching the prose around it.
//
// Go's printer rather than a substring search of the file, because the question is
// "does this CONDITION test the flag", and a substring search of the file cannot
// tell a condition from a sentence explaining one. A guard that cries wolf over
// comments is a guard that gets deleted.
func renderExpr(t *testing.T, node ast.Node) string {
	t.Helper()
	var out bytes.Buffer
	if err := format.Node(&out, token.NewFileSet(), node); err != nil {
		t.Fatalf("render expression: %v", err)
	}
	return out.String()
}
