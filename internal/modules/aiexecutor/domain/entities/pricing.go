package entities

import (
	"time"

	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aimodel"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aiprovider"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AIPricingConfig struct {
	AuditableEntity
	Provider            aiprovider.Provider `gorm:"type:varchar(50);not null"`
	Model               aimodel.Model       `gorm:"type:varchar(100);not null"`
	InputUSDPerMillion  decimal.Decimal     `gorm:"type:numeric(20,8);not null"`
	OutputUSDPerMillion decimal.Decimal     `gorm:"type:numeric(20,8);not null"`
	Currency            string              `gorm:"type:varchar(10);not null;default:'VND'"`
	USDToVND            decimal.Decimal     `gorm:"type:numeric(20,4);not null"`
	Version             string              `gorm:"type:varchar(50);not null;default:'v1'"`
	EffectiveFrom       time.Time           `gorm:"type:timestamptz;not null;default:now()"`
	EffectiveTo         *time.Time          `gorm:"type:timestamptz"`
	CreatedBy           *uuid.UUID          `gorm:"type:uuid"`
}
