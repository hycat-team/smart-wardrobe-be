package repositories

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// JSONField is a jsonb column value. Value() returns string (not []byte)
// because the runtime uses pgx simple_protocol for the Supavisor pooler
// (FR-046): a []byte driver value is encoded as bytea and rejected by a jsonb
// column with SQLSTATE 22P02 / 42601.
type JSONField json.RawMessage

func (j JSONField) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

func (j *JSONField) Scan(value any) error {
	if value == nil {
		*j = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*j = append((*j)[:0], v...)
	case string:
		*j = append((*j)[:0], v...)
	default:
		return fmt.Errorf("unsupported JSONField value %T", value)
	}
	return nil
}

// Outbox transactional publishing port. Owned by the shared domain layer so
// application use cases and module contracts can write outbox records without
// importing infrastructure/messaging (constitution §IV, feature 017).
// The GORM implementation lives in shared/infrastructure/messaging.

type OutboxStatus string

const (
	OutboxStatusPending    OutboxStatus = "pending"
	OutboxStatusProcessing OutboxStatus = "processing"
	OutboxStatusPublished  OutboxStatus = "published"
	OutboxStatusFailed     OutboxStatus = "failed"
)

type OutboxRecord struct {
	ID          uuid.UUID    `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Topic       string       `json:"topic" gorm:"type:varchar(255);not null"`
	Payload     JSONField    `json:"payload" gorm:"type:jsonb;not null"`
	Headers     JSONField    `json:"headers" gorm:"type:jsonb;default:'{}'"`
	Status      OutboxStatus `json:"status" gorm:"type:varchar(50);not null;default:'pending'"`
	CreatedAt   time.Time    `json:"created_at" gorm:"type:timestamptz;not null;default:now()"`
	PublishedAt *time.Time   `json:"published_at" gorm:"type:timestamptz"`
	Attempts    int          `json:"attempts" gorm:"type:int;not null;default:0"`
	LastError   *string      `json:"last_error" gorm:"type:text"`
	UpdatedAt   time.Time    `json:"updated_at" gorm:"type:timestamptz;not null;default:now()"`
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
	// CountPending trả về số record còn chờ publish (gauge outbox_pending).
	CountPending(ctx context.Context) (int64, error)
	SendNotification(ctx context.Context) error
}
