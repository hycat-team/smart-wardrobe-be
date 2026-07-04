package entities

import (
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandcustomerjoinedsource"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandcustomerstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandstatus"

	shared_entities "smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type Brand struct {
	shared_entities.AuditableEntity
	Slug               string                  `gorm:"type:varchar(100);uniqueIndex;not null"`
	Name               string                  `gorm:"type:varchar(255);not null"`
	Description        *string                 `gorm:"type:text"`
	LogoURL            *string                 `gorm:"column:logo_url;type:varchar(500)"`
	LogoPublicID       *string                 `gorm:"column:logo_public_id;type:varchar(255)"`
	BackgroundURL      *string                 `gorm:"column:background_url;type:varchar(500)"`
	BackgroundPublicID *string                 `gorm:"column:background_public_id;type:varchar(255)"`
	Status             brandstatus.BrandStatus `gorm:"type:varchar(50);not null;default:PENDING_REVIEW"`
	CreatedByUserID    uuid.UUID               `gorm:"type:uuid;not null"`
	ApprovedByUserID   *uuid.UUID              `gorm:"type:uuid"`
	ApprovedAt         *time.Time              `gorm:"type:timestamp with time zone"`
	Members            []*BrandMember          `gorm:"foreignKey:BrandID"`
	Customers          []*BrandCustomer        `gorm:"foreignKey:BrandID"`
}

type BrandMember struct {
	shared_entities.AuditableEntity
	BrandID uuid.UUID                           `gorm:"type:uuid;not null;uniqueIndex:uq_brand_members_brand_user"`
	Brand   *Brand                              `gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
	UserID  uuid.UUID                           `gorm:"type:uuid;not null;uniqueIndex:uq_brand_members_brand_user"`
	Role    brandmemberrole.BrandMemberRole     `gorm:"type:varchar(50);not null"`
	Status  brandmemberstatus.BrandMemberStatus `gorm:"type:varchar(50);not null;default:ACTIVE"`
}

type BrandCustomer struct {
	shared_entities.AuditableEntity
	BrandID              uuid.UUID                                           `gorm:"type:uuid;not null"`
	Brand                *Brand                                              `gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
	UserID               *uuid.UUID                                          `gorm:"type:uuid"`
	CustomerName         *string                                             `gorm:"type:varchar(255)"`
	PhoneE164            *string                                             `gorm:"type:varchar(50)"`
	PhoneHash            *string                                             `gorm:"type:varchar(255)"`
	ExternalCustomerCode *string                                             `gorm:"type:varchar(100)"`
	JoinedSource         brandcustomerjoinedsource.BrandCustomerJoinedSource `gorm:"type:varchar(50);not null"`
	Status               brandcustomerstatus.BrandCustomerStatus             `gorm:"type:varchar(50);not null;default:ACTIVE"`
	JoinedAt             time.Time                                           `gorm:"type:timestamp with time zone;not null;default:now()"`
	ClaimedAt            *time.Time                                          `gorm:"type:timestamp with time zone"`
	CreatedByMemberID    *uuid.UUID                                          `gorm:"type:uuid"`
	CreatedByMember      *BrandMember                                        `gorm:"foreignKey:CreatedByMemberID;constraint:OnDelete:SET NULL"`
	LoyaltyAccount       *LoyaltyAccount                                     `gorm:"foreignKey:BrandCustomerID"`
}
