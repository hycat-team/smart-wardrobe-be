package subscriptioneventtype

import "testing"

// 025 T103 (FR-092): the shared plan-change type set is CLOSED, and this test is
// the only thing that notices when somebody widens it.
//
// The compiler cannot see this rule. Adding a ninth constant to the `const` block
// and one more entry to `All` compiles cleanly, every existing test stays green,
// and the failure lands in the CONSUMER, far from the edit: the AI service rejects
// a type it does not know, dead-letters the message, and every campaign grant from
// that deploy on is lost. Nothing reports it at the point of the change.
//
// That is why the assertion is an exact list rather than a minimum count. "At
// least seven" would still pass on an eighth entry, which is precisely the edit
// FR-092 forbids.
//
// The expectation is spelled out in full so that ADDING a type forces a
// deliberate edit here, where the consequences are written down, instead of
// passing unnoticed.
func TestPlanChangeTypeSetIsExactlyTheSevenDeclaredTypes(t *testing.T) {
	want := []Type{
		ACTIVATED,
		EXTENDED,
		UPGRADED,
		OVERLAID,
		LIFETIME_FALLBACK_RESTORED,
		SUBSCRIPTION_DOWNGRADED_TO_FREE,
		AUTO_RENEW_UPDATED,
	}

	if len(All) != len(want) {
		t.Errorf("FR-092: the closed set has %d entries, want exactly %d. "+
			"A new plan-change type is a CROSS-SERVICE contract change (FR-058): the AI service "+
			"rejects anything outside this set and dead-letters the message, so every campaign "+
			"grant would be lost in the deploy that added it without the receiver",
			len(All), len(want))
	}

	for _, typ := range want {
		if !All[typ] {
			t.Errorf("FR-092: %q is missing from All, but it is declared in the const block; "+
				"IsValid would reject an event this service publishes", typ)
		}
	}

	// Every key of All must be one of the declared constants. A literal string
	// added straight to the map would compile, would be "valid", and would have
	// no constant to reference — so nothing downstream could ever publish it.
	for typ := range All {
		if !declaredConstant(typ) {
			t.Errorf("FR-092: %q is a key of All but has no exported constant; every member of the "+
				"closed set must be declared in the const block so both services name it identically", typ)
		}
	}
}

// TestIsValidAgreesWithTheClosedSet keeps the two accessors from drifting apart.
//
// `All` and `IsValid` are the only two doors into the set. If they were derived
// from different lists, one publisher could accept a type the consumer rejects,
// and the message would still reach the dead-letter queue.
func TestIsValidAgreesWithTheClosedSet(t *testing.T) {
	for typ := range All {
		if !IsValid(typ) {
			t.Errorf("IsValid(%q) = false but the type is in All", typ)
		}
	}

	// A type nobody declared must be refused rather than guessed at. Guessing
	// writes an AI entitlement grant that should not exist, and the customer is
	// charged Premium prices for it.
	if IsValid(Type("CAMPAIGN_GRANTED")) {
		t.Error("FR-092: an undeclared type must be refused; the consumer must never guess what to do with one")
	}
	if IsValid(Type("RENEWED")) {
		t.Error(`"RENEWED" is deliberately absent: the consumer once handled it, no path ever published it, and the handling was dead code`)
	}
	if IsValid(Type("EXPIRED")) {
		t.Error(`"EXPIRED" is deliberately absent, for the same reason as "RENEWED"`)
	}
}

// declaredConstant reports whether typ has a matching constant in the declared
// set. Written as a literal list rather than by parsing the source so that adding
// a constant alone does not silently widen what this file considers declared —
// that has to be a conscious edit here, next to this explanation.
func declaredConstant(typ Type) bool {
	switch typ {
	case ACTIVATED, EXTENDED, UPGRADED, OVERLAID,
		LIFETIME_FALLBACK_RESTORED, SUBSCRIPTION_DOWNGRADED_TO_FREE, AUTO_RENEW_UPDATED:
		return true
	default:
		return false
	}
}
