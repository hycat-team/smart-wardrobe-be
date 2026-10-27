package persistence

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	commercemigrations "smart-wardrobe-be/services/commerce/database/migrations"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"
	shared_db "smart-wardrobe-be/shared/infrastructure/db"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 033 T022 - the foundamental DB contract of the personalization schema and of
// the two repositories on top of it (fashion.user_style_profiles,
// fashion.style_signals).
//
// THE SCHEMA IS NOT BUILT HERE. AutoMigrate would derive tables from the Go
// structs and therefore produce a shape that only RESEMBLES production: it
// cannot express the partial unique feedback index, the CHECK constraints or
// the cross-schema cascade. Executing the real migration is the only way these
// assertions mean anything, so the harness REQUIRES a database built the
// documented way (docs/development/local-setup.md §12):
//
//	psql "$TEST_DATABASE_URL" -f database/init/00-init.sql   (then 01/02/03)
//	make migrate-up svc=account   DB_NAME=<the same database>
//	make migrate-up svc=commerce  DB_NAME=<the same database>
//
// The tests skip when TEST_DATABASE_URL is unset, so they compile and pass
// cleanly on a machine without Postgres.

const (
	// personalizationSearchPath mirrors configs/config.yaml
	// `database.search_path`. The migration creates module schemas while the
	// repositories use unqualified names, so a DSN without it fails with
	// "relation does not exist" even though the schema is correct.
	personalizationSearchPath = "public,identity,wardrobe,fashion,brand,subscription,aiexecutor,dashboard,shared,community"

	// commerceMigrationTable is the per-service Goose table of Commerce
	// (Makefile GOOSE_TABLE for svc=commerce), NOT the Account/legacy
	// `goose_db_version`.
	commerceMigrationTable = "goose_db_version_commerce"

	// personalizationMigrationVersion is `<ts>_add_personalization.sql`.
	personalizationMigrationVersion = int64(20261027092936)

	// personalizationMigrationFile is matched against the Goose source path so
	// the assertion does not restate the timestamp twice.
	personalizationMigrationFile = "_add_personalization.sql"

	// personalizationFixturePrefix namespaces every row this file writes, so a
	// run against a shared test database can collide with nothing.
	personalizationFixturePrefix = "verify-033-pers-"

	// personalizationLockKey serialises the two persistence packages that share
	// one TEST_DATABASE_URL. The schema test below rolls the migration DOWN and
	// UP again; a wardrobe test reading `last_recommended_at` at that moment
	// would fail on a dropped column. The key string is duplicated verbatim in
	// wardrobe_repo_personalization_test.go on purpose: the two files are in
	// different packages and a shared test package for one lock is more surface
	// than the lock itself. It is a session advisory lock on a pinned
	// connection because the contention is across `go test` package binaries,
	// which share nothing but the database.
	personalizationLockKey = "closy.personalization-schema-fixtures"

	personalizationLockWait = 60 * time.Second
)

// openPersonalizationTestDB connects, applies the Commerce migrations and takes
// the cross-package advisory lock.
func openPersonalizationTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping personalization schema integration test")
	}

	dbConn, err := gorm.Open(postgres.Open(personalizationTestDSN(dsn)), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	// Register the pool cleanup first so it runs LAST (cleanups are LIFO).
	t.Cleanup(func() {
		if sqlDB, dbErr := dbConn.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	})

	if !personalizationBaseSchemaReady(t, dbConn) {
		t.Fatalf("TEST_DATABASE_URL is not a migration-built database: apply database/init/*.sql, then " +
			"`make migrate-up svc=account DB_NAME=<that database>` and " +
			"`make migrate-up svc=commerce DB_NAME=<that database>` (see docs/development/local-setup.md §12)")
	}

	// The migrations, not AutoMigrate (026 D-01). Idempotent: a no-op when the
	// database is already up to date.
	if err := shared_db.RunMigrations(context.Background(), dbConn, commercemigrations.EmbedFS,
		commerceMigrationTable, true, zap.NewNop()); err != nil {
		t.Fatalf("run commerce migrations: %v", err)
	}

	lockPersonalizationSchema(t, dbConn)
	return dbConn
}

