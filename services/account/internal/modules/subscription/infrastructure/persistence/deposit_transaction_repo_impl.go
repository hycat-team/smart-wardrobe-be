package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	shared_repos "smart-wardrobe-be/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
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

func (r *DepositTransactionRepository) ListAdmin(ctx context.Context, filter repositories.AdminDepositTransactionFilter, page, limit int) ([]*entities.DepositTransaction, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	db := r.GetDB(ctx).Model(&entities.DepositTransaction{})
	if filter.Status != nil {
		db = db.Where("status = ?", int16(*filter.Status))
	}
	if filter.FromDate != nil {
		db = db.Where("created_at >= ?", *filter.FromDate)
	}
	if filter.ToDate != nil {
		// Half-open (R1/data-model §3): ToDate is the exclusive end of
		// [FromDate, ToDate), i.e. EndOfDayInVN(toDate) = 00:00 of the next VN
		// day. `<=` would double-count the instant that starts the next period.
		db = db.Where("created_at < ?", *filter.ToDate)
	}
	if filter.OrderCode != nil {
		db = db.Where("order_code = ?", *filter.OrderCode)
	}
	if filter.UserID != nil {
		db = db.Where("user_id = ?", *filter.UserID)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []*entities.DepositTransaction
	if err := db.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
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

// GetAdminSubscriptionStats aggregates deposit transactions in the half-open
// period [from, to) (029 R1/data-model §3): callers pass day boundaries
// (from = StartOfDayInVN, to = EndOfDayInVN of the last day) and a zero bound
// leaves that side open.
//
// Until US4 (029 T039) both the revenue SUM and the payOS counters still
// filter transaction_type = direct_purchase; this task only replaces the
// internally derived VNDayRange(snapshotDate) with the caller's period.
func (r *DepositTransactionRepository) GetAdminSubscriptionStats(ctx context.Context, from, to time.Time) (*dto.AdminSubscriptionStatsDTO, error) {
	type revenueResult struct {
		TotalRevenue decimal.Decimal
		TotalCount   int
		SuccessCount int
		PendingCount int
		FailedCount  int
		VoidedCount  int
	}
	var result revenueResult

	// The three status buckets below partition every PayOS deposit status
	// together with success, so downstream reports stop deriving "failed" as
	// total − success. Buckets:
	//   pending: in-flight statuses (depositstatus.ActivePaymentStatuses)
	//   failed:  genuinely failed (FailedLegacy, CreationFailed,
	//             InvestigationRequired)
	//   voided:  closed without payment (Cancelled, Expired, Refunded)
	query := `
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE status = ?), 0) AS total_revenue,
			COUNT(*) AS total_count,
			COUNT(*) FILTER (WHERE status = ?) AS success_count,
			COUNT(*) FILTER (WHERE status IN ?) AS pending_count,
			COUNT(*) FILTER (WHERE status IN ?) AS failed_count,
			COUNT(*) FILTER (WHERE status IN ?) AS voided_count
		FROM deposit_transactions
		WHERE transaction_type = ?`
	args := []any{
		depositstatus.Success,
		depositstatus.Success,
		depositstatus.ActivePaymentStatuses,
		payOSFailedStatuses,
		payOSVoidedStatuses,
		string(deposittransactiontype.DirectPurchase),
	}
	if !from.IsZero() {
		query += " AND created_at >= ?"
		args = append(args, from)
	}
	if !to.IsZero() {
		query += " AND created_at < ?"
		args = append(args, to)
	}

	err := r.GetDB(ctx).Raw(query, args...).Scan(&result).Error
	if err != nil {
		return nil, err
	}

	return &dto.AdminSubscriptionStatsDTO{
		TotalSubscriptionRevenue: result.TotalRevenue,
		PayOSTransactionCount:    result.TotalCount,
		PayOSSuccessCount:        result.SuccessCount,
		PayOSPendingCount:        result.PendingCount,
		PayOSFailedCount:         result.FailedCount,
		PayOSVoidedCount:         result.VoidedCount,
	}, nil
}

// payOSFailedStatuses are deposit statuses that genuinely failed and need
// operator attention or customer support.
var payOSFailedStatuses = []depositstatus.DepositStatus{
	depositstatus.FailedLegacy,
	depositstatus.CreationFailed,
	depositstatus.InvestigationRequired,
}

// payOSVoidedStatuses are deposit statuses closed without any payment reaching
// the gateway outcome: the customer cancelled, the link expired, or the amount
// was refunded.
var payOSVoidedStatuses = []depositstatus.DepositStatus{
	depositstatus.Cancelled,
	depositstatus.Expired,
	depositstatus.Refunded,
}

// GetRevenueBreakdown sums successful DirectPurchase revenue grouped by plan and
// month over the half-open period [fromDate, toDate) (029 R1/data-model §3):
// callers pass day boundaries (toDate = EndOfDayInVN of the last day), and a
// zero side leaves that side open. The caller owns the day arithmetic —
// adding a day here was the off-by-7h this standard removed.
func (r *DepositTransactionRepository) GetRevenueBreakdown(ctx context.Context, fromDate, toDate time.Time) ([]dto.RevenueBreakdownDTO, error) {
	query := r.GetDB(ctx).
		Table("deposit_transactions").
		Select("plan_code, DATE_TRUNC('month', created_at)::date AS month, COUNT(*) AS transaction_count, SUM(amount) AS revenue").
		Where("status = ? AND transaction_type = ?", depositstatus.Success, deposittransactiontype.DirectPurchase)
	if !fromDate.IsZero() {
		query = query.Where("created_at >= ?", fromDate)
	}
	if !toDate.IsZero() {
		query = query.Where("created_at < ?", toDate)
	}
	var rows []dto.RevenueBreakdownDTO
	err := query.Group("plan_code, DATE_TRUNC('month', created_at)").
		Order("month ASC, plan_code ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
