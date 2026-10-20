package persistence

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	accountmigrations "smart-wardrobe-be/services/account/database/migrations"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// 031 T009 — foundational Postgres coverage for the validity-interval table.
//
// The migration tests need a database that is EMPTY of the feature's schema and
// whose data can be seeded BEFORE the migration runs: the backfill can only be
// observed on rows that existed at migration time, and the preflight must be
// shown to abort a migration rather than skip a corrupt window. The shared
// TEST_DATABASE_URL is already migrated and shared with the rest of the suite,
// so these tests create a throwaway database on the same server and run the
// migration's Up section against it. The repository tests use the same scratch
// database, so they exercise the schema production actually ships.
//
// The harness skips (never fails) when TEST_DATABASE_URL is absent, when it is
// not a URL DSN, or when the connected role lacks CREATEDB — the same
// "environment not provisioned" convention as every Postgres-backed test in
// this service.

const plan031MigrationFile = "20261020150000_add_user_plan_periods.sql"

// TestIntegrationUserPlanPeriodsBackfillAndCutover proves the migration inserts
// exactly one interval per existing subscription, snapshots the plan name, and
// writes the cutover row (031 data-model §4).
func TestIntegrationUserPlanPeriodsBackfillAndCutover(t *testing.T) {
	_, db, closeScratch, closeBase := openPlan031ScratchDB(t)
	defer closeBase()
	defer closeScratch()

	createPlan031BaseSchema(t, db)

	freePlanID := seedPlan031Plan(t, db, "Free")
	premiumPlanID := seedPlan031Plan(t, db, "Premium Monthly")

	premiumUser := seedPlan031User(t, db)
	freeUser := seedPlan031User(t, db)
	futureUser := seedPlan031User(t, db)

	premiumStart := time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)
	premiumExpiry := time.Date(2026, 10, 1, 8, 0, 0, 0, time.UTC)
	freeStart := time.Date(2026, 9, 15, 8, 0, 0, 0, time.UTC)
	futureStart := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	futureExpiry := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)

	seedPlan031Subscription(t, db, premiumUser, premiumPlanID, "premium-monthly", 1, 1, premiumStart, &premiumExpiry)
	seedPlan031Subscription(t, db, freeUser, freePlanID, "free", 0, 0, freeStart, nil)
	seedPlan031Subscription(t, db, futureUser, premiumPlanID, "premium-monthly", 1, 1, futureStart, &futureExpiry)

	before := time.Now().UTC().Add(-time.Minute)
	if err := applyPlan031Migration(t, db); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	after := time.Now().UTC().Add(time.Minute)

	var intervalCount int64
	if err := db.Raw("SELECT COUNT(*) FROM subscription.user_plan_periods").Scan(&intervalCount).Error; err != nil {
		t.Fatalf("count intervals: %v", err)
	}
	if intervalCount != 3 {
		t.Fatalf("backfill created %d interval(s), want exactly one per subscription (3)", intervalCount)
	}

	repo := NewUserPlanPeriodRepository(db)
	assertBackfilledInterval(t, repo, premiumUser, premiumPlanID, "premium-monthly", "Premium Monthly", 1, 1, premiumStart, &premiumExpiry)
	assertBackfilledInterval(t, repo, freeUser, freePlanID, "free", "Free", 0, 0, freeStart, nil)
	assertBackfilledInterval(t, repo, futureUser, premiumPlanID, "premium-monthly", "Premium Monthly", 1, 1, futureStart, &futureExpiry)

	meta, err := repo.GetHistoryCutover(context.Background())
	if err != nil {
		t.Fatalf("read cutover: %v", err)
	}
	if meta == nil {
		t.Fatal("the migration must write the cutover row; GetHistoryCutover returned nil")
	}
	if meta.StartedAt.Before(before) || meta.StartedAt.After(after) {
		t.Errorf("cutover = %s, want an instant inside the migration window [%s, %s]", meta.StartedAt, before, after)
	}

	// FR-001: a later catalogue rename must not rewrite the history the user
	// actually saw.
	if err := db.Exec("UPDATE subscription.subscription_plans SET name = ? WHERE id = ?", "Renamed Later", premiumPlanID).Error; err != nil {
		t.Fatalf("rename plan: %v", err)
	}
	periods, err := repo.GetByUserID(context.Background(), premiumUser)
	if err != nil {
		t.Fatalf("read history after rename: %v", err)
	}
	if len(periods) != 1 {
		t.Fatalf("history after rename has %d row(s), want 1", len(periods))
	}
	if periods[0].PlanName != "Premium Monthly" {
		t.Errorf("plan_name = %q after the catalogue rename, want the snapshot %q", periods[0].PlanName, "Premium Monthly")
	}
}

