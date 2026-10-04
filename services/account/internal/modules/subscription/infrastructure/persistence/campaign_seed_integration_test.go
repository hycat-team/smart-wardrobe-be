package persistence

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	accountmigrations "smart-wardrobe-be/services/account/database/migrations"
	shareddb "smart-wardrobe-be/shared/infrastructure/db"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 026 T059 (FR-005, FR-090, FR-091, FR-092) ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â the SEED migration is truthful and the
// ROLLBACK is safe.
//
// THIS IS quickstart V4, and V4 is the one scenario the safety table in `quickstart.md`
// Ãƒâ€šÃ‚Â§0 forbids on production: it runs migrations and rolls them back, and rolling back
// drops a table. So this file is written to be READ on a review machine and RUN on a
// throwaway copy ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â which is why the destructive half needs an explicit acknowledgement
// in addition to `TEST_DATABASE_URL`, explained below.
//
// WHAT EACH TEST PROVES, and why the split matters:
//
//	seed values   ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â the six values FR-090 fixes, so "moving launch-2026-10 into the
//	                registry" provably means the same thing as the YAML it replaces;
//	seed twice    ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â US6 scenario 2, re-running the INSERT produces no duplicate;
//	rollback      ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â US6 scenario 4, TWO steps, and `campaign_claims` survives both.
//
// WHY THE ROLLBACK NEEDS TWO RUNS, stated because it is the thing an operator gets
// wrong. `goose down` rolls back EXACTLY ONE VERSION per invocation (see the `down`
// target in the Makefile, which passes no count). 026 added TWO adjacent migrations ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â
// the schema and then the seed ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â so one `make migrate-down svc=account` undoes the SEED
// and leaves the table standing. An operator who ran it once, saw `campaigns` still
// there, and concluded the rollback was broken has just learned nothing and changed
// nothing; the table comes back on the next boot's `migrate-up`. The test asserts the
// SEQUENCE and the reason, not just the end state, because the end state alone would
// pass with a single-step rollback that also worked.

// The two 026 migrations, by name fragment rather than by full timestamp: a rename is a
// legitimate thing for a human to do, and the assertion that matters is that these two
// EXIST, that the seed's version is HIGHER than the schema's (so `down` reaches the seed
// first), and that their Down sections do the right thing.
const (
	registrySchemaMigration = "add_campaigns"
	registrySeedMigration   = "seed_launch_campaign"
	seededCampaignCode      = "launch-2026-10"
)

// requireRollbackConsent gates the DESTRUCTIVE half.
//
// `TEST_DATABASE_URL` alone is not enough here, and the difference from every other
// Postgres-backed test in this repository is the reason. Those tests add rows and delete
// their own rows; this one DROPS `subscription.campaigns`, which would leave every other
// test in this package ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â and any service sharing that database ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â running against a
// schema that is not there. quickstart V4 says the same thing in prose ("chÃƒÂ¡Ã‚ÂºÃ‚Â¡y trÃƒÆ’Ã‚Âªn bÃƒÂ¡Ã‚ÂºÃ‚Â£n
// sao"); this is the machine-readable form of it.
//
// The acknowledgement is a second environment variable rather than a flag because a flag
// can be pasted from a wiki and an environment variable names the run it was given to.
func requireRollbackConsent(t *testing.T) {
	t.Helper()
	if os.Getenv("TEST_DATABASE_ALLOW_REGISTRY_ROLLBACK") != "true" {
		t.Skip("TEST_DATABASE_ALLOW_REGISTRY_ROLLBACK is not \"true\"; skipping the registry rollback " +
			"test. It DROPs subscription.campaigns, so it must be run against a throwaway copy of the " +
			"database, never against one other tests share (quickstart V4)")
	}
}

