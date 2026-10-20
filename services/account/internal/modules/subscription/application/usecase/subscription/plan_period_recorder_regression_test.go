package subscription

import (
	"context"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/services/account/internal/modules/subscription/infrastructure/persistence"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"
	shareddb "smart-wardrobe-be/shared/infrastructure/db"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// TestIntegrationPlanPeriodRecorderClosesFinitePaidIntervals is the regression
// guard for a defect in 031's first recorder: it looked up the current interval
// with `valid_to IS NULL`, while every finite paid interval stores
// valid_to = expires_at. A paid plan therefore looked "missing":
//
//   - an upgrade found no current row and inserted a second interval that
//     overlapped the still-open paid window (the exclusion constraint rejected
//     it, or, in the fallback variant, the overlap was silently accepted);
//   - an EXTENDED inserted a new row instead of moving valid_to (FR-004);
//   - a downgrade could not close the paid interval at all.
//
// The test drives the real recorder against the real repository and asserts the
// persisted intervals, not just the branch taken.
func TestIntegrationPlanPeriodRecorderClosesFinitePaidIntervals(t *testing.T) {
	dbConn := openSubscriptionTestDB(t)
	ctx := context.Background()

	planRepo := persistence.NewSubscriptionPlanRepository(dbConn)
	periodRepo := persistence.NewUserPlanPeriodRepository(dbConn)
	recorder := NewPlanPeriodRecorder(periodRepo)
	uow := shareddb.NewGormUnitOfWork(dbConn)

	userID := uuid.New()
	premium := &entities.SubscriptionPlan{
		Slug:               "regression-premium-" + userID.String(),
		Name:               "Regression Premium",
		Price:              decimal.NewFromInt(99000),
		MaxWardrobeItems:   100,
		MaxOutfits:         30,
		AiOutfitDailyQuota: 10,
		AiChatDailyQuota:   20,
		IsActive:           true,
		PlanKind:           plankind.Finite,
		TierRank:           1,
		DurationDays:       planPeriodDurationDays(30),
	}
	premium.ID = uuid.New()
	pro := &entities.SubscriptionPlan{
		Slug:               "regression-pro-" + userID.String(),
		Name:               "Regression Pro",
		Price:              decimal.NewFromInt(199000),
		MaxWardrobeItems:   500,
		MaxOutfits:         100,
		AiOutfitDailyQuota: 30,
		AiChatDailyQuota:   50,
		IsActive:           true,
		PlanKind:           plankind.Finite,
		TierRank:           2,
		DurationDays:       planPeriodDurationDays(30),
	}
	pro.ID = uuid.New()

	ensureSubscriptionUser(t, dbConn, userID, "plan-period-regression")
	if err := planRepo.Create(ctx, premium); err != nil {
		t.Fatalf("seed premium plan: %v", err)
	}
	if err := planRepo.Create(ctx, pro); err != nil {
		t.Fatalf("seed pro plan: %v", err)
	}
	t.Cleanup(func() {
		statements := []struct {
			sql  string
			args []any
		}{
			{`DELETE FROM subscription.user_plan_periods WHERE user_id = ?`, []any{userID}},
			{`DELETE FROM subscription.subscription_plans WHERE id IN (?, ?)`, []any{premium.ID, pro.ID}},
			{`DELETE FROM identity.users WHERE id = ?`, []any{userID}},
		}
		for _, statement := range statements {
			if err := dbConn.Exec(statement.sql, statement.args...).Error; err != nil {
				t.Errorf("cleanup %q: %v", statement.sql, err)
			}
		}
	})

	record := func(t *testing.T, event *entities.UserSubscriptionEvent, sub *entities.UserSubscription, plan *entities.SubscriptionPlan) {
		t.Helper()
		if err := uow.Execute(ctx, func(txCtx context.Context) error {
			return recorder.Record(txCtx, event, sub, plan)
		}); err != nil {
			t.Fatalf("record %s: %v", event.EventType, err)
		}
	}

	activatedAt := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	firstExpiry := activatedAt.Add(30 * 24 * time.Hour)
	sub := &entities.UserSubscription{
		UserID:             userID,
		SubscriptionPlanID: premium.ID,
		CurrentPlanCode:    premium.Slug,
		CurrentTierRank:    premium.TierRank,
		CurrentPlanKind:    premium.PlanKind,
		StartedAt:          activatedAt,
		ExpiresAt:          &firstExpiry,
	}

	// 1. Activate a finite paid plan: one closed-at-expiry interval.
	record(t, &entities.UserSubscriptionEvent{
		EventKey:    "031-regression:activate:" + userID.String(),
		UserID:      userID,
		EventType:   string(subscriptioneventtype.ACTIVATED),
		OccurredAt:  activatedAt,
		EffectiveAt: activatedAt,
	}, sub, premium)

	periods := planPeriodHistory(t, periodRepo, ctx, userID)
	if len(periods) != 1 {
		t.Fatalf("after activation history has %d interval(s), want 1", len(periods))
	}
	if periods[0].PlanCode != premium.Slug || periods[0].ValidTo == nil || !periods[0].ValidTo.Equal(firstExpiry) {
		t.Fatalf("activated interval = (%s, valid_from %s, valid_to %v), want the finite %s window ending %s",
			periods[0].PlanCode, periods[0].ValidFrom, periods[0].ValidTo, premium.Slug, firstExpiry)
	}

	// 2. Upgrade before the contractual expiry. The recorder must find the
	// finite paid interval, move its valid_to back to the switch instant, and
	// insert the new interval at that same instant.
	upgradedAt := activatedAt.Add(10 * 24 * time.Hour)
	proExpiry := upgradedAt.Add(30 * 24 * time.Hour)
	sub.SubscriptionPlanID = pro.ID
	sub.CurrentPlanCode = pro.Slug
	sub.CurrentTierRank = pro.TierRank
	sub.CurrentPlanKind = pro.PlanKind
	sub.StartedAt = upgradedAt
	sub.ExpiresAt = &proExpiry
	record(t, &entities.UserSubscriptionEvent{
		EventKey:    "031-regression:upgrade:" + userID.String(),
		UserID:      userID,
		EventType:   string(subscriptioneventtype.UPGRADED),
		OccurredAt:  upgradedAt,
		EffectiveAt: upgradedAt,
	}, sub, pro)

	periods = planPeriodHistory(t, periodRepo, ctx, userID)
	if len(periods) != 2 {
		t.Fatalf("after upgrade history has %d interval(s), want exactly 2: one closed paid interval and one new interval; rows: %s",
			len(periods), planPeriodSummary(periods))
	}
	closedPaid, newPaid := periods[0], periods[1]
	if closedPaid.PlanCode != premium.Slug || closedPaid.ValidTo == nil || !closedPaid.ValidTo.Equal(upgradedAt) {
		t.Errorf("closed paid interval = (%s, valid_from %s, valid_to %v), want (%s, %s, %s)",
			closedPaid.PlanCode, closedPaid.ValidFrom, closedPaid.ValidTo, premium.Slug, activatedAt, upgradedAt)
	}
	if newPaid.PlanCode != pro.Slug || !newPaid.ValidFrom.Equal(upgradedAt) || newPaid.ValidTo == nil || !newPaid.ValidTo.Equal(proExpiry) {
		t.Errorf("new paid interval = (%s, valid_from %s, valid_to %v), want (%s, %s, %s)",
			newPaid.PlanCode, newPaid.ValidFrom, newPaid.ValidTo, pro.Slug, upgradedAt, proExpiry)
	}
	if closedPaid.ValidTo != nil && !newPaid.ValidFrom.Equal(*closedPaid.ValidTo) {
		t.Errorf("new interval starts at %s but the previous one ends at %s: the switch must be contiguous",
			newPaid.ValidFrom, *closedPaid.ValidTo)
	}
	assertNoPlanPeriodOverlap(t, dbConn, userID)

	// The current look-up must return the new finite interval even though it has
	// a non-NULL valid_to; that is the property the old query got wrong.
	current, err := periodRepo.GetCurrentByUserIDWithLock(ctx, userID)
	if err != nil {
		t.Fatalf("read current interval after upgrade: %v", err)
	}
	if current == nil || current.ID != newPaid.ID || current.PlanCode != pro.Slug {
		t.Fatalf("current interval after upgrade = %v, want the new %s interval %s", current, pro.Slug, newPaid.ID)
	}

	// 3. Extend the new holding: valid_to moves, no row is inserted (FR-004).
	extendedExpiry := proExpiry.Add(30 * 24 * time.Hour)
	sub.ExpiresAt = &extendedExpiry
	record(t, &entities.UserSubscriptionEvent{
		EventKey:    "031-regression:extend:" + userID.String(),
		UserID:      userID,
		EventType:   string(subscriptioneventtype.EXTENDED),
		OccurredAt:  proExpiry,
		EffectiveAt: proExpiry,
	}, sub, pro)

	periods = planPeriodHistory(t, periodRepo, ctx, userID)
	if len(periods) != 2 {
		t.Fatalf("after extend history has %d interval(s), want still 2: a renewal must not split one holding; rows: %s",
			len(periods), planPeriodSummary(periods))
	}
	if periods[1].PlanCode != pro.Slug || periods[1].ValidTo == nil || !periods[1].ValidTo.Equal(extendedExpiry) {
		t.Errorf("extended interval = (%s, valid_to %v), want (%s, valid_to %s)",
			periods[1].PlanCode, periods[1].ValidTo, pro.Slug, extendedExpiry)
	}
	if periods[0].ValidTo == nil || !periods[0].ValidTo.Equal(upgradedAt) {
		t.Errorf("extend moved the closed interval's valid_to to %v; it must stay %s", periods[0].ValidTo, upgradedAt)
	}
	assertNoPlanPeriodOverlap(t, dbConn, userID)
}

func planPeriodHistory(t *testing.T, repo repositories.IUserPlanPeriodRepository, ctx context.Context, userID uuid.UUID) []*entities.UserPlanPeriod {
	t.Helper()
	periods, err := repo.GetByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("read history for %s: %v", userID, err)
	}
	return periods
}

