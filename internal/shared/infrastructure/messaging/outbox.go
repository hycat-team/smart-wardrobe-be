package messaging

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type OutboxStatus string

const (
	OutboxStatusPending   OutboxStatus = "pending"
	OutboxStatusPublished OutboxStatus = "published"
	OutboxStatusFailed    OutboxStatus = "failed"
)

type OutboxRecord struct {
	ID          uuid.UUID       `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Topic       string          `json:"topic" gorm:"type:varchar(255);not null"`
	Payload     json.RawMessage `json:"payload" gorm:"type:jsonb;not null"`
	Headers     json.RawMessage `json:"headers" gorm:"type:jsonb;default:'{}'"`
	Status      OutboxStatus    `json:"status" gorm:"type:varchar(50);not null;default:'pending'"`
	CreatedAt   time.Time       `json:"created_at" gorm:"type:timestamptz;not null;default:now()"`
	PublishedAt *time.Time      `json:"published_at" gorm:"type:timestamptz"`
	Attempts    int             `json:"attempts" gorm:"type:int;not null;default:0"`
	LastError   *string         `json:"last_error" gorm:"type:text"`
	UpdatedAt   time.Time       `json:"updated_at" gorm:"type:timestamptz;not null;default:now()"`
}

func (OutboxRecord) TableName() string {
	return "outbox_records"
}

type OutboxRepository interface {
	Create(ctx context.Context, record *OutboxRecord) error
	// ClaimToPublish claims up to `limit` outbox records that are ready to be
	// published to RabbitMQ: new pending records plus failed records that have
	// exhausted the retry backoff (attempts < maxAttempts and updated_at older
	// than retryAfter). Uses FOR UPDATE SKIP LOCKED for atomic per-worker claiming.
	ClaimToPublish(ctx context.Context, limit int, maxAttempts int, retryAfter time.Duration) ([]OutboxRecord, error)
	MarkPublished(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, err error) error
	// DeletePublishedBefore deletes published records older than `since` (cleanup).
	DeletePublishedBefore(ctx context.Context, since time.Time) (int64, error)
	SendNotification(ctx context.Context) error
}
