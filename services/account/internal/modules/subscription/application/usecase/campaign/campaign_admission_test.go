package campaign

import (
	"testing"
	"time"

	identitydto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	sharedconfig "smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
)

// 025 T039-T040: the first seven conditions of `data-model.md` §6 — the ones
// that need no lock.
//
// Splitting them from the quota tests is deliberate. FR-011, FR-017 and FR-091
// are about WHICH campaign an account belongs to and whether it is eligible at
// all; none of them is about budget. Testing them through the quota path would
// mean a test that fails could be failing for either reason, and the diagnosis
// would be guesswork.

// TestCampaignAdmissionPicksTheLatestStartingCampaign is SC-001's selection
// half: with two campaigns both live, the one that starts LATEST wins, and the
// order the messages arrive in is irrelevant.
//
// The messages are delivered out of order deliberately, because that is the case
// the rule exists for (FR-011): an account that registers just after the second
// campaign opens must land in the second one, and a message that took five
// minutes in the queue must not push it back into the first.
func TestCampaignAdmissionPicksTheLatestStartingCampaign(t *testing.T) {
	earlyStart := baseInstant().Add(-2 * time.Hour)
	lateStart := baseInstant().Add(-1 * time.Hour)

	campaigns := []sharedconfig.CampaignConfig{
		testCampaignConfig("first-campaign", "premium-monthly", 50, 5, &earlyStart, nil),
		testCampaignConfig("second-campaign", "premium-quarterly", 50, 5, &lateStart, nil),
	}
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, campaigns, claims)

	// The account was created inside BOTH windows: after the first opened and
	// after the second opened.
	registeredAt := baseInstant().Add(-30 * time.Minute)
	payload := identitydto.UserCreatedEventPayload{UserID: uuid.New(), RegisteredAt: registeredAt}

	if err := h.uc.ProcessUserCreated(t.Context(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	claim := claims.claimFor(payload.UserID)
	if claim == nil {
		t.Fatalf("expected a claim to be written")
	}
	if claim.CampaignCode != "second-campaign" {
		t.Fatalf("expected the latest-starting campaign to win, got %q", claim.CampaignCode)
	}
	if claim.PlanSlug != "premium-quarterly" {
		t.Fatalf("expected the winning campaign's plan, got %q", claim.PlanSlug)
	}

	// The losing campaign's counters must be untouched: selecting a campaign is
	// not the same as spending from it.
	if got := claims.countBucket("first-campaign", entities.GrantBucketMain); got != 0 {
		t.Fatalf("expected 0 claims on the losing campaign, got %d", got)
	}
}

// TestCampaignAdmissionIgnoresMessageArrivalOrder is the same rule stated as an
// invariant: the account's campaign is a function of WHEN THE ACCOUNT WAS
// CREATED, never of when the message was processed.
//
// `data-model.md` §6.1 uses `account.created_at` on both sides of the window test
// for exactly this reason. Had it used "now", a message delayed past the end of
// the campaign would fall out of the window and the account would be discarded for
// no reason at all.
func TestCampaignAdmissionIgnoresMessageArrivalOrder(t *testing.T) {
	startsAt := baseInstant().Add(-24 * time.Hour)
	endsAt := baseInstant().Add(24 * time.Hour)
	campaigns := []sharedconfig.CampaignConfig{
		testCampaignConfig("launch-2026-10", "premium-monthly", 50, 5, &startsAt, &endsAt),
	}
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, campaigns, claims)

	registeredAt := baseInstant().Add(-time.Hour)
	first := identitydto.UserCreatedEventPayload{UserID: uuid.New(), RegisteredAt: registeredAt}
	second := identitydto.UserCreatedEventPayload{UserID: uuid.New(), RegisteredAt: registeredAt}

	// The account that arrives first is NOT the one registered first; the rule
	// does not care.
	if err := h.uc.ProcessUserCreated(t.Context(), second); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.uc.ProcessUserCreated(t.Context(), first); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, payload := range []identitydto.UserCreatedEventPayload{first, second} {
		claim := claims.claimFor(payload.UserID)
		if claim == nil {
			t.Fatalf("expected both accounts to be granted regardless of arrival order")
		}
		if claim.CampaignCode != "launch-2026-10" {
			t.Fatalf("expected campaign launch-2026-10, got %q", claim.CampaignCode)
		}
	}
}

// TestCampaignAdmissionRejectsAccountOutsideTheWindow is FR-017: an account
// created before the campaign opens, or after it closes, is not eligible — and the
// test uses the message's own `registered_at`, not the processing clock.
func TestCampaignAdmissionRejectsAccountOutsideTheWindow(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)
	endsAt := baseInstant().Add(time.Hour)
	campaigns := []sharedconfig.CampaignConfig{
		testCampaignConfig("launch-2026-10", "premium-monthly", 50, 5, &startsAt, &endsAt),
	}
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, campaigns, claims)

	tooEarly := identitydto.UserCreatedEventPayload{
		UserID:       uuid.New(),
		RegisteredAt: startsAt.Add(-time.Second),
	}
	if h.uc.ProcessUserCreated(t.Context(), tooEarly) != nil {
		t.Fatalf("unexpected error: account before the window is a skip, not an error")
	}
	if claims.claimFor(tooEarly.UserID) != nil {
		t.Fatalf("an account created before the campaign opens must not be granted (FR-017)")
	}

	tooLate := identitydto.UserCreatedEventPayload{
		UserID:       uuid.New(),
		RegisteredAt: endsAt.Add(time.Second),
	}
	if h.uc.ProcessUserCreated(t.Context(), tooLate) != nil {
		t.Fatalf("unexpected error: account after the window is a skip, not an error")
	}
	if claims.claimFor(tooLate.UserID) != nil {
		t.Fatalf("an account created after the campaign ends must not be granted (FR-017)")
	}
}

