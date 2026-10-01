package subscription

import (
	"context"
	"fmt"
	"testing"
	"time"

	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/shared/domain/constants/roleslug"

	"github.com/google/uuid"
)

// slugBatchSize is large enough that a per-slug implementation would be obviously
// wrong rather than merely slow, and small enough to keep the test instant.
const slugBatchSize = 50

// 024 T150: a batch grant must resolve every plan slug in ONE query, not one query
// per slug.
//
// This was not only a performance concern, because of where the work sat. The 60
// second budget FR-076 requires used to be opened AFTER this step, so the slug
// lookups ran on the unbounded request context. A 500-item list naming 500 distinct
// slugs meant 500 sequential round trips that no deadline applied to, all of them
// before the administrator's clock had even started.
func TestAdminGrantBatchResolvesEverySlugInOneQuery(t *testing.T) {
	users := make([]*identity_dto.UserBriefContractDTO, 0, slugBatchSize)
	pairs := make([][2]string, 0, slugBatchSize)
	for i := 0; i < slugBatchSize; i++ {
		email := fmt.Sprintf("bulk%d@example.com", i)
		users = append(users, grantTestUser(fmt.Sprintf("bulk%d", i), email, roleslug.User, false))

		// Every item names a DIFFERENT slug, which is the case the per-slug loop
		// made worst. A list repeating one slug would have looked fine.
		slug := fmt.Sprintf("plan-%02d", i)
		pairs = append(pairs, [2]string{email, slug})
	}
	h := newGrantHarness(t, users...)
	for i := 0; i < slugBatchSize; i++ {
		plan := renewalTestPlan()
		plan.ID = uuid.New()
		plan.Slug = fmt.Sprintf("plan-%02d", i)
		h.plans.planBySlug[plan.Slug] = plan
	}

	if _, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(pairs...), "idem-BULK", nil); err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}

	if h.plans.slugBatchCalls != 1 {
		t.Errorf("%d mã gọi khác nhau phải tra cứu bằng MỘT truy vấn, thực tế = %d lần",
			slugBatchSize, h.plans.slugBatchCalls)
	}
	if len(h.plans.slugBatchSlugs) != slugBatchSize {
		t.Errorf("truy vấn phải mang đủ %d mã gọi, mang %d mã gọi", slugBatchSize, len(h.plans.slugBatchSlugs))
	}

	// A regression back to the loop would show up here as a count of
	// slugBatchSize rather than 1, which is the number an operator would feel.
	if h.plans.slugCalls != 1 {
		t.Errorf("tổng số lần tra cứu mã gọi phải bằng 1, thực tế = %d", h.plans.slugCalls)
	}
	if h.plans.activeOnlyCalls != 0 {
		t.Errorf("không được dùng GetBySlug (chỉ lấy plan đang hoạt động), thực tế = %d lần", h.plans.activeOnlyCalls)
	}
}

// A repeated slug must still be collapsed: one query, but not the same slug twice.
func TestAdminGrantBatchDeduplicatesSlugsInTheBatch(t *testing.T) {
	users := make([]*identity_dto.UserBriefContractDTO, 0, 6)
	pairs := make([][2]string, 0, 6)
	for i := 0; i < 6; i++ {
		email := fmt.Sprintf("dup%d@example.com", i)
		users = append(users, grantTestUser(fmt.Sprintf("dup%d", i), email, roleslug.User, false))
		pairs = append(pairs, [2]string{email, "premium"})
	}
	h := newGrantHarness(t, users...)

	if _, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(pairs...), "idem-DUP", nil); err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}

	if h.plans.slugBatchCalls != 1 {
		t.Fatalf("phải tra cứu đúng một lần, thực tế = %d", h.plans.slugBatchCalls)
	}
	if len(h.plans.slugBatchSlugs) != 1 || h.plans.slugBatchSlugs[0] != "premium" {
		t.Errorf("mã gọi lặp phải được gộp còn một, thực tế = %v", h.plans.slugBatchSlugs)
	}
}

// The batch must not weaken T135: a DEACTIVATED plan still has to come back and be
// reported as `plan_inactive` rather than `plan_not_found`.
//
// This is the case a batch lookup is most likely to break by accident, because
// "find the plans in this list" invites a `WHERE is_active = true`.
func TestAdminGrantBatchBatchLookupStillReportsInactivePlans(t *testing.T) {
	h := newGrantHarness(t, grantTestUser("inactive", "inactive@example.com", roleslug.User, false))

	inactive := renewalTestPlan()
	inactive.Slug = "retired"
	inactive.IsActive = false
	h.plans.planBySlug[inactive.Slug] = inactive

	res, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"inactive@example.com", "retired"},
	), "idem-INACTIVE", nil)
	if err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}
	if res.Results[0].Reason != sub_dto.AdminGrantReasonPlanInactive {
		t.Errorf("plan đã bị vô hiệu hóa phải báo plan_inactive, thực tế = %s", res.Results[0].Reason)
	}
}

// 024 T150: the time budget must cover the pre-loop work.
//
// The budget used to be opened after `Resolve`, `resolveGrantPlans` and
// `loadPlannedVersions`, so FR-076's "answer within 60 seconds" was not enforced
// for any of them.
//
// The deadline is observed inside the plan repository because that is where the
// question is actually asked: not "does some later step have a deadline" but "did
// the budget already exist by the time the first query went out".
func TestAdminGrantBatchTimeBudgetCoversPreLoopWork(t *testing.T) {
	h := newGrantHarness(t, grantTestUser("budget", "budget@example.com", roleslug.User, false))

	if _, err := h.uc.AdminGrantBatch(context.Background(), h.adminID, grantTestItems(
		[2]string{"budget@example.com", "premium"},
	), "idem-BUDGET", nil); err != nil {
		t.Fatalf("AdminGrantBatch lỗi: %v", err)
	}

	if !h.plans.sawDeadline {
		t.Error("ngân sách thời gian phải có hiệu lực TRƯỚC khi tra cứu mã gọi, không phải sau đó")
	}
	if h.plans.remaining <= 0 {
		t.Errorf("thời gian còn lại phải lớn hơn 0, thực tế = %v", h.plans.remaining)
	}
	// The budget is sized from the list length (5s for one item, 60s for 500), so a
	// single-item list must not be granted the whole 60 seconds.
	if h.plans.remaining > 30*time.Second {
		t.Errorf("một mục không nên được cấp cả ngân sách 60 giây, còn lại = %v", h.plans.remaining)
	}
}
