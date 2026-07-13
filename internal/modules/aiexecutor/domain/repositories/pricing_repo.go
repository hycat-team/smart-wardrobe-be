package repositories

import (
	"context"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aimodel"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aiprovider"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type IPricingConfigRepository interface {
	shared_repos.IGenericRepository[entities.AIPricingConfig, uuid.UUID]
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.AIPricingConfig, error)
	GetActiveByProviderAndModel(ctx context.Context, provider aiprovider.Provider, model aimodel.Model) (*entities.AIPricingConfig, error)
}