func planPeriodSummary(periods []*entities.UserPlanPeriod) string {
	summary := ""
	for i, period := range periods {
		if i > 0 {
			summary += ", "
		}
		summary += period.PlanCode + " [" + period.ValidFrom.String()
		if period.ValidTo == nil {
			summary += ", open)"
		} else {
			summary += ", " + period.ValidTo.String() + ")"
		}
	}
	return summary
}

func assertNoPlanPeriodOverlap(t *testing.T, dbConn *gorm.DB, userID uuid.UUID) {
	t.Helper()
	var overlaps int64
	err := dbConn.Raw(`
		SELECT COUNT(*)
		FROM subscription.user_plan_periods AS a
		JOIN subscription.user_plan_periods AS b
		  ON a.user_id = b.user_id
		 AND a.id < b.id
		 AND tstzrange(a.valid_from, a.valid_to, '[)') && tstzrange(b.valid_from, b.valid_to, '[)')
		WHERE a.user_id = ?`, userID).Scan(&overlaps).Error
	if err != nil {
		t.Fatalf("count overlapping intervals for %s: %v", userID, err)
	}
	if overlaps != 0 {
		t.Fatalf("user %s has %d overlapping interval pair(s); the recorder must close the previous interval before opening the next", userID, overlaps)
	}
}

func planPeriodDurationDays(days int) *int {
	return &days
}
