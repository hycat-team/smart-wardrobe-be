package persistence

import (
	"context"
	"errors"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefit/benefitstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandchat/conversationstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandcustomerstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/loyaltypointlotstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/loyaltytransactiontype"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BrandRepository struct {
	shared_persist.GenericRepository[entities.Brand, uuid.UUID]
}

func NewBrandRepository(db *gorm.DB) repositories.IBrandRepository {
	relations := []string{}
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

func (r *BrandRepository) GetAdminBrandStats(ctx context.Context, snapshotDate time.Time) (*dto.BrandAdminStatsDTO, error) {
	db := r.GetDB(ctx)

	var totalBrands int64
	if err := db.Model(&entities.Brand{}).Count(&totalBrands).Error; err != nil {
		return nil, err
	}

	var pendingBrands int64
	if err := db.Model(&entities.Brand{}).Where("status = ?", string(brandstatus.PendingReview)).Count(&pendingBrands).Error; err != nil {
		return nil, err
	}

	return &dto.BrandAdminStatsDTO{
		SnapshotDate:        snapshotDate,
		TotalBrands:         int(totalBrands),
		PendingReviewBrands: int(pendingBrands),
	}, nil
}

func (r *BrandRepository) GetBrandAnalyticsStats(ctx context.Context, brandID uuid.UUID, snapshotDate time.Time) (*dto.BrandAnalyticsStatsDTO, error) {
	db := r.GetDB(ctx)

	var totalMembers int64
	_ = db.Table("brand_customers").Where("brand_id = ? AND status = ?", brandID, string(brandcustomerstatus.Active)).Count(&totalMembers).Error

	var pointsIssued int64
	_ = db.Table("loyalty_point_transactions").
		Where("brand_id = ? AND transaction_type = ?", brandID, string(loyaltytransactiontype.Earn)).
		Select("COALESCE(SUM(points_delta), 0)").Scan(&pointsIssued).Error

	var pointsRedeemed int64
	_ = db.Table("loyalty_point_transactions").
		Where("brand_id = ? AND transaction_type = ?", brandID, string(loyaltytransactiontype.Redeem)).
		Select("COALESCE(SUM(points_delta), 0)").Scan(&pointsRedeemed).Error

	var expiringPoints int64
	_ = db.Table("loyalty_point_lots").
		Where("brand_id = ? AND status = ? AND expires_at >= ? AND expires_at <= ?", brandID, string(loyaltypointlotstatus.Active), snapshotDate, snapshotDate.AddDate(0, 0, 30)).
		Select("COALESCE(SUM(remaining_points), 0)").Scan(&expiringPoints).Error

	var activeVouchers int64
	_ = db.Table("brand_benefits").
		Where("brand_id = ? AND status = ?", brandID, string(benefitstatus.Active)).Count(&activeVouchers).Error

	var vouchersRedeemed int64
	_ = db.Table("benefit_redemptions").
		Where("brand_id = ?", brandID).Count(&vouchersRedeemed).Error

	var sampleVotesCount int64
	_ = db.Table("digital_sample_responses").
		Joins("JOIN brand_items ON brand_items.id = digital_sample_responses.brand_item_id").
		Where("brand_items.brand_id = ?", brandID).Count(&sampleVotesCount).Error

	var sampleAvgRating float64
	_ = db.Table("digital_sample_responses").
		Joins("JOIN brand_items ON brand_items.id = digital_sample_responses.brand_item_id").
		Where("brand_items.brand_id = ? AND rating IS NOT NULL", brandID).
		Select("COALESCE(AVG(rating), 0)").Scan(&sampleAvgRating).Error

	var openTickets int64
	_ = db.Table("brand_conversations").
		Where("brand_id = ? AND status = ?", brandID, string(conversationstatus.Open)).Count(&openTickets).Error

	return &dto.BrandAnalyticsStatsDTO{
		BrandID:                 brandID,
		SnapshotDate:            snapshotDate,
		TotalMembers:            int(totalMembers),
		PointsIssued:            int(pointsIssued),
		PointsRedeemed:          int(pointsRedeemed),
		PointsExpiring30d:       int(expiringPoints),
		PointsLiabilityValueVND: float64(expiringPoints) * 1000.0,
		ActiveVouchers:          int(activeVouchers),
		VouchersRedeemed:        int(vouchersRedeemed),
		DigitalSampleVotesCount: int(sampleVotesCount),
		DigitalSampleAvgRating:  sampleAvgRating,
		OpenCSTickets:           int(openTickets),
	}, nil
}

func (r *BrandRepository) GetSampleLabFeedbacksDrillDown(ctx context.Context, itemID uuid.UUID, page, limit int) ([]dto.SampleFeedbackDrillDownDTO, int64, error) {
	db := r.GetDB(ctx).Table("digital_sample_responses").Where("brand_item_id = ?", itemID)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	type SampleResponseRow struct {
		ID          uuid.UUID `gorm:"column:id"`
		BrandItemID uuid.UUID `gorm:"column:brand_item_id"`
		UserID      uuid.UUID `gorm:"column:user_id"`
		Rating      float64   `gorm:"column:rating"`
		Comment     string    `gorm:"column:comment"`
		CreatedAt   time.Time `gorm:"column:created_at"`
	}

	var rows []SampleResponseRow
	err := db.Order("created_at DESC").Offset(offset).Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	items := make([]dto.SampleFeedbackDrillDownDTO, 0, len(rows))
	for _, row := range rows {
		items = append(items, dto.SampleFeedbackDrillDownDTO{
			FeedbackID: row.ID.String(),
			ItemID:     row.BrandItemID,
			UserID:     row.UserID,
			Rating:     row.Rating,
			Comment:    row.Comment,
			CreatedAt:  row.CreatedAt,
		})
	}

	return items, total, nil
}
