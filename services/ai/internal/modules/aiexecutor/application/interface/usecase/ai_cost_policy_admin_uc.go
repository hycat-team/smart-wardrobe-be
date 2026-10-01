package usecase

import (
	"context"

	"smart-wardrobe-be/services/ai/dto/aiexecutor"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"

	"github.com/google/uuid"
)

// IAICostPolicyAdminUseCase is the admin gate-2 programming interface:
// view and edit the per-day paid attempt limits the AI engine enforces
// (024 FR-004/FR-004b).
type IAICostPolicyAdminUseCase interface {
	ListCostPolicies(ctx context.Context) ([]*dto.AICostPolicyDTO, error)
	UpdateCostPolicyOperation(ctx context.Context, operationID uuid.UUID, operation aioperation.AIOperation, req dto.UpdateCostPolicyOperationReq, actor dto.CostPolicyActor) (*dto.UpdateCostPolicyOperationResp, error)
}
