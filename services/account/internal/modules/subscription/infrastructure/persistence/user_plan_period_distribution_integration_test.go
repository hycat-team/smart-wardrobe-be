package persistence

import (
	"context"
	"testing"
	"time"

	subscriptiondto "smart-wardrobe-be/services/account/dto/subscription"
	adminstats "smart-wardrobe-be/services/account/internal/modules/subscription/application/usecase/admin_stats"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// 031 T011 — Postgres coverage for the distribution report over the validity
// intervals (FR-007/FR-008/FR-010/FR-012, data-model §5/§6).
//
// The fixture deliberately backdates the cutover row before the reported
// period, then moves it into the middle of the period for the clipping case.
// The same user can hold two plans in the period (mid-period switch) and can
// downgrade AFTER the period: the old projection lost both facts, the
// interval table must keep them.
func TestIntegrationUserPlanPeriodDistributionLifecycles(t *testing.T) {
	_, db, closeScratch, closeBase := openPlan031ScratchDB(t)
	defer closeBase()
	defer closeScratch()

	createPlan031BaseSchema(t, db)
	if err := applyPlan031Migration(t, db); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	premiumPlanID := seedPlan031Plan(t, db, "Premium Monthly")
	switchAPlanID := seedPlan031Plan(t, db, "Switch A")
	switchBPlanID := seedPlan031Plan(t, db, "Switch B")
	freePlanID := seedPlan031Plan(t, db, "Free")
	lifetimePlanID := seedPlan031Plan(t, db, "Lifetime")
	legacyPlanID := seedPlan031Plan(t, db, "Legacy Monthly")
	newPlanID := seedPlan031Plan(t, db, "New Monthly")

	downgradedUser := seedPlan031User(t, db)
	switchedUser := seedPlan031User(t, db)
	freeUser := seedPlan031User(t, db)
	lifetimeUser := seedPlan031User(t, db)
	futureUser := seedPlan031User(t, db)
	legacyUser := seedPlan031User(t, db)
	newUser := seedPlan031User(t, db)

	// Relative instants keep the one-sided and snapshot assertions true on any
	// run date: the reported window is in the past, the future interval starts
	// after now, and the open free/lifetime intervals contain now.
	now := time.Now().UTC().Truncate(time.Second)
	periodFrom := now.AddDate(0, 0, -60)
	periodTo := now.AddDate(0, 0, -30)
	cutover := periodFrom.AddDate(0, 0, -10)
	setPlan031Cutover(t, db, cutover)

	insertPeriod := func(userID, planID uuid.UUID, code, name string, tierRank int, planKind int16, from time.Time, to *time.Time, key string) {
		t.Helper()
		if err := db.Exec(`
			INSERT INTO subscription.user_plan_periods
				(user_id, subscription_plan_id, plan_code, plan_name, tier_rank, plan_kind, valid_from, valid_to, source_event_key)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			userID, planID, code, name, tierRank, planKind, from, to, key).Error; err != nil {
			t.Fatalf("insert interval %s: %v", key, err)
		}
	}

	// Downgraded AFTER the reported period: premium in the period, free now.
	// The old projection would have shown only "free", losing this user from
	// the premium bucket (FR-007 acceptance scenario 1).
	downgradePremiumStart := periodFrom.AddDate(0, 0, -5)
	downgradeAt := periodTo.AddDate(0, 0, 10)
	insertPeriod(downgradedUser, premiumPlanID, "premium-monthly", "Premium Monthly", 1, 1, downgradePremiumStart, &downgradeAt, "T011:downgraded:premium")
	insertPeriod(downgradedUser, freePlanID, "free", "Free", 0, 0, downgradeAt, nil, "T011:downgraded:free")

	// Mid-period switch: the same user must appear in BOTH buckets (FR-007
	// acceptance scenario 2).
	switchAt := periodFrom.AddDate(0, 0, 15)
	switchAEnd := switchAt
	switchBEnd := periodTo.AddDate(0, 0, 5)
	insertPeriod(switchedUser, switchAPlanID, "switch-a", "Switch A", 1, 1, periodFrom.AddDate(0, 0, -5), &switchAEnd, "T011:switch:a")
	insertPeriod(switchedUser, switchBPlanID, "switch-b", "Switch B", 1, 1, switchAt, &switchBEnd, "T011:switch:b")

	// Free account for the whole period: open-ended interval, valid_to NULL.
	insertPeriod(freeUser, freePlanID, "free", "Free", 0, 0, periodFrom.AddDate(0, 0, -100), nil, "T011:free")
	// Lifetime account: also open-ended, but plan_kind 2.
	insertPeriod(lifetimeUser, lifetimePlanID, "lifetime", "Lifetime", 2, 2, periodFrom.AddDate(0, 0, -100), nil, "T011:lifetime")

	// Future-dated open interval: a from-only period must exclude it because
	// the absent end means "up to now" (FR-008/A2).
	futureFrom := now.AddDate(0, 0, 10)
	insertPeriod(futureUser, premiumPlanID, "premium-monthly", "Premium Monthly", 1, 1, futureFrom, nil, "T011:future")

	// Legacy history before a mid-period cutover, and a plan that starts after
	// it. Clipping must drop the first and keep the second.
	partialCutover := periodFrom.AddDate(0, 0, 10)
	legacyEnd := partialCutover
	insertPeriod(legacyUser, legacyPlanID, "legacy-monthly", "Legacy Monthly", 1, 1, periodFrom, &legacyEnd, "T011:legacy")
	insertPeriod(newUser, newPlanID, "new-monthly", "New Monthly", 1, 1, partialCutover, &switchBEnd, "T011:new")

	uc := adminstats.NewAdminStatsUseCase(nil, nil, NewUserPlanPeriodRepository(db), logger.New("test", "", "error", false))
	ctx := context.Background()

	// --- Full period after the cutover: complete -------------------------
	full, err := uc.GetSubscriptionDistribution(ctx, periodFrom, periodTo)
	if err != nil {
		t.Fatalf("full-period distribution: %v", err)
	}
	if full.DataCompleteness != adminstats.DistributionCompletenessComplete {
		t.Fatalf("full-period completeness = %q, want %q", full.DataCompleteness, adminstats.DistributionCompletenessComplete)
	}
	if full.DataAvailableFrom == nil || !full.DataAvailableFrom.Equal(cutover) {
		t.Fatalf("dataAvailableFrom = %v, want %s", full.DataAvailableFrom, cutover)
	}

	fullCounts := plan031DistributionCounts(full)
	assertPlan031Bucket(t, fullCounts, "premium-monthly", 1, "the downgraded user must stay in premium for the reported period")
	assertPlan031Bucket(t, fullCounts, "switch-a", 1, "the user held plan A during the first half")
	assertPlan031Bucket(t, fullCounts, "switch-b", 1, "the user held plan B during the second half")
	assertPlan031Bucket(t, fullCounts, "free", 1, "only the free account overlaps this period; the downgraded free row starts after it")
	assertPlan031Bucket(t, fullCounts, "lifetime", 1, "the lifetime interval is open-ended and overlaps the period")
	assertPlan031Bucket(t, fullCounts, "legacy-monthly", 1, "the legacy row is inside this complete period (the cutover is earlier)")
	assertPlan031Bucket(t, fullCounts, "new-monthly", 1, "the replacement plan is also inside this complete period")

	// totalUsers is the sum of buckets, not a distinct-user count: the
	// switcher contributes twice and the legacy/replacement pair adds two
	// more rows for two users.
	if sum := plan031DistributionSum(full); sum != 7 {
		t.Fatalf("bucket sum = %d, want 7 (premium + switch A + switch B + free + lifetime + legacy + new)", sum)
	}

	// --- From-only period: no future interval ----------------------------
	fromOnly, err := uc.GetSubscriptionDistribution(ctx, periodFrom, time.Time{})
	if err != nil {
		t.Fatalf("from-only distribution: %v", err)
	}
	fromOnlyCounts := plan031DistributionCounts(fromOnly)
	assertPlan031Bucket(t, fromOnlyCounts, "premium-monthly", 1, "the future-dated premium interval must be excluded by valid_from <= now()")
	// The window now reaches up to now, so the downgraded user's open free
	// interval (which starts after the reported period) is inside it too.
	assertPlan031Bucket(t, fromOnlyCounts, "free", 2, "the downgraded user is on free from the downgrade instant up to now")
	if sum := plan031DistributionSum(fromOnly); sum != 8 {
		t.Fatalf("from-only bucket sum = %d, want 8; the future open interval must not add a bucket", sum)
	}

	// --- To-only period: the repository receives (-infinity, to) ---------
	toOnly, err := uc.GetSubscriptionDistribution(ctx, time.Time{}, periodTo)
	if err != nil {
		t.Fatalf("to-only distribution: %v", err)
	}
	if toOnly.DataCompleteness != adminstats.DistributionCompletenessPartial {
		t.Fatalf("to-only completeness = %q, want partial: a zero start means 'from the beginning of history', which predates the cutover",
			toOnly.DataCompleteness)
	}
	if sum := plan031DistributionSum(toOnly); sum != 7 {
		t.Fatalf("to-only bucket sum = %d, want 7", sum)
	}

	// --- No period: live snapshot ----------------------------------------
	snapshot, err := uc.GetSubscriptionDistribution(ctx, time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("snapshot distribution: %v", err)
	}
	if snapshot.DataCompleteness != adminstats.DistributionCompletenessComplete {
		t.Fatalf("snapshot completeness = %q, want complete", snapshot.DataCompleteness)
	}
	snapshotCounts := plan031DistributionCounts(snapshot)
	assertPlan031Bucket(t, snapshotCounts, "free", 2, "the downgraded user and the free account both hold free now")
	assertPlan031Bucket(t, snapshotCounts, "lifetime", 1, "the lifetime interval contains now()")
	assertPlan031Bucket(t, snapshotCounts, "premium-monthly", 0, "the future-dated premium interval does not contain now()")

	// --- Period straddling the cutover: partial + clipped -----------------
	setPlan031Cutover(t, db, partialCutover)

	partial, err := uc.GetSubscriptionDistribution(ctx, periodFrom, periodTo)
	if err != nil {
		t.Fatalf("partial distribution: %v", err)
	}
	if partial.DataCompleteness != adminstats.DistributionCompletenessPartial {
		t.Fatalf("partial completeness = %q, want partial", partial.DataCompleteness)
	}
	if partial.DataAvailableFrom == nil || !partial.DataAvailableFrom.Equal(partialCutover) {
		t.Fatalf("dataAvailableFrom = %v, want the mid-period cutover %s", partial.DataAvailableFrom, partialCutover)
	}
	partialCounts := plan031DistributionCounts(partial)
	assertPlan031Bucket(t, partialCounts, "legacy-monthly", 0, "an interval ending exactly at the cutover has no overlap with the clipped window")
	assertPlan031Bucket(t, partialCounts, "new-monthly", 1, "the interval that starts at the cutover is inside the clipped window")
	assertPlan031Bucket(t, partialCounts, "premium-monthly", 1, "the downgraded premium interval still overlaps the part after the cutover")
	if sum := plan031DistributionSum(partial); sum != 6 {
		t.Fatalf("partial bucket sum = %d, want 6: the clipped window drops only the legacy row", sum)
	}
}

// setPlan031Cutover moves the cutover row so the fixture can describe both an
// old cutover (complete period) and one in the middle of the period (partial).
func setPlan031Cutover(t *testing.T, db *gorm.DB, at time.Time) {
	t.Helper()
	if err := db.Exec(`
		INSERT INTO subscription.plan_history_meta (id, started_at, note)
		VALUES ('cutover', ?, 'T011 fixture')
		ON CONFLICT (id) DO UPDATE SET started_at = EXCLUDED.started_at`,
		at).Error; err != nil {
		t.Fatalf("set cutover %s: %v", at, err)
	}
}

func plan031DistributionCounts(result *subscriptiondto.SubscriptionDistributionResultDTO) map[string]int64 {
	counts := make(map[string]int64, len(result.Items))
	for _, item := range result.Items {
		counts[item.PlanCode] = item.UserCount
	}
	return counts
}

func plan031DistributionSum(result *subscriptiondto.SubscriptionDistributionResultDTO) int64 {
	var sum int64
	for _, item := range result.Items {
		sum += item.UserCount
	}
	return sum
}

func assertPlan031Bucket(t *testing.T, counts map[string]int64, planCode string, want int64, reason string) {
	t.Helper()
	if got := counts[planCode]; got != want {
		t.Fatalf("bucket %q = %d, want %d (%s); all buckets: %v", planCode, got, want, reason, counts)
	}
}
