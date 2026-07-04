package entities

import (
	"smart-wardrobe-be/internal/shared/domain/constants/shared/messagesender"

	shared_entities "smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type ConversationalContext struct {
	shared_entities.AuditableEntity
	UserID         uuid.UUID `gorm:"type:uuid;not null"`
	Title          string    `gorm:"type:varchar(255);not null"`
	ContextSummary string    `gorm:"type:text;default:''"`
	IsArchived     bool      `gorm:"type:boolean;not null;default:false"`
}

type Message struct {
	shared_entities.BaseEntity
	ContextID    uuid.UUID                   `gorm:"type:uuid;not null"`
	Context      *ConversationalContext      `gorm:"foreignKey:ContextID;constraint:OnDelete:CASCADE"`
	Sender       messagesender.MessageSender `gorm:"type:varchar(50);not null"`
	Content      string                      `gorm:"type:text;not null"`
	IsSummarized bool                        `gorm:"column:is_summarized;type:boolean;not null;default:false"`
}
