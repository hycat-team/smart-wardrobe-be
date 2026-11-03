package admin

import (
	"context"

	identity_contract "smart-wardrobe-be/services/account/contract/identity"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AdminBillingUseCase provides read access to deposit transactions for the
// admin billing endpoints. It wraps the repository so handlers stay thin and
// respect the layer boundaries.
type AdminBillingUseCase struct {
	depositRepo repositories.IDepositTransactionRepository
	// userBrief is the identity module's ALREADY-EXISTING read contract
	// (`GetByID`). The subscription module does not import identity's mapper or
	// add a method to the gRPC-served interface: it maps the six fields it wants
	// to publish in its own mapper (`deposit_transaction.go`).
	userBrief identity_contract.IUserBriefContract
	// planRepo resolves the plan named by `target_plan_id`. This is the
	// subscription module's OWN plan catalogue — same module, same database, not
	// a cross-module call.
	planRepo repositories.ISubscriptionPlanRepository
	log      logger.Interface
}

func NewAdminBillingUseCase(
	depositRepo repositories.IDepositTransactionRepository,
	userBrief identity_contract.IUserBriefContract,
	planRepo repositories.ISubscriptionPlanRepository,
	log logger.Interface,
) usecase.IAdminBillingUseCase {
	return &AdminBillingUseCase{
		depositRepo: depositRepo,
		userBrief:   userBrief,
		planRepo:    planRepo,
		log:         log,
	}
}

func (uc *AdminBillingUseCase) ListTransactions(ctx context.Context, filter usecase.AdminBillingFilter, page, limit int) ([]*entities.DepositTransaction, int64, error) {
	return uc.depositRepo.ListAdmin(ctx, repositories.AdminDepositTransactionFilter{
		Status:    filter.Status,
		FromDate:  filter.FromDate,
		ToDate:    filter.ToDate,
		OrderCode: filter.OrderCode,
		UserID:    filter.UserID,
	}, page, limit)
}

// GetTransactionDetail reads one transaction and resolves the two display
// labels the detail screen shows instead of raw ids.
//
// ENRICHMENT IS BEST-EFFORT. The reconciliation screen exists so an operator can
// fix a stuck payment; a failing label lookup must not take the transaction away
// from them. The failure is logged and the label comes back nil, which the DTO
// reports as null and the frontend falls back to the id (029 FR-007 keeps the
// key present either way).
//
// A missing label is also a normal outcome, not an error: `GetByID` on the
// identity contract answers (nil, nil) for a soft-deleted account, a plan can
// have been removed from the catalogue, and a wallet top-up carries no
// `target_plan_id` at all — in which case the plan lookup is skipped entirely.
func (uc *AdminBillingUseCase) GetTransactionDetail(ctx context.Context, id uuid.UUID) (*usecase.AdminTransactionDetail, error) {
	tx, err := uc.depositRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// The generic repository reports a miss as (nil, nil); the admin contract
	// promises 404 for an unknown transaction (029 data-model §8), and a 200
	// with an empty body would read as "found, no data".
	if tx == nil {
		return nil, apperror.NewNotFound("Không tìm thấy giao dịch.")
	}

	detail := &usecase.AdminTransactionDetail{Transaction: tx}

	if user, userErr := uc.userBrief.GetByID(ctx, tx.UserID); userErr != nil {
		uc.log.Warn("admin transaction detail: user lookup failed; the transaction is returned without its buyer label",
			zap.String("transactionId", tx.ID.String()),
			zap.String("userId", tx.UserID.String()),
			zap.Error(userErr))
	} else {
		detail.User = user
	}

	if tx.SubscriptionPlanID != nil {
		plans, planErr := uc.planRepo.GetByIDs(ctx, []uuid.UUID{*tx.SubscriptionPlanID})
		if planErr != nil {
			uc.log.Warn("admin transaction detail: plan lookup failed; the transaction is returned without its plan label",
				zap.String("transactionId", tx.ID.String()),
				zap.String("targetPlanId", tx.SubscriptionPlanID.String()),
				zap.Error(planErr))
		} else if len(plans) > 0 {
			detail.Plan = plans[0]
		}
	}

	return detail, nil
}

func (uc *AdminBillingUseCase) GetTransactionByOrderCode(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error) {
	return uc.depositRepo.GetByOrderCode(ctx, orderCode)
}
