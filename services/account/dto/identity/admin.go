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
	// UserID filters by exact account id. It takes precedence over Query so an
	// administrator can paste a UUID from another admin screen without also
	// matching a name fragment.
	UserID *string `form:"userId"`
	// PlanID filters by the account's EFFECTIVE membership plan (the plan the
	// admin table prints), not the raw `current_plan_code` column: an expired
	// finite plan already shows its fallback plan in the UI, so filtering the
	// raw column would list accounts under a plan the operator cannot see.
	//
	// The id comes from GET /admin/subscriptions/plans. Identity cannot join
	// the subscription tables (constitution §II), so the contract resolves the
	// matching user ids first.
	PlanID *string `form:"planId"`
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

// AdminUserDetailRes is gone on purpose: the detail endpoint now answers with
// UserRes, whose embedded Subscription is the same projection the list uses.
// The old envelope repeated the subscription as a second, differently-shaped
// field (`any`), so the two copies could disagree and every reader had to pick
// one. See docs/api/identity/admin-user-api.md.

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
