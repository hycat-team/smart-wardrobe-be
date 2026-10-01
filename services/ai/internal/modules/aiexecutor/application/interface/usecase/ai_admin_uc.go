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
	// ListGrants is kept: it is the observation path and has no write side.
	// CreateGrant was removed in 024 (FR-035) because it could never succeed and
	// was a second writer on a table the AI service must own alone.
	ListGrants(ctx context.Context, userID uuid.UUID) ([]*dto.AIPolicyGrantDTO, error)
}
