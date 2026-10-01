package entities

import (
	"strings"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/authprovider"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/userstatus"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	"smart-wardrobe-be/shared/domain/constants/shared/gender"

	"github.com/google/uuid"
)

type User struct {
	AuditableEntity
	SoftDeleteEntity
	Username       string                    `gorm:"type:varchar(255);not null;index:idx_users_username_lower,unique,expression:lower(username)"`
	Email          string                    `gorm:"type:varchar(255);not null;index:idx_users_email_lower,unique,expression:lower(email)"`
	PasswordHash   *string                   `gorm:"type:varchar(255)"`
	Provider       authprovider.AuthProvider `gorm:"type:varchar(50);not null;default:'local'"`
	ProviderID     *string                   `gorm:"type:varchar(255)"`
	EmailVerified  bool                      `gorm:"type:boolean;not null;default:false"`
	FirstName      *string                   `gorm:"type:varchar(255)"`
	LastName       *string                   `gorm:"type:varchar(255)"`
	DateOfBirth    *time.Time                `gorm:"type:date"`
	Address        *string                   `gorm:"type:varchar(255)"`
	Gender         *gender.Gender            `gorm:"type:int"`
	RoleSlug       roleslug.RoleSlug         `gorm:"type:varchar(50);not null"`
	BodyProfile    *BodyProfile              `gorm:"type:jsonb"`
	Status         userstatus.UserStatus     `gorm:"type:smallint;not null;default:0"`
	StyleProfile   *UserStyleProfile         `gorm:"foreignKey:UserID"`
	AvatarUrl      *string                   `gorm:"type:varchar(500)"`
	AvatarPublicID *string                   `gorm:"type:varchar(255)"`
	// 025 T017 (FR-050): true only when the account was created through the
	// administrator endpoint.
	//
	// It has to be a stored column and not something derived: the two self-serve
	// doors and the administrator door build structurally identical rows, so
	// nothing else in the record says which one created it. The campaign
	// catch-up sweep reads the database and never sees the creation event, so
	// without this field it could not tell an administrator's test account from
	// a customer's - and would spend real customers' promotional quota on it.
	//
	// NOT derivable from EmailVerified either: an administrator may create an
	// already-verified account, and every self-serve account is verified. One
	// field is not evidence about the other.
	//
	// Set at exactly one write site (admin_uc.go, FR-091); the two self-serve
	// doors leave it at the false default.
	CreatedByAdmin bool `gorm:"type:boolean;not null;default:false"`
}

func (u *User) GetEffectiveBodyProfile() *BodyProfile {
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
	UserID          uuid.UUID        `gorm:"type:uuid;primaryKey"`
	TasteEmbedding  Vector           `gorm:"type:vector(768)"`
	PreferredColors *PreferredColors `gorm:"type:jsonb"`
	UpdatedAt       time.Time        `gorm:"type:timestamp with time zone;not null;default:now()"`
	User            *User            `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

type RefreshToken struct {
	AuditableEntity
	UserID       uuid.UUID  `gorm:"type:uuid;not null"`
	User         *User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Token        string     `gorm:"type:varchar(500);uniqueIndex;not null"`
	ExpiresAt    time.Time  `gorm:"type:timestamp with time zone;not null"`
	IsRevoked    bool       `gorm:"type:boolean;not null;default:false"`
	DeviceName   *string    `gorm:"type:varchar(255)"`
	DeviceIP     *string    `gorm:"type:varchar(45)"`
	UserAgent    *string    `gorm:"type:varchar(512)"`
	LastActiveAt *time.Time `gorm:"type:timestamp with time zone"`
}
