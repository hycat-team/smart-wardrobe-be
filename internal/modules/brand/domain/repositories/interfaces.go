package repositories

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type BrandFilter struct {
	Status *brandstatus.BrandStatus
	Query  *string
	Page   int
	Limit  int
}

type BrandListResult struct {
	Brands     []*entities.Brand
	TotalCount int64
}

type IBrandRepository interface {
	shared_repos.IGenericRepository[entities.Brand, uuid.UUID]
	GetBySlug(ctx context.Context, slug string) (*entities.Brand, error)
	GetActive(ctx context.Context) ([]*entities.Brand, error)
	GetActiveFiltered(ctx context.Context, filter BrandFilter) (*BrandListResult, error)
	GetBrandsForAdmin(ctx context.Context, filter BrandFilter) (*BrandListResult, error)
	GetAdminBrandStats(ctx context.Context, snapshotDate time.Time) (*dto.BrandAdminStatsDTO, error)
	GetBrandAnalyticsStats(ctx context.Context, brandID uuid.UUID, snapshotDate time.Time) (*dto.BrandAnalyticsStatsDTO, error)
}

type IBrandMemberRepository interface {
	shared_repos.IGenericRepository[entities.BrandMember, uuid.UUID]
	GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandMember, error)
	GetByBrandAndUserIDs(ctx context.Context, brandID uuid.UUID, userIDs []uuid.UUID) ([]*entities.BrandMember, error)
	GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandMember, error)
	GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.BrandMember, error)
	BulkCreate(ctx context.Context, members []*entities.BrandMember) error
	BulkUpdate(ctx context.Context, members []*entities.BrandMember) error
}