// TestIntegrationSeededCampaignCarriesTheDocumentedValues is FR-090's whole content.
//
// The six values are the point of the seed: they were copied from
// `configs/config.production.yaml`, and if any of them had been mistyped then switching
// the source from configuration to the registry would silently change the meaning of a
// promotion that is already running for real customers (spec.md CÃƒÆ’Ã‚Â¢u 5).
//
// `starts_at` IS COMPARED AS AN INSTANT and not as a string, because the column is
// `TIMESTAMPTZ`: a seed that stored `+07:00` and a test that compared it to `Z` would
// either fail for a reason that is not a defect or ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â worse ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â pass while the campaign
// opens seven hours late.
func TestIntegrationSeededCampaignCarriesTheDocumentedValues(t *testing.T) {
	dbConn := openRegistryTestDB(t)

	row := dbConn.Raw(
		`SELECT plan_slug, quota, reserve, starts_at, ends_at, closed_at, version
		   FROM subscription.campaigns WHERE code = ?`, seededCampaignCode,
	).Row()

	var (
		planSlug   string
		quota      int
		reserve    int
		startsAt   time.Time
		endsAt     *time.Time
		closedAt   *time.Time
		version    int
		scannedOne bool
	)
	if err := row.Scan(&planSlug, &quota, &reserve, &startsAt, &endsAt, &closedAt, &version); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("FR-090: the seed migration must have inserted %s", seededCampaignCode)
		}
		// `sql.ErrNoRows` is what a raw scan returns for an absent row; GORM does not
		// wrap it, so it is compared by message rather than by importing `database/sql`
		// into a test that otherwise needs no driver types.
		if strings.Contains(err.Error(), "no rows") {
			t.Fatalf("FR-090: the seed migration must have inserted %s", seededCampaignCode)
		}
		t.Fatalf("read the seeded campaign: %v", err)
	}
	scannedOne = true
	if !scannedOne {
		t.Fatal("the seeded campaign was not read")
	}

	if planSlug != "premium-monthly" || quota != 50 || reserve != 5 {
		t.Errorf("FR-090: the seed must carry premium-monthly/50/5, got %q/%d/%d",
			planSlug, quota, reserve)
	}
	if !startsAt.Equal(time.Date(2026, 10, 2, 0, 0, 0, 0, time.FixedZone("UTC+7", 7*60*60))) {
		t.Errorf("FR-090: starts_at = %s, want 2026-10-02T00:00:00+07:00 as an instant", startsAt)
	}
	if endsAt != nil {
		t.Errorf("FR-090: the seeded campaign is open-ended, ends_at = %v", endsAt)
	}
	if closedAt != nil {
		t.Errorf("FR-031: a seeded campaign must not be closed, closed_at = %v", closedAt)
	}
	if version != 1 {
		t.Errorf("FR-055: a seeded campaign must start at version 1, got %d", version)
	}
}

// TestIntegrationSeedInsertIsIdempotent is US6 scenario 2 ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â "run the migration again and
// nothing is duplicated, nothing errors" ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â asserted by REPLAYING THE INSERT rather
// than by re-running the whole migration set.
//
// The statement is taken out of the migration file itself (see `migrationSection`), so
// the test cannot drift from what actually ships: an assertion written with a
// hand-copied INSERT would keep passing after someone edited the seed.
func TestIntegrationSeedInsertIsIdempotent(t *testing.T) {
	dbConn := openRegistryTestDB(t)

	insert, ok := migrationSection(t, registrySeedMigration, "Up")
	if !ok {
		return
	}
	if !strings.Contains(insert, "ON CONFLICT") {
		t.Fatalf("FR-090/US6.2: the seed INSERT must be idempotent (`ON CONFLICT ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â¦ DO NOTHING`), got:\n%s",
			insert)
	}

	// TWO REPLAYS, because "running it twice does nothing" is only proved by a second
	// run: a plain `INSERT` fails on the second attempt, which is the failure this test
	// exists to prevent on a deploy that re-runs migrations on every boot.
	for attempt := 1; attempt <= 2; attempt++ {
		if err := dbConn.Exec(insert).Error; err != nil {
			t.Fatalf("US6.2: replaying the seed INSERT (attempt %d) must not error, got %v", attempt, err)
		}
	}

	var count int64
	if err := dbConn.Raw(
		`SELECT count(*) FROM subscription.campaigns WHERE code = ?`, seededCampaignCode,
	).Scan(&count).Error; err != nil {
		t.Fatalf("count the seeded rows: %v", err)
	}
	if count != 1 {
		t.Errorf("US6.2: %d rows for %s after two replays, want exactly 1", count, seededCampaignCode)
	}
}

