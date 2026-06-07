package dto

import (
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
)

type GetUsersQueryReq struct {
	shared_dto.PaginationQuery
	RoleSlug *string `form:"roleSlug"`
	IsActive *bool   `form:"isActive"`
	Query    *string `form:"q"`
}

type AdminUserListRes = shared_dto.PaginationResult[*UserRes]

