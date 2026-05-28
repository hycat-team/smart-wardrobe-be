package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/subscription/application/dto"

	"github.com/google/uuid"
)

type ISubscriptionPurchaseUseCase interface {
	CreateDirectPurchase(ctx context.Context, userID uuid.UUID, req *dto.DirectPurchaseReq) (*dto.PaymentLinkDTO, error)
	PurchasePlanWithWallet(ctx context.Context, userID uuid.UUID, planSlug string) error
}
