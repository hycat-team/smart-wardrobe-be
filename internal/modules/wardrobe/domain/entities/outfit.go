package entities

import (
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/wardrobe/outfititemcontext"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobe/outfitstatus"

	shared_entities "smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type Outfit struct {
	shared_entities.AuditableEntity
	shared_entities.SoftDeleteEntity
	UserID        uuid.UUID                 `gorm:"type:uuid;not null"`
	Name          string                    `gorm:"type:varchar(255);not null"`
	Description   *string                   `gorm:"type:text"`
	CoverImageUrl *string                   `gorm:"type:varchar(500)"`
	CoverPublicID *string                   `gorm:"type:varchar(255)"`
	Status        outfitstatus.OutfitStatus `gorm:"type:smallint;not null;default:1"`
	Items         []*OutfitItem             `gorm:"foreignKey:OutfitID;constraint:OnDelete:CASCADE"`
}

type OutfitItem struct {
	OutfitID      uuid.UUID                           `gorm:"type:uuid;primaryKey"`
	Outfit        *Outfit                             `gorm:"foreignKey:OutfitID;constraint:OnDelete:CASCADE"`
	FashionItemID uuid.UUID                           `gorm:"type:uuid;primaryKey"`
	ItemContext   outfititemcontext.OutfitItemContext `gorm:"type:varchar(50);primaryKey;not null"`
	WardrobeItem  *WardrobeItem                       `gorm:"foreignKey:FashionItemID;references:FashionItemID"`
	PositionX     float64                             `gorm:"type:double precision;not null;default:0.0"`
	PositionY     float64                             `gorm:"type:double precision;not null;default:0.0"`
	Scale         float64                             `gorm:"type:double precision;not null;default:1.0"`
	LayerOrder    int16                               `gorm:"type:smallint;not null;default:1"`
	CreatedAt     time.Time                           `gorm:"type:timestamp with time zone;not null;default:now()"`
	UpdatedAt     time.Time                           `gorm:"type:timestamp with time zone;not null;default:now()"`
}
