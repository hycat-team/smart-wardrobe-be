package dto

// PlanChangedEventPayload is the RabbitMQ message published by the
// subscription module when a user's plan changes (consumed by the AI grant worker).
//
// The shape is the contract between two services, and neither side may import
// the other's DTO — JSON is the agreement. Field 1-6 keep their original names
// and meanings for backward compatibility; 7-14 were added by feature 024
// because the receiver could not compute a renewal window or fill the mandatory
// plan_id column without them.
//
// eventVersion exists so the receiver can refuse a payload it does not
// understand instead of writing half a record. See
// contracts/subscription-plan-changed-event.md.
type PlanChangedEventPayload struct {
	// EventVersion is 2 for the current contract. A payload without the field
	// is version 1 (the six original fields) and is handled with the
	// missing-field rules.
	EventVersion int    `json:"eventVersion"`
	UserID       string `json:"userId"`
	// EventType is copied verbatim from the user_subscription_events row, never
	// hand-written at the call site, so the two sides cannot drift apart.
	EventType string `json:"eventType"`
	// ToPlanID is new. ai_policy_grants.plan_id is NOT NULL and its foreign key
	// was dropped, so nothing on the receiver side can recover it; if the
	// publisher omits this, the receiver cannot fill the column.
	ToPlanID       string `json:"toPlanId"`
	ToPlanCode     string `json:"toPlanCode"`
	ToTierRank     int    `json:"toTierRank"`
	AICostPolicyID string `json:"aiCostPolicyId"`
	OccurredAt     string `json:"occurredAt"`
	EffectiveAt    string `json:"effectiveAt"`
	// EffectiveTo is empty for a plan with no end date (free, lifetime). The
	// receiver needs it to compute the start of a renewal period.
	EffectiveTo string `json:"effectiveTo"`
	// DurationDays is 0 for a plan with no end date. Also new, and equally
	// required: without it the EXTENDED rule cannot be applied at all.
	DurationDays int `json:"durationDays"`
	// SourceEventID is the id of the user_subscription_events row, and it is
	// the receiver's only duplicate key. A version-1 payload has none, so it
	// cannot be de-duplicated and is refused.
	SourceEventID string `json:"sourceEventId"`
	// SourceDepositTransactionID is empty for a grant that did not come from a
	// payment (administrator grant, default provisioning, revocation).
	SourceDepositTransactionID string `json:"sourceDepositTransactionId"`
	// FromPlanCode is empty the first time an account ever gets a plan.
	FromPlanCode string `json:"fromPlanCode"`
}
