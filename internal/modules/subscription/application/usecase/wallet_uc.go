package usecase

import (
	"context"
	"fmt"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
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

	if wallet == nil {
		err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
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
				Balance:   0,
				Currency:  "VND",
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := uc.walletRepo.Create(txCtx, newWallet); err != nil {
				return err
			}
			wallet = newWallet
			return nil
		})
		if err != nil {
			wallet, err = uc.walletRepo.GetByUserID(ctx, userID)
			if err != nil || wallet == nil {
				return nil, errorcode.NewInternalError("Lỗi khi khởi tạo hoặc truy vấn ví người dùng")
			}
		}
	}

	return &dto.WalletDTO{
		UserID:    wallet.UserID,
		Balance:   wallet.Balance,
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
			Amount:          s.Amount,
			TransactionType: s.TransactionType,
			PreviousBalance: s.PreviousBalance,
			NewBalance:      s.NewBalance,
			Description:     s.Description,
			CreatedAt:       s.CreatedAt,
		})
	}
	return dtos, nil
}

func (uc *WalletUseCase) CreateWalletTopUp(ctx context.Context, userID uuid.UUID, req *dto.WalletTopUpReq) (*dto.PaymentLinkDTO, error) {
	if req.Amount < 1000.00 {
		return nil, errorcode.NewBadRequest("Số tiền nạp tối thiểu là 1,000 VND")
	}

	var checkoutUrl string
	var orderCode int64

	err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		tx := &entities.DepositTransaction{
			UserID:          userID,
			Amount:          req.Amount,
			Currency:        "VND",
			Status:          depositstatus.Pending,
			TransactionType: "WALLET_TOPUP",
			PaymentUrl:      nil,
		}

		if err := uc.depositTxRepo.Create(txCtx, tx); err != nil {
			return errorcode.NewInternalError("Lỗi khi khởi tạo giao dịch nạp tiền")
		}

		returnUrl := req.ReturnUrl
		if returnUrl == "" {
			returnUrl = uc.cfg.PayOS.ReturnUrl
		}
		cancelUrl := req.CancelUrl
		if cancelUrl == "" {
			cancelUrl = uc.cfg.PayOS.CancelUrl
		}

		description := fmt.Sprintf("Top up wallet sum %d", int(tx.Amount))
		var err error
		checkoutUrl, err = uc.paymentGateway.CreateCheckoutSession(txCtx, &payment.CheckoutSessionReq{
			OrderCode:   tx.OrderCode,
			Amount:      tx.Amount,
			Description: description,
			ReturnUrl:   returnUrl,
			CancelUrl:   cancelUrl,
		})
		if err != nil {
			return errorcode.NewInternalError("Không thể khởi tạo liên kết thanh toán với cổng ngân hàng")
		}

		tx.PaymentUrl = &checkoutUrl
		if err := uc.depositTxRepo.Update(txCtx, tx); err != nil {
			return errorcode.NewInternalError("Lỗi khi cập nhật liên kết thanh toán")
		}

		orderCode = tx.OrderCode
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &dto.PaymentLinkDTO{
		PaymentUrl: checkoutUrl,
		OrderCode:  orderCode,
	}, nil
}
