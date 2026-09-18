package contract

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"

	"github.com/google/uuid"
)

type ICategoryContract interface {
	GetCategory(ctx context.Context, id uuid.UUID) (*dto.CategoryContractDTO, error)
}
