package campaign

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	identitydto "smart-wardrobe-be/services/account/dto/identity"
	notificationcontract "smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	sharedevent "smart-wardrobe-be/shared/application/event"
	sharedconfig "smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
)

// 025 T046-T050: the grant itself, once the quota rules have said yes.
//
// Phase 3 proved the decision without granting anything. These tests prove the
// consequences: the plan lands, the AI entitlement follows it, the history row
// describes the NEW term, and either all of it happens or none of it does.

// TestCampaignGrantAppliesPlanWithAutoRenewOff is SC-006 / FR-035.
//
// Auto-renew OFF is not a detail. `ApplySubscriptionPlanWithEvent` clears the flag
// only on certain branches, and a flag left ON means the renewal worker will later
// attempt a payment for a customer who never bought anything — they were handed a
// month for registering, and they get charged for it.
func TestCampaignGrantAppliesPlanWithAutoRenewOff(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)
	campaign := testCampaignConfig("launch-2026-10", "premium-monthly", 50, 5, &startsAt, nil)
	h := newTestUseCase(t, []sharedconfig.CampaignConfig{campaign}, newFakeClaimRepo())

	payload := identitydto.UserCreatedEventPayload{UserID: uuid.New(), RegisteredAt: baseInstant()}
	if !h.granted(t, payload) {
		t.Fatalf("expected the account to be granted the campaign plan")
	}

	sub, err := h.subs.GetByUserID(context.Background(), payload.UserID)
	if err != nil || sub == nil {
		t.Fatalf("read granted subscription: %v", err)
	}
	if sub.CurrentPlanCode != "premium-monthly" {
		t.Fatalf("expected plan premium-monthly, got %q", sub.CurrentPlanCode)
	}
	if sub.IsAutoRenewEnabled {
		t.Fatalf("expected auto-renew OFF on a promotional grant (FR-035); nobody consented to be charged")
	}
	// FR-019: the term is the PLAN's duration — 30 days for premium-monthly, not
	// a literal written here — anchored on the GRANT instant. This account
	// registered at the harness clock, so the anchor is not observable from here;
	// TestCampaignGrantWritesAClaimExpiryMatchingTheSubscription pins it down
	// with a genuinely late account.
	wantExpiry := baseInstant().AddDate(0, 0, 30)
	if sub.ExpiresAt == nil {
		t.Fatalf("expected a finite expiry, got nil")
	}
	if !sub.ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("expected expires_at = grant instant + 30d = %s, got %s", wantExpiry, sub.ExpiresAt)
	}
}

// TestCampaignGrantWritesAClaimExpiryMatchingTheSubscription guards a split-brain
// that two records describing one grant can fall into.
//
// `campaign_claims.expires_at` was anchored on the account's registration while
// `user_subscriptions.expires_at` was anchored on the grant instant. The two rows
// therefore disagreed, and the claim row is what the admin query and the audit
// trail read — so an operator would have been shown a date the customer did not
// actually have. With a late message the gap is as wide as the delay.
//
// The account here registered eight hours ago, which is the case where the two
// anchors differ enough to tell apart.
func TestCampaignGrantWritesAClaimExpiryMatchingTheSubscription(t *testing.T) {
	registeredAt := baseInstant().Add(-8 * time.Hour)
	startsAt := registeredAt.Add(-time.Hour)
	campaign := testCampaignConfig("launch-2026-10", "premium-monthly", 50, 5, &startsAt, nil)
	h := newTestUseCase(t, []sharedconfig.CampaignConfig{campaign}, newFakeClaimRepo())

	payload := identitydto.UserCreatedEventPayload{UserID: uuid.New(), RegisteredAt: registeredAt}
	if !h.granted(t, payload) {
		t.Fatalf("expected a late message to still grant")
	}

	claim := h.claims.claimFor(payload.UserID)
	if claim == nil {
		t.Fatalf("expected a claim row for the granted account")
	}
	sub, err := h.subs.GetByUserID(context.Background(), payload.UserID)
	if err != nil || sub == nil {
		t.Fatalf("read granted subscription: %v", err)
	}

	if !claim.ExpiresAt.Equal(*sub.ExpiresAt) {
		t.Fatalf("claim expires_at = %s but the subscription expires_at = %s; one grant must not end on two different dates",
			claim.ExpiresAt, *sub.ExpiresAt)
	}
	if claim.ExpiresAt.Equal(registeredAt.AddDate(0, 0, 30)) {
		t.Fatalf("claim expires_at = %s is still anchored on the registration time %s", claim.ExpiresAt, registeredAt)
	}

	// Lateness must cost the customer nothing: still a full term from the grant.
	wantExpiry := baseInstant().AddDate(0, 0, 30)
	if !claim.ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("expected a full 30-day term from the grant instant (%s), got %s", wantExpiry, claim.ExpiresAt)
	}
}

