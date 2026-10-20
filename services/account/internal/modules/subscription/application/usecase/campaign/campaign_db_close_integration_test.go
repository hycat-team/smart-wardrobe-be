package campaign

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	accountmigrations "smart-wardrobe-be/services/account/database/migrations"
	identitydto "smart-wardrobe-be/services/account/dto/identity"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	shareddb "smart-wardrobe-be/shared/infrastructure/db"
	campaigndb "smart-wardrobe-be/shared/testing/campaigndb"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// 026 T049 — quickstart V2, and the FR-045 half of quickstart V1 step 3–5, against a
// REAL Postgres.
//
// WHAT ONLY A REAL DATABASE CAN SHOW HERE. The claim T049 makes has two halves and
// neither is reachable through a fake:
//
//   - after a close, a NEW account gets nothing, and the observable reason is
//     `campaign_sleeping` — the exact code 025 already used for a finished campaign,
//     which is what keeps FR-050's twelve `reason` codes unchanged (spec.md Câu 4);
//   - a grant that is ALREADY PAST conditions 1-7 when the operator closes must be
//     stopped anyway, and it is stopped by a `SELECT ... FOR SHARE` that BLOCKS on the
//     close and then re-reads the row. A fake cannot make one call wait for another,
//     and the unit test in `registry_uc_close_test.go` cannot either: it decides which
//     answer the use case produces, not that the database refuses to produce two at
//     once.
//
// THE HARNESS IS THE ONE THIS PACKAGE ALREADY USES (026 T038): `TEST_DATABASE_URL`
// plus a `t.Skip`, and the schema from the EMBEDDED GOOSE MIGRATIONS rather than
// `AutoMigrate` — a schema built from the Go structs would have no CHECK constraints
// and no immutability trigger, so it is not the schema production runs.
//
// IT NEEDS A DATABASE ALREADY PROVISIONED WITH THE BASE `identity` SCHEMA, because the
// claim ledger's `user_id` carries a foreign key onto `identity.users(id)` and no
// migration in this repository creates that table.

// dbClosePrefix namespaces every campaign this file writes, so a run against a shared
// test database cannot collide with a real one or delete it. It is distinct from the
// open-half prefix so the two suites' cleanups cannot delete each other's rows.
const dbClosePrefix = "verify-026-close-"

// openCloseTestDB connects and migrates, skipping when no database is provisioned.
//
// The `prefix` argument exists so each file cleans up ONLY its own rows: the cleanup
// deletes by code prefix, and a shared helper with one hard-coded prefix would make
// every test in this package delete every other one's campaigns.
func openCloseTestDB(t *testing.T, prefix string) *gorm.DB {
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
	// The migrations, not `AutoMigrate`: the closed_at column and the immutability
	// trigger are part of what 026 changed (see the header).
	if err := shareddb.RunMigrations(context.Background(), dbConn, accountmigrations.EmbedFS, "goose_db_version", true, zap.NewNop()); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	// Registered before the prefix cleanup below, so the seeded campaign is restored
	// LAST (cleanups are last-in-first-out) — 026 T081.
	campaigndb.LockAndPurge(t, dbConn)
	isolateSeededCampaign(t, dbConn)
	t.Cleanup(func() {
		// Claims first: the ledger names the campaign as text and has no foreign key,
		// so the order is tidiness rather than a constraint (FR-091).
		dbConn.Exec("DELETE FROM subscription.campaign_claims WHERE campaign_code LIKE ?", prefix+"%")
		dbConn.Exec("DELETE FROM subscription.campaigns WHERE code LIKE ?", prefix+"%")
	})
	return dbConn
}

// openRunningCampaign creates a campaign whose window is already open, which is the
// state quickstart V1 step 2 grants from.
func openRunningCampaign(t *testing.T, f *dbGrantFixture, suffix string) string {
	t.Helper()
	code := dbClosePrefix + suffix
	planSlug := dbClosePrefix + "premium-monthly"
	quota := 3
	if _, err := f.registry.Create(context.Background(), uuid.New(), &sub_dto.CampaignCreateReq{
		Code:     code,
		PlanSlug: &planSlug,
		Quota:    &quota,
		StartsAt: baseInstant().Add(-24 * time.Hour).Format(time.RFC3339),
	}, nil); err != nil {
		t.Fatalf("create the campaign: %v", err)
	}
	// THE PRECONDITION (026 T081): this suite owns its own prefix and nothing else.
	// Without it the seeded `launch-2026-10` — which starts at exactly `baseInstant()`,
	// later than this fixture's campaign — wins FR-040 and every claim below is made
	// about the seed.
	requireRegistryIsolatedTo(t, f.campaign, dbClosePrefix, code)
	return code
}

