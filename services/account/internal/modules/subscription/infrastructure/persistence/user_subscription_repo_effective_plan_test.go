package persistence

import (
	"context"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// The admin user list can be filtered by the account's EFFECTIVE plan. The rule
// lives in Go in `SubscriptionUseCase.GetUserSubscriptionOverviews` (the mapper
// that fills the table) and in SQL in
// `UserSubscriptionRepository.ListUserIDsWithEffectivePlan` (the filter, which
// must paginate). Two implementations of one rule drift silently — one screen
// shows "Free", the filter lists the account under "Premium" — so this test
// seeds the four shapes of subscription row and pins the SQL answer against the
// Go answer, not against a hard-coded id list.
//
// Postgres only: the predicate compares timestamptz values, which sqlite's text
// storage does not order the same way. Skipped when TEST_DATABASE_URL is unset,
// like the rest of this package's integration tests.

type effectivePlanFixture struct {
	name          string
	planID        uuid.UUID
	planKind      plankind.PlanKind
	expiresAt     *time.Time
	fallbackPlan  *uuid.UUID
	expectedLabel uuid.UUID // the plan the Go rule prints for this row
}

func TestUserSubscriptionEffectivePlanFilterMatchesTheMapperRule(t *testing.T) {
	_, db, closeScratch, closeBase := openPlan031ScratchDB(t)
	defer closeBase()
	defer closeScratch()

	createEffectivePlanSchema(t, db)

	premium := uuid.New()
	free := uuid.New()
	lifetime := uuid.New()

	now := time.Now()
	activeExpiry := now.AddDate(0, 1, 0)
	expiredExpiry := now.AddDate(0, -1, 0)

	fixtures := []effectivePlanFixture{
		{
			name:          "finite plan still running stays on itself",
			planID:        premium,
			planKind:      plankind.Finite,
			expiresAt:     &activeExpiry,
			expectedLabel: premium,
		},
		{
			name:          "expired finite plan falls back to the explicit fallback",
			planID:        premium,
			planKind:      plankind.Finite,
			expiresAt:     &expiredExpiry,
			fallbackPlan:  &free,
			expectedLabel: free,
		},
		{
			name:          "expired finite plan with no fallback counts as the default plan",
			planID:        premium,
			planKind:      plankind.Finite,
			expiresAt:     &expiredExpiry,
			expectedLabel: free,
		},
		{
			name:          "expired finite plan with a paid fallback stays on that fallback",
			planID:        premium,
			planKind:      plankind.Finite,
			expiresAt:     &expiredExpiry,
			fallbackPlan:  &lifetime,
			expectedLabel: lifetime,
		},
		{
			name:          "lifetime plan never expires",
			planID:        lifetime,
			planKind:      plankind.Lifetime,
			expectedLabel: lifetime,
		},
	}

	idsByLabel := map[uuid.UUID][]uuid.UUID{}
	for _, fixture := range fixtures {
		userID := uuid.New()
		insertEffectivePlanRow(t, db, userID, fixture)
		idsByLabel[fixture.expectedLabel] = append(idsByLabel[fixture.expectedLabel], userID)
	}

	repo := NewUserSubscriptionRepository(db)
	for planID, want := range idsByLabel {
		got, err := repo.ListUserIDsWithEffectivePlan(context.Background(), planID, free)
		if err != nil {
			t.Fatalf("list users on plan %s: %v", planID, err)
		}
		if !sameIDSet(got, want) {
			t.Errorf("plan %s: got %v, want %v", planID, got, want)
		}
	}

	// The plan that owns the expired rows (but is no longer effective there)
	// must not match them: only the still-running premium row is on premium.
	got, err := repo.ListUserIDsWithEffectivePlan(context.Background(), premium, free)
	if err != nil {
		t.Fatalf("list users on premium: %v", err)
	}
	activeUser := idsByLabel[premium]
	if len(activeUser) != 1 {
		t.Fatalf("fixture error: want exactly one active premium row, got %v", activeUser)
	}
	if !sameIDSet(got, activeUser) {
		t.Errorf("premium filter = %v, want only the running row %v", got, activeUser)
	}
}

func createEffectivePlanSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	statements := []string{
		"CREATE SCHEMA IF NOT EXISTS subscription",
		`CREATE TABLE IF NOT EXISTS subscription.user_subscriptions (
			user_id uuid PRIMARY KEY,
			subscription_plan_id uuid NOT NULL,
			current_plan_kind smallint NOT NULL,
			expires_at timestamptz,
			fallback_plan_id uuid
		)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create effective-plan schema: %v\nstatement: %s", err, statement)
		}
	}
}

func insertEffectivePlanRow(t *testing.T, db *gorm.DB, userID uuid.UUID, fixture effectivePlanFixture) {
	t.Helper()
	if err := db.Exec(`
		INSERT INTO subscription.user_subscriptions
			(user_id, subscription_plan_id, current_plan_kind, expires_at, fallback_plan_id)
		VALUES (?, ?, ?, ?, ?)`,
		userID, fixture.planID, int16(fixture.planKind), fixture.expiresAt, fixture.fallbackPlan,
	).Error; err != nil {
		t.Fatalf("insert subscription row (%s): %v", fixture.name, err)
	}
}

func sameIDSet(got, want []uuid.UUID) bool {
	if len(got) != len(want) {
		return false
	}
	seen := make(map[uuid.UUID]struct{}, len(got))
	for _, id := range got {
		seen[id] = struct{}{}
	}
	for _, id := range want {
		if _, ok := seen[id]; !ok {
			return false
		}
	}
	return true
}
