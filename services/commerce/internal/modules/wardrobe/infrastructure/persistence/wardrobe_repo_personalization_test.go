package persistence

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/repositories"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 033 T022 - the wardrobe half of the personalization foundation: the new
// `last_recommended_at` column and the three repository reads/writes the
// rotation pipeline (D9) depends on.
//
// THE SCHEMA IS NOT BUILT HERE. AutoMigrate cannot express the column's shape
// in production nor prove the migration ran, so the harness REQUIRES a database
// built the documented way (docs/development/local-setup.md §12):
//
//	psql "$TEST_DATABASE_URL" -f database/init/00-init.sql   (then 01/02/03)
//	make migrate-up svc=account   DB_NAME=<the same database>
//	make migrate-up svc=commerce  DB_NAME=<the same database>
//
// The tests skip when TEST_DATABASE_URL is unset, so they compile and pass
// cleanly on a machine without Postgres.

const (
	// wardrobePersonalizationSearchPath mirrors configs/config.yaml
	// `database.search_path`: the repository uses unqualified names, so a DSN
	// without it fails with "relation does not exist" even though the schema is
	// correct.
	wardrobePersonalizationSearchPath = "public,identity,wardrobe,fashion,brand,subscription,aiexecutor,dashboard,shared,community"

	// wardrobePersonalizationLockKey is duplicated verbatim in
	// services/commerce/internal/modules/fashion/infrastructure/persistence/
	// personalization_schema_test.go. That test rolls the commerce migration
	// DOWN and UP again; this package reads `last_recommended_at` at the same
	// time when `go test ./...` runs the two package binaries in parallel. A
	// session advisory lock on a pinned connection serialises them, because the
	// contention is across processes and the database is all they share.
	wardrobePersonalizationLockKey = "closy.personalization-schema-fixtures"

	wardrobePersonalizationLockWait = 60 * time.Second

	// wardrobePersonalizationFixturePrefix namespaces every user this file
	// writes, so a run against a shared test database collides with nothing.
	wardrobePersonalizationFixturePrefix = "verify-033-wardrobe-"
)

// TestIntegrationWardrobeLastRecommendedAt covers the new write path: the
// recommendation pipeline stamps `last_recommended_at` best-effort and leaves
// `last_used_at` alone (D9, FR-054).
func TestIntegrationWardrobeLastRecommendedAt(t *testing.T) {
	dbConn := openWardrobePersonalizationTestDB(t)
	repo := NewWardrobeItemRepository(dbConn)
	ctx := context.Background()

	userID := createWardrobePersonalizationTestUser(t, dbConn)
	item := createWardrobePersonalizationTestItem(t, repo, userID, itemtype.UserItem, wardrobestatus.InWardrobe, nil, nil)
	otherItem := createWardrobePersonalizationTestItem(t, repo, userID, itemtype.UserItem, wardrobestatus.InWardrobe, nil, nil)

	if item.LastRecommendedAt != nil {
		t.Fatalf("a freshly inserted item already carries last_recommended_at: %v", item.LastRecommendedAt)
	}

	recommendedAt := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Microsecond)
	if err := repo.TouchLastRecommendedAt(ctx, []uuid.UUID{item.ID}, recommendedAt); err != nil {
		t.Fatalf("TouchLastRecommendedAt: %v", err)
	}

	stored := getWardrobePersonalizationTestItem(t, dbConn, item.ID)
	if stored.LastRecommendedAt == nil || !stored.LastRecommendedAt.UTC().Equal(recommendedAt) {
		t.Fatalf("last_recommended_at = %v, want %s", stored.LastRecommendedAt, recommendedAt)
	}
	if stored.LastUsedAt != nil {
		t.Fatalf("TouchLastRecommendedAt must not touch last_used_at, got %v", stored.LastUsedAt)
	}

	untouched := getWardrobePersonalizationTestItem(t, dbConn, otherItem.ID)
	if untouched.LastRecommendedAt != nil {
		t.Fatalf("an item outside the id list was stamped: %v", untouched.LastRecommendedAt)
	}

	// An empty id list is a no-op, not an error: the caller may have returned
	// only catalog suggestions.
	if err := repo.TouchLastRecommendedAt(ctx, nil, recommendedAt); err != nil {
		t.Fatalf("TouchLastRecommendedAt with no ids must be a no-op, got %v", err)
	}
}

