package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/domain/constants/currency"
	"smart-wardrobe-be/internal/shared/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/internal/shared/domain/constants/walletstatementtype"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/utils/errorutils"
	"smart-wardrobe-be/pkg/utils/timeutils"

	"github.com/google/uuid"
)

type SubscriptionPurchaseUseCase struct {
	walletRepo     repositories.IUserWalletRepository
	depositTxRepo  repositories.IDepositTransactionRepository
	statementRepo  repositories.IWalletStatementRepository
	planRepo       repositories.ISubscriptionPlanRepository
	userSubRepo    repositories.IUserSubscriptionRepository
	paymentGateway payment.IPaymentGatewayService
	uow            shared_repos.IUnitOfWork
	cfg            *config.Config
}

func NewSubscriptionPurchaseUseCase(
	walletRepo repositories.IUserWalletRepository,
	depositTxRepo repositories.IDepositTransactionRepository,
	statementRepo repositories.IWalletStatementRepository,
	planRepo repositories.ISubscriptionPlanRepository,
	userSubRepo repositories.IUserSubscriptionRepository,
	paymentGateway payment.IPaymentGatewayService,
	uow shared_repos.IUnitOfWork,
	cfg *config.Config,
) uc_interfaces.ISubscriptionPurchaseUseCase {
	return &SubscriptionPurchaseUseCase{
		walletRepo:     walletRepo,
		depositTxRepo:  depositTxRepo,
		statementRepo:  statementRepo,
		planRepo:       planRepo,
		userSubRepo:    userSubRepo,
		paymentGateway: paymentGateway,
		uow:            uow,
		cfg:            cfg,
	}
}

func (uc *SubscriptionPurchaseUseCase) CreateDirectPurchase(ctx context.Context, userID uuid.UUID, req *dto.DirectPurchaseReq) (*dto.PaymentLinkDTO, error) {
	plan, err := uc.planRepo.GetBySlug(ctx, req.PlanSlug)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, errorcode.NewNotFound("Không tìm thấy thông tin gói hội viên yêu cầu")
	}

	if plan.Price <= 0.00 {
		return nil, errorcode.NewBadRequest("Không thể đăng ký trực tiếp gói hội viên miễn phí")
	}

	sub, err := uc.userSubRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	now := timeutils.GetNow(uc.cfg.Database.TimeZone)
	if sub != nil && sub.IsActive && sub.SubscriptionPlanID == plan.ID {
		if sub.ExpiresAt == nil || sub.ExpiresAt.After(now) {
			return nil, errorcode.NewConflict("Bạn đã đăng ký gói hội viên này rồi")
		}
	}

	var checkoutURL string
	var orderCode int64

	createDirectPurchase := func(txCtx context.Context) error {
		tx := &entities.DepositTransaction{
			UserID:             userID,
			Amount:             plan.Price,
			Currency:           currency.VND,
			Status:             depositstatus.Pending,
			TransactionType:    deposittransactiontype.DirectPurchase,
			SubscriptionPlanID: &plan.ID,
			OrderCode:          timeutils.GenerateOrderCode(),
			PaymentUrl:         nil,
		}

		if err := uc.depositTxRepo.Create(txCtx, tx); err != nil {
			return errorcode.NewInternalError("Lỗi khi tạo giao dịch thanh toán trực tiếp")
		}

		returnURL := req.ReturnUrl
		if returnURL == "" {
			returnURL = uc.cfg.PayOS.ReturnUrl
		}
		cancelURL := req.CancelUrl
		if cancelURL == "" {
			cancelURL = uc.cfg.PayOS.CancelUrl
		}

		normalizedPlanName := strings.ReplaceAll(plan.Name, " ", "")
		description := fmt.Sprintf("Purchase plan %s", normalizedPlanName)
		if len(description) > 25 {
			description = description[:25]
		}

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
			return errorcode.NewInternalError("Lỗi khi liên kết địa chỉ thanh toán")
		}

		orderCode = tx.OrderCode
		return nil
	}

	if err := uc.uow.Execute(ctx, createDirectPurchase); err != nil {
		return nil, err
	}

	return &dto.PaymentLinkDTO{
		PaymentUrl: checkoutURL,
		OrderCode:  orderCode,
	}, nil
}

func (uc *SubscriptionPurchaseUseCase) PurchasePlanWithWallet(ctx context.Context, userID uuid.UUID, planSlug string) error {
	plan, err := uc.planRepo.GetBySlug(ctx, planSlug)
	if err != nil {
		return errorcode.NewInternalError("Lỗi khi tìm kiếm thông tin gói cước")
	}
	if plan == nil {
		return errorcode.NewNotFound("Không tìm thấy thông tin gói hội viên yêu cầu")
	}

	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		now := timeutils.GetNow(uc.cfg.Database.TimeZone)
		description := fmt.Sprintf("Đăng ký gói hội viên %s thành công qua ví nội bộ", plan.Name)

		sub, isNewSub, err := uc.getOrInitLockedSubscriptionForPurchase(txCtx, userID, now)
		if err != nil {
			return err
		}

		if plan.Price == 0 {
			uc.applyPlanToSubscriptionEntity(sub, isNewSub, plan, now)
			if err := uc.persistSubscriptionForPurchase(txCtx, sub, isNewSub); err != nil {
				return err
			}
			return nil
		}

		wallet, isNewWallet, err := uc.getOrInitLockedWalletForPurchase(txCtx, userID, now)
		if err != nil {
			return err
		}

		prevBalance, err := uc.applyWalletDebitToLockedWallet(txCtx, wallet, isNewWallet, plan.Price, now)
		if err != nil {
			return err
		}

		if err := uc.createPurchaseWalletStatement(txCtx, userID, plan.Price, prevBalance, wallet.Balance, description); err != nil {
			return err
		}

		uc.applyPlanToSubscriptionEntity(sub, isNewSub, plan, now)
		if err := uc.persistSubscriptionForPurchase(txCtx, sub, isNewSub); err != nil {
			return err
		}

		return nil
	})
}

