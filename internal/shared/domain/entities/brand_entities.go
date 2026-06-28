package entities

import (
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/benefit/benefitfeaturecode"
	"smart-wardrobe-be/internal/shared/domain/constants/benefit/benefitredemptionstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/benefit/benefitstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/benefit/benefittype"
	"smart-wardrobe-be/internal/shared/domain/constants/benefit/benefitunlocktype"
	"smart-wardrobe-be/internal/shared/domain/constants/brandchat/conversationstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brandchat/senderrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brandcustomerjoinedsource"
	"smart-wardrobe-be/internal/shared/domain/constants/brandcustomerstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/branditem/branditemtype"
	"smart-wardrobe-be/internal/shared/domain/constants/branditem/votetype"
	"smart-wardrobe-be/internal/shared/domain/constants/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brandmemberstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brandstatus"

	"github.com/google/uuid"
)

type Brand struct {
	AuditableEntity
	Slug             string                  `gorm:"type:varchar(100);uniqueIndex;not null"`
	Name             string                  `gorm:"type:varchar(255);not null"`
	Description      *string                 `gorm:"type:text"`
	LogoURL          *string                 `gorm:"column:logo_url;type:varchar(500)"`
	Status           brandstatus.BrandStatus `gorm:"type:varchar(50);not null;default:PENDING_REVIEW"`
	CreatedByUserID  uuid.UUID               `gorm:"type:uuid;not null"`
	CreatedByUser    *User                   `gorm:"foreignKey:CreatedByUserID;constraint:OnDelete:RESTRICT"`
	ApprovedByUserID *uuid.UUID              `gorm:"type:uuid"`
	ApprovedByUser   *User                   `gorm:"foreignKey:ApprovedByUserID;constraint:OnDelete:SET NULL"`
	ApprovedAt       *time.Time              `gorm:"type:timestamp with time zone"`
	Members          []*BrandMember          `gorm:"foreignKey:BrandID"`
	Customers        []*BrandCustomer        `gorm:"foreignKey:BrandID"`
}

type BrandMember struct {
	AuditableEntity
	BrandID uuid.UUID                           `gorm:"type:uuid;not null;uniqueIndex:uq_brand_members_brand_user"`
	Brand   *Brand                              `gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
	UserID  uuid.UUID                           `gorm:"type:uuid;not null;uniqueIndex:uq_brand_members_brand_user"`
	User    *User                               `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Role    brandmemberrole.BrandMemberRole     `gorm:"type:varchar(50);not null"`
	Status  brandmemberstatus.BrandMemberStatus `gorm:"type:varchar(50);not null;default:ACTIVE"`
}

type BrandCustomer struct {
	AuditableEntity
	BrandID              uuid.UUID                                           `gorm:"type:uuid;not null"`
	Brand                *Brand                                              `gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
	UserID               *uuid.UUID                                          `gorm:"type:uuid"`
	User                 *User                                               `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL"`
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
}

type BrandBenefit struct {
	AuditableEntity
	BrandID        uuid.UUID                              `gorm:"type:uuid;not null"`
	Brand          *Brand                                 `gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
	Name           string                                 `gorm:"type:varchar(255);not null"`
	Description    *string                                `gorm:"type:text"`
	BenefitType    benefittype.BenefitType                `gorm:"type:varchar(50);not null"`
	UnlockType     benefitunlocktype.BenefitUnlockType    `gorm:"type:varchar(50);not null"`
	RequiredPoints *int                                   `gorm:"type:int"`
	RequiredTierID *uuid.UUID                             `gorm:"type:uuid"`
	RequiredTier   *LoyaltyTier                           `gorm:"foreignKey:RequiredTierID;constraint:OnDelete:SET NULL"`
	FeatureCode    *benefitfeaturecode.BenefitFeatureCode `gorm:"type:varchar(100)"`
	FeatureConfig  JSONDocument                           `gorm:"type:jsonb"`
	Status         benefitstatus.BenefitStatus            `gorm:"type:varchar(50);not null;default:ACTIVE"`
}

type BenefitRedemption struct {
	AuditableEntity
	BenefitID       uuid.UUID                                       `gorm:"type:uuid;not null"`
	Benefit         *BrandBenefit                                   `gorm:"foreignKey:BenefitID;constraint:OnDelete:CASCADE"`
	BrandID         uuid.UUID                                       `gorm:"type:uuid;not null"`
	Brand           *Brand                                          `gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
	BrandCustomerID uuid.UUID                                       `gorm:"type:uuid;not null"`
	BrandCustomer   *BrandCustomer                                  `gorm:"foreignKey:BrandCustomerID;constraint:OnDelete:CASCADE"`
	UserID          *uuid.UUID                                      `gorm:"type:uuid"`
	User            *User                                           `gorm:"foreignKey:UserID;constraint:OnDelete:SET NULL"`
	PointsSpent     int                                             `gorm:"type:int;not null;default:0"`
	Status          benefitredemptionstatus.BenefitRedemptionStatus `gorm:"type:varchar(50);not null;default:REDEEMED"`
	RedeemedAt      time.Time                                       `gorm:"type:timestamp with time zone;not null;default:now()"`
	UsedAt          *time.Time                                      `gorm:"type:timestamp with time zone"`
	ExpiresAt       *time.Time                                      `gorm:"type:timestamp with time zone"`
}

