package usecase

import (
	"context"

	"smart-wardrobe-be/services/ai/dto/aiexecutor"
)

type IGrantUseCase interface {
	HandlePlanChanged(ctx context.Context, payload dto.PlanChangedEventPayload) error
}
