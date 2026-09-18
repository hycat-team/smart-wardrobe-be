package admin

import (
	"context"
	"time"

	"smart-wardrobe-be/services/ai/dto/aiexecutor"
	aierrors "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/errors"
	uc_interfaces "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
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
		res[idx] = &dto.AIModelConfigDTO{
			ID:                  c.ID,
			Provider:            c.Provider,
			Model:               c.Model,
			InputUSDPerMillion:  c.InputUSDPerMillion,
			OutputUSDPerMillion: c.OutputUSDPerMillion,
			Currency:            c.Currency,
			USDToVND:            c.USDToVND,
			Version:             c.Version,
			EffectiveFrom:       c.EffectiveFrom,
			EffectiveTo:         c.EffectiveTo,
		}
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

	return &dto.AIModelConfigDTO{
		ID:                  config.ID,
		Provider:            config.Provider,
		Model:               config.Model,
		InputUSDPerMillion:  config.InputUSDPerMillion,
		OutputUSDPerMillion: config.OutputUSDPerMillion,
		Currency:            config.Currency,
		USDToVND:            config.USDToVND,
		Version:             config.Version,
		EffectiveFrom:       config.EffectiveFrom,
		EffectiveTo:         config.EffectiveTo,
	}, nil
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

		res[idx] = &dto.AIUsageItemDTO{
			ID:                 e.ID,
			RequestID:          e.RequestID,
			UserID:             uid,
			Operation:          string(e.Operation),
			Status:             string(e.Status),
			PromptTokens:       e.PromptTokens,
			OutputTokens:       e.OutputTokens,
			ThinkingTokens:     e.ThinkingTokens,
			ActualCostMicroVND: e.ActualCostMicroVND,
			Provider:           provider,
			Model:              model,
			CreatedAt:          e.CreatedAt,
		}
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
		var policyBrief *dto.AICostPolicyBriefRes
		if policy, ok := policyMap[g.PolicyID]; ok {
			policyBrief = &dto.AICostPolicyBriefRes{
				ID:      policy.ID,
				Code:    policy.Code,
				Name:    policy.Name,
				Version: policy.Version,
			}
		}
		res[idx] = &dto.AIPolicyGrantDTO{
			ID:             g.ID,
			UserID:         g.UserID,
			AICostPolicy:   policyBrief,
			AICostPolicyID: g.PolicyID,
			EffectiveFrom:  g.EffectiveFrom,
			EffectiveTo:    g.EffectiveTo,
			CreatedAt:      g.CreatedAt,
		}
	}
	return res, nil
}

func (uc *AIAdminUseCase) CreateGrant(ctx context.Context, req dto.CreateAIGrantReq) (*dto.AIPolicyGrantDTO, error) {
	now := time.Now()
	grant := &entities.AIPolicyGrant{
		UserID:        req.UserID,
		PolicyID:      req.AICostPolicyID,
		EffectiveFrom: now,
		EffectiveTo:   req.EffectiveTo,
	}

	txFn := func(txCtx context.Context) error {
		// Expire prior active grants if necessary
		_ = uc.aiCostRepo.ExpireGrantsByUserID(txCtx, req.UserID, now)
		return uc.aiCostRepo.CreateGrant(txCtx, grant)
	}

	if err := uc.uow.Execute(ctx, txFn); err != nil {
		return nil, err
	}

	return &dto.AIPolicyGrantDTO{
		ID:             grant.ID,
		UserID:         grant.UserID,
		AICostPolicyID: grant.PolicyID,
		EffectiveFrom:  grant.EffectiveFrom,
		EffectiveTo:    grant.EffectiveTo,
		CreatedAt:      grant.CreatedAt,
	}, nil
}
