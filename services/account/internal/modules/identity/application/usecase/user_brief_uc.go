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

// GetByUsernamesOrEmails resolves identifiers to accounts, EXCLUDING
// soft-deleted ones (FR-049).
//
// 024 T136: the comment that used to live here claimed the opposite - that
// soft-deleted accounts are "RETURNED with IsDeleted set". They never were: the
// repository query filters `is_deleted = false`, so they arrived as absent. That
// comment was written for the admin grant flow and had it believe a code path
// existed that did not.
//
// The grant flow now has its own method below, so it can say "this account was
// deleted" instead of "no account matches" - and this one keeps its filtering,
// because commerce (brand members, community follows) shares it and must not see
// deleted accounts.
func (uc *UserBriefUseCase) GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*dto.UserBriefContractDTO, error) {
	return uc.briefsFromUsers(ctx, uc.userRepo.GetByUsernamesOrEmails, identifiers)
}

// GetByUsernamesOrEmailsIncludingDeleted is the admin grant path's lookup.
//
// It returns soft-deleted rows so the caller can report `user_soft_deleted`
// (contracts/admin-grant-api.md, FR-049). Nothing else may use it: the flag is
// information for a decision, not an invitation to act on a deleted account.
func (uc *UserBriefUseCase) GetByUsernamesOrEmailsIncludingDeleted(ctx context.Context, identifiers []string) ([]*dto.UserBriefContractDTO, error) {
	return uc.briefsFromUsers(ctx, uc.userRepo.GetByUsernamesOrEmailsIncludingDeleted, identifiers)
}

// briefsFromUsers maps repository rows to the contract DTO.
//
// Shared by both lookups so the mapping cannot drift: the difference between them
// is which rows the query returns, nothing about how a row becomes a DTO.
func (uc *UserBriefUseCase) briefsFromUsers(
	ctx context.Context,
	fetch func(context.Context, []string) ([]*entities.User, error),
	identifiers []string,
) ([]*dto.UserBriefContractDTO, error) {
	if len(identifiers) == 0 {
		return nil, nil
	}
	users, err := fetch(ctx, identifiers)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}
	result := make([]*dto.UserBriefContractDTO, 0, len(users))
	for _, user := range users {
		if user == nil {
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
