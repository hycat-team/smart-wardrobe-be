package campaign

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	identitydto "smart-wardrobe-be/services/account/dto/identity"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"

	"github.com/google/uuid"
)

// 026 T063 (US4, FR-013, FR-023) — moving the window against a REAL Postgres.
//
// WHAT ONLY A REAL DATABASE CAN SHOW HERE. The unit tests in `registry_uc_time_test.go`
// decide WHICH ANSWER the use case produces; this file shows that the answer is the one
// the OPERATOR observes, and it pins three facts a fake cannot:
//
//   - the moved `ends_at` is what the DERIVED STATUS is computed from, so a campaign an
//     operator pulled back into the past reports `expired` on the next read rather than
//     `running` (US4.2);
//   - the reason code an account then sees is `campaign_sleeping` — one of the twelve
//     already in production — and NOT a new one, because FR-050 forbids adding any;
//   - `chk_campaigns_window` still refuses a window that is illegal, so the application
//     layer's 422 is a friendly answer rather than the only line of defence.
//
// THE GRANTS ALREADY WRITTEN ARE THE POINT. Each claim snapshotted its own
// `registered_at` and its own budget share (FR-023), so pulling the window back must not
// touch them — and this file asserts that through the ledger rather than by not looking.
//
// THE HARNESS IS THE ONE THIS PACKAGE ALREADY USES (026 T038/T049): `TEST_DATABASE_URL`
// plus a `t.Skip`, and the schema from the EMBEDDED GOOSE MIGRATIONS rather than
// `AutoMigrate`, because a schema built from the Go structs would carry none of the CHECK
// constraints US4 leans on.

// dbTimePrefix namespaces every campaign this file writes, so a run against a shared
// test database cannot collide with a real one and cannot delete another suite's rows.
const dbTimePrefix = "verify-026-time-"

// postgresCheckViolation is SQLSTATE 23514: "a CHECK constraint said no".
//
// IT IS RESTATED HERE rather than imported, because `campaign_repo_impl.go` declares the
// same idea in the persistence package and this is a test: a test that reached into
// another package for a constant would couple the campaign use-case tests to a file they
// are not about.
const postgresCheckViolation = "23514"

// openRunningCampaignWithWindow creates a campaign through the registry's own use case,
// which is the only way a campaign is written in operation (FR-005, FR-062).
func openRunningCampaignWithWindow(t *testing.T, f *dbGrantFixture, suffix string, endsAt *time.Time) string {
	t.Helper()
	code := dbTimePrefix + suffix
	planSlug := dbTimePrefix + "premium-monthly"
	quota := 3
	var field *string
	if endsAt != nil {
		raw := endsAt.Format(time.RFC3339)
		field = &raw
	}
	if _, err := f.registry.Create(context.Background(), uuid.New(), &sub_dto.CampaignCreateReq{
		Code:     code,
		PlanSlug: &planSlug,
		Quota:    &quota,
		StartsAt: baseInstant().Add(-24 * time.Hour).Format(time.RFC3339),
		EndsAt:   field,
	}, nil); err != nil {
		t.Fatalf("create the campaign: %v", err)
	}
	// THE PRECONDITION (026 T081). This file's whole subject is WHICH campaign an
	// account lands in after the window moves, so a foreign row in the registry is
	// fatal to the claim rather than incidental: the seeded `launch-2026-10` starts at
	// exactly `baseInstant()` and FR-040 gives it the selection.
	requireRegistryIsolatedTo(t, f.campaign, dbTimePrefix, code)
	return code
}

// moveWindowEnd edits `ends_at` through the registry use case, reading the version back
// first because FR-055 compares against the version the client held.
//
// The read is not a way around the conditional UPDATE — a wrong version would still
// produce a 412. It is here because this test is the CLIENT, and a client sends the
// version it read.
//
// THE BODY IS BUILT AS JSON AND DECODED, rather than as a Go request literal, so the
// request that reaches the use case is one a client could actually send — including the
// `null` that clears the window end (T061).
func moveWindowEnd(t *testing.T, f *dbGrantFixture, campaignCode string, endsAt *time.Time) *sub_dto.CampaignSummaryRes {
	t.Helper()
	ctx := context.Background()

	current, err := f.campaign.GetByCode(ctx, campaignCode)
	if err != nil || current == nil {
		t.Fatalf("read the campaign before editing it: err=%v row=%v", err, current)
	}
	body := `{"version":` + strconv.Itoa(current.Version) + `,"endsAt":`
	if endsAt == nil {
		body += "null"
	} else {
		body += `"` + endsAt.Format(time.RFC3339) + `"`
	}
	body += `}`

	summary, err := f.registry.Update(ctx, uuid.New(), campaignCode, decodeUpdateReq(t, body), nil)
	if err != nil {
		t.Fatalf("move the window end of %s: %v", campaignCode, err)
	}
	return summary
}

