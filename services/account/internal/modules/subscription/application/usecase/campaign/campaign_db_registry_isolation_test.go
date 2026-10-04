package campaign

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"

	"gorm.io/gorm"
)

// 026 T081 — the campaign registry must be ISOLATED in every Postgres-backed fixture of
// this package, and that isolation must be ASSERTED rather than assumed.
//
// WHY THIS FILE EXISTS, and why it is the first thing T081 does.
//
// The seed migration `20261003165500_seed_launch_campaign.sql` inserts the production
// campaign `launch-2026-10` with `starts_at = '2026-10-02T00:00:00+07:00'`. Every
// integration fixture in this package creates its campaign at
// `baseInstant().Add(-24 * time.Hour)`, and `baseInstant()` (`campaign_harness_test.go`)
// is EXACTLY `2026-10-02T00:00:00+07:00` — the seeded row's own `starts_at`.
//
// So on any database these fixtures can reach, the seeded row is a candidate for every
// account they create: it is not closed, it has no `ends_at`, and its window contains
// the fixture's `registered_at`. FR-040 then picks the LARGEST `starts_at`, which is the
// seeded row's, and the assertion below `claim.CampaignCode != campaignCode` is made
// against the wrong campaign. A test that grants from `launch-2026-10` fails, and one
// whose fixture campaign names a plan nobody inserted passes for the wrong reason
// (`plan_unavailable`) — which is worse, because it is green.
//
// THE TWO THINGS THIS FILE DOES, and neither of them is optional:
//
//  1. `isolateSeededCampaign` removes the row for the duration of one test and restores
//     it afterwards, so the registry holds what the fixture put there.
//  2. `requireRegistryIsolatedTo` ASSERTS that. The cleanup above is a silent
//     mechanism: if it were removed, or if the seeded row were re-inserted by a
//     migration replay, or if somebody left a campaign behind in a shared test
//     database, every assertion below would keep passing against the wrong campaign.
//     A precondition that names the offending file is the only thing that turns that
//     class of mistake into a failure at the right place.
//
// WHY DELETE AND RESTORE RATHER THAN CLOSE.
//
// `ListOpen` filters `closed_at IS NULL`, so a closure would be enough for the grant
// path — but a closed row is still in `ListAll`, the OPERATOR list, and FR-032 adds
// `status = closed` precisely so an operator can see it. Leaving a closed seed behind
// would make `requireRegistryIsolatedTo`'s "exactly this file's codes" assertion
// impossible to state, which is the whole reason it is stated that way.
//
// THE ROW IS RESTORED VERBATIM, every column, and the restore is VERIFIED. A test that
// deletes a seeded row and puts back an approximation of it would break
// `campaign_seed_integration_test.go` in another package for every later run — a
// failure with no obvious cause, in a file that has nothing to do with this one.
//
// WHAT THIS DOES NOT CLAIM. It neutralises the registry for THIS fixture. It does not
// serialise suites: two packages running against the same `TEST_DATABASE_URL` at the
// same moment can still see each other's rows, and the seeded row is the one row two
// packages legitimately care about. See the note on `isolateSeededCampaign`.

// dbSeededCampaignCode is the campaign the seed migration inserts.
//
// It is stated here rather than imported from the persistence package's own test
// constant because reaching into another package's test identifier couples two files
// that must be able to disagree: the seed value belongs to the migration, and a test in
// this package should be able to keep working if the seed ever moves.
const dbSeededCampaignCode = "launch-2026-10"

