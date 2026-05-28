package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SubscriptionUseCase struct {
	db                   *gorm.DB
	subContract          contract.ISubscriptionModuleContract
	userSubRepo          repositories.IUserSubscriptionRepository
	planRepo             repositories.ISubscriptionPlanRepository
	walletRepo           repositories.IUserWalletRepository
	statementRepo        repositories.IWalletStatementRepository
	quotaRepo            repositories.IUserDailyQuotaRepository
}

func NewSubscriptionUseCase(
	db *gorm.DB,
	subContract contract.ISubscriptionModuleContract,
	userSubRepo repositories.IUserSubscriptionRepository,
	planRepo repositories.ISubscriptionPlanRepository,
	walletRepo repositories.IUserWalletRepository,
	statementRepo repositories.IWalletStatementRepository,
	quotaRepo repositories.IUserDailyQuotaRepository,
) uc_interfaces.ISubscriptionUseCase {
	return &SubscriptionUseCase{
		db:            db,
		subContract:   subContract,
		userSubRepo:   userSubRepo,
		planRepo:      planRepo,
		walletRepo:    walletRepo,
		statementRepo: statementRepo,
		quotaRepo:     quotaRepo,
	}
}

func (uc *SubscriptionUseCase) GetDailyQuota(ctx context.Context, userID uuid.UUID) (*contract.UserSubscriptionDTO, error) {
	return uc.subContract.GetAndResetDailyQuota(ctx, userID)
}

func (uc *SubscriptionUseCase) ProcessScheduledRenewals(ctx context.Context) error {
	var expiredSubs []*entities.UserSubscription
	err := uc.db.Preload("SubscriptionPlan").
		Where("is_active = ? AND expires_at <= ?", true, time.Now()).
		Find(&expiredSubs).Error
	if err != nil {
		return fmt.Errorf("failed to query expired subscriptions: %w", err)
	}

	freePlanID, err := uc.subContract.GetDefaultSubscriptionPlanID(ctx)
	if err != nil {
		return fmt.Errorf("failed to load fallback plan identification: %w", err)
	}

	var freePlan entities.SubscriptionPlan
	if err := uc.db.Where("id = ?", freePlanID).First(&freePlan).Error; err != nil {
		return fmt.Errorf("failed to load standard free tier plan: %w", err)
	}

	for _, sub := range expiredSubs {
		if sub.SubscriptionPlan == nil || sub.SubscriptionPlan.Price <= 0 {
			continue
		}

		plan := sub.SubscriptionPlan

		err = uc.db.Transaction(func(dbTx *gorm.DB) error {
			var wallet entities.UserWallet
			err := dbTx.Where("user_id = ?", sub.UserID).First(&wallet).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					wallet = entities.UserWallet{
						UserID:    sub.UserID,
						Balance:   0,
						Currency:  "VND",
						CreatedAt: time.Now(),
					}
				} else {
					return err
				}
			}

			if sub.IsAutoRenewEnabled && wallet.Balance >= plan.Price {
				prevBalance := wallet.Balance
				wallet.Balance -= plan.Price
				wallet.UpdatedAt = time.Now()

				if err := dbTx.Save(&wallet).Error; err != nil {
					return err
				}

				days := 30
				if plan.DurationDays != nil {
					days = *plan.DurationDays
				}

				newExpiry := time.Now().AddDate(0, 0, days)
				sub.ExpiresAt = &newExpiry
				sub.UpdatedAt = time.Now()

				if err := dbTx.Save(sub).Error; err != nil {
					return err
				}

				statement := entities.WalletStatement{
					UserID:          sub.UserID,
					Amount:          -plan.Price,
					TransactionType: "SUBSCRIPTION_RENEWAL",
					PreviousBalance: prevBalance,
					NewBalance:      wallet.Balance,
					Description:     fmt.Sprintf("Auto-renewed subscription plan %s", plan.Name),
				}

				if err := dbTx.Save(&statement).Error; err != nil {
					return err
				}

				log.Printf("Successfully auto-renewed user %s with plan %s", sub.UserID, plan.Name)

			} else {
				sub.SubscriptionPlanID = freePlan.ID
				sub.ExpiresAt = nil
				sub.UpdatedAt = time.Now()

				if err := dbTx.Save(sub).Error; err != nil {
					return err
				}

				var quota entities.UserDailyQuota
				err = dbTx.Where("user_id = ?", sub.UserID).First(&quota).Error
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						quota = entities.UserDailyQuota{
							UserID:        sub.UserID,
							LastResetDate: time.Now(),
							CreatedAt:     time.Now(),
						}
						if err := dbTx.Save(&quota).Error; err != nil {
							return err
						}
					} else {
						return err
					}
				}

				log.Printf("Auto-renewal disabled or insufficient funds, downgraded user %s back to standard free plan", sub.UserID)
			}

			return nil
		})

		if err != nil {
			log.Printf("Failed to process renewal sequence for user %s: %v", sub.UserID, err)
		}
	}

	return nil
}

func (uc *SubscriptionUseCase) ToggleAutoRenew(ctx context.Context, userID uuid.UUID) (bool, error) {
	var sub entities.UserSubscription
	err := uc.db.Where("user_id = ?", userID).First(&sub).Error
	if err != nil {
		return false, fmt.Errorf("failed to retrieve user subscription: %w", err)
	}

	sub.IsAutoRenewEnabled = !sub.IsAutoRenewEnabled
	sub.UpdatedAt = time.Now()

	err = uc.db.Save(&sub).Error
	if err != nil {
		return false, fmt.Errorf("failed to save user subscription: %w", err)
	}

	return sub.IsAutoRenewEnabled, nil
}
