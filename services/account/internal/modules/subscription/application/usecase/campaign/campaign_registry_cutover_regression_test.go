package campaign

import (
	"testing"
	"time"

	identitydto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

// 026 T024 (FR-042 / FR-043): conditions 1-9 of `data-model.md` §6 keep their
// 025 behaviour, and this file is the machine-readable statement of that.
//
// WHY A FILE WHEN THE 025 SUITE ALREADY EXISTS. `campaign_admission_test.go`,
// `campaign_quota_test.go`, `campaign_grant_test.go`, `campaign_multi_campaign_test.go`
// and `campaign_sweep_test.go` are still here, still pass, and now build
// `[]*entities.Campaign` instead of `[]sharedconfig.CampaignConfig` — so they ARE
// the regression run. What they do not do is say WHICH properties were at risk, and
// a reader cannot tell which of those assertions cover the move and which merely
// happen to pass. This file names the nine conditions and the four budget
// invariants, states for each what would have broken it, and asserts them through
// the registry-sourced path.
//
// The properties that a source-of-truth change can plausibly break are all here:
// duplicate prevention (condition 9), `grant_bucket` as the frozen enum, the
// `fills_main_quota` watermark, the priority rule built on that watermark, and the
// `quota + reserve` ceiling. A registry bug would corrupt counts; a config-shaped
// bug would grant twice.

// registrySignups is the fixture: `count` accounts, ordered so that the ones
// registered EARLIEST arrive LAST.
//
// The order is not decoration. The reserve exists for accounts created BEFORE the
// watermark (FR-031), so the accounts that fill the main budget have to be the
// later-registered ones, and only then does the earliest-registered account qualify
// for the reserve. Deliver them in registration order instead and the reserve is
// correctly refused for everybody — a test would then be measuring the wrong
// property while looking like it measured the right one.
func registrySignups(count int, startsAt time.Time) []identitydto.UserCreatedEventPayload {
	out := make([]identitydto.UserCreatedEventPayload, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, identitydto.UserCreatedEventPayload{
			UserID:       uuid.New(),
			RegisteredAt: startsAt.Add(time.Duration(count-i) * time.Minute),
		})
	}
	return out
}

// TestRegressionCondition9RefusesASecondGrantForOneAccount is FR-043's duplicate
// guard, driven from the registry.
//
// `UNIQUE (user_id)` was always the mechanism; what changed is WHERE the campaign
// row comes from. A registry that returned a copy per read could in principle let
// two paths compute two different budgets for the same account, and the guard would
// then be the only thing standing between a customer and two promotional terms.
func TestRegressionCondition9RefusesASecondGrantForOneAccount(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, []*entities.Campaign{
		testCampaign("launch-2026-10", "premium-monthly", 10, 2, &startsAt, nil),
	}, claims)

	payload := registrySignups(1, startsAt)[0]
	if !h.granted(t, payload) {
		t.Fatal("precondition: the first delivery must grant")
	}

	before := claims.total()
	beforeEvents := len(h.events.published())

	// The same message redelivered, and a SECOND account on the same campaign: the
	// first must not be granted twice and the second must be.
	if h.granted(t, payload) {
		t.Error("FR-025/FR-043: a redelivered message must not produce a second grant")
	}
	if claims.total() != before {
		t.Errorf("claims = %d, want %d — condition 9 must refuse the duplicate claim", claims.total(), before)
	}
	if len(h.events.published()) != beforeEvents {
		t.Error("a refused duplicate must not publish a second plan change")
	}

	other := registrySignups(1, startsAt)[0]
	if !h.granted(t, other) {
		t.Error("a DIFFERENT account on the same campaign must still be granted — the guard is " +
			"per account, not per campaign")
	}
	if got := claims.countBucket("launch-2026-10", entities.GrantBucketMain); got != 2 {
		t.Errorf("main claims = %d, want 2", got)
	}
}

