package persistence

import (
	"context"
	"errors"
	"strings"

	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BrandRepository struct {
	shared_persist.GenericRepository[entities.Brand, uuid.UUID]
}

func NewBrandRepository(db *gorm.DB) repositories.IBrandRepository {
	relations := []string{"CreatedByUser", "ApprovedByUser"}
	return &BrandRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.Brand, uuid.UUID](db, relations),
	}
}

func (r *BrandRepository) GetBySlug(ctx context.Context, slug string) (*entities.Brand, error) {
	var brand entities.Brand
	err := r.GetDB(ctx).Where("slug = ?", slug).First(&brand).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &brand, nil
}

func (r *BrandRepository) GetActive(ctx context.Context) ([]*entities.Brand, error) {
	var brands []*entities.Brand
	err := r.GetDB(ctx).
		Where("status = ?", brandstatus.Active).
		Order("created_at DESC").
		Find(&brands).Error
	return brands, err
}

func (r *BrandRepository) GetActiveFiltered(ctx context.Context, filter repositories.BrandFilter) (*repositories.BrandListResult, error) {
	activeStatus := brandstatus.Active
	filter.Status = &activeStatus
	return r.getBrands(ctx, filter)
}

func (r *BrandRepository) GetBrandsForAdmin(ctx context.Context, filter repositories.BrandFilter) (*repositories.BrandListResult, error) {
	return r.getBrands(ctx, filter)
}

func (r *BrandRepository) getBrands(ctx context.Context, filter repositories.BrandFilter) (*repositories.BrandListResult, error) {
	db := r.GetDB(ctx).Model(&entities.Brand{})

	if filter.Status != nil && *filter.Status != "" {
		db = db.Where("status = ?", *filter.Status)
	}

	if filter.Query != nil && *filter.Query != "" {
		queryStr := "%" + strings.ToLower(*filter.Query) + "%"
		db = db.Where("LOWER(name) LIKE ? OR LOWER(slug) LIKE ?", queryStr, queryStr)
	}

	var totalCount int64
	if err := db.Count(&totalCount).Error; err != nil {
		return nil, err
	}

	var brands []*entities.Brand
	paginationQuery := shared_dto.PaginationQuery{
		Page:  filter.Page,
		Limit: filter.Limit,
	}
	db = shared_persist.ApplyPagination(db, paginationQuery)

	err := db.Order("created_at DESC").Find(&brands).Error
	if err != nil {
		return nil, err
	}

	return &repositories.BrandListResult{
		Brands:     brands,
		TotalCount: totalCount,
	}, nil
}