// closeCampaign force-closes through the registry's own use case, reading the version
// back from the row first because FR-055 compares against the version the client held.
//
// The read is not a shortcut around the conditional UPDATE — the UPDATE still compares
// it, and a wrong version would still produce a 412. It is here because this test is
// the CLIENT, and a client sends the version it read.
func closeCampaign(t *testing.T, f *dbGrantFixture, campaignCode string) *entities.Campaign {
	t.Helper()
	ctx := context.Background()

	current, err := f.campaign.GetByCode(ctx, campaignCode)
	if err != nil || current == nil {
		t.Fatalf("read the campaign before closing it: err=%v row=%v", err, current)
	}
	reason := "Hết ngân sách khuyến mãi (verify-026)"
	summary, err := f.registry.Close(ctx, uuid.New(), campaignCode, &sub_dto.CampaignCloseReq{
		Version: current.Version,
		Reason:  &reason,
	}, nil)
	if err != nil {
		t.Fatalf("close the campaign: %v", err)
	}
	// FR-032: the response the operator sees says `closed`, not `expired` and not
	// `running`. Asserted through the use case because that is the code path the
	// endpoint calls.
	if summary.Status != sub_dto.CampaignStatusClosed {
		t.Errorf("the close response reported status %q, want %q", summary.Status, sub_dto.CampaignStatusClosed)
	}
	if summary.ClosedAt == nil {
		t.Error("the close response must carry closedAt (FR-051)")
	}
	return current
}

// TestIntegrationAClosedCampaignGrantsANewAccountNothing is US2 acceptance scenario 2
// and quickstart V2 in full.
//
// TWO ASSERTIONS, AND THE SECOND IS THE ONE THAT PROTECTS A DEPLOYED CONTRACT. "No
// claim" alone would also be satisfied by a campaign that failed for an unrelated
// reason — a missing plan, a spent budget, a broken message. So the reason code is
// asserted as well, which means the reason has to be the RIGHT one and not merely a
// code from the published twelve (FR-050).
//
// WHICH CODE, AND WHY IT IS `campaign_not_configured` RATHER THAN `campaign_sleeping`.
//
// With the registry isolated (026 T081), this file's campaign is the only row in it, so
// closing it leaves `ListOpen` EMPTY — and an empty open registry is condition 1's own
// answer, `campaign_not_configured` (`admission_uc.go`). That is the settled behaviour
// and it is not changed here: a closed campaign is not a candidate on ANY path
// (FR-031/FR-070), so on the message path there is nothing left to be "asleep about".
//
// `campaign_sleeping` IS still the answer for a closed campaign, and it is asserted
// where a closed campaign can still BE a candidate: the evaluator, in
// `TestEvaluatorExcludesForceClosedCampaigns` (`admission_uc_test.go`), and the
// operator's eligibility look-up, which is scoped to one code
// (`campaign_query_eligibility_test.go`). Both use a registry whose closed row is
// still listed, which is exactly the difference between the two questions.
//
// The earlier version of this test expected `campaign_sleeping` here and passed only
// because the seeded `launch-2026-10` was still open behind the fixture's closed
// campaign — the isolation defect T081 fixes. It asserted a reason the message path does
// not produce for this state.
func TestIntegrationAClosedCampaignGrantsANewAccountNothing(t *testing.T) {
	dbConn := openCloseTestDB(t, dbClosePrefix)
	ctx := context.Background()
	f := newDBGrantFixture(t, dbConn)

	campaignCode := openRunningCampaign(t, f, "no-new-grants")
	planSlug := dbClosePrefix + "premium-monthly"
	insertTestPlan(t, dbConn, planSlug)

	// STEP 3 of quickstart V1 — close it.
	closeCampaign(t, f, campaignCode)

	// The registry really is closed, read back through the same method the operator
	// screen uses. Asserting the response alone would pass even if the write were
	// discarded.
	stored, err := f.campaign.GetByCode(ctx, campaignCode)
	if err != nil || stored == nil {
		t.Fatalf("the closed campaign must be readable: err=%v row=%v", err, stored)
	}
	if stored.ClosedAt == nil {
		t.Fatal("FR-031: closed_at must be set on the stored row")
	}
	if stored.Version < 2 {
		t.Errorf("FR-055: version = %d, want it incremented by the close", stored.Version)
	}

	// STEP 4 — a second account registers after the closure.
	registeredAt := baseInstant()
	userID := newTestAccount(t, dbConn, registeredAt)
	if err := f.grant.ProcessUserCreated(ctx, identitydto.UserCreatedEventPayload{
		UserID:       userID,
		RegisteredAt: registeredAt,
	}); err != nil {
		t.Fatalf("a skip is a success, not a retryable failure: %v", err)
	}

	if _, found, err := f.claims.GetByUser(ctx, userID); err != nil {
		t.Fatalf("read the claim ledger: %v", err)
	} else if found {
		t.Errorf("FR-031/US2.2: a closed campaign must grant nothing, but %s holds a claim", userID)
	}
	// The customer must keep whatever they had — a closed campaign grants nobody
	// anything and revokes nothing (FR-033), and "not eligible for the promotion" is
	// not the same statement as "your subscription was cancelled".
	sub, err := f.subs.GetByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("read the subscription: %v", err)
	}
	if sub != nil && sub.CurrentPlanCode == planSlug {
		t.Errorf("an account must not be put on the campaign plan by a closed campaign (%q)", sub.CurrentPlanCode)
	}

	// The reason code, and it is read from the log because FR-061 puts it there.
	//
	// `campaign_not_configured`, and the test's header says why: with this suite's own
	// campaign closed and the registry isolated, `ListOpen` is empty, which is
	// condition 1's answer. `campaign_sleeping` for a closed campaign is asserted at the
	// evaluator and at the eligibility look-up, where a closed row is still listed.
	reason, ok := f.log.skipReasonFor(userID)
	if !ok {
		t.Fatal("the skip must be logged with a reason code (FR-061)")
	}
	if reason != SkipCampaignNotConfigured {
		t.Errorf("reason = %q, want %q — the closed campaign is no longer a candidate at all, so the "+
			"registry is empty. FR-050 keeps the reason set at exactly twelve codes and this is one of them; "+
			"see the test's header for why a closed campaign answers %q elsewhere",
			reason, SkipCampaignNotConfigured, SkipCampaignSleeping)
	}

	// AND THE REGISTRY REALLY IS EMPTY OF CANDIDATES, read back through the very
	// method the grant path draws from. Without it the reason above would be
	// indistinguishable from "some other campaign was awake".
	open, err := f.campaign.ListOpen(ctx)
	if err != nil {
		t.Fatalf("ListOpen after the close: %v", err)
	}
	if len(open) != 0 {
		t.Errorf("expected no candidate after closing the only campaign, got %d", len(open))
	}
}