// TestIntegrationUserPlanPeriodsOverlapGuard proves the adaptive constraint:
// the exclusion variant rejects overlapping intervals with 23P01, the fallback
// variant rejects a second open interval with 23505 (031 data-model §1).
func TestIntegrationUserPlanPeriodsOverlapGuard(t *testing.T) {
	_, db, closeScratch, closeBase := openPlan031ScratchDB(t)
	defer closeBase()
	defer closeScratch()

	createPlan031BaseSchema(t, db)
	freePlanID := seedPlan031Plan(t, db, "Free")
	userID := seedPlan031User(t, db)
	seedPlan031Subscription(t, db, userID, freePlanID, "free", 0, 0, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), nil)

	if err := applyPlan031Migration(t, db); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	exclusion := plan031HasExclusionConstraint(t, db)
	fallback := plan031HasFallbackIndex(t, db)
	if !exclusion && !fallback {
		t.Fatal("neither ex_user_plan_periods_no_overlap nor ux_user_plan_periods_open exists; the non-overlap guard is missing in both variants")
	}

	err := db.Exec(`
		INSERT INTO subscription.user_plan_periods
			(user_id, subscription_plan_id, plan_code, plan_name, tier_rank, plan_kind, valid_from, valid_to, source_event_key)
		VALUES (?, ?, 'free', 'Free', 0, 0, now() - interval '1 hour', NULL, 'OVERLAP-PROBE')`,
		userID, freePlanID).Error
	if err == nil {
		t.Fatal("a second open interval for the same user must be rejected")
	}

	if exclusion {
		if !strings.Contains(err.Error(), "23P01") {
			t.Errorf("exclusion variant rejected the overlap with %v, want SQLSTATE 23P01", err)
		}
	} else {
		if !strings.Contains(err.Error(), "23505") {
			t.Errorf("fallback variant rejected the second open interval with %v, want SQLSTATE 23505", err)
		}
	}
}

// TestIntegrationUserPlanPeriodsFallbackWithoutExtensionPrivilege runs the
// migration as a role that cannot create btree_gist and asserts the documented
// fallback: no exclusion constraint, ux_user_plan_periods_open present, and a
// second open interval rejected with 23505.
//
// It needs a superuser to create the throwaway role; on a managed database
// without that privilege the subtest skips rather than pretending it ran.
func TestIntegrationUserPlanPeriodsFallbackWithoutExtensionPrivilege(t *testing.T) {
	base, db, closeScratch, closeBase := openPlan031ScratchDB(t)
	defer closeBase()

	var isSuperuser bool
	if err := db.Raw("SELECT rolsuper FROM pg_roles WHERE rolname = current_user").Scan(&isSuperuser).Error; err != nil {
		t.Fatalf("read role attributes: %v", err)
	}
	if !isSuperuser {
		closeScratch()
		t.Skip("fallback migration test needs a superuser to create a restricted role")
	}

	roleName := fmt.Sprintf("plan031_restricted_%d", time.Now().UnixNano())
	if err := base.Exec("CREATE ROLE " + quotePlan031Identifier(roleName) + " NOLOGIN").Error; err != nil {
		closeScratch()
		t.Fatalf("create restricted role: %v", err)
	}
	defer func() {
		// Drop the database first: the role owns the tables it created there,
		// and DROP ROLE refuses while it still owns objects.
		closeScratch()
		if err := base.Exec("DROP ROLE IF EXISTS " + quotePlan031Identifier(roleName)).Error; err != nil {
			t.Errorf("drop restricted role: %v", err)
		}
	}()

	createPlan031BaseSchema(t, db)

	quotedRole := quotePlan031Identifier(roleName)
	grants := []string{
		"GRANT USAGE, CREATE ON SCHEMA subscription TO " + quotedRole,
		"GRANT USAGE ON SCHEMA identity TO " + quotedRole,
		"GRANT SELECT, REFERENCES ON subscription.subscription_plans TO " + quotedRole,
		"GRANT SELECT, REFERENCES ON subscription.user_subscriptions TO " + quotedRole,
		"GRANT SELECT, REFERENCES ON identity.users TO " + quotedRole,
	}
	for _, grant := range grants {
		if err := db.Exec(grant).Error; err != nil {
			t.Fatalf("grant %q: %v", grant, err)
		}
	}

	freePlanID := seedPlan031Plan(t, db, "Free")
	userID := seedPlan031User(t, db)
	seedPlan031Subscription(t, db, userID, freePlanID, "free", 0, 0, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), nil)

	// Fresh databases may inherit btree_gist from template1. Removing it is what
	// forces the CREATE EXTENSION branch to be attempted (and refused).
	if err := db.Exec("DROP EXTENSION IF EXISTS btree_gist").Error; err != nil {
		t.Fatalf("drop inherited btree_gist: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql.DB: %v", err)
	}
	ctx := context.Background()
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatalf("pin connection: %v", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "SET ROLE "+quotedRole); err != nil {
		t.Fatalf("set restricted role: %v", err)
	}
	defer func() { _, _ = conn.ExecContext(ctx, "RESET ROLE") }()

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin migration transaction: %v", err)
	}
	for _, statement := range splitPlan031Statements(plan031MigrationUp(t)) {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			_ = tx.Rollback()
			t.Fatalf("fallback migration statement failed: %v\nstatement: %s", err, statement)
		}
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit fallback migration: %v", err)
	}

	var exclusionCount int64
	if err := db.Raw("SELECT COUNT(*) FROM pg_constraint WHERE conname = 'ex_user_plan_periods_no_overlap'").Scan(&exclusionCount).Error; err != nil {
		t.Fatalf("read exclusion constraint: %v", err)
	}
	if exclusionCount != 0 {
		t.Fatal("the restricted role created the exclusion constraint; the test did not exercise the fallback")
	}
	if !plan031HasFallbackIndex(t, db) {
		t.Fatal("ux_user_plan_periods_open is missing after the fallback migration")
	}

	err = db.Exec(`
		INSERT INTO subscription.user_plan_periods
			(user_id, subscription_plan_id, plan_code, plan_name, tier_rank, plan_kind, valid_from, valid_to, source_event_key)
		VALUES (?, ?, 'free', 'Free', 0, 0, now() - interval '1 hour', NULL, 'OVERLAP-PROBE')`,
		userID, freePlanID).Error
	if err == nil {
		t.Fatal("the fallback variant must reject a second open interval")
	}
	if !strings.Contains(err.Error(), "23505") {
		t.Errorf("fallback variant rejected the second open interval with %v, want SQLSTATE 23505", err)
	}
}