// TestIntegrationPullingTheWindowIntoThePastExpiresTheCampaign is US4 acceptance
// scenario 2 and the first half of US4's Independent Test.
//
// THREE ASSERTIONS, IN THE ORDER THEY FAIL. The stored row, then the derived status the
// operator's screen shows, then the grant — because each one narrows where a failure can
// be: a wrong row is a persistence defect, a wrong status over a right row is a state
// machine defect, and a grant after `expired` is the defect the whole story exists to
// prevent.
func TestIntegrationPullingTheWindowIntoThePastExpiresTheCampaign(t *testing.T) {
	dbConn := openCloseTestDB(t, dbTimePrefix)
	ctx := context.Background()
	f := newDBGrantFixture(t, dbConn)

	campaignCode := openRunningCampaignWithWindow(t, f, "pull-back", nil)
	insertTestPlan(t, dbConn, dbTimePrefix+"premium-monthly")

	// BASELINE — the campaign is running and grants, so every later assertion is about
	// the MOVE and not about a fixture that never worked.
	first := newTestAccount(t, dbConn, baseInstant())
	if err := f.grant.ProcessUserCreated(ctx, identitydto.UserCreatedEventPayload{
		UserID:       first,
		RegisteredAt: baseInstant(),
	}); err != nil {
		t.Fatalf("process the signup message: %v", err)
	}
	if _, found, err := f.claims.GetByUser(ctx, first); err != nil {
		t.Fatalf("read the claim ledger: %v", err)
	} else if !found {
		t.Fatalf("precondition: the open campaign must grant, and %s holds no claim", first)
	}

	// THE MOVE — `ends_at` lands an hour ago, which is AFTER `starts_at`, so the merged
	// window is legal and FR-013 has nothing to say about it.
	past := baseInstant().Add(-time.Hour)
	summary := moveWindowEnd(t, f, campaignCode, &past)

	// 1. THE STORED ROW. Read back through the repository the operator's screen uses, so
	// a discarded write cannot pass this test.
	stored, err := f.campaign.GetByCode(ctx, campaignCode)
	if err != nil || stored == nil {
		t.Fatalf("the campaign row must be readable: err=%v row=%v", err, stored)
	}
	if stored.EndsAt == nil || !stored.EndsAt.Equal(past) {
		t.Errorf("ends_at = %v, want the requested %s", stored.EndsAt, past)
	}
	if stored.StartsAt.IsZero() {
		t.Error("starts_at must survive the edit")
	}
	// The edit bumped the version, so the next edit from the same screen is not stale.
	if stored.Version < 2 {
		t.Errorf("FR-055: version = %d, want it incremented by the window edit", stored.Version)
	}

	// 2. THE DERIVED STATUS. `expired` rather than `running`, and this is US4.2 verbatim.
	if got := deriveCampaignStatus(stored, 1, 1, baseInstant()); got != sub_dto.CampaignStatusExpired {
		t.Errorf("status = %q, want %q — quickstart US4 step 2 observes exactly this", got,
			sub_dto.CampaignStatusExpired)
	}
	if summary.Status != sub_dto.CampaignStatusExpired {
		t.Errorf("the edit response reports status %q, want %q", summary.Status,
			sub_dto.CampaignStatusExpired)
	}

	// 3. THE GRANT. A new account inside the OLD window now gets nothing.
	second := newTestAccount(t, dbConn, baseInstant())
	if err := f.grant.ProcessUserCreated(ctx, identitydto.UserCreatedEventPayload{
		UserID:       second,
		RegisteredAt: baseInstant(),
	}); err != nil {
		t.Fatalf("a skip is a success, not a retryable failure: %v", err)
	}
	if _, found, err := f.claims.GetByUser(ctx, second); err != nil {
		t.Fatalf("read the claim ledger: %v", err)
	} else if found {
		t.Errorf("US4.2: a campaign whose ends_at moved into the past must grant nothing, "+
			"but %s holds a claim", second)
	}
	reason, ok := f.log.skipReasonFor(second)
	if !ok {
		t.Fatal("the skip must be logged with a reason code (FR-061)")
	}
	if reason != SkipCampaignSleeping {
		t.Errorf("reason = %q, want %q — FR-050 keeps exactly twelve reason codes and "+
			"`campaign_sleeping` is the one 025 already used for a finished campaign", reason,
			SkipCampaignSleeping)
	}

	// AND THE GRANT ALREADY WRITTEN IS UNTOUCHED. FR-023's promise is that moving the
	// window does not break the claims that exist, and the ledger is where that is true.
	if claim, found, err := f.claims.GetByUser(ctx, first); err != nil {
		t.Fatalf("re-read the first claim: %v", err)
	} else if !found || claim.CampaignCode != campaignCode {
		t.Errorf("FR-023: moving ends_at must not remove or re-point an existing claim, got %+v",
			claim)
	}
	// The budget is frozen because the campaign HAS granted, whatever its window now says
	// — the condition is "has granted", not "is running" (FR-020, spec.md Câu 3).
	if granted, err := f.campaign.ExistsClaims(ctx, campaignCode); err != nil {
		t.Fatalf("ExistsClaims: %v", err)
	} else if !granted {
		t.Error("FR-020: the campaign must still report that it has granted")
	}
}

