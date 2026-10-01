package admin

import (
	"context"
	"time"

	"smart-wardrobe-be/services/ai/dto/aiexecutor"
	aierrors "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/errors"
	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	aiexecutor_mapper "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/mapper"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

type AIAdminUseCase struct {
	pricingRepo repositories.IPricingConfigRepository
	aiCostRepo  repositories.IAICostRepository
	uow         shared_repos.IUnitOfWork
}

func NewAIAdminUseCase(
	pricingRepo repositories.IPricingConfigRepository,
	aiCostRepo repositories.IAICostRepository,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IAIAdminUseCase {
	return &AIAdminUseCase{
		pricingRepo: pricingRepo,
		aiCostRepo:  aiCostRepo,
		uow:         uow,
	}
}

func (uc *AIAdminUseCase) ListPricing(ctx context.Context) ([]*dto.AIModelConfigDTO, error) {
	configs, err := uc.pricingRepo.ListAdmin(ctx)
	if err != nil {
		return nil, err
	}

	res := make([]*dto.AIModelConfigDTO, len(configs))
	for idx, c := range configs {
		res[idx] = aiexecutor_mapper.AIModelConfigToDTO(c)
	}
	return res, nil
}

func (uc *AIAdminUseCase) UpdatePricing(ctx context.Context, id uuid.UUID, req dto.UpdatePricingReq) (*dto.AIModelConfigDTO, error) {
	config, err := uc.pricingRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, aierrors.ErrPricingConfigNotFound()
	}

	if req.InputUSDPerMillion != nil {
		config.InputUSDPerMillion = *req.InputUSDPerMillion
	}
	if req.OutputUSDPerMillion != nil {
		config.OutputUSDPerMillion = *req.OutputUSDPerMillion
	}
	if req.USDToVND != nil {
		config.USDToVND = *req.USDToVND
	}
	if req.EffectiveTo != nil {
		if req.EffectiveTo.Before(config.EffectiveFrom) {
			return nil, aierrors.ErrEffectiveToBeforeEffectiveFrom()
		}
		config.EffectiveTo = req.EffectiveTo
	}

	if err := uc.pricingRepo.Update(ctx, config); err != nil {
		return nil, err
	}

	return aiexecutor_mapper.AIModelConfigToDTO(config), nil
}

func (uc *AIAdminUseCase) GetUserUsage(ctx context.Context, userID uuid.UUID, fromDate, toDate *time.Time, page, limit int) ([]*dto.AIUsageItemDTO, int64, error) {
	events, total, err := uc.aiCostRepo.GetUsageByUser(ctx, userID, fromDate, toDate, page, limit)
	if err != nil {
		return nil, 0, err
	}

	res := make([]*dto.AIUsageItemDTO, len(events))
	for idx, e := range events {
		var uid uuid.UUID
		if e.UserID != nil {
			uid = *e.UserID
		}
		var provider, model string
		if e.Provider != nil {
			provider = *e.Provider
		}
		if e.Model != nil {
			model = *e.Model
		}

		res[idx] = aiexecutor_mapper.AIUsageItemToDTO(e, uid, provider, model)
	}
	return res, total, nil
}

func (uc *AIAdminUseCase) ListGrants(ctx context.Context, userID uuid.UUID) ([]*dto.AIPolicyGrantDTO, error) {
	grants, err := uc.aiCostRepo.ListGrantsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	policyIDs := make(map[uuid.UUID]bool)
	for _, g := range grants {
		policyIDs[g.PolicyID] = true
	}
	policyMap := make(map[uuid.UUID]*entities.AICostPolicy)
	for pid := range policyIDs {
		policy, _ := uc.aiCostRepo.GetPolicyByID(ctx, pid)
		if policy != nil {
			policyMap[pid] = policy
		}
	}

	res := make([]*dto.AIPolicyGrantDTO, len(grants))
	for idx, g := range grants {
		// 024 T127: the policy rows were read once above; converting them here
		// keeps that batching. A nil brief is the case worth seeing - the grant
		// exists but the policy it points at is gone.
		policyBrief := aiexecutor_mapper.AICostPolicyBriefToDTO(policyMap[g.PolicyID])
		res[idx] = aiexecutor_mapper.AIPolicyGrantToDTO(g, policyBrief)
	}
	return res, nil
}