// TestIntegrationUserPlanPeriodsPreflightAborts proves a corrupt finite window
// stops the migration with the offending user in the message, instead of being
// skipped and silently removing that user from every report (data-model §4).
func TestIntegrationUserPlanPeriodsPreflightAborts(t *testing.T) {
	_, db, closeScratch, closeBase := openPlan031ScratchDB(t)
	defer closeBase()
	defer closeScratch()

	createPlan031BaseSchema(t, db)
	planID := seedPlan031Plan(t, db, "Premium Monthly")
	userID := seedPlan031User(t, db)
	startedAt := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	expiresAt := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	seedPlan031Subscription(t, db, userID, planID, "premium-monthly", 1, 1, startedAt, &expiresAt)

	err := applyPlan031Migration(t, db)
	if err == nil {
		t.Fatal("the migration must abort on a finite subscription whose expires_at <= started_at")
	}
	if !strings.Contains(err.Error(), userID.String()) {
		t.Errorf("migration error %q does not name the offending user_id %s", err, userID)
	}
	if !strings.Contains(err.Error(), "expires_at <= started_at") {
		t.Errorf("migration error %q does not explain the corrupt window", err)
	}

	var tableExists bool
	if err := db.Raw("SELECT to_regclass('subscription.user_plan_periods') IS NOT NULL").Scan(&tableExists).Error; err != nil {
		t.Fatalf("check rolled-back table: %v", err)
	}
	if tableExists {
		t.Error("the failed migration left subscription.user_plan_periods behind; the Up section must run in one transaction")
	}
}