// capturedRegistryRow is one `subscription.campaigns` row, read verbatim.
//
// EVERY COLUMN, and that is the point: the restore has to put back what was there, and
// a struct that carried only the six documented values would silently reset `version`,
// `closed_at` and the timestamps on a database where somebody had changed them.
type capturedRegistryRow struct {
	Code      string     `gorm:"column:code"`
	PlanSlug  string     `gorm:"column:plan_slug"`
	Quota     int        `gorm:"column:quota"`
	Reserve   int        `gorm:"column:reserve"`
	StartsAt  time.Time  `gorm:"column:starts_at"`
	EndsAt    *time.Time `gorm:"column:ends_at"`
	ClosedAt  *time.Time `gorm:"column:closed_at"`
	Version   int        `gorm:"column:version"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
}

const selectRegistryColumns = `code, plan_slug, quota, reserve, starts_at, ends_at,
	closed_at, version, created_at, updated_at`

// isolateSeededCampaign removes the seeded production campaign for the duration of one
// test and restores it in `t.Cleanup`.
//
// A MISSING ROW IS NOT AN ERROR. A database where the seed was rolled back, or one a
// developer emptied on purpose, needs no isolation, and failing here would make a
// legitimate environment look broken.
//
// THE DELETE IS VERIFIED rather than assumed: a fixture that silently ran against a
// registry still holding the seed is the exact failure this task exists to stop, and
// "the DELETE reported no error" is not evidence that the row is gone.
//
// CALL IT AFTER THE MIGRATIONS AND BEFORE THE PREFIX CLEANUP IS REGISTERED. Cleanups
// run last-in-first-out, so this order makes the prefix cleanup remove the fixture's own
// rows first and the seed come back last — which is the order that leaves the database
// as it was found even if a test failed halfway.
//
// IT DOES NOT SERIALISE SUITES, and cannot from inside one file. `make test-integration`
// runs `go test ./internal/modules/subscription/...`, which executes the `campaign` and
// `persistence` package binaries IN PARALLEL against one `TEST_DATABASE_URL`. While this
// row is deleted, `campaign_seed_integration_test.go`'s
// `TestIntegrationSeededCampaignCarriesTheDocumentedValues` would find no row and fail.
// The window is one fixture's duration and the collision needs both tests to overlap
// exactly, so the honest description is "a rare flake", not "a broken suite" — but it is
// real, and closing it needs a lock both packages take. That is a decision for whoever
// provisions the database, not something to decide inside a test helper.
func isolateSeededCampaign(t *testing.T, dbConn *gorm.DB) {
	t.Helper()
	ctx := context.Background()

	rows, err := readRegistryRow(ctx, dbConn, dbSeededCampaignCode)
	if err != nil {
		t.Fatalf("read the seeded campaign %s before isolating it: %v", dbSeededCampaignCode, err)
	}
	if len(rows) == 0 {
		return
	}
	seeded := rows[0]

	// REGISTERED BEFORE THE DELETE, so a failure between the two still restores the row.
	// The cleanup is a DELETE plus an INSERT of the values captured above, so running it
	// on an untouched database is a no-op with the same contents.
	t.Cleanup(func() { restoreSeededCampaign(t, dbConn, seeded) })

	if err := dbConn.WithContext(ctx).
		Exec(`DELETE FROM subscription.campaigns WHERE code = ?`, seeded.Code).Error; err != nil {
		t.Fatalf("isolate the seeded campaign %s: %v", dbSeededCampaignCode, err)
	}
	var remaining int64
	if err := dbConn.WithContext(ctx).Raw(
		`SELECT count(*) FROM subscription.campaigns WHERE code = ?`, seeded.Code,
	).Scan(&remaining).Error; err != nil {
		t.Fatalf("count %s after isolating it: %v", seeded.Code, err)
	}
	if remaining != 0 {
		t.Fatalf("the seeded campaign %s is still in the registry after DELETE; every assertion in "+
			"this package would be made against the wrong campaign", seeded.Code)
	}
}

// restoreSeededCampaign puts the captured row back, exactly as it was.
//
// THE DELETE BEFORE THE INSERT is not defensive noise: the seed migration is
// idempotent and its own test replays the INSERT, so a row can reappear under this code
// while the fixture runs.
//
// `ON CONFLICT DO NOTHING` THEN, and the reason is the verification below rather than
// politeness: if something else won the race for this code, a plain INSERT would fail
// with a primary-key violation that reads like a schema problem, while DO NOTHING lets
// the comparison report the ACTUAL difference — "the row that is there is not the row
// that was captured", which is the fact worth knowing.
//
// FAILURES ARE REPORTED WITH `t.Errorf` AND NOT `t.Fatalf`. This runs from `t.Cleanup`,
// where a fatal error abandons the cleanups registered after it — and abandoning them
// is how a failed restore would also leave the fixture's own rows behind.
func restoreSeededCampaign(t *testing.T, dbConn *gorm.DB, seeded capturedRegistryRow) {
	t.Helper()
	ctx := context.Background()

	if err := dbConn.WithContext(ctx).
		Exec(`DELETE FROM subscription.campaigns WHERE code = ?`, seeded.Code).Error; err != nil {
		t.Errorf("clear %s before restoring it: %v", seeded.Code, err)
		return
	}
	err := dbConn.WithContext(ctx).Exec(
		`INSERT INTO subscription.campaigns
		   (code, plan_slug, quota, reserve, starts_at, ends_at, closed_at, version, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (code) DO NOTHING`,
		seeded.Code, seeded.PlanSlug, seeded.Quota, seeded.Reserve, seeded.StartsAt,
		seeded.EndsAt, seeded.ClosedAt, seeded.Version, seeded.CreatedAt, seeded.UpdatedAt,
	).Error
	if err != nil {
		t.Errorf("restore the seeded campaign %s (%s/%d/%d, starts_at %s): %v. A test database left "+
			"without it fails campaign_seed_integration_test.go for every later run, with no visible cause",
			seeded.Code, seeded.PlanSlug, seeded.Quota, seeded.Reserve, seeded.StartsAt.UTC(), err)
		return
	}

	restored, err := readRegistryRow(ctx, dbConn, seeded.Code)
	if err != nil {
		t.Errorf("re-read the restored campaign %s: %v", seeded.Code, err)
		return
	}
	if len(restored) != 1 {
		t.Errorf("expected exactly 1 row for the restored %s, got %d", seeded.Code, len(restored))
		return
	}
	got := restored[0]
	if got.PlanSlug != seeded.PlanSlug || got.Quota != seeded.Quota || got.Reserve != seeded.Reserve ||
		!got.StartsAt.Equal(seeded.StartsAt) || !sameInstantPtr(got.EndsAt, seeded.EndsAt) ||
		!sameInstantPtr(got.ClosedAt, seeded.ClosedAt) || got.Version != seeded.Version {
		t.Errorf("the seeded campaign %s was not restored verbatim: before %+v after %+v", seeded.Code, seeded, got)
	}
}

// readRegistryRow reads the named campaign's row, or nothing when there is none.
func readRegistryRow(
	ctx context.Context,
	dbConn *gorm.DB,
	campaignCode string,
) ([]capturedRegistryRow, error) {
	var rows []capturedRegistryRow
	err := dbConn.WithContext(ctx).Raw(
		`SELECT `+selectRegistryColumns+` FROM subscription.campaigns WHERE code = ?`, campaignCode,
	).Scan(&rows).Error
	return rows, err
}

// sameInstantPtr compares two nullable instants as instants.
//
// `Equal` on a nil pointer is a nil-pointer dereference, and "absent" is one of the
// two values this file has to compare correctly: `ends_at IS NULL` on the seeded
// campaign is part of what FR-090 fixes.
func sameInstantPtr(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

// requireRegistryIsolatedTo is the PRECONDITION every Postgres-backed fixture in this
// package asserts before it makes a claim about a grant.
//
// It states the registry holds this file's campaigns and nothing else, in both
// directions: a foreign code is the seeded row (or a leftover from another run) that
// would win FR-040's ordering, and a missing own code means the fixture did not create
// what it is about to assert on.
//
// THE FAILURE NAMES THE FILE. `ownerPrefix` is unique per fixture file, and the caller's
// file name is read from the stack, so a dirty registry produces "which suite is this?"
// rather than a downstream assertion about the wrong campaign — which is the difference
// between a five-minute fix and an afternoon.
//
// `ListAll` RATHER THAN `ListOpen`, deliberately. The operator list shows closed
// campaigns (FR-032), so it is the stricter question: a leftover CLOSED campaign would
// satisfy a `ListOpen` check while still being a row somebody has to delete.
func requireRegistryIsolatedTo(
	t *testing.T,
	registry repositories.ICampaignRepository,
	ownerPrefix string,
	ownCodes ...string,
) {
	t.Helper()
	rows, err := registry.ListAll(t.Context())
	if err != nil {
		t.Fatalf("%s: read the registry for the isolation precondition: %v", callerFileName(2), err)
	}
	present := make(map[string]bool, len(rows))
	for _, row := range rows {
		if row != nil {
			present[row.Code] = true
		}
	}
	var foreign []string
	for code := range present {
		if !strings.HasPrefix(code, ownerPrefix) {
			foreign = append(foreign, code)
		}
	}
	if len(foreign) > 0 {
		sort.Strings(foreign)
		t.Fatalf("%s: registry isolation precondition failed. This suite owns only %q, and the registry "+
			"also holds %v. FR-040 selects the campaign with the LARGEST starts_at, so a foreign row wins "+
			"the grant and every assertion below is made against the wrong campaign — %q is the seeded "+
			"production campaign and belongs to no fixture",
			callerFileName(2), ownerPrefix, foreign, dbSeededCampaignCode)
	}
	for _, code := range ownCodes {
		if !present[code] {
			t.Fatalf("%s: registry isolation precondition failed. The fixture's own campaign %q is not "+
				"in the registry, so the assertions that follow would be about nothing",
				callerFileName(2), code)
		}
	}
}

// callerFileName names the Go file `skip` frames above the caller.
//
// A test failure that cannot say WHICH fixture produced it is the failure this package's
// headers spend their length avoiding, and the file is the only thing that identifies a
// fixture — `t.Name()` gives the test function, and several of them share one.
func callerFileName(skip int) string {
	if _, file, _, ok := runtime.Caller(skip); ok {
		return filepath.Base(file)
	}
	return fmt.Sprintf("frame %d", skip)
}
