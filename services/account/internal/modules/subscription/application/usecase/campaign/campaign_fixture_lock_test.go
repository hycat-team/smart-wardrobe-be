package campaign

import (
	"os"
	"strings"
	"testing"

	"smart-wardrobe-be/services/account/internal/modules/subscription/infrastructure/persistence"
	campaigndb "smart-wardrobe-be/shared/testing/campaigndb"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestIntegrationCampaignOpenerPurgesStaleFixturesUnderTheLock is the regression
// guard for 031/D-01.
//
// A previous run that failed midway, or another package running in parallel
// against the same TEST_DATABASE_URL, can leave `verify-026-*` rows in the
// registry. `requireRegistryIsolatedTo` would then fail every later suite - and
// rightly so, because FR-040 would pick the leftover row. The opener now takes
// the cross-package advisory lock and purges those rows before any fixture
// asserts isolation.
//
// The test seeds the stale row BEFORE opening, exactly as a killed run would,
// and then proves the normal opener removed it.
func TestIntegrationCampaignOpenerPurgesStaleFixturesUnderTheLock(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping campaign fixture cleanup test")
	}
	raw, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("connect test database: %v", err)
	}
	staleCode := campaigndb.FixturePrefix + "stale-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	if err := raw.Exec(`
		INSERT INTO subscription.campaigns (code, plan_slug, quota, reserve, starts_at)
		VALUES (?, 'premium-monthly', 1, 0, now() - interval '1 day')`, staleCode).Error; err != nil {
		t.Fatalf("seed stale campaign fixture %s: %v", staleCode, err)
	}
	t.Cleanup(func() {
		raw.Exec(`DELETE FROM subscription.campaigns WHERE code = ?`, staleCode)
		if sqlDB, dbErr := raw.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
	})
	if sqlDB, dbErr := raw.DB(); dbErr == nil {
		_ = sqlDB.Close()
	}

	// The ordinary fixture opener: it acquires the advisory lock (which also
	// serializes this package against the persistence package), purges the
	// namespace, and only then isolates the seeded campaign.
	dbConn := openAuditTestDB(t)

	var remaining int64
	if err := dbConn.Raw(
		`SELECT count(*) FROM subscription.campaigns WHERE code = ?`, staleCode,
	).Scan(&remaining).Error; err != nil {
		t.Fatalf("count stale fixture after opener: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("the opener left stale fixture %s in the registry; a later suite would fail its isolation precondition", staleCode)
	}

	requireRegistryIsolatedTo(t, persistence.NewCampaignRepository(dbConn), dbAuditPrefix)
}