func (uc *SubscriptionPurchaseUseCase) getOrInitLockedSubscriptionForPurchase(txCtx context.Context, userID uuid.UUID, now time.Time) (*entities.UserSubscription, bool, error) {
	sub, err := uc.userSubRepo.GetByUserIDWithLock(txCtx, userID)
	if err != nil {
		return nil, false, errorcode.NewInternalError("Lỗi khi kiểm tra thông tin gói hội viên hiện tại")
	}
	if sub != nil {
		return sub, false, nil
	}

	return &entities.UserSubscription{
		UserID:    userID,
		CreatedAt: now,
	}, true, nil
}

func (uc *SubscriptionPurchaseUseCase) getOrInitLockedWalletForPurchase(txCtx context.Context, userID uuid.UUID, now time.Time) (*entities.UserWallet, bool, error) {
	wallet, err := uc.walletRepo.GetByUserIDWithLock(txCtx, userID)
	if err != nil {
		return nil, false, errorcode.NewInternalError("Lỗi khi truy vấn thông tin số dư ví")
	}
	if wallet != nil {
		return wallet, false, nil
	}

	return &entities.UserWallet{
		UserID:    userID,
		Balance:   0,
		Currency:  currency.VND,
		CreatedAt: now,
		UpdatedAt: now,
	}, true, nil
}

func (uc *SubscriptionPurchaseUseCase) applyWalletDebitToLockedWallet(txCtx context.Context, wallet *entities.UserWallet, isNewWallet bool, amount float64, now time.Time) (float64, error) {
	if wallet.Balance < amount {
		return 0, errorcode.NewBadRequest("Số dư tài khoản nội bộ không đủ để thực hiện giao dịch")
	}

	prevBalance := wallet.Balance
	wallet.Balance -= amount
	wallet.UpdatedAt = now

	if isNewWallet {
		if err := uc.walletRepo.Create(txCtx, wallet); err != nil {
			return 0, errorcode.NewInternalError("Lỗi khi khởi tạo ví mới")
		}
	} else {
		if err := uc.walletRepo.Update(txCtx, wallet); err != nil {
			return 0, errorcode.NewInternalError("Lỗi khi cập nhật số dư ví tài khoản")
		}
	}

	return prevBalance, nil
}

func (uc *SubscriptionPurchaseUseCase) createPurchaseWalletStatement(txCtx context.Context, userID uuid.UUID, amount float64, prevBalance float64, newBalance float64, description string) error {
	statement := &entities.WalletStatement{
		UserID:          userID,
		Amount:          -amount,
		TransactionType: walletstatementtype.SubscriptionPurchase,
		PreviousBalance: prevBalance,
		NewBalance:      newBalance,
		Description:     description,
	}

	if err := uc.statementRepo.Create(txCtx, statement); err != nil {
		return errorcode.NewInternalError("Lỗi khi lưu lịch sử biến động số dư ví")
	}

	return nil
}

func (uc *SubscriptionPurchaseUseCase) applyPlanToSubscriptionEntity(sub *entities.UserSubscription, isNewSub bool, plan *entities.SubscriptionPlan, now time.Time) {
	var expiresAt *time.Time

	if plan.DurationDays == nil {
		expiresAt = nil
	} else {
		days := *plan.DurationDays
		var expiry time.Time
		if !isNewSub && sub.IsActive && sub.ExpiresAt != nil && sub.ExpiresAt.After(now) && sub.SubscriptionPlanID == plan.ID {
			expiry = sub.ExpiresAt.AddDate(0, 0, days)
		} else {
			expiry = now.AddDate(0, 0, days)
		}
		expiresAt = &expiry
	}

	sub.SubscriptionPlanID = plan.ID
	sub.ExpiresAt = expiresAt
	sub.IsActive = true
	sub.UpdatedAt = now
}

func (uc *SubscriptionPurchaseUseCase) persistSubscriptionForPurchase(txCtx context.Context, sub *entities.UserSubscription, isNewSub bool) error {
	if isNewSub {
		if err := uc.userSubRepo.Create(txCtx, sub); err != nil {
			return errorcode.NewInternalError("Lỗi khi kích hoạt gói hội viên mới")
		}
		return nil
	}

	if err := uc.userSubRepo.Update(txCtx, sub); err != nil {
		return errorcode.NewInternalError("Lỗi khi cập nhật thời hạn gói hội viên")
	}

	return nil
}