// TestIntegrationRegistryRollbackNeedsTwoStepsAndKeepsClaims is US6 scenario 4, and it
// is the only test in 026 that destroys schema.
//
// THE ORDER IS THE ASSERTION. The seed's Down deletes one row; the schema's Down drops
// the table. Rolling them back in the wrong order would either fail (the seed's DELETE
// against a dropped table) or, worse, appear to succeed while leaving the seeded row
// behind in a table that no longer exists to describe it. So the test walks the two
// versions in the order goose would and asserts the intermediate state as well as the
// final one ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â the intermediate state being "the table is still there, the seeded row is
// gone, the claims are untouched".
func TestIntegrationRegistryRollbackNeedsTwoStepsAndKeepsClaims(t *testing.T) {
	requireRollbackConsent(t)
	dbConn := openRegistryTestDB(t)

	// One real claim on the seeded campaign, so "the grants survive" is a statement
	// about a non-empty table rather than about an empty one.
	seedRegistryClaim(t, dbConn, seededCampaignCode)

	claimsBefore := countRows(t, dbConn, "subscription.campaign_claims")

	seedDown, seedOK := migrationSection(t, registrySeedMigration, "Down")
	schemaDown, schemaOK := migrationSection(t, registrySchemaMigration, "Down")
	if !seedOK || !schemaOK {
		t.Fatalf("FR-092: both 026 migrations must carry a `-- +goose Down` section "+
			"(seed=%t, schema=%t); without one the rollback would leave the registry behind",
			seedOK, schemaOK)
	}

	// The SEED version must be HIGHER than the schema's, which is what makes `goose down`
	// reach the SEED first and forces the two-run sequence quickstart V4 spells out. That
	// comparison is arithmetic on file NAMES and needs neither this database nor a
	// dropped table, so it lives in its own test ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â
	// `TestTheSeedMigrationRunsAfterTheSchemaMigration` ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â and is not re-checked here: a
	// guard behind `requireRollbackConsent` would be a guard that never runs.
	t.Cleanup(func() {
		// Leave the database as it was found: without this, every other Postgres-backed
		// test in this package would fail against a dropped registry.
		if err := shareddb.RunMigrations(context.Background(), dbConn, accountmigrations.EmbedFS,
			"goose_db_version", true, zap.NewNop()); err != nil {
			t.Errorf("restore the schema after the rollback test: %v", err)
		}
	})

	// ---- STEP 1 of 2: `make migrate-down svc=account` (the seed) ----
	if err := dbConn.Exec(seedDown).Error; err != nil {
		t.Fatalf("US6.4: the seed rollback must succeed, got %v", err)
	}
	if !tableExists(t, dbConn, "subscription.campaigns") {
		t.Fatal("the FIRST rollback must not drop the table ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â `goose down` rolls back ONE version, " +
			"which is the seed. A single-run rollback that dropped the table would mean the seed " +
			"migration carried the schema's Down section")
	}
	var seededLeft int64
	if err := dbConn.Raw(
		`SELECT count(*) FROM subscription.campaigns WHERE code = ?`, seededCampaignCode,
	).Scan(&seededLeft).Error; err != nil {
		t.Fatalf("count the seeded rows after the first rollback: %v", err)
	}
	if seededLeft != 0 {
		t.Errorf("US6.4: the first rollback must remove the seeded row, %d left", seededLeft)
	}
	if got := countRows(t, dbConn, "subscription.campaign_claims"); got != claimsBefore {
		t.Errorf("FR-092: campaign_claims changed from %d to %d rows during the seed rollback",
			claimsBefore, got)
	}

	// ---- STEP 2 of 2: `make migrate-down svc=account` again (the schema) ----
	if err := dbConn.Exec(schemaDown).Error; err != nil {
		t.Fatalf("US6.4: the schema rollback must succeed, got %v", err)
	}
	if tableExists(t, dbConn, "subscription.campaigns") {
		t.Error("US6.4: the SECOND rollback must drop subscription.campaigns, and it did not")
	}
	// THE CLAIM THAT MATTERS MOST, and the reason FR-091 says `campaign_claims` carries
	// no foreign key: dropping the registry costs the registry and NOT ONE GRANT. A
	// foreign key here would cascade-delete every campaign's history on rollback.
	if got := countRows(t, dbConn, "subscription.campaign_claims"); got != claimsBefore {
		t.Errorf("FR-092: campaign_claims went from %d to %d rows across the rollback ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â "+
			"rolling back the registry must cost no grant", claimsBefore, got)
	}
}