// personalizationTestDSN adds the search_path when the caller did not set one;
// a caller that set it explicitly keeps control.
func personalizationTestDSN(dsn string) string {
	if strings.Contains(dsn, "search_path=") {
		return dsn
	}
	if u, err := url.Parse(dsn); err == nil && u.Scheme != "" {
		q := u.Query()
		q.Set("search_path", personalizationSearchPath)
		u.RawQuery = q.Encode()
		return u.String()
	}
	return dsn + " search_path=" + personalizationSearchPath
}

// personalizationBaseSchemaReady probes the pieces this file depends on: the
// Account migration chain (identity.users, the module schemas) and the shared
// outbox table. The personalization objects themselves are applied afterwards.
func personalizationBaseSchemaReady(t *testing.T, dbConn *gorm.DB) bool {
	t.Helper()

	var ready bool
	if err := dbConn.Raw(`
		SELECT to_regclass('identity.users') IS NOT NULL
		   AND to_regclass('wardrobe.wardrobe_items') IS NOT NULL
		   AND to_regclass('fashion.fashion_items') IS NOT NULL`).Scan(&ready).Error; err != nil {
		t.Fatalf("probe the base schema: %v", err)
	}
	return ready
}

// lockPersonalizationSchema pins a connection and takes a session advisory lock
// for the lifetime of the test.
func lockPersonalizationSchema(t *testing.T, dbConn *gorm.DB) {
	t.Helper()

	sqlDB, err := dbConn.DB()
	if err != nil {
		t.Fatalf("get sql.DB: %v", err)
	}
	ctx := context.Background()
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatalf("pin a connection for the advisory lock: %v", err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Logf("close the personalization lock connection: %v", err)
		}
	})

	if _, err := conn.ExecContext(ctx, fmt.Sprintf("SET lock_timeout = '%dms'", personalizationLockWait.Milliseconds())); err != nil {
		t.Fatalf("set lock_timeout: %v", err)
	}
	if _, err := conn.ExecContext(ctx,
		"SELECT pg_advisory_lock(hashtextextended($1, 0))", personalizationLockKey); err != nil {
		t.Fatalf("take advisory lock %q: %v", personalizationLockKey, err)
	}
	t.Cleanup(func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := conn.ExecContext(unlockCtx,
			"SELECT pg_advisory_unlock(hashtextextended($1, 0))", personalizationLockKey); err != nil {
			// Closing the pinned connection releases the session lock anyway.
			t.Logf("release advisory lock %q: %v", personalizationLockKey, err)
		}
	})
}

