package mapper

import (
	"time"

	identity_dto "smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
)

// MapToFollowUserRes maps a follow edge + user brief to the list item shape.
func MapToFollowUserRes(user *identity_dto.UserBriefContractDTO, relation string, followedAt time.Time) *dto.FollowUserRes {
	if user == nil {
		return nil
	}
	return &dto.FollowUserRes{
		User:       mapToCommunityUser(user),
		Relation:   relation,
		FollowedAt: followedAt,
	}
}

// MapToPublicProfileRes builds the public profile view.
func MapToPublicProfileRes(user *identity_dto.UserBriefContractDTO, stats dto.PublicProfileStatsRes, isFollowing, isMe bool) *dto.PublicProfileRes {
	if user == nil {
		return nil
	}
	return &dto.PublicProfileRes{
		User:        mapToCommunityUser(user),
		Stats:       stats,
		IsFollowing: isFollowing,
		IsMe:        isMe,
	}
}

// MapToCommunityUserRes maps a user brief to the canonical user shape.
func MapToCommunityUserRes(user *identity_dto.UserBriefContractDTO) *dto.CommunityUserRes {
	return mapToCommunityUser(user)
}
