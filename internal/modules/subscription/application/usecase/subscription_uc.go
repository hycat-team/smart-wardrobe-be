package usecase

import (
	"context"
	"fmt"

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
	subContract   contract.ISubscriptionModuleContract
	userSubRepo   repositories.IUserSubscriptionRepository
	planRepo      repositories.ISubscriptionPlanRepository
	walletRepo    repositories.IUserWalletRepository
	statementRepo repositories.IWalletStatementRepository
	quotaRepo     repositories.IUserDailyQuotaRepository
	cfg           *config.Config
	l             logger.Interface
}

func NewSubscriptionUseCase(
	uow shared_repos.IUnitOfWork,
	subContract contract.ISubscriptionModuleContract,
	userSubRepo repositories.IUserSubscriptionRepository,
	planRepo repositories.ISubscriptionPlanRepository,
	walletRepo repositories.IUserWalletRepository,
	statementRepo repositories.IWalletStatementRepository,
	quotaRepo repositories.IUserDailyQuotaRepository,
	cfg *config.Config,
	l logger.Interface,
) uc_interfaces.ISubscriptionUseCase {
	return &SubscriptionUseCase{
		uow:           uow,
		subContract:   subContract,
		userSubRepo:   userSubRepo,
		planRepo:      planRepo,
		walletRepo:    walletRepo,
		statementRepo: statementRepo,
		quotaRepo:     quotaRepo,
		cfg:           cfg,
		l:             l,
	}
}

func (uc *SubscriptionUseCase) GetDailyQuota(ctx context.Context, userID uuid.UUID) (*contract.UserSubscriptionDTO, error) {
	return uc.subContract.GetAndResetDailyQuota(ctx, userID)
}

func (uc *SubscriptionUseCase) ProcessScheduledRenewals(ctx context.Context) error {
	now := timeutils.GetNow(uc.cfg.Database.TimeZone)
	expiredSubs, err := uc.userSubRepo.GetActiveExpiredSubscriptions(ctx, now)
	if err != nil {
		return errorcode.NewInternalError("Lỗi khi truy vấn danh sách gói hội viên hết hạn")
	}

	freePlanID, err := uc.subContract.GetDefaultSubscriptionPlanID(ctx)
	if err != nil {
		return errorcode.NewInternalError("Lỗi khi tải thông tin cấu hình gói hội viên mặc định")
	}

	freePlan, err := uc.planRepo.GetByID(ctx, freePlanID)
	if err != nil || freePlan == nil {
		return errorcode.NewInternalError("Lỗi khi tải thông tin gói hội viên miễn phí tiêu chuẩn")
	}

	for _, sub := range expiredSubs {
		if sub.SubscriptionPlan == nil || sub.SubscriptionPlan.Price <= 0 {
			continue
		}

		plan := sub.SubscriptionPlan

		err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
			wallet, err := uc.walletRepo.GetByUserIDWithLock(txCtx, sub.UserID)
			if err != nil {
				return err
			}
			isNewWallet := false
			if wallet == nil {
				wallet = &entities.UserWallet{
					UserID:    sub.UserID,
					Balance:   0,
					Currency:  "VND",
					CreatedAt: now,
					UpdatedAt: now,
				}
				isNewWallet = true
			}

			if sub.IsAutoRenewEnabled && wallet.Balance >= plan.Price {
				prevBalance := wallet.Balance
				wallet.Balance -= plan.Price
				wallet.UpdatedAt = now

				if isNewWallet {
					if err := uc.walletRepo.Create(txCtx, wallet); err != nil {
						return err
					}
				} else {
					if err := uc.walletRepo.Update(txCtx, wallet); err != nil {
						return err
					}
				}

				days := 30
				if plan.DurationDays != nil {
					days = *plan.DurationDays
				}

				newExpiry := now.AddDate(0, 0, days)
				sub.ExpiresAt = &newExpiry
				sub.UpdatedAt = now

				if err := uc.userSubRepo.Update(txCtx, sub); err != nil {
					return err
				}

				statement := &entities.WalletStatement{
					UserID:          sub.UserID,
					Amount:          -plan.Price,
					TransactionType: "SUBSCRIPTION_RENEWAL",
					PreviousBalance: prevBalance,
					NewBalance:      wallet.Balance,
					Description:     fmt.Sprintf("Auto-renewed subscription plan %s", plan.Name),
				}

				if err := uc.statementRepo.Create(txCtx, statement); err != nil {
					return err
				}

				uc.l.Info(fmt.Sprintf("Successfully auto-renewed user %s with plan %s", sub.UserID, plan.Name))

			} else {
				sub.SubscriptionPlanID = freePlan.ID
				sub.ExpiresAt = nil
				sub.IsActive = false
				sub.UpdatedAt = now

				if err := uc.userSubRepo.Update(txCtx, sub); err != nil {
					return err
				}

				uc.l.Info(fmt.Sprintf("Auto-renewal disabled or insufficient funds, downgraded user %s back to standard free plan", sub.UserID))
			}

			return nil
		})

		if err != nil {
			uc.l.Error(fmt.Sprintf("Failed to process renewal sequence for user %s: %v", sub.UserID, err))
		}
	}

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

	sub.IsAutoRenewEnabled = enable
	sub.UpdatedAt = timeutils.GetNow(uc.cfg.Database.TimeZone)

	err = uc.userSubRepo.Update(ctx, sub)
	if err != nil {
		return false, err
	}

	return sub.IsAutoRenewEnabled, nil
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
