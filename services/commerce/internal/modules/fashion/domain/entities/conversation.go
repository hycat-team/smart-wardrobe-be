package entities

import (
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/constants/messagesender"

	"github.com/google/uuid"
)

type ConversationalContext struct {
	AuditableEntity
	UserID         uuid.UUID `gorm:"type:uuid;not null"`
	Title          string    `gorm:"type:varchar(255);not null"`
	ContextSummary string    `gorm:"type:text;default:''"`
	IsArchived     bool      `gorm:"type:boolean;not null;default:false"`
}

type Message struct {
	BaseEntity
	ContextID    uuid.UUID                   `gorm:"type:uuid;not null"`
	Context      *ConversationalContext      `gorm:"foreignKey:ContextID;constraint:OnDelete:CASCADE"`
	Sender       messagesender.MessageSender `gorm:"type:varchar(50);not null"`
	Content      string                      `gorm:"type:text;not null"`
	IsSummarized bool                        `gorm:"column:is_summarized;type:boolean;not null;default:false"`
}
