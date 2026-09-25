package usecase

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"

	"github.com/google/uuid"
)

type IFollowUseCase interface {
	ToggleFollow(ctx context.Context, userID uuid.UUID, username string, isFollowing bool) error
	ListFollows(ctx context.Context, username string, viewerUserID *uuid.UUID, query dto.GetFollowsQueryReq) (*dto.FollowUserListRes, error)
}

type IProfileUseCase interface {
	GetPublicProfile(ctx context.Context, username string, viewerUserID *uuid.UUID) (*dto.PublicProfileRes, error)
}