// TestRegressionGrantBucketIsWrittenAtGrantTime is research.md R2b, unchanged by
// the move and the reason the admin numbers stay correct.
//
// The failure it guards is specific: deriving the bucket from the row's position, or
// from `NOT fills_main_quota`, would tie the meaning of HISTORICAL rows to the
// registry's current `quota` — so raising quota on a live campaign would silently
// reclassify earlier claims and move the watermark.
func TestRegressionGrantBucketIsWrittenAtGrantTime(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, []*entities.Campaign{
		// The campaign 025 ran: quota 2, reserve 1.
		testCampaign("launch-2026-10", "premium-monthly", 2, 1, &startsAt, nil),
	}, claims)

	payloads := registrySignups(3, startsAt)
	granted := 0
	for _, payload := range payloads {
		if h.granted(t, payload) {
			granted++
		}
	}

	if granted != 3 {
		t.Fatalf("granted %d of 3, want 3 (quota 2 + reserve 1)", granted)
	}
	if got := claims.countBucket("launch-2026-10", entities.GrantBucketMain); got != 2 {
		t.Errorf("main claims = %d, want 2", got)
	}
	if got := claims.countBucket("launch-2026-10", entities.GrantBucketReserve); got != 1 {
		t.Errorf("reserve claims = %d, want 1", got)
	}
	// The watermark is the claim that FILLED the main budget, and exactly one row
	// carries it.
	if got := claims.watermarkCount("launch-2026-10"); got != 1 {
		t.Errorf("watermark rows = %d, want exactly 1 (026 FR-043: fills_main_quota is the FR-030 "+
			"watermark, and a second one would break ux_campaign_claims_watermark)", got)
	}
}

// TestRegressionWatermarkDecidesTheReserve is FR-030/FR-031's priority rule.
//
// The reserve is for accounts created BEFORE the watermark. After the move the
// campaign's `quota` is a database column read at run time, so the boundary the
// comparison is measured against and the budget it is measured against come from
// two different reads — and if they disagreed, either a customer would be refused
// who qualified, or one would be granted who did not.
func TestRegressionWatermarkDecidesTheReserve(t *testing.T) {
	// The campaign opens three hours ago, so an account registered 30 minutes
	// before the first delivery is still inside the window: an out-of-window refusal
	// here would masquerade as a reserve refusal.
	startsAt := baseInstant().Add(-3 * time.Hour)
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, []*entities.Campaign{
		testCampaign("launch-2026-10", "premium-monthly", 2, 1, &startsAt, nil),
	}, claims)

	first := registrySignups(1, startsAt)[0]
	second := identitydto.UserCreatedEventPayload{
		UserID:       uuid.New(),
		RegisteredAt: first.RegisteredAt.Add(time.Minute),
	}
	third := identitydto.UserCreatedEventPayload{
		UserID:       uuid.New(),
		RegisteredAt: second.RegisteredAt.Add(time.Minute),
	}

	h.granted(t, first)
	h.granted(t, second)

	// `third` registered AFTER the watermark, so the reserve does not belong to it.
	if h.granted(t, third) {
		t.Error("FR-031: the reserve is for accounts created BEFORE the frozen watermark")
	}
	if got := claims.countBucket("launch-2026-10", entities.GrantBucketReserve); got != 0 {
		t.Errorf("reserve claims = %d, want 0", got)
	}

	// An account created before the watermark DOES get the reserve. Its message has
	// to arrive late, which is exactly the case the mechanism exists for.
	late := identitydto.UserCreatedEventPayload{
		UserID:       uuid.New(),
		RegisteredAt: first.RegisteredAt.Add(-time.Minute),
	}
	if !h.granted(t, late) {
		t.Errorf("an account created before the watermark must draw from the reserve; last skip: %v",
			h.logger.skipsFor(late.UserID))
	}
	if got := claims.countBucket("launch-2026-10", entities.GrantBucketReserve); got != 1 {
		t.Errorf("reserve claims = %d, want 1", got)
	}
}

// TestRegressionTheHardCapIsNeverExceeded is FR-022's ceiling and the one number a
// campaign may not cross.
//
// Each budget part is bounded by its OWN counter rather than by a comparison against
// `quota + reserve`, so the total ceiling holds by construction. A registry read
// that returned a stale or duplicated campaign row is exactly the kind of change
// that could reintroduce a total comparison against a number nobody re-read.
func TestRegressionTheHardCapIsNeverExceeded(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)
	const quota, reserve = 2, 1

	claims := newFakeClaimRepo()
	h := newTestUseCase(t, []*entities.Campaign{
		testCampaign("launch-2026-10", "premium-monthly", quota, reserve, &startsAt, nil),
	}, claims)

	granted := 0
	payloads := registrySignups(6, startsAt)
	for _, payload := range payloads {
		if h.granted(t, payload) {
			granted++
		}
	}

	if granted != quota+reserve {
		t.Errorf("granted %d, want exactly %d — the reserve is a CEILING, not a target, and the "+
			"total must never exceed quota+reserve", granted, quota+reserve)
	}
	if got := claims.total(); got != quota+reserve {
		t.Errorf("claims written = %d, want %d", got, quota+reserve)
	}
	// Every refusal past the ceiling is NAMED, so an operator watching the queue can
	// tell "we ran out" from "something is broken" (FR-061).
	refused := 0
	for _, payload := range payloads {
		if claims.claimFor(payload.UserID) == nil {
			refused++
		}
	}
	if refused != len(payloads)-(quota+reserve) {
		t.Errorf("%d accounts refused, want %d — every account past the ceiling must be skipped "+
			"with a code, not dropped", refused, len(payloads)-(quota+reserve))
	}
	if reasons := h.logger.skipsFor(payloads[len(payloads)-1].UserID); len(reasons) == 0 {
		t.Error("the refused account must leave a skip reason behind (FR-061): an operator reading " +
			"a quiet log would conclude the mechanism is broken rather than full")
	}
}

