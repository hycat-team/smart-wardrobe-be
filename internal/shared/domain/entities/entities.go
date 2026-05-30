package entities

import (
	"smart-wardrobe-be/internal/shared/domain/constants/depositstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/gender"
	"smart-wardrobe-be/internal/shared/domain/constants/userstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/constants/postitemstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	"smart-wardrobe-be/internal/shared/domain/constants/messagesender"
	"smart-wardrobe-be/internal/shared/domain/constants/posttype"
	"smart-wardrobe-be/internal/shared/domain/constants/currency"
	"smart-wardrobe-be/internal/shared/domain/constants/deposittransactiontype"
	"smart-wardrobe-be/internal/shared/domain/constants/walletstatementtype"
	"time"

	"github.com/google/uuid"
)

type SubscriptionPlan struct {
	AuditableEntity
	Slug               string  `gorm:"type:varchar(100);uniqueIndex;not null"`
	Name               string  `gorm:"type:varchar(100);not null"`
	Price              float64 `gorm:"type:numeric(12,2);not null;default:0.00"`
	MaxWardrobeItems   int     `gorm:"type:int;not null"`
	MaxOutfits         int     `gorm:"type:int;not null"`
	AiOutfitDailyQuota int     `gorm:"type:int;not null"`
	AiChatDailyQuota   int     `gorm:"type:int;not null"`
	DurationDays       *int    `gorm:"type:int"`
	IsActive           bool    `gorm:"type:boolean;not null;default:true"`
}

type User struct {
	AuditableEntity
	SoftDeleteEntity
	Username     string                `gorm:"type:varchar(255);not null"`
	Email        string                `gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash string                `gorm:"type:varchar(255);not null"`
	FirstName    *string               `gorm:"type:varchar(255)"`
	LastName     *string               `gorm:"type:varchar(255)"`
	DateOfBirth  *time.Time            `gorm:"type:date"`
	Address      *string               `gorm:"type:varchar(255)"`
	Gender       *gender.Gender        `gorm:"type:int"`
	RoleSlug     roleslug.RoleSlug     `gorm:"type:varchar(50);not null"`
	BodyProfile  *bodyProfile          `gorm:"type:jsonb"`
	Status       userstatus.UserStatus `gorm:"type:smallint;not null;default:0"`
	StyleProfile *UserStyleProfile     `gorm:"foreignKey:UserID"`
	AvatarUrl      *string               `gorm:"type:varchar(500)"`
	AvatarPublicID *string               `gorm:"type:varchar(255)"`
}

type UserSubscription struct {
	UserID             uuid.UUID         `gorm:"type:uuid;primaryKey"`
	SubscriptionPlanID uuid.UUID         `gorm:"type:uuid;not null"`
	SubscriptionPlan   *SubscriptionPlan `gorm:"foreignKey:SubscriptionPlanID;constraint:OnDelete:RESTRICT"`
	ExpiresAt          *time.Time        `gorm:"type:timestamp with time zone"`
	IsActive           bool              `gorm:"type:boolean;not null;default:true"`
	IsAutoRenewEnabled bool              `gorm:"type:boolean;not null;default:false"`
	CreatedAt          time.Time         `gorm:"type:timestamp with time zone;not null;default:now()"`
	UpdatedAt          time.Time         `gorm:"type:timestamp with time zone;not null;default:now()"`
}

type UserDailyQuota struct {
	UserID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	OutfitRecommendCount int       `gorm:"type:int;not null;default:0"`
	AiUsageCount         int       `gorm:"type:int;not null;default:0"`
	LastResetDate        time.Time `gorm:"type:date;not null"`
	CreatedAt            time.Time `gorm:"type:timestamp with time zone;not null;default:now()"`
	UpdatedAt            time.Time `gorm:"type:timestamp with time zone;not null;default:now()"`
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

type ConversationalContext struct {
	AuditableEntity
	UserID         uuid.UUID `gorm:"type:uuid;not null"`
	User           *User     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Title          string    `gorm:"type:varchar(255);not null"`
	ContextSummary string    `gorm:"type:text;default:''"`
	IsArchived     bool      `gorm:"type:boolean;not null;default:false"`
}

type Message struct {
	BaseEntity
	ContextID uuid.UUID              `gorm:"type:uuid;not null"`
	Context   *ConversationalContext `gorm:"foreignKey:ContextID;constraint:OnDelete:CASCADE"`
	Sender    messagesender.MessageSender `gorm:"type:varchar(50);not null"` // 'user' or 'ai'
	Content   string                 `gorm:"type:text;not null"`
}

type Category struct {
	AuditableEntity
	Name string `gorm:"type:varchar(100);uniqueIndex;not null"`
	Slug string `gorm:"type:varchar(100);uniqueIndex;not null"`
}

type WardrobeItem struct {
	AuditableEntity
	SoftDeleteEntity
	UserID     uuid.UUID `gorm:"type:uuid;not null"`
	User       *User     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	CategoryID uuid.UUID `gorm:"type:uuid;not null"`
	Category   *Category `gorm:"foreignKey:CategoryID;constraint:OnDelete:RESTRICT"`
	ImageUrl      string    `gorm:"type:varchar(500);not null"`
	ImagePublicID string    `gorm:"type:varchar(255);not null"`
	Color         *string   `gorm:"type:varchar(50)"`
	Style         *string   `gorm:"type:varchar(100)"`
	Material      *string   `gorm:"type:varchar(100)"`
	Pattern       *string   `gorm:"type:varchar(100)"`
	Fit           *string   `gorm:"type:varchar(50)"`
	Seasonality   *string   `gorm:"type:varchar(100)"`
	Description   *string   `gorm:"type:text"`
	Status        wardrobestatus.WardrobeItemStatus `gorm:"type:smallint;not null;default:0"` // 'in_wardrobe', 'selling', 'sold'
	Embedding     Vector    `gorm:"type:vector(768)"`
}

type Outfit struct {
	AuditableEntity
	SoftDeleteEntity
	UserID      uuid.UUID `gorm:"type:uuid;not null"`
	User        *User     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Name        string    `gorm:"type:varchar(255);not null"`
	Description *string   `gorm:"type:text"`
}

type OutfitItem struct {
	OutfitID   uuid.UUID     `gorm:"type:uuid;primaryKey"`
	Outfit     *Outfit       `gorm:"foreignKey:OutfitID;constraint:OnDelete:CASCADE"`
	ItemID     uuid.UUID     `gorm:"type:uuid;primaryKey"`
	Wardrobe   *WardrobeItem `gorm:"foreignKey:ItemID;constraint:OnDelete:CASCADE"`
	LayerOrder int16         `gorm:"type:smallint;not null;default:1"`
	CreatedAt  time.Time     `gorm:"type:timestamp with time zone;not null;default:now()"`
	UpdatedAt  time.Time     `gorm:"type:timestamp with time zone;not null;default:now()"`
}

type Post struct {
	AuditableEntity
	SoftDeleteEntity
	UserID       uuid.UUID `gorm:"type:uuid;not null"`
	User         *User     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	PostType     posttype.PostType    `gorm:"type:varchar(50);not null"`
	Content      string    `gorm:"type:text;not null"`
	ContactInfo  *string   `gorm:"type:varchar(255)"`
	TotalPrice   float64   `gorm:"type:decimal(12,2);default:0.00"`
	LikeCount    int       `gorm:"type:int;default:0"`
	CommentCount int       `gorm:"type:int;default:0"`
}

type PostScoreSnapshot struct {
	PostID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	Post               *Post     `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
	GlobalHotnessScore float64   `gorm:"type:double precision;default:0.0"`
	LastCalculatedAt   time.Time `gorm:"type:timestamp with time zone;not null;default:now()"`
}

