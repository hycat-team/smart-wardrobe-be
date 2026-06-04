package entities

import (
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/gender"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	"smart-wardrobe-be/internal/shared/domain/constants/userstatus"

	"github.com/google/uuid"
)

type User struct {
	AuditableEntity
	SoftDeleteEntity
	Username       string                `gorm:"type:varchar(255);not null"`
	Email          string                `gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash   string                `gorm:"type:varchar(255);not null"`
	FirstName      *string               `gorm:"type:varchar(255)"`
	LastName       *string               `gorm:"type:varchar(255)"`
	DateOfBirth    *time.Time            `gorm:"type:date"`
	Address        *string               `gorm:"type:varchar(255)"`
	Gender         *gender.Gender        `gorm:"type:int"`
	RoleSlug       roleslug.RoleSlug     `gorm:"type:varchar(50);not null"`
	BodyProfile    *bodyProfile          `gorm:"type:jsonb"`
	Status         userstatus.UserStatus `gorm:"type:smallint;not null;default:0"`
	StyleProfile   *UserStyleProfile     `gorm:"foreignKey:UserID"`
	AvatarUrl      *string               `gorm:"type:varchar(500)"`
	AvatarPublicID *string               `gorm:"type:varchar(255)"`
}

type UserStyleProfile struct {
	UserID          uuid.UUID `gorm:"type:uuid;primaryKey"`
	TasteEmbedding  Vector    `gorm:"type:vector(768)"`
	PreferredColors *string   `gorm:"type:jsonb"`
	UpdatedAt       time.Time `gorm:"type:timestamp with time zone;not null;default:now()"`
	User            *User     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

type RefreshToken struct {
	AuditableEntity
	UserID    uuid.UUID `gorm:"type:uuid;not null"`
	User      *User     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Token     string    `gorm:"type:varchar(500);uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"type:timestamp with time zone;not null"`
	IsRevoked bool      `gorm:"type:boolean;not null;default:false"`
}