// TestIntegrationUserPlanPeriodsDistributionCountsDistinct covers the report
// query: COUNT(DISTINCT user_id) for a user who left and returned to the same
// plan, the four bound shapes, and the snapshot semantics.
func TestIntegrationUserPlanPeriodsDistributionCountsDistinct(t *testing.T) {
	_, db, closeScratch, closeBase := openPlan031ScratchDB(t)
	defer closeBase()
	defer closeScratch()

	createPlan031BaseSchema(t, db)
	premiumPlanID := seedPlan031Plan(t, db, "Premium Monthly")
	freePlanID := seedPlan031Plan(t, db, "Free")

	userReturning := seedPlan031User(t, db)
	userContinuous := seedPlan031User(t, db)
	userFree := seedPlan031User(t, db)
	userFuture := seedPlan031User(t, db)

	// Relative instants keep the snapshot and one-sided assertions true on any
	// run date: the controlled intervals are in the past, the free one is open,
	// and the only future row starts after now.
	now := time.Now().UTC()
	backfillStart := now.Add(-365 * 24 * time.Hour)
	backfillExpiry := now.Add(-300 * 24 * time.Hour)
	for _, userID := range []uuid.UUID{userReturning, userContinuous, userFuture} {
		seedPlan031Subscription(t, db, userID, premiumPlanID, "premium-monthly", 1, 1, backfillStart, &backfillExpiry)
	}
	seedPlan031Subscription(t, db, userFree, freePlanID, "free", 0, 0, backfillStart, nil)

	if err := applyPlan031Migration(t, db); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	if err := db.Exec("DELETE FROM subscription.user_plan_periods").Error; err != nil {
		t.Fatalf("clear backfill rows: %v", err)
	}

	insertPeriod := func(userID, planID uuid.UUID, code, name string, kind int16, from time.Time, to *time.Time, key string) {
		t.Helper()
		if err := db.Exec(`
			INSERT INTO subscription.user_plan_periods
				(user_id, subscription_plan_id, plan_code, plan_name, tier_rank, plan_kind, valid_from, valid_to, source_event_key)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			// Fixtures use tier_rank == plan_kind; both slots get the same value.
			userID, planID, code, name, kind, kind, from, to, key).Error; err != nil {
			t.Fatalf("insert interval %s: %v", key, err)
		}
	}

	premiumFrom := now.Add(-180 * 24 * time.Hour)
	premiumMidEnd := now.Add(-120 * 24 * time.Hour)
	premiumAgainFrom := now.Add(-90 * 24 * time.Hour)
	premiumEnd := now.Add(-30 * 24 * time.Hour)
	futureFrom := now.Add(30 * 24 * time.Hour)

	// Same plan, two disjoint intervals: must count once (FR-010).
	insertPeriod(userReturning, premiumPlanID, "premium-monthly", "Premium Monthly", 1, premiumFrom, &premiumMidEnd, "T009:returning:1")
	insertPeriod(userReturning, premiumPlanID, "premium-monthly", "Premium Monthly", 1, premiumAgainFrom, &premiumEnd, "T009:returning:2")
	insertPeriod(userContinuous, premiumPlanID, "premium-monthly", "Premium Monthly", 1, premiumFrom, &premiumEnd, "T009:continuous")
	insertPeriod(userFree, freePlanID, "free", "Free", 0, premiumFrom, nil, "T009:free")
	// Future-dated and open: present only when the period reaches it.
	insertPeriod(userFuture, premiumPlanID, "premium-monthly", "Premium Monthly", 1, futureFrom, nil, "T009:future")

	repo := NewUserPlanPeriodRepository(db)
	ctx := context.Background()

	both := plan031Counts(t, repo, ctx, premiumFrom, premiumEnd)
	if both["premium-monthly"] != 2 {
		t.Errorf("[both bounds] premium = %d, want 2 (the returning user counts once)", both["premium-monthly"])
	}
	if both["free"] != 1 {
		t.Errorf("[both bounds] free = %d, want 1", both["free"])
	}

	fromOnly := plan031Counts(t, repo, ctx, premiumFrom, time.Time{})
	if fromOnly["premium-monthly"] != 2 {
		t.Errorf("[from only] premium = %d, want 2; a future-dated interval must be excluded by valid_from <= now()", fromOnly["premium-monthly"])
	}
	if fromOnly["free"] != 1 {
		t.Errorf("[from only] free = %d, want 1", fromOnly["free"])
	}
	if _, ok := fromOnly["premium-monthly"]; !ok {
		t.Fatal("premium bucket is absent; the following assertions would be vacuous")
	}

	toOnly := plan031Counts(t, repo, ctx, time.Time{}, premiumEnd)
	if toOnly["premium-monthly"] != 2 {
		t.Errorf("[to only] premium = %d, want 2", toOnly["premium-monthly"])
	}
	if toOnly["free"] != 1 {
		t.Errorf("[to only] free = %d, want 1", toOnly["free"])
	}

	snapshot := plan031Counts(t, repo, ctx, time.Time{}, time.Time{})
	if snapshot["free"] != 1 {
		t.Errorf("[snapshot] free = %d, want 1", snapshot["free"])
	}
	if snapshot["premium-monthly"] != 0 {
		t.Errorf("[snapshot] premium = %d, want 0: the only premium interval starts in the future and cannot be a current holding", snapshot["premium-monthly"])
	}
}

// TestIntegrationUserPlanPeriodsHistoryByUser covers US2: the documented
// read-only per-user look-up must return what an operator needs to settle a
// complaint. The test builds the acceptance lifecycle (activate → extend →
// upgrade → downgrade) through the repository write methods, then reads the
// history back and asserts chronological order, contiguous
// continuous switches, one row per holding (EXTEND moves valid_to only), open
// free/lifetime intervals, and an honest gap after a lapse.
func TestIntegrationUserPlanPeriodsHistoryByUser(t *testing.T) {
	_, db, closeScratch, closeBase := openPlan031ScratchDB(t)
	defer closeBase()
	defer closeScratch()

	createPlan031BaseSchema(t, db)
	if err := applyPlan031Migration(t, db); err != nil {
		t.Fatalf("apply migration: %v", err)
	}

	repo := NewUserPlanPeriodRepository(db)
	ctx := context.Background()

	premiumPlanID := seedPlan031Plan(t, db, "Premium Monthly")
	proPlanID := seedPlan031Plan(t, db, "Pro Yearly")
	freePlanID := seedPlan031Plan(t, db, "Free")
	lifetimePlanID := seedPlan031Plan(t, db, "Lifetime")

	t.Run("four-step lifecycle is ordered and contiguous", func(t *testing.T) {
		userID := seedPlan031User(t, db)
		activatedAt := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
		firstExpiry := time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC)
		extendedExpiry := time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC)
		upgradedAt := time.Date(2026, 2, 15, 8, 0, 0, 0, time.UTC)
		downgradedAt := time.Date(2026, 4, 15, 8, 0, 0, 0, time.UTC)

		// 1. Activate: one paid interval [activatedAt, firstExpiry).
		mustCreatePlan031Period(t, repo, ctx, newPlan031Period(
			userID, premiumPlanID, "premium-monthly", "Premium Monthly",
			plankind.Finite, 1, activatedAt, &firstExpiry, "T024:activate",
		))

		// 2. Extend the same holding: valid_to moves, no row is inserted (FR-004).
		current := mustCurrentPlan031Period(t, repo, ctx, userID)
		if err := repo.UpdateValidTo(ctx, current.ID, &extendedExpiry); err != nil {
			t.Fatalf("extend interval: %v", err)
		}
		afterExtend, err := repo.GetByUserID(ctx, userID)
		if err != nil {
			t.Fatalf("read history after extend: %v", err)
		}
		if len(afterExtend) != 1 {
			t.Fatalf("EXTENDED produced %d row(s), want still 1: a renewal must not split one holding", len(afterExtend))
		}
		if afterExtend[0].ValidTo == nil || !afterExtend[0].ValidTo.Equal(extendedExpiry) {
			t.Errorf("valid_to after extend = %v, want %s", afterExtend[0].ValidTo, extendedExpiry)
		}

		// 3. Upgrade at upgradedAt: close the finite current interval before its
		// contractual expiry and open the new plan at the same instant, so the
		// two periods connect. CloseCurrent must move the existing valid_to
		// earlier; the recorder no longer works around it with a full-row Update.
		mustCloseCurrentPlan031Period(t, repo, ctx, userID, upgradedAt)
		mustCreatePlan031Period(t, repo, ctx, newPlan031Period(
			userID, proPlanID, "pro-yearly", "Pro Yearly",
			plankind.Finite, 2, upgradedAt, &downgradedAt, "T024:upgrade",
		))

		// 4. Downgrade at downgradedAt: close the paid plan and open free with
		// valid_to NULL (open-ended).
		mustCloseCurrentPlan031Period(t, repo, ctx, userID, downgradedAt)
		mustCreatePlan031Period(t, repo, ctx, newPlan031Period(
			userID, freePlanID, "free", "Free",
			plankind.DefaultFree, 0, downgradedAt, nil, "T024:downgrade",
		))

		periods, err := repo.GetByUserID(ctx, userID)
		if err != nil {
			t.Fatalf("read lifecycle history: %v", err)
		}
		plan031AssertHistory(t, periods,
			[]string{"premium-monthly", "pro-yearly", "free"},
			[]time.Time{activatedAt, upgradedAt, downgradedAt},
		)
		if periods[len(periods)-1].ValidTo != nil {
			t.Errorf("free interval valid_to = %s, want NULL after the downgrade", periods[len(periods)-1].ValidTo)
		}

		// The downgrade ends with an open-ended interval, so the FOR UPDATE
		// look-up must find exactly that row.
		current = mustCurrentPlan031Period(t, repo, ctx, userID)
		if current.PlanCode != "free" || current.ValidTo != nil {
			t.Errorf("current interval after downgrade = (%s, %v), want the open free row", current.PlanCode, current.ValidTo)
		}
	})

	t.Run("a lapse leaves a gap", func(t *testing.T) {
		userID := seedPlan031User(t, db)
		firstStart := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
		firstEnd := time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC)
		secondStart := time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC)
		secondEnd := time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC)

		// Insert the later window first: the read must sort by valid_from,
		// not return rows in insertion order.
		mustCreatePlan031Period(t, repo, ctx, newPlan031Period(
			userID, premiumPlanID, "premium-monthly", "Premium Monthly",
			plankind.Finite, 1, secondStart, &secondEnd, "T024:lapse:second",
		))
		mustCreatePlan031Period(t, repo, ctx, newPlan031Period(
			userID, premiumPlanID, "premium-monthly", "Premium Monthly",
			plankind.Finite, 1, firstStart, &firstEnd, "T024:lapse:first",
		))

		periods, err := repo.GetByUserID(ctx, userID)
		if err != nil {
			t.Fatalf("read lapsed history: %v", err)
		}
		if len(periods) != 2 {
			t.Fatalf("lapsed history has %d interval(s), want 2", len(periods))
		}
		if periods[0].ValidTo == nil {
			t.Fatal("the first interval is open; a lapse cannot be represented")
		}
		if !periods[1].ValidFrom.After(*periods[0].ValidTo) {
			t.Errorf("second interval starts at %s, want after the first ended at %s: a lapse must leave a gap, not be invented away",
				periods[1].ValidFrom, *periods[0].ValidTo)
		}
	})

	t.Run("free and lifetime intervals are open-ended", func(t *testing.T) {
		freeUser := seedPlan031User(t, db)
		lifetimeUser := seedPlan031User(t, db)
		start := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)

		mustCreatePlan031Period(t, repo, ctx, newPlan031Period(
			freeUser, freePlanID, "free", "Free",
			plankind.DefaultFree, 0, start, nil, "T024:free",
		))
		mustCreatePlan031Period(t, repo, ctx, newPlan031Period(
			lifetimeUser, lifetimePlanID, "lifetime", "Lifetime",
			plankind.Lifetime, 9, start, nil, "T024:lifetime",
		))

		for name, userID := range map[string]uuid.UUID{"free": freeUser, "lifetime": lifetimeUser} {
			periods, err := repo.GetByUserID(ctx, userID)
			if err != nil {
				t.Fatalf("read %s history: %v", name, err)
			}
			if len(periods) != 1 {
				t.Fatalf("%s history has %d interval(s), want 1", name, len(periods))
			}
			if periods[0].ValidTo != nil {
				t.Errorf("%s interval valid_to = %s, want NULL: an unlimited plan never expires", name, periods[0].ValidTo)
			}
		}
	})
}

// ---- migration harness -----------------------------------------------------

// TestPlan031MigrationSplitterKeepsDollarQuotedBlocks is a no-database guard for
// the harness above: a `;` inside a DO body must not split the statement, or the
// migration tests would execute half a PL/pgSQL block and report a parse error
// that has nothing to do with the migration.
func TestPlan031MigrationSplitterKeepsDollarQuotedBlocks(t *testing.T) {
	statements := splitPlan031Statements(plan031MigrationUp(t))

	const wantStatements = 8 // 2 tables + 3 indexes + adaptive + preflight + backfill
	if len(statements) != wantStatements {
		t.Fatalf("split the Up section into %d statement(s), want %d:\n%v", len(statements), wantStatements, statements)
	}
	for _, statement := range statements {
		if strings.HasPrefix(statement, "DO") && !strings.Contains(statement, "$$") {
			t.Errorf("a DO block lost its dollar quoting during the split: %q", statement)
		}
	}
}

func plan031MigrationUp(t *testing.T) string {
	t.Helper()
	content, err := accountmigrations.EmbedFS.ReadFile(plan031MigrationFile)
	if err != nil {
		t.Fatalf("read embedded migration %s: %v", plan031MigrationFile, err)
	}
	sql := string(content)
	upIndex := strings.Index(sql, "-- +goose Up")
	downIndex := strings.Index(sql, "-- +goose Down")
	if upIndex < 0 || downIndex < 0 || downIndex <= upIndex {
		t.Fatalf("%s does not carry the expected goose Up/Down markers", plan031MigrationFile)
	}
	return sql[upIndex+len("-- +goose Up") : downIndex]
}

func applyPlan031Migration(t *testing.T, db *gorm.DB) error {
	t.Helper()
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	for _, statement := range splitPlan031Statements(plan031MigrationUp(t)) {
		if err := tx.Exec(statement).Error; err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

// splitPlan031Statements divides the migration into executable statements. It
// understands dollar-quoted DO blocks and single-quoted literals, which is the
// minimum needed to keep a `;` inside either from ending a statement. Line
// comments are dropped; the migration uses no block comments.
func splitPlan031Statements(sql string) []string {
	var statements []string
	var builder strings.Builder
	inSingleQuote := false
	inDollarQuote := false
	dollarTag := ""

	for i := 0; i < len(sql); i++ {
		ch := sql[i]

		if inDollarQuote {
			if ch == '$' && strings.HasPrefix(sql[i:], dollarTag) {
				builder.WriteString(dollarTag)
				i += len(dollarTag) - 1
				inDollarQuote = false
				continue
			}
			builder.WriteByte(ch)
			continue
		}

		if inSingleQuote {
			builder.WriteByte(ch)
			if ch == '\'' {
				if i+1 < len(sql) && sql[i+1] == '\'' {
					builder.WriteByte(sql[i+1])
					i++
					continue
				}
				inSingleQuote = false
			}
			continue
		}

		if ch == '-' && i+1 < len(sql) && sql[i+1] == '-' {
			for i < len(sql) && sql[i] != '\n' {
				i++
			}
			builder.WriteByte('\n')
			continue
		}

		if ch == '\'' {
			inSingleQuote = true
			builder.WriteByte(ch)
			continue
		}

		if ch == '$' {
			end := i + 1
			for end < len(sql) && (isPlan031IdentifierByte(sql[end])) {
				end++
			}
			if end < len(sql) && sql[end] == '$' {
				dollarTag = sql[i : end+1]
				builder.WriteString(dollarTag)
				i = end
				inDollarQuote = true
				continue
			}
		}

		if ch == ';' {
			if statement := strings.TrimSpace(builder.String()); statement != "" {
				statements = append(statements, statement)
			}
			builder.Reset()
			continue
		}

		builder.WriteByte(ch)
	}

	if statement := strings.TrimSpace(builder.String()); statement != "" {
		statements = append(statements, statement)
	}
	return statements
}

func isPlan031IdentifierByte(b byte) bool {
	return b == '_' ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9')
}

func openPlan031ScratchDB(t *testing.T) (base, scratch *gorm.DB, closeScratch, closeBase func()) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping plan-period migration integration test")
	}
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Scheme == "" || strings.TrimPrefix(parsed.Path, "/") == "" {
		t.Skip("TEST_DATABASE_URL must be a postgres URL DSN for the migration harness")
	}

	base, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}

	name := fmt.Sprintf("plan031_t009_%d", time.Now().UnixNano())
	if err := base.Exec("CREATE DATABASE " + quotePlan031Identifier(name)).Error; err != nil {
		if sqlDB, dbErr := base.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		t.Skipf("cannot create the scratch database (the TEST_DATABASE_URL role needs CREATEDB): %v", err)
	}

	scratchURL := *parsed
	scratchURL.Path = "/" + name
	query := scratchURL.Query()
	query.Set("search_path", "public,identity,subscription")
	scratchURL.RawQuery = query.Encode()

	scratch, err = gorm.Open(postgres.Open(scratchURL.String()), &gorm.Config{})
	if err != nil {
		_ = base.Exec("DROP DATABASE IF EXISTS " + quotePlan031Identifier(name) + " WITH (FORCE)").Error
		if sqlDB, dbErr := base.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		t.Fatalf("connect scratch database: %v", err)
	}

	closeScratch = func() {
		if sqlDB, dbErr := scratch.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		if err := base.Exec("DROP DATABASE IF EXISTS " + quotePlan031Identifier(name) + " WITH (FORCE)").Error; err != nil {
			t.Logf("drop scratch database %s: %v", name, err)
		}
	}
	closeBase = func() {
		if sqlDB, dbErr := base.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	}
	return base, scratch, closeScratch, closeBase
}

func createPlan031BaseSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	statements := []string{
		"CREATE SCHEMA IF NOT EXISTS identity",
		"CREATE SCHEMA IF NOT EXISTS subscription",
		"CREATE TABLE IF NOT EXISTS identity.users (id uuid PRIMARY KEY)",
		"CREATE TABLE IF NOT EXISTS subscription.subscription_plans (id uuid PRIMARY KEY, name varchar(100) NOT NULL)",
		`CREATE TABLE IF NOT EXISTS subscription.user_subscriptions (
			user_id uuid PRIMARY KEY,
			subscription_plan_id uuid NOT NULL,
			current_plan_code varchar(100) NOT NULL,
			current_tier_rank integer NOT NULL,
			current_plan_kind smallint NOT NULL,
			started_at timestamptz NOT NULL,
			expires_at timestamptz
		)`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create minimal base schema: %v\nstatement: %s", err, statement)
		}
	}
}

func seedPlan031Plan(t *testing.T, db *gorm.DB, name string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if err := db.Exec("INSERT INTO subscription.subscription_plans (id, name) VALUES (?, ?)", id, name).Error; err != nil {
		t.Fatalf("seed plan %q: %v", name, err)
	}
	return id
}

func seedPlan031User(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	if err := db.Exec("INSERT INTO identity.users (id) VALUES (?)", id).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return id
}

func seedPlan031Subscription(
	t *testing.T,
	db *gorm.DB,
	userID, planID uuid.UUID,
	planCode string,
	tierRank int,
	planKind int,
	startedAt time.Time,
	expiresAt *time.Time,
) {
	t.Helper()
	if err := db.Exec(`
		INSERT INTO subscription.user_subscriptions
			(user_id, subscription_plan_id, current_plan_code, current_tier_rank, current_plan_kind, started_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID, planID, planCode, tierRank, planKind, startedAt, expiresAt).Error; err != nil {
		t.Fatalf("seed subscription: %v", err)
	}
}

func assertBackfilledInterval(
	t *testing.T,
	repo repositories.IUserPlanPeriodRepository,
	userID, planID uuid.UUID,
	planCode, planName string,
	tierRank int,
	planKind int16,
	validFrom time.Time,
	validTo *time.Time,
) {
	t.Helper()
	periods, err := repo.GetByUserID(context.Background(), userID)
	if err != nil {
		t.Fatalf("read backfilled interval for %s: %v", userID, err)
	}
	if len(periods) != 1 {
		t.Fatalf("user %s has %d interval(s) after backfill, want exactly 1", userID, len(periods))
	}
	period := periods[0]
	if period.SubscriptionPlanID != planID || period.PlanCode != planCode || period.PlanName != planName {
		t.Errorf("backfilled plan snapshot = (%s, %s, %s), want (%s, %s, %s)",
			period.SubscriptionPlanID, period.PlanCode, period.PlanName, planID, planCode, planName)
	}
	if period.TierRank != tierRank || int16(period.PlanKind) != planKind {
		t.Errorf("backfilled rank/kind = (%d, %d), want (%d, %d)", period.TierRank, period.PlanKind, tierRank, planKind)
	}
	if !period.ValidFrom.Equal(validFrom) {
		t.Errorf("backfilled valid_from = %s, want started_at %s", period.ValidFrom, validFrom)
	}
	if !plan031TimesEqual(period.ValidTo, validTo) {
		t.Errorf("backfilled valid_to = %v, want expires_at %v", period.ValidTo, validTo)
	}
	wantKey := "BACKFILL:" + userID.String()
	if period.SourceEventKey == nil || *period.SourceEventKey != wantKey {
		t.Errorf("backfilled source_event_key = %v, want %q", period.SourceEventKey, wantKey)
	}
}

func newPlan031Period(
	userID, planID uuid.UUID,
	planCode, planName string,
	planKind plankind.PlanKind,
	tierRank int,
	validFrom time.Time,
	validTo *time.Time,
	sourceEventKey string,
) *entities.UserPlanPeriod {
	key := sourceEventKey
	return &entities.UserPlanPeriod{
		UserID:             userID,
		SubscriptionPlanID: planID,
		PlanCode:           planCode,
		PlanName:           planName,
		TierRank:           tierRank,
		PlanKind:           planKind,
		ValidFrom:          validFrom,
		ValidTo:            validTo,
		SourceEventKey:     &key,
	}
}

func mustCreatePlan031Period(t *testing.T, repo repositories.IUserPlanPeriodRepository, ctx context.Context, period *entities.UserPlanPeriod) {
	t.Helper()
	if err := repo.Create(ctx, period); err != nil {
		t.Fatalf("create interval starting %s: %v", period.ValidFrom, err)
	}
}

func mustCloseCurrentPlan031Period(t *testing.T, repo repositories.IUserPlanPeriodRepository, ctx context.Context, userID uuid.UUID, closeAt time.Time) {
	t.Helper()
	current := mustCurrentPlan031Period(t, repo, ctx, userID)
	if err := repo.CloseCurrent(ctx, current.ID, closeAt); err != nil {
		t.Fatalf("close current interval %s at %s: %v", current.ID, closeAt, err)
	}
}

// mustCurrentPlan031Period returns the interval a plan change would close or
// extend. It uses the recorder's own look-up: the row with the greatest
// valid_from, whether open-ended or finite. A paid window carries its
// contractual expiry, so restricting the query to `valid_to IS NULL` would find
// nothing.
func mustCurrentPlan031Period(t *testing.T, repo repositories.IUserPlanPeriodRepository, ctx context.Context, userID uuid.UUID) *entities.UserPlanPeriod {
	t.Helper()
	current, err := repo.GetCurrentByUserIDWithLock(ctx, userID)
	if err != nil {
		t.Fatalf("read current interval for %s: %v", userID, err)
	}
	if current == nil {
		t.Fatalf("user %s has no intervals; the lifecycle fixture is incomplete", userID)
	}
	return current
}

// plan031AssertHistory asserts the read the operator documents: intervals in
// chronological order, each one a positive half-open range, and every interval
// after the first starting exactly where the previous closed (US2).
func plan031AssertHistory(t *testing.T, periods []*entities.UserPlanPeriod, wantCodes []string, wantStarts []time.Time) {
	t.Helper()
	if len(periods) != len(wantCodes) {
		t.Fatalf("history has %d interval(s), want %d", len(periods), len(wantCodes))
	}
	for i, period := range periods {
		if period.PlanCode != wantCodes[i] {
			t.Errorf("interval %d plan_code = %q, want %q; GetByUserID must order by valid_from", i, period.PlanCode, wantCodes[i])
		}
		if !period.ValidFrom.Equal(wantStarts[i]) {
			t.Errorf("interval %d valid_from = %s, want %s", i, period.ValidFrom, wantStarts[i])
		}
		if period.ValidTo != nil && !period.ValidTo.After(period.ValidFrom) {
			t.Errorf("interval %d is not a positive half-open range: [%s, %s)", i, period.ValidFrom, *period.ValidTo)
		}
		if i == 0 {
			continue
		}
		previous := periods[i-1]
		if previous.ValidTo == nil {
			t.Fatalf("interval %d is open-ended but interval %d follows it", i-1, i)
		}
		if !period.ValidFrom.Equal(*previous.ValidTo) {
			t.Errorf("interval %d starts at %s, want the previous end %s: a continuous switch must not leave a gap or an overlap",
				i, period.ValidFrom, *previous.ValidTo)
		}
	}
}

func plan031TimesEqual(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

func plan031HasExclusionConstraint(t *testing.T, db *gorm.DB) bool {
	t.Helper()
	var count int64
	if err := db.Raw("SELECT COUNT(*) FROM pg_constraint WHERE conname = 'ex_user_plan_periods_no_overlap'").Scan(&count).Error; err != nil {
		t.Fatalf("read exclusion constraint: %v", err)
	}
	return count > 0
}

func plan031HasFallbackIndex(t *testing.T, db *gorm.DB) bool {
	t.Helper()
	var exists bool
	if err := db.Raw("SELECT to_regclass('subscription.ux_user_plan_periods_open') IS NOT NULL").Scan(&exists).Error; err != nil {
		t.Fatalf("read fallback index: %v", err)
	}
	return exists
}

func plan031Counts(
	t *testing.T,
	repo repositories.IUserPlanPeriodRepository,
	ctx context.Context,
	from, to time.Time,
) map[string]int64 {
	t.Helper()
	rows, err := repo.GetDistribution(ctx, from, to)
	if err != nil {
		t.Fatalf("GetDistribution(%s, %s): %v", from, to, err)
	}
	counts := make(map[string]int64, len(rows))
	for _, row := range rows {
		counts[row.PlanCode] = row.UserCount
	}
	return counts
}

func quotePlan031Identifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
