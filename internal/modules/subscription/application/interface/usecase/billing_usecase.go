package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type IBillingUseCase interface {
	GetWallet(ctx context.Context, userID uuid.UUID) (*dto.WalletDTO, error)
	GetWalletStatements(ctx context.Context, userID uuid.UUID) ([]*dto.WalletStatementDTO, error)
	CreateWalletTopUp(ctx context.Context, userID uuid.UUID, req *dto.WalletTopUpReq) (*dto.PaymentLinkDTO, error)
	CreateDirectPurchase(ctx context.Context, userID uuid.UUID, req *dto.DirectPurchaseReq) (*dto.PaymentLinkDTO, error)
	ProcessWebhook(ctx context.Context, rawBody []byte, signature string) error
	GetPlans(ctx context.Context) ([]*entities.SubscriptionPlan, error)
}
