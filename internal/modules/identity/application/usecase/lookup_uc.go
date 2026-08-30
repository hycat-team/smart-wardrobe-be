package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/identity/application/dto"
	identityerrors "smart-wardrobe-be/internal/modules/identity/application/errors"
	"smart-wardrobe-be/internal/modules/identity/application/mapper"
	"smart-wardrobe-be/internal/modules/identity/domain/entities"

	"github.com/google/uuid"
)

func (uc *UserUseCase) GetByID(ctx context.Context, userID uuid.UUID) (*dto.UserRes, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsDeleted {
		return nil, identityerrors.ErrUserProfileNotFound()
	}

	sub, err := uc.subscriptionContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		sub = nil
	}

	return mapper.MapToUserRes(user, sub), nil
}

func (uc *UserUseCase) GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*dto.UserRes, error) {
	if len(userIDs) == 0 {
		return nil, nil
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

	subscriptionsByUserID, err := uc.subscriptionContract.GetUserSubscriptionOverviews(ctx, userIDs)
	if err != nil {
		subscriptionsByUserID = nil
	}

	result := make([]*dto.UserRes, 0, len(userIDs))
	for _, userID := range userIDs {
		user := usersByID[userID]
		if user == nil {
			continue
		}
		result = append(result, mapper.MapToUserRes(user, subscriptionsByUserID[user.ID]))
	}

	return result, nil
}

func (uc *UserUseCase) GetByUsername(ctx context.Context, username string) (*dto.UserRes, error) {
	user, err := uc.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsDeleted {
		return nil, identityerrors.ErrUserProfileNotFound()
	}

	sub, err := uc.subscriptionContract.GetUserSubscriptionOverview(ctx, user.ID)
	if err != nil {
		sub = nil
	}

	return mapper.MapToUserRes(user, sub), nil
}

func (uc *UserUseCase) GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*dto.UserRes, error) {
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

	userIDs := make([]uuid.UUID, 0, len(users))
	for _, u := range users {
		if u != nil && !u.IsDeleted {
			userIDs = append(userIDs, u.ID)
		}
	}

	subscriptionsByUserID, err := uc.subscriptionContract.GetUserSubscriptionOverviews(ctx, userIDs)
	if err != nil {
		subscriptionsByUserID = nil
	}

	result := make([]*dto.UserRes, 0, len(users))
	for _, user := range users {
		if user == nil || user.IsDeleted {
			continue
		}
		result = append(result, mapper.MapToUserRes(user, subscriptionsByUserID[user.ID]))
	}

	return result, nil
}
