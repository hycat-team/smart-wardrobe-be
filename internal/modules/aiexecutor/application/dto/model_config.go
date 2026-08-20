package dto

import (
	"time"

	shared_dto "smart-wardrobe-be/internal/shared/application/dto"

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

type PricingConfigDTO = AIModelConfigDTO

type UpdatePricingReq struct {
	InputUSDPerMillion  *decimal.Decimal `json:"inputUsdPerMillion,omitempty" label:"giá input USD / 1M token"`
	OutputUSDPerMillion *decimal.Decimal `json:"outputUsdPerMillion,omitempty" label:"giá output USD / 1M token"`
	USDToVND            *decimal.Decimal `json:"usdToVnd,omitempty" label:"tỉ giá USD sang VND"`
	EffectiveTo         *time.Time       `json:"effectiveTo,omitempty" label:"thời hạn hiệu lực"`
}

type CreateAIGrantReq struct {
	UserID         uuid.UUID  `json:"userId" binding:"required" label:"mã người dùng"`
	AICostPolicyID uuid.UUID  `json:"aiCostPolicyId" binding:"required" label:"chính sách AI chi phí"`
	EffectiveTo    *time.Time `json:"effectiveTo,omitempty" label:"thời hạn hiệu lực"`
}

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
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"userId"`
	AICostPolicyID uuid.UUID  `json:"aiCostPolicyId"`
	EffectiveFrom  time.Time  `json:"effectiveFrom"`
	EffectiveTo    *time.Time `json:"effectiveTo,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}

type AIUsageListRes = shared_dto.PaginationResult[*AIUsageItemDTO]
