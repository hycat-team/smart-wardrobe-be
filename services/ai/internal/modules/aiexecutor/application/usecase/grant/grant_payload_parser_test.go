package grant

import (
	"errors"
	"testing"
	"time"

	aiexecutor_dto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	sharedevent "smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"

	"github.com/google/uuid"
)

const (
	validUserID   = "ad11ad11-ad11-ad11-ad11-ad11ad11ad12"
	validPolicyID = "aa000000-0000-0000-0000-000000000002"
	validPlanID   = "cb591a26-9f4a-4e86-b489-8d195c80521e"
	validEventID  = "11111111-2222-3333-4444-555555555555"
)

func validPayload() aiexecutor_dto.PlanChangedEventPayload {
	now := time.Date(2026, 9, 29, 8, 15, 0, 0, time.UTC)
	return aiexecutor_dto.PlanChangedEventPayload{
		EventVersion:   2,
		UserID:         validUserID,
		EventType:      string(subscriptioneventtype.ACTIVATED),
		ToPlanID:       validPlanID,
		ToPlanCode:     "premium-monthly",
		ToTierRank:     1,
		AICostPolicyID: validPolicyID,
		OccurredAt:     now.Format(time.RFC3339Nano),
		EffectiveAt:    now.Format(time.RFC3339Nano),
		EffectiveTo:    now.AddDate(0, 0, 30).Format(time.RFC3339Nano),
		DurationDays:   30,
		SourceEventID:  validEventID,
		FromPlanCode:   "free",
	}
}

func requireReason(t *testing.T, err error, wantReason string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected a rejection with reason %q, got nil", wantReason)
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected *ParseError, got %T: %v", err, err)
	}
	if parseErr.Reason != wantReason {
		t.Fatalf("expected reason %q, got %q (%s)", wantReason, parseErr.Reason, parseErr.Message)
	}
}

func TestValidPayloadIsAccepted(t *testing.T) {
	event, err := ParsePlanChangedPayload(testLogger(), validPayload())
	if err != nil {
		t.Fatalf("unexpected rejection: %v", err)
	}
	if event.UserID != uuid.MustParse(validUserID) {
		t.Fatalf("user id was not parsed: %s", event.UserID)
	}
	if event.ToPlanID != uuid.MustParse(validPlanID) {
		t.Fatalf("toPlanId was not parsed: %s", event.ToPlanID)
	}
	if event.SourceEventID != uuid.MustParse(validEventID) {
		t.Fatalf("sourceEventId was not parsed: %s", event.SourceEventID)
	}
	if event.EffectiveTo == nil {
		t.Fatal("effectiveTo was dropped")
	}
	if event.DurationDays != 30 {
		t.Fatalf("durationDays was dropped: %d", event.DurationDays)
	}
}

// TestNewerVersionIsRefused is the rule that stops a record being written from a
// payload whose meaning is unknown: the result would look valid and be wrong.
func TestNewerVersionIsRefused(t *testing.T) {
	payload := validPayload()
	payload.EventVersion = SupportedEventVersion + 1

	requireReason(t, mustReject(t, payload), ReasonUnsupportedVersion)
}

// TestMissingVersionIsTreatedAsVersionOne: a payload sitting in the queue at
// deploy time has no eventVersion, and is read under the OLDER rules rather than
// refused for an unknown version.
//
// 024 T137: those older rules now include refusing the event, because a version-1
// payload carries no `toPlanId` and this receiver may not derive one - reading the
// Account service's subscription table to do it is exactly what FR-024 forbids, and
// `plan_id` is NOT NULL so a nil UUID would have been accepted silently.
//
// The previous version of this test asserted the opposite, with a comment saying
// "the writer must then refuse" - and the writer did not. That gap is why this
// test now fails against the old code on purpose.
func TestMissingVersionIsTreatedAsVersionOne(t *testing.T) {
	payload := validPayload()
	payload.EventVersion = 0
	// A genuine version-1 payload: no toPlanId, no effectiveTo, no duration.
	payload.ToPlanID = ""
	payload.EffectiveTo = ""
	payload.DurationDays = 0
	payload.SourceEventID = validEventID

	// Refused as a MISSING REQUIRED FIELD, not as an unknown version - the
	// distinction matters, because an operator reading "unsupported_event_version"
	// would go looking for a deployment problem that does not exist.
	requireReason(t, mustReject(t, payload), ReasonMissingRequiredField)
}

// A version-1 payload that DOES carry a plan id is still accepted. The version
// decides which fields are required, so refusing every version-1 event would drop
// traffic that is perfectly usable.
func TestVersionOneWithAPlanIdIsAccepted(t *testing.T) {
	payload := validPayload()
	payload.EventVersion = 1
	payload.SourceEventID = validEventID

	event, err := ParsePlanChangedPayload(testLogger(), payload)
	if err != nil {
		t.Fatalf("ban v1 co toPlanId phai duoc chap nhan: %v", err)
	}
	if event.EventVersion != 1 {
		t.Errorf("expected version 1, got %d", event.EventVersion)
	}
	if event.ToPlanID == uuid.Nil {
		t.Error("toPlanId phai duoc doc")
	}
	// effectiveTo was introduced in version 2, so it stays absent - and the grant
	// window handles that rather than the parser inventing a value.
	if event.EffectiveTo != nil {
		t.Errorf("ban v1 khong co effectiveTo, co = %v", event.EffectiveTo)
	}
}

