package dto

import (
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
)

type AdminGetPostsQueryReq struct {
	shared_dto.PaginationQuery
	PostType  *string `form:"postType"`
	IsDeleted *bool   `form:"isDeleted"`
	Query     *string `form:"q"`
}

type AdminGetPostItemsQueryReq struct {
	shared_dto.PaginationQuery
	Status        *int `form:"status"`
	TransferState *int `form:"transferState"`
}

type AdminPostListRes = shared_dto.PaginationResult[*PostRes]
type AdminPostItemListRes = shared_dto.PaginationResult[*PostItemRes]