// TestCampaignGrantTermIsAnchoredOnRegistrationNotProcessing proves FR-078: a
// message that arrives late still grants the term the account was promised.
//
// Anchoring on the processing clock would silently shorten the term by exactly
// the queue latency, and every account would receive a different term depending on
// when its message happened to be consumed. Nobody would notice: the customer gets
// Premium, just less of it.
// TestCampaignGrantEventIsStampedOnProcessingNotRegistration guards the one bug
// in this feature that only an end-to-end run against a real AI service exposed.
//
// The grant used to be stamped with the account's REGISTRATION time. The signup
// transaction provisions the free plan microseconds later and stamps THAT event
// with a real "now", so the campaign event arrived carrying an `occurredAt` older
// than the record already on file. The AI service enforces a monotonic guard on
// `effective_from` and rejected the grant as `stale_event`, then dead-lettered the
// message. The account sat on premium-monthly with the free plan's AI
// entitlements and no way back: the message was gone and the claim was written.
//
// The guard is correct and predates this feature, so the event is stamped with the
// processing clock. The term is grant time plus the plan's own duration, which is
// marginally MORE than registration-anchored for a late message — the right
// direction for a promotion.
func TestCampaignGrantEventIsStampedOnProcessingNotRegistration(t *testing.T) {
	registeredAt := baseInstant().Add(-8 * time.Hour)
	startsAt := registeredAt.Add(-time.Hour)
	campaign := testCampaignConfig("launch-2026-10", "premium-monthly", 50, 5, &startsAt, nil)
	h := newTestUseCase(t, []sharedconfig.CampaignConfig{campaign}, newFakeClaimRepo())

	// Registered eight hours ago; the harness clock is "now". A message this late
	// must still grant (FR-078), which is the half that was always right.
	payload := identitydto.UserCreatedEventPayload{UserID: uuid.New(), RegisteredAt: registeredAt}
	if !h.granted(t, payload) {
		t.Fatalf("expected a late message to still grant")
	}

	sub, err := h.subs.GetByUserID(context.Background(), payload.UserID)
	if err != nil {
		t.Fatalf("read subscription: %v", err)
	}

	// The published event is what the AI service orders by. It must be at or
	// after the free plan's provisioning event, or the AI rejects it.
	published := h.publisher.payloads()
	if len(published) != 1 {
		t.Fatalf("expected exactly 1 published plan-change event, got %d", len(published))
	}
	body := asMap(t, published[0].Payload)
	rawOccurredAt, _ := body["occurredAt"].(string)
	gotOccurredAt, err := time.Parse(time.RFC3339Nano, rawOccurredAt)
	if err != nil {
		t.Fatalf("occurredAt %q is not RFC3339: %v", rawOccurredAt, err)
	}
	if gotOccurredAt.Before(baseInstant().UTC()) {
		t.Fatalf("event occurredAt = %s is older than the processing clock %s; the AI service will reject it as stale_event",
			gotOccurredAt.UTC(), baseInstant().UTC())
	}
	if gotOccurredAt.Equal(registeredAt.UTC()) {
		t.Fatalf("event occurredAt = %s equals the registration time; this is the stale_event defect", gotOccurredAt.UTC())
	}

	// The term still spans a full month from the grant, so a late message does
	// not hand the customer a shortened promotion.
	wantExpiry := baseInstant().AddDate(0, 0, 30)
	if sub.ExpiresAt == nil || !sub.ExpiresAt.Equal(wantExpiry) {
		t.Fatalf("expected a full 30-day term from the grant instant (%s), got %v", wantExpiry, sub.ExpiresAt)
	}
}

