package entities

import (
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/shared/currency"

	
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ProviderPaymentEvent struct {
	AuditableEntity
	Provider          string            `gorm:"type:varchar(50);not null;uniqueIndex:ux_provider_reference,priority:1"`
	ProviderReference string            `gorm:"type:varchar(255);not null;uniqueIndex:ux_provider_reference,priority:2"`
	EventCode         string            `gorm:"type:varchar(100);not null"`
	OrderCode         int64             `gorm:"type:bigint;not null"`
	PaymentLinkID     string            `gorm:"type:varchar(255)"`
	Amount            decimal.Decimal   `gorm:"type:numeric(12,2);not null"`
	Currency          currency.Currency `gorm:"type:varchar(10);not null"`
}

type ProviderWebhookInbox struct {
	AuditableEntity
	Provider             string            `gorm:"type:varchar(50);not null"`
	ProviderReference    *string           `gorm:"type:varchar(255)"`
	EventCode            string            `gorm:"type:varchar(100);not null"`
	OrderCode            int64             `gorm:"type:bigint;not null"`
	PaymentLinkID        *string           `gorm:"type:varchar(255)"`
	Amount               decimal.Decimal   `gorm:"type:numeric(12,2);not null"`
	Currency             currency.Currency `gorm:"type:varchar(10);not null"`
	CanonicalPayloadHash string            `gorm:"type:varchar(64);not null"`
	RawPayload           JSONDocument `gorm:"type:jsonb;not null"`
	ProcessingStatus     string            `gorm:"type:varchar(50);not null"`
	ProcessingAttempts   int               `gorm:"type:int;not null;default:0"`
	NextProcessingAt     *time.Time        `gorm:"type:timestamp with time zone"`
	ProcessingToken      *uuid.UUID        `gorm:"type:uuid"`
	ProcessingLeaseUntil *time.Time        `gorm:"type:timestamp with time zone"`
	ProcessingError      *string           `gorm:"type:text"`
	ReceivedAt           time.Time         `gorm:"type:timestamp with time zone;not null;default:now()"`
	ProcessedAt          *time.Time        `gorm:"type:timestamp with time zone"`
}
