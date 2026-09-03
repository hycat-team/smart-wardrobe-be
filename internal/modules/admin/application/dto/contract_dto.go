package dto

import (
	"time"

	"github.com/google/uuid"
)

type AuditLogContractDTO struct {
	ID         uuid.UUID  `json:"id"`
	ActorID    uuid.UUID  `json:"actor_id"`
	Action     string     `json:"action"`
	TargetType string     `json:"target_type"`
	TargetID   *uuid.UUID `json:"target_id"`
	CreatedAt  time.Time  `json:"created_at"`
}
