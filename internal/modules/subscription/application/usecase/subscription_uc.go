package usecase

import (
	"context"
	"fmt"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/logger"
	"smart-wardrobe-be/pkg/utils/timeutils"

	"github.com/google/uuid"
)

type SubscriptionUseCase struct {
	uow           shared_repos.IUnitOfWork
	userSubRepo   repositories.IUserSubscriptionRepository
	planRepo      repositories.ISubscriptionPlanRepository
	walletRepo    repositories.IUserWalletRepository
	statementRepo repositories.IWalletStatementRepository
	quotaRepo     repositories.IUserDailyQuotaRepository
	cfg           *config.Config
	log           logger.Interface

	planContract  contract.ISubscriptionPlanContract
	quotaContract contract.IUserQuotaContract
}

func NewSubscriptionUseCase(
	uow shared_repos.IUnitOfWork,
	userSubRepo repositories.IUserSubscriptionRepository,
	planRepo repositories.ISubscriptionPlanRepository,
	walletRepo repositories.IUserWalletRepository,
	statementRepo repositories.IWalletStatementRepository,
	quotaRepo repositories.IUserDailyQuotaRepository,
	cfg *config.Config,
	log logger.Interface,
	planContract contract.ISubscriptionPlanContract,
	quotaContract contract.IUserQuotaContract,
) uc_interfaces.ISubscriptionUseCase {
	return &SubscriptionUseCase{
		uow:           uow,
		userSubRepo:   userSubRepo,
		planRepo:      planRepo,
		walletRepo:    walletRepo,
		statementRepo: statementRepo,
		quotaRepo:     quotaRepo,
		cfg:           cfg,
		log:           log,
		planContract:  planContract,
		quotaContract: quotaContract,
	}
}

func (uc *SubscriptionUseCase) GetDailyQuota(ctx context.Context, userID uuid.UUID) (*contract.UserSubscriptionDTO, error) {
	return uc.quotaContract.GetAndResetDailyQuota(ctx, userID)
}

func (uc *SubscriptionUseCase) GetPlans(ctx context.Context) ([]*dto.SubscriptionPlanDTO, error) {
	plans, err := uc.planRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	dtoPlans := make([]*dto.SubscriptionPlanDTO, 0, len(plans))
	for _, plan := range plans {
		dtoPlans = append(dtoPlans, &dto.SubscriptionPlanDTO{
			ID:                 plan.ID,
			Slug:               plan.Slug,
			Name:               plan.Name,
			Price:              plan.Price,
			MaxWardrobeItems:   plan.MaxWardrobeItems,
			MaxOutfits:         plan.MaxOutfits,
			AiOutfitDailyQuota: plan.AiOutfitDailyQuota,
			AiChatDailyQuota:   plan.AiChatDailyQuota,
			DurationDays:       plan.DurationDays,
		})
	}
	return dtoPlans, nil
}

const (
	renewalStatusRenewed    = "renewed"
	renewalStatusDowngraded = "downgraded"
	renewalStatusSkipped    = "skipped"
)

