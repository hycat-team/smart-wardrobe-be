package entities

import (
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/itemtype"
	"smart-wardrobe-be/internal/shared/domain/constants/messagesender"
	"smart-wardrobe-be/internal/shared/domain/constants/outfititemcontext"
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
	ContextID    uuid.UUID                   `gorm:"type:uuid;not null"`
	Context      *ConversationalContext      `gorm:"foreignKey:ContextID;constraint:OnDelete:CASCADE"`
	Sender       messagesender.MessageSender `gorm:"type:varchar(50);not null"` // 'user' or 'ai'
	Content      string                      `gorm:"type:text;not null"`
	IsSummarized bool                        `gorm:"column:is_summarized;type:boolean;not null;default:false"`
}

type Category struct {
	AuditableEntity
	Name      string `gorm:"type:varchar(100);uniqueIndex;not null"`
	Slug      string `gorm:"type:varchar(100);uniqueIndex;not null"`
	SortOrder int    `gorm:"column:sort_order;type:int;not null;default:0"`
}

type WardrobeItem struct {
	AuditableEntity
	SoftDeleteEntity
	UserID        uuid.UUID                         `gorm:"type:uuid;not null"`
	User          *User                             `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	FashionItemID uuid.UUID                         `gorm:"column:fashion_item_id;type:uuid;not null"`
	FashionItem   *FashionItem                      `gorm:"foreignKey:FashionItemID;constraint:OnDelete:RESTRICT"`
	Price         *float64                          `gorm:"column:purchase_price;type:decimal(12,2)"`
	Status        wardrobestatus.WardrobeItemStatus `gorm:"type:smallint;not null;default:0"` // 'in_wardrobe', 'selling', 'sold'
	ItemType      itemtype.ItemType                 `gorm:"type:smallint;not null;default:0"` // 0: UserItem, 1: SystemCatalogItem
	LastUsedAt    *time.Time                        `gorm:"type:timestamp with time zone"`
}

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
	Items         []*OutfitItem             `gorm:"foreignKey:OutfitID;constraint:OnDelete:CASCADE"`
}

type OutfitItem struct {
	OutfitID      uuid.UUID                           `gorm:"type:uuid;primaryKey"`
	Outfit        *Outfit                             `gorm:"foreignKey:OutfitID;constraint:OnDelete:CASCADE"`
	FashionItemID uuid.UUID                           `gorm:"type:uuid;primaryKey"`
	FashionItem   *FashionItem                        `gorm:"foreignKey:FashionItemID;constraint:OnDelete:RESTRICT"`
	ItemContext   outfititemcontext.OutfitItemContext `gorm:"type:varchar(50);primaryKey;not null"`
	WardrobeItem  *WardrobeItem                       `gorm:"foreignKey:FashionItemID;references:FashionItemID"`
	PositionX     float64                             `gorm:"type:double precision;not null;default:0.0"`
	PositionY     float64                             `gorm:"type:double precision;not null;default:0.0"`
	Scale         float64                             `gorm:"type:double precision;not null;default:1.0"`
	LayerOrder    int16                               `gorm:"type:smallint;not null;default:1"`
	CreatedAt     time.Time                           `gorm:"type:timestamp with time zone;not null;default:now()"`
	UpdatedAt     time.Time                           `gorm:"type:timestamp with time zone;not null;default:now()"`
}
