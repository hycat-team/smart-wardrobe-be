package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/identity/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/identity/application/mapper"
	"smart-wardrobe-be/internal/modules/identity/domain/entities"
	"smart-wardrobe-be/internal/modules/identity/domain/repositories"

	"github.com/google/uuid"
)

type UserBriefUseCase struct {
	userRepo repositories.IUserRepository
}

func NewUserBriefUseCase(
	userRepo repositories.IUserRepository,
) uc_interfaces.IUserBriefUseCase {
	return &UserBriefUseCase{
		userRepo: userRepo,
	}
}

func (uc *UserBriefUseCase) GetByID(ctx context.Context, userID uuid.UUID) (*dto.UserBriefRes, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsDeleted {
		return nil, nil
	}
	return mapper.MapToUserBrief(user), nil
}

func (uc *UserBriefUseCase) GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*dto.UserBriefRes, error) {
	if len(userIDs) == 0 {
		return []*dto.UserBriefRes{}, nil
	}
	users, err := uc.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	usersByID := make(map[uuid.UUID]*entities.User, len(users))
	for _, user := range users {
		if user == nil || user.IsDeleted {
			continue
		}
		usersByID[user.ID] = user
	}
	result := make([]*dto.UserBriefRes, 0, len(userIDs))
	for _, userID := range userIDs {
		user := usersByID[userID]
		if user == nil {
			continue
		}
		result = append(result, mapper.MapToUserBrief(user))
	}
	return result, nil
}

func (uc *UserBriefUseCase) GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*dto.UserBriefRes, error) {
	if len(identifiers) == 0 {
		return nil, nil
	}
	users, err := uc.userRepo.GetByUsernamesOrEmails(ctx, identifiers)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}
	result := make([]*dto.UserBriefRes, 0, len(users))
	for _, user := range users {
		if user == nil || user.IsDeleted {
			continue
		}
		result = append(result, mapper.MapToUserBrief(user))
	}
	return result, nil
}
