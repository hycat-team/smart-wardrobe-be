package persistence

import (
	"context"

	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"
	shared_repos "smart-wardrobe-be/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AICostPolicyAuditRepository struct {
	*shared_repos.GenericRepository[aiexecutor_entities.AICostPolicyAuditLog, uuid.UUID]
}

func NewAICostPolicyAuditRepository(db *gorm.DB) repositories.IAICostPolicyAuditRepository {
	// No relations: the audit entry is a flat row, so preloading would only add
	// a query per write.
	return &AICostPolicyAuditRepository{
		GenericRepository: shared_repos.NewGenericRepository[aiexecutor_entities.AICostPolicyAuditLog, uuid.UUID](db, nil),
	}
}

// Create resolves the database handle through GetDB so a transaction injected
// into ctx by IUnitOfWork.Execute is reused. Writing on the raw handle instead
// would put the audit row outside the transaction, and a failed outbox publish
// could then no longer roll it back (FR-004c).
func (r *AICostPolicyAuditRepository) Create(ctx context.Context, log *aiexecutor_entities.AICostPolicyAuditLog) error {
	return r.GetDB(ctx).Create(log).Error
}

// 024 T178: `CountByActorAndTime` is removed — see the note on the interface. It had no
// caller.
