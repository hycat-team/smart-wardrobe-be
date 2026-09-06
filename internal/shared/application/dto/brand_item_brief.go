package dto

import "github.com/google/uuid"

// BrandItemBriefRes is a flattened projection of
// brand_dto.BrandItemContractDTO (drops Description/CreatedAt/UpdatedAt/TaskID)
// shared by wardrobe item responses and the fashion recommendation output.
// It lives in shared because both modules embed it while the source of truth is
// the brand contract — keep JSON fields in sync with the source.
type BrandItemBriefRes struct {
	ID          uuid.UUID `json:"id"`
	BrandID     uuid.UUID `json:"brandId"`
	BrandName   string    `json:"brandName"`
	ProductCode *string   `json:"productCode,omitempty"`
	ItemType    string    `json:"itemType"`
	Name        string    `json:"name"`
	Price       *float64  `json:"price,omitempty"`
	Status      string    `json:"status"`
}
