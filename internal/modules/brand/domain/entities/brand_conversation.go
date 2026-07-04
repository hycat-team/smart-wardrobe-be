package entities

import (
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandchat/conversationstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandchat/senderrole"

	shared_entities "smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type BrandConversation struct {
	shared_entities.AuditableEntity
	BrandID         uuid.UUID                             `gorm:"type:uuid;not null;uniqueIndex:uq_brand_conversations_brand_user"`
	Brand           *Brand                                `gorm:"foreignKey:BrandID;constraint:OnDelete:CASCADE"`
	UserID          uuid.UUID                             `gorm:"type:uuid;not null;uniqueIndex:uq_brand_conversations_brand_user"`
	Status          conversationstatus.ConversationStatus `gorm:"type:varchar(50);not null;default:OPEN"`
	LastMessageAt   *time.Time                            `gorm:"type:timestamp with time zone"`
	UserLastReadAt  *time.Time                            `gorm:"type:timestamp with time zone"`
	StaffLastReadAt *time.Time                            `gorm:"type:timestamp with time zone"`
	ClosedAt        *time.Time                            `gorm:"type:timestamp with time zone"`
	ClosedByUserID  *uuid.UUID                            `gorm:"type:uuid"`
}

type BrandConversationMessage struct {
	ID             uuid.UUID             `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ConversationID uuid.UUID             `gorm:"type:uuid;not null"`
	Conversation   *BrandConversation    `gorm:"foreignKey:ConversationID;constraint:OnDelete:CASCADE"`
	SenderUserID   *uuid.UUID            `gorm:"type:uuid"`
	SenderRole     senderrole.SenderRole `gorm:"type:varchar(50);not null"`
	Message        string                `gorm:"type:text;not null"`
	CreatedAt      time.Time             `gorm:"type:timestamp with time zone;not null;default:now()"`
}
