package persistence

import (
	"sort"
	"strings"
	"testing"

	"gorm.io/gorm"
)

// 026 T027 (FR-091): 026 must add the registry and change NOTHING else.
//
// THE FAILURE THIS GUARDS. The registry introduces a table the two 025 tables
// could have been pointed at. Adding a foreign key `campaign_claims.campaign_code
// -> campaigns.code` looks like free referential integrity and is in fact a data
// loss waiting to happen: those columns are historical TEXT keys, the ledger can
// legitimately name a code that is not in the registry, and a migration cannot
// faithfully reconstruct attributes for every historical code. Worse, the
// constraint could BLOCK the migration on live data — the migration that was
// supposed to be a no-meaning-change cut-over would become the risky one
// (spec.md Câu 5).
//
// So the requirement is stated as an assertion rather than as a prohibition in a
// comment: after the 026 migrations, the two 025 tables must still carry EXACTLY
// the columns and the indexes 025 gave them, and `subscription.campaigns` must have
// no foreign key in either direction.
//
// It reads the migrated schema rather than the Go structs because `AutoMigrate`
// would report what the structs imply instead of what production runs.

// TestIntegrationCampaignClaimsSchemaIsUnchanged pins `campaign_claims` to the
// shape 025 T016 created — ten columns and the four indexes its migration names.
func TestIntegrationCampaignClaimsSchemaIsUnchanged(t *testing.T) {
	dbConn := openRegistryTestDB(t)

	assertColumns(t, dbConn, "subscription", "campaign_claims", []string{
		"expires_at",
		"fills_main_quota",
		"grant_bucket",
		"granted_at",
		"id",
		"plan_id",
		"plan_slug",
		"registered_at",
		"user_id",
	})
	assertIndexes(t, dbConn, "subscription", "campaign_claims", []string{
		"campaign_claims_pkey",
		"ix_campaign_claims_campaign_bucket",
		"ix_campaign_claims_campaign_granted",
		"ux_campaign_claims_user",
		"ux_campaign_claims_watermark",
	})
}

// TestIntegrationCampaignSweepRunsSchemaIsUnchanged pins the sweep health table to
// the shape 025 T079a created. It is the one row per campaign that survives a
// close (spec.md Câu 9), so quietly losing a column would lose the operator's view
// of a campaign that stopped granting.
func TestIntegrationCampaignSweepRunsSchemaIsUnchanged(t *testing.T) {
	dbConn := openRegistryTestDB(t)

	assertColumns(t, dbConn, "subscription", "campaign_sweep_runs", []string{
		"campaign_code",
		"degraded",
		"degraded_reason",
		"last_sweep_at",
		"updated_at",
	})
	assertIndexes(t, dbConn, "subscription", "campaign_sweep_runs", []string{
		"campaign_sweep_runs_pkey",
	})
}

// TestIntegrationRegistryCarriesNoForeignKey is FR-091 stated directly for the new
// table.
//
// There is nothing for it to point at: the registry is the root of this part of the
// model, and `plan_slug` is deliberately a TEXT reference checked at RUN time
// (FR-014) so that loading configuration never reaches into the database. A foreign
// key to `subscription_plans` would also make a campaign un-creatable while a plan
// is paused, which is a failure nobody would understand from the message.
func TestIntegrationRegistryCarriesNoForeignKey(t *testing.T) {
	dbConn := openRegistryTestDB(t)

	var count int64
	if err := dbConn.Raw(`
		SELECT COUNT(*)
		  FROM pg_constraint c
		  JOIN pg_class t ON t.oid = c.conrelid
		  JOIN pg_namespace n ON n.oid = t.relnamespace
		 WHERE n.nspname = 'subscription'
		   AND t.relname = 'campaigns'
		   AND c.contype = 'f'`).Scan(&count).Error; err != nil {
		t.Fatalf("query foreign keys on subscription.campaigns: %v", err)
	}
	if count != 0 {
		t.Errorf("FR-091: subscription.campaigns must carry NO foreign key, found %d. "+
			"plan_slug is checked at RUN time on purpose (FR-014) so that loading configuration "+
			"never reaches into the database, and a paused plan must not make a campaign "+
			"un-creatable", count)
	}
}

