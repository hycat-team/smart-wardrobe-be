package entities

import (
	"time"

	"github.com/google/uuid"
)

type FashionItem struct {
	AuditableEntity
	CategoryID              *uuid.UUID `gorm:"type:uuid"`
	Category                *Category  `gorm:"foreignKey:CategoryID;constraint:OnDelete:RESTRICT"`
	ImageUrl                string     `gorm:"type:varchar(500);not null"`
	ImagePublicID           string     `gorm:"type:varchar(255);not null"`
	Color                   *string    `gorm:"type:varchar(50)"`
	ColorHex                *string    `gorm:"column:color_hex;type:varchar(7)"`
	ColorHue                *float64   `gorm:"column:color_hue;type:double precision"`
	ColorSaturation         *float64   `gorm:"column:color_saturation;type:double precision"`
	ColorLightness          *float64   `gorm:"column:color_lightness;type:double precision"`
	Style                   *string    `gorm:"type:varchar(100)"`
	Material                *string    `gorm:"type:varchar(100)"`
	Pattern                 *string    `gorm:"type:varchar(100)"`
	Fit                     *string    `gorm:"type:varchar(50)"`
	Seasonality             *string    `gorm:"type:varchar(100)"`
	Description             *string    `gorm:"type:text"`
	Embedding               Vector     `gorm:"type:vector(768)"`
	ProcessingRetryCount    int        `gorm:"type:int;not null;default:0"`
	ProcessingVersion       int        `gorm:"type:int;not null;default:0"`
	ProcessingStartedAt     *time.Time `gorm:"type:timestamp with time zone"`
	LastProcessingAttemptAt *time.Time `gorm:"type:timestamp with time zone"`
	ProcessingErrorReason   *string    `gorm:"type:text"`
	ReviewReason            *string    `gorm:"type:varchar(100)"`
}
