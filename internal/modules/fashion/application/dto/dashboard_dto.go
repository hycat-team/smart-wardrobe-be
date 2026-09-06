package dto

import (
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"

	"github.com/google/uuid"
)

// DashboardCategoryDTO is the category slice served by the fashion dashboard reader.
type DashboardCategoryDTO struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// DashboardFashionItemCategoryDTO re-exports the repository read model
// consumed by the dashboard module via IFashionDashboardReader.
type DashboardFashionItemCategoryDTO = repositories.DashboardFashionItemCategoryDTO
