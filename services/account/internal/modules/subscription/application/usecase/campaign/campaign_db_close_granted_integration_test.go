package campaign

import (
	"context"
	"testing"
	"time"

	identitydto "smart-wardrobe-be/services/account/dto/identity"
	sub_dto "smart-wardrobe-be/services/account/dto/subscription"

	"github.com/google/uuid"
)

// 026 T047 (FR-033, SC-002) — closing a campaign that has ALREADY GRANTED, against a
// real Postgres.
//
// FR-033 IS THE ONLY RULE IN 026 THAT SAYS "CHANGE NOTHING", and it is the one a
// customer would notice if it broke. Closing a campaign stops FUTURE grants; it must
// not touch a single account that was already given the plan, because that account's
// month is paid for and an operator stopping a promotion is not a decision about
// people who already got it.
//
// WHY NO FAKE CAN PROVE IT. `CampaignRegistryUseCase` holds no subscription repository
// and no claim repository, so nothing it is able to do can reach `user_subscriptions`
// — which is the strongest structural argument for the rule, and also why the rule
// needs a test that reads those rows rather than one that watches the use case. A
// regression that gave the writer a subscription repository, or a campaign close that
// revoked through the same subscription workflow the grant uses, would compile and
// pass every unit test in the package.
//
// THE HARNESS IS SHARED WITH `campaign_db_grant_integration_test.go` (T038) and
// `campaign_db_close_integration_test.go` (T049): `TEST_DATABASE_URL` plus a
// `t.Skip`, the schema from the embedded Goose migrations, and its own code prefix so
// a run against a shared test database deletes only its own rows.

// dbCloseGrantedPrefix namespaces this file's campaigns, kept distinct from the other
// close suite's prefix so the two cleanups cannot delete each other's rows.
const dbCloseGrantedPrefix = "verify-026-closed-granted-"

