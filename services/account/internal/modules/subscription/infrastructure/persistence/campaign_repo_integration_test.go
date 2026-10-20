package persistence

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	accountmigrations "smart-wardrobe-be/services/account/database/migrations"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	shareddb "smart-wardrobe-be/shared/infrastructure/db"
	campaigndb "smart-wardrobe-be/shared/testing/campaigndb"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// 026 T022 — the campaign registry against a REAL Postgres.
//
// WHY THESE EXIST AND WHY NO UNIT TEST CAN REPLACE THEM. Every method on
// `ICampaignRepository` is a SQL statement whose behaviour lives in the database:
// the conditional `UPDATE ... WHERE version = ?` that FR-055 depends on, the
// `EXISTS` that FR-020 depends on, and above all `SELECT ... FOR SHARE`, which is
// the entire mechanism behind FR-045. A fake cannot show that a concurrent close
// BLOCKS, and a fake certainly cannot show that the immutability trigger refuses
// a hand-written UPDATE.
//
// THE HARNESS IS THE REPO'S, NOT A NEW ONE. `TEST_DATABASE_URL` plus a `t.Skip` is
// how every Postgres-backed test in this service already behaves
// (`application/usecase/subscription/renewal_integration_test.go`,
// `database/migrations/migration_integration_test.go`), and the schema comes from
// the EMBEDDED GOOSE MIGRATIONS rather than from `AutoMigrate`. That second point
// is the one that matters: `AutoMigrate` builds tables from the Go structs, so it
// would not create the `CHECK` constraints or the trigger, and a test running
// against it would be testing a schema production does not have.
//
// The tests need a database that has already been provisioned with the base
// `identity` schema, because the claim ledger's `user_id` carries a foreign key to
// `identity.users` and no migration in this repository creates that table. Point
// `TEST_DATABASE_URL` at a migrated database, the same one quickstart uses.

const (
	// integrationCampaignPrefix keeps every row this file writes in a namespace of
	// its own, so a run against a shared test database cannot collide with a real
	// campaign or delete one.
	integrationCampaignPrefix = "verify-026-repo-"
)

// openRegistryTestDB connects and migrates, skipping when no database is
// provisioned. Shared with the FR-091 schema guard in this package.
func openRegistryTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping campaign registry integration test")
	}
	dbConn, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	// `gorm.Open` creates a new pool per call. Without this cleanup, a re-run
	// (`-count=3`) and the other integration package running in parallel pile
	// pools up until Postgres answers `too many clients`. Registered before the
	// fixture cleanups and the advisory lock so LIFO closes the pool last.
	t.Cleanup(func() {
		if sqlDB, dbErr := dbConn.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	})
	if err := shareddb.RunMigrations(context.Background(), dbConn, accountmigrations.EmbedFS, "goose_db_version", true, zap.NewNop()); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	campaigndb.LockAndPurge(t, dbConn)
	t.Cleanup(func() {
		dbConn.Exec("DELETE FROM subscription.campaign_claims WHERE campaign_code LIKE ?", integrationCampaignPrefix+"%")
		dbConn.Exec("DELETE FROM subscription.campaigns WHERE code LIKE ?", integrationCampaignPrefix+"%")
	})
	return dbConn
}

// integrationCampaign seeds one registry row with a code in this file's namespace.
func integrationCampaign(t *testing.T, repo repositories.ICampaignRepository, suffix string) *entities.Campaign {
	t.Helper()
	campaign := &entities.Campaign{
		Code:     integrationCampaignPrefix + suffix,
		PlanSlug: "premium-monthly",
		Quota:    50,
		Reserve:  5,
		StartsAt: time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC),
	}
	if err := repo.Create(context.Background(), campaign); err != nil {
		t.Fatalf("create campaign %s: %v", campaign.Code, err)
	}
	return campaign
}

