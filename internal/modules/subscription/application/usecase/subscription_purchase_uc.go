package usecase

import (
	"context"
	"fmt"
	"strings"

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
		return nil, errorcode.NewInternalError("Lỗi khi tìm kiếm thông tin gói hội viên")
	}
	if plan == nil {
		return nil, errorcode.NewNotFound("Không tìm thấy thông tin gói hội viên yêu cầu")
	}

	if plan.Price <= 0.00 {
		return nil, errorcode.NewBadRequest("Không thể đăng ký trực tiếp gói hội viên miễn phí")
	}

	sub, err := uc.userSubRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, errorcode.NewInternalError("Lỗi khi kiểm tra thông tin gói hội viên hiện tại")
	}
	now := timeutils.GetNow(uc.cfg.Database.TimeZone)
	if sub != nil && sub.IsActive && sub.SubscriptionPlanID == plan.ID {
		if sub.ExpiresAt == nil || sub.ExpiresAt.After(now) {
			return nil, errorcode.NewConflict("Bạn đã đăng ký gói hội viên này rồi")
		}
	}

	var checkoutUrl string
	var orderCode int64

	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		tx := &entities.DepositTransaction{
			UserID:             userID,
			Amount:             plan.Price,
			Currency:           "VND",
			Status:             depositstatus.Pending,
			TransactionType:    "DIRECT_PURCHASE",
			SubscriptionPlanID: &plan.ID,
			PaymentUrl:         nil,
		}

		if err := uc.depositTxRepo.Create(txCtx, tx); err != nil {
			return errorcode.NewInternalError("Lỗi khi tạo giao dịch thanh toán trực tiếp")
		}

		returnUrl := req.ReturnUrl
		if returnUrl == "" {
			returnUrl = uc.cfg.Server.FrontEndOrigin
		}
		cancelUrl := req.CancelUrl
		if cancelUrl == "" {
			cancelUrl = uc.cfg.Server.FrontEndOrigin
		}

		normalizedPlanName := strings.ReplaceAll(plan.Name, " ", "")
		description := fmt.Sprintf("Purchase plan %s", normalizedPlanName)
		if len(description) > 25 {
			description = description[:25]
		}

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
			return errorcode.NewInternalError("Lỗi khi liên kết địa chỉ thanh toán")
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
		desc := fmt.Sprintf("Đăng ký gói hội viên %s thành công qua ví nội bộ", plan.Name)
		if err := processWalletTransaction(txCtx, uc.walletRepo, uc.statementRepo, userID, -plan.Price, "SUBSCRIPTION_PURCHASE", desc, nil, now); err != nil {
			return err
		}

		if err := applySubscriptionPlan(txCtx, uc.userSubRepo, userID, plan, now); err != nil {
			return err
		}

		return nil
	})
}