// TestTheSeedMigrationRunsAfterTheSchemaMigration is FR-090's ordering half, and it is a
// SEPARATE test for a reason that is about gates, not about style.
//
// IT WAS PREVIOUSLY INSIDE `TestIntegrationRegistryRollbackNeedsTwoStepsAndKeepsClaims`,
// where it sat behind two guards it does not need: `requireRollbackConsent` (skip unless
// `TEST_DATABASE_ALLOW_REGISTRY_ROLLBACK=true`) and `openRegistryTestDB` (skip unless
// `TEST_DATABASE_URL` is set). The comparison itself is arithmetic on two file NAMES ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â
// `migrationVersions` reads `fs.ReadDir(accountmigrations.EmbedFS, ".")` and parses the
// timestamp prefixes ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â so it touched no database and dropped no table, and it had never
// executed anywhere. The assertion existed and proved nothing.
//
// WHY THE ORDER IS LOAD-BEARING RATHER THAN COSMETIC. `goose down` rolls back EXACTLY ONE
// version per invocation. With the seed numbered BELOW the schema, the first rollback
// would `DROP subscription.campaigns` and the second would then `DELETE FROM
// subscription.campaigns WHERE code = 'launch-2026-10'` against a table that no longer
// exists ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â the rollback would either fail outright or, worse, leave the seeded row behind
// in a table that no longer exists to describe it. That is why quickstart V4 needs TWO
// runs, and an operator who learns that only by reading this assertion is an operator
// who learns it by breaking a rollback.
//
// IT IS NOT NAMED `TestIntegrationÃƒÂ¢Ã¢â€šÂ¬Ã‚Â¦` although it lives in an integration test file,
// because it needs no database, and the name is the fastest signal a reader has.
func TestTheSeedMigrationRunsAfterTheSchemaMigration(t *testing.T) {
	schemaVersion, seedVersion := migrationVersions(t)
	if schemaVersion == 0 || seedVersion == 0 {
		t.Fatalf("FR-090: both 026 migrations must be embedded (schema=%d, seed=%d). The two-run "+
			"rollback quickstart V4 spells out only makes sense if there are TWO of them, so a "+
			"missing migration is this assertion's problem rather than something it can ignore",
			schemaVersion, seedVersion)
	}
	if seedVersion <= schemaVersion {
		t.Errorf("FR-090/FR-092: the seed migration (%d) MUST be numbered ABOVE the schema "+
			"migration (%d). `goose down` rolls back one version per run, so this order is what "+
			"makes the first run undo the seeded ROW and the second drop the TABLE ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â the "+
			"two-run sequence quickstart V4 requires. Reversed, the table would be dropped before "+
			"the row is deleted from it", seedVersion, schemaVersion)
	}
}