// TestCampaignGrantRereadsSubscriptionBeforePublishing is R13 / SC-008, and it is
// the single most expensive bug in this feature.
//
// `BuildPlanChangedPayload` reads `effectiveTo` from `sub.ExpiresAt`, and
// `ApplySubscriptionPlanWithEvent` returns the HISTORY ROW — not the updated
// subscription. A caller that reuses the pre-apply row therefore publishes an
// EMPTY `effectiveTo`, and the AI service reads that as a LIFETIME grant: one month
// of Premium handed to an account as unlimited AI access. Nothing errors. The money
// is real and the quota is not.
func TestCampaignGrantRereadsSubscriptionBeforePublishing(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)
	campaign := testCampaignConfig("launch-2026-10", "premium-monthly", 50, 5, &startsAt, nil)
	h := newTestUseCase(t, []sharedconfig.CampaignConfig{campaign}, newFakeClaimRepo())

	payload := identitydto.UserCreatedEventPayload{UserID: uuid.New(), RegisteredAt: baseInstant()}
	if !h.granted(t, payload) {
		t.Fatalf("expected the account to be granted")
	}

	published := h.publisher.payloads()
	if len(published) != 1 {
		t.Fatalf("expected exactly 1 published plan-change event, got %d", len(published))
	}
	payloadText := asMap(t, published[0].Payload)

	effectiveTo, ok := payloadText["effectiveTo"].(string)
	if !ok || effectiveTo == "" {
		t.Fatalf("effectiveTo must be present and non-empty: an empty one reads as a LIFETIME grant on the AI side (R13), got %#v", payloadText["effectiveTo"])
	}
	wantExpiry := payload.RegisteredAt.AddDate(0, 0, 30)
	if !strings.HasPrefix(effectiveTo, wantExpiry.UTC().Format("2006-01-02")) {
		t.Fatalf("expected effectiveTo to match the granted term %s, got %q", wantExpiry, effectiveTo)
	}
}

// TestCampaignGrantUpdatesAiEntitlementToPremium is FR-037 / T048: the published
// event carries the Premium plan's id AND its AI cost policy, so the AI service
// moves the account to the Premium policy without any change to services/ai
// (FR-097).
//
// It is asserted on the published payload rather than on the AI service's tables
// because the AI service is out of scope by FR-097 and is not modified. What this
// proves is that everything the AI service needs to do the work correctly IS in
// the message.
func TestCampaignGrantUpdatesAiEntitlementToPremium(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)
	campaign := testCampaignConfig("launch-2026-10", "premium-monthly", 50, 5, &startsAt, nil)
	h := newTestUseCase(t, []sharedconfig.CampaignConfig{campaign}, newFakeClaimRepo())

	payload := identitydto.UserCreatedEventPayload{UserID: uuid.New(), RegisteredAt: baseInstant()}
	if !h.granted(t, payload) {
		t.Fatalf("expected the account to be granted")
	}

	plan, _ := h.plans.GetBySlugAnyStatus(context.Background(), "premium-monthly")
	body := asMap(t, h.publisher.payloads()[0].Payload)

	if got := body["toPlanCode"]; got != plan.Slug {
		t.Fatalf("expected toPlanCode %q, got %v", plan.Slug, got)
	}
	if got := body["toPlanId"]; got != plan.ID.String() {
		t.Fatalf("expected toPlanId %q so the AI service can write a grant, got %v", plan.ID, got)
	}
	if got := body["aiCostPolicyId"]; got != plan.AICostPolicyID.String() {
		t.Fatalf("expected aiCostPolicyId %q so the entitlement becomes Premium's, got %v", plan.AICostPolicyID, got)
	}
	// R8: the event TYPE is not asserted. It is `ACTIVATED`, same as an
	// administrator's grant, and asserting it would bake in a behaviour that is
	// correct rather than a requirement. The observable consequence is the payload
	// above.
}

// TestCampaignGrantWritesExactlyOneHistoryRow is FR-095 / T105's sibling here: one
// grant, one history row.
//
// A grant with no history row is invisible to the admin look-up; two rows for one
// grant means the AI service may apply the entitlement twice.
func TestCampaignGrantWritesExactlyOneHistoryRow(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)
	campaign := testCampaignConfig("launch-2026-10", "premium-monthly", 50, 5, &startsAt, nil)
	h := newTestUseCase(t, []sharedconfig.CampaignConfig{campaign}, newFakeClaimRepo())

	payload := identitydto.UserCreatedEventPayload{UserID: uuid.New(), RegisteredAt: baseInstant()}
	if !h.granted(t, payload) {
		t.Fatalf("expected the account to be granted")
	}
	rows := h.events.published()
	if len(rows) != 1 {
		t.Fatalf("expected exactly 1 plan-change history row for one grant, got %d", len(rows))
	}
	if got := rows[0].Event.UserID; got != payload.UserID {
		t.Fatalf("history row belongs to %s, want %s", got, payload.UserID)
	}
}

