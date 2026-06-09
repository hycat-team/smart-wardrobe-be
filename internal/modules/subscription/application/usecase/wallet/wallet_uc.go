package wallet

import (
	"context"
	"fmt"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/currency"
	"smart-wardrobe-be/internal/shared/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/internal/shared/domain/entities"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/utils/errorutils"
	"smart-wardrobe-be/pkg/utils/timeutils"

	"github.com/google/uuid"
)

type WalletUseCase struct {
	walletRepo     repositories.IUserWalletRepository
	depositTxRepo  repositories.IDepositTransactionRepository
	statementRepo  repositories.IWalletStatementRepository
	paymentGateway payment.IPaymentGatewayService
	uow            shared_repos.IUnitOfWork
	cfg            *config.Config
}

func NewWalletUseCase(
	walletRepo repositories.IUserWalletRepository,
	depositTxRepo repositories.IDepositTransactionRepository,
	statementRepo repositories.IWalletStatementRepository,
	paymentGateway payment.IPaymentGatewayService,
	uow shared_repos.IUnitOfWork,
	cfg *config.Config,
) uc_interfaces.IWalletUseCase {
	return &WalletUseCase{
		walletRepo:     walletRepo,
		depositTxRepo:  depositTxRepo,
		statementRepo:  statementRepo,
		paymentGateway: paymentGateway,
		uow:            uow,
		cfg:            cfg,
	}
}

func (uc *WalletUseCase) GetWallet(ctx context.Context, userID uuid.UUID) (*dto.WalletDTO, error) {
	wallet, err := uc.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if wallet != nil {
		return &dto.WalletDTO{
			UserID:    wallet.UserID,
			Balance:   sharedmoney.ToFloat(wallet.Balance),
			Currency:  wallet.Currency,
			UpdatedAt: wallet.UpdatedAt,
		}, nil
	}

	createNewWallet := func(txCtx context.Context) error {
		existing, err := uc.walletRepo.GetByUserIDWithLock(txCtx, userID)
		if err != nil {
			return err
		}
		if existing != nil {
			wallet = existing
			return nil
		}

		now := timeutils.GetNow(uc.cfg.Database.TimeZone)
		newWallet := &entities.UserWallet{
			UserID:    userID,
			Balance:   sharedmoney.Zero,
			Currency:  currency.VND,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := uc.walletRepo.Create(txCtx, newWallet); err != nil {
			return err
		}
		wallet = newWallet
		return nil
	}

	if err = uc.uow.Execute(ctx, createNewWallet); err != nil || wallet == nil {
		return nil, subscriptionerrors.ErrWalletNotFound
	}

	return &dto.WalletDTO{
		UserID:    wallet.UserID,
		Balance:   sharedmoney.ToFloat(wallet.Balance),
		Currency:  wallet.Currency,
		UpdatedAt: wallet.UpdatedAt,
	}, nil
}

func (uc *WalletUseCase) GetWalletStatements(ctx context.Context, userID uuid.UUID) ([]*dto.WalletStatementDTO, error) {
	statements, err := uc.statementRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	dtos := make([]*dto.WalletStatementDTO, 0, len(statements))
	for _, s := range statements {
		dtos = append(dtos, &dto.WalletStatementDTO{
			ID:              s.ID,
			UserID:          s.UserID,
			Amount:          sharedmoney.ToFloat(s.Amount),
			TransactionType: s.TransactionType,
			PreviousBalance: sharedmoney.ToFloat(s.PreviousBalance),
			NewBalance:      sharedmoney.ToFloat(s.NewBalance),
			Description:     s.Description,
			CreatedAt:       s.CreatedAt,
		})
	}
	return dtos, nil
}

func (uc *WalletUseCase) CreateWalletTopUp(ctx context.Context, userID uuid.UUID, req *dto.WalletTopUpReq) (*dto.PaymentLinkDTO, error) {
	amount, err := sharedmoney.FromFloatAmount(req.Amount)
	if err != nil {
		return nil, subscriptionerrors.ErrInvalidDepositAmount
	}

	var checkoutURL string
	var orderCode int64

	createWalletTopUp := func(txCtx context.Context) error {
		tx := &entities.DepositTransaction{
			UserID:          userID,
			Amount:          amount,
			Currency:        currency.VND,
			Status:          depositstatus.Pending,
			TransactionType: deposittransactiontype.WalletTopup,
			OrderCode:       timeutils.GenerateOrderCode(),
			PaymentUrl:      nil,
		}

		if err := uc.depositTxRepo.Create(txCtx, tx); err != nil {
			return subscriptionerrors.ErrDepositInitFailed
		}

		returnURL := req.ReturnUrl
		if returnURL == "" {
			returnURL = uc.cfg.PayOS.ReturnUrl
		}
		cancelURL := req.CancelUrl
		if cancelURL == "" {
			cancelURL = uc.cfg.PayOS.CancelUrl
		}

		amountVND, err := sharedmoney.ToMinorUnits(tx.Amount, currency.VND)
		if err != nil {
			return subscriptionerrors.ErrDepositMustBeInteger
		}
		description := fmt.Sprintf("Nạp vào ví %d VNĐ", amountVND)
		checkoutURL, err = uc.paymentGateway.CreateCheckoutSession(txCtx, &payment.CheckoutSessionReq{
			OrderCode:   tx.OrderCode,
			Amount:      tx.Amount,
			Description: description,
			ReturnUrl:   returnURL,
			CancelUrl:   cancelURL,
		})
		if err != nil {
			return errorutils.WrapError(err, "Không thể khởi tạo liên kết thanh toán với cổng ngân hàng")
		}

		tx.PaymentUrl = &checkoutURL
		if err := uc.depositTxRepo.Update(txCtx, tx); err != nil {
			return subscriptionerrors.ErrPaymentLinkUpdateFailed
		}

		orderCode = tx.OrderCode
		return nil
	}

	if err := uc.uow.Execute(ctx, createWalletTopUp); err != nil {
		return nil, err
	}

	return &dto.PaymentLinkDTO{
		PaymentUrl: checkoutURL,
		OrderCode:  orderCode,
	}, nil
}