// TestIntegrationRegistryDownSectionsDoNotTouchTheClaimLedger is the STATIC half of the
// safety claim, and it needs no database at all.
//
// The two Down sections are read as TEXT so that the property is checked even where the
// rollback cannot be run. A `TRUNCATE subscription.campaign_claims`, or a `DROP` of the
// ledger, would be invisible to the behavioural test above in the one environment where
// that test is skipped ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â which is every developer machine and CI without a provisioned
// database, and therefore most of the places a regression would be introduced.
func TestIntegrationRegistryDownSectionsDoNotTouchTheClaimLedger(t *testing.T) {
	for _, name := range []string{registrySchemaMigration, registrySeedMigration} {
		down, ok := migrationSection(t, name, "Down")
		if !ok {
			t.Fatalf("FR-092: %s must carry a `-- +goose Down` section", name)
		}
		for _, forbidden := range []string{
			"campaign_claims",
			"TRUNCATE",
			"DELETE FROM subscription.campaigns WHERE code NOT IN",
		} {
			// Matched against the SQL with its COMMENTS REMOVED, which is the whole
			// subtlety of this test: the seed's Down section explains at length that
			// `campaign_claims` is untouched, and a naive search for the table name would
			// fail on the very sentence that documents the safety.
			if strings.Contains(stripSQLComments(down), forbidden) {
				t.Errorf("FR-091/FR-092: the executable Down section of %s mentions %q. Rolling back the "+
					"registry must touch `subscription.campaigns` ONLY:\n%s", name, forbidden, down)
			}
		}
	}
}

// TestIntegrationSeedDownIsScopedToTheSeededRow is the other static half, and it is the
// one that protects an operator's work.
//
// The seed's Down deletes by CODE. A `DELETE FROM subscription.campaigns` without a
// `WHERE` would be a valid rollback on a fresh database and catastrophic on a live one ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â
// it would remove every campaign an administrator created through the new API. That
// damage happens on the deploy where someone rolls back, which is the worst possible
// moment to discover it.
func TestIntegrationSeedDownIsScopedToTheSeededRow(t *testing.T) {
	down, ok := migrationSection(t, registrySeedMigration, "Down")
	if !ok {
		t.Fatalf("FR-092: %s must carry a `-- +goose Down` section", registrySeedMigration)
	}
	if !strings.Contains(down, "WHERE code = '"+seededCampaignCode+"'") {
		t.Errorf("FR-092: the seed's Down must delete ONLY `WHERE code = '%s'`, so rolling back "+
			"cannot delete campaigns an operator created through the API. Got:\n%s",
			seededCampaignCode, down)
	}
}

// ---- helpers ----

// migrationSection returns the SQL between one `-- +goose <Section>` marker and the
// next one, for the migration whose file name contains `nameFragment`.
//
// IT READS THE EMBEDDED FILES RATHER THAN THE DIRECTORY, because the embedded set is
// what the binary ships and what `RunMigrations` applies. Reading the directory would
// let the two drift ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â a migration left on disk but excluded from the embed would be
// asserted here and never applied.
// IT CLOSES THE `Up` SECTION AT THE `Down` MARKER, because an `Up` section that runs
// to end-of-file also swallows the `Down` body, and for the seed migration that body
// is a `DELETE FROM subscription.campaigns WHERE code = 'launch-2026-10'`. A caller
// asking for the `Up` SQL and then executing it ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â `TestIntegrationSeedInsertIsIdempotent`
// does exactly that ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â would insert the row and immediately delete it again, then count
// zero and fail for a reason that has nothing to do with idempotency. `Down` is the last
// section of a goose migration, so leaving its close marker empty is harmless there.
func migrationSection(t *testing.T, nameFragment string, section string) (string, bool) {
	t.Helper()

	entries, err := fs.ReadDir(accountmigrations.EmbedFS, ".")
	if err != nil {
		t.Fatalf("read the embedded migrations: %v", err)
	}

	openMarker := "-- +goose " + section
	closeMarker := ""
	if section == "Up" {
		closeMarker = "-- +goose Down"
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.Contains(entry.Name(), nameFragment) {
			continue
		}
		body, readErr := fs.ReadFile(accountmigrations.EmbedFS, entry.Name())
		if readErr != nil {
			t.Fatalf("read %s: %v", entry.Name(), readErr)
		}
		return betweenMarkers(string(body), openMarker, closeMarker), true
	}
	t.Errorf("no embedded migration matches %q; the 026 registry migrations must be present", nameFragment)
	return "", false
}