// newCommerceMigrationProvider builds a Goose provider over the embedded
// Commerce tree so a single migration can be rolled back and re-applied.
func newCommerceMigrationProvider(t *testing.T, dbConn *gorm.DB) *goose.Provider {
	t.Helper()

	sqlDB, err := dbConn.DB()
	if err != nil {
		t.Fatalf("get sql.DB: %v", err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, commercemigrations.EmbedFS,
		goose.WithTableName(commerceMigrationTable),
		goose.WithAllowOutofOrder(true))
	if err != nil {
		t.Fatalf("build the goose provider: %v", err)
	}
	return provider
}

// TestIntegrationPersonalizationMigrationUpDownUp is the migration half of
// SC-006: the feature's DDL must be reversible without touching anything it did
// not create.
func TestIntegrationPersonalizationMigrationUpDownUp(t *testing.T) {
	dbConn := openPersonalizationTestDB(t)
	provider := newCommerceMigrationProvider(t, dbConn)
	ctx := context.Background()

	// Whatever happens below, put the schema back before the package finishes.
	t.Cleanup(func() {
		if _, err := provider.Up(context.Background()); err != nil {
			t.Logf("re-apply commerce migrations after the down/up cycle: %v", err)
		}
	})

	assertPersonalizationObjects(t, dbConn, true)

	// Down rolls back the newest applied Commerce migration. Pinning that it is
	// OUR migration keeps a later feature from silently turning this test into
	// a check of a different migration.
	statuses, err := provider.Status(ctx)
	if err != nil {
		t.Fatalf("read migration status: %v", err)
	}
	var newest *goose.MigrationStatus
	for _, status := range statuses {
		if status.State != goose.StateApplied {
			continue
		}
		if newest == nil || status.Source.Version > newest.Source.Version {
			newest = status
		}
	}
	if newest == nil {
		t.Fatalf("no applied Commerce migration found in %s", commerceMigrationTable)
	}
	if !strings.HasSuffix(newest.Source.Path, personalizationMigrationFile) {
		t.Skipf("the newest applied Commerce migration is %s (%d), not %s: re-point this test before "+
			"running the down/up cycle", newest.Source.Path, newest.Source.Version, personalizationMigrationFile)
	}

	result, err := provider.Down(ctx)
	if err != nil {
		t.Fatalf("roll the personalization migration back: %v", err)
	}
	if result == nil || result.Source == nil {
		t.Fatal("Down returned no source; cannot tell what was rolled back")
	}
	if result.Source.Version != personalizationMigrationVersion {
		t.Fatalf("Down rolled back version %d, want %d (%s)",
			result.Source.Version, personalizationMigrationVersion, personalizationMigrationFile)
	}
	assertPersonalizationObjects(t, dbConn, false)

	// Up again: the Down block must have dropped only its own objects, or the
	// re-apply fails on something that already exists.
	if _, err := provider.Up(ctx); err != nil {
		t.Fatalf("re-apply the personalization migration: %v", err)
	}
	assertPersonalizationObjects(t, dbConn, true)

	version, err := provider.GetDBVersion(ctx)
	if err != nil {
		t.Fatalf("read the commerce migration version: %v", err)
	}
	if version != personalizationMigrationVersion {
		t.Fatalf("commerce migration version = %d after up/down/up, want %d", version, personalizationMigrationVersion)
	}
}

// assertPersonalizationObjects checks everything the migration is responsible
// for, in both directions.
func assertPersonalizationObjects(t *testing.T, dbConn *gorm.DB, present bool) {
	t.Helper()

	expected := map[string]bool{
		"fashion.user_style_profiles": present,
		"fashion.style_signals":       present,
	}
	for relation, want := range expected {
		var got bool
		if err := dbConn.Raw("SELECT to_regclass(?) IS NOT NULL", relation).Scan(&got).Error; err != nil {
			t.Fatalf("probe %s: %v", relation, err)
		}
		if got != want {
			t.Fatalf("%s exists = %v, want %v", relation, got, want)
		}
	}

	var columnPresent bool
	if err := dbConn.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'wardrobe' AND table_name = 'wardrobe_items'
			  AND column_name = 'last_recommended_at')`).Scan(&columnPresent).Error; err != nil {
		t.Fatalf("probe wardrobe_items.last_recommended_at: %v", err)
	}
	if columnPresent != present {
		t.Fatalf("wardrobe.wardrobe_items.last_recommended_at exists = %v, want %v", columnPresent, present)
	}

	if !present {
		return
	}

	// Down drops the new column and nothing else: `last_used_at` predates this
	// feature and must survive the rollback.
	var recencyColumns int64
	if err := dbConn.Raw(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = 'wardrobe' AND table_name = 'wardrobe_items'
		  AND column_name IN ('last_used_at', 'last_recommended_at')`).Scan(&recencyColumns).Error; err != nil {
		t.Fatalf("count wardrobe recency columns: %v", err)
	}
	if recencyColumns != 2 {
		t.Fatalf("wardrobe_items has %d of the two recency columns, want 2 (Down must not drop last_used_at)",
			recencyColumns)
	}
}

