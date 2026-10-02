package subscription

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
)

// 025 T093 / FR-051: the campaign exclusion must not reach the MANUAL grant.
//
// Its own file, beside the other FR-051 half rather than inside
// `admin_grant_batch_uc_test.go`, because that file is already past the test-file
// length threshold and this is a different subject: not "how a grant works" but
// "what a grant is allowed to know about".

// TestAdminGrantBatchIgnoresTheCampaignAdministratorExclusion is T093 / FR-051.
//
// FR-050 excludes administrator-created accounts from the CAMPAIGN. FR-051 keeps
// the other promise apart from it: granting a plan by hand must not be affected by
// the campaign mechanism existing at all. Those are different subjects — a
// promotion applies to some accounts and not others, while a manual grant is a
// decision about ONE named account — so a rule reaching from the first to the
// second is a category error, and an expensive one: an operator compensating a
// customer would be told the account is ineligible for a reason that has nothing
// to do with the grant in front of them.
//
// The account here is `created_by_admin = true` with the ordinary `user` role,
// which is the combination FR-050 is actually about and deliberately NOT the case
// `TestAdminGrantBatchRefusesAdministratorTarget` covers. That one refuses an
// account whose ROLE is `admin`, because granting Premium to staff charges the
// company for its own employee (FR-056). `role_slug` and `created_by_admin` are
// different facts about different people, and reading the second as the first
// would refuse compensation for a real customer every single time staff happened
// to create their account — which is exactly how a real account gets created.
func TestAdminGrantBatchIgnoresTheCampaignAdministratorExclusion(t *testing.T) {
	// An account an administrator created, holding the ordinary customer role.
	staffCreated := grantTestUser("duck.nguyen", "duck.nguyen@example.com", roleslug.User, false)
	staffCreated.CreatedByAdmin = true
	ordinary := grantTestUser("khach", "khach@example.com", roleslug.User, false)
	h := newGrantHarness(t, staffCreated, ordinary)

	// Premise first: without it this test would pass against a harness that never
	// set the flag, and would then be measuring nothing at all.
	if !staffCreated.CreatedByAdmin {
		t.Fatal("premise broken: the account under test is not administrator-created, so this test proves nothing about FR-051")
	}

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"duck.nguyen@example.com", "premium"},
		[2]string{"khach@example.com", "premium"},
	), "key-1", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}

	// Both accounts are granted. The campaign reason has no vocabulary on this path
	// at all: a refused item reports one of the AdminGrantReason* codes, and none
	// of them may name the campaign or the exclusion.
	for i, identifier := range []string{"duck.nguyen@example.com", "khach@example.com"} {
		if res.Results[i].Status != sub_dto.AdminGrantStatusSuccess {
			t.Errorf("tài khoản %s phải được cấp thủ công dù cơ chế chiến dịch có tồn tại hay không (FR-051), có = %s (%s, reason=%s)",
				identifier, res.Results[i].Status, res.Results[i].Message, res.Results[i].Reason)
		}
		reason := strings.ToLower(string(res.Results[i].Reason))
		if strings.Contains(reason, "campaign") || strings.Contains(reason, "created_by_admin") {
			t.Errorf("lý do của đường cấp thủ công lấy từ cơ chế chiến dịch: %s", res.Results[i].Reason)
		}
	}
	if res.Succeeded != 2 || res.Failed != 0 {
		t.Errorf("cả hai tài khoản phải được cấp, có = %d thành công / %d lỗi", res.Succeeded, res.Failed)
	}

	// The ledger, not the return value: a path that answered "granted" while
	// writing nothing would satisfy every assertion above.
	for _, account := range []*identity_dto.UserBriefContractDTO{staffCreated, ordinary} {
		sub := h.subs.subs[account.ID]
		if sub == nil {
			t.Fatalf("không có dòng gói cho %s sau khi cấp thủ công", account.Email)
		}
		if sub.CurrentPlanCode != "premium" {
			t.Errorf("tài khoản %s phải nằm trên gói premium, có = %q", account.Email, sub.CurrentPlanCode)
		}
	}
}

// TestManualGrantPathNeverReadsTheCampaignConfiguration is the structural half of
// FR-051, and the assertion the behavioural test above cannot make.
//
// The test above proves the grant WORKS. It cannot prove WHY, and the failure
// mode FR-051 names is a grant that still succeeds but is now GATED: a future
// change that consults `cfg.Campaigns`, or tests `CreatedByAdmin`, before applying
// a hand-written grant would pass the test above on any day no campaign happened
// to be configured, and fail in production the week one is. A test that depends on
// the day of the week is not a test of the rule.
//
// The compiler cannot see this either — reading a configuration key and branching
// on a struct field are ordinary statements — so the rule is checked by reading
// the source of the package that owns the manual grant. Test files are excluded
// for the reason the sibling scans in the campaign package exclude them: this very
// test names the forbidden identifiers in order to search for them, and a guard
// that fails on itself is a guard that gets deleted.
func TestManualGrantPathNeverReadsTheCampaignConfiguration(t *testing.T) {
	forbidden := []string{
		// The exclusion itself. Nothing in a manual grant may consult it.
		"adminSkipReason", "SkipCreatedByAdmin", "CreatedByAdmin",
		// Nor may the path know a campaign exists. FR-051 says the grant is
		// unaffected by the MECHANISM, which is stronger than "unaffected by its
		// current configuration": a path that reads the campaign list can be made
		// to branch on it by a later edit, and that edit would compile right here.
		"Campaigns", "CampaignConfig", "campaignEvaluator", "SkipReason",
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("đọc thư mục gói: %v", err)
	}
	var hits []string
	scanned := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		scanned++
		content, readErr := os.ReadFile(name)
		if readErr != nil {
			t.Fatalf("đọc %s: %v", name, readErr)
		}
		for _, needle := range forbidden {
			if strings.Contains(string(content), needle) {
				hits = append(hits, name+": "+needle)
			}
		}
	}
	if scanned == 0 {
		t.Fatal("không tìm thấy tệp nguồn nào của gói; phép kiểm tra sẽ pass vô nghĩa")
	}
	if len(hits) > 0 {
		t.Errorf("FR-051: đường cấp gói thủ công của quản trị viên không được biết cơ chế chiến dịch tồn tại. "+
			"Việc cấp tay là một quyết định về MỘT tài khoản cụ thể, không phải về một chương trình khuyến mãi.\nPhát hiện:\n  %s",
			strings.Join(hits, "\n  "))
	}
}