// TestIntegrationClosingACampaignThatGrantedLeavesGrantedAccountsUntouched is US2
// acceptance scenario 3 and FR-033.
//
// THE ASSERTION IS AN EQUALITY, NOT AN ABSENCE. "The account is still on the plan"
// would pass for a campaign that granted nothing, and for an account that was never
// granted anything at all. Both the plan code and the expiry are captured BEFORE the
// close and compared AFTER it, because either alone would leave a hole: a revocation
// that kept the plan code and dropped the term would satisfy the first and destroy the
// customer's month.
func TestIntegrationClosingACampaignThatGrantedLeavesGrantedAccountsUntouched(t *testing.T) {
	dbConn := openCloseTestDB(t, dbCloseGrantedPrefix)
	ctx := context.Background()
	f := newDBGrantFixture(t, dbConn)

	planSlug := dbCloseGrantedPrefix + "premium-monthly"
	insertTestPlan(t, dbConn, planSlug)

	// STEP 1 — a running campaign with room, and an account that qualifies for it.
	campaignCode := dbCloseGrantedPrefix + "granted"
	quota := 3
	planField := planSlug
	if _, err := f.registry.Create(ctx, uuid.New(), &sub_dto.CampaignCreateReq{
		Code:     campaignCode,
		PlanSlug: &planField,
		Quota:    &quota,
		StartsAt: baseInstant().Add(-24 * time.Hour).Format(time.RFC3339),
	}, nil); err != nil {
		t.Fatalf("create the campaign: %v", err)
	}

	// THE PRECONDITION (026 T081). The "the account holds a claim" precondition below
	// is only a real one if the claim came from THIS campaign: the seeded
	// `launch-2026-10` starts at exactly `baseInstant()` and would otherwise win
	// FR-040, and the claim would name the seed rather than the fixture.
	requireRegistryIsolatedTo(t, f.campaign, dbCloseGrantedPrefix, campaignCode)

	registeredAt := baseInstant()
	grantedUserID := newTestAccount(t, dbConn, registeredAt)
	if err := f.grant.ProcessUserCreated(ctx, identitydto.UserCreatedEventPayload{
		UserID:       grantedUserID,
		RegisteredAt: registeredAt,
	}); err != nil {
		t.Fatalf("process the signup message: %v", err)
	}

	// PRECONDITION, and it is a real one: without a grant there is nothing for FR-033
	// to protect, and the test would pass for the wrong reason.
	claim, found, err := f.claims.GetByUser(ctx, grantedUserID)
	if err != nil || !found || claim == nil {
		t.Fatalf("precondition: the account must hold a claim before the close (err=%v found=%v)", err, found)
	}

	before, err := f.subs.GetByUserID(ctx, grantedUserID)
	if err != nil || before == nil {
		t.Fatalf("read the subscription before the close: err=%v sub=%v", err, before)
	}
	if before.CurrentPlanCode != planSlug {
		t.Fatalf("precondition: current_plan_code = %q, want %q", before.CurrentPlanCode, planSlug)
	}
	if before.ExpiresAt == nil {
		t.Fatal("precondition: the granted subscription must carry an expiry, or there is nothing to protect")
	}
	planCodeBefore := before.CurrentPlanCode
	expiresAtBefore := *before.ExpiresAt

	// STEP 2 — the operator closes it.
	closeCampaign(t, f, campaignCode)

	// STEP 3 — nothing about that account moved.
	after, err := f.subs.GetByUserID(ctx, grantedUserID)
	if err != nil || after == nil {
		t.Fatalf("read the subscription after the close: err=%v sub=%v", err, after)
	}
	if after.CurrentPlanCode != planCodeBefore {
		t.Errorf("FR-033: current_plan_code changed from %q to %q — closing a campaign must not revoke a grant",
			planCodeBefore, after.CurrentPlanCode)
	}
	if after.ExpiresAt == nil {
		t.Fatalf("FR-033: the granted term was removed entirely (was %s); closing must not end it",
			expiresAtBefore)
	}
	if !after.ExpiresAt.Equal(expiresAtBefore) {
		t.Errorf("FR-033: expires_at moved from %s to %s — the customer's paid month must not be shortened",
			expiresAtBefore, after.ExpiresAt)
	}

	// The LEDGER is untouched too, and it matters for a second reason: a close that
	// rewrote the claim row would corrupt `grant_bucket` and the priority watermark
	// that FR-043 requires to keep 025's behaviour, which would show up weeks later as
	// wrong budget numbers rather than as anything this test could name.
	claimAfter, found, err := f.claims.GetByUser(ctx, grantedUserID)
	if err != nil || !found || claimAfter == nil {
		t.Fatalf("FR-033: the claim row must survive the close (err=%v found=%v)", err, found)
	}
	if claimAfter.CampaignCode != claim.CampaignCode ||
		claimAfter.GrantBucket != claim.GrantBucket ||
		!claimAfter.ExpiresAt.Equal(claim.ExpiresAt) {
		t.Errorf("FR-033/FR-043: the claim row changed: before %+v after %+v", claim, claimAfter)
	}

	// And the campaign is genuinely closed, so the test is not passing because the
	// close did nothing at all.
	stored, err := f.campaign.GetByCode(ctx, campaignCode)
	if err != nil || stored == nil {
		t.Fatalf("read the campaign after the close: err=%v row=%v", err, stored)
	}
	if stored.ClosedAt == nil {
		t.Error("FR-031: the campaign must be closed, or none of the assertions above mean anything")
	}
}