func (uc *SubscriptionUseCase) ProcessScheduledRenewals(ctx context.Context) error {
	now := timeutils.GetNow(uc.cfg.Database.TimeZone)
	freePlanID, err := uc.planContract.GetDefaultSubscriptionPlanID(ctx)
	if err != nil {
		return errorcode.NewInternalError("Lỗi khi tải thông tin cấu hình gói hội viên mặc định")
	}

	var lastUserID uuid.UUID
	var lastExpiresAt time.Time
	limit := 100

	var renewedCount, downgradedCount, skippedCount, failedCount int

	for {
		expiredSubs, err := uc.userSubRepo.GetActiveExpiredSubscriptionsBatch(ctx, now, lastUserID, lastExpiresAt, limit)
		if err != nil {
			return errorcode.NewInternalError("Lỗi khi truy vấn danh sách gói hội viên hết hạn")
		}

		if len(expiredSubs) == 0 {
			break
		}

		for _, sub := range expiredSubs {
			if sub.ExpiresAt == nil {
				skippedCount++
				continue
			}

			cursorUserID := sub.UserID
			cursorExpiresAt := *sub.ExpiresAt

			lastUserID = cursorUserID
			lastExpiresAt = cursorExpiresAt

			resultStatus := renewalStatusSkipped

			processSubFn := func(txCtx context.Context) error {
				lockedSub, err := uc.userSubRepo.GetActiveExpiredSubscriptionByUserIDWithLock(txCtx, sub.UserID, now)
				if err != nil {
					return err
				}
				if lockedSub == nil {
					resultStatus = renewalStatusSkipped
					return nil
				}

				if lockedSub.SubscriptionPlan == nil || lockedSub.SubscriptionPlan.Price <= 0 {
					resultStatus = renewalStatusSkipped
					return nil
				}

				plan := lockedSub.SubscriptionPlan

				if !lockedSub.IsAutoRenewEnabled {
					lockedSub.SubscriptionPlanID = freePlanID
					lockedSub.ExpiresAt = nil
					lockedSub.IsActive = false
					lockedSub.UpdatedAt = now

					if err := uc.userSubRepo.Update(txCtx, lockedSub); err != nil {
						return err
					}
					resultStatus = renewalStatusDowngraded
					return nil
				}

				wallet, err := uc.walletRepo.GetByUserIDWithLock(txCtx, lockedSub.UserID)
				if err != nil {
					return err
				}

				if wallet != nil && wallet.Balance >= plan.Price {
					prevBalance := wallet.Balance
					wallet.Balance -= plan.Price
					wallet.UpdatedAt = now

					if err := uc.walletRepo.Update(txCtx, wallet); err != nil {
						return err
					}

					days := 30
					if plan.DurationDays != nil {
						days = *plan.DurationDays
					}

					newExpiry := now.AddDate(0, 0, days)
					lockedSub.ExpiresAt = &newExpiry
					lockedSub.UpdatedAt = now

					if err := uc.userSubRepo.Update(txCtx, lockedSub); err != nil {
						return err
					}

					statement := &entities.WalletStatement{
						UserID:          lockedSub.UserID,
						Amount:          -plan.Price,
						TransactionType: "SUBSCRIPTION_RENEWAL",
						PreviousBalance: prevBalance,
						NewBalance:      wallet.Balance,
						Description:     fmt.Sprintf("Auto-renewed subscription plan %s", plan.Name),
					}

					if err := uc.statementRepo.Create(txCtx, statement); err != nil {
						return err
					}
					resultStatus = renewalStatusRenewed
				} else {
					lockedSub.SubscriptionPlanID = freePlanID
					lockedSub.ExpiresAt = nil
					lockedSub.IsActive = false
					lockedSub.UpdatedAt = now

					if err := uc.userSubRepo.Update(txCtx, lockedSub); err != nil {
						return err
					}
					resultStatus = renewalStatusDowngraded
				}

				return nil
			}

			if err = uc.uow.Execute(ctx, processSubFn); err != nil {
				failedCount++
				uc.log.Error(fmt.Sprintf("Failed to process renewal sequence for user %s: %v", sub.UserID, err))
			} else {
				switch resultStatus {
				case renewalStatusRenewed:
					renewedCount++
				case renewalStatusDowngraded:
					downgradedCount++
				case renewalStatusSkipped:
					skippedCount++
				}
			}
		}

		if len(expiredSubs) < limit {
			break
		}
	}

	uc.log.Info(fmt.Sprintf("Processed scheduled renewals summary: renewed=%d, downgraded=%d, skipped=%d, failed=%d", renewedCount, downgradedCount, skippedCount, failedCount))
	return nil
}

func (uc *SubscriptionUseCase) SetAutoRenewStatus(ctx context.Context, userID uuid.UUID, enable bool) (bool, error) {
	sub, err := uc.userSubRepo.GetByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	if sub == nil {
		return false, errorcode.NewNotFound("Không tìm thấy thông tin gói hội viên của người dùng.")
	}

	if sub.IsAutoRenewEnabled == enable {
		return sub.IsAutoRenewEnabled, nil
	}

	if enable && sub.ExpiresAt != nil && sub.ExpiresAt.Before(time.Now()) {
		return false, errorcode.NewBadRequest("Gói hội viên đã hết hạn, không thể thiết lập tự động gia hạn.")
	}

	sub.IsAutoRenewEnabled = enable
	sub.UpdatedAt = timeutils.GetNow(uc.cfg.Database.TimeZone)

	err = uc.userSubRepo.Update(ctx, sub)
	if err != nil {
		return false, err
	}

	return sub.IsAutoRenewEnabled, nil
}

