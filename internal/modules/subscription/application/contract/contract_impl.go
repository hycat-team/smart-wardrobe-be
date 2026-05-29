package contract

import (
	"context"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"time"

	"github.com/google/uuid"
)

// SubscriptionModuleContractImpl orchestrates external subscription queries and quota updates
type SubscriptionModuleContractImpl struct {
	planRepo  repositories.ISubscriptionPlanRepository
	subRepo   repositories.IUserSubscriptionRepository
	quotaRepo repositories.IUserDailyQuotaRepository
}

// NewSubscriptionModuleContractImpl initializes a new contract implementation instance
func NewSubscriptionModuleContractImpl(
	planRepo repositories.ISubscriptionPlanRepository,
	subRepo repositories.IUserSubscriptionRepository,
	quotaRepo repositories.IUserDailyQuotaRepository,
) contract.ISubscriptionModuleContract {
	return &SubscriptionModuleContractImpl{
		planRepo:  planRepo,
		subRepo:   subRepo,
		quotaRepo: quotaRepo,
	}
}

// GetDefaultSubscriptionPlanID retrieves default free tier ID
func (impl *SubscriptionModuleContractImpl) GetDefaultSubscriptionPlanID(ctx context.Context) (uuid.UUID, error) {
	plan, err := impl.planRepo.GetDefaultPlan(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	if plan == nil {
		return uuid.Nil, errorcode.NewNotFound("Không tìm thấy gói hội viên mặc định")
	}
	return plan.ID, nil
}

// IsPremiumPlan checks if plan corresponds to a premium price tier
func (impl *SubscriptionModuleContractImpl) IsPremiumPlan(ctx context.Context, planID uuid.UUID) (bool, error) {
	plan, err := impl.planRepo.GetByID(ctx, planID)
	if err != nil {
		return false, err
	}
	if plan == nil || !plan.IsActive {
		return false, nil
	}
	return plan.Price > 0, nil
}

// InitializeUserSubscription sets up default free plan subscription and daily quota records
func (impl *SubscriptionModuleContractImpl) InitializeUserSubscription(ctx context.Context, userID uuid.UUID) error {
	defaultPlanID, err := impl.GetDefaultSubscriptionPlanID(ctx)
	if err != nil {
		return err
	}

	newSub := &entities.UserSubscription{
		UserID:             userID,
		SubscriptionPlanID: defaultPlanID,
		IsActive:           true,
	}
	err = impl.subRepo.Create(ctx, newSub)
	if err != nil {
		return err
	}

	newQuota := &entities.UserDailyQuota{
		UserID:        userID,
		LastResetDate: time.Now(),
	}
	return impl.quotaRepo.Create(ctx, newQuota)
}

// GetUserSubscription loads subscription details and daily quotas aggregated from multiple tables
func (impl *SubscriptionModuleContractImpl) GetUserSubscription(ctx context.Context, userID uuid.UUID) (*contract.UserSubscriptionDTO, error) {
	sub, err := impl.subRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, errorcode.NewNotFound("Không tìm thấy thông tin của gói hội viên")
	}

	quota, err := impl.quotaRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if quota == nil {
		return nil, errorcode.NewNotFound("Không tìm thấy thông tin hạn mức đã sử dụng")
	}

	plan := sub.SubscriptionPlan
	if plan == nil {
		p, err := impl.planRepo.GetByID(ctx, sub.SubscriptionPlanID)
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
func (impl *SubscriptionModuleContractImpl) GetUserSubscriptionOverview(ctx context.Context, userID uuid.UUID) (*contract.UserSubscriptionOverviewDTO, error) {
	sub, err := impl.subRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, errorcode.NewNotFound("Không tìm thấy thông tin của gói hội viên")
	}

	plan := sub.SubscriptionPlan
	if plan == nil {
		p, err := impl.planRepo.GetByID(ctx, sub.SubscriptionPlanID)
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

// GetAndResetDailyQuota evaluates daily resets lazily and retrieves the fresh resource counters
func (impl *SubscriptionModuleContractImpl) GetAndResetDailyQuota(ctx context.Context, userID uuid.UUID) (*contract.UserSubscriptionDTO, error) {
	sub, err := impl.subRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, errorcode.NewNotFound("Không tìm thấy thông tin của gói hội viên")
	}

	quota, err := impl.quotaRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if quota == nil {
		return nil, errorcode.NewNotFound("Không tìm thấy thông tin hạn mức đã sử dụng")
	}

	now := time.Now()
	lastReset := quota.LastResetDate
	if now.Year() > lastReset.Year() ||
		(now.Year() == lastReset.Year() && now.Month() > lastReset.Month()) ||
		(now.Year() == lastReset.Year() && now.Month() == lastReset.Month() && now.Day() > lastReset.Day()) {

		quota.OutfitRecommendCount = 0
		quota.AiUsageCount = 0
		quota.LastResetDate = now

		err = impl.quotaRepo.Update(ctx, quota)
		if err != nil {
			return nil, err
		}
	}

	plan := sub.SubscriptionPlan
	if plan == nil {
		p, err := impl.planRepo.GetByID(ctx, sub.SubscriptionPlanID)
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

// UpdateOutfitQuota alters daily recommended outfit generations count
func (impl *SubscriptionModuleContractImpl) UpdateOutfitQuota(ctx context.Context, userID uuid.UUID, count int, resetDate bool) error {
	quota, err := impl.quotaRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if quota == nil {
		return errorcode.NewNotFound("Không tìm thấy thông tin hạn mức đã sử dụng")
	}

	quota.OutfitRecommendCount = count
	if resetDate {
		quota.LastResetDate = time.Now()
	}
	return impl.quotaRepo.Update(ctx, quota)
}

// UpdateAiChatQuota alters daily AI chatbot usage count
func (impl *SubscriptionModuleContractImpl) UpdateAiChatQuota(ctx context.Context, userID uuid.UUID, count int, resetDate bool) error {
	quota, err := impl.quotaRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if quota == nil {
		return errorcode.NewNotFound("Không tìm thấy thông tin hạn mức đã sử dụng")
	}

	quota.AiUsageCount = count
	if resetDate {
		quota.LastResetDate = time.Now()
	}
	return impl.quotaRepo.Update(ctx, quota)
}

// ResetDailyQuotas resets daily usage parameters
func (impl *SubscriptionModuleContractImpl) ResetDailyQuotas(ctx context.Context, userID uuid.UUID) error {
	quota, err := impl.quotaRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if quota == nil {
		return errorcode.NewNotFound("Không tìm thấy thông tin hạn mức đã sử dụng")
	}

	quota.OutfitRecommendCount = 0
	quota.AiUsageCount = 0
	quota.LastResetDate = time.Now()
	return impl.quotaRepo.Update(ctx, quota)
}
