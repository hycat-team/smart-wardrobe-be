package dto

import (
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/brandcustomerjoinedsource"
	"smart-wardrobe-be/internal/shared/domain/constants/brandcustomerstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brandmemberstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brandstatus"

	"github.com/google/uuid"
)

type CreateBrandReq struct {
	Slug        string  `json:"slug" binding:"required,max=100" label:"slug brand"`
	Name        string  `json:"name" binding:"required,max=255" label:"ten brand"`
	Description *string `json:"description" binding:"omitempty" label:"mo ta"`
	LogoURL     *string `json:"logoUrl" binding:"omitempty,url" label:"logo brand"`
}

type UpdateBrandStatusReq struct {
	Status brandstatus.BrandStatus `json:"status" binding:"required" label:"trang thai brand"`
}

type AddBrandMemberReq struct {
	UserID uuid.UUID                       `json:"userId" binding:"required" label:"ma user"`
	Role   brandmemberrole.BrandMemberRole `json:"role" binding:"required" label:"vai tro brand"`
}

type CreateOfflineBrandCustomerReq struct {
	CustomerName         *string `json:"customerName" binding:"omitempty,max=255"`
	PhoneE164            string  `json:"phoneE164" binding:"required,max=50"`
	ExternalCustomerCode *string `json:"externalCustomerCode" binding:"omitempty,max=100"`
}

type BrandRes struct {
	ID               uuid.UUID               `json:"id"`
	Slug             string                  `json:"slug"`
	Name             string                  `json:"name"`
	Description      *string                 `json:"description"`
	LogoURL          *string                 `json:"logoUrl"`
	Status           brandstatus.BrandStatus `json:"status"`
	CreatedByUserID  uuid.UUID               `json:"createdByUserId"`
	ApprovedByUserID *uuid.UUID              `json:"approvedByUserId"`
	ApprovedAt       *time.Time              `json:"approvedAt"`
	CreatedAt        time.Time               `json:"createdAt"`
	UpdatedAt        time.Time               `json:"updatedAt"`
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

type BrandCustomerRes struct {
	ID                   uuid.UUID                                           `json:"id"`
	BrandID              uuid.UUID                                           `json:"brandId"`
	UserID               *uuid.UUID                                          `json:"userId"`
	CustomerName         *string                                             `json:"customerName"`
	PhoneE164            *string                                             `json:"phoneE164"`
	ExternalCustomerCode *string                                             `json:"externalCustomerCode"`
	JoinedSource         brandcustomerjoinedsource.BrandCustomerJoinedSource `json:"joinedSource"`
	Status               brandcustomerstatus.BrandCustomerStatus             `json:"status"`
	JoinedAt             time.Time                                           `json:"joinedAt"`
	ClaimedAt            *time.Time                                          `json:"claimedAt"`
	CreatedByMemberID    *uuid.UUID                                          `json:"createdByMemberId"`
	CreatedAt            time.Time                                           `json:"createdAt"`
	UpdatedAt            time.Time                                           `json:"updatedAt"`
}
