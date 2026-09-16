package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/services/ai/dto/aiexecutor"

	"github.com/google/uuid"
)

type IAIAdminUseCase interface {
	ListPricing(ctx context.Context) ([]*dto.AIModelConfigDTO, error)
	UpdatePricing(ctx context.Context, id uuid.UUID, req dto.UpdatePricingReq) (*dto.AIModelConfigDTO, error)
	GetUserUsage(ctx context.Context, userID uuid.UUID, fromDate, toDate *time.Time, page, limit int) ([]*dto.AIUsageItemDTO, int64, error)
	ListGrants(ctx context.Context, userID uuid.UUID) ([]*dto.AIPolicyGrantDTO, error)
	CreateGrant(ctx context.Context, req dto.CreateAIGrantReq) (*dto.AIPolicyGrantDTO, error)
}
