package subscription

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	accountmigrations "smart-wardrobe-be/services/account/database/migrations"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/infrastructure/persistence"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"
	shareddb "smart-wardrobe-be/shared/infrastructure/db"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// 031 — fallback-variant concurrency coverage.
//
// The shared TEST_DATABASE_URL normally carries the exclusion constraint, so a
// concurrency test there cannot prove the partial-unique fallback is safe. This
// file creates a throwaway database, applies the real 031 migration, then
// replaces the constraint with the documented fallback index - exactly the
// schema a deployment without CREATE EXTENSION privilege gets. Two concurrent
// switches must still leave zero overlap because the recorder takes the
// per-user advisory lock before reading the current interval.

const planRecorderFallbackMigrationFile = "20261020150000_add_user_plan_periods.sql"

// TestIntegrationPlanPeriodRecorderFallbackConcurrentChanges is the fallback
// counterpart of T030. Both transactions reach the advisory lock together; the
// winner closes the initial interval and inserts its own, then the loser reads
// the committed row and replaces it in place at the shared effective instant.
// The fallback index only protects the open-ended row, so a serialization
// failure here would surface as an overlapping pair of closed intervals.
func TestIntegrationPlanPeriodRecorderFallbackConcurrentChanges(t *testing.T) {
	dbConn := openPlanRecorderFallbackDB(t)
	ctx := context.Background()

	if planRecorderHasExclusionConstraint(t, dbConn) {
		t.Fatal("the scratch database still has ex_user_plan_periods_no_overlap; the fallback variant is not being exercised")
	}
	if !planRecorderHasFallbackIndex(t, dbConn) {
		t.Fatal("the scratch database is missing ux_user_plan_periods_open")
	}

	userID := uuid.New()
	if err := dbConn.Exec(`INSERT INTO identity.users (id) VALUES (?)`, userID).Error; err != nil {
		t.Fatalf("seed fixture user: %v", err)
	}

	t0 := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	initialExpiry := t0.Add(60 * 24 * time.Hour)
	changedAt := t0.Add(10 * 24 * time.Hour)
	changedExpiry := t0.Add(40 * 24 * time.Hour)

	premium := recorderTestPlan("premium", "Premium", plankind.Finite, 1)
	first := recorderTestPlan("first", "First", plankind.Finite, 2)
	second := recorderTestPlan("second", "Second", plankind.Finite, 3)
	for _, plan := range []*entities.SubscriptionPlan{premium, first, second} {
		if err := dbConn.Exec(`INSERT INTO subscription.subscription_plans (id, name) VALUES (?, ?)`, plan.ID, plan.Name).Error; err != nil {
			t.Fatalf("seed plan %q: %v", plan.Slug, err)
		}
	}

	periodRepo := persistence.NewUserPlanPeriodRepository(dbConn)
	recorder := NewPlanPeriodRecorder(periodRepo)
	uow := shareddb.NewGormUnitOfWork(dbConn)

	activate := recorderTestEvent(userID, "031-recorder-fallback:activate", subscriptioneventtype.ACTIVATED, t0)
	if err := uow.Execute(ctx, func(txCtx context.Context) error {
		return recorder.Record(txCtx, activate, planRecorderSub(userID, premium, t0, &initialExpiry), premium)
	}); err != nil {
		t.Fatalf("record initial activation: %v", err)
	}

	candidates := []struct {
		plan  *entities.SubscriptionPlan
		sub   *entities.UserSubscription
		event *entities.UserSubscriptionEvent
	}{
		{
			plan:  first,
			sub:   planRecorderSub(userID, first, changedAt, &changedExpiry),
			event: recorderTestEvent(userID, "031-recorder-fallback:first", subscriptioneventtype.UPGRADED, changedAt),
		},
		{
			plan:  second,
			sub:   planRecorderSub(userID, second, changedAt, &changedExpiry),
			event: recorderTestEvent(userID, "031-recorder-fallback:second", subscriptioneventtype.UPGRADED, changedAt),
		},
	}

	ready := make(chan struct{}, len(candidates))
	release := make(chan struct{})
	recorders := make([]PlanPeriodRecorder, len(candidates))
	for i := range candidates {
		recorders[i] = NewPlanPeriodRecorder(&barrierPlanPeriodRepo{
			IUserPlanPeriodRepository: periodRepo,
			ready:                     ready,
			release:                   release,
		})
	}

	results := make([]error, len(candidates))
	var wg sync.WaitGroup
	for i := range candidates {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			results[index] = uow.Execute(context.Background(), func(txCtx context.Context) error {
				return recorders[index].Record(txCtx, candidates[index].event, candidates[index].sub, candidates[index].plan)
			})
		}(i)
	}

	for i := 0; i < len(candidates); i++ {
		select {
		case <-ready:
		case <-time.After(10 * time.Second):
			t.Fatal("timed out waiting for both transactions to reach the advisory lock")
		}
	}
	close(release)
	wg.Wait()

	for i, err := range results {
		if err != nil {
			t.Errorf("change %d failed: %v; the advisory lock must serialize the writers in the fallback variant", i, err)
		}
	}

	periods := planPeriodHistory(t, periodRepo, ctx, userID)
	if len(periods) != 2 {
		t.Fatalf("history has %d interval(s), want 2: the initial holding plus the last committed change; rows: %s",
			len(periods), planPeriodSummary(periods))
	}
	planRecorderAssertInterval(t, periods[0], premium.Slug, t0, &changedAt)
	planRecorderAssertContiguous(t, periods)
	assertNoPlanPeriodOverlap(t, dbConn, userID)

	current, err := periodRepo.GetCurrentByUserIDWithLock(ctx, userID)
	if err != nil {
		t.Fatalf("read current interval after the race: %v", err)
	}
	if current == nil || !current.ValidFrom.Equal(changedAt) {
		t.Fatalf("current interval after the race = %v, want a row starting at %s", current, changedAt)
	}
	if current.PlanCode != first.Slug && current.PlanCode != second.Slug {
		t.Fatalf("current plan after the race = %q, want one of the two candidates", current.PlanCode)
	}
	planRecorderAssertInterval(t, periods[1], current.PlanCode, changedAt, &changedExpiry)
}

