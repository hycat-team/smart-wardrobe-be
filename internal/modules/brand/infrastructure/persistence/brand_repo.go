package persistence

import (
	"context"
	"errors"
	"strings"

	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberstatus"
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

type BrandMemberRepository struct {
	shared_persist.GenericRepository[entities.BrandMember, uuid.UUID]
}

func NewBrandMemberRepository(db *gorm.DB) repositories.IBrandMemberRepository {
	return &BrandMemberRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.BrandMember, uuid.UUID](db, []string{"User", "Brand"}),
	}
}

func (r *BrandMemberRepository) GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandMember, error) {
	var member entities.BrandMember
	err := r.GetQueryWithPreload(ctx).
		Where("brand_id = ? AND user_id = ?", brandID, userID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &member, nil
}

func (r *BrandMemberRepository) GetByBrandAndUserIDs(ctx context.Context, brandID uuid.UUID, userIDs []uuid.UUID) ([]*entities.BrandMember, error) {
	if len(userIDs) == 0 {
		return []*entities.BrandMember{}, nil
	}
	var members []*entities.BrandMember
	err := r.GetQueryWithPreload(ctx).
		Where("brand_id = ? AND user_id IN ?", brandID, userIDs).
		Find(&members).Error
	return members, err
}

func (r *BrandMemberRepository) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandMember, error) {
	var members []*entities.BrandMember
	err := r.GetQueryWithPreload(ctx).
		Where("brand_id = ?", brandID).
		Order("created_at ASC").
		Find(&members).Error
	return members, err
}

func (r *BrandMemberRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.BrandMember, error) {
	var members []*entities.BrandMember
	err := r.GetQueryWithPreload(ctx).
		Preload("Brand").
		Where("user_id = ? AND status = ?", userID, brandmemberstatus.Active).
		Order("created_at ASC").
		Find(&members).Error
	return members, err
}

type BrandCustomerRepository struct {
	shared_persist.GenericRepository[entities.BrandCustomer, uuid.UUID]
}

func NewBrandCustomerRepository(db *gorm.DB) repositories.IBrandCustomerRepository {
	return &BrandCustomerRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.BrandCustomer, uuid.UUID](
			db,
			[]string{"User", "Brand"},
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

	// Since we added LoyaltyAccount relation, we preload it and its nested tier
	db = db.Preload("LoyaltyAccount").Preload("LoyaltyAccount.CurrentTier")

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
