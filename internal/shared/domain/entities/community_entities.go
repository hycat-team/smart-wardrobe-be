package entities

import (
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/itemcondition"
	"smart-wardrobe-be/internal/shared/domain/constants/postitemstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/posttype"
	"smart-wardrobe-be/internal/shared/domain/constants/transferstate"

	"github.com/google/uuid"
)

type Post struct {
	AuditableEntity
	SoftDeleteEntity
	UserID         uuid.UUID         `gorm:"type:uuid;not null"`
	User           *User             `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	PostType       posttype.PostType `gorm:"type:varchar(50);not null"`
	Title          *string           `gorm:"type:varchar(255)"`
	Content        string            `gorm:"type:text;not null"`
	ContactInfo    *string           `gorm:"type:varchar(255)"`
	TotalPrice     float64           `gorm:"type:decimal(12,2);default:0.00"`
	LikeCount      int               `gorm:"type:int;default:0"`
	CommentCount   int               `gorm:"type:int;default:0"`
	HotnessDirtyAt *time.Time        `gorm:"type:timestamp with time zone"`
}

type PostScoreSnapshot struct {
	PostID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	Post               *Post     `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
	GlobalHotnessScore float64   `gorm:"type:double precision;default:0.0"`
	LastCalculatedAt   time.Time `gorm:"type:timestamp with time zone;not null;default:now()"`
}

type PostItem struct {
	AuditableEntity
	PostID        uuid.UUID                     `gorm:"type:uuid;not null;uniqueIndex:unique_post_item"`
	Post          *Post                         `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
	ItemID        uuid.UUID                     `gorm:"type:uuid;not null;uniqueIndex:unique_post_item"`
	WardrobeItem  *WardrobeItem                 `gorm:"foreignKey:ItemID;constraint:OnDelete:CASCADE"`
	Price         float64                       `gorm:"type:decimal(12,2);not null;default:0.00"`
	ItemCondition itemcondition.ItemCondition   `gorm:"type:smallint;not null;default:1"`
	Status        postitemstatus.PostItemStatus `gorm:"type:smallint;not null;default:1"` // 0: hidden, 1: available, 2: sold
	BuyerUserID   *uuid.UUID                    `gorm:"type:uuid"`
	BuyerUser     *User                         `gorm:"foreignKey:BuyerUserID;constraint:OnDelete:SET NULL"`
	TransferState transferstate.TransferState   `gorm:"type:smallint;not null;default:0"` // 0 none, 1 pending, 2 accepted, 3 declined
	SoldAt        *time.Time                    `gorm:"type:timestamp with time zone"`
}

type PostMedia struct {
	AuditableEntity
	PostID    uuid.UUID `gorm:"type:uuid;not null"`
	Post      *Post     `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
	MediaType string    `gorm:"type:varchar(20);not null"`
	MediaURL  string    `gorm:"type:varchar(500);not null"`
	PublicID  *string   `gorm:"type:varchar(255)"`
	SortOrder int16     `gorm:"type:smallint;not null;default:0"`
}

type Comment struct {
	AuditableEntity
	SoftDeleteEntity
	PostID          uuid.UUID  `gorm:"type:uuid;not null"`
	Post            *Post      `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
	UserID          uuid.UUID  `gorm:"type:uuid;not null"`
	User            *User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	ParentCommentID *uuid.UUID `gorm:"type:uuid"`
	ParentComment   *Comment   `gorm:"foreignKey:ParentCommentID;constraint:OnDelete:CASCADE"`
	Content         string     `gorm:"type:text;not null"`
}

type Like struct {
	BaseEntity
	UserID    uuid.UUID  `gorm:"type:uuid;not null"`
	User      *User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	PostID    *uuid.UUID `gorm:"type:uuid"`
	Post      *Post      `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
	CommentID *uuid.UUID `gorm:"type:uuid"`
	Comment   *Comment   `gorm:"foreignKey:CommentID;constraint:OnDelete:CASCADE"`
}