// TestCampaignGrantIsAtomicWhenPublishingFails is SC-003 / FR-039: if the
// plan-change event cannot be published, NOTHING survives.
//
// The two half-states are both silent and both expensive:
//   - a plan applied with no event: the AI service keeps the free-tier policy, so a
//     customer on Premium prices gets free-tier AI, and nothing reports an error;
//   - a claim written with no plan: the catch-up sweep finds the claim-free account
//     later and grants it a second time.
//
// The fake unit of work here does not roll back, so the test asserts on what the
// code DOES do with a failure — return it, and report it as transient so the
// message is retried rather than dead-lettered.
func TestCampaignGrantIsAtomicWhenPublishingFails(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)
	campaign := testCampaignConfig("launch-2026-10", "premium-monthly", 50, 5, &startsAt, nil)
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, []sharedconfig.CampaignConfig{campaign}, claims)
	h.publisher.failNext = true

	payload := identitydto.UserCreatedEventPayload{UserID: uuid.New(), RegisteredAt: baseInstant()}
	err := h.uc.ProcessUserCreated(context.Background(), payload)
	if err == nil {
		t.Fatalf("expected an error when the plan-change event cannot be published (FR-039)")
	}
	var retryable sharedevent.RetryableError
	if !errors.As(err, &retryable) {
		t.Fatalf("expected a RetryableError so the message is retried rather than dead-lettered, got %T", err)
	}
	// FR-049's counterpart: the email must not have gone out for a grant that
	// rolled back. A recipient must never be told about a plan that does not exist.
	if got := len(h.notif.emails()); got != 0 {
		t.Fatalf("expected no email for a rolled-back grant, got %d", got)
	}
	// FR-049: a failed email must not lose the grant, so this is asserted in the
	// opposite direction below.
}

// TestCampaignGrantSurvivesEmailFailure is FR-049: the email is best-effort.
//
// Reporting a failed email as a failed grant would send the catch-up sweep round
// again and hand the customer a second plan — a self-inflicted duplicate, caused
// by a mail server being down.
func TestCampaignGrantSurvivesEmailFailure(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)
	campaign := testCampaignConfig("launch-2026-10", "premium-monthly", 50, 5, &startsAt, nil)
	h := newTestUseCase(t, []sharedconfig.CampaignConfig{campaign}, newFakeClaimRepo())
	h.notif.fail = true

	payload := identitydto.UserCreatedEventPayload{UserID: uuid.New(), RegisteredAt: baseInstant()}
	if err := h.uc.ProcessUserCreated(context.Background(), payload); err != nil {
		t.Fatalf("a failed email must not fail the grant (FR-049): %v", err)
	}
	sub, _ := h.subs.GetByUserID(context.Background(), payload.UserID)
	if sub == nil || sub.CurrentPlanCode != "premium-monthly" {
		t.Fatalf("expected the plan to be applied despite the email failure (FR-049)")
	}
	// FR-049 also asks for a trace of the failure.
	found := false
	for _, line := range h.logger.lines() {
		if line.level == "warn" && line.userID == payload.UserID.String() {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a warning recording the email failure (FR-049 asks for a trace)")
	}
}