// openPlanRecorderFallbackDB creates an isolated database with the 031 schema
// in the fallback (partial-unique) shape. It skips, never fails, when the
// environment cannot create a database - the same convention as the other
// Postgres-backed harnesses in this service.
func openPlanRecorderFallbackDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping fallback recorder integration test")
	}
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Scheme == "" || strings.TrimPrefix(parsed.Path, "/") == "" {
		t.Skip("TEST_DATABASE_URL must be a postgres URL DSN for the fallback harness")
	}

	base, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect base test database: %v", err)
	}

	name := fmt.Sprintf("plan031_recorder_fallback_%d", time.Now().UnixNano())
	if err := base.Exec("CREATE DATABASE " + quotePlanRecorderIdentifier(name)).Error; err != nil {
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

	scratch, err := gorm.Open(postgres.Open(scratchURL.String()), &gorm.Config{})
	if err != nil {
		_ = base.Exec("DROP DATABASE IF EXISTS " + quotePlanRecorderIdentifier(name) + " WITH (FORCE)").Error
		if sqlDB, dbErr := base.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		t.Fatalf("connect scratch database: %v", err)
	}

	t.Cleanup(func() {
		if sqlDB, dbErr := scratch.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		if err := base.Exec("DROP DATABASE IF EXISTS " + quotePlanRecorderIdentifier(name) + " WITH (FORCE)").Error; err != nil {
			t.Errorf("drop scratch database %s: %v", name, err)
		}
		if sqlDB, dbErr := base.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	})

	createPlanRecorderBaseSchema(t, scratch)
	if err := applyPlanRecorderMigration(t, scratch); err != nil {
		t.Fatalf("apply 031 migration to scratch database: %v", err)
	}

	// Force the fallback shape. The scratch database is owned by the test role,
	// which can install btree_gist, so the real migration created the exclusion
	// constraint; replacing it with the documented partial unique index is what
	// a restricted deployment looks like.
	if planRecorderHasExclusionConstraint(t, scratch) {
		if err := scratch.Exec("ALTER TABLE subscription.user_plan_periods DROP CONSTRAINT ex_user_plan_periods_no_overlap").Error; err != nil {
			t.Fatalf("drop exclusion constraint: %v", err)
		}
	}
	if err := scratch.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS ux_user_plan_periods_open
			ON subscription.user_plan_periods (user_id)
			WHERE valid_to IS NULL`).Error; err != nil {
		t.Fatalf("create fallback partial unique index: %v", err)
	}

	return scratch
}

// createPlanRecorderBaseSchema creates only the tables the migration reads and
// references. The recorder test seeds plans fine because the minimal catalogue
// carries the id and name the period snapshot needs.
func createPlanRecorderBaseSchema(t *testing.T, db *gorm.DB) {
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

func applyPlanRecorderMigration(t *testing.T, db *gorm.DB) error {
	t.Helper()
	content, err := accountmigrations.EmbedFS.ReadFile(planRecorderFallbackMigrationFile)
	if err != nil {
		t.Fatalf("read embedded migration %s: %v", planRecorderFallbackMigrationFile, err)
	}
	sql := string(content)
	upIndex := strings.Index(sql, "-- +goose Up")
	downIndex := strings.Index(sql, "-- +goose Down")
	if upIndex < 0 || downIndex < 0 || downIndex <= upIndex {
		t.Fatalf("%s does not carry the expected goose Up/Down markers", planRecorderFallbackMigrationFile)
	}

	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	for _, statement := range splitPlanRecorderStatements(sql[upIndex+len("-- +goose Up") : downIndex]) {
		if err := tx.Exec(statement).Error; err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

// splitPlanRecorderStatements divides the migration into executable statements.
// It understands dollar-quoted DO blocks and single-quoted literals, which is
// the minimum needed to keep a `;` inside either from ending a statement. It
// mirrors the splitter in the persistence migration harness because Go test
// helpers are not importable across packages.
func splitPlanRecorderStatements(sql string) []string {
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
			for end < len(sql) && isPlanRecorderIdentifierByte(sql[end]) {
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

func isPlanRecorderIdentifierByte(b byte) bool {
	return b == '_' ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9')
}

func planRecorderHasFallbackIndex(t *testing.T, db *gorm.DB) bool {
	t.Helper()
	var exists bool
	if err := db.Raw("SELECT to_regclass('subscription.ux_user_plan_periods_open') IS NOT NULL").Scan(&exists).Error; err != nil {
		t.Fatalf("read fallback index: %v", err)
	}
	return exists
}

func quotePlanRecorderIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
