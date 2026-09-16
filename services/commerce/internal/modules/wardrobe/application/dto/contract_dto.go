package dto

import (
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/wardrobestatus"

	"github.com/google/uuid"
)

// WardrobeItemContractDTO represents the pure data container of a wardrobe item
// for inter-module communication via IWardrobeContract.
// It contains ONLY properties owned by the Wardrobe domain plus foreign key references.
// Cross-module entity pointers (e.g. FashionItem, BrandItem) MUST NEVER be placed here.
type WardrobeItemContractDTO struct {
	ID            uuid.UUID                         `json:"id"`
	UserID        uuid.UUID                         `json:"userId"`
	Price         *float64                          `json:"price,omitempty"`
	Status        wardrobestatus.WardrobeItemStatus `json:"status"`
	IsLocked      bool                              `json:"isLocked"`
	ItemContext   string                            `json:"itemContext,omitempty"`
	FashionItemID *uuid.UUID                        `json:"fashionItemId,omitempty"`
	BrandItemID   *uuid.UUID                        `json:"brandItemId,omitempty"`
	CreatedAt     time.Time                         `json:"createdAt"`
	LastUsedAt    *time.Time                        `json:"lastUsedAt,omitempty"`
	IsDeleted     bool                              `json:"isDeleted"`
	TaskID        *uuid.UUID                        `json:"taskId,omitempty"`
}
