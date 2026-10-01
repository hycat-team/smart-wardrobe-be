package mapper

import (
	"errors"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

// planChangedEventPayload is the publisher-side view of the 14-field
// subscription.plan.changed contract.
//
// It is a local type on purpose. Constitution §II forbids sharing a DTO type
// across services — the JSON is the contract, and a shared struct would make a
// field rename on one side compile cleanly on the other while breaking every
// message in flight. The consumer declares its own copy in
// services/ai/dto/aiexecutor; the two must be changed together.
type planChangedEventPayload struct {
	EventVersion               int    `json:"eventVersion"`
	UserID                     string `json:"userId"`
	EventType                  string `json:"eventType"`
	ToPlanID                   string `json:"toPlanId"`
	ToPlanCode                 string `json:"toPlanCode"`
	ToTierRank                 int    `json:"toTierRank"`
	AICostPolicyID             string `json:"aiCostPolicyId"`
	OccurredAt                 string `json:"occurredAt"`
	EffectiveAt                string `json:"effectiveAt"`
	EffectiveTo                string `json:"effectiveTo"`
	DurationDays               int    `json:"durationDays"`
	SourceEventID              string `json:"sourceEventId"`
	SourceDepositTransactionID string `json:"sourceDepositTransactionId"`
	FromPlanCode               string `json:"fromPlanCode"`
}

// CurrentPlanChangedEventVersion is the contract version this publisher emits.
// The consumer refuses anything higher and handles anything lower with the
// missing-field rules, so this is the only place the number lives.
const CurrentPlanChangedEventVersion = 2

// BuildPlanChangedPayload assembles the payload from the history row, the
// subscription and the plan.
//
// Three rules are load-bearing and each exists because getting it wrong is
// invisible until the AI service writes a broken record:
//
//   - eventType is read from the history row rather than passed in, so the
//     sender and receiver can never disagree about what happened (FR-019).
//   - toPlanId comes from plan.ID. ai_policy_grants.plan_id is NOT NULL and its
//     foreign key was dropped, so the receiver has no other way to fill it.
//   - effectiveTo and durationDays are what let the receiver compute the start
//     of a renewal period. Without them a renewal cannot be placed correctly.
//
// The values are read AFTER the history row has been written, because GORM
// assigns the primary key during INSERT and the receiver needs it as its
// duplicate key.
//
// The policy snapshot is deliberately NOT included: it belongs to the AI
// service's own table, and copying a configuration document across a service
// boundary would create a second source of truth for pricing (FR-027).
func BuildPlanChangedPayload(
	event *entities.UserSubscriptionEvent,
	sub *entities.UserSubscription,
	plan *entities.SubscriptionPlan,
) planChangedEventPayload {
	payload := planChangedEventPayload{
		EventVersion:  CurrentPlanChangedEventVersion,
		UserID:        event.UserID.String(),
		EventType:     event.EventType,
		ToTierRank:    derefInt(event.ToTierRank),
		OccurredAt:    event.OccurredAt.UTC().Format(time.RFC3339Nano),
		EffectiveAt:   event.EffectiveAt.UTC().Format(time.RFC3339Nano),
		SourceEventID: event.ID.String(),
		FromPlanCode:  derefString(event.FromPlanCode),
	}

	if plan != nil {
		payload.ToPlanID = plan.ID.String()
		payload.ToPlanCode = plan.Slug
		payload.AICostPolicyID = plan.AICostPolicyID.String()
		if plan.DurationDays != nil {
			payload.DurationDays = *plan.DurationDays
		}
	} else {
		// No plan row means the change moved the account TO the default free
		// plan. The free plan's own row is required to fill these, and without
		// it the receiver cannot write a valid grant — so it is left empty and
		// the receiver rejects the message with a precise reason rather than
		// writing a record with a blank plan id.
		payload.ToPlanCode = derefString(event.ToPlanCode)
	}

	if event.SourceDepositTransactionID != nil {
		payload.SourceDepositTransactionID = event.SourceDepositTransactionID.String()
	}

	// An open-ended plan has no expiry. The field must still be present in the
	// JSON so the receiver can tell "no end date" from "version 1 did not send
	// this field" — those two mean completely different things for the EXTENDED
	// rule.
	if sub != nil && sub.ExpiresAt != nil {
		payload.EffectiveTo = sub.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}

	return payload
}

// ValidatePlanChangedPayload refuses a payload the receiver is guaranteed to reject.
//
// 024 T155. `BuildPlanChangedPayload` cannot fail - it returns a value - so it happily
// produced a message with an EMPTY `toPlanId` when the caller passed no plan row.
// The comment on that branch said the receiver "rejects the message with a precise
// reason rather than writing a record with a blank plan id", and that is true. But
// the sender was the one that could have known, and it published anyway, so FR-012's
// "every change publishes an event" quietly became "every change publishes an event
// nobody can act on" - with no log, no metric and no error anywhere.
//
// The two fields checked here are the ones the AI side requires (FR-022/FR-023, T137):
// a grant row cannot be written without `toPlanId`, and cannot be de-duplicated
// without `sourceEventId`.
//
// Validation lives next to the construction rather than at the twelve call sites, so
// a new caller cannot forget it.
func ValidatePlanChangedPayload(payload planChangedEventPayload) error {
	// Both ids are stringified UUIDs, so "absent" has two spellings: the empty string
	// (no plan row at all) and the nil UUID (a plan struct whose id was never assigned).
	// Only checking one of them lets the other through, and the nil UUID is the more
	// dangerous of the two: it looks like a real identifier all the way into the
	// receiver's grant table.
	if isUnsetUUID(payload.ToPlanID) {
		return errors.New("payload thiếu toPlanId: thay đổi gói không dựng được thông điệp để receiver dùng")
	}
	if isUnsetUUID(payload.SourceEventID) {
		return errors.New("payload thiếu sourceEventId: receiver không thể chống trùng")
	}
	return nil
}

// isUnsetUUID reports whether an id is absent in either spelling.
func isUnsetUUID(value string) bool {
	if value == "" {
		return true
	}
	parsed, err := uuid.Parse(value)
	if err != nil {
		// Not parseable is not usable either. Treating it as "set" would let a
		// malformed id through to the receiver, which rejects it with a reason nobody
		// on the sending side would ever see.
		return true
	}
	return parsed == uuid.Nil
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func derefInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}
