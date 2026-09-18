package repositories

import (
	"context"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

type ISubscriptionPlanRepository interface {
	repositories.IGenericRepository[entities.SubscriptionPlan, uuid.UUID]
	GetDefaultPlan(ctx context.Context) (*entities.SubscriptionPlan, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.SubscriptionPlan, error)
	GetBySlug(ctx context.Context, slug string) (*entities.SubscriptionPlan, error)
	GetBySlugAnyStatus(ctx context.Context, slug string) (*entities.SubscriptionPlan, error)
	ListAdmin(ctx context.Context) ([]*entities.SubscriptionPlan, error)
}