// TestIntegrationWardrobeGetStaleActiveItems covers the rotation query
// contract: the user's own items, InWardrobe and UserItem only, ordered
// `last_used_at ASC NULLS FIRST, last_recommended_at ASC NULLS FIRST`.
func TestIntegrationWardrobeGetStaleActiveItems(t *testing.T) {
	dbConn := openWardrobePersonalizationTestDB(t)
	repo := NewWardrobeItemRepository(dbConn)
	ctx := context.Background()

	userID := createWardrobePersonalizationTestUser(t, dbConn)
	otherUserID := createWardrobePersonalizationTestUser(t, dbConn)

	now := time.Now().UTC()
	fortyDaysAgo := now.AddDate(0, 0, -40).Truncate(time.Microsecond)
	twoDaysAgo := now.AddDate(0, 0, -2).Truncate(time.Microsecond)
	yesterday := now.AddDate(0, 0, -1).Truncate(time.Microsecond)

	// Never used, never recommended: the very first slot.
	neverUsed := createWardrobePersonalizationTestItem(t, repo, userID, itemtype.UserItem, wardrobestatus.InWardrobe, nil, nil)
	// Old use, never recommended: second.
	oldUse := createWardrobePersonalizationTestItem(t, repo, userID, itemtype.UserItem, wardrobestatus.InWardrobe, &fortyDaysAgo, nil)
	// Old use, recommended yesterday: third (the second sort term decides).
	oldUseRecommended := createWardrobePersonalizationTestItem(t, repo, userID, itemtype.UserItem, wardrobestatus.InWardrobe, &fortyDaysAgo, nil)
	if err := repo.TouchLastRecommendedAt(ctx, []uuid.UUID{oldUseRecommended.ID}, yesterday); err != nil {
		t.Fatalf("stamp last_recommended_at on the fixture item: %v", err)
	}
	// Recently used: last.
	recentUse := createWardrobePersonalizationTestItem(t, repo, userID, itemtype.UserItem, wardrobestatus.InWardrobe, &twoDaysAgo, nil)

	// Everything below must be excluded from the stale pool.
	createWardrobePersonalizationTestItem(t, repo, userID, itemtype.SystemCatalogItem, wardrobestatus.InWardrobe, nil, nil)
	createWardrobePersonalizationTestItem(t, repo, userID, itemtype.UserItem, wardrobestatus.Selling, nil, nil)
	createWardrobePersonalizationTestItem(t, repo, userID, itemtype.UserItem, wardrobestatus.Sold, nil, nil)
	deleted := createWardrobePersonalizationTestItem(t, repo, userID, itemtype.UserItem, wardrobestatus.InWardrobe, nil, nil)
	if err := dbConn.Exec("UPDATE wardrobe.wardrobe_items SET is_deleted = TRUE WHERE id = ?", deleted.ID).Error; err != nil {
		t.Fatalf("soft-delete the fixture item: %v", err)
	}
	createWardrobePersonalizationTestItem(t, repo, otherUserID, itemtype.UserItem, wardrobestatus.InWardrobe, nil, nil)

	items, err := repo.GetStaleActiveItems(ctx, userID, 20)
	if err != nil {
		t.Fatalf("GetStaleActiveItems: %v", err)
	}

	want := []uuid.UUID{neverUsed.ID, oldUse.ID, oldUseRecommended.ID, recentUse.ID}
	got := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		got = append(got, item.ID)
	}
	if len(got) != len(want) {
		t.Fatalf("GetStaleActiveItems returned %d items (%v), want %d (%v): catalog, non-InWardrobe, "+
			"soft-deleted and other users' rows must be excluded", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("stale order = %v, want %v (last_used_at ASC NULLS FIRST, then last_recommended_at ASC NULLS FIRST)",
				got, want)
		}
	}

	limited, err := repo.GetStaleActiveItems(ctx, userID, 2)
	if err != nil {
		t.Fatalf("GetStaleActiveItems with a limit: %v", err)
	}
	if len(limited) != 2 || limited[0].ID != neverUsed.ID || limited[1].ID != oldUse.ID {
		t.Fatalf("limited stale page = %v, want the first two of %v", itemIDs(limited), want)
	}
}

