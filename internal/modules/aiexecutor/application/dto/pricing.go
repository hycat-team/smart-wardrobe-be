package dto

import (
	"time"

	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aimodel"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aiprovider"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PricingConfigDTO struct {
	ID                  uuid.UUID           `json:"id"`
	Provider            aiprovider.Provider `json:"provider"`
	Model               aimodel.Model       `json:"model"`
	InputUSDPerMillion  decimal.Decimal     `json:"inputUsdPerMillion"`
	OutputUSDPerMillion decimal.Decimal     `json:"outputUsdPerMillion"`
	Currency            string              `json:"currency"`
	USDToVND            decimal.Decimal     `json:"usdToVnd"`
	Version             string              `json:"version"`
	EffectiveFrom       time.Time           `json:"effectiveFrom"`
	EffectiveTo         *time.Time          `json:"effectiveTo,omitempty"`
}
