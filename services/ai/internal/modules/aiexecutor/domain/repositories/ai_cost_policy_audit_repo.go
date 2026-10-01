package repositories

import (
	"context"

	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
)

// IAICostPolicyAuditRepository writes the AI service's own audit trail for
// administrator edits of the cost policy configuration (024 FR-004b).
//
// It deliberately does NOT reuse shared.admin_audit_logs: that table belongs to
// the admin module of the Account service, and writing to it from AI would be
// the constitution §I violation this feature exists to remove.
type IAICostPolicyAuditRepository interface {
	// Create must honour the transaction injected into ctx by IUnitOfWork.Execute, so
	// a failed outbox write rolls the audit row back too (FR-004c).
	Create(ctx context.Context, log *aiexecutor_entities.AICostPolicyAuditLog) error

	// 024 T178: `CountByActorAndTime` is GONE from this interface. It counted entries by
	// one administrator in a window, and nothing ever called it: the rate-limit check
	// that would have used it lives elsewhere, and a bare count query nobody issues is
	// an invitation to read the audit trail as if it were being read. Removing the
	// method also removes its implementation and the test fake that had to satisfy it.
}
