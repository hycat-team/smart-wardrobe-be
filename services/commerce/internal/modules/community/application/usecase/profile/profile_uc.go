package profile

import (
	"context"

	identitycontract "smart-wardrobe-be/services/account/contract/identity"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	communityerrors "smart-wardrobe-be/services/commerce/internal/modules/community/application/errors"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/mapper"
	usecase_shared "smart-wardrobe-be/services/commerce/internal/modules/community/application/usecase/shared"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"
	"smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
)

type ProfileUseCase struct {
	cfg        *config.Config
	followRepo repositories.IFollowRepository
	statsRepo  repositories.IUserFollowStatsRepository
	postRepo   repositories.IPostRepository
	userBrief  identitycontract.IUserBriefContract
}

func NewProfileUseCase(
	cfg *config.Config,
	followRepo repositories.IFollowRepository,
	statsRepo repositories.IUserFollowStatsRepository,
	postRepo repositories.IPostRepository,
	userBrief identitycontract.IUserBriefContract,
) uc_interfaces.IProfileUseCase {
	return &ProfileUseCase{
		cfg:        cfg,
		followRepo: followRepo,
		statsRepo:  statsRepo,
		postRepo:   postRepo,
		userBrief:  userBrief,
	}
}

func (uc *ProfileUseCase) GetPublicProfile(ctx context.Context, username string, viewerUserID *uuid.UUID) (*dto.PublicProfileRes, error) {
	target, err := usecase_shared.ResolveUserByUsername(ctx, uc.userBrief, username)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, communityerrors.ErrUserNotFound()
	}

	isFollowing := false
	isMe := false
	if viewerUserID != nil {
		isMe = *viewerUserID == target.ID
		if !isMe {
			if exists, ferr := uc.followRepo.Exists(ctx, *viewerUserID, target.ID); ferr == nil {
				isFollowing = exists
			}
		}
	}

	return mapper.MapToPublicProfileRes(target, uc.statsFor(ctx, target.ID), isFollowing, isMe), nil
}

func (uc *ProfileUseCase) statsFor(ctx context.Context, userID uuid.UUID) dto.PublicProfileStatsRes {
	stats := dto.PublicProfileStatsRes{}

	denormalized, err := uc.statsRepo.GetByUserID(ctx, userID)
	if err == nil && denormalized != nil {
		stats.FollowerCount = int64(denormalized.FollowerCount)
		stats.FollowingCount = int64(denormalized.FollowingCount)
	} else {
		if count, cerr := uc.followRepo.CountFollowers(ctx, userID); cerr == nil {
			stats.FollowerCount = count
		}
		if count, cerr := uc.followRepo.CountFollowing(ctx, userID); cerr == nil {
			stats.FollowingCount = count
		}
	}

	if count, cerr := uc.postRepo.CountByAuthor(ctx, userID); cerr == nil {
		stats.PostCount = count
	}
	return stats
}