// TestRegressionAnAccountAlreadyOnTheCampaignPlanIsRefused is condition 5 (FR-018),
// which depends on the CHOSEN campaign's plan and therefore on the selection still
// returning a real registry row.
func TestRegressionAnAccountAlreadyOnTheCampaignPlanIsRefused(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, []*entities.Campaign{
		testCampaign("launch-2026-10", "premium-monthly", 50, 5, &startsAt, nil),
	}, claims)

	payload := registrySignups(1, startsAt)[0]
	h.subs.put(&entities.UserSubscription{
		UserID:          payload.UserID,
		CurrentPlanCode: "premium-monthly",
		ExpiresAt:       timePtr(baseInstant().Add(10 * 24 * time.Hour)),
	})

	if h.granted(t, payload) {
		t.Error("FR-018: an account already on the campaign's plan with a live term must not be " +
			"granted another term")
	}
	reason, ok := h.lastSkip(t, payload.UserID)
	if !ok || reason != SkipAlreadyOnPlan {
		t.Errorf("reason = %q (found=%v), want %q", reason, ok, SkipAlreadyOnPlan)
	}
}

// TestRegressionAMissingPlanSkipsWithPlanUnavailable is FR-014's runtime rule, and
// the one 026 kept UNCHANGED on purpose.
//
// The registry is written without checking the catalogue, because loading
// configuration must not reach into the database and a paused plan must not make a
// campaign un-creatable. The consequence is that a wrong `plan_slug` is only visible
// per grant, and the skip code is how an operator discovers it.
func TestRegressionAMissingPlanSkipsWithPlanUnavailable(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, []*entities.Campaign{
		testCampaign("launch-2026-10", "khong-ton-tai-goi", 50, 5, &startsAt, nil),
	}, claims)

	payload := registrySignups(1, startsAt)[0]
	if h.granted(t, payload) {
		t.Fatal("a campaign whose plan does not exist must grant nothing")
	}
	reason, ok := h.lastSkip(t, payload.UserID)
	if !ok || reason != SkipPlanUnavailable {
		t.Errorf("reason = %q (found=%v), want %q — this code is how an operator finds a wrong plan_slug",
			reason, ok, SkipPlanUnavailable)
	}
	if claims.total() != 0 {
		t.Errorf("claims written = %d, want 0: a grant must not be written for a plan that "+
			"does not exist, or the catch-up sweep would keep re-deciding it", claims.total())
	}
}

// TestRegressionTheCatchupSweepAndTheQueueReachTheSameAnswer is FR-080, and the
// property most at risk from a source-of-truth change: both paths must READ THE SAME
// registry through the SAME evaluator.
//
// If the sweep walked a list the queue did not, one account could be granted by one
// and refused by the other, with every individual log line looking reasonable.
func TestRegressionTheCatchupSweepAndTheQueueReachTheSameAnswer(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)
	campaign := testCampaign("launch-2026-10", "premium-monthly", 5, 1, &startsAt, nil)

	account := sweepAccount(baseInstant().Add(-10*time.Minute), false)
	harness, _, sweep := newTestSweep(t, []*entities.Campaign{campaign}, []*identitydto.UserBriefContractDTO{account})
	claims := harness.claims

	if _, err := sweep.Sweep(t.Context()); err != nil {
		t.Fatalf("sweep: %v", err)
	}

	// The sweep re-derives the answer by calling the shared entry point, so the
	// evidence is the ledger, not the sweep's own report.
	if got := claims.total(); got != 1 {
		t.Errorf("claims = %d, want 1 — the sweep must reach the SAME rules through the SAME "+
			"registry the message path reads (FR-080)", got)
	}
	if claims.claimFor(account.ID) == nil {
		t.Error("the swept account must hold the claim; a grant recorded only in the report would " +
			"be granted again on the next run")
	}
	// And the claim names the campaign the evaluator chose, not the one the sweep
	// happened to be iterating.
	if claim := claims.claimFor(account.ID); claim.CampaignCode != campaign.Code {
		t.Errorf("claim names %q, want %q", claim.CampaignCode, campaign.Code)
	}
}