// betweenMarkers returns the text after `open` and before `close` (to the end when
// `close` is empty). Marker LINES are dropped so that the result is executable SQL:
// leaving `-- +goose StatementBegin` in the middle of a statement would make
// `dbConn.Exec` of it fail for a reason that has nothing to do with the rollback.
func betweenMarkers(body, open, close string) string {
	start := strings.Index(body, open)
	if start < 0 {
		return ""
	}
	rest := body[start+len(open):]
	end := len(rest)
	if close != "" {
		if idx := strings.Index(rest, close); idx >= 0 {
			end = idx
		}
	}
	lines := strings.Split(rest[:end], "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "-- +goose") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}

// migrationVersions returns the goose version numbers of the two 026 migrations, parsed
// out of their file names.
func migrationVersions(t *testing.T) (schemaVersion, seedVersion int64) {
	t.Helper()

	entries, err := fs.ReadDir(accountmigrations.EmbedFS, ".")
	if err != nil {
		t.Fatalf("read the embedded migrations: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		version, ok := leadingVersion(entry.Name())
		if !ok {
			continue
		}
		switch {
		case strings.Contains(entry.Name(), registrySchemaMigration):
			schemaVersion = version
		case strings.Contains(entry.Name(), registrySeedMigration):
			seedVersion = version
		}
	}
	return schemaVersion, seedVersion
}

// stripSQLComments removes `--` line comments and `/* ÃƒÂ¢Ã¢â€šÂ¬Ã‚Â¦ */` blocks so a safety check
// reads EXECUTABLE SQL rather than prose.
//
// The migrations in this repository comment extensively ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â and their Down sections
// explain their own safety in the comment above the statement ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â so a check that matched
// raw text would either fail on the documentation of the property it is asserting or,
// worse, be weakened by an exception carved out for "just the comment".
//
// IT IS NOT A SQL PARSER and does not try to be: it does not understand string
// literals, so a literal containing `--` would be truncated. None of the 026 Down
// sections has one, and the failure mode is a false POSITIVE (a check that refuses to
// match), which is the safe direction for a guard.
func stripSQLComments(sql string) string {
	for {
		start := strings.Index(sql, "/*")
		if start < 0 {
			break
		}
		end := strings.Index(sql[start:], "*/")
		if end < 0 {
			sql = sql[:start]
			break
		}
		sql = sql[:start] + " " + sql[start+end+2:]
	}

	lines := strings.Split(sql, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			line = line[:idx]
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

// leadingVersion reads the `20261003165440_` prefix goose orders by.
func leadingVersion(name string) (int64, bool) {
	underscore := strings.Index(name, "_")
	if underscore <= 0 {
		return 0, false
	}
	version, err := strconv.ParseInt(name[:underscore], 10, 64)
	if err != nil {
		return 0, false
	}
	return version, true
}

// tableExists reports whether a table is present, which is how the test observes the
// INTERMEDIATE state of a two-step rollback.
func tableExists(t *testing.T, dbConn *gorm.DB, qualifiedName string) bool {
	t.Helper()

	var found bool
	err := dbConn.Raw(
		`SELECT EXISTS (SELECT 1 FROM information_schema.tables
		                WHERE table_schema = split_part(?, '.', 1)
		                  AND table_name   = split_part(?, '.', 2))`,
		qualifiedName, qualifiedName,
	).Scan(&found).Error
	if err != nil {
		t.Fatalf("check whether %s exists: %v", qualifiedName, err)
	}
	return found
}

// countRows counts every row of a table.
//
// Used only on `campaign_claims`, whose foreign key to `identity.users` cascades on
// user deletion: the count is therefore taken with the fixture users still in place, and
// it is a COUNT of the whole table so the assertion is about the ledger rather than
// about this test's own rows.
func countRows(t *testing.T, dbConn *gorm.DB, qualifiedName string) int64 {
	t.Helper()

	var count int64
	if err := dbConn.Raw("SELECT count(*) FROM " + qualifiedName).Scan(&count).Error; err != nil {
		t.Fatalf("count %s: %v", qualifiedName, err)
	}
	return count
}
