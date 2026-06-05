package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/internal/shared/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/internal/shared/domain/constants/walletstatementtype"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/utils/timeutils"

	"github.com/shopspring/decimal"
)

type PaymentWebhookUseCase struct {
	cfg            *config.Config
	walletRepo     repositories.IUserWalletRepository
	depositTxRepo  repositories.IDepositTransactionRepository
	statementRepo  repositories.IWalletStatementRepository
	planRepo       repositories.ISubscriptionPlanRepository
	userSubRepo    repositories.IUserSubscriptionRepository
	paymentGateway payment.IPaymentGatewayService
	uow            shared_repos.IUnitOfWork
}

func NewPaymentWebhookUseCase(
	cfg *config.Config,
	walletRepo repositories.IUserWalletRepository,
	depositTxRepo repositories.IDepositTransactionRepository,
	statementRepo repositories.IWalletStatementRepository,
	planRepo repositories.ISubscriptionPlanRepository,
	userSubRepo repositories.IUserSubscriptionRepository,
	paymentGateway payment.IPaymentGatewayService,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IPaymentWebhookUseCase {
	return &PaymentWebhookUseCase{
		cfg:            cfg,
		walletRepo:     walletRepo,
		depositTxRepo:  depositTxRepo,
		statementRepo:  statementRepo,
		planRepo:       planRepo,
		userSubRepo:    userSubRepo,
		paymentGateway: paymentGateway,
		uow:            uow,
	}
}

func (uc *PaymentWebhookUseCase) ProcessWebhook(ctx context.Context, rawBody []byte, signature string) error {
	// Verify the webhook authenticity by validating its cryptographic signature with the gateway's public keys.
	_, err := uc.paymentGateway.VerifyWebhook(ctx, rawBody, signature)
	if err != nil {
		return apperror.NewBadRequest("Lỗi khi xác thực chữ ký webhook")
	}

	// Parse the webhook callback payload structure.
	var payload struct {
		Code    string `json:"code"`
		Desc    string `json:"desc"`
		Success bool   `json:"success"`
		Data    struct {
			OrderCode              int64       `json:"orderCode"`
			Amount                 json.Number `json:"amount"`
			Description            string      `json:"description"`
			AccountNumber          string      `json:"accountNumber"`
			Reference              string      `json:"reference"`
			TransactionDateTime    string      `json:"transactionDateTime"`
			Currency               string      `json:"currency"`
			PaymentLinkId          string      `json:"paymentLinkId"`
			Code                   string      `json:"code"`
			Desc                   string      `json:"desc"`
			CounterAccountBankId   string      `json:"counterAccountBankId"`
			CounterAccountBankName string      `json:"counterAccountBankName"`
			CounterAccountName     string      `json:"counterAccountName"`
			CounterAccountNumber   string      `json:"counterAccountNumber"`
			VirtualAccountName     string      `json:"virtualAccountName"`
			VirtualAccountNumber   string      `json:"virtualAccountNumber"`
		} `json:"data"`
		Signature string `json:"signature"`
	}

	decoder := json.NewDecoder(bytes.NewReader(rawBody))
	decoder.UseNumber() // Use json.Number to avoid precision loss on large integer numbers
	if err := decoder.Decode(&payload); err != nil {
		return apperror.NewBadRequest("Dữ liệu webhook không đúng định dạng cấu trúc")
	}

	// Check transaction status codes. Code "00" indicates success.
	// If the transaction failed on the gateway side, we just return nil (we don't credit users).
	if payload.Code != "00" || payload.Data.Code != "00" {
		return nil
	}

	// Convert amount string from payload to decimal.
	webhookAmount, err := decimal.NewFromString(payload.Data.Amount.String())
	if err != nil {
		return apperror.NewBadRequest("Số tiền webhook không hợp lệ")
	}

	// Query the matching pending deposit transaction record by its OrderCode.
	tx, err := uc.depositTxRepo.GetByOrderCode(ctx, payload.Data.OrderCode)
	if err != nil {
		return apperror.NewInternalError("Lỗi khi truy vấn thông tin giao dịch nạp tiền")
	}
	if tx == nil {
		return apperror.NewNotFound(fmt.Sprintf("Không tìm thấy giao dịch nạp tiền với mã đơn hàng %d", payload.Data.OrderCode))
	}

	// Verify the received payment amount matches the expected record amount to prevent fraud/underpayment.
	if !webhookAmount.Equal(tx.Amount) {
		return apperror.NewBadRequest("Số tiền thanh toán thực tế không khớp với số tiền của giao dịch")
	}

	// Optimistic check - if the transaction is already marked success, skip reprocessing (idempotency).
	if tx.Status == depositstatus.Success {
		return nil
	}

	// Serialize gateway data payload for audit logs.
	rawBytes, err := json.Marshal(payload.Data)
	if err != nil {
		return apperror.NewInternalError("Lỗi khi chuyển đổi thông tin cổng thanh toán")
	}
	detailsStr := string(rawBytes)

	reference := payload.Data.Reference

	// Execute transaction processing workflow within a database transaction.
	processPaymentWebhook := func(txCtx context.Context) error {
		// Re-fetch the transaction record using row-level locking (FOR UPDATE)
		// to guarantee thread safety and prevent double-processing (idempotency guard).
		lockedTx, err := uc.depositTxRepo.GetByOrderCodeWithLock(txCtx, payload.Data.OrderCode)
		if err != nil {
			return apperror.NewInternalError("Lỗi khi khóa dữ liệu giao dịch")
		}
		if lockedTx == nil {
			return apperror.NewNotFound(fmt.Sprintf("Không tìm thấy giao dịch nạp tiền với mã đơn hàng %d", payload.Data.OrderCode))
		}

		// Double-check the status inside the locked transaction context.
		if lockedTx.Status == depositstatus.Success {
			return nil
		}

		// Update the transaction status to SUCCESS and record references.
		now := timeutils.GetNow(uc.cfg.Database.TimeZone)
		lockedTx.Status = depositstatus.Success
		lockedTx.GatewayReference = &reference
		lockedTx.GatewayDetails = &detailsStr

		if err := uc.depositTxRepo.Update(txCtx, lockedTx); err != nil {
			return apperror.NewInternalError("Lỗi khi hoàn tất hồ sơ giao dịch nạp tiền")
		}

		// Route processing based on the TransactionType.
		switch lockedTx.TransactionType {
		case deposittransactiontype.WalletTopup:
			// Top-up flow: Credit the user's wallet.
			if err := uc.executeWalletTopUpWorkflow(txCtx, lockedTx, now); err != nil {
				return err
			}
		case deposittransactiontype.DirectPurchase:
			// Direct purchase flow: Apply/extend the subscription plan directly.
			if err := uc.executeDirectPurchaseWorkflow(txCtx, lockedTx, now); err != nil {
				return err
			}
		}

		return nil
	}
	return uc.uow.Execute(ctx, processPaymentWebhook)
}

// executeWalletTopUpWorkflow credits the user wallet and logs the statement.
func (uc *PaymentWebhookUseCase) executeWalletTopUpWorkflow(txCtx context.Context, tx *entities.DepositTransaction, now time.Time) error {
	desc := "Nạp tiền thành công vào ví tài khoản hệ thống"
	if err := processWalletTransaction(txCtx, uc.walletRepo, uc.statementRepo, tx.UserID, tx.Amount, walletstatementtype.Topup, desc, &tx.ID, now); err != nil {
		return err
	}
	return nil
}

// executeDirectPurchaseWorkflow fetches the targeted subscription plan and applies it.
func (uc *PaymentWebhookUseCase) executeDirectPurchaseWorkflow(txCtx context.Context, tx *entities.DepositTransaction, now time.Time) error {
	if tx.SubscriptionPlanID == nil {
		return apperror.NewBadRequest("Thiếu liên kết gói hội viên trong giao dịch thanh toán")
	}

	plan, err := uc.planRepo.GetByID(txCtx, *tx.SubscriptionPlanID)
	if err != nil {
		return apperror.NewInternalError("Lỗi khi tải thông tin gói hội viên đích")
	}
	if plan == nil {
		return apperror.NewNotFound("Không tìm thấy thông tin gói hội viên đích")
	}

	// Apply the subscription plan configuration to the user's subscription entity.
	if err := applySubscriptionPlan(txCtx, uc.userSubRepo, tx.UserID, plan, now); err != nil {
		return err
	}

	return nil
}

