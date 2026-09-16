package webhook

import (
	"context"
	"encoding/json"
	"time"

	identity_contract "smart-wardrobe-be/services/account/contract/identity"
	notificationContract "smart-wardrobe-be/services/account/internal/modules/notification/presentation/contract"
	subscriptionerrors "smart-wardrobe-be/services/account/internal/modules/subscription/application/errors"
	subscriptioncaching "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/caching"
	"smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/payment"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/depositstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/config"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/pkg/logger"

	"go.uber.org/zap"
)

type WebhookPayload struct {
	Code      string             `json:"code"`
	Desc      string             `json:"desc"`
	Success   bool               `json:"success"`
	Data      WebhookPayloadData `json:"data"`
	Signature string             `json:"signature"`
}

type WebhookPayloadData struct {
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
}

type PaymentWebhookUseCase struct {
	cfg                   *config.Config
	walletRepo            repositories.IUserWalletRepository
	depositTxRepo         repositories.IDepositTransactionRepository
	statementRepo         repositories.IWalletStatementRepository
	planRepo              repositories.ISubscriptionPlanRepository
	userSubRepo           repositories.IUserSubscriptionRepository
	paymentGateway        payment.IPaymentGatewayService
	uow                   shared_repos.IUnitOfWork
	paymentEventRepo      repositories.IPaymentEventRepository
	inboxRepo             repositories.IWebhookInboxRepository
	subscriptionEventRepo repositories.IUserSubscriptionEventRepository
	notifContract         notificationContract.INotificationContract
	userBriefContract     identity_contract.IUserBriefContract
	log                   logger.Interface
	overviewCache         subscriptioncaching.ISubscriptionOverviewCache
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
	paymentEventRepo repositories.IPaymentEventRepository,
	inboxRepo repositories.IWebhookInboxRepository,
	subscriptionEventRepo repositories.IUserSubscriptionEventRepository,
	notifContract notificationContract.INotificationContract,
	userBriefContract identity_contract.IUserBriefContract,
	log logger.Interface,
	overviewCache subscriptioncaching.ISubscriptionOverviewCache,
) uc_interfaces.IPaymentWebhookUseCase {
	return &PaymentWebhookUseCase{
		cfg:                   cfg,
		walletRepo:            walletRepo,
		depositTxRepo:         depositTxRepo,
		statementRepo:         statementRepo,
		planRepo:              planRepo,
		userSubRepo:           userSubRepo,
		paymentGateway:        paymentGateway,
		uow:                   uow,
		paymentEventRepo:      paymentEventRepo,
		inboxRepo:             inboxRepo,
		subscriptionEventRepo: subscriptionEventRepo,
		notifContract:         notifContract,
		userBriefContract:     userBriefContract,
		log:                   log,
		overviewCache:         overviewCache,
	}
}

func (uc *PaymentWebhookUseCase) ProcessWebhook(ctx context.Context, rawBody []byte, signature string) error {
	payload, err := uc.parseVerifiedWebhookPayload(ctx, rawBody, signature)
	if err != nil {
		return err
	}
	inbox, err := uc.persistWebhookInbox(ctx, payload)
	if err != nil {
		return err
	}

	if !uc.isSuccessfulWebhook(payload) {
		uc.logIgnoredWebhook(payload)
		if inbox != nil {
			_ = uc.inboxRepo.MarkProcessed(ctx, inbox.ID, time.Now().UTC())
		}
		return nil
	}

	tx, webhookAmount, err := uc.loadAndValidateDepositTransaction(ctx, payload)
	if err != nil {
		if inbox != nil {
			_ = uc.inboxRepo.MarkInvestigation(ctx, inbox.ID, time.Now().UTC(), err.Error())
		}
		return err
	}

	if tx.Status == depositstatus.Success {
		return nil
	}

	detailsStr, err := uc.serializeWebhookDetails(payload)
	if err != nil {
		return err
	}

	if err := uc.processLockedWebhookTransaction(ctx, payload, tx, webhookAmount, detailsStr); err != nil {
		if inbox != nil {
			_ = uc.inboxRepo.MarkRetry(ctx, inbox.ID, time.Now().UTC(), err.Error())
		}
		uc.log.Error("Failed to process payment webhook",
			zap.Int64("order_code", payload.Data.OrderCode),
			zap.String("gateway_reference", payload.Data.Reference),
			zap.String("currency", payload.Data.Currency),
			zap.String("result", "failed"),
			zap.Error(err),
		)
		return err
	}
	if inbox != nil {
		_ = uc.inboxRepo.MarkProcessed(ctx, inbox.ID, time.Now().UTC())
	}
	// The payment may have upgraded/downgraded the subscription — drop the
	// cached overview after success (spec 019 FR-008).
	if uc.overviewCache != nil {
		_ = uc.overviewCache.Invalidate(ctx, tx.UserID)
	}

	uc.log.Info("Processed payment webhook successfully",
		zap.Int64("order_code", payload.Data.OrderCode),
		zap.String("gateway_reference", payload.Data.Reference),
		zap.String("currency", payload.Data.Currency),
		zap.String("result", "processed"),
	)
	return nil
}

func (uc *PaymentWebhookUseCase) CompleteVerifiedPayment(ctx context.Context, info *payment.PaymentLinkInfo) error {
	if info == nil || info.Status != payment.ProviderPaid {
		return subscriptionerrors.ErrProcessPaymentGatewayFailed()
	}
	amount := info.AmountPaid
	if amount.IsZero() {
		amount = info.Amount
	}
	payload := &WebhookPayload{Code: "00", Success: true, Data: WebhookPayloadData{OrderCode: info.OrderCode, Amount: json.Number(amount.String()), Reference: info.Reference, Currency: info.Currency, PaymentLinkId: info.PaymentLinkID, Code: "00"}}
	tx, verifiedAmount, err := uc.loadAndValidateDepositTransaction(ctx, payload)
	if err != nil {
		return err
	}
	details, err := uc.serializeWebhookDetails(payload)
	if err != nil {
		return err
	}
	return uc.processLockedWebhookTransaction(ctx, payload, tx, verifiedAmount, details)
}