// TestIntegrationTheCloseReportsClosedWhileTheBudgetIsUntouched is the honest
// counterpart to the same rule, and it is a deliberate assertion about a LIMITATION
// rather than a wish.
//
// This endpoint's response is built by the registry writer, which holds no claim
// repository: `grantedMainCount` and friends read as zero there and are correct on the
// next `GET`. The state that matters — `status`, `closedAt`, the version — comes from
// the row and is exact, so the test pins THAT rather than pretending the whole summary
// is a fresh read. It is written down here because a reader of `Close`'s header will
// want to know the gap is known rather than overlooked.
//
// (The general shape is proved by the read-side assertions in
// `campaign_db_close_integration_test.go` and in `campaign_query_status_test.go`; what
// is specific here is that a closed campaign with grants behind it still reports
// `closed` and not one of the budget states.)
func TestIntegrationTheCloseReportsClosedWhileTheBudgetIsUntouched(t *testing.T) {
	dbConn := openCloseTestDB(t, dbCloseGrantedPrefix)
	ctx := context.Background()
	f := newDBGrantFixture(t, dbConn)

	planSlug := dbCloseGrantedPrefix + "reporting"
	insertTestPlan(t, dbConn, planSlug)

	campaignCode := dbCloseGrantedPrefix + "reporting"
	quota := 2
	planField := planSlug
	if _, err := f.registry.Create(ctx, uuid.New(), &sub_dto.CampaignCreateReq{
		Code:     campaignCode,
		PlanSlug: &planField,
		Quota:    &quota,
		StartsAt: baseInstant().Add(-24 * time.Hour).Format(time.RFC3339),
	}, nil); err != nil {
		t.Fatalf("create the campaign: %v", err)
	}

	// THE PRECONDITION (026 T081), and here it guards an assertion that reads as a
	// GRANT: with the seeded `launch-2026-10` still in the registry it would take the
	// selection, and this test's "the grant survives the close" would be about the
	// seed's grant rather than its own.
	requireRegistryIsolatedTo(t, f.campaign, dbCloseGrantedPrefix, campaignCode)

	registeredAt := baseInstant()
	userID := newTestAccount(t, dbConn, registeredAt)
	if err := f.grant.ProcessUserCreated(ctx, identitydto.UserCreatedEventPayload{
		UserID:       userID,
		RegisteredAt: registeredAt,
	}); err != nil {
		t.Fatalf("process the signup message: %v", err)
	}

	summary, err := f.registry.Close(ctx, uuid.New(), campaignCode, &sub_dto.CampaignCloseReq{
		Version: 1,
		Reason:  strPtr("Dừng chiến dịch (verify-026)"),
	}, nil)
	if err != nil {
		t.Fatalf("close the campaign: %v", err)
	}

	// FR-032: with grants behind it, the five-state machine would have answered
	// `running`. The closed column outranks it, which is the whole of FR-032.
	if summary.Status != sub_dto.CampaignStatusClosed {
		t.Errorf("status = %q, want %q even though the campaign has budget to spare",
			summary.Status, sub_dto.CampaignStatusClosed)
	}
	if summary.ClosedAt == nil {
		t.Error("the close response must carry closedAt")
	}

	// The row itself still carries the grant, so the budget numbers the READ path
	// derives are unchanged by a close — nothing here touches the ledger.
	granted, err := f.campaign.ExistsClaims(ctx, campaignCode)
	if err != nil {
		t.Fatalf("ExistsClaims: %v", err)
	}
	if !granted {
		t.Error("precondition: the campaign must still hold its grant after a close")
	}
	if _, found, err := f.claims.GetByUser(ctx, userID); err != nil {
		t.Fatalf("read the claim ledger: %v", err)
	} else if !found {
		t.Errorf("FR-033: the grant must survive the close")
	}
	// The claim's own expiry is the customer's term, and it is the same value the
	// subscription carries — the two must not drift apart.
	claim, _, err := f.claims.GetByUser(ctx, userID)
	if err != nil || claim == nil {
		t.Fatalf("read the claim: err=%v claim=%v", err, claim)
	}
	sub, err := f.subs.GetByUserID(ctx, userID)
	if err != nil || sub == nil || sub.ExpiresAt == nil {
		t.Fatalf("read the subscription: err=%v sub=%v", err, sub)
	}
	if !claim.ExpiresAt.Equal(*sub.ExpiresAt) {
		t.Errorf("the claim's term %s and the subscription's %s must describe the same grant",
			claim.ExpiresAt, *sub.ExpiresAt)
	}
}
