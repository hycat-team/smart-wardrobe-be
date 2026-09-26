package entities

import (
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/mediatype"

	"github.com/google/uuid"
)

type PostMedia struct {
	ID        uuid.UUID           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PostID    uuid.UUID           `gorm:"type:uuid;not null;index"`
	MediaType mediatype.MediaType `gorm:"type:varchar(10);not null"`
	MediaURL  string              `gorm:"type:varchar(500);not null"`
	PublicID  *string             `gorm:"type:varchar(255)"`
	SortOrder int16               `gorm:"type:smallint;not null;default:0"`
}