type BrandConversation struct {
	AuditableEntity
	BrandID       uuid.UUID                             `gorm:"type:uuid;not null;uniqueIndex:uq_brand_conversations_brand_user"`
	Brand         *Brand                                `gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
	UserID        uuid.UUID                             `gorm:"type:uuid;not null;uniqueIndex:uq_brand_conversations_brand_user"`
	User          *User                                 `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Status        conversationstatus.ConversationStatus `gorm:"type:varchar(50);not null;default:OPEN"`
	LastMessageAt *time.Time                            `gorm:"type:timestamp with time zone"`
}

type BrandConversationMessage struct {
	ID             uuid.UUID             `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ConversationID uuid.UUID             `gorm:"type:uuid;not null"`
	Conversation   *BrandConversation    `gorm:"foreignKey:ConversationID;constraint:OnDelete:CASCADE"`
	SenderUserID   *uuid.UUID            `gorm:"type:uuid"`
	SenderUser     *User                 `gorm:"foreignKey:SenderUserID;constraint:OnDelete:SET NULL"`
	SenderRole     senderrole.SenderRole `gorm:"type:varchar(50);not null"`
	Message        string                `gorm:"type:text;not null"`
	CreatedAt      time.Time             `gorm:"type:timestamp with time zone;not null;default:now()"`
}

type BrandItem struct {
	AuditableEntity
	BrandID       uuid.UUID                       `gorm:"type:uuid;not null;uniqueIndex:uq_brand_items_brand_product_code"`
	Brand         *Brand                          `gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
	FashionItemID uuid.UUID                       `gorm:"type:uuid;not null;uniqueIndex:uq_brand_items_fashion_item"`
	FashionItem   *FashionItem                    `gorm:"foreignKey:FashionItemID;constraint:OnDelete:CASCADE"`
	ProductCode   *string                         `gorm:"type:varchar(100);uniqueIndex:uq_brand_items_brand_product_code"`
	Name          string                          `gorm:"type:varchar(255);not null"`
	Description   *string                         `gorm:"type:text"`
	Price         *float64                        `gorm:"type:decimal(12,2)"`
	ItemType      branditemtype.BrandItemType     `gorm:"type:varchar(50);not null"`
	Status        branditemstatus.BrandItemStatus `gorm:"type:varchar(50);not null;default:DRAFT"`
}

type DigitalSampleResponse struct {
	ID           uuid.UUID          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BrandItemID  uuid.UUID          `gorm:"type:uuid;not null"`
	BrandItem    *BrandItem         `gorm:"foreignKey:BrandItemID;constraint:OnDelete:CASCADE"`
	UserID       uuid.UUID          `gorm:"type:uuid;not null"`
	User         *User              `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	OutfitID     *uuid.UUID         `gorm:"type:uuid"`
	Outfit       *Outfit            `gorm:"foreignKey:OutfitID;constraint:OnDelete:SET NULL"`
	VoteType     *votetype.VoteType `gorm:"type:varchar(50)"`
	Rating       *int               `gorm:"type:int"`
	FeedbackText *string            `gorm:"type:text"`
	CreatedAt    time.Time          `gorm:"type:timestamp with time zone;not null;default:now()"`
}