type PostItem struct {
	AuditableEntity
	PostID        uuid.UUID     `gorm:"type:uuid;not null;uniqueIndex:unique_post_item"`
	Post          *Post         `gorm:"foreignKey:PostID;constraint:OnDelete:CASCADE"`
	ItemID        uuid.UUID     `gorm:"type:uuid;not null;uniqueIndex:unique_post_item"`
	WardrobeItem  *WardrobeItem `gorm:"foreignKey:ItemID;constraint:OnDelete:CASCADE"`
	Price         float64       `gorm:"type:decimal(12,2);not null;default:0.00"`
	ItemCondition int16         `gorm:"type:smallint;not null;default:1"`
	Status        postitemstatus.PostItemStatus `gorm:"type:smallint;not null;default:1"` // 0: hidden, 1: available, 2: sold
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

type UserWallet struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	User      *User     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Balance   float64   `gorm:"type:numeric(12,2);not null;default:0.00"`
	Currency  currency.Currency    `gorm:"type:varchar(10);not null;default:'VND'"`
	CreatedAt time.Time `gorm:"type:timestamp with time zone;not null;default:now()"`
	UpdatedAt time.Time `gorm:"type:timestamp with time zone;not null;default:now()"`
}

type DepositTransaction struct {
	AuditableEntity
	UserID             uuid.UUID         `gorm:"type:uuid;not null"`
	User               *User             `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Amount             float64                     `gorm:"type:numeric(12,2);not null"`
	Currency           currency.Currency           `gorm:"type:varchar(10);not null;default:'VND'"`
	Status             depositstatus.DepositStatus `gorm:"type:smallint;not null;default:0"` // 0: PENDING, 1: SUCCESS, 2: FAILED
	TransactionType    deposittransactiontype.DepositTransactionType `gorm:"type:varchar(50);not null"`        // DIRECT_PURCHASE, WALLET_TOPUP
	SubscriptionPlanID *uuid.UUID        `gorm:"type:uuid"`
	SubscriptionPlan   *SubscriptionPlan `gorm:"foreignKey:SubscriptionPlanID;constraint:OnDelete:SET NULL"`
	OrderCode          int64             `gorm:"type:bigint;autoIncrement;uniqueIndex;not null"`
	GatewayReference   *string           `gorm:"type:varchar(255);uniqueIndex"` // PayOS payment transaction reference
	GatewayDetails     *string           `gorm:"type:text"`                     // Raw JSON string payload from gateway
	PaymentUrl         *string           `gorm:"type:varchar(500)"`
}

type WalletStatement struct {
	AuditableEntity
	UserID          uuid.UUID  `gorm:"type:uuid;not null"`
	User            *User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Amount          float64    `gorm:"type:numeric(12,2);not null"`
	TransactionType walletstatementtype.WalletStatementType `gorm:"type:varchar(50);not null"` // TOPUP, SUBSCRIPTION_PURCHASE, SUBSCRIPTION_RENEWAL
	PreviousBalance float64    `gorm:"type:numeric(12,2);not null"`
	NewBalance      float64    `gorm:"type:numeric(12,2);not null"`
	ReferenceID     *uuid.UUID `gorm:"type:uuid"`
	Description     string     `gorm:"type:varchar(255);not null"`
}