// TestIntegrationGetByCodeSeparatesAbsentFromFault pins the FR-054 split: an
// unknown code is `nil, nil` (the caller answers 404), while a broken database is
// an error. Collapsing the two would turn an outage into "this campaign does not
// exist".
func TestIntegrationGetByCodeSeparatesAbsentFromFault(t *testing.T) {
	dbConn := openRegistryTestDB(t)
	repo := NewCampaignRepository(dbConn)
	seeded := integrationCampaign(t, repo, "get-by-code")

	found, err := repo.GetByCode(context.Background(), seeded.Code)
	if err != nil {
		t.Fatalf("GetByCode: %v", err)
	}
	if found == nil || found.Code != seeded.Code {
		t.Fatalf("GetByCode(%s) = %+v, want the seeded row", seeded.Code, found)
	}
	if found.Quota != 50 || found.Reserve != 5 || found.EndsAt != nil {
		t.Errorf("round trip lost values: quota=%d reserve=%d endsAt=%v", found.Quota, found.Reserve, found.EndsAt)
	}
	if found.Version != 1 {
		t.Errorf("version = %d, want the 1 the column default installed", found.Version)
	}

	absent, err := repo.GetByCode(context.Background(), integrationCampaignPrefix+"does-not-exist")
	if err != nil {
		t.Fatalf("an unknown code must not be an error: %v", err)
	}
	if absent != nil {
		t.Errorf("an unknown code must read as nil, got %+v", absent)
	}
}

