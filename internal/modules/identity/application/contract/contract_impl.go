package contract

import (
	"context"
	"errors"
	"smart-wardrobe-be/internal/modules/identity/contract"
	"smart-wardrobe-be/internal/modules/identity/domain/repositories"

	"github.com/google/uuid"
)

type IdentityModuleContractImpl struct {
	userRepo repositories.IUserRepository
}

func NewIdentityModuleContractImpl(repo repositories.IUserRepository) contract.IIdentityModuleContract {
	return &IdentityModuleContractImpl{
		userRepo: repo,
	}
}

func (impl *IdentityModuleContractImpl) GetUserByID(ctx context.Context, id uuid.UUID) (*contract.PublicUserDTO, error) {
	user, err := impl.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsDeleted {
		return nil, errors.New("user not found or deleted")
	}

	return &contract.PublicUserDTO{
		ID:                   user.ID,
		Username:             user.Username,
		Email:                user.Email,
		RoleSlug:             user.RoleSlug,
		SubscriptionPlanID:   user.SubscriptionPlanID,
		OutfitRecommendCount: user.OutfitRecommendCount,
		AiUsageCount:         user.AiUsageCount,
		LastResetDate:        user.LastResetDate,
	}, nil
}

func (impl *IdentityModuleContractImpl) UpdateOutfitQuota(ctx context.Context, userID uuid.UUID, count int, resetDate bool) error {
	user, err := impl.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	return impl.userRepo.UpdateOutfitQuota(ctx, user, count, resetDate)
}

func (impl *IdentityModuleContractImpl) UpdateAiChatQuota(ctx context.Context, userID uuid.UUID, count int, resetDate bool) error {
	user, err := impl.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	return impl.userRepo.UpdateAiChatQuota(ctx, user, count, resetDate)
}

func (impl *IdentityModuleContractImpl) ResetDailyQuotas(ctx context.Context, userID uuid.UUID) error {
	user, err := impl.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}
	return impl.userRepo.ResetDailyQuotas(ctx, user)
}
