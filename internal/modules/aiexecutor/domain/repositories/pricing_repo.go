package repositories

import (
	"context"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type IAIModelConfigRepository interface {
	shared_repos.IGenericRepository[entities.AIModelConfig, uuid.UUID]
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.AIModelConfig, error)
	GetActiveByProviderAndModel(ctx context.Context, provider string, model string) (*entities.AIModelConfig, error)
}

type IPricingConfigRepository = IAIModelConfigRepository
