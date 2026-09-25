package entities

import (
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/poststatus"

	"github.com/google/uuid"
)

type PostComment struct {
	AuditableEntity
	PostID          uuid.UUID             `gorm:"type:uuid;not null;index"`
	UserID          uuid.UUID             `gorm:"type:uuid;not null"`
	ParentCommentID *uuid.UUID            `gorm:"type:uuid"`
	RootCommentID   *uuid.UUID            `gorm:"type:uuid"`
	Content         string                `gorm:"type:text;not null"`
	Status          poststatus.PostStatus `gorm:"type:varchar(20);not null;default:'published'"`
	ModeratedBy     *uuid.UUID            `gorm:"type:uuid"`
}