// TestIntegrationPushingTheWindowForwardGrantsAgain is US4 acceptance scenario 1 and the
// second half of US4's Independent Test: the same kind of campaign, opened back up.
//
// IT STARTS FROM A CAMPAIGN THAT HAS GRANTED in production terms only by accident — the
// test does not grant first — and that is deliberate. The claim US4 makes is about the
// WINDOW being editable, and a campaign nobody was granted from would prove nothing about
// FR-023, which is precisely that the window stays editable once it HAS granted.
func TestIntegrationPushingTheWindowForwardGrantsAgain(t *testing.T) {
	dbConn := openCloseTestDB(t, dbTimePrefix)
	ctx := context.Background()
	f := newDBGrantFixture(t, dbConn)

	// Opened already expired: `ends_at` an hour ago, `starts_at` a day ago.
	past := baseInstant().Add(-time.Hour)
	campaignCode := openRunningCampaignWithWindow(t, f, "push-forward", &past)
	insertTestPlan(t, dbConn, dbTimePrefix+"premium-monthly")

	// While expired, a new account gets nothing — the precondition for the move.
	blocked := newTestAccount(t, dbConn, baseInstant())
	if err := f.grant.ProcessUserCreated(ctx, identitydto.UserCreatedEventPayload{
		UserID:       blocked,
		RegisteredAt: baseInstant(),
	}); err != nil {
		t.Fatalf("a skip is a success, not a retryable failure: %v", err)
	}
	if _, found, err := f.claims.GetByUser(ctx, blocked); err != nil {
		t.Fatalf("read the claim ledger: %v", err)
	} else if found {
		t.Fatalf("precondition: an expired campaign must grant nothing, and %s holds a claim", blocked)
	}

	// THE MOVE — a month ahead, which is after `starts_at` and after the clock.
	future := baseInstant().AddDate(0, 1, 0)
	summary := moveWindowEnd(t, f, campaignCode, &future)
	if summary.Status != sub_dto.CampaignStatusRunning {
		t.Errorf("status = %q, want %q once the window covers the clock again", summary.Status,
			sub_dto.CampaignStatusRunning)
	}

	// A SECOND account, created inside the new window, is granted.
	granted := newTestAccount(t, dbConn, baseInstant())
	if err := f.grant.ProcessUserCreated(ctx, identitydto.UserCreatedEventPayload{
		UserID:       granted,
		RegisteredAt: baseInstant(),
	}); err != nil {
		t.Fatalf("process the signup message: %v", err)
	}
	claim, found, err := f.claims.GetByUser(ctx, granted)
	if err != nil {
		t.Fatalf("read the claim ledger: %v", err)
	}
	if !found || claim == nil {
		t.Fatalf("US4.1: a campaign whose ends_at moved into the future must grant again; "+
			"%s holds no claim", granted)
	}
	if claim.CampaignCode != campaignCode {
		t.Errorf("the claim names campaign %q, want %q", claim.CampaignCode, campaignCode)
	}

	// AND THE ACCOUNT REFUSED EARLIER IS STILL REFUSED. `UNIQUE (user_id)` plus the
	// window the claim would have had to satisfy means reopening the window cannot be
	// used to hand out the plan somebody was legitimately refused earlier.
	if _, stillThere, err := f.claims.GetByUser(ctx, blocked); err != nil {
		t.Fatalf("re-read the refused account: %v", err)
	} else if stillThere {
		t.Errorf("%s was granted while the campaign was expired; reopening the window must not "+
			"retroactively hand out the plan it was refused", blocked)
	}
}

