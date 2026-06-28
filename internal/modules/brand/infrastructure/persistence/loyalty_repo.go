package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/loyaltypointlotstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_persist "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LoyaltyProgramRepository struct {
	shared_persist.GenericRepository[entities.LoyaltyProgram, uuid.UUID]
}

func NewLoyaltyProgramRepository(db *gorm.DB) repositories.ILoyaltyProgramRepository {
	return &LoyaltyProgramRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.LoyaltyProgram, uuid.UUID](db, nil),
	}
}

func (r *LoyaltyProgramRepository) GetActiveByBrandID(ctx context.Context, brandID uuid.UUID) (*entities.LoyaltyProgram, error) {
	var program entities.LoyaltyProgram
	err := r.GetDB(ctx).Where("brand_id = ? AND is_active = ?", brandID, true).First(&program).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &program, nil
}

type LoyaltyTierRepository struct {
	shared_persist.GenericRepository[entities.LoyaltyTier, uuid.UUID]
}

func NewLoyaltyTierRepository(db *gorm.DB) repositories.ILoyaltyTierRepository {
	return &LoyaltyTierRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.LoyaltyTier, uuid.UUID](db, nil),
	}
}

func (r *LoyaltyTierRepository) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.LoyaltyTier, error) {
	var tiers []*entities.LoyaltyTier
	err := r.GetDB(ctx).
		Where("brand_id = ?", brandID).
		Order("rank ASC").
		Find(&tiers).Error
	return tiers, err
}

func (r *LoyaltyTierRepository) GetHighestEligibleBySpend(ctx context.Context, brandID uuid.UUID, totalSpend float64) (*entities.LoyaltyTier, error) {
	var tier entities.LoyaltyTier
	err := r.GetDB(ctx).
		Where("brand_id = ? AND min_total_spend <= ?", brandID, totalSpend).
		Order("min_total_spend DESC, rank DESC").
		First(&tier).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tier, nil
}

type LoyaltyAccountRepository struct {
	shared_persist.GenericRepository[entities.LoyaltyAccount, uuid.UUID]
}

func NewLoyaltyAccountRepository(db *gorm.DB) repositories.ILoyaltyAccountRepository {
	return &LoyaltyAccountRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.LoyaltyAccount, uuid.UUID](db, []string{"CurrentTier"}),
	}
}

func (r *LoyaltyAccountRepository) GetByBrandCustomerID(ctx context.Context, brandCustomerID uuid.UUID) (*entities.LoyaltyAccount, error) {
	var account entities.LoyaltyAccount
	err := r.GetQueryWithPreload(ctx).Where("brand_customer_id = ?", brandCustomerID).First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (r *LoyaltyAccountRepository) GetByBrandCustomerIDForUpdate(ctx context.Context, brandCustomerID uuid.UUID) (*entities.LoyaltyAccount, error) {
	var account entities.LoyaltyAccount
	err := r.GetQueryWithPreload(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("brand_customer_id = ?", brandCustomerID).
		First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (r *LoyaltyAccountRepository) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*entities.LoyaltyAccount, error) {
	var account entities.LoyaltyAccount
	err := r.GetQueryWithPreload(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", id).
		First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (r *LoyaltyAccountRepository) GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.LoyaltyAccount, error) {
	var account entities.LoyaltyAccount
	err := r.GetQueryWithPreload(ctx).Where("brand_id = ? AND user_id = ?", brandID, userID).First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

type LoyaltyPointTransactionRepository struct {
	shared_persist.GenericRepository[entities.LoyaltyPointTransaction, uuid.UUID]
}

func NewLoyaltyPointTransactionRepository(db *gorm.DB) repositories.ILoyaltyPointTransactionRepository {
	return &LoyaltyPointTransactionRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.LoyaltyPointTransaction, uuid.UUID](db, nil),
	}
}

func (r *LoyaltyPointTransactionRepository) GetByBrandAndIdempotencyKey(ctx context.Context, brandID uuid.UUID, idempotencyKey string) (*entities.LoyaltyPointTransaction, error) {
	var tx entities.LoyaltyPointTransaction
	err := r.GetDB(ctx).Where("brand_id = ? AND idempotency_key = ?", brandID, idempotencyKey).First(&tx).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

func (r *LoyaltyPointTransactionRepository) GetByLoyaltyAccountID(ctx context.Context, loyaltyAccountID uuid.UUID) ([]*entities.LoyaltyPointTransaction, error) {
	var transactions []*entities.LoyaltyPointTransaction
	err := r.GetDB(ctx).
		Where("loyalty_account_id = ?", loyaltyAccountID).
		Order("created_at ASC, id ASC").
		Find(&transactions).Error
	return transactions, err
}

type LoyaltyPointLotRepository struct {
	shared_persist.GenericRepository[entities.LoyaltyPointLot, uuid.UUID]
}

func NewLoyaltyPointLotRepository(db *gorm.DB) repositories.ILoyaltyPointLotRepository {
	return &LoyaltyPointLotRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.LoyaltyPointLot, uuid.UUID](db, nil),
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

type BrandCustomerClaimRepository struct {
	shared_persist.GenericRepository[entities.BrandCustomerClaim, uuid.UUID]
}

func NewBrandCustomerClaimRepository(db *gorm.DB) repositories.IBrandCustomerClaimRepository {
	return &BrandCustomerClaimRepository{
		GenericRepository: *shared_persist.NewGenericRepository[entities.BrandCustomerClaim, uuid.UUID](db, nil),
	}
}

func (r *BrandCustomerClaimRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*entities.BrandCustomerClaim, error) {
	var claim entities.BrandCustomerClaim
	err := r.GetDB(ctx).Where("claim_token_hash = ?", tokenHash).First(&claim).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &claim, nil
}
