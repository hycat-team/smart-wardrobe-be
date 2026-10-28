package entities

import (
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/branditem/branditemtype"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/branditem/votetype"

	"github.com/google/uuid"
)

type BrandItem struct {
	AuditableEntity
	BrandID       uuid.UUID                       `gorm:"type:uuid;not null;uniqueIndex:uq_brand_items_brand_product_code"`
	Brand         *Brand                          `gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
	FashionItemID *uuid.UUID                      `gorm:"type:uuid;uniqueIndex:uq_brand_items_fashion_item"`
	ProductCode   *string                         `gorm:"type:varchar(100);uniqueIndex:uq_brand_items_brand_product_code"`
	Name          string                          `gorm:"type:varchar(255);not null"`
	Description   *string                         `gorm:"type:text"`
	Price         *float64                        `gorm:"type:decimal(12,2)"`
	ItemType      branditemtype.BrandItemType     `gorm:"type:varchar(50);not null"`
	Status        branditemstatus.BrandItemStatus `gorm:"type:varchar(50);not null;default:draft"`
	TaskID        *uuid.UUID                      `gorm:"column:task_id;type:uuid"`
}

type DigitalSampleResponse struct {
	ID           uuid.UUID          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	BrandItemID  uuid.UUID          `gorm:"type:uuid;not null"`
	BrandItem    *BrandItem         `gorm:"foreignKey:BrandItemID;constraint:OnDelete:CASCADE"`
	UserID       uuid.UUID          `gorm:"type:uuid;not null"`
	OutfitID     *uuid.UUID         `gorm:"type:uuid"`
	VoteType     *votetype.VoteType `gorm:"type:varchar(50)"`
	Rating       *int               `gorm:"type:int"`
	FeedbackText *string            `gorm:"type:text"`
	CreatedAt    time.Time          `gorm:"type:timestamp with time zone;not null;default:now()"`
}
