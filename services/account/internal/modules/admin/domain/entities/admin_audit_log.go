package entities

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type JSONDocument map[string]any

func (j JSONDocument) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}
	return json.Marshal(j)
}

func (j *JSONDocument) Scan(value any) error {
	if value == nil {
		*j = JSONDocument{}
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, j)
}

type AdminAuditLog struct {
	ID          uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ActorUserID uuid.UUID    `gorm:"type:uuid;not null;index:idx_admin_audit_logs_actor"`
	Action      string       `gorm:"type:varchar(100);not null"`
	TargetType  string       `gorm:"type:varchar(50);not null;index:idx_admin_audit_logs_target,priority:1"`
	TargetID    *uuid.UUID   `gorm:"type:uuid;index:idx_admin_audit_logs_target,priority:2"`
	Payload     JSONDocument `gorm:"type:jsonb;not null;default:'{}'"`
	RequestIP   *string      `gorm:"type:varchar(45)"`
	CreatedAt   time.Time    `gorm:"type:timestamp with time zone;not null;default:now();index:idx_admin_audit_logs_created_at"`
}

func (AdminAuditLog) TableName() string {
	return "shared.admin_audit_logs"
}
