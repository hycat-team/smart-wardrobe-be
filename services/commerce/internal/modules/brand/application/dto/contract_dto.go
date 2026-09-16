package dto

import (
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/branditem/branditemtype"

	"github.com/google/uuid"
)

// BrandItemContractDTO represents the pure data container of a brand item
// for inter-module communication via IBrandContract.
// It contains ONLY properties owned by the Brand domain plus foreign key references.
// Cross-module entity pointers (e.g. FashionItem) MUST NEVER be placed here.
type BrandItemContractDTO struct {
	ID            uuid.UUID                       `json:"id"`
	BrandID       uuid.UUID                       `json:"brandId"`
	BrandName     string                          `json:"brandName,omitempty"`
	FashionItemID uuid.UUID                       `json:"fashionItemId"`
	ProductCode   *string                         `json:"productCode,omitempty"`
	Name          string                          `json:"name"`
	Description   *string                         `json:"description,omitempty"`
	Price         *float64                        `json:"price,omitempty"`
	ItemType      branditemtype.BrandItemType     `json:"itemType"`
	Status        branditemstatus.BrandItemStatus `json:"status"`
	CreatedAt     time.Time                       `json:"createdAt"`
	UpdatedAt     time.Time                       `json:"updatedAt"`
	TaskID        *uuid.UUID                      `json:"taskId,omitempty"`
}