// TestIntegrationGetByCodeForShareBlocksAConcurrentClose is FR-045's whole point,
// and the one thing no fake can express.
//
// A grant re-reads the campaign row under a SHARE lock while it writes the claim.
// A close arriving in that window must WAIT: otherwise the claim commits after the
// close and a campaign that an operator has already closed grants one more plan.
func TestIntegrationGetByCodeForShareBlocksAConcurrentClose(t *testing.T) {
	dbConn := openRegistryTestDB(t)
	repo := NewCampaignRepository(dbConn)
	seeded := integrationCampaign(t, repo, "for-share")

	// The real transaction, entered the way the grant path enters it: `uow.Execute`
	// injects the handle and `GetByCodeForShare` picks it up through `GetDB`. Using
	// the unit of work rather than a hand-rolled transaction is what makes this a
	// test of the production path.
	uow := shareddb.NewGormUnitOfWork(dbConn)
	err := uow.Execute(context.Background(), func(txCtx context.Context) error {
		locked, err := repo.GetByCodeForShare(txCtx, seeded.Code)
		if err != nil {
			return err
		}
		if locked == nil {
			return errRegistryLockReadEmpty
		}

		// A SECOND connection: the pool hands it out because the transaction above is
		// still open. If `FOR SHARE` were really taken, this UPDATE blocks.
		blockedCtx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer cancel()
		result := dbConn.WithContext(blockedCtx).
			Model(&entities.Campaign{}).
			Where("code = ?", seeded.Code).
			Update("updated_at", gorm.Expr("now()"))
		if result.Error == nil {
			t.Error("FR-045: a concurrent close must WAIT for the share lock, but the UPDATE went through — " +
				"a campaign could be closed and still grant one more plan")
		}
		return nil
	})
	if errors.Is(err, errRegistryLockReadEmpty) {
		t.Fatal("the locked read returned no row")
	}
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

// errRegistryLockReadEmpty is a sentinel for the one failure inside the transaction
// that is an assertion rather than a database error, so the assertion message is
// not swallowed into a wrapped error.
var errRegistryLockReadEmpty = errors.New("campaign registry integration: locked read returned no row")

// TestIntegrationUpdateIsConditionalOnTheVersion is FR-055.
//
// Two administrators editing the same campaign: the loser must change NOTHING and
// learn that it changed nothing, because that is what turns into 412 rather than a
// silent overwrite and an audit trail that does not match the final state.
func TestIntegrationUpdateIsConditionalOnTheVersion(t *testing.T) {
	dbConn := openRegistryTestDB(t)
	repo := NewCampaignRepository(dbConn)
	seeded := integrationCampaign(t, repo, "if-match")

	stale := *seeded
	stale.Quota = 11
	updated, err := repo.Update(context.Background(), &stale, seeded.Version+5)
	if err != nil {
		t.Fatalf("Update with a stale version must not be an error: %v", err)
	}
	if updated {
		t.Fatal("a stale version must change 0 rows")
	}

	fresh := *seeded
	fresh.Quota = 60
	updated, err = repo.Update(context.Background(), &fresh, seeded.Version)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if !updated {
		t.Fatal("the matching version must change exactly 1 row")
	}

	after, err := repo.GetByCode(context.Background(), seeded.Code)
	if err != nil || after == nil {
		t.Fatalf("re-read: %v", err)
	}
	if after.Quota != 60 {
		t.Errorf("quota = %d, want 60", after.Quota)
	}
	if after.Version != seeded.Version+1 {
		t.Errorf("version = %d, want it incremented to %d by the statement itself", after.Version, seeded.Version+1)
	}
}

// TestIntegrationUpdateRefusesTheBudgetOnceGrantsExist is FR-020/FR-021 through
// the database, which is the layer a hand-written UPDATE cannot argue with.
func TestIntegrationUpdateRefusesTheBudgetOnceGrantsExist(t *testing.T) {
	dbConn := openRegistryTestDB(t)
	repo := NewCampaignRepository(dbConn)
	seeded := integrationCampaign(t, repo, "budget-locked")

	// Before any grant the budget is editable (FR-022).
	before := *seeded
	before.Quota = 40
	updated, err := repo.Update(context.Background(), &before, seeded.Version)
	if err != nil {
		t.Fatalf("Update before any grant must be allowed: %v", err)
	}
	if !updated {
		t.Fatal("FR-022: with no claims the budget MUST be editable")
	}

	seedRegistryClaim(t, dbConn, seeded.Code)

	current, err := repo.GetByCode(context.Background(), seeded.Code)
	if err != nil || current == nil {
		t.Fatalf("re-read: %v", err)
	}
	after := *current
	after.Quota = current.Quota + 1
	if _, err := repo.Update(context.Background(), &after, current.Version); err == nil {
		t.Error("FR-021: the trigger MUST refuse a quota change once grants exist, and it did not")
	}

	unchanged, err := repo.GetByCode(context.Background(), seeded.Code)
	if err != nil || unchanged == nil {
		t.Fatalf("re-read after the refused update: %v", err)
	}
	if unchanged.Quota != current.Quota {
		t.Errorf("quota moved to %d after a refused update, want %d — the value must be untouched",
			unchanged.Quota, current.Quota)
	}
}

// TestIntegrationListOpenExcludesClosedCampaigns is FR-070/FR-031 and the half of
// the sweep rule that makes "close it early" mean anything.
func TestIntegrationListOpenExcludesClosedCampaigns(t *testing.T) {
	dbConn := openRegistryTestDB(t)
	repo := NewCampaignRepository(dbConn)
	open1 := integrationCampaign(t, repo, "open-one")
	open2 := integrationCampaign(t, repo, "open-two")
	closed := integrationCampaign(t, repo, "closed")

	if _, err := repo.Close(context.Background(), closed.Code, closed.Version); err != nil {
		t.Fatalf("Close: %v", err)
	}

	open, err := repo.ListOpen(context.Background())
	if err != nil {
		t.Fatalf("ListOpen: %v", err)
	}
	if containsRegistryCode(open, closed.Code) {
		t.Errorf("FR-031: %s was closed and must not appear in ListOpen", closed.Code)
	}
	for _, code := range []string{open1.Code, open2.Code} {
		if !containsRegistryCode(open, code) {
			t.Errorf("%s is open and must appear in ListOpen", code)
		}
	}

	// `ListAll` is the operator list and MUST still show the closed campaign —
	// FR-032 adds `status = closed` precisely so an operator can see it.
	all, err := repo.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if !containsRegistryCode(all, closed.Code) {
		t.Errorf("FR-032: the admin list must still show the closed campaign %s", closed.Code)
	}
}

// TestIntegrationListAllOrdersNewestFirstWithCodeTieBreak is FR-040/FR-041.
//
// Two campaigns may share a `starts_at` — an operator can create the second from
// any screen at any time — and the answer must not depend on which row the
// database returned first, or the same account could land in a different campaign
// on two consecutive reads.
func TestIntegrationListAllOrdersNewestFirstWithCodeTieBreak(t *testing.T) {
	dbConn := openRegistryTestDB(t)
	repo := NewCampaignRepository(dbConn)

	shared := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	earlier := shared.Add(-24 * time.Hour)
	for _, spec := range []struct {
		suffix string
		start  time.Time
	}{
		{"order-earlier", earlier},
		{"order-tie-b", shared},
		{"order-tie-a", shared},
	} {
		campaign := &entities.Campaign{
			Code:     integrationCampaignPrefix + spec.suffix,
			PlanSlug: "premium-monthly",
			Quota:    5,
			StartsAt: spec.start,
		}
		if err := repo.Create(context.Background(), campaign); err != nil {
			t.Fatalf("create %s: %v", campaign.Code, err)
		}
	}

	all, err := repo.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	var ordered []string
	for _, campaign := range all {
		if campaign != nil && len(campaign.Code) > len(integrationCampaignPrefix) &&
			campaign.Code[:len(integrationCampaignPrefix)] == integrationCampaignPrefix {
			ordered = append(ordered, campaign.Code)
		}
	}
	want := []string{
		integrationCampaignPrefix + "order-tie-b",
		integrationCampaignPrefix + "order-tie-a",
		integrationCampaignPrefix + "order-earlier",
	}
	if len(ordered) != len(want) {
		t.Fatalf("got %d seeded campaigns in order %v, want %d in %v", len(ordered), ordered, len(want), want)
	}
	for i := range want {
		if ordered[i] != want[i] {
			t.Fatalf("order = %v, want %v (newest starts_at first, then code DESC)", ordered, want)
		}
	}
}

// TestIntegrationExistsClaimsTracksTheLedger is FR-020's application-layer half:
// the read the 422 answer is built on.
func TestIntegrationExistsClaimsTracksTheLedger(t *testing.T) {
	dbConn := openRegistryTestDB(t)
	repo := NewCampaignRepository(dbConn)
	seeded := integrationCampaign(t, repo, "exists-claims")

	has, err := repo.ExistsClaims(context.Background(), seeded.Code)
	if err != nil {
		t.Fatalf("ExistsClaims: %v", err)
	}
	if has {
		t.Error("a campaign with no grants must report false, or every budget would look locked")
	}

	seedRegistryClaim(t, dbConn, seeded.Code)

	has, err = repo.ExistsClaims(context.Background(), seeded.Code)
	if err != nil {
		t.Fatalf("ExistsClaims: %v", err)
	}
	if !has {
		t.Error("a campaign with a grant must report true, or the budget would look editable")
	}
}

// TestIntegrationRegistryHasNoDeletePath is FR-005/FR-034 as a structural
// assertion rather than a review convention.
//
// It runs on the INTERFACE, not on the implementation, because the interface is
// what every caller holds: a `Delete` added to the interface would compile
// everywhere and orphan a ledger that references campaigns by text.
func TestIntegrationRegistryHasNoDeletePath(t *testing.T) {
	var registry repositories.ICampaignRepository
	registryType := reflect.TypeOf(&registry).Elem()
	for _, forbidden := range []string{"Delete", "Remove", "Purge", "Destroy", "Drop"} {
		if _, found := registryType.MethodByName(forbidden); found {
			t.Errorf("ICampaignRepository MUST NOT expose %s (FR-005): there is no foreign key from "+
				"`campaign_claims` to `campaigns`, so deleting a row would orphan every grant that "+
				"names its code", forbidden)
		}
	}
}

// ---- helpers ----

// seedRegistryClaim inserts one claim for a campaign, which is what makes the
// budget immutable and what `ExistsClaims` reports.
//
// The claim's `user_id` carries a foreign key to `identity.users`, so an account
// row is inserted first. The columns are named explicitly rather than through the
// identity module's entity: this package may not import another module's
// `domain/entities` (constitution §II), and the columns it needs are the ones the
// NOT NULL constraints demand.
func seedRegistryClaim(t *testing.T, dbConn *gorm.DB, campaignCode string) {
	t.Helper()

	userID := uuid.New()
	if err := dbConn.Exec(
		`INSERT INTO identity.users (id, username, email, role_slug, created_at, updated_at)
		 VALUES (?, ?, ?, ?, now(), now())`,
		userID,
		"verify-026-"+userID.String()[:8],
		"verify-026-"+userID.String()[:8]+"@example.test",
		"user",
	).Error; err != nil {
		t.Fatalf("seed identity user for the claim: %v", err)
	}

	if err := dbConn.Exec(
		`INSERT INTO subscription.campaign_claims
		     (id, user_id, campaign_code, plan_slug, plan_id, registered_at, expires_at, grant_bucket)
		 VALUES (?, ?, ?, ?, ?, now(), now() + interval '30 days', 'main')`,
		uuid.New(), userID, campaignCode, "premium-monthly", uuid.New(),
	).Error; err != nil {
		t.Fatalf("seed claim: %v", err)
	}
}

func containsRegistryCode(rows []*entities.Campaign, code string) bool {
	for _, row := range rows {
		if row != nil && row.Code == code {
			return true
		}
	}
	return false
}