// TestIntegrationAGrantInFlightIsStoppedByTheShareLock is FR-045, and it is the one
// thing in this feature that no amount of fake can express.
//
// THE SEQUENCE IS DELIBERATE, AND EVERY STEP OF IT IS NECESSARY:
//
//  1. A transaction takes `SELECT ... FOR SHARE` on the campaign row.
//  2. THE SAME TRANSACTION writes `closed_at`, which stays INVISIBLE to everybody else
//     until it commits. So the grant's `ListOpen` still sees an open campaign —
//     without this the grant would be skipped at condition 2 and FR-045 would never
//     be reached, which is exactly the shortcut a weaker version of this code would
//     take.
//  3. The signup message is processed on a SECOND connection. It passes conditions
//     1-7 and then BLOCKS on the row, because `FOR SHARE` and the close's row lock
//     conflict.
//  4. The transaction commits. The grant wakes up, re-reads the row, and sees
//     `closed_at`.
//
// Without step 3 nothing here proves anything: a re-read that never happened would
// produce exactly the same final state, because the campaign is closed either way. The
// blocking IS the evidence that the read took a lock inside the grant's own
// transaction.
func TestIntegrationAGrantInFlightIsStoppedByTheShareLock(t *testing.T) {
	dbConn := openCloseTestDB(t, dbClosePrefix)
	ctx := context.Background()
	f := newDBGrantFixture(t, dbConn)

	campaignCode := openRunningCampaign(t, f, "in-flight")
	insertTestPlan(t, dbConn, dbClosePrefix+"premium-monthly")

	registeredAt := baseInstant()
	userID := newTestAccount(t, dbConn, registeredAt)
	payload := identitydto.UserCreatedEventPayload{UserID: userID, RegisteredAt: registeredAt}

	// The grant runs on its own goroutine and reports through this channel, so the
	// assertion below can tell "still waiting" from "finished".
	done := make(chan error, 1)
	blocked := false

	uow := shareddb.NewGormUnitOfWork(dbConn)
	err := uow.Execute(ctx, func(txCtx context.Context) error {
		// STEP 1 — the share lock, taken the way the grant path takes it.
		locked, err := f.campaign.GetByCodeForShare(txCtx, campaignCode)
		if err != nil {
			return err
		}
		if locked == nil {
			return errCloseLockReadEmpty
		}
		// STEP 2 — close it in this transaction, still uncommitted.
		if _, err := f.campaign.Close(txCtx, campaignCode, locked.Version); err != nil {
			return err
		}

		// STEP 3 — the grant, on the pool's OTHER connection.
		go func() { done <- f.grant.ProcessUserCreated(ctx, payload) }()
		select {
		case grantErr := <-done:
			t.Errorf("FR-045: the grant completed while the close was still pending (err=%v). "+
				"A close that a grant can overtake leaves one more plan handed out after the "+
				"operator stopped the campaign", grantErr)
		case <-time.After(500 * time.Millisecond):
			blocked = true
		}
		// Returning commits, which is what releases the lock and lets the grant finish.
		return nil
	})
	if err != nil {
		t.Fatalf("the closing transaction failed: %v", err)
	}
	if !blocked {
		t.Fatal("the grant must WAIT for the share lock (FR-045); it did not")
	}

	// STEP 4 — the grant resumes and re-reads the row the close committed.
	select {
	case grantErr := <-done:
		if grantErr != nil {
			t.Fatalf("a grant stopped by a close is a SKIP, not a retryable failure: %v", grantErr)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the grant never finished after the close committed")
	}

	if _, found, err := f.claims.GetByUser(ctx, userID); err != nil {
		t.Fatalf("read the claim ledger: %v", err)
	} else if found {
		t.Errorf("FR-045: the campaign was closed before the claim was written, so no claim may exist")
	}
	reason, ok := f.log.skipReasonFor(userID)
	if !ok {
		t.Fatal("the skip must be logged with a reason code (FR-061)")
	}
	if reason != SkipCampaignSleeping {
		t.Errorf("reason = %q, want %q — a closed campaign is asleep, not merely unconfigured",
			reason, SkipCampaignSleeping)
	}
}

// TestIntegrationClosingARunningCampaignStopsItFromTheCandidateSet is the OTHER half
// of FR-045, and it is what makes the sweep path stop as well.
//
// `ListOpen` filters `closed_at IS NULL`, so a closed campaign is not a candidate on
// ANY path — the message path, the catch-up sweep and the operator look-up all draw
// from that one predicate. A close that only guarded the grant transaction would leave
// the sweep walking a campaign nobody wanted any more.
func TestIntegrationClosingARunningCampaignStopsItFromTheCandidateSet(t *testing.T) {
	dbConn := openCloseTestDB(t, dbClosePrefix)
	ctx := context.Background()
	f := newDBGrantFixture(t, dbConn)

	campaignCode := openRunningCampaign(t, f, "candidate-set")
	open, err := f.campaign.ListOpen(ctx)
	if err != nil {
		t.Fatalf("ListOpen before the close: %v", err)
	}
	if !containsDBCode(open, campaignCode) {
		t.Fatalf("precondition: %s must be a candidate before it is closed", campaignCode)
	}

	closeCampaign(t, f, campaignCode)

	open, err = f.campaign.ListOpen(ctx)
	if err != nil {
		t.Fatalf("ListOpen after the close: %v", err)
	}
	if containsDBCode(open, campaignCode) {
		t.Errorf("FR-070: a closed campaign must not be a candidate for any grant path")
	}

	// `ListAll` is the OPERATOR list and must still show it — FR-032 adds
	// `status = closed` precisely so an operator can see that a campaign was closed
	// and when. A list that hid closed campaigns would make that status unreachable.
	all, err := f.campaign.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if !containsDBCode(all, campaignCode) {
		t.Errorf("FR-032: the admin list must still show the closed campaign %s", campaignCode)
	}
}

// containsDBCode is the membership check both listings share.
func containsDBCode(rows []*entities.Campaign, code string) bool {
	for _, row := range rows {
		if row != nil && row.Code == code {
			return true
		}
	}
	return false
}

// errCloseLockReadEmpty is a sentinel for the one failure inside the transaction that
// is an assertion rather than a database error, so the assertion message is not
// swallowed into a wrapped error.
var errCloseLockReadEmpty = errors.New("campaign close integration: locked read returned no row")
