package usecase

import (
	"context"
	"smart-wardrobe-be/internal/modules/subscription/application/dto"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/shared/observability/workerlog"

	"github.com/google/uuid"
)

type ISubscriptionUseCase interface {
	GetDailyQuota(ctx context.Context, userID uuid.UUID) (*contract.UserSubscriptionDTO, error)
	ProcessScheduledRenewals(ctx context.Context, run *workerlog.Run) error
	SetAutoRenewStatus(ctx context.Context, userID uuid.UUID, enable bool) (bool, error)
	GetPlans(ctx context.Context) ([]*dto.SubscriptionPlanDTO, error)

	contract.IUserSubscriptionContract
}
