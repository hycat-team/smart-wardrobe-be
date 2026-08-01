package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/internal/modules/brand/domain/constants/loyaltypointlotstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LoyaltyPointLotRepository struct {
	shared_persist.GenericRepository[entities.LoyaltyPointLot, uuid.UUID]
}

func NewLoyaltyPointLotRepository(db *gorm.DB) repositories.ILoyaltyPointLotRepository {
	relations := []string{"LoyaltyAccount", "Brand", "BrandCustomer", "EarnTransaction"}
	return &LoyaltyPointLotRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.LoyaltyPointLot, uuid.UUID](db, relations),
	}
}

func (r *LoyaltyPointLotRepository) ListRedeemableLotsForUpdate(ctx context.Context, loyaltyAccountID uuid.UUID, now time.Time) ([]*entities.LoyaltyPointLot, error) {
	var lots []*entities.LoyaltyPointLot
	err := r.GetDB(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("loyalty_account_id = ? AND status = ? AND remaining_points > 0 AND (expires_at IS NULL OR expires_at > ?)", loyaltyAccountID, loyaltypointlotstatus.Active, now).
		Order("expires_at ASC NULLS LAST, created_at ASC").
		Find(&lots).Error
	return lots, err
}

func (r *LoyaltyPointLotRepository) ListExpiredLotsForUpdate(ctx context.Context, loyaltyAccountID uuid.UUID, now time.Time) ([]*entities.LoyaltyPointLot, error) {
	var lots []*entities.LoyaltyPointLot
	err := r.GetDB(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("loyalty_account_id = ? AND status = ? AND remaining_points > 0 AND expires_at IS NOT NULL AND expires_at <= ?", loyaltyAccountID, loyaltypointlotstatus.Active, now).
		Order("expires_at ASC, created_at ASC").
		Find(&lots).Error
	return lots, err
}

func (r *LoyaltyPointLotRepository) UpdateLotRemainingAndStatus(ctx context.Context, lotID uuid.UUID, remainingPoints int, status loyaltypointlotstatus.LoyaltyPointLotStatus) error {
	return r.GetDB(ctx).
		Model(&entities.LoyaltyPointLot{}).
		Where("id = ?", lotID).
		Updates(map[string]any{
			"remaining_points": remainingPoints,
			"status":           status,
			"updated_at":       time.Now().UTC(),
		}).Error
}

func (r *LoyaltyPointLotRepository) ListAccountsWithExpiredLots(ctx context.Context, now time.Time, limit int) ([]uuid.UUID, error) {
	if limit <= 0 {
		limit = 100
	}
	var accountIDs []uuid.UUID
	err := r.GetDB(ctx).
		Model(&entities.LoyaltyPointLot{}).
		Distinct("loyalty_account_id").
		Where("status = ? AND remaining_points > 0 AND expires_at IS NOT NULL AND expires_at <= ?", loyaltypointlotstatus.Active, now).
		Limit(limit).
		Pluck("loyalty_account_id", &accountIDs).Error
	return accountIDs, err
}

func (r *LoyaltyPointLotRepository) GetNearestExpiringActiveLot(ctx context.Context, loyaltyAccountID uuid.UUID, now time.Time) (*entities.LoyaltyPointLot, error) {
	var lot entities.LoyaltyPointLot
	err := r.GetDB(ctx).
		Where("loyalty_account_id = ? AND status = ? AND remaining_points > 0 AND expires_at IS NOT NULL AND expires_at > ?", loyaltyAccountID, loyaltypointlotstatus.Active, now).
		Order("expires_at ASC, created_at ASC").
		First(&lot).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &lot, nil
}

func (r *LoyaltyPointLotRepository) ListByAccountID(ctx context.Context, loyaltyAccountID uuid.UUID, status *loyaltypointlotstatus.LoyaltyPointLotStatus, expiresAt *time.Time, page int, limit int) ([]*entities.LoyaltyPointLot, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	query := r.GetDB(ctx).Where("loyalty_account_id = ?", loyaltyAccountID)
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	if expiresAt != nil {
		query = query.Where("expires_at <= ?", *expiresAt)
	}
	var lots []*entities.LoyaltyPointLot
	err := query.
		Order("expires_at ASC NULLS LAST, created_at ASC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&lots).Error
	return lots, err
}

func (r *LoyaltyPointLotRepository) SumExpiringPointsByMonth(ctx context.Context, brandID uuid.UUID, status loyaltypointlotstatus.LoyaltyPointLotStatus) ([]repositories.MonthPointSum, error) {
	var rows []repositories.MonthPointSum
	err := r.GetDB(ctx).
		Table("loyalty_point_lots").
		Select("DATE_TRUNC('month', expires_at)::date AS month, COALESCE(SUM(remaining_points), 0) AS points_expiring").
		Where("brand_id = ? AND status = ? AND remaining_points > 0 AND expires_at IS NOT NULL", brandID, string(status)).
		Group("DATE_TRUNC('month', expires_at)").
		Order("month ASC").
		Limit(6).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
