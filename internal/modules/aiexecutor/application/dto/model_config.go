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

type PricingConfigDTO = AIModelConfigDTO