// TestIntegrationNoForeignKeyPointsAtTheRegistry is the other half of FR-091: not
// even an INCOMING reference may exist, because the ledger's `campaign_code` is
// free text by design (spec.md Câu 5, research.md R5).
func TestIntegrationNoForeignKeyPointsAtTheRegistry(t *testing.T) {
	dbConn := openRegistryTestDB(t)

	var offenders []string
	if err := dbConn.Raw(`
		SELECT src.relname
		  FROM pg_constraint c
		  JOIN pg_class src ON src.oid = c.conrelid
		  JOIN pg_class tgt ON tgt.oid = c.confrelid
		  JOIN pg_namespace ns ON ns.oid = src.relnamespace
		  JOIN pg_namespace nt ON nt.oid = tgt.relnamespace
		 WHERE c.contype = 'f'
		   AND nt.nspname = 'subscription'
		   AND tgt.relname = 'campaigns'
		   AND src.relname <> 'campaigns'`).Scan(&offenders).Error; err != nil {
		t.Fatalf("query foreign keys pointing at subscription.campaigns: %v", err)
	}
	if len(offenders) > 0 {
		sort.Strings(offenders)
		t.Errorf("FR-091: these tables must not reference subscription.campaigns: %s. "+
			"Their `campaign_code` columns are historical text keys, and constraining them "+
			"would orphan real grants and could block the migration on live data",
			strings.Join(offenders, ", "))
	}
}

// TestIntegrationRegistryKeepsItsShape pins the ten columns FR-010 requires, so a
// later "just add a column" is at least visible in the same diff as the entity that
// reads it.
func TestIntegrationRegistryKeepsItsShape(t *testing.T) {
	dbConn := openRegistryTestDB(t)

	assertColumns(t, dbConn, "subscription", "campaigns", []string{
		"closed_at",
		"code",
		"created_at",
		"ends_at",
		"plan_slug",
		"quota",
		"reserve",
		"starts_at",
		"updated_at",
		"version",
	})
	assertIndexes(t, dbConn, "subscription", "campaigns", []string{
		"campaigns_pkey",
		"ix_campaigns_starts_at",
	})
}

// ---- helpers ----

// assertColumns compares the table's column names, sorted, against the expected
// set. Equality rather than containment: an ADDED column is exactly the change
// FR-091 forbids, and a containment check would wave it through.
func assertColumns(t *testing.T, dbConn *gorm.DB, schema, table string, want []string) {
	t.Helper()

	var got []string
	if err := dbConn.Raw(`
		SELECT column_name FROM information_schema.columns
		 WHERE table_schema = ? AND table_name = ?`, schema, table).Scan(&got).Error; err != nil {
		t.Fatalf("query columns of %s.%s: %v", schema, table, err)
	}
	assertSameNames(t, "FR-091", "columns of "+schema+"."+table, got, want)
}

// assertIndexes compares the index names on one table, ignoring the indexes
// PostgreSQL creates for constraints it names itself. Those names are a
// PostgreSQL implementation detail — a UNIQUE constraint's index is called
// `<constraint>` or `<table>_<column>_key` depending on how it was written — and
// asserting on them would fail on a rename that changes no behaviour.
func assertIndexes(t *testing.T, dbConn *gorm.DB, schema, table string, want []string) {
	t.Helper()

	var rows []struct {
		Name  string
		Owner string
	}
	if err := dbConn.Raw(`
		SELECT i.relname AS name, COALESCE(c.conname, '') AS owner
		  FROM pg_class t
		  JOIN pg_namespace n ON n.oid = t.relnamespace
		  JOIN pg_index ix ON ix.indrelid = t.oid
		  JOIN pg_class i ON i.oid = ix.indexrelid
		  LEFT JOIN pg_constraint c ON c.conrelid = t.oid AND c.conindid = ix.indexrelid
		 WHERE n.nspname = ? AND t.relname = ?`, schema, table).Scan(&rows).Error; err != nil {
		t.Fatalf("query indexes of %s.%s: %v", schema, table, err)
	}

	var got []string
	for _, row := range rows {
		if row.Owner == "" {
			// A bare CREATE INDEX, named by the migration. Those are the ones worth
			// pinning: an index nobody declared is a performance regression waiting
			// to be discovered as a slow signup path.
			got = append(got, row.Name)
		}
	}
	assertSameNames(t, "FR-091", "explicit indexes on "+schema+"."+table, got, want)
}

// assertSameNames compares two name lists and reports the difference in both
// directions, so a failure says what arrived and what went.
//
// `requirement` names the rule being asserted — FR-091 here, FR-010 for the registry's
// declared column list — because the two callers ask visibly different questions of the
// same helper and a failure message that says only which names differ leaves the reader
// guessing which contract was broken.
func assertSameNames(t *testing.T, requirement, label string, got, want []string) {
	t.Helper()

	sort.Strings(got)
	sorted := make([]string, len(want))
	copy(sorted, want)
	sort.Strings(sorted)

	if strings.Join(got, ",") == strings.Join(sorted, ",") {
		return
	}
	t.Errorf("%s: the %s changed.\n  got:  %v\n  want: %v",
		requirement, label, got, sorted)
}
