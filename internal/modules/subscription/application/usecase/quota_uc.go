package usecase

import (
	"context"
	"time"

	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type UserQuotaUseCase struct {
	quotaRepo repositories.IUserDailyQuotaRepository
	subRepo   repositories.IUserSubscriptionRepository
	planRepo  repositories.ISubscriptionPlanRepository
}

func NewUserQuotaUseCase(
	quotaRepo repositories.IUserDailyQuotaRepository,
	subRepo repositories.IUserSubscriptionRepository,
	planRepo repositories.ISubscriptionPlanRepository,
) uc_interfaces.IUserQuotaUseCase {
	return &UserQuotaUseCase{
		quotaRepo: quotaRepo,
		subRepo:   subRepo,
		planRepo:  planRepo,
	}
}

func (uc *UserQuotaUseCase) getOrCreateUserSubscription(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	sub, err := uc.subRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		defaultPlan, err := uc.planRepo.GetDefaultPlan(ctx)
		if err != nil {
			return nil, err
		}
		if defaultPlan == nil {
			return nil, errorcode.NewNotFound("Không tìm thấy cấu hình gói hội viên mặc định")
		}

		sub = &entities.UserSubscription{
			UserID:             userID,
			SubscriptionPlanID: defaultPlan.ID,
			IsActive:           true,
			SubscriptionPlan:   defaultPlan,
		}
		err = uc.subRepo.Create(ctx, sub)
		if err != nil {
			return nil, err
		}
	}
	return sub, nil
}

func (uc *UserQuotaUseCase) getOrCreateUserDailyQuota(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error) {
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

// checkAndResetDailyQuota fetches the user's daily quota and lazily performs reset if a new day has arrived
func (uc *UserQuotaUseCase) checkAndResetDailyQuota(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error) {
	quota, err := uc.getOrCreateUserDailyQuota(ctx, userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	lastReset := quota.LastResetDate
	if now.Year() > lastReset.Year() ||
		(now.Year() == lastReset.Year() && now.Month() > lastReset.Month()) ||
		(now.Year() == lastReset.Year() && now.Month() == lastReset.Month() && now.Day() > lastReset.Day()) {

		quota.OutfitRecommendCount = 0
		quota.AiUsageCount = 0
		quota.LastResetDate = now

		err = uc.quotaRepo.Update(ctx, quota)
		if err != nil {
			return nil, err
		}
	}

	return quota, nil
}

// GetAndResetDailyQuota evaluates daily resets lazily and retrieves the fresh resource counters
func (uc *UserQuotaUseCase) GetAndResetDailyQuota(ctx context.Context, userID uuid.UUID) (*contract.UserSubscriptionDTO, error) {
	sub, err := uc.getOrCreateUserSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}

	quota, err := uc.checkAndResetDailyQuota(ctx, userID)
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
			return nil, errorcode.NewNotFound("Không tìm thấy thông tin gói hội viên của người dùng")
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

// UpdateOutfitQuota alters daily recommended outfit generations count
func (uc *UserQuotaUseCase) UpdateOutfitQuota(ctx context.Context, userID uuid.UUID, count int) error {
	sub, err := uc.getOrCreateUserSubscription(ctx, userID)
	if err != nil {
		return err
	}

	quota, err := uc.checkAndResetDailyQuota(ctx, userID)
	if err != nil {
		return err
	}

	plan := sub.SubscriptionPlan
	if plan == nil {
		p, err := uc.planRepo.GetByID(ctx, sub.SubscriptionPlanID)
		if err != nil {
			return err
		}
		if p == nil {
			return errorcode.NewNotFound("Không tìm thấy thông tin gói hội viên của người dùng")
		}
		plan = p
	}

	newCount := quota.OutfitRecommendCount + count
	if newCount > plan.AiOutfitDailyQuota {
		return errorcode.NewBadRequest("Hạn mức sử dụng AI tạo trang phục trong ngày đã hết")
	}

	quota.OutfitRecommendCount = newCount
	return uc.quotaRepo.Update(ctx, quota)
}

// UpdateAiChatQuota alters daily AI chatbot usage count
func (uc *UserQuotaUseCase) UpdateAiChatQuota(ctx context.Context, userID uuid.UUID, count int) error {
	sub, err := uc.getOrCreateUserSubscription(ctx, userID)
	if err != nil {
		return err
	}

	quota, err := uc.checkAndResetDailyQuota(ctx, userID)
	if err != nil {
		return err
	}

	plan := sub.SubscriptionPlan
	if plan == nil {
		p, err := uc.planRepo.GetByID(ctx, sub.SubscriptionPlanID)
		if err != nil {
			return err
		}
		if p == nil {
			return errorcode.NewNotFound("Không tìm thấy thông tin gói hội viên của người dùng")
		}
		plan = p
	}

	newCount := quota.AiUsageCount + count
	if newCount > plan.AiChatDailyQuota {
		return errorcode.NewBadRequest("Hạn mức sử dụng AI Chatbot trong ngày đã hết")
	}

	quota.AiUsageCount = newCount
	return uc.quotaRepo.Update(ctx, quota)
}
