package subscription

import (
	"context"
	"strings"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
)

// 024 T147: plan resolution for the batch grant path.

// Split out of admin_grant_batch_uc.go, which was over the 800-line failure
// threshold in code-hygiene H1. The split follows the boundary that already
// existed logically: this file answers "which plan is this slug, and if not, why",
// and nothing else in the path does.

func (uc *SubscriptionUseCase) resolveGrantPlans(ctx context.Context, items []sub_dto.AdminGrantBatchItem) (map[int]*entities.SubscriptionPlan, map[int]planIssue) {
	plans := make(map[int]*entities.SubscriptionPlan, len(items))
	issues := make(map[int]planIssue)

	slugs := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		slug := strings.ToLower(strings.TrimSpace(item.PlanSlug))
		if slug == "" {
			continue
		}
		// 024 T125: an over-long slug is never collected, so it never reaches
		// `GetBySlug`. Checking it here rather than only in the classification
		// loop below matters because this is the loop that decides WHICH QUERIES
		// RUN.
		if ExceedsPlanSlugLength(slug) {
			continue
		}
		if _, ok := seen[slug]; ok {
			continue
		}
		seen[slug] = struct{}{}
		slugs = append(slugs, slug)
	}

	// 024 T150: ONE query for every distinct slug.
	//
	// This used to be a loop of `GetBySlugAnyStatus`, one round trip per distinct
	// slug. A 500-item list naming 500 different slugs therefore cost 500
	// sequential queries - and all of them ran before the request's time budget was
	// even started, so FR-076's 60 seconds did not apply to any of it. That is the
	// part that makes this more than a performance change: the pre-loop I/O was
	// unbounded, not merely slow.
	//
	// `slug` is uniquely indexed, so `IN (...)` stays index-backed at any list size.
	loaded, err := uc.planRepo.GetBySlugsAnyStatus(ctx, slugs)
	if err != nil {
		// A lookup failure is reported per item as "not found" rather than as an
		// infrastructure error: the item is refused either way, and a distinct
		// reason code would promise the administrator a retry that has no different
		// outcome. The map is simply left empty, which is what "nothing resolved"
		// looks like to the classification loop below.
		loaded = nil
	}

	for idx, item := range items {
		slug := strings.ToLower(strings.TrimSpace(item.PlanSlug))
		// 024 T125: refused BEFORE the lookup, for the same reason FR-062
		// refuses an over-long identifier. A slug far longer than the 100-char
		// column cannot use the index, so one such item turns the whole
		// request's slug lookup into table scans.
		if ExceedsPlanSlugLength(slug) {
			issues[idx] = planIssue{
				reason:  sub_dto.AdminGrantReasonPlanSlugTooLong,
				message: mapper.MsgAdminGrantPlanSlugTooLong,
			}
			continue
		}
		plan := loaded[slug]
		switch {
		case plan == nil:
			issues[idx] = planIssue{
				reason:  sub_dto.AdminGrantReasonPlanNotFound,
				message: mapper.MsgAdminGrantPlanNotFound,
			}
		case !plan.IsActive:
			// An inactive plan is refused rather than silently substituted: a
			// grant is a promise about what the customer will get.
			issues[idx] = planIssue{
				reason:  sub_dto.AdminGrantReasonPlanInactive,
				message: mapper.MsgAdminGrantPlanInactive,
			}
		default:
			plans[idx] = plan
		}
	}
	return plans, issues
}
