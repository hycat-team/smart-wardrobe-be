package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AIModelConfigDTO struct {
	ID                  uuid.UUID       `json:"id"`
	Provider            string          `json:"provider"`
	Model               string          `json:"model"`
	InputUSDPerMillion  decimal.Decimal `json:"inputUsdPerMillion"`
	OutputUSDPerMillion decimal.Decimal `json:"outputUsdPerMillion"`
	Currency            string          `json:"currency"`
	USDToVND            decimal.Decimal `json:"usdToVnd"`
	Version             string          `json:"version"`
	EffectiveFrom       time.Time       `json:"effectiveFrom"`
	EffectiveTo         *time.Time      `json:"effectiveTo,omitempty"`
}

// PricingConfigDTO re-exports the admin pricing config shape for the pricing
// cache (application <-> infrastructure) — single source of truth stays AIModelConfigDTO.
type PricingConfigDTO = AIModelConfigDTO

type UpdatePricingReq struct {
	InputUSDPerMillion  *decimal.Decimal `json:"inputUsdPerMillion,omitempty" label:"giá input USD / 1M token"`
	OutputUSDPerMillion *decimal.Decimal `json:"outputUsdPerMillion,omitempty" label:"giá output USD / 1M token"`
	USDToVND            *decimal.Decimal `json:"usdToVnd,omitempty" label:"tỉ giá USD sang VND"`
	EffectiveTo         *time.Time       `json:"effectiveTo,omitempty" label:"thời hạn hiệu lực"`
}

// CreateAIGrantReq is gone as of feature 024 (FR-027, FR-035).
//
// The endpoint that used it, POST /api/v1/admin/ai/grants, never returned 201:
// it left policy_snapshot unset on a NOT NULL column and status empty on a
// column with a CHECK constraint, so every call failed at the database. Keeping
// it would have kept a SECOND writer on aiexecutor.ai_policy_grants inside the
// AI service itself — the same class of cross-boundary write this feature exists
// to delete, just moved one service to the left.
//
// The administrator now grants a plan through the Account service, which emits
// the same event every other plan change emits. GET /grants is kept: it is the
// observation path and has no write side.

type AIUsageItemDTO struct {
	ID                 uuid.UUID `json:"id"`
	RequestID          uuid.UUID `json:"requestId"`
	UserID             uuid.UUID `json:"userId"`
	Operation          string    `json:"operation"`
	Status             string    `json:"status"`
	PromptTokens       int64     `json:"promptTokens"`
	OutputTokens       int64     `json:"outputTokens"`
	ThinkingTokens     int64     `json:"thinkingTokens"`
	ActualCostMicroVND int64     `json:"actualCostMicroVnd"`
	Provider           string    `json:"provider"`
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"createdAt"`
}

type AIPolicyGrantDTO struct {
	ID             uuid.UUID             `json:"id"`
	UserID         uuid.UUID             `json:"userId"`
	AICostPolicy   *AICostPolicyBriefRes `json:"aiCostPolicy,omitempty"`
	AICostPolicyID uuid.UUID             `json:"aiCostPolicyId"`
	EffectiveFrom  time.Time             `json:"effectiveFrom"`
	EffectiveTo    *time.Time            `json:"effectiveTo,omitempty"`
	CreatedAt      time.Time             `json:"createdAt"`
}

type AICostPolicyBriefRes struct {
	ID      uuid.UUID `json:"id"`
	Code    string    `json:"code"`
	Name    string    `json:"name"`
	Version int64     `json:"version"`
}
