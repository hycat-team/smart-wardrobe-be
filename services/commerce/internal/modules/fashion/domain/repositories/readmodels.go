package repositories

import "github.com/google/uuid"

// DashboardFashionItemCategoryDTO is a repository read model owned by the
// domain layer (feature 017); application/dto aliases it for compatibility.
type DashboardFashionItemCategoryDTO struct {
	FashionItemID uuid.UUID  `json:"fashionItemId"`
	CategoryID    *uuid.UUID `json:"categoryId,omitempty"`
}
