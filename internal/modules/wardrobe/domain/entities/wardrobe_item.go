package entities

import (
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/itemtype"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/wardrobestatus"

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
}
