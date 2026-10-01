package entities

import (
	"time"

	"github.com/google/uuid"
)

// AICostPolicyAuditLog records one administrator edit of the AI cost policy
// configuration (024 FR-004b). It is the AI service's OWN table — writing to
// shared.admin_audit_logs directly would be the same cross-service table
// ownership violation that feature 024 removes from ai_policy_grants
// (constitution §I). To surface the entry on Account's central audit screen
// the same transaction publishes an admin.audit.log outbox record instead
// (FR-004c).
type AICostPolicyAuditLog struct {
	AuditableEntity
	ActorUserID uuid.UUID `gorm:"type:uuid;not null"`
	ChangedAt   time.Time `gorm:"type:timestamptz;not null"`
	RequestIP   string    `gorm:"type:varchar(64)"`
	PolicyCode  string    `gorm:"type:varchar(100);not null"`
	Operation   string    `gorm:"type:varchar(30);not null"`
	FieldName   string    `gorm:"type:varchar(60);not null"`
	// OldValue and NewValue are pointers so that "was absent" stays
	// distinguishable from "was set to the empty string". Collapsing both into
	// "" would make a later investigation unable to tell a deletion from a
	// set-to-empty edit.
	OldValue *string `gorm:"type:varchar(255)"`
	NewValue *string `gorm:"type:varchar(255)"`
	Action   string  `gorm:"type:varchar(60);not null"`
}
