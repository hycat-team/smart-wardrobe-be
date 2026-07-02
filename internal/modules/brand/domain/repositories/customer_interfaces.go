package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandcustomerstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type BrandCustomerFilter struct {
	BrandID uuid.UUID
	Status  *brandcustomerstatus.BrandCustomerStatus
	Query   *string
	Page    int
	Limit   int
}

type BrandCustomerListResult struct {
	Customers  []*entities.BrandCustomer
	TotalCount int64
}

type IBrandCustomerRepository interface {
	shared_repos.IGenericRepository[entities.BrandCustomer, uuid.UUID]
	GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandCustomer, error)
	GetByBrandAndPhoneHash(ctx context.Context, brandID uuid.UUID, phoneHash string) (*entities.BrandCustomer, error)
	GetByBrandAndExternalCode(ctx context.Context, brandID uuid.UUID, externalCustomerCode string) (*entities.BrandCustomer, error)
	GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandCustomer, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.BrandCustomer, error)
	GetByBrandIDPaginated(ctx context.Context, filter BrandCustomerFilter) (*BrandCustomerListResult, error)
	CountByBrandID(ctx context.Context, brandID uuid.UUID) (int64, error)
	CountByBrandIDs(ctx context.Context, brandIDs []uuid.UUID) (map[uuid.UUID]int64, error)
}

type IBrandCustomerClaimRepository interface {
	shared_repos.IGenericRepository[entities.BrandCustomerClaim, uuid.UUID]
	GetByTokenHash(ctx context.Context, tokenHash string) (*entities.BrandCustomerClaim, error)
	GetActiveByCustomerID(ctx context.Context, brandCustomerID uuid.UUID, now time.Time) ([]*entities.BrandCustomerClaim, error)
	GetByCustomerID(ctx context.Context, brandCustomerID uuid.UUID) ([]*entities.BrandCustomerClaim, error)
}
