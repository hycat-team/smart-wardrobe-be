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

// 024 T153, re-stated by 027 D-08.
//
// The comment on `ResolveExpiryFor` used to claim it was "THE expiry rule for every
// path that changes a plan". It never was: `Activate`, `Grant`, `state_machine` and
// `renewal_payment` each compute an expiry themselves, and purchase settlement stacks
// through `ApplyTransition`.
//
// 027/D-08 removed its last production caller as well. The workflow that admin and
// campaign grants share is ADMINISTRATIVE, so it now calls `ResolveAdminExpiryFor`
// and starts a fresh period at `now`. `ResolveExpiryFor` stays as the documented rule
// for a period the customer paid for, with no caller today.
//
// These assertions read the call graph with the parser, because what needs protecting
// is the CLAIM in those comments. If a path starts calling one of these methods, this
// fails and the comment gets rewritten in the same change.
func TestExpiryCallGraphMatchesTheComments(t *testing.T) {
	resolveCalls := callSites(t, "ResolveExpiryFor")
	if len(resolveCalls) != 0 {
		t.Errorf("ResolveExpiryFor hiện không còn production caller (027/D-08): purchase đi qua "+
			"ApplyTransition, đường hành chính dùng ResolveAdminExpiryFor. Các file đang gọi: %v", resolveCalls)
	}

	adminCalls := callSites(t, "ResolveAdminExpiryFor")
	// The corrected comment names the shared apply workflow as the only production
	// caller: administrator batch grant and campaign grant both reach it.
	if !hasFile(adminCalls, "internal_subscription_apply_workflow.go") {
		t.Errorf("ResolveAdminExpiryFor phải được gọi từ apply workflow, các file đang gọi: %v", adminCalls)
	}

	// The comment says this is the administrative rule and that the paid paths do NOT
	// use it. A new caller makes that wording stale.
	if len(adminCalls) != 1 {
		t.Errorf("ResolveAdminExpiryFor được gọi từ %d file (%v); comment nói một production "+
			"caller, phải liệt kê lại", len(adminCalls), adminCalls)
	}

	stillHandRolled := []string{
		"subscription_behavior.go", // Activate and Grant compute expiryFor themselves
		"state_machine.go",         // its own AddProductDays copy
		"renewal_payment.go",       // reads DurationDays directly
	}
	for _, file := range stillHandRolled {
		if hasFile(adminCalls, file) {
			t.Errorf("%s đã bắt đầu gọi ResolveAdminExpiryFor - phải cập nhật comment của hàm "+
				"và bỏ nó khỏi danh sách known gap, không được để hai nơi cùng nói một đằng", file)
		}
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

// callSites returns the production files that CALL the named method.
//
// The parser is used rather than a text search because the identifier also appears
// in the declaration and its doc comment. Counting strings would either miss a real
// call or include the file that defines the method.
func callSites(t *testing.T, method string) []string {
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
			if !ok || sel.Sel.Name != method {
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
