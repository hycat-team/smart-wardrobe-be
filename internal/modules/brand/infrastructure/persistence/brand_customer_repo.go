package persistence

import (
	"context"
	"errors"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BrandCustomerRepository struct {
	shared_persist.GenericRepository[entities.BrandCustomer, uuid.UUID]
}

func NewBrandCustomerRepository(db *gorm.DB) repositories.IBrandCustomerRepository {
	return &BrandCustomerRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.BrandCustomer, uuid.UUID](
			db,
			[]string{"User", "Brand", "LoyaltyAccount", "LoyaltyAccount.CurrentTier"},
		),
	}
}

func (r *BrandCustomerRepository) GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandCustomer, error) {
	var customer entities.BrandCustomer
	err := r.GetQueryWithPreload(ctx).
		Where("brand_id = ? AND user_id = ?", brandID, userID).
		First(&customer).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &customer, nil
}

func (r *BrandCustomerRepository) GetByBrandAndPhoneHash(ctx context.Context, brandID uuid.UUID, phoneHash string) (*entities.BrandCustomer, error) {
	var customer entities.BrandCustomer
	err := r.GetQueryWithPreload(ctx).
		Where("brand_id = ? AND phone_hash = ?", brandID, phoneHash).
		First(&customer).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &customer, nil
}

func (r *BrandCustomerRepository) GetByBrandAndExternalCode(ctx context.Context, brandID uuid.UUID, externalCustomerCode string) (*entities.BrandCustomer, error) {
	var customer entities.BrandCustomer
	err := r.GetQueryWithPreload(ctx).
		Where("brand_id = ? AND external_customer_code = ?", brandID, externalCustomerCode).
		First(&customer).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &customer, nil
}

func (r *BrandCustomerRepository) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandCustomer, error) {
	var customers []*entities.BrandCustomer
	err := r.GetQueryWithPreload(ctx).
		Where("brand_id = ?", brandID).
		Order("created_at DESC").
		Find(&customers).Error
	return customers, err
}

func (r *BrandCustomerRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.BrandCustomer, error) {
	var customers []*entities.BrandCustomer
	err := r.GetQueryWithPreload(ctx).
		Preload("Brand").
		Where("user_id = ?", userID).
		Find(&customers).Error
	return customers, err
}

func (r *BrandCustomerRepository) GetByBrandIDPaginated(ctx context.Context, filter repositories.BrandCustomerFilter) (*repositories.BrandCustomerListResult, error) {
	db := r.GetQueryWithPreload(ctx).Where("brand_id = ?", filter.BrandID)

	if filter.Status != nil && *filter.Status != "" {
		db = db.Where("status = ?", *filter.Status)
	}

	if filter.Query != nil && *filter.Query != "" {
		queryStr := "%" + strings.ToLower(*filter.Query) + "%"
		db = db.Where("LOWER(customer_name) LIKE ? OR phone_e164 LIKE ? OR LOWER(external_customer_code) LIKE ?", queryStr, queryStr, queryStr)
	}

	var totalCount int64
	if err := db.Model(&entities.BrandCustomer{}).Count(&totalCount).Error; err != nil {
		return nil, err
	}

	paginationQuery := shared_dto.PaginationQuery{
		Page:  filter.Page,
		Limit: filter.Limit,
	}
	db = shared_persist.ApplyPagination(db, paginationQuery)

	var customers []*entities.BrandCustomer
	if err := db.Order("created_at DESC").Find(&customers).Error; err != nil {
		return nil, err
	}

	return &repositories.BrandCustomerListResult{
		Customers:  customers,
		TotalCount: totalCount,
	}, nil
}

func (r *BrandCustomerRepository) CountByBrandID(ctx context.Context, brandID uuid.UUID) (int64, error) {
	var count int64
	err := r.GetDB(ctx).Model(&entities.BrandCustomer{}).Where("brand_id = ?", brandID).Count(&count).Error
	return count, err
}

func (r *BrandCustomerRepository) CountByBrandIDs(ctx context.Context, brandIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	if len(brandIDs) == 0 {
		return map[uuid.UUID]int64{}, nil
	}
	type result struct {
		BrandID uuid.UUID
		Count   int64
	}
	var rows []result
	err := r.GetDB(ctx).Model(&entities.BrandCustomer{}).
		Select("brand_id, COUNT(*) AS count").
		Where("brand_id IN ?", brandIDs).
		Group("brand_id").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	counts := make(map[uuid.UUID]int64, len(rows))
	for _, row := range rows {
		counts[row.BrandID] = row.Count
	}
	return counts, nil
}
