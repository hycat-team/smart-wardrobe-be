package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/utils/timeutils"
)

type PaymentWebhookUseCase struct {
	walletRepo     repositories.IUserWalletRepository
	depositTxRepo  repositories.IDepositTransactionRepository
	statementRepo  repositories.IWalletStatementRepository
	planRepo       repositories.ISubscriptionPlanRepository
	userSubRepo    repositories.IUserSubscriptionRepository
	paymentGateway payment.IPaymentGatewayService
	uow            shared_repos.IUnitOfWork
	cfg            *config.Config
}

func NewPaymentWebhookUseCase(
	walletRepo repositories.IUserWalletRepository,
	depositTxRepo repositories.IDepositTransactionRepository,
	statementRepo repositories.IWalletStatementRepository,
	planRepo repositories.ISubscriptionPlanRepository,
	userSubRepo repositories.IUserSubscriptionRepository,
	paymentGateway payment.IPaymentGatewayService,
	uow shared_repos.IUnitOfWork,
	cfg *config.Config,
) uc_interfaces.IPaymentWebhookUseCase {
	return &PaymentWebhookUseCase{
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

func (uc *PaymentWebhookUseCase) ProcessWebhook(ctx context.Context, rawBody []byte, signature string) error {
	_, err := uc.paymentGateway.VerifyWebhook(ctx, rawBody, signature)
	if err != nil {
		return errorcode.NewBadRequest("Lỗi khi xác thực chữ ký webhook")
	}

	var payload struct {
		Code    string `json:"code"`
		Desc    string `json:"desc"`
		Success bool   `json:"success"`
		Data    struct {
			OrderCode              int64   `json:"orderCode"`
			Amount                 float64 `json:"amount"`
			Description            string  `json:"description"`
			AccountNumber          string  `json:"accountNumber"`
			Reference              string  `json:"reference"`
			TransactionDateTime    string  `json:"transactionDateTime"`
			Currency               string  `json:"currency"`
			PaymentLinkId          string  `json:"paymentLinkId"`
			Code                   string  `json:"code"`
			Desc                   string  `json:"desc"`
			CounterAccountBankId   string  `json:"counterAccountBankId"`
			CounterAccountBankName string  `json:"counterAccountBankName"`
			CounterAccountName     string  `json:"counterAccountName"`
			CounterAccountNumber   string  `json:"counterAccountNumber"`
			VirtualAccountName     string  `json:"virtualAccountName"`
			VirtualAccountNumber   string  `json:"virtualAccountNumber"`
		} `json:"data"`
		Signature string `json:"signature"`
	}

	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return errorcode.NewBadRequest("Dữ liệu webhook không đúng định dạng cấu trúc")
	}

	if payload.Code != "00" || payload.Data.Code != "00" {
		return nil
	}

	tx, err := uc.depositTxRepo.GetByOrderCode(ctx, payload.Data.OrderCode)
	if err != nil {
		return errorcode.NewInternalError("Lỗi khi truy vấn thông tin giao dịch nạp tiền")
	}
	if tx == nil {
		return errorcode.NewNotFound(fmt.Sprintf("Không tìm thấy giao dịch nạp tiền với mã đơn hàng %d", payload.Data.OrderCode))
	}

	if payload.Data.Amount < tx.Amount || payload.Data.Amount != tx.Amount {
		return errorcode.NewBadRequest("Số tiền thanh toán thực tế không khớp hoặc nhỏ hơn số tiền của giao dịch")
	}

	if tx.Status == depositstatus.Success {
		return nil
	}

	rawBytes, err := json.Marshal(payload.Data)
	if err != nil {
		return errorcode.NewInternalError("Lỗi khi chuyển đổi thông tin cổng thanh toán")
	}
	detailsStr := string(rawBytes)

	reference := payload.Data.Reference

	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		lockedTx, err := uc.depositTxRepo.GetByOrderCodeWithLock(txCtx, payload.Data.OrderCode)
		if err != nil {
			return errorcode.NewInternalError("Lỗi khi khóa dữ liệu giao dịch")
		}
		if lockedTx == nil {
			return errorcode.NewNotFound(fmt.Sprintf("Không tìm thấy giao dịch nạp tiền với mã đơn hàng %d", payload.Data.OrderCode))
		}

		if lockedTx.Status == depositstatus.Success {
			return nil
		}

		now := timeutils.GetNow(uc.cfg.Database.TimeZone)
		lockedTx.Status = depositstatus.Success
		lockedTx.GatewayReference = &reference
		lockedTx.GatewayDetails = &detailsStr

		if err := uc.depositTxRepo.Update(txCtx, lockedTx); err != nil {
			return errorcode.NewInternalError("Lỗi khi hoàn tất hồ sơ giao dịch nạp tiền")
		}

		switch lockedTx.TransactionType {
		case "WALLET_TOPUP":
			if err := uc.executeWalletTopUpWorkflow(txCtx, lockedTx, now); err != nil {
				return err
			}
		case "DIRECT_PURCHASE":
			if err := uc.executeDirectPurchaseWorkflow(txCtx, lockedTx, now); err != nil {
				return err
			}
		}

		return nil
	})
}

func (uc *PaymentWebhookUseCase) executeWalletTopUpWorkflow(txCtx context.Context, tx *entities.DepositTransaction, now time.Time) error {
	desc := "Nạp tiền thành công vào ví tài khoản hệ thống"
	if err := processWalletTransaction(txCtx, uc.walletRepo, uc.statementRepo, tx.UserID, tx.Amount, "TOPUP", desc, &tx.ID, now); err != nil {
		return err
	}
	return nil
}

func (uc *PaymentWebhookUseCase) executeDirectPurchaseWorkflow(txCtx context.Context, tx *entities.DepositTransaction, now time.Time) error {
	if tx.SubscriptionPlanID == nil {
		return errorcode.NewBadRequest("Thiếu liên kết gói hội viên trong giao dịch thanh toán")
	}

	plan, err := uc.planRepo.GetByID(txCtx, *tx.SubscriptionPlanID)
	if err != nil {
		return errorcode.NewInternalError("Lỗi khi tải thông tin gói hội viên đích")
	}
	if plan == nil {
		return errorcode.NewNotFound("Không tìm thấy thông tin gói hội viên đích")
	}

	if err := applySubscriptionPlan(txCtx, uc.userSubRepo, tx.UserID, plan, now); err != nil {
		return err
	}

	return nil
}
