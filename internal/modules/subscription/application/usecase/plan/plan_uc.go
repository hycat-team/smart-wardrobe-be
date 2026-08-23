package plan

import (
	"context"

	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/subscription/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/subscription/application/mapper"
	"smart-wardrobe-be/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SubscriptionPlanUseCase struct {
	planRepo repositories.ISubscriptionPlanRepository
}

func NewSubscriptionPlanUseCase(
	planRepo repositories.ISubscriptionPlanRepository,
) uc_interfaces.ISubscriptionPlanUseCase {
	return &SubscriptionPlanUseCase{
		planRepo: planRepo,
	}
}

// GetPlans retrieves all active subscription plans for public consumption
func (uc *SubscriptionPlanUseCase) GetPlans(ctx context.Context) ([]*dto.SubscriptionPlanBriefRes, error) {
	plans, err := uc.planRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	activePlans := make([]*entities.SubscriptionPlan, 0, len(plans))
	for _, p := range plans {
		if p != nil && p.IsActive {
			activePlans = append(activePlans, p)
		}
	}

	return mapper.MapToSubscriptionPlanBriefList(activePlans), nil
}

// ListAdminPlans retrieves all subscription plans (including inactive) for admin
func (uc *SubscriptionPlanUseCase) ListAdminPlans(ctx context.Context) ([]*dto.SubscriptionPlanDTO, error) {
	plans, err := uc.planRepo.ListAdmin(ctx)
	if err != nil {
		return nil, err
	}
	return mapper.MapToSubscriptionPlanDTOList(plans), nil
}

// GetDefaultSubscriptionPlanID retrieves default free tier ID
func (uc *SubscriptionPlanUseCase) GetDefaultSubscriptionPlanID(ctx context.Context) (uuid.UUID, error) {
	plan, err := uc.planRepo.GetDefaultPlan(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	if plan == nil {
		return uuid.Nil, subscriptionerrors.ErrDefaultPlanConfigNotFound()
	}
	return plan.ID, nil
}

// IsPremiumPlan checks if plan corresponds to a premium price tier
func (uc *SubscriptionPlanUseCase) IsPremiumPlan(ctx context.Context, planID uuid.UUID) (bool, error) {
	plan, err := uc.planRepo.GetByID(ctx, planID)
	if err != nil {
		return false, err
	}
	if plan == nil || !plan.IsActive {
		return false, nil
	}
	return plan.Price.GreaterThan(sharedmoney.Zero), nil
}

// CreatePlan creates a new subscription plan, verifying that slug is unique
func (uc *SubscriptionPlanUseCase) CreatePlan(ctx context.Context, req *dto.CreateSubscriptionPlanReq) (*dto.SubscriptionPlanDTO, error) {
	existing, err := uc.planRepo.GetBySlugAnyStatus(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, subscriptionerrors.ErrPlanSlugAlreadyExists()
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	plan := &entities.SubscriptionPlan{
		Slug:               req.Slug,
		Name:               req.Name,
		Price:              decimal.NewFromFloat(req.Price),
		MaxWardrobeItems:   req.MaxWardrobeItems,
		MaxOutfits:         req.MaxOutfits,
		AiOutfitDailyQuota: req.AiOutfitDailyQuota,
		AiChatDailyQuota:   req.AiChatDailyQuota,
		DurationDays:       req.DurationDays,
		PlanKind:           req.PlanKind,
		TierRank:           req.TierRank,
		IsActive:           isActive,
		PricingVersion:     1,
	}

	if err := uc.planRepo.Create(ctx, plan); err != nil {
		return nil, err
	}

	return mapper.MapToSubscriptionPlanDTO(plan), nil
}

// UpdatePlan updates an existing subscription plan's properties
func (uc *SubscriptionPlanUseCase) UpdatePlan(ctx context.Context, id uuid.UUID, req *dto.UpdateSubscriptionPlanReq) (*dto.SubscriptionPlanDTO, error) {
	plan, err := uc.planRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, subscriptionerrors.ErrSubscriptionPlanNotFound()
	}

	if req.Name != nil {
		plan.Name = *req.Name
	}
	if req.Price != nil {
		plan.Price = decimal.NewFromFloat(*req.Price)
	}
	if req.MaxWardrobeItems != nil {
		plan.MaxWardrobeItems = *req.MaxWardrobeItems
	}
	if req.MaxOutfits != nil {
		plan.MaxOutfits = *req.MaxOutfits
	}
	if req.AiOutfitDailyQuota != nil {
		plan.AiOutfitDailyQuota = *req.AiOutfitDailyQuota
	}
	if req.AiChatDailyQuota != nil {
		plan.AiChatDailyQuota = *req.AiChatDailyQuota
	}
	if req.DurationDays != nil {
		plan.DurationDays = req.DurationDays
	}
	if req.TierRank != nil {
		plan.TierRank = *req.TierRank
	}
	if req.IsActive != nil {
		plan.IsActive = *req.IsActive
	}

	if err := uc.planRepo.Update(ctx, plan); err != nil {
		return nil, err
	}

	return mapper.MapToSubscriptionPlanDTO(plan), nil
}

// SetPlanActive toggles the active state of a subscription plan
func (uc *SubscriptionPlanUseCase) SetPlanActive(ctx context.Context, id uuid.UUID, isActive bool) (*dto.SubscriptionPlanDTO, error) {
	plan, err := uc.planRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, subscriptionerrors.ErrSubscriptionPlanNotFound()
	}

	plan.IsActive = isActive
	if err := uc.planRepo.Update(ctx, plan); err != nil {
		return nil, err
	}

	return mapper.MapToSubscriptionPlanDTO(plan), nil
}
