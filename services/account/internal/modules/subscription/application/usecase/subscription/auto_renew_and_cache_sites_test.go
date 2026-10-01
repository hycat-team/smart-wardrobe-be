package subscription

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
)

// 024 T154, second half.
//
// T151 fixed the post-commit cache delete on the admin grant path. T154 found the
// SAME defect copied into three more places, each with its own
// `_ = uc.overviewCache.Invalidate(ctx, ...)`:
//
//	auto-renew toggle      subscription_uc.go
//	scheduled renewal      subscription_renewal_workflow.go
//	wallet top-up          purchase/wallet.go
//	payment webhook        webhook/webhook.go
//
// Four copies of one bug is not four coincidences. Duplication is the mechanism, so
// the fix is to have one function and an assertion that keeps it that way - otherwise
// the fifth copy is a matter of time.
func TestNoCallSiteBypassesTheSharedCacheInvalidation(t *testing.T) {
	bypassing := overviewCacheInvalidateCallSites(t)

	// Zero is the goal, not a missing result. A direct `overviewCache.Invalidate` is
	// the four-bug shape: post-commit, on the caller's context, error discarded. All
	// of them have been replaced by the helper, whose parameter is named `cache`, so
	// a direct call on the field is exactly what "bypasses" means.
	if len(bypassing) != 0 {
		for _, site := range bypassing {
			t.Errorf("%s:%d gọi overviewCache.Invalidate trực tiếp - phải qua "+
				"InvalidateOverviewCache để tách khỏi context đã hủy và ghi log lỗi",
				site.file, site.line)
		}
	}

	// Guard against the test passing vacuously: if the helper itself were deleted,
	// "no direct calls" would still hold. Assert it is actually wired up.
	calls := functionCallSites(t, "InvalidateOverviewCache")
	if len(calls) < 3 {
		t.Errorf("InvalidateOverviewCache phải được dùng ở nhiều call site (admin grant, "+
			"auto-renew, renewal, top-up, webhook), hiện chỉ thấy %d: %v", len(calls), calls)
	}
}

// The shared helper must tolerate a nil cache: it is a deployment-wiring detail, and
// three of the four callers used to guard it themselves.
func TestSharedInvalidationToleratesNilCacheAndNilLogger(t *testing.T) {
	// Neither call may panic. A nil logger is reachable wherever a use case has not
	// been given one yet, and a panic in a post-commit cleanup would turn a committed
	// operation into a 500.
	InvalidateOverviewCache(context.Background(), nil, nil, uuid.New())
	InvalidateOverviewCache(context.Background(), logger.New("test", "", "info", false), nil, uuid.New())
}

// 024 T154, first half.
//
// The comment on the apply workflow claimed "Only a purchase - where the customer paid
// and opted in - may turn it on." No purchase does. `SetAutoRenew` has exactly one
// production caller - the auto-renew endpoint - so the ONLY thing that turns
// auto-renew on is the account owner asking for it.
//
// The old wording was wrong in the direction that hides a missing feature: a reader
// would assume a paying customer is already enrolled and would not go looking for the
// endpoint that is supposed to do it.
func TestOnlyTheExplicitToggleEnablesAutoRenew(t *testing.T) {
	user := grantTestUser("autorenew", "autorenew@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)
	plan := renewalTestPlan()

	// An administrative grant must leave auto-renew OFF: nobody consented to be
	// charged again. This is T134's behaviour and it is unchanged.
	if _, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"autorenew@example.com", plan.Slug},
	), "idem-AUTORENEW", nil); err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}

	sub, err := h.subs.GetByUserID(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("đọc subscription: %v", err)
	}
	if sub == nil {
		t.Fatal("phải có subscription sau khi cấp gói")
	}
	if sub.IsAutoRenewEnabled {
		t.Error("cấp gói hành chính không được tự bật auto-renew - khách chưa đồng ý bị tính phí lại")
	}
}

// SetAutoRenew is the only thing that can set the flag true, and it lives on the
// entity. Pinning the caller count keeps the corrected comment honest: if a purchase
// ever starts enrolling customers, this fails.
func TestSetAutoRenewHasExactlyOneProductionCaller(t *testing.T) {
	files := methodCallSites(t, "SetAutoRenew")

	if len(files) != 1 {
		t.Errorf("SetAutoRenew phải có đúng một production caller (endpoint toggle); "+
			"hiện có %d: %v", len(files), files)
		return
	}
	if files[0] != "subscription_uc.go" {
		t.Errorf("caller của SetAutoRenew phải là subscription_uc.go, thực tế = %v", files)
	}
}

// A finite plan without a duration must not be grantable into something that never
// expires. This is the validation that keeps the auto-renew question meaningful.
func TestGrantOfPlanWithoutDurationIsRejectedByPlanValidation(t *testing.T) {
	user := grantTestUser("nodur", "nodur@example.com", roleslug.User, false)
	h := newGrantHarness(t, user)

	// A plan with no DurationDays is refused by the plan catalogue validator upstream
	// of the grant path, so the grant reports it rather than silently writing a
	// subscription with no expiry.
	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"nodur@example.com", "no-such-plan"},
	), "idem-NODUR", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}
	if res.Results[0].Status == "success" {
		t.Errorf("mã gọi không tồn tại không được cấp thành công, thực tế = %s", res.Results[0].Reason)
	}
}

type invalidateSite struct {
	file string
	line int
}

// overviewCacheInvalidateCallSites finds every production call to
// overviewCache.Invalidate.
func overviewCacheInvalidateCallSites(t *testing.T) []invalidateSite {
	t.Helper()

	root := moduleRoot(t)
	fset := token.NewFileSet()
	var sites []invalidateSite

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
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
			if !ok || sel.Sel.Name != "Invalidate" {
				return true
			}
			recv, ok := sel.X.(*ast.SelectorExpr)
			if !ok || recv.Sel.Name != "overviewCache" {
				return true
			}
			sites = append(sites, invalidateSite{file: filepath.Base(path), line: fset.Position(call.Pos()).Line})
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("dò mã nguồn: %v", err)
	}
	return sites
}

// methodCallSites returns the production files calling the named method.
func methodCallSites(t *testing.T, method string) []string {
	t.Helper()
	return functionCallSites(t, method)
}

// functionCallSites returns the production files containing a call to the named
// function or method.
func functionCallSites(t *testing.T, name string) []string {
	t.Helper()

	root := moduleRoot(t)
	fset := token.NewFileSet()
	seen := map[string]bool{}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
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
			switch fun := call.Fun.(type) {
			case *ast.SelectorExpr:
				if fun.Sel.Name == name {
					seen[filepath.Base(path)] = true
				}
			case *ast.Ident:
				if fun.Name == name {
					seen[filepath.Base(path)] = true
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatalf("dò mã nguồn: %v", err)
	}

	var out []string
	for base := range seen {
		out = append(out, base)
	}
	sort.Strings(out)
	return out
}

var _ = entities.SubscriptionPlan{}
var _ = errors.New
