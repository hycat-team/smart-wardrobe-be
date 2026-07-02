package dto

import (
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandstatus"

	"github.com/google/uuid"
)

type CreateBrandReq struct {
	Slug         string  `json:"slug" binding:"required,max=100" label:"slug thương hiệu"`
	Name         string  `json:"name" binding:"required,max=255" label:"tên thương hiệu"`
	Description  *string `json:"description" binding:"omitempty" label:"mô tả"`
	LogoURL      *string `json:"logoUrl" binding:"omitempty,url" label:"logo thương hiệu"`
	LogoPublicID *string `json:"logoPublicId" binding:"omitempty,max=255" label:"mã ảnh logo thương hiệu"`
}

type UpdateBrandLogoReq struct {
	LogoURL      string `json:"logoUrl" binding:"required,url" label:"logo thương hiệu"`
	LogoPublicID string `json:"logoPublicId" binding:"required,max=255" label:"mã ảnh logo thương hiệu"`
}

type UpdateBrandImagesReq struct {
	LogoURL            *string `json:"logoUrl" binding:"omitempty,url" label:"logo thương hiệu"`
	LogoPublicID       *string `json:"logoPublicId" binding:"omitempty,max=255" label:"mã ảnh logo thương hiệu"`
	BackgroundURL      *string `json:"backgroundUrl" binding:"omitempty,url" label:"ảnh nền thương hiệu"`
	BackgroundPublicID *string `json:"backgroundPublicId" binding:"omitempty,max=255" label:"mã ảnh nền thương hiệu"`
}

type UpdateBrandStatusReq struct {
	Status brandstatus.BrandStatus `json:"status" binding:"required" label:"trạng thái thương hiệu"`
}

type BrandRes struct {
	ID                 uuid.UUID               `json:"id"`
	Slug               string                  `json:"slug"`
	Name               string                  `json:"name"`
	Description        *string                 `json:"description"`
	LogoURL            *string                 `json:"logoUrl"`
	LogoPublicID       *string                 `json:"logoPublicId"`
	BackgroundURL      *string                 `json:"backgroundUrl"`
	BackgroundPublicID *string                 `json:"backgroundPublicId"`
	Status             brandstatus.BrandStatus `json:"status"`
	TotalCustomer      *int                    `json:"totalCustomer,omitempty"`
	CreatedByUserID    uuid.UUID               `json:"createdByUserId"`
	ApprovedByUserID   *uuid.UUID              `json:"approvedByUserId"`
	ApprovedAt         *time.Time              `json:"approvedAt"`
	CreatedAt          time.Time               `json:"createdAt"`
	UpdatedAt          time.Time               `json:"updatedAt"`
}

type PortalBrandRes struct {
	BrandRes
	MemberID     uuid.UUID                           `json:"memberId"`
	MemberRole   brandmemberrole.BrandMemberRole     `json:"memberRole"`
	MemberStatus brandmemberstatus.BrandMemberStatus `json:"memberStatus"`
}

type AddBrandMembersReq struct {
	Members []AddBrandMemberItemReq `json:"members" binding:"required,min=1,max=50,dive" label:"danh sách thành viên"`
}

type AddBrandMemberItemReq struct {
	EmailOrUsername string                          `json:"emailOrUsername" binding:"required,max=255" label:"email hoặc tên đăng nhập"`
	Role            brandmemberrole.BrandMemberRole `json:"role" binding:"required" label:"vai trò thành viên"`
}

type AddBrandMemberItemResult struct {
	EmailOrUsername string          `json:"emailOrUsername"`
	Member          *BrandMemberRes `json:"member,omitempty"`
	ReasonCode      string          `json:"reasonCode,omitempty"`
	Message         string          `json:"message,omitempty"`
}

type AddBrandMembersRes struct {
	Created []AddBrandMemberItemResult `json:"created"`
	Updated []AddBrandMemberItemResult `json:"updated"`
	Failed  []AddBrandMemberItemResult `json:"failed"`
}

type BrandMemberRes struct {
	ID        uuid.UUID                           `json:"id"`
	BrandID   uuid.UUID                           `json:"brandId"`
	UserID    uuid.UUID                           `json:"userId"`
	Role      brandmemberrole.BrandMemberRole     `json:"role"`
	Status    brandmemberstatus.BrandMemberStatus `json:"status"`
	CreatedAt time.Time                           `json:"createdAt"`
	UpdatedAt time.Time                           `json:"updatedAt"`
}
