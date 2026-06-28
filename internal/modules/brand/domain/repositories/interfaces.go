package repositories

import (
	"context"

	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type IBrandRepository interface {
	shared_repos.IGenericRepository[entities.Brand, uuid.UUID]
	GetBySlug(ctx context.Context, slug string) (*entities.Brand, error)
	GetActive(ctx context.Context) ([]*entities.Brand, error)
}

type IBrandMemberRepository interface {
	shared_repos.IGenericRepository[entities.BrandMember, uuid.UUID]
	GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandMember, error)
	GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandMember, error)
}

type IBrandCustomerRepository interface {
	shared_repos.IGenericRepository[entities.BrandCustomer, uuid.UUID]
	GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandCustomer, error)
	GetByBrandAndPhoneHash(ctx context.Context, brandID uuid.UUID, phoneHash string) (*entities.BrandCustomer, error)
	GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandCustomer, error)
}