// TestCampaignGrantRecordsCampaignOriginInTheAuditTrail is FR-038 / FR-094: the
// audit row names the campaign and records NO administrator.
//
// The origin lives here and not in the plan-change history because the history is
// a cross-service contract and may not grow a field for it (FR-058). An operator
// answering "who gave this customer Premium?" reads the audit trail.
func TestCampaignGrantRecordsCampaignOriginInTheAuditTrail(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)
	campaign := testCampaignConfig("launch-2026-10", "premium-monthly", 50, 5, &startsAt, nil)
	h := newTestUseCase(t, []sharedconfig.CampaignConfig{campaign}, newFakeClaimRepo())

	payload := identitydto.UserCreatedEventPayload{UserID: uuid.New(), RegisteredAt: baseInstant()}
	if !h.granted(t, payload) {
		t.Fatalf("expected the account to be granted")
	}

	records := h.audit.all()
	if len(records) != 1 {
		t.Fatalf("expected exactly 1 audit record, got %d", len(records))
	}
	record := records[0]
	if record.Payload["campaignCode"] != "launch-2026-10" {
		t.Fatalf("expected the audit payload to name the campaign, got %#v", record.Payload["campaignCode"])
	}
	if record.Payload["grantSource"] != "campaign" {
		t.Fatalf("expected grantSource=campaign, got %#v", record.Payload["grantSource"])
	}
	// "No administrator performed this" is the zero UUID, because the column is
	// NOT NULL and a sentinel is what says "nobody" rather than "unknown".
	if record.ActorIsZero() != true {
		t.Fatalf("expected no administrator as the actor of a campaign grant (FR-038), got %v", record.ActorID())
	}
}

// TestCampaignGrantDoesNotExtendAnExistingPremiumTerm is SC-009 / FR-018: an
// account already on the campaign's plan with a live term is NOT granted, and its
// expiry is left exactly as it was.
func TestCampaignGrantDoesNotExtendAnExistingPremiumTerm(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)
	campaign := testCampaignConfig("launch-2026-10", "premium-monthly", 50, 5, &startsAt, nil)
	claims := newFakeClaimRepo()
	h := newTestUseCase(t, []sharedconfig.CampaignConfig{campaign}, claims)

	// Already on Premium with ten days left.
	userID := uuid.New()
	existingExpiry := baseInstant().AddDate(0, 0, 10)
	plan, _ := h.plans.GetBySlugAnyStatus(context.Background(), "premium-monthly")
	h.subs.put(&entities.UserSubscription{
		UserID:             userID,
		SubscriptionPlanID: plan.ID,
		CurrentPlanCode:    "premium-monthly",
		CurrentTierRank:    1,
		CurrentPlanKind:    1,
		StartedAt:          baseInstant().AddDate(0, 0, -20),
		ExpiresAt:          &existingExpiry,
		IsAutoRenewEnabled: false,
	})

	payload := identitydto.UserCreatedEventPayload{UserID: userID, RegisteredAt: baseInstant()}
	if h.granted(t, payload) {
		t.Fatalf("an account already on the campaign's plan must not be granted again (FR-018)")
	}
	reason, ok := h.lastSkip(t, userID)
	if !ok || reason != SkipAlreadyOnPlan {
		t.Fatalf("expected skip reason %q, got %q (found=%v)", SkipAlreadyOnPlan, reason, ok)
	}

	sub, _ := h.subs.GetByUserID(context.Background(), userID)
	if sub.ExpiresAt == nil || !sub.ExpiresAt.Equal(existingExpiry) {
		t.Fatalf("the term must be left exactly as it was: want %s, got %v", existingExpiry, sub.ExpiresAt)
	}
	if got := claims.total(); got != 0 {
		t.Fatalf("expected no claim written for a refused account, got %d", got)
	}
}

