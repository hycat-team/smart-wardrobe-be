package usecase

import (
	"context"
	"time"

	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"

	"github.com/google/uuid"
)

// AdminBillingFilter filters deposit transactions for the admin list endpoint.
type AdminBillingFilter struct {
	Status    *depositstatus.DepositStatus
	FromDate  *time.Time
	ToDate    *time.Time
	OrderCode *int64
	UserID    *uuid.UUID
}

// AdminTransactionDetail is one transaction together with the two labels the
// admin detail screen displays instead of raw ids: the buyer and the plan the
// purchase was aimed at.
//
// The transaction stays the ENTITY, not a DTO (029 R5): mapping to the response
// shape belongs to `application/mapper`, and the handler is the layer that knows
// which of the two endpoints it is answering. User and Plan are enrichment and
// may be nil — a soft-deleted account, a plan removed from the catalogue, a
// wallet top-up with no plan at all, or a lookup that failed and was degraded
// (the reconciliation screen must stay usable).
type AdminTransactionDetail struct {
	Transaction *entities.DepositTransaction
	User        *identity_dto.UserBriefContractDTO
	Plan        *entities.SubscriptionPlan
}

type IAdminBillingUseCase interface {
	ListTransactions(ctx context.Context, filter AdminBillingFilter, page, limit int) ([]*entities.DepositTransaction, int64, error)
	// GetTransactionDetail returns the transaction plus its display labels. The
	// list endpoint keeps using ListTransactions and never pays for the two
	// extra lookups: it shows ids only, by design.
	GetTransactionDetail(ctx context.Context, id uuid.UUID) (*AdminTransactionDetail, error)
	GetTransactionByOrderCode(ctx context.Context, orderCode int64) (*entities.DepositTransaction, error)
}
