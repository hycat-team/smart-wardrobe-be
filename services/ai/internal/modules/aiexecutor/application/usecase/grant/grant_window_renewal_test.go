package grant

import (
	"testing"

	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"
)

// 024 T168 / FR-032: a renewal is the only change type whose window is computed from
// durationDays, so a renewal without one has no defined window at all.
//
// It used to be accepted anyway. EventGrantWindow fell through to the ordinary
// branch for a missing or non-positive value, which set the start to EffectiveAt and
// the status to Active — so the entitlement was handed over the moment the event
// arrived rather than at "new expiry minus duration". That opens the account early:
// the customer is using a paid feature for a period they have not paid for, and
// nothing in the system objects.

func TestRenewalWithoutPositiveDurationDaysIsRefused(t *testing.T) {
	for _, tc := range []struct {
		name         string
		durationDays int
	}{
		{name: "durationDays missing", durationDays: 0},
		{name: "durationDays negative", durationDays: -30},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := validPayload()
			payload.EventType = string(subscriptioneventtype.EXTENDED)
			payload.DurationDays = tc.durationDays

			requireReason(t, mustReject(t, payload), ReasonMissingRequiredField)
		})
	}
}

// The other change types are untouched by this rule. ACTIVATED with a zero duration is
// a plan with no end date, which is legitimate, and the guard must not swallow it.
func TestNonRenewalWithZeroDurationDaysIsStillAccepted(t *testing.T) {
	payload := validPayload()
	payload.EventType = string(subscriptioneventtype.ACTIVATED)
	payload.DurationDays = 0

	event, err := ParsePlanChangedPayload(testLogger(), payload)
	if err != nil {
		t.Fatalf("a plan with no end date is valid and must not be refused: %v", err)
	}
	if event.DurationDays != 0 {
		t.Fatalf("durationDays must stay 0, got = %d", event.DurationDays)
	}
}

// A renewal WITH a positive duration is the normal path and must still work, otherwise
// this guard has simply broken renewals.
func TestRenewalWithPositiveDurationDaysIsAccepted(t *testing.T) {
	payload := validPayload()
	payload.EventType = string(subscriptioneventtype.EXTENDED)
	payload.DurationDays = 30

	if _, err := ParsePlanChangedPayload(testLogger(), payload); err != nil {
		t.Fatalf("a renewal with a valid durationDays must be accepted: %v", err)
	}
}