// TestIntegrationTheWindowCheckRefusesAnIllegalWindowAtTheSchema is the BACKSTOP half of
// FR-013, and it is the only assertion in this file that bypasses the application on
// purpose.
//
// T062 proves the use case answers 422 before any I/O. This proves the column still
// refuses the write afterwards — which is the argument FR-021 makes for the budget and
// FR-062 repeats for the table: "no one edits this with direct SQL" is a rule that
// application discipline alone cannot bind, and a registry people reach for SQL over is
// exactly where that assumption fails.
//
// THE CAMPAIGN HAS GRANTED, on purpose, so the two sets of invariants can be told apart:
// the budget columns are already frozen by `trg_campaigns_immutable_budget`, while the
// window columns are still writable by SQL as long as the window is LEGAL.
func TestIntegrationTheWindowCheckRefusesAnIllegalWindowAtTheSchema(t *testing.T) {
	dbConn := openCloseTestDB(t, dbTimePrefix)
	ctx := context.Background()
	f := newDBGrantFixture(t, dbConn)

	campaignCode := openRunningCampaignWithWindow(t, f, "schema-check", nil)
	insertTestPlan(t, dbConn, dbTimePrefix+"premium-monthly")

	granted := newTestAccount(t, dbConn, baseInstant())
	if err := f.grant.ProcessUserCreated(ctx, identitydto.UserCreatedEventPayload{
		UserID:       granted,
		RegisteredAt: baseInstant(),
	}); err != nil {
		t.Fatalf("process the signup message: %v", err)
	}
	if has, err := f.campaign.ExistsClaims(ctx, campaignCode); err != nil || !has {
		t.Fatalf("precondition: the campaign must have granted: err=%v granted=%v", err, has)
	}

	// A window that closes BEFORE it opens — the shape `chk_campaigns_window` names.
	err := dbConn.Exec(
		`UPDATE subscription.campaigns SET ends_at = starts_at - INTERVAL '1 day' WHERE code = ?`,
		campaignCode).Error
	if err == nil {
		t.Fatal("FR-013: chk_campaigns_window must refuse a window that ends before it starts, " +
			"even from direct SQL")
	}
	assertCheckViolation(t, err, "chk_campaigns_window")

	// And the value is unchanged, which is the half that makes the constraint a
	// guarantee rather than an error message.
	stored, err := f.campaign.GetByCode(ctx, campaignCode)
	if err != nil || stored == nil {
		t.Fatalf("re-read the campaign: err=%v row=%v", err, stored)
	}
	if stored.EndsAt != nil {
		t.Errorf("the refused UPDATE must leave ends_at alone, got %v", stored.EndsAt)
	}
}

// assertCheckViolation checks that `err` is a PostgreSQL CHECK failure, by SQLSTATE.
//
// BY STATE AND NOT BY MESSAGE, because the message is the one thing a migration is free
// to reword; `23514` is the code that means "a CHECK constraint said no", and it is the
// same code `trg_campaigns_immutable_budget` raises for a budget edit (FR-021). Asserting
// on the text would make this test fail when somebody improves the wording, which is the
// opposite of what a test is for.
//
// THE STATE IS READ THROUGH AN INTERFACE rather than a `*pgconn.PgError`, for the reason
// `campaign_repo_impl.go` gives: naming the driver type would turn a transitive
// dependency of the GORM postgres driver into a direct one, for a type assertion.
func assertCheckViolation(t *testing.T, err error, constraint string) {
	t.Helper()
	type sqlStateCarrier interface{ SQLState() string }
	var carrier sqlStateCarrier
	if !errors.As(err, &carrier) {
		t.Fatalf("%s: expected a PostgreSQL error carrying a SQLSTATE, got %T: %v",
			constraint, err, err)
	}
	if state := carrier.SQLState(); state != postgresCheckViolation {
		t.Errorf("%s: expected SQLSTATE %s (check_violation), got %q", constraint,
			postgresCheckViolation, state)
	}
}
