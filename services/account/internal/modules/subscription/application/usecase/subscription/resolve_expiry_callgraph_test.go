package subscription

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

// 024 T153.
//
// The comment on `ResolveExpiryFor` used to claim it was "THE expiry rule for every
// path that changes a plan", and that FR-072 requires admin grant, auto-renewal and
// purchase to share one rule. Neither was true: it had exactly one production caller,
// and `Activate`, `Grant`, `state_machine` and `renewal_payment` each computed an
// expiry on their own.
//
// An inaccurate comment is not neutral. It told the next reader that the other paths
// had already been migrated, so nobody went looking - and the drift it described is
// exactly what it invited.
//
// These tests assert the call graph with the parser rather than by reading the comment
// back, because the behaviour is unchanged by this task and what needs protecting is
// the CLAIM. If someone later consolidates the paths, this fails and the comment gets
// rewritten in the same change - which is the entire point.
func TestResolveExpiryForCallerSetMatchesWhatTheCommentClaims(t *testing.T) {
	calls := resolveExpiryCallSites(t)

	// The corrected comment names the shared apply workflow as the only production
	// caller: admin grant, purchase settlement and default provisioning all reach it.
	if !hasFile(calls, "internal_subscription_apply_workflow.go") {
		t.Errorf("ResolveExpiryFor phải được gọi từ apply workflow, các file đang gọi: %v", calls)
	}

	// The corrected comment lists these as NOT going through it. If one starts calling
	// it, the "known gap" wording is stale and has to be rewritten in the same change.
	stillHandRolled := []string{
		"subscription_behavior.go", // Activate and Grant compute expiryFor themselves
		"state_machine.go",         // its own AddProductDays copy
		"renewal_payment.go",       // reads DurationDays directly
	}
	for _, file := range stillHandRolled {
		if hasFile(calls, file) {
			t.Errorf("%s đã bắt đầu gọi ResolveExpiryFor - phải cập nhật comment của hàm "+
				"và bỏ nó khỏi danh sách known gap, không được để hai nơi cùng nói một đằng", file)
		}
	}

	// The corrected comment says "one production caller". A second caller makes that
	// enumeration incomplete even if the paths above still avoid it.
	if len(calls) != 1 {
		t.Errorf("ResolveExpiryFor được gọi từ %d file (%v); comment nói một production "+
			"caller, phải liệt kê lại", len(calls), calls)
	}
}

// The other half of the corrected comment: `Activate` keeps working the expiry out
// itself, and keeps its own stricter validation.
//
// This is pinned because the obvious fix for the drift - routing everything through
// ResolveExpiryFor - would change behaviour. `Activate` returns an ERROR when a
// finite plan carries no duration, where `ResolveExpiryFor` returns a nil expiry. The
// two are not interchangeable, which is why consolidation is its own task instead of
// something to slip in while editing a comment.
func TestActivateKeepsItsOwnExpirySemantics(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	days := 30

	plan := &entities.SubscriptionPlan{
		Slug:         "with-duration",
		DurationDays: &days,
		PlanKind:     plankind.Finite,
	}
	plan.ID = uuid.New()

	// A finite plan with no duration is an ERROR here, where the shared rule would
	// have quietly produced a subscription that never expires.
	noDuration := &entities.SubscriptionPlan{Slug: "no-duration", PlanKind: plankind.Finite}
	noDuration.ID = uuid.New()

	sub := &entities.UserSubscription{CurrentPlanKind: plankind.DefaultFree}
	if _, err := sub.Activate(noDuration, entities.JSONDocument(`{}`), "key-1", now); err == nil {
		t.Error("Activate với gói hữu hạn thiếu DurationDays phải trả lỗi; " +
			"nếu im lặng thành công thì tài khoản sẽ có gói không bao giờ hết hạn")
	}

	sub2 := &entities.UserSubscription{CurrentPlanKind: plankind.DefaultFree}
	if _, err := sub2.Activate(plan, entities.JSONDocument(`{}`), "key-2", now); err != nil {
		t.Fatalf("Activate với gói hợp lệ phải thành công: %v", err)
	}
	if sub2.ExpiresAt == nil {
		t.Fatal("Activate phải đặt ngày hết hạn")
	}
	if want := now.AddDate(0, 0, days); !sub2.ExpiresAt.Equal(want) {
		t.Errorf("ngày hết hạn = %v, mong đợi %v", sub2.ExpiresAt, want)
	}
}

// resolveExpiryCallSites returns the production files that CALL
// ResolveExpiryFor.
//
// The parser is used rather than a text search because the identifier also appears
// in the declaration and its doc comment. Counting strings would either miss a real
// call or include the file that defines the method.
func resolveExpiryCallSites(t *testing.T) []string {
	t.Helper()

	root := moduleRoot(t)
	fset := token.NewFileSet()
	var calls []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Test files are excluded by the suffix check below; testdata is not Go.
			if info.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "ResolveExpiryFor" {
				return true
			}
			calls = append(calls, filepath.Base(path))
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("dò mã nguồn: %v", err)
	}

	sort.Strings(calls)
	return dedupe(calls)
}

func dedupe(in []string) []string {
	var out []string
	for i, item := range in {
		if i == 0 || in[i-1] != item {
			out = append(out, item)
		}
	}
	return out
}

func hasFile(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

// moduleRoot walks up from the test's directory until it finds go.mod.
func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("không tìm thấy go.mod từ thư mục test")
		}
		dir = parent
	}
}
