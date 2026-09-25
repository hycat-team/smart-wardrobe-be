package dto

import (
	"time"

	shared_dto "smart-wardrobe-be/shared/application/dto"
)

type FollowReq struct {
	IsFollowing *bool `json:"isFollowing" binding:"required" label:"trạng thái theo dõi"`
}

type FollowUserRes struct {
	User       *CommunityUserRes `json:"user"`
	Relation   string            `json:"relation"`
	FollowedAt time.Time         `json:"followedAt"`
}

type GetFollowsQueryReq struct {
	shared_dto.PaginationQuery
	Type string `form:"type" label:"loại danh sách"`
	Q    string `form:"q" label:"từ khóa"`
}

type FollowUserListRes = shared_dto.PaginationResult[*FollowUserRes]
