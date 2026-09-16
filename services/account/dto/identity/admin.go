package dto

import (
	"time"

	shared_dto "smart-wardrobe-be/shared/application/dto"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	"smart-wardrobe-be/shared/domain/constants/shared/gender"

	"github.com/google/uuid"
)

type GetUsersQueryReq struct {
	shared_dto.PaginationQuery
	RoleSlug *string `form:"roleSlug"`
	IsActive *bool   `form:"isActive"`
	Query    *string `form:"q"`
}

type AdminUserListRes = shared_dto.PaginationResult[*UserRes]

type CreateAdminUserReq struct {
	Username    string             `json:"username" binding:"required,username" label:"tên đăng nhập"`
	Email       string             `json:"email" binding:"required,email" label:"email"`
	Password    string             `json:"password" binding:"required,password_complexity" label:"mật khẩu"`
	FirstName   string             `json:"firstName" binding:"required" label:"tên"`
	LastName    *string            `json:"lastName" binding:"omitempty" label:"họ"`
	DateOfBirth *string            `json:"dateOfBirth" binding:"omitempty,datetime=2006-01-02" label:"ngày sinh"`
	Gender      *gender.Gender     `json:"gender" binding:"omitempty,oneof=0 1 2 3" label:"giới tính"`
	RoleSlug    *roleslug.RoleSlug `json:"roleSlug" label:"vai trò"`
}

type UpdateUserRoleReq struct {
	RoleSlug roleslug.RoleSlug `json:"roleSlug" binding:"required" label:"vai trò"`
}

type AdminUserDetailRes struct {
	User         *UserRes `json:"user"`
	Subscription any      `json:"subscription,omitempty"`
}

type UserSessionItemDTO struct {
	ID           uuid.UUID  `json:"id"`
	DeviceName   *string    `json:"deviceName,omitempty"`
	DeviceIP     *string    `json:"deviceIp,omitempty"`
	UserAgent    *string    `json:"userAgent,omitempty"`
	LastActiveAt *time.Time `json:"lastActiveAt,omitempty"`
	ExpiresAt    time.Time  `json:"expiresAt"`
	IsRevoked    bool       `json:"isRevoked"`
	CreatedAt    time.Time  `json:"createdAt"`
}