// TestCampaignGrantSendsOneEmailWithNoCampaignQuota is T068 / SC-005, from the
// grant's side: the customer is told what they got and when it ends, and NOT how
// many giveaways are left (FR-046).
//
// The template's own content is asserted in the notification module's test; what
// belongs here is that exactly one email is enqueued, that it uses the campaign
// template rather than the administrator's, and that the parameters carry no
// campaign quota figure.
func TestCampaignGrantSendsOneEmailWithNoCampaignQuota(t *testing.T) {
	startsAt := baseInstant().Add(-time.Hour)
	campaign := testCampaignConfig("launch-2026-10", "premium-monthly", 50, 5, &startsAt, nil)
	h := newTestUseCase(t, []sharedconfig.CampaignConfig{campaign}, newFakeClaimRepo())

	payload := identitydto.UserCreatedEventPayload{UserID: uuid.New(), RegisteredAt: baseInstant()}
	if !h.granted(t, payload) {
		t.Fatalf("expected the account to be granted")
	}

	emails := h.notif.emails()
	if len(emails) != 1 {
		t.Fatalf("expected exactly 1 email per grant (FR-044), got %d", len(emails))
	}
	if emails[0].Template != string(notificationcontract.TemplateCampaignGrant) {
		t.Fatalf("expected the campaign template, got %q — reusing admin_grant would tell a campaign customer text written for an administrator (R12)", emails[0].Template)
	}
	if emails[0].UserID != payload.UserID {
		t.Fatalf("expected the email addressed to the granted account, got %s", emails[0].UserID)
	}
	if emails[0].Params["PlanName"] != "Premium Tháng" {
		t.Fatalf("expected the plan name in the email, got %#v", emails[0].Params["PlanName"])
	}
	if emails[0].Params["ExpiresAt"] == nil || emails[0].Params["ExpiresAt"] == "" {
		t.Fatalf("expected the expiry in the email (FR-044), got %#v", emails[0].Params["ExpiresAt"])
	}
	// FR-046: the remaining-giveaway count is irrelevant to this customer and is
	// how a campaign turns into a queue at the door.
	for key := range emails[0].Params {
		if strings.Contains(strings.ToLower(key), "quota") ||
			strings.Contains(strings.ToLower(key), "remaining") ||
			strings.Contains(strings.ToLower(key), "campaign") {
			t.Fatalf("the email must not carry campaign accounting (FR-046), found parameter %q", key)
		}
	}
}

// TestCampaignGrantSkipsEmailForAccountsThatWereNotGranted is FR-048, stated from
// the grant's side: the refused paths never reach the notification.
func TestCampaignGrantSkipsEmailForAccountsThatWereNotGranted(t *testing.T) {
	// A BOUNDED campaign, so both sides of the window rule can be exercised. With
	// an open-ended one an account registered later is still eligible — that is
	// FR-078 working as intended, not a window failure.
	startsAt := baseInstant().Add(-time.Hour)
	endsAt := baseInstant().Add(time.Hour)
	campaigns := []sharedconfig.CampaignConfig{
		testCampaignConfig("launch-2026-10", "premium-monthly", 50, 5, &startsAt, &endsAt),
	}
	h := newTestUseCase(t, campaigns, newFakeClaimRepo())

	// FR-018: already on the campaign's plan with a live term.
	alreadyOnPlan := uuid.New()
	existingExpiry := baseInstant().AddDate(0, 0, 5)
	plan, _ := h.plans.GetBySlugAnyStatus(context.Background(), "premium-monthly")
	h.subs.put(&entities.UserSubscription{
		UserID:             alreadyOnPlan,
		SubscriptionPlanID: plan.ID,
		CurrentPlanCode:    "premium-monthly",
		CurrentTierRank:    1,
		CurrentPlanKind:    1,
		StartedAt:          baseInstant().AddDate(0, 0, -25),
		ExpiresAt:          &existingExpiry,
	})

	refusals := map[string]identitydto.UserCreatedEventPayload{
		// FR-091: an administrator's own account.
		"created_by_admin": {UserID: uuid.New(), RegisteredAt: baseInstant(), CreatedByAdmin: true},
		// FR-017: created after the campaign closed. The window is evaluated
		// against `registered_at`, so this fails even though the processing clock
		// is comfortably inside the campaign.
		"outside_window": {UserID: uuid.New(), RegisteredAt: baseInstant().Add(time.Hour)},
		// FR-017: created before it opened — the other side of the same window.
		"before_start": {UserID: uuid.New(), RegisteredAt: baseInstant().Add(-72 * time.Hour)},
		// FR-018: already holds the plan.
		"already_on_plan": {UserID: alreadyOnPlan, RegisteredAt: baseInstant()},
	}
	for name, payload := range refusals {
		if err := h.uc.ProcessUserCreated(context.Background(), payload); err != nil {
			t.Fatalf("%s: unexpected error: %v", name, err)
		}
		if reason, ok := h.lastSkip(t, payload.UserID); !ok {
			t.Fatalf("%s: expected a recorded skip reason (FR-061)", name)
		} else if reason == "" {
			t.Fatalf("%s: expected a non-empty skip reason", name)
		}
	}
	if got := len(h.notif.emails()); got != 0 {
		t.Fatalf("expected NO email for accounts that were not granted (FR-048), got %d", got)
	}
	if got := len(h.audit.all()); got != 0 {
		t.Fatalf("expected no audit record for accounts that were not granted, got %d", got)
	}
}
