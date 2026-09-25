package dto

import (
	shared_dto "smart-wardrobe-be/shared/application/dto"
)

type SearchQueryReq struct {
	shared_dto.PaginationQuery
	Q        string `form:"q" label:"từ khóa"`
	Type     string `form:"type" label:"loại kết quả"`
	PostType string `form:"postType" label:"loại bài đăng"`
}

type SearchRes struct {
	Users shared_dto.PaginationResult[*CommunityUserRes] `json:"users"`
	Posts shared_dto.PaginationResult[*PostRes]          `json:"posts"`
}
