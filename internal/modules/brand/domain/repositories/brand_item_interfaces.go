package repositories

import (
	"context"

	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type IBrandItemRepository interface {
	shared_repos.IGenericRepository[entities.BrandItem, uuid.UUID]
	GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandItem, error)
	GetByBrandIDs(ctx context.Context, brandIDs []uuid.UUID) ([]*entities.BrandItem, error)
	GetByProductCode(ctx context.Context, brandID uuid.UUID, code string) (*entities.BrandItem, error)
	GetByFashionItemID(ctx context.Context, fashionItemID uuid.UUID) (*entities.BrandItem, error)
}

type IDigitalSampleResponseRepository interface {
	shared_repos.IGenericRepository[entities.DigitalSampleResponse, uuid.UUID]
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.DigitalSampleResponse, error)
	GetByBrandItemID(ctx context.Context, brandItemID uuid.UUID) ([]*entities.DigitalSampleResponse, error)
}
