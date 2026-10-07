package grant

import (
	"fmt"
	"time"

	aiexecutor_dto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	sharedevent "smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/domain/constants/subscriptioneventtype"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// SupportedEventVersion is the highest contract version this receiver
// understands. A payload above it is refused rather than guessed at: writing a
// grant from a payload whose meaning is unknown produces a record that looks
// valid and is not, which is worse than refusing (FR-020).
const SupportedEventVersion = 2

// PlanChangedEvent is the validated, typed form of the incoming payload.
type PlanChangedEvent struct {
	EventVersion int
	UserID       uuid.UUID
	EventType    subscriptioneventtype.Type
	ToPlanID     uuid.UUID
	ToPlanCode   string
	ToTierRank   int
	AICostPolicy uuid.UUID
	OccurredAt   time.Time
	EffectiveAt  time.Time
	// EffectiveTo is nil for a plan with no end date. It is what makes the
	// renewal-window rule computable at all.
	EffectiveTo *time.Time
	// DurationDays is 0 for a plan with no end date.
	DurationDays int
	// SourceEventID is the receiver's ONLY duplicate key. Without it an event
	// cannot be de-duplicated, so it is refused rather than processed blind.
	SourceEventID              uuid.UUID
	SourceDepositTransactionID *uuid.UUID
	FromPlanCode               string
}

// ParseError is a classified rejection: PermanentError never succeeds on
// redelivery, RetryableError might.
//
// The distinction is the whole point. A missing required field will still be
// missing on the fifth attempt, so redelivering it just burns the delivery
// limit and delays the dead-letter signal. A database blip will not recur, so
// redelivering it is exactly right.
type ParseError struct {
	// Reason is a stable code from the event contract, so the counter can
	// separate causes instead of lumping every rejection together.
	Reason  string
	Message string
	err     error
}

func (e *ParseError) Error() string { return e.Message }
func (e *ParseError) Unwrap() error { return e.err }

// AsEventError converts a ParseError into the transport-level classification the
// broker layer understands.
func (e *ParseError) AsEventError() error {
	if e.isRetryable() {
		return sharedevent.RetryableError{Err: e}
	}
	return sharedevent.PermanentError{Err: e}
}

// isRetryable reports whether redelivery could plausibly change the outcome.
func (e *ParseError) isRetryable() bool {
	return e.Reason == ReasonTransientFailure
}

// Reason codes recorded when a payload is rejected. They are the vocabulary the
// observability counters are keyed by, so they must stay stable.
const (
	ReasonUnsupportedVersion   = "unsupported_event_version"
	ReasonUnknownEventType     = "unknown_event_type"
	ReasonMissingRequiredField = "missing_required_field"
	ReasonMissingSourceEventID = "missing_source_event_id"
	ReasonMalformedTimestamp   = "malformed_timestamp"
)

// ParsePlanChangedPayload validates and converts an incoming payload.
//
// The old implementation discarded every parse error and produced a zero-valued
// event, which then flowed into the grant writer. That is why a malformed or
// incomplete payload failed deep inside a database insert, or — worse —
// succeeded with empty values.
//
// Every rejection carries a reason code, and every rejection is permanent except
// a transient one, so a redelivery loop cannot burn the delivery limit on a
// payload that will never become valid.
//
// `log` receives the warnings this function raises — FR-020's record of an older
// payload version, which is a warning and not a rejection because a version-1
// payload sitting in the queue at deploy time is expected. It is the SERVICE
// logger, passed in rather than built here, so the line reaches the same sink as
// every other warning and can be correlated. A nil logger is tolerated and simply
// drops the warning; the parse result is unaffected either way.
func ParsePlanChangedPayload(log logger.Interface, payload aiexecutor_dto.PlanChangedEventPayload) (*PlanChangedEvent, error) {
	// A missing eventVersion means a version-1 payload, which is the six
	// original fields. Version 0 is the sentinel for "not stated" and takes the
	// older-payload branch, not the newer-payload branch.
	version := payload.EventVersion
	if version == 0 {
		version = 1
	}
	if version > SupportedEventVersion {
		return nil, &ParseError{
			Reason:  ReasonUnsupportedVersion,
			Message: fmt.Sprintf("Phiên bản nội dung sự kiện %d mới hơn mức hệ thống hiểu (%d).", version, SupportedEventVersion),
		}
	}

	// An unknown change type is refused, never routed to a default branch. The
	// old default branch created a grant for every unrecognised type, which is
	// how toggling auto-renew used to revoke and re-grant entitlements.
	if !subscriptioneventtype.IsValid(subscriptioneventtype.Type(payload.EventType)) {
		return nil, &ParseError{
			Reason:  ReasonUnknownEventType,
			Message: fmt.Sprintf("Loại thay đổi gói không hợp lệ: %q.", payload.EventType),
		}
	}

	event := &PlanChangedEvent{
		EventVersion: version,
		EventType:    subscriptioneventtype.Type(payload.EventType),
		ToPlanCode:   payload.ToPlanCode,
		ToTierRank:   payload.ToTierRank,
		DurationDays: payload.DurationDays,
		FromPlanCode: payload.FromPlanCode,
	}

	// toPlanCode is what the grant row records and what an operator reads when
	// investigating. An empty one would produce a grant that names no plan at
	// all, so it is refused here rather than written.
	if payload.ToPlanCode == "" {
		return nil, &ParseError{
			Reason:  ReasonMissingRequiredField,
			Message: "Thiếu toPlanCode.",
		}
	}
	var err error
	if event.UserID, err = uuid.Parse(payload.UserID); err != nil {
		return nil, &ParseError{
			Reason:  ReasonMissingRequiredField,
			Message: "Thiếu hoặc sai định dạng userId.",
			err:     err,
		}
	}
	// aiCostPolicyId is what the grant row points at, and the foreign key to the
	// plan was dropped, so a missing value here cannot be recovered downstream.
	if event.AICostPolicy, err = uuid.Parse(payload.AICostPolicyID); err != nil {
		return nil, &ParseError{
			Reason:  ReasonMissingRequiredField,
			Message: "Thiếu hoặc sai định dạng aiCostPolicyId.",
			err:     err,
		}
	}
	// A timestamp that is absent and one that is present but unparseable are
	// different faults and get different codes: absent means the publisher did
	// not send a required field, unparseable means it sent something that cannot
	// be a timestamp. Lumping them together hides a broken producer behind a
	// "malformed data" label.
	if event.OccurredAt, err = parseRequiredTimestamp(payload.OccurredAt); err != nil {
		return nil, &ParseError{
			Reason:  timestampReason(payload.OccurredAt, ReasonMissingRequiredField),
			Message: "Thiếu hoặc sai định dạng occurredAt.",
			err:     err,
		}
	}
	if event.EffectiveAt, err = parseRequiredTimestamp(payload.EffectiveAt); err != nil {
		return nil, &ParseError{
			Reason:  timestampReason(payload.EffectiveAt, ReasonMissingRequiredField),
			Message: "Thiếu hoặc sai định dạng effectiveAt.",
			err:     err,
		}
	}

	// 024 T137: `toPlanId` is read whenever it is PRESENT, whatever the version,
	// and is then REQUIRED.
	//
	// Reading it unconditionally is what lets a version-1 payload that happens to
	// carry the field still be processed (FR-020: a missing field is "undetermined,
	// record a warning", not "refuse"). Requiring it afterwards is what stops the one
	// that does not carry it.
	//
	// The previous code parsed the field only for version 2 and above, and never
	// checked the result for version 1. `ToPlanID` therefore stayed at its zero
	// value, the NOT NULL `plan_id` column accepted the nil UUID, and a grant was
	// written pointing at a plan that does not exist.
	//
	// 024 T172: this used to add that such a row was "invisible to FR-036's
	// catalogue check, because that check looks a plan up BY id". There is no
	// catalogue check — the AI service looks up the COST POLICY and never the plan,
	// because reading the Account service's plan catalogue would violate constitution
	// §I. So nothing would have caught that row later. The guard below is the only
	// thing standing between a nil plan id and a grant that records nothing usable.
	//
	// FR-024 is why nothing downstream can repair this: deriving the plan id would
	// mean reading the Account service's subscription table.
	if payload.ToPlanID != "" {
		if event.ToPlanID, err = uuid.Parse(payload.ToPlanID); err != nil {
			return nil, &ParseError{
				Reason:  ReasonMissingRequiredField,
				Message: "Thiếu hoặc sai định dạng toPlanId. Cột plan_id là bắt buộc và hệ thống AI không được suy ra từ dữ liệu của dịch vụ Account.",
				err:     err,
			}
		}
	}

	// Fields introduced in version 2. Their absence is a warning, not a
	// rejection - a version-1 payload is legitimately in the queue at deploy
	// time - EXCEPT for sourceEventId, which is the only duplicate key there
	// is. Processing an event that cannot be de-duplicated would create a fresh
	// grant on every redelivery, which is precisely the invariant this feature
	// exists to restore.
	if version >= 2 {
		if payload.EffectiveTo != "" {
			effectiveTo, parseErr := parseTimestamp(payload.EffectiveTo)
			if parseErr != nil {
				return nil, &ParseError{
					Reason:  ReasonMalformedTimestamp,
					Message: "Sai định dạng effectiveTo.",
					err:     parseErr,
				}
			}
			event.EffectiveTo = &effectiveTo
		}
	} else if log != nil {
		// 024 T171: this used to build its own logger inline with `logger.New("test",
		// "", "info", false)`. In production code that means the environment is
		// literally "test", nothing reaches a file, and the line bypasses the logger
		// the service already has — so FR-020's "record a warning" was satisfied in
		// the source and nowhere else. This is the ONLY warning on this branch, and
		// the signal that a publisher is still emitting a version we have moved past.
		//
		// `user_id` and `sourceEventId` are here so an operator who sees it can find
		// the event; without them the warning names a condition nobody can act on.
		log.Warn("plan-changed event uses an older payload version; entitlement fields are unavailable",
			zap.Int("event_version", version),
			zap.String("user_id", payload.UserID),
			zap.String("source_event_id", payload.SourceEventID),
			zap.String("event_type", payload.EventType),
		)
	}

	// FR-022/FR-023: the plan identifier is required, in EVERY version. Refused,
	// not repaired - a version-1 payload in the queue at deploy time is expected and
	// harmless to drop, whereas a grant that points nowhere is not.
	if event.ToPlanID == uuid.Nil {
		return nil, &ParseError{
			Reason:  ReasonMissingRequiredField,
			Message: "Sự kiện không có toPlanId. Trường này là bắt buộc và hệ thống AI không được suy ra từ dữ liệu của dịch vụ Account, nên bản ghi quyền không thể dựng lại.",
		}
	}

	if payload.SourceEventID == "" {
		return nil, &ParseError{
			Reason:  ReasonMissingSourceEventID,
			Message: "Thiếu sourceEventId. Đây là khoá chống xử lý trùng duy nhất, nên sự kiện không thể xử lý an toàn.",
		}
	}
	if event.SourceEventID, err = uuid.Parse(payload.SourceEventID); err != nil {
		return nil, &ParseError{
			Reason:  ReasonMissingSourceEventID,
			Message: "Sai định dạng sourceEventId.",
			err:     err,
		}
	}

	if payload.SourceDepositTransactionID != "" {
		depositID, parseErr := uuid.Parse(payload.SourceDepositTransactionID)
		if parseErr != nil {
			return nil, &ParseError{
				Reason:  ReasonMissingRequiredField,
				Message: "Sai định dạng sourceDepositTransactionId.",
				err:     parseErr,
			}
		}
		event.SourceDepositTransactionID = &depositID
	}

	// 024 T168: a renewal is the one change type whose window is COMPUTED from
	// durationDays, so without it the rule FR-032 states is undefined.
	//
	// EventGrantWindow used to fall through to the ordinary branch when the value
	// was absent or non-positive, which set the start to EffectiveAt and the status to
	// Active. For a renewal that hands the entitlement over the moment the event
	// arrives instead of at "new expiry minus duration" — opening the account early,
	// which grants a period nobody paid for and does it silently. That is the exact
	// outcome the comment inside EventGrantWindow says to avoid.
	//
	// Refused at the parser rather than patched downstream, so nothing is written and
	// the existing `missing_required_field` reason applies: no new reason code, no
	// change to the dead-letter contract.
	//
	// 027: the rule applies only when the event CARRIES an end date. It used to apply
	// unconditionally, which refused a change to an open-ended plan - one with no
	// expiry - where durationDays is legitimately 0 and, more to the point, is never
	// read: EventGrantWindow returns the EffectiveAt-to-nil window before it ever
	// looks at the duration. The unconditional form therefore refused exactly the
	// events it had no reason to, and it is what blocked the grant repair: re-granting
	// the same open-ended plan (the repair path) publishes EXTENDED with
	// durationDays 0, and every one of those was dead-lettered. The guard is still
	// load-bearing for a renewal that has an end date, which is the case T168 was
	// written about; it is only scoped to where the computation actually happens.
	if event.EventType == subscriptioneventtype.EXTENDED && event.EffectiveTo != nil && event.DurationDays <= 0 {
		return nil, &ParseError{
			Reason:  ReasonMissingRequiredField,
			Message: "Sự kiện gia hạn phải có durationDays lớn hơn 0 để xác định kỳ quyền đã trả.",
		}
	}

	return event, nil
}

func parseTimestamp(value string) (time.Time, error) {
	if value == "" {
		// 024 T187: `fmt.Errorf` is KEPT here, which is the exception in the
		// classification note in the errors package. Every caller of this helper
		// discards the text: each one wraps the failure in a ParseError carrying its
		// own Vietnamese Message, and only `errors.Is/As` over the cause survives.
		// Putting an apperror here would attach an HTTP status to an error that
		// never reaches HTTP — the transport decision is the Reason code on the
		// ParseError, and a status code beside it would be a second, wrong answer.
		return time.Time{}, fmt.Errorf("timestamp is empty")
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func parseRequiredTimestamp(value string) (time.Time, error) {
	return parseTimestamp(value)
}

// timestampReason separates "the publisher omitted a required field" from "the
// publisher sent something unparseable".
func timestampReason(value, missingReason string) string {
	if value == "" {
		return missingReason
	}
	return ReasonMalformedTimestamp
}
