package repositories

import (
	"context"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type ISubscriptionPlanRepository interface {
	repositories.IGenericRepository[entities.SubscriptionPlan, uuid.UUID]
	GetDefaultPlan(ctx context.Context) (*entities.SubscriptionPlan, error)
}
