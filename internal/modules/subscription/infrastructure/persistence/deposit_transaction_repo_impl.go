package persistence

import (
	"context"
	"errors"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/infrastructure/repositories"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DepositTransactionRepository struct {
	*shared_repos.GenericRepository[entities.DepositTransaction, uuid.UUID]
}

func NewDepositTransactionRepository(dbConn *gorm.DB) repositories.IDepositTransactionRepository {
	relations := []string{}
	return &DepositTransactionRepository{
		GenericRepository: shared_repos.NewGenericRepository[entities.DepositTransaction, uuid.UUID](dbConn, relations),
	}
}

func (r *DepositTransactionRepository) GetByGatewayReference(ctx context.Context, reference string) (*entities.DepositTransaction, error) {
	var tx entities.DepositTransaction
	err := r.GetDB(ctx).Where("successful_provider_reference = ?", reference).First(&tx).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

func (r *DepositTransactionRepository) GetByOrderCode(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error) {
	var tx entities.DepositTransaction
	err := r.GetDB(ctx).Where("order_code = ?", orderCode).First(&tx).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

func (r *DepositTransactionRepository) GetByOrderCodeWithLock(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error) {
	var tx entities.DepositTransaction
	err := r.GetDB(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("order_code = ?", orderCode).First(&tx).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tx, nil
}

func (r *DepositTransactionRepository) HasPendingDirectPurchase(ctx context.Context, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.GetDB(ctx).Model(&entities.DepositTransaction{}).
		Where("user_id = ? AND status IN ? AND transaction_type = ?", userID, depositstatus.ActivePaymentStatuses, deposittransactiontype.DirectPurchase).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *DepositTransactionRepository) GetActiveDirectPurchase(ctx context.Context, userID uuid.UUID) (*entities.DepositTransaction, error) {
	var tx entities.DepositTransaction
	err := r.GetDB(ctx).Where("user_id = ? AND status IN ? AND transaction_type = ?", userID, depositstatus.ActivePaymentStatuses, deposittransactiontype.DirectPurchase).Order("created_at DESC").First(&tx).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &tx, nil
}
func (r *DepositTransactionRepository) ClaimReconciliationCandidates(ctx context.Context, limit int, lease time.Duration) ([]*repositories.ClaimedDepositTransaction, error) {
	if limit <= 0 {
		limit = 50
	}
	var rows []*entities.DepositTransaction
	var results []*repositories.ClaimedDepositTransaction
	err := r.GetDB(ctx).Transaction(func(db *gorm.DB) error {
		query := db.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("(status = ? AND expires_at <= NOW()) OR (status IN ? AND next_reconciliation_at <= NOW()) OR (status = ? AND processing_lease_until <= NOW())", depositstatus.Pending, []depositstatus.DepositStatus{depositstatus.Creating, depositstatus.ReconciliationRequired}, depositstatus.Reconciling).
			Order("COALESCE(next_reconciliation_at, expires_at) ASC").Limit(limit)
		if err := query.Find(&rows).Error; err != nil {
			return err
		}
		until := time.Now().UTC().Add(lease)
		for _, row := range rows {
			claimedFromStatus := row.Status
			token := uuid.New()
			row.Status = depositstatus.Reconciling
			row.ProcessingToken = &token
			row.ProcessingLeaseUntil = &until
			if err := db.Model(row).Updates(map[string]any{
				"status":                 row.Status,
				"processing_token":       token,
				"processing_lease_until": gorm.Expr("NOW() + (? * interval '1 second')", int(lease.Seconds())),
			}).Error; err != nil {
				return err
			}
			results = append(results, &repositories.ClaimedDepositTransaction{
				Transaction:       row,
				ClaimedFromStatus: claimedFromStatus,
				ProcessingToken:   token,
			})
		}
		return nil
	})
	return results, err
}

func (r *DepositTransactionRepository) UpdateWithToken(ctx context.Context, orderCode int64, token uuid.UUID, updates map[string]any) (int64, error) {
	db := r.GetDB(ctx)
	res := db.Model(&entities.DepositTransaction{}).
		Where("order_code = ? AND status = ? AND processing_token = ?", orderCode, depositstatus.Reconciling, token).
		Updates(updates)
	return res.RowsAffected, res.Error
}