// TestIntegrationWardrobeGetActiveSystemCatalogItemsByIDs covers the clone
// filter (D6): only rows that are a system catalog item, still InWardrobe and
// not deleted may be copied into a user's wardrobe.
func TestIntegrationWardrobeGetActiveSystemCatalogItemsByIDs(t *testing.T) {
	dbConn := openWardrobePersonalizationTestDB(t)
	repo := NewWardrobeItemRepository(dbConn)
	ctx := context.Background()

	userID := createWardrobePersonalizationTestUser(t, dbConn)
	// Catalog rows are created by an admin and still carry that admin's user_id
	// (BatchUploadWardrobeItems), so they are modelled with a real second
	// account rather than a nil id the FK would reject.
	catalogueOwnerID := createWardrobePersonalizationTestUser(t, dbConn)

	fashionIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	// The clone candidate set: the fifth id indexes the catalog row that is NOT
	// requested below, so it must stay out of the query entirely.
	candidateIDs := fashionIDs[:4]

	active := createWardrobePersonalizationTestItem(t, repo, catalogueOwnerID, itemtype.SystemCatalogItem, wardrobestatus.InWardrobe, nil, &fashionIDs[0])
	// The user's own copy of the same fashion item must never be cloneable.
	createWardrobePersonalizationTestItem(t, repo, userID, itemtype.UserItem, wardrobestatus.InWardrobe, nil, &fashionIDs[1])
	createWardrobePersonalizationTestItem(t, repo, catalogueOwnerID, itemtype.SystemCatalogItem, wardrobestatus.Selling, nil, &fashionIDs[2])
	deleted := createWardrobePersonalizationTestItem(t, repo, catalogueOwnerID, itemtype.SystemCatalogItem, wardrobestatus.InWardrobe, nil, &fashionIDs[3])
	if err := dbConn.Exec("UPDATE wardrobe.wardrobe_items SET is_deleted = TRUE WHERE id = ?", deleted.ID).Error; err != nil {
		t.Fatalf("soft-delete the catalog fixture: %v", err)
	}
	// A perfectly valid catalog row that the caller did not ask for.
	createWardrobePersonalizationTestItem(t, repo, catalogueOwnerID, itemtype.SystemCatalogItem, wardrobestatus.InWardrobe, nil, &fashionIDs[4])

	items, err := repo.GetActiveSystemCatalogItemsByIDs(ctx, candidateIDs)
	if err != nil {
		t.Fatalf("GetActiveSystemCatalogItemsByIDs: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("GetActiveSystemCatalogItemsByIDs returned %d rows (%s), want 1: only the active catalog row "+
			"is cloneable", len(items), describeWardrobePersonalizationItems(items))
	}
	if items[0].ID != active.ID {
		t.Fatalf("GetActiveSystemCatalogItemsByIDs returned %s, want the active catalog row %s", items[0].ID, active.ID)
	}

	empty, err := repo.GetActiveSystemCatalogItemsByIDs(ctx, nil)
	if err != nil {
		t.Fatalf("GetActiveSystemCatalogItemsByIDs with no ids: %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("GetActiveSystemCatalogItemsByIDs with no ids returned %d rows, want 0", len(empty))
	}
}

// openWardrobePersonalizationTestDB connects, proves the migration ran and
// takes the cross-package advisory lock.
func openWardrobePersonalizationTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping wardrobe personalization integration test")
	}

	dbConn, err := gorm.Open(postgres.Open(wardrobePersonalizationTestDSN(dsn)), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	// Registered first so it runs LAST (cleanups are LIFO), after the lock is
	// released.
	t.Cleanup(func() {
		if sqlDB, dbErr := dbConn.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	})

	var migrationApplied bool
	if err := dbConn.Raw(`
		SELECT to_regclass('identity.users') IS NOT NULL
		   AND to_regclass('wardrobe.wardrobe_items') IS NOT NULL
		   AND EXISTS (
		       SELECT 1 FROM information_schema.columns
		       WHERE table_schema = 'wardrobe' AND table_name = 'wardrobe_items'
		         AND column_name = 'last_recommended_at')`).Scan(&migrationApplied).Error; err != nil {
		t.Fatalf("probe the personalization migration: %v", err)
	}
	if !migrationApplied {
		t.Fatalf("TEST_DATABASE_URL is not a migration-built database: apply database/init/*.sql, then " +
			"`make migrate-up svc=account DB_NAME=<that database>` and " +
			"`make migrate-up svc=commerce DB_NAME=<that database>` (see docs/development/local-setup.md §12)")
	}

	lockWardrobePersonalizationSchema(t, dbConn)
	return dbConn
}

