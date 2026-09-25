package entities

import (
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/hiddenreason"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/poststatus"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/posttype"

	"github.com/google/uuid"
)

type Post struct {
	AuditableEntity
	PublicID     string                     `gorm:"type:varchar(32);not null;uniqueIndex"`
	UserID       uuid.UUID                  `gorm:"type:uuid;not null"`
	PostType     posttype.PostType          `gorm:"type:varchar(20);not null"`
	Title        *string                    `gorm:"type:varchar(150)"`
	Content      string                     `gorm:"type:text;not null;default:''"`
	OutfitID     *uuid.UUID                 `gorm:"type:uuid"`
	LikeCount    int                        `gorm:"type:integer;not null;default:0"`
	CommentCount int                        `gorm:"type:integer;not null;default:0"`
	Status       poststatus.PostStatus      `gorm:"type:varchar(20);not null;default:'published'"`
	ModeratedBy  *uuid.UUID                 `gorm:"type:uuid"`
	HiddenReason *hiddenreason.HiddenReason `gorm:"type:varchar(30)"`
	Media        []*PostMedia               `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
}
