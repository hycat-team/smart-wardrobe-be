package usecase

import (
	"context"
	"smart-wardrobe-be/services/account/dto/subscription"
	shared_dto "smart-wardrobe-be/shared/application/dto"

	"github.com/google/uuid"
)

type IWalletUseCase interface {
	GetWallet(ctx context.Context, userID uuid.UUID) (*dto.WalletDTO, error)
	GetWalletStatements(ctx context.Context, userID uuid.UUID, query dto.GetWalletStatementsQueryReq) (*shared_dto.PaginationResult[*dto.WalletStatementDTO], error)
	CreateWalletTopUp(ctx context.Context, userID uuid.UUID, req *dto.WalletTopUpReq) (*dto.PaymentLinkDTO, error)
}