// TestCampaignAdmissionSkipsAdministratorCreatedAccount is T040 / FR-091: an
// account created through the administrator endpoint is skipped, and the reason
// is its OWN code.
//
// Not merely "not eligible". An administrator's own test accounts are perfectly
// valid accounts, and a campaign wall of `not_eligible` would send an operator
// hunting for a configuration error that does not exist — while the real problem,
// the test accounts eating a launch campaign's 50 slots, stays invisible.
func TestCampaignAdmissionSkipsAdministratorCreatedAccount(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)
	campaigns := []sharedconfig.CampaignConfig{
		testCampaignConfig("launch-2026-10", "premium-monthly", 50, 5, &startsAt, nil),
	}
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, campaigns, claims)

	payload := identitydto.UserCreatedEventPayload{
		UserID:         uuid.New(),
		RegisteredAt:   baseInstant(),
		CreatedByAdmin: true,
	}
	if err := h.uc.ProcessUserCreated(t.Context(), payload); err != nil {
		t.Fatalf("unexpected error: an administrator account is a skip, not an error")
	}
	if claims.claimFor(payload.UserID) != nil {
		t.Fatalf("an administrator-created account must never consume campaign quota (FR-091)")
	}
	reason, ok := h.lastSkip(t, payload.UserID)
	if !ok || reason != SkipCreatedByAdmin {
		t.Fatalf("expected skip reason %q, got %q (found=%v)", SkipCreatedByAdmin, reason, ok)
	}

	// The counter is untouched, so a customer's slot is still available.
	if got := claims.total(); got != 0 {
		t.Fatalf("expected no claim written, got %d", got)
	}
}

// TestCampaignAdmissionSkipsWhenNoCampaignIsConfigured proves condition 1, and
// that "the mechanism is switched off" is distinguishable from "the campaign ran
// out" — the two look identical from outside and mean opposite things to whoever
// has to act on it.
func TestCampaignAdmissionSkipsWhenNoCampaignIsConfigured(t *testing.T) {
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, nil, claims)

	payload := identitydto.UserCreatedEventPayload{UserID: uuid.New(), RegisteredAt: baseInstant()}
	if err := h.uc.ProcessUserCreated(t.Context(), payload); err != nil {
		t.Fatalf("an unconfigured campaign is a skip, not an error: %v", err)
	}
	reason, ok := h.lastSkip(t, payload.UserID)
	if !ok || reason != SkipCampaignNotConfigured {
		t.Fatalf("expected skip reason %q, got %q (found=%v)", SkipCampaignNotConfigured, reason, ok)
	}
}

// TestCampaignAdmissionSkipsAfterCampaignEnd proves the campaign CLOSES. An
// open-ended campaign (ends_at omitted) is covered by the granting tests; this
// one is the boundary, and it is the reason `ends_at` is optional rather than
// required (FR-009).
func TestCampaignAdmissionSkipsAfterCampaignEnd(t *testing.T) {
	startsAt := baseInstant().Add(-2 * time.Hour)
	endsAt := baseInstant().Add(-time.Hour)
	campaigns := []sharedconfig.CampaignConfig{
		testCampaignConfig("launch-2026-10", "premium-monthly", 50, 5, &startsAt, &endsAt),
	}
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, campaigns, claims)

	payload := identitydto.UserCreatedEventPayload{
		UserID: uuid.New(),
		// Inside the window by creation time: eligibility is judged on this, and it
		// passed. What closed the campaign is the CLOCK.
		RegisteredAt: baseInstant().Add(-30 * time.Minute),
	}
	if err := h.uc.ProcessUserCreated(t.Context(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.claimFor(payload.UserID) != nil {
		t.Fatalf("a campaign past its declared end must not grant (FR-013)")
	}
	reason, ok := h.lastSkip(t, payload.UserID)
	if !ok || reason != SkipCampaignSleeping {
		t.Fatalf("expected skip reason %q, got %q (found=%v)", SkipCampaignSleeping, reason, ok)
	}
}

// TestCampaignAdmissionSkipsBeforeCampaignStart proves condition 2's other
// branch: a campaign that has not opened yet is `campaign_not_started`, which is
// a different code from `campaign_sleeping` because the remedy is different —
// one is "we will start", the other is "we are done".
func TestCampaignAdmissionSkipsBeforeCampaignStart(t *testing.T) {
	startsAt := baseInstant().Add(time.Hour)
	campaigns := []sharedconfig.CampaignConfig{
		testCampaignConfig("launch-2026-10", "premium-monthly", 50, 5, &startsAt, nil),
	}
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, campaigns, claims)

	payload := identitydto.UserCreatedEventPayload{UserID: uuid.New(), RegisteredAt: baseInstant()}
	if err := h.uc.ProcessUserCreated(t.Context(), payload); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	reason, ok := h.lastSkip(t, payload.UserID)
	if !ok || reason != SkipCampaignNotStarted {
		t.Fatalf("expected skip reason %q, got %q (found=%v)", SkipCampaignNotStarted, reason, ok)
	}
}