// TestUnknownEventTypeIsRefused is the direct fix for the old default branch,
// which created a grant for every unrecognised type.
func TestUnknownEventTypeIsRefused(t *testing.T) {
	for _, eventType := range []string{"RENEWED", "EXPIRED", "CANCELLED", "", "activated"} {
		payload := validPayload()
		payload.EventType = eventType
		requireReason(t, mustReject(t, payload), ReasonUnknownEventType)
	}
}

func TestAllSevenKnownTypesAreAccepted(t *testing.T) {
	for _, eventType := range []subscriptioneventtype.Type{
		subscriptioneventtype.ACTIVATED,
		subscriptioneventtype.EXTENDED,
		subscriptioneventtype.UPGRADED,
		subscriptioneventtype.OVERLAID,
		subscriptioneventtype.LIFETIME_FALLBACK_RESTORED,
		subscriptioneventtype.SUBSCRIPTION_DOWNGRADED_TO_FREE,
		subscriptioneventtype.AUTO_RENEW_UPDATED,
	} {
		payload := validPayload()
		payload.EventType = string(eventType)
		if _, err := ParsePlanChangedPayload(testLogger(), payload); err != nil {
			t.Fatalf("type %s must be accepted, got: %v", eventType, err)
		}
	}
}

func TestMissingRequiredFieldsAreRefused(t *testing.T) {
	cases := map[string]func(p *aiexecutor_dto.PlanChangedEventPayload){
		"userId":         func(p *aiexecutor_dto.PlanChangedEventPayload) { p.UserID = "" },
		"aiCostPolicyId": func(p *aiexecutor_dto.PlanChangedEventPayload) { p.AICostPolicyID = "" },
		"toPlanId":       func(p *aiexecutor_dto.PlanChangedEventPayload) { p.ToPlanID = "" },
		"toPlanCode":     func(p *aiexecutor_dto.PlanChangedEventPayload) { p.ToPlanCode = "" },
		"occurredAt":     func(p *aiexecutor_dto.PlanChangedEventPayload) { p.OccurredAt = "" },
		"effectiveAt":    func(p *aiexecutor_dto.PlanChangedEventPayload) { p.EffectiveAt = "" },
	}
	for field, mutate := range cases {
		payload := validPayload()
		mutate(&payload)
		requireReason(t, mustReject(t, payload), ReasonMissingRequiredField)
		_ = field
	}
}

// TestMissingSourceEventIDIsRefused is the strictest rule in the contract.
// Without it the event cannot be de-duplicated, so every redelivery would create
// another grant — the exact invariant this feature exists to restore.
func TestMissingSourceEventIDIsRefused(t *testing.T) {
	for _, value := range []string{"", "not-a-uuid"} {
		payload := validPayload()
		payload.SourceEventID = value
		requireReason(t, mustReject(t, payload), ReasonMissingSourceEventID)
	}
}

func TestMalformedTimestampsAreRefused(t *testing.T) {
	payload := validPayload()
	payload.EffectiveTo = "31/12/2026"
	requireReason(t, mustReject(t, payload), ReasonMalformedTimestamp)
}

func TestMalformedDepositTransactionIDIsRefused(t *testing.T) {
	payload := validPayload()
	payload.SourceDepositTransactionID = "not-a-uuid"
	requireReason(t, mustReject(t, payload), ReasonMissingRequiredField)
}

// TestRejectionsArePermanentByDefault: a payload that will still be invalid on
// the fifth attempt must not burn the delivery limit, or the dead-letter signal
// arrives five times later than it should.
func TestRejectionsArePermanentByDefault(t *testing.T) {
	payload := validPayload()
	payload.EventVersion = 99
	_, err := ParsePlanChangedPayload(testLogger(), payload)
	if err == nil {
		t.Fatal("expected a rejection")
	}
	var parseErr *ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected *ParseError, got %T", err)
	}
	var permanent sharedevent.PermanentError
	if !errors.As(parseErr.AsEventError(), &permanent) {
		t.Fatal("an unsupported version must be a permanent error")
	}
}

func TestTransientFailureIsRetryable(t *testing.T) {
	parseErr := &ParseError{Reason: ReasonTransientFailure, Message: "database timeout"}
	var retryable sharedevent.RetryableError
	if !errors.As(parseErr.AsEventError(), &retryable) {
		t.Fatal("a transient failure must be retryable")
	}
}

func mustReject(t *testing.T, payload aiexecutor_dto.PlanChangedEventPayload) error {
	t.Helper()
	_, err := ParsePlanChangedPayload(testLogger(), payload)
	if err == nil {
		t.Fatal("expected the payload to be rejected")
	}
	return err
}
