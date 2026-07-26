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
}

func (OutboxRecord) TableName() string {
	return "outbox_records"
}

type OutboxRepository interface {
	Create(ctx context.Context, record *OutboxRecord) error
	ClaimPending(ctx context.Context, limit int) ([]OutboxRecord, error)
	MarkPublished(ctx context.Context, id uuid.UUID) error
	MarkFailed(ctx context.Context, id uuid.UUID, err error) error
	SendNotification(ctx context.Context) error
}
