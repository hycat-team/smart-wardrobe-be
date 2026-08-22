package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/fashion/application/dto"

	"github.com/google/uuid"
)

type ICategoryContract interface {
	GetCategory(ctx context.Context, id uuid.UUID) (*dto.CategoryRes, error)
}
