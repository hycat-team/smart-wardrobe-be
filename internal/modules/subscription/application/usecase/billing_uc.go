package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BillingUseCase struct {
	db             *gorm.DB
	walletRepo     repositories.IUserWalletRepository
	depositTxRepo  repositories.IDepositTransactionRepository
	statementRepo  repositories.IWalletStatementRepository
	planRepo       repositories.ISubscriptionPlanRepository
	userSubRepo    repositories.IUserSubscriptionRepository
	quotaRepo      repositories.IUserDailyQuotaRepository
	paymentGateway payment.IPaymentGatewayService
	cfg            *config.Config
}

func NewBillingUseCase(
	db *gorm.DB,
	walletRepo repositories.IUserWalletRepository,
	depositTxRepo repositories.IDepositTransactionRepository,
	statementRepo repositories.IWalletStatementRepository,
	planRepo repositories.ISubscriptionPlanRepository,
	userSubRepo repositories.IUserSubscriptionRepository,
	quotaRepo repositories.IUserDailyQuotaRepository,
	paymentGateway payment.IPaymentGatewayService,
	cfg *config.Config,
) uc_interfaces.IBillingUseCase {
	return &BillingUseCase{
		db:             db,
		walletRepo:     walletRepo,
		depositTxRepo:  depositTxRepo,
		statementRepo:  statementRepo,
		planRepo:       planRepo,
		userSubRepo:    userSubRepo,
		quotaRepo:      quotaRepo,
		paymentGateway: paymentGateway,
		cfg:            cfg,
	}
}