func (uc *SubscriptionUseCase) getOrCreateUserSubscription(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	sub, err := uc.userSubRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		defaultPlanID, err := uc.planContract.GetDefaultSubscriptionPlanID(ctx)
		if err != nil {
			return nil, err
		}
		defaultPlan, err := uc.planRepo.GetByID(ctx, defaultPlanID)
		if err != nil {
			return nil, err
		}
		if defaultPlan == nil {
			return nil, errorcode.NewNotFound("Không tìm thấy cấu hình gói hội viên mặc định")
		}

		sub = &entities.UserSubscription{
			UserID:             userID,
			SubscriptionPlanID: defaultPlanID,
			IsActive:           true,
			SubscriptionPlan:   defaultPlan,
		}
		err = uc.userSubRepo.Create(ctx, sub)
		if err != nil {
			return nil, err
		}
	}
	return sub, nil
}

func (uc *SubscriptionUseCase) getOrCreateUserDailyQuota(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error) {
	quota, err := uc.quotaRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if quota == nil {
		quota = &entities.UserDailyQuota{
			UserID:               userID,
			OutfitRecommendCount: 0,
			AiUsageCount:         0,
			LastResetDate:        time.Now(),
		}
		err = uc.quotaRepo.Create(ctx, quota)
		if err != nil {
			return nil, err
		}
	}
	return quota, nil
}

// GetUserSubscription loads subscription details and daily quotas aggregated from multiple tables
func (uc *SubscriptionUseCase) GetUserSubscription(ctx context.Context, userID uuid.UUID) (*contract.UserSubscriptionDTO, error) {
	sub, err := uc.getOrCreateUserSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}

	quota, err := uc.getOrCreateUserDailyQuota(ctx, userID)
	if err != nil {
		return nil, err
	}

	plan := sub.SubscriptionPlan
	if plan == nil {
		p, err := uc.planRepo.GetByID(ctx, sub.SubscriptionPlanID)
		if err != nil {
			return nil, err
		}
		if p == nil {
			return nil, errorcode.NewNotFound("Không tìm thấy thông tin của gói hội viên")
		}
		plan = p
	}

	return &contract.UserSubscriptionDTO{
		PlanID:               plan.ID,
		PlanName:             plan.Name,
		PlanSlug:             plan.Slug,
		ExpiresAt:            sub.ExpiresAt,
		IsAutoRenewEnabled:   sub.IsAutoRenewEnabled,
		MaxWardrobeItems:     plan.MaxWardrobeItems,
		MaxOutfits:           plan.MaxOutfits,
		AiOutfitDailyQuota:   plan.AiOutfitDailyQuota,
		AiChatDailyQuota:     plan.AiChatDailyQuota,
		OutfitRecommendCount: quota.OutfitRecommendCount,
		AiUsageCount:         quota.AiUsageCount,
		LastResetDate:        quota.LastResetDate,
	}, nil
}

// GetUserSubscriptionOverview loads ONLY subscription details without high-frequency daily quota metrics
func (uc *SubscriptionUseCase) GetUserSubscriptionOverview(ctx context.Context, userID uuid.UUID) (*contract.UserSubscriptionOverviewDTO, error) {
	sub, err := uc.getOrCreateUserSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}

	plan := sub.SubscriptionPlan
	if plan == nil {
		p, err := uc.planRepo.GetByID(ctx, sub.SubscriptionPlanID)
		if err != nil {
			return nil, err
		}
		if p == nil {
			return nil, errorcode.NewNotFound("Không tìm thấy thông tin của gói hội viên")
		}
		plan = p
	}

	return &contract.UserSubscriptionOverviewDTO{
		PlanID:             plan.ID,
		PlanName:           plan.Name,
		PlanSlug:           plan.Slug,
		ExpiresAt:          sub.ExpiresAt,
		IsAutoRenewEnabled: sub.IsAutoRenewEnabled,
		MaxWardrobeItems:   plan.MaxWardrobeItems,
		MaxOutfits:         plan.MaxOutfits,
		AiOutfitDailyQuota: plan.AiOutfitDailyQuota,
		AiChatDailyQuota:   plan.AiChatDailyQuota,
	}, nil
}
