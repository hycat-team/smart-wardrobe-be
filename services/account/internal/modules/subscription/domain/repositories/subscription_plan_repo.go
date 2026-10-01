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
	// GetBySlugsAnyStatus resolves many slugs in one query, ignoring activation
	// status exactly as GetBySlugAnyStatus does.
	//
	// It exists for the batch grant path (024 T150). A 500-item list naming 500
	// distinct slugs used to cost 500 sequential round trips inside a single
	// request, all of them before the request's own time budget was even started.
	//
	// The returned map holds only the slugs that exist. A missing slug is absent
	// from it rather than present-and-nil, so a caller must treat "absent" and
	// "found" as different outcomes - which is the distinction the
	// plan_not_found reason code depends on.
	GetBySlugsAnyStatus(ctx context.Context, slugs []string) (map[string]*entities.SubscriptionPlan, error)
	ListAdmin(ctx context.Context) ([]*entities.SubscriptionPlan, error)
}
