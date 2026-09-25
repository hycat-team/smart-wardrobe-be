package usecase

import (
	"context"

	"smart-wardrobe-be/services/account/dto/identity"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"

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

func (uc *UserBriefUseCase) GetByID(ctx context.Context, userID uuid.UUID) (*dto.UserBriefContractDTO, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsDeleted {
		return nil, nil
	}
	return mapper.MapToUserBrief(user), nil
}

func (uc *UserBriefUseCase) GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*dto.UserBriefContractDTO, error) {
	if len(userIDs) == 0 {
		return []*dto.UserBriefContractDTO{}, nil
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
	result := make([]*dto.UserBriefContractDTO, 0, len(userIDs))
	for _, userID := range userIDs {
		user := usersByID[userID]
		if user == nil {
			continue
		}
		result = append(result, mapper.MapToUserBrief(user))
	}
	return result, nil
}

func (uc *UserBriefUseCase) GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*dto.UserBriefContractDTO, error) {
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
	result := make([]*dto.UserBriefContractDTO, 0, len(users))
	for _, user := range users {
		if user == nil || user.IsDeleted {
			continue
		}
		result = append(result, mapper.MapToUserBrief(user))
	}
	return result, nil
}

// SearchUserBriefs returns up to limit users whose username or name matches the
// keyword (case-insensitive, partial). Used by the community search feature.
func (uc *UserBriefUseCase) SearchUserBriefs(ctx context.Context, keyword string, limit int) ([]*dto.UserBriefContractDTO, error) {
	users, err := uc.userRepo.SearchByKeyword(ctx, keyword, limit)
	if err != nil {
		return nil, err
	}
	result := make([]*dto.UserBriefContractDTO, 0, len(users))
	for _, user := range users {
		if user == nil || user.IsDeleted {
			continue
		}
		result = append(result, mapper.MapToUserBrief(user))
	}
	return result, nil
}
