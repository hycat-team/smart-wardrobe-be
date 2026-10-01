package dto

import "github.com/google/uuid"

// ---------------------------------------------------------------------------
// Admin gate-2 API: ai_cost_policies / ai_cost_policy_operations
// (024 FR-004, FR-004b, contracts/admin-cost-policy-api.md)
// ---------------------------------------------------------------------------
//
// Gate 1 is subscription_plans.ai_outfit_daily_quota / ai_chat_daily_quota —
// the numbers the customer sees, edited through the Account service.
// Gate 2 is the table below — the numbers the AI engine actually enforces.
// The two are in different schemas owned by different services with no foreign
// key between them, so they can and do drift apart. See data-model.md §1b.

// CostPolicyActor is the administrator context of one request. It is passed in
// explicitly rather than read from a gin.Context so the use case stays free of
// the presentation layer (constitution §IV). It lives in the dto package
// because the use case interface references it, and importing the use case
// implementation from the interface package would create a cycle.
type CostPolicyActor struct {
	UserID    uuid.UUID
	RequestIP string
}

// AICostPolicyOperationDTO is one operation row of a cost policy.
//
// MaxPaidAttemptsPerDay is a pointer: NULL means "this policy does not use a
// per-day attempt limit", which is a deliberate value for a strict policy, not
// missing data (FR-005e). In strict mode the column is never read at all, so
// NULL is the honest way to say so.
type AICostPolicyOperationDTO struct {
	ID                    uuid.UUID `json:"id"`
	Operation             string    `json:"operation"`
	NormalRoute           string    `json:"normalRoute"`
	ReducedRoute          string    `json:"reducedRoute"`
	FreeRoute             string    `json:"freeRoute"`
	MaxPaidAttemptsPerDay *int      `json:"maxPaidAttemptsPerDay"`
	PaidFallbackEnabled   bool      `json:"paidFallbackEnabled"`
	IsEnabled             bool      `json:"isEnabled"`
}

// AICostPolicyDTO is one cost policy together with its four operations, so an
// administrator can inspect gate 2 without opening a SQL client.
type AICostPolicyDTO struct {
	ID               uuid.UUID                  `json:"id"`
	Code             string                     `json:"code"`
	Name             string                     `json:"name"`
	Version          int64                      `json:"version"`
	EnforcementMode  string                     `json:"enforcementMode"`
	PeriodDays       int                        `json:"periodDays"`
	HardCostMicroVND *int64                     `json:"hardCostMicroVnd"`
	IsActive         bool                       `json:"isActive"`
	Operations       []AICostPolicyOperationDTO `json:"operations"`
}

// UpdateCostPolicyOperationReq is the body of
// PUT /admin/ai/cost-policies/:id/operations/:operation.
//
// The intent to set NULL is expressed through ClearMaxPaidAttempts rather than
// a JSON null: with a plain *int, `{"maxPaidAttemptsPerDay": null}` and an
// omitted field are indistinguishable after unmarshalling, and the two mean
// opposite things. Sending both a value and the flag is a contradiction and is
// rejected with 400 (contracts/admin-cost-policy-api.md §6).
type UpdateCostPolicyOperationReq struct {
	MaxPaidAttemptsPerDay *int `json:"maxPaidAttemptsPerDay,omitempty" binding:"omitempty,min=0" label:"giới hạn lượt trả phí mỗi ngày"`
	ClearMaxPaidAttempts  bool `json:"clearMaxPaidAttempts,omitempty" label:"đặt giới hạn lượt thành NULL"`
}

// CostPolicyReplayWarning is the mandatory warning attached to a successful
// update (FR-004b(2)).
//
// maxPaidAttemptsPerDay belongs to the entitlement group, which is frozen into
// the grant snapshot at grant time (FR-029b). Editing the column therefore
// changes nothing for accounts that already hold a grant row, and openGrantCount
// — the rows with effective_to IS NULL — never refresh on their own at all.
type CostPolicyReplayWarning struct {
	ReplayRequired   bool   `json:"replayRequired"`
	FrozenGrantCount int64  `json:"frozenGrantCount"`
	OpenGrantCount   int64  `json:"openGrantCount"`
	Message          string `json:"message"`
}

// UpdateCostPolicyOperationResp is the body returned after a successful update.
type UpdateCostPolicyOperationResp struct {
	PolicyCode            string                   `json:"policyCode"`
	Operation             string                   `json:"operation"`
	MaxPaidAttemptsPerDay *int                     `json:"maxPaidAttemptsPerDay"`
	Warning               *CostPolicyReplayWarning `json:"warning,omitempty"`
}