// TestIntegrationPersonalizationSchemaShape pins the parts of the DDL the
// application depends on: the columns, the CHECK constraints, the partial
// unique feedback index and both cascades to identity.users.
func TestIntegrationPersonalizationSchemaShape(t *testing.T) {
	dbConn := openPersonalizationTestDB(t)

	profileColumns := []string{
		"user_id", "styles", "excluded_styles", "enabled", "style_embedding", "embedding_hash",
		"embedding_model", "learned_styles", "learned_embedding", "learned_at", "created_at", "updated_at",
	}
	var profileColumnCount int64
	if err := dbConn.Raw(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = 'fashion' AND table_name = 'user_style_profiles'
		  AND column_name IN ?`, profileColumns).Scan(&profileColumnCount).Error; err != nil {
		t.Fatalf("probe user_style_profiles columns: %v", err)
	}
	if profileColumnCount != int64(len(profileColumns)) {
		t.Fatalf("fashion.user_style_profiles has %d of the %d expected columns",
			profileColumnCount, len(profileColumns))
	}

	signalColumns := []string{
		"id", "user_id", "source", "outfit_id", "recommendation_id", "fashion_item_ids",
		"style_snapshot", "created_at",
	}
	var signalColumnCount int64
	if err := dbConn.Raw(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = 'fashion' AND table_name = 'style_signals'
		  AND column_name IN ?`, signalColumns).Scan(&signalColumnCount).Error; err != nil {
		t.Fatalf("probe style_signals columns: %v", err)
	}
	if signalColumnCount != int64(len(signalColumns)) {
		t.Fatalf("fashion.style_signals has %d of the %d expected columns",
			signalColumnCount, len(signalColumns))
	}

	// data-model.md §2: five checks on the profile, §3: two on the signal.
	var profileChecks, signalChecks int64
	if err := dbConn.Raw(`
		SELECT COUNT(*) FROM pg_constraint
		WHERE conrelid = 'fashion.user_style_profiles'::regclass AND contype = 'c'`).Scan(&profileChecks).Error; err != nil {
		t.Fatalf("count user_style_profiles checks: %v", err)
	}
	if profileChecks != 5 {
		t.Fatalf("fashion.user_style_profiles carries %d CHECK constraints, want 5", profileChecks)
	}
	if err := dbConn.Raw(`
		SELECT COUNT(*) FROM pg_constraint
		WHERE conrelid = 'fashion.style_signals'::regclass AND contype = 'c'`).Scan(&signalChecks).Error; err != nil {
		t.Fatalf("count style_signals checks: %v", err)
	}
	if signalChecks != 2 {
		t.Fatalf("fashion.style_signals carries %d CHECK constraints, want 2", signalChecks)
	}

	// Both FKs must cascade: deleting an account deletes its taste data.
	for _, table := range []string{"fashion.user_style_profiles", "fashion.style_signals"} {
		var cascadeFKs int64
		if err := dbConn.Raw(`
			SELECT COUNT(*) FROM pg_constraint
			WHERE conrelid = ?::regclass AND contype = 'f' AND confdeltype = 'c'`, table).Scan(&cascadeFKs).Error; err != nil {
			t.Fatalf("probe cascade FK on %s: %v", table, err)
		}
		if cascadeFKs < 1 {
			t.Fatalf("%s has no ON DELETE CASCADE foreign key to identity.users", table)
		}
	}

	// The partial unique index is the database-level duplicate-feedback guard.
	var feedbackIndexDef, userIndexDef string
	if err := dbConn.Raw(`
		SELECT indexdef FROM pg_indexes
		WHERE schemaname = 'fashion' AND indexname = 'ux_style_signals_feedback'`).Scan(&feedbackIndexDef).Error; err != nil {
		t.Fatalf("probe ux_style_signals_feedback: %v", err)
	}
	if !strings.Contains(feedbackIndexDef, "UNIQUE") || !strings.Contains(feedbackIndexDef, "WHERE") ||
		!strings.Contains(feedbackIndexDef, "recommendation_id IS NOT NULL") {
		t.Fatalf("ux_style_signals_feedback is not the expected partial unique index: %q", feedbackIndexDef)
	}
	if err := dbConn.Raw(`
		SELECT indexdef FROM pg_indexes
		WHERE schemaname = 'fashion' AND indexname = 'idx_style_signals_user_created'`).Scan(&userIndexDef).Error; err != nil {
		t.Fatalf("probe idx_style_signals_user_created: %v", err)
	}
	if !strings.Contains(userIndexDef, "user_id") || !strings.Contains(userIndexDef, "created_at") {
		t.Fatalf("idx_style_signals_user_created does not index (user_id, created_at): %q", userIndexDef)
	}
}

