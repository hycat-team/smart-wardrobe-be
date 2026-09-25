package dto

import (
	shared_dto "smart-wardrobe-be/shared/application/dto"
)

type PublicProfileStatsRes struct {
	FollowerCount  int64 `json:"followerCount"`
	FollowingCount int64 `json:"followingCount"`
	PostCount      int64 `json:"postCount"`
}

type PublicProfileRes struct {
	User        *CommunityUserRes     `json:"user"`
	Stats       PublicProfileStatsRes `json:"stats"`
	IsFollowing bool                  `json:"isFollowing"`
	IsMe        bool                  `json:"isMe"`
}

type GetUserPostsQueryReq struct {
	shared_dto.PaginationQuery
}
