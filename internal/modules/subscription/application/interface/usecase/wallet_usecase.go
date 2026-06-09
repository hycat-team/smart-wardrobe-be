package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"

	"github.com/google/uuid"
)

type IWalletUseCase interface {
	GetWallet(ctx context.Context, userID uuid.UUID) (*dto.WalletDTO, error)
	GetWalletStatements(ctx context.Context, userID uuid.UUID, query dto.GetWalletStatementsQueryReq) (*shared_dto.PaginationResult[*dto.WalletStatementDTO], error)
	CreateWalletTopUp(ctx context.Context, userID uuid.UUID, req *dto.WalletTopUpReq) (*dto.PaymentLinkDTO, error)
}