// wardrobePersonalizationTestDSN adds the search_path when the caller did not
// set one; a caller that set it explicitly keeps control.
func wardrobePersonalizationTestDSN(dsn string) string {
	if strings.Contains(dsn, "search_path=") {
		return dsn
	}
	if u, err := url.Parse(dsn); err == nil && u.Scheme != "" {
		q := u.Query()
		q.Set("search_path", wardrobePersonalizationSearchPath)
		u.RawQuery = q.Encode()
		return u.String()
	}
	return dsn + " search_path=" + wardrobePersonalizationSearchPath
}

// lockWardrobePersonalizationSchema pins a connection and takes the session
// advisory lock shared with the fashion persistence schema test.
func lockWardrobePersonalizationSchema(t *testing.T, dbConn *gorm.DB) {
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

	if _, err := conn.ExecContext(ctx, fmt.Sprintf("SET lock_timeout = '%dms'", wardrobePersonalizationLockWait.Milliseconds())); err != nil {
		t.Fatalf("set lock_timeout: %v", err)
	}
	if _, err := conn.ExecContext(ctx,
		"SELECT pg_advisory_lock(hashtextextended($1, 0))", wardrobePersonalizationLockKey); err != nil {
		t.Fatalf("take advisory lock %q: %v", wardrobePersonalizationLockKey, err)
	}
	t.Cleanup(func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := conn.ExecContext(unlockCtx,
			"SELECT pg_advisory_unlock(hashtextextended($1, 0))", wardrobePersonalizationLockKey); err != nil {
			// Closing the pinned connection releases the session lock anyway.
			t.Logf("release advisory lock %q: %v", wardrobePersonalizationLockKey, err)
		}
	})
}

// createWardrobePersonalizationTestUser inserts the identity row
// wardrobe_items references and registers its removal (the FK cascades).
func createWardrobePersonalizationTestUser(t *testing.T, dbConn *gorm.DB) uuid.UUID {
	t.Helper()

	userID := uuid.New()
	handle := fmt.Sprintf("%s%s", wardrobePersonalizationFixturePrefix, userID.String())
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

// createWardrobePersonalizationTestItem goes through the real repository so the
// fixture has exactly the shape production writes.
func createWardrobePersonalizationTestItem(
	t *testing.T,
	repo repositories.IWardrobeItemRepository,
	userID uuid.UUID,
	kind itemtype.ItemType,
	status wardrobestatus.WardrobeItemStatus,
	lastUsedAt *time.Time,
	fashionItemID *uuid.UUID,
) *entities.WardrobeItem {
	t.Helper()

	item := &entities.WardrobeItem{
		UserID:            userID,
		FashionItemID:     fashionItemID,
		Status:            status,
		ItemType:          kind,
		LastUsedAt:        lastUsedAt,
		LastRecommendedAt: nil,
	}
	if err := repo.Create(context.Background(), item); err != nil {
		t.Fatalf("insert the fixture wardrobe item: %v", err)
	}
	return item
}

func getWardrobePersonalizationTestItem(t *testing.T, dbConn *gorm.DB, id uuid.UUID) *entities.WardrobeItem {
	t.Helper()

	var item entities.WardrobeItem
	if err := dbConn.Where("id = ?", id).First(&item).Error; err != nil {
		t.Fatalf("read the fixture item %s: %v", id, err)
	}
	return &item
}

func itemIDs(items []*entities.WardrobeItem) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

// describeWardrobePersonalizationItems renders the filter fields of each row so
// a failure says WHICH row escaped the filter instead of only its id.
func describeWardrobePersonalizationItems(items []*entities.WardrobeItem) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		fashionItemID := "<nil>"
		if item.FashionItemID != nil {
			fashionItemID = item.FashionItemID.String()
		}
		parts = append(parts, fmt.Sprintf("{id:%s user:%s type:%d status:%d deleted:%v fashionItem:%s}",
			item.ID, item.UserID, item.ItemType, item.Status, item.IsDeleted, fashionItemID))
	}
	return strings.Join(parts, " ")
}
