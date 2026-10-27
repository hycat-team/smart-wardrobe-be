package entities

import (
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/wardrobestatus"

	"github.com/google/uuid"
)

type WardrobeItem struct {
	AuditableEntity
	SoftDeleteEntity
	UserID        uuid.UUID                         `gorm:"type:uuid;not null"`
	FashionItemID *uuid.UUID                        `gorm:"column:fashion_item_id;type:uuid"`
	Price         *float64                          `gorm:"column:purchase_price;type:decimal(12,2)"`
	Status        wardrobestatus.WardrobeItemStatus `gorm:"type:smallint;not null;default:0"`
	ItemType      itemtype.ItemType                 `gorm:"type:smallint;not null;default:0"`
	LastUsedAt    *time.Time                        `gorm:"type:timestamp with time zone"`
	// LastRecommendedAt is the last time this item was returned by a
	// recommendation (feature 033). It feeds the recency rotation only; it is
	// updated best-effort and is never backfilled.
	LastRecommendedAt *time.Time `gorm:"column:last_recommended_at;type:timestamp with time zone"`
	TaskID            *uuid.UUID `gorm:"column:task_id;type:uuid"`
}
