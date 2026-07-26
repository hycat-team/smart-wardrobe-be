package wallet

import (
	"context"
	"fmt"
	"time"

	"smart-wardrobe-be/internal/config"
	notificationContract "smart-wardrobe-be/internal/modules/notification/presentation/contract"
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/application/mapper"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/pkg/utils/errorutils"
	"smart-wardrobe-be/internal/pkg/utils/timeutils"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/shared/currency"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type WalletUseCase struct {
	walletRepo      repositories.IUserWalletRepository
	depositTxRepo   repositories.IDepositTransactionRepository
	statementRepo   repositories.IWalletStatementRepository
	paymentGateway  payment.IPaymentGatewayService
	uow             shared_repos.IUnitOfWork
	cfg             *config.Config
	notifContract   notificationContract.INotificationContract
	userContactRepo repositories.IUserContactRepository
}

func NewWalletUseCase(
	walletRepo repositories.IUserWalletRepository,
	depositTxRepo repositories.IDepositTransactionRepository,
	statementRepo repositories.IWalletStatementRepository,
	paymentGateway payment.IPaymentGatewayService,
	uow shared_repos.IUnitOfWork,
	cfg *config.Config,
	notifContract notificationContract.INotificationContract,
	userContactRepo repositories.IUserContactRepository,
) uc_interfaces.IWalletUseCase {
	return &WalletUseCase{
		walletRepo:      walletRepo,
		depositTxRepo:   depositTxRepo,
		statementRepo:   statementRepo,
		paymentGateway:  paymentGateway,
		uow:             uow,
		cfg:             cfg,
		notifContract:   notifContract,
		userContactRepo: userContactRepo,
	}
}

func (uc *WalletUseCase) GetWallet(ctx context.Context, userID uuid.UUID) (*dto.WalletDTO, error) {
	wallet, err := uc.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if wallet != nil {
		return mapper.MapToWalletDTO(wallet), nil
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
		return nil, subscriptionerrors.ErrWalletNotFound()
	}

	return mapper.MapToWalletDTO(wallet), nil
}

func (uc *WalletUseCase) GetWalletStatements(ctx context.Context, userID uuid.UUID, query dto.GetWalletStatementsQueryReq) (*shared_dto.PaginationResult[*dto.WalletStatementDTO], error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}

	totalItems, err := uc.statementRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	paginationQuery := shared_dto.PaginationQuery{
		Page:  page,
		Limit: limit,
	}

	statements, err := uc.statementRepo.GetByUserID(ctx, userID, paginationQuery)
	if err != nil {
		return nil, err
	}

	return &shared_dto.PaginationResult[*dto.WalletStatementDTO]{
		Items:    mapper.MapToWalletStatementDTOList(statements),
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, totalItems),
	}, nil
}

func (uc *WalletUseCase) CreateWalletTopUp(ctx context.Context, userID uuid.UUID, req *dto.WalletTopUpReq) (*dto.PaymentLinkDTO, error) {
	amount, err := sharedmoney.FromFloatAmount(req.Amount)
	if err != nil {
		return nil, subscriptionerrors.ErrInvalidDepositAmount()
	}

	now := timeutils.GetNow(uc.cfg.Database.TimeZone)
	expiresAt := now.Add(time.Duration(uc.cfg.PayOS.ExpiredMinutes) * time.Minute)
	nextReconcile := now.Add(30 * time.Second)
	var tx *entities.DepositTransaction
	if err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		tx = &entities.DepositTransaction{
			UserID:               userID,
			Amount:               amount,
			ExpectedAmount:       amount,
			Currency:             currency.VND,
			Status:               depositstatus.Creating,
			TransactionType:      deposittransactiontype.WalletTopup,
			ExpiresAt:            &expiresAt,
			NextReconciliationAt: &nextReconcile,
		}
		if err := uc.depositTxRepo.Create(txCtx, tx); err != nil {
			return subscriptionerrors.ErrDepositInitFailed()
		}
		return nil
	}); err != nil {
		return nil, err
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
		return nil, subscriptionerrors.ErrDepositMustBeInteger()
	}
	result, gatewayErr := uc.paymentGateway.CreateCheckoutSession(ctx, &payment.CheckoutSessionReq{OrderCode: tx.OrderCode, Amount: tx.Amount, Description: fmt.Sprintf("Nạp vào ví %d VNĐ", amountVND), ReturnUrl: returnURL, CancelUrl: cancelURL, ExpiresAt: expiresAt})
	if err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		locked, err := uc.depositTxRepo.GetByOrderCodeWithLock(txCtx, tx.OrderCode)
		if err != nil || locked == nil {
			return subscriptionerrors.ErrLockTransactionFailed()
		}
		if gatewayErr == nil && result != nil && result.Outcome == payment.OutcomeSucceeded {
			locked.Status = depositstatus.Pending
			locked.PaymentUrl = &result.CheckoutURL
		} else if result != nil && result.Outcome == payment.OutcomeKnownFailure && !result.Retryable {
			locked.Status = depositstatus.CreationFailed
			locked.FailureReason = &result.ErrorCode
		} else {
			locked.Status = depositstatus.ReconciliationRequired
			locked.ReconciliationAttempts++
		}
		if err := uc.depositTxRepo.Update(txCtx, locked); err != nil {
			return err
		}
		if userEmail, emailErr := uc.userContactRepo.GetEmailByUserID(txCtx, userID); emailErr == nil && userEmail != "" {
			_ = uc.notifContract.Send(txCtx, notificationContract.ChannelEmail, notificationContract.TemplateTopupSuccess, map[string]any{
				"Amount":     sharedmoney.ToFloat(amount),
				"NewBalance": sharedmoney.ToFloat(sharedmoney.Zero),
			}, userEmail)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if gatewayErr != nil && result != nil && result.Outcome == payment.OutcomeKnownFailure {
		return nil, errorutils.WrapError(gatewayErr, "Không thể khởi tạo liên kết thanh toán với cổng ngân hàng")
	}
	url := ""
	responseStatus := depositstatus.ReconciliationRequired
	if result != nil {
		url = result.CheckoutURL
		if result.Outcome == payment.OutcomeSucceeded {
			responseStatus = depositstatus.Pending
		} else if result.Outcome == payment.OutcomeKnownFailure && !result.Retryable {
			responseStatus = depositstatus.CreationFailed
		}
	}
	return &dto.PaymentLinkDTO{PaymentUrl: url, OrderCode: tx.OrderCode, PaymentStatus: responseStatus, ExpiresAt: &expiresAt, NextReconciliationAt: &nextReconcile}, nil
}
