package usecase

import (
	"context"

	"smart-wardrobe-be/services/account/dto/subscription"

	"github.com/google/uuid"
)

type ISubscriptionPlanUseCase interface {
	GetPlans(ctx context.Context) ([]*dto.SubscriptionPlanBriefRes, error)
	ListAdminPlans(ctx context.Context) ([]*dto.SubscriptionPlanDTO, error)
	GetDefaultSubscriptionPlanID(ctx context.Context) (uuid.UUID, error)
	IsPremiumPlan(ctx context.Context, planID uuid.UUID) (bool, error)
	CreatePlan(ctx context.Context, req *dto.CreateSubscriptionPlanReq) (*dto.SubscriptionPlanDTO, error)
	UpdatePlan(ctx context.Context, id uuid.UUID, req *dto.UpdateSubscriptionPlanReq) (*dto.SubscriptionPlanDTO, error)
	SetPlanActive(ctx context.Context, id uuid.UUID, isActive bool) (*dto.SubscriptionPlanDTO, error)
}