// TestIntegrationStyleSignalFeedbackIsUniquePerRecommendation covers the
// partial unique index through the repository: the second like for the same
// recommendation fails, while outfit-save signals (NULL recommendation_id) do
// not collide with each other.
func TestIntegrationStyleSignalFeedbackIsUniquePerRecommendation(t *testing.T) {
	dbConn := openPersonalizationTestDB(t)
	repo := NewStyleSignalRepository(dbConn)
	ctx := context.Background()

	userID := createPersonalizationTestUser(t, dbConn)
	recommendationID := uuid.New()
	itemIDs := []uuid.UUID{uuid.New(), uuid.New()}

	like := &entities.StyleSignal{
		ID:               uuid.New(),
		UserID:           userID,
		Source:           entities.StyleSignalSourceFeedbackLike,
		RecommendationID: &recommendationID,
		FashionItemIDs:   itemIDs,
	}
	inserted, err := repo.InsertIfAbsent(ctx, like)
	if err != nil {
		t.Fatalf("insert the first feedback signal: %v", err)
	}
	if !inserted {
		t.Fatal("the first feedback signal was not written")
	}

	duplicate := &entities.StyleSignal{
		ID:               uuid.New(),
		UserID:           userID,
		Source:           entities.StyleSignalSourceFeedbackLike,
		RecommendationID: &recommendationID,
		FashionItemIDs:   itemIDs,
	}
	if _, err := repo.InsertIfAbsent(ctx, duplicate); err == nil {
		t.Fatal("a second like for the same recommendation was accepted; ux_style_signals_feedback is missing")
	}

	// A dislike on the same recommendation is a DIFFERENT source and must pass.
	dislike := &entities.StyleSignal{
		ID:               uuid.New(),
		UserID:           userID,
		Source:           entities.StyleSignalSourceFeedbackDislike,
		RecommendationID: &recommendationID,
		FashionItemIDs:   itemIDs,
	}
	if inserted, err := repo.InsertIfAbsent(ctx, dislike); err != nil || !inserted {
		t.Fatalf("a dislike for the same recommendation was rejected: inserted=%v err=%v", inserted, err)
	}

	// Two outfit-save signals carry no recommendation id; the partial index
	// does not apply and both are kept.
	for i := 0; i < 2; i++ {
		saved := &entities.StyleSignal{
			ID:             uuid.New(),
			UserID:         userID,
			Source:         entities.StyleSignalSourceOutfitSaved,
			FashionItemIDs: itemIDs,
		}
		if inserted, err := repo.InsertIfAbsent(ctx, saved); err != nil || !inserted {
			t.Fatalf("outfit-save signal %d was rejected: inserted=%v err=%v", i, inserted, err)
		}
	}

	count, err := repo.CountByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("count signals: %v", err)
	}
	if count != 4 {
		t.Fatalf("signal count = %d, want 4 (1 like + 1 dislike + 2 saves)", count)
	}
}

// TestIntegrationStyleSignalChecksRejectInvalidRows covers the two CHECK
// constraints the learning pipeline relies on.
func TestIntegrationStyleSignalChecksRejectInvalidRows(t *testing.T) {
	dbConn := openPersonalizationTestDB(t)
	repo := NewStyleSignalRepository(dbConn)
	ctx := context.Background()

	userID := createPersonalizationTestUser(t, dbConn)
	itemIDs := []uuid.UUID{uuid.New()}

	invalidSource := &entities.StyleSignal{
		ID:             uuid.New(),
		UserID:         userID,
		Source:         99,
		FashionItemIDs: itemIDs,
	}
	if _, err := repo.InsertIfAbsent(ctx, invalidSource); err == nil {
		t.Fatal("source 99 was accepted; chk_style_signals_source is missing")
	}

	emptyItems := &entities.StyleSignal{
		ID:             uuid.New(),
		UserID:         userID,
		Source:         entities.StyleSignalSourceOutfitSaved,
		FashionItemIDs: []uuid.UUID{},
	}
	if _, err := repo.InsertIfAbsent(ctx, emptyItems); err == nil {
		t.Fatal("an empty fashion_item_ids was accepted; chk_style_signals_items is missing")
	}
}

