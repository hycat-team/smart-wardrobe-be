package entities

import (
	"strings"
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/identity/roleslug"
	"smart-wardrobe-be/internal/shared/domain/constants/identity/userstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/shared/gender"

	shared_entities "smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type User struct {
	shared_entities.AuditableEntity
	shared_entities.SoftDeleteEntity
	Username       string                `gorm:"type:varchar(255);not null;index:idx_users_username_lower,unique,expression:lower(username)"`
	Email          string                `gorm:"type:varchar(255);not null;index:idx_users_email_lower,unique,expression:lower(email)"`
	PasswordHash   string                `gorm:"type:varchar(255);not null"`
	FirstName      *string               `gorm:"type:varchar(255)"`
	LastName       *string               `gorm:"type:varchar(255)"`
	DateOfBirth    *time.Time            `gorm:"type:date"`
	Address        *string               `gorm:"type:varchar(255)"`
	Gender         *gender.Gender        `gorm:"type:int"`
	RoleSlug       roleslug.RoleSlug     `gorm:"type:varchar(50);not null"`
	BodyProfile    *shared_entities.BodyProfile          `gorm:"type:jsonb"`
	Status         userstatus.UserStatus `gorm:"type:smallint;not null;default:0"`
	StyleProfile   *UserStyleProfile     `gorm:"foreignKey:UserID"`
	AvatarUrl      *string               `gorm:"type:varchar(500)"`
	AvatarPublicID *string               `gorm:"type:varchar(255)"`
}

func (u *User) GetEffectiveBodyProfile() *shared_entities.BodyProfile {
	if u == nil || u.BodyProfile == nil {
		return nil
	}

	effective := *u.BodyProfile
	if effective.VerifiedByUser && strings.TrimSpace(effective.BodyShape) != "" {
		return &effective
	}

	if effective.InferredByAI != nil && strings.TrimSpace(effective.InferredByAI.BodyShape) != "" {
		effective.BodyShape = strings.TrimSpace(effective.InferredByAI.BodyShape)
	}

	return &effective
}

type UserStyleProfile struct {
	UserID          uuid.UUID                          `gorm:"type:uuid;primaryKey"`
	TasteEmbedding  shared_entities.Vector             `gorm:"type:vector(768)"`
	PreferredColors *shared_entities.PreferredColors   `gorm:"type:jsonb"`
	UpdatedAt       time.Time                          `gorm:"type:timestamp with time zone;not null;default:now()"`
	User            *User                              `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

type RefreshToken struct {
	shared_entities.AuditableEntity
	UserID    uuid.UUID `gorm:"type:uuid;not null"`
	User      *User     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Token     string    `gorm:"type:varchar(500);uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"type:timestamp with time zone;not null"`
	IsRevoked bool      `gorm:"type:boolean;not null;default:false"`
}
