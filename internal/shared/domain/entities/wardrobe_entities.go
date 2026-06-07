package entities

import (
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"
	"smart-wardrobe-be/internal/shared/domain/constants/messagesender"
	"smart-wardrobe-be/internal/shared/domain/constants/outfitstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"

	"github.com/google/uuid"
)

type ConversationalContext struct {
	AuditableEntity
	UserID         uuid.UUID `gorm:"type:uuid;not null"`
	User           *User     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Title          string    `gorm:"type:varchar(255);not null"`
	ContextSummary string    `gorm:"type:text;default:''"`
	IsArchived     bool      `gorm:"type:boolean;not null;default:false"`
}

type Message struct {
	BaseEntity
	ContextID uuid.UUID                   `gorm:"type:uuid;not null"`
	Context   *ConversationalContext      `gorm:"foreignKey:ContextID;constraint:OnDelete:CASCADE"`
	Sender    messagesender.MessageSender `gorm:"type:varchar(50);not null"` // 'user' or 'ai'
	Content   string                      `gorm:"type:text;not null"`
}

type Category struct {
	AuditableEntity
	Name string `gorm:"type:varchar(100);uniqueIndex;not null"`
	Slug string `gorm:"type:varchar(100);uniqueIndex;not null"`
}

type WardrobeItem struct {
	AuditableEntity
	SoftDeleteEntity
	UserID        uuid.UUID                         `gorm:"type:uuid;not null"`
	User          *User                             `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	CategoryID    *uuid.UUID                        `gorm:"type:uuid"`
	Category      *Category                         `gorm:"foreignKey:CategoryID;constraint:OnDelete:RESTRICT"`
	ImageUrl      string                            `gorm:"type:varchar(500);not null"`
	ImagePublicID string                            `gorm:"type:varchar(255);not null"`
	Color         *string                           `gorm:"type:varchar(50)"`
	Style         *string                           `gorm:"type:varchar(100)"`
	Material      *string                           `gorm:"type:varchar(100)"`
	Pattern       *string                           `gorm:"type:varchar(100)"`
	Fit           *string                           `gorm:"type:varchar(50)"`
	Seasonality   *string                           `gorm:"type:varchar(100)"`
	Description   *string                           `gorm:"type:text"`
	Price         *float64                          `gorm:"type:decimal(12,2)"`
	Status        wardrobestatus.WardrobeItemStatus `gorm:"type:smallint;not null;default:0"` // 'in_wardrobe', 'selling', 'sold'
	ItemType      itemtype.ItemType                 `gorm:"type:smallint;not null;default:0"` // 0: UserItem, 1: SystemCatalogItem
	Embedding     Vector                            `gorm:"type:vector(768)"`
	LastUsedAt    *time.Time                        `gorm:"type:timestamp with time zone"`
}

type Outfit struct {
	AuditableEntity
	SoftDeleteEntity
	UserID        uuid.UUID                 `gorm:"type:uuid;not null"`
	User          *User                     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Name          string                    `gorm:"type:varchar(255);not null"`
	Description   *string                   `gorm:"type:text"`
	CoverImageUrl *string                   `gorm:"type:varchar(500)"`
	CoverPublicID *string                   `gorm:"type:varchar(255)"`
	Status        outfitstatus.OutfitStatus `gorm:"type:smallint;not null;default:1"` // 1: Active, 0: Draft
}

type OutfitItem struct {
	OutfitID   uuid.UUID     `gorm:"type:uuid;primaryKey"`
	Outfit     *Outfit       `gorm:"foreignKey:OutfitID;constraint:OnDelete:CASCADE"`
	ItemID     uuid.UUID     `gorm:"type:uuid;primaryKey"`
	Wardrobe   *WardrobeItem `gorm:"foreignKey:ItemID;constraint:OnDelete:CASCADE"`
	PositionX  float64       `gorm:"type:double precision;not null;default:0.0"`
	PositionY  float64       `gorm:"type:double precision;not null;default:0.0"`
	Scale      float64       `gorm:"type:double precision;not null;default:1.0"`
	LayerOrder int16         `gorm:"type:smallint;not null;default:1"`
	CreatedAt  time.Time     `gorm:"type:timestamp with time zone;not null;default:now()"`
	UpdatedAt  time.Time     `gorm:"type:timestamp with time zone;not null;default:now()"`
}