// TestIntegrationStyleSignalInsertIsIdempotentByID covers D7: the outbox is
// at-least-once, so the same signal id delivered twice must be a no-op.
func TestIntegrationStyleSignalInsertIsIdempotentByID(t *testing.T) {
	dbConn := openPersonalizationTestDB(t)
	repo := NewStyleSignalRepository(dbConn)
	ctx := context.Background()

	userID := createPersonalizationTestUser(t, dbConn)
	signalID := uuid.New()
	itemIDs := []uuid.UUID{uuid.New()}

	first := &entities.StyleSignal{
		ID:             signalID,
		UserID:         userID,
		Source:         entities.StyleSignalSourceOutfitSaved,
		FashionItemIDs: itemIDs,
	}
	if inserted, err := repo.InsertIfAbsent(ctx, first); err != nil || !inserted {
		t.Fatalf("first delivery: inserted=%v err=%v", inserted, err)
	}

	redelivery := &entities.StyleSignal{
		ID:             signalID,
		UserID:         userID,
		Source:         entities.StyleSignalSourceOutfitSaved,
		FashionItemIDs: itemIDs,
	}
	inserted, err := repo.InsertIfAbsent(ctx, redelivery)
	if err != nil {
		t.Fatalf("redelivery must not fail: %v", err)
	}
	if inserted {
		t.Fatal("the redelivered signal was inserted again; ON CONFLICT (id) DO NOTHING is missing")
	}

	count, err := repo.CountByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("count signals: %v", err)
	}
	if count != 1 {
		t.Fatalf("signal count after a redelivery = %d, want 1", count)
	}
}

// TestIntegrationPersonalizationCascadesOnUserDelete covers the FK cascade:
// deleting an account must remove both its profile and its signals.
func TestIntegrationPersonalizationCascadesOnUserDelete(t *testing.T) {
	dbConn := openPersonalizationTestDB(t)
	profileRepo := NewUserStyleProfileRepository(dbConn)
	signalRepo := NewStyleSignalRepository(dbConn)
	ctx := context.Background()

	userID := createPersonalizationTestUser(t, dbConn)
	if err := profileRepo.Upsert(ctx, &entities.UserStyleProfile{
		UserID:         userID,
		Styles:         entities.StyleList{"minimal"},
		ExcludedStyles: entities.StyleList{"sporty"},
		Enabled:        true,
		LearnedStyles:  entities.StyleWeightMap{},
	}); err != nil {
		t.Fatalf("create the style profile: %v", err)
	}
	if _, err := signalRepo.InsertIfAbsent(ctx, &entities.StyleSignal{
		ID:             uuid.New(),
		UserID:         userID,
		Source:         entities.StyleSignalSourceOutfitSaved,
		FashionItemIDs: []uuid.UUID{uuid.New()},
	}); err != nil {
		t.Fatalf("create the style signal: %v", err)
	}

	if err := dbConn.Exec("DELETE FROM identity.users WHERE id = ?", userID).Error; err != nil {
		t.Fatalf("delete the user: %v", err)
	}

	profile, err := profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("read the profile after the user was deleted: %v", err)
	}
	if profile != nil {
		t.Fatal("the style profile survived the deletion of its user; the FK is not ON DELETE CASCADE")
	}
	count, err := signalRepo.CountByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("count signals after the user was deleted: %v", err)
	}
	if count != 0 {
		t.Fatalf("%d style signal(s) survived the deletion of their user; the FK is not ON DELETE CASCADE", count)
	}
}

// createPersonalizationTestUser inserts the identity row both new tables
// reference and registers its removal. Only the NOT NULL columns are supplied:
// password_hash became nullable in 021 and the rest carry defaults.
func createPersonalizationTestUser(t *testing.T, dbConn *gorm.DB) uuid.UUID {
	t.Helper()

	userID := uuid.New()
	handle := fmt.Sprintf("%s%s", personalizationFixturePrefix, userID.String())
	if err := dbConn.Exec(
		`INSERT INTO identity.users (id, username, email, role_slug) VALUES (?, ?, ?, ?)`,
		userID, handle, handle+"@example.test", "user").Error; err != nil {
		t.Fatalf("insert the fixture user: %v", err)
	}
	t.Cleanup(func() {
		if err := dbConn.Exec("DELETE FROM identity.users WHERE id = ?", userID).Error; err != nil {
			t.Logf("delete the fixture user %s: %v", userID, err)
		}
	})
	return userID
}