func (uc *BillingUseCase) GetWallet(ctx context.Context, userID uuid.UUID) (*dto.WalletDTO, error) {
	wallet, err := uc.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if wallet == nil {
		newWallet := &entities.UserWallet{
			UserID:    userID,
			Balance:   0,
			Currency:  "VND",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if err := uc.walletRepo.Create(ctx, newWallet); err != nil {
			return nil, err
		}
		wallet = newWallet
	}

	return &dto.WalletDTO{
		UserID:    wallet.UserID,
		Balance:   wallet.Balance,
		Currency:  wallet.Currency,
		UpdatedAt: wallet.UpdatedAt,
	}, nil
}

func (uc *BillingUseCase) GetWalletStatements(ctx context.Context, userID uuid.UUID) ([]*dto.WalletStatementDTO, error) {
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

func (uc *BillingUseCase) CreateWalletTopUp(ctx context.Context, userID uuid.UUID, req *dto.WalletTopUpReq) (*dto.PaymentLinkDTO, error) {
	tx := &entities.DepositTransaction{
		UserID:          userID,
		Amount:          req.Amount,
		Currency:        "VND",
		Status:          "PENDING",
		TransactionType: "WALLET_TOPUP",
		PaymentUrl:      nil,
	}

	if err := uc.depositTxRepo.Create(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to log deposit transaction: %w", err)
	}

	returnUrl := req.ReturnUrl
	if returnUrl == "" {
		returnUrl = uc.cfg.Server.FrontEndOrigin
	}
	cancelUrl := req.CancelUrl
	if cancelUrl == "" {
		cancelUrl = uc.cfg.Server.FrontEndOrigin
	}

	description := fmt.Sprintf("Top up wallet sum %d", int(tx.Amount))
	checkoutUrl, err := uc.paymentGateway.CreateCheckoutSession(ctx, &payment.CheckoutSessionReq{
		OrderCode:   tx.OrderCode,
		Amount:      tx.Amount,
		Description: description,
		ReturnUrl:   returnUrl,
		CancelUrl:   cancelUrl,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate checkout link: %w", err)
	}

	tx.PaymentUrl = &checkoutUrl
	if err := uc.depositTxRepo.Update(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to link checkout url: %w", err)
	}

	return &dto.PaymentLinkDTO{
		PaymentUrl: checkoutUrl,
		OrderCode:  tx.OrderCode,
	}, nil
}

func (uc *BillingUseCase) CreateDirectPurchase(ctx context.Context, userID uuid.UUID, req *dto.DirectPurchaseReq) (*dto.PaymentLinkDTO, error) {
	plan, err := uc.planRepo.GetByID(ctx, req.SubscriptionPlanID)
	if err != nil {
		return nil, fmt.Errorf("error searching subscription plan: %w", err)
	}
	if plan == nil {
		return nil, errors.New("subscription plan not found")
	}

	tx := &entities.DepositTransaction{
		UserID:             userID,
		Amount:             plan.Price,
		Currency:           "VND",
		Status:             "PENDING",
		TransactionType:    "DIRECT_PURCHASE",
		SubscriptionPlanID: &plan.ID,
		PaymentUrl:         nil,
	}

	if err := uc.depositTxRepo.Create(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to log purchase transaction: %w", err)
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

	checkoutUrl, err := uc.paymentGateway.CreateCheckoutSession(ctx, &payment.CheckoutSessionReq{
		OrderCode:   tx.OrderCode,
		Amount:      tx.Amount,
		Description: description,
		ReturnUrl:   returnUrl,
		CancelUrl:   cancelUrl,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate checkout link: %w", err)
	}

	tx.PaymentUrl = &checkoutUrl
	if err := uc.depositTxRepo.Update(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to link checkout url: %w", err)
	}

	return &dto.PaymentLinkDTO{
		PaymentUrl: checkoutUrl,
		OrderCode:  tx.OrderCode,
	}, nil
}

func (uc *BillingUseCase) ProcessWebhook(ctx context.Context, rawBody []byte, signature string) error {
	payloadMap, err := uc.paymentGateway.VerifyWebhook(ctx, rawBody, signature)
	if err != nil {
		return fmt.Errorf("failed to verify webhook signature: %w", err)
	}

	var orderCode int64
	if val, ok := payloadMap["orderCode"]; ok {
		switch v := val.(type) {
		case float64:
			orderCode = int64(v)
		case int64:
			orderCode = v
		case int:
			orderCode = int64(v)
		}
	}

	var code string
	if val, ok := payloadMap["code"]; ok {
		if s, ok := val.(string); ok {
			code = s
		}
	}

	webhookCode, _ := payloadMap["webhook_code"].(string)

	if webhookCode != "00" || code != "00" {
		return nil
	}

	tx, err := uc.depositTxRepo.GetByOrderCode(ctx, orderCode)
	if err != nil {
		return fmt.Errorf("error querying transaction mapping: %w", err)
	}
	if tx == nil {
		return fmt.Errorf("deposit transaction not found for order code %d", orderCode)
	}

	if tx.Status == "SUCCESS" {
		return nil
	}

	rawBytes, err := json.Marshal(payloadMap)
	if err != nil {
		return fmt.Errorf("failed to marshal gateway details: %w", err)
	}
	detailsStr := string(rawBytes)

	var reference string
	if val, ok := payloadMap["reference"]; ok {
		reference, _ = val.(string)
	}

	return uc.db.Transaction(func(dbTx *gorm.DB) error {
		tx.Status = "SUCCESS"
		tx.GatewayReference = &reference
		tx.GatewayDetails = &detailsStr

		if err := dbTx.Save(tx).Error; err != nil {
			return fmt.Errorf("failed to complete deposit record: %w", err)
		}

		switch tx.TransactionType {
		case "WALLET_TOPUP":
			var wallet entities.UserWallet
			err := dbTx.Where("user_id = ?", tx.UserID).First(&wallet).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					wallet = entities.UserWallet{
						UserID:    tx.UserID,
						Balance:   0,
						Currency:  "VND",
						CreatedAt: time.Now(),
					}
				} else {
					return fmt.Errorf("failed to read user wallet: %w", err)
				}
			}

			prevBalance := wallet.Balance
			wallet.Balance += tx.Amount
			wallet.UpdatedAt = time.Now()

			if err := dbTx.Save(&wallet).Error; err != nil {
				return fmt.Errorf("failed to update wallet balance: %w", err)
			}

			statement := entities.WalletStatement{
				UserID:          tx.UserID,
				Amount:          tx.Amount,
				TransactionType: "TOPUP",
				PreviousBalance: prevBalance,
				NewBalance:      wallet.Balance,
				ReferenceID:     &tx.ID,
				Description:     "Successfully topped up internal wallet balance",
			}

			if err := dbTx.Save(&statement).Error; err != nil {
				return fmt.Errorf("failed to write audit statement: %w", err)
			}

		case "DIRECT_PURCHASE":
			if tx.SubscriptionPlanID == nil {
				return errors.New("missing subscription plan link in transaction")
			}

			var plan entities.SubscriptionPlan
			if err := dbTx.Where("id = ?", *tx.SubscriptionPlanID).First(&plan).Error; err != nil {
				return fmt.Errorf("failed to load target plan: %w", err)
			}

			var sub entities.UserSubscription
			err := dbTx.Where("user_id = ?", tx.UserID).First(&sub).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					sub = entities.UserSubscription{
						UserID:    tx.UserID,
						CreatedAt: time.Now(),
					}
				} else {
					return fmt.Errorf("failed to read active subscription: %w", err)
				}
			}

			days := 30
			if plan.DurationDays != nil {
				days = *plan.DurationDays
			}

			var expiresAt time.Time
			if sub.IsActive && sub.ExpiresAt != nil && sub.ExpiresAt.After(time.Now()) {
				expiresAt = sub.ExpiresAt.AddDate(0, 0, days)
			} else {
				expiresAt = time.Now().AddDate(0, 0, days)
			}

			sub.SubscriptionPlanID = plan.ID
			sub.ExpiresAt = &expiresAt
			sub.IsActive = true
			sub.UpdatedAt = time.Now()

			if err := dbTx.Save(&sub).Error; err != nil {
				return fmt.Errorf("failed to link plan to subscription: %w", err)
			}

			var quota entities.UserDailyQuota
			err = dbTx.Where("user_id = ?", tx.UserID).First(&quota).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					quota = entities.UserDailyQuota{
						UserID:        tx.UserID,
						LastResetDate: time.Now(),
						CreatedAt:     time.Now(),
					}
					if err := dbTx.Save(&quota).Error; err != nil {
						return fmt.Errorf("failed to initialize daily quota log: %w", err)
					}
				} else {
					return fmt.Errorf("failed to read quota logs: %w", err)
				}
			}
		}

		return nil
	})
}

func (uc *BillingUseCase) GetPlans(ctx context.Context) ([]*entities.SubscriptionPlan, error) {
	plans, err := uc.planRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve subscription plans: %w", err)
	}
	return plans, nil
}
