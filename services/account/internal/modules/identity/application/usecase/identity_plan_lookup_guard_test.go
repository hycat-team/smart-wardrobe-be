package usecase_test

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

// 025 T104 (FR-098): the account-creation module must not look up a PAID plan or
// a cost policy. `identity.user.created` carries the account id, the source door,
// the creation instant and the administrator flag — nothing else (FR-087).
//
// WHY THIS NEEDS A SCAN. The compiler cannot see it: reading a plan is ordinary
// code, and a function that fetched `premium-monthly` to decide what to put in
// the payload would compile, pass every behavioural test in this package, and pass
// FR-085 too (FR-085 only forbids reading the `campaigns` CONFIGURATION). Nor can
// a unit test that calls the doors it knows about — it cannot tell "no lookup
// happened" from "the lookup happened somewhere I did not look".
//
// THE FAILURE IS A COUPLING, NOT A LEAK. Each topic has exactly one owner:
// `identity.user.created` is owned by the identity module and carries account
// facts; `subscription.plan.changed` is owned by the subscription module and
// carries plan ids and cost policy ids. The moment the identity module reads a
// paid plan to enrich the event, one module knows about the other's business —
// and adding a second consumer of the event would then require editing this
// module, which is exactly what FR-090 forbids.
//
// The scan is over the AST, not over the raw text, and that is deliberate: this
// package's files discuss plans and campaigns in their COMMENTS constantly (they
// have to explain why they do not read them), so a text scan would either fail on
// its own documentation or force the needles to be weakened until they matched
// nothing. An AST sees identifiers, selectors and string literals — the three
// things a lookup actually consists of.

// paidPlanLookupIdents are identifiers that can only mean "somebody went and read
// a plan": repository types and the two functions the codebase resolves a plan
// through.
var paidPlanLookupIdents = map[string]string{
	"GetBySlugAnyStatus":            "resolves a plan row by slug, any status — the plan-catalog lookup",
	"LoadPlanForSubscription":       "resolves the plan behind a subscription row",
	"IPlanRepository":               "the plan catalogue's repository interface",
	"SubscriptionPlanRepository":    "its implementation",
	"SubscriptionPlan":              "the plan entity itself",
	"AICostPolicyID":                "the plan's AI cost policy",
	"AICostPolicyId":                "the plan's AI cost policy",
	"IUserSubscriptionProvisioning": "provisions a plan — see the note below on why this one is allowed",
}

// paidPlanLookupLiterals are raw strings that would only be written to reach the
// plan catalogue or a paid plan's identity. Matched case-insensitively on the
// literal's own text.
//
// NOTE ON WHAT IS NOT HERE, and it matters:
//
//   - "premium" is here, because a paid plan's slug written into source is a plan
//     lookup wearing a disguise.
//   - "free" is deliberately NOT here, and neither is the provisioning contract.
//     `EnsureDefaultSubscription` provisions the FREE default plan inside the
//     signup transaction and is required by 024 T196; the identity module not
//     having that would be the bug. FR-098 is about a PAID plan.
//   - `PlanSlug`/`PlanName` are NOT here. `mapper.MapToUserRes` copies them out of
//     a contract DTO into a user RESPONSE. That is not a lookup — the subscription
//     module already did the lookup — and banning the field would break a read
//     path that has nothing to do with the event payload.
var paidPlanLookupLiterals = []string{
	"subscription_plans",
	"current_plan_code",
	"ai_cost_policy_id",
	"plan_slug",
	"premium",
}

// forbiddenSubscriptionImports are the packages a plan lookup would have to come
// from. The identity module may import `.../subscription/application/errors` —
// classifying a provisioning failure is not reading a plan — and nothing else
// under the subscription module today.
var forbiddenSubscriptionImports = []string{
	"/modules/subscription/domain/repositories",
	"/modules/subscription/domain/entities",
	"/modules/subscription/infrastructure",
	"/modules/subscription/application/usecase",
	"/modules/subscription/application/mapper",
	"/subscription_contract.IPlanRepository",
}

// findPaidPlanLookups walks one Go file and returns every hit, as
// "line: what" strings. Comments are not parsed, so a file may discuss plans
// freely.
func findPaidPlanLookups(path string) ([]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}

	var hits []string
	record := func(pos token.Pos, what string) {
		hits = append(hits, fset.Position(pos).String()+": "+what)
	}

	for _, spec := range file.Imports {
		imported := strings.Trim(spec.Path.Value, `"`)
		for _, forbidden := range forbiddenSubscriptionImports {
			if strings.Contains(imported, forbidden) {
				record(spec.Pos(), "imports "+imported+" — a plan lookup lives behind that package")
			}
		}
	}

	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.Ident:
			if what, ok := paidPlanLookupIdents[n.Name]; ok && !allowedIdent(n.Name) {
				record(n.Pos(), n.Name+" — "+what)
			}
		case *ast.BasicLit:
			if n.Kind != token.STRING {
				return true
			}
			text := strings.ToLower(strings.Trim(n.Value, "`\""))
			for _, needle := range paidPlanLookupLiterals {
				if strings.Contains(text, needle) {
					record(n.Pos(), "string literal "+n.Value+" contains "+needle)
					break
				}
			}
		}
		return true
	})

	return hits, nil
}

