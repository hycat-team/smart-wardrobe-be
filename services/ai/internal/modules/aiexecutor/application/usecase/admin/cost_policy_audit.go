package admin

import (
	"strconv"

	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/shared/domain/constants/adminaudit"
)

// adminauditAICostPolicyUpdate is re-exported under a local alias so the audit
// row and the outbox payload below cannot drift onto two different literals.
// action/targetType are plain strings all the way down to
// shared.admin_audit_logs, so a typo produces no error at all — it just writes
// a value that is absent from the catalogue and that no dashboard can group by.
const adminauditAICostPolicyUpdate = adminaudit.AICostPolicyUpdate

const adminauditAICostPolicyTarget = adminaudit.AICostPolicy

// costPolicyAuditEventPayload is the body published on admin.audit.log.
//
// The shape must match services/account/dto/admin.RecordAuditLogInput field for
// field: the existing AuditLogConsumerWorker unmarshals straight into that
// struct, and a mismatched payload is a PermanentError that lands in the
// dead-letter queue with no retry (FR-004d).
type costPolicyAuditEventPayload struct {
	ActorUserID string `json:"actorUserId"`
	Action      string `json:"action"`
	TargetType  string `json:"targetType"`
	TargetID    any    `json:"targetId"`
	RequestIP   string `json:"requestIp"`
	Payload     any    `json:"payload"`
}

// costPolicyChangePayload carries which field moved from which value to which,
// so the central audit screen can render a diff instead of only "something
// changed".
type costPolicyChangePayload struct {
	PolicyCode string  `json:"policyCode"`
	Operation  string  `json:"operation"`
	FieldName  string  `json:"fieldName"`
	OldValue   *string `json:"oldValue"`
	NewValue   *string `json:"newValue"`
}

func buildCostPolicyAuditEvent(
	policy *aiexecutor_entities.AICostPolicy,
	op *aiexecutor_entities.AICostPolicyOperation,
	oldValue, newValue *int,
	actor CostPolicyActor,
) costPolicyAuditEventPayload {
	return costPolicyAuditEventPayload{
		ActorUserID: actor.UserID.String(),
		Action:      string(adminauditAICostPolicyUpdate),
		TargetType:  string(adminauditAICostPolicyTarget),
		TargetID:    nil,
		RequestIP:   actor.RequestIP,
		Payload: costPolicyChangePayload{
			PolicyCode: policy.Code,
			Operation:  string(op.Operation),
			FieldName:  costPolicyFieldMaxPaidAttemptsPerDay,
			OldValue:   formatMaxPaidAttempts(oldValue),
			NewValue:   formatMaxPaidAttempts(newValue),
		},
	}
}

// formatMaxPaidAttempts renders the limit as a string, mapping NULL to nil
// rather than to "". Collapsing the two would make a later investigation unable
// to tell "the field was cleared" from "the field was set to an empty value" —
// an audit trail that cannot answer that question is worthless.
func formatMaxPaidAttempts(value *int) *string {
	if value == nil {
		return nil
	}
	formatted := strconv.Itoa(*value)
	return &formatted
}
