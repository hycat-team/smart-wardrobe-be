package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
)

type IGrantUseCase interface {
	HandlePlanChanged(ctx context.Context, payload dto.PlanChangedPayload) error
}