// allowedIdent carves out the free-plan provisioning path, which 024 made
// mandatory inside the signup transaction and which is not a paid-plan lookup.
func allowedIdent(name string) bool {
	return strings.Contains(name, "Provisioning") && strings.Contains(name, "Subscription")
}

// TestIdentityModuleNeverLooksUpAPaidPlan is the FR-098 guard.
func TestIdentityModuleNeverLooksUpAPaidPlan(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "modules", "identity")
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("cannot locate the identity module at %s: %v", root, err)
	}

	scanned := 0
	var hits []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		// Test files are excluded: FR-098 is a rule about PRODUCTION code, and
		// this very test spells the forbidden identifiers out in order to search
		// for them. Including them would make the scan fail on itself, which
		// trains everyone to ignore it.
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		scanned++
		found, err := findPaidPlanLookups(path)
		if err != nil {
			return err
		}
		if len(found) > 0 {
			rel, _ := filepath.Rel(root, path)
			hits = append(hits, rel+"\n    "+strings.Join(found, "\n    "))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scanning the identity module failed: %v", err)
	}

	// A scan that visited nothing passes vacuously, and a vacuous guard is worse
	// than none: it looks like coverage.
	if scanned == 0 {
		t.Fatalf("the scan visited no production file at %s, so it can never fail", root)
	}

	if len(hits) > 0 {
		sort.Strings(hits)
		t.Errorf("FR-098: modules/identity must not look up a paid plan or a cost policy. "+
			"identity.user.created carries account facts only (FR-087); the plan and its cost policy "+
			"travel on subscription.plan.changed. Reading a plan here makes one module depend on the "+
			"other's business, and a second consumer of the event would then force an edit in this "+
			"module (FR-090).\nFound in:\n  %s",
			strings.Join(hits, "\n  "))
	}
}

// TestPaidPlanLookupScanActuallyCatchesALookup is the negative control.
//
// A guard that cannot fail is indistinguishable from a guard that is not looking
// for the right thing. Both snippets below are code that would satisfy every
// other test in this package: they compile, they do not touch the `campaigns`
// configuration (so FR-085's scan stays green), and they would produce a correct
// `identity.user.created` payload. They are caught here and nowhere else.
func TestPaidPlanLookupScanActuallyCatchesALookup(t *testing.T) {
	catching := map[string]string{
		"plan repository call": `package p
func grant(uc C) { plan, _ := uc.GetBySlugAnyStatus(ctx, "free"); _ = plan }`,
		"cost policy read": `package p
func payload(p *entities.SubscriptionPlan) string { return p.AICostPolicyID.String() }`,
		"paid plan slug literal": `package p
const slug = "premium-monthly"`,
		"raw plan table": `package p
const q = "SELECT slug FROM subscription_plans WHERE is_active"`,
		"forbidden import": `package p
import "smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"`,
	}
	for name, src := range catching {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "snippet.go")
			if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
				t.Fatal(err)
			}
			hits, err := findPaidPlanLookups(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(hits) == 0 {
				t.Errorf("the scan missed a snippet that looks up a plan:\n%s", src)
			}
		})
	}

	// And the half that must NOT trip it, or the guard becomes a wall nobody can
	// work around: provisioning the FREE default plan is required inside the
	// signup transaction (024 T196), and the user response carries plan fields
	// copied out of a contract DTO.
	allowed := map[string]string{
		"free default provisioning": `package p
func create(uc C, id uuid.UUID) error { return uc.EnsureDefaultSubscription(ctx, id) }`,
		"plan fields copied into a response": `package p
func toRes(sub *dto.UserSubscriptionContractDTO) dto.UserRes {
	return dto.UserRes{PlanSlug: sub.PlanSlug, PlanName: sub.PlanName}
}`,
		"a comment that discusses plans": "package p\n// The identity module must never read a paid plan.\nfunc f() {}",
	}
	for name, src := range allowed {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "snippet.go")
			if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
				t.Fatal(err)
			}
			hits, err := findPaidPlanLookups(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(hits) > 0 {
				t.Errorf("the scan flagged legitimate code: %v\n%s", hits, src)
			}
		})
	}
}
