package mapper

import (
	aiexecutor_dto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"

	"github.com/google/uuid"
)

// 024 T127. The aiexecutor module had NO mapper package and every use case built
// its DTOs inline. The constitution requires mapper functions to live in
// `application/mapper` and never in use-case files, and `plan.md` assumed such a
// package already existed - so this one is created rather than reused.
//
// Three of the conversions below predate feature 024. They moved because leaving
// them would have created a module that is half conforming, which is harder to
// reason about than one that is fully fixed: the next reader could not tell
// whether a mapping in a use case file was an oversight or a decision.

// AIModelConfigToDTO converts a pricing configuration row.
//
// Shared by the list and single-update paths. They used to declare the same ten
// fields independently, so a field added to the entity reached one response and
// not the other - and the difference only showed up in production.
func AIModelConfigToDTO(c *aiexecutor_entities.AIModelConfig) *aiexecutor_dto.AIModelConfigDTO {
	if c == nil {
		return nil
	}
	return &aiexecutor_dto.AIModelConfigDTO{
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

// AIUsageItemToDTO converts a usage ledger row.
//
// `provider` and `model` arrive separately because the entity holds them as
// nullable pointers while the admin response always wants plain strings. The
// dereference happens here rather than at the call site so every caller unwraps
// them the same way.
func AIUsageItemToDTO(
	e *aiexecutor_entities.AIUsageEvent,
	uid uuid.UUID,
	provider string,
	model string,
) *aiexecutor_dto.AIUsageItemDTO {
	if e == nil {
		return nil
	}
	return &aiexecutor_dto.AIUsageItemDTO{
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

// AIPolicyGrantToDTO converts a grant row, with its policy summary attached.
//
// `policyBrief` is passed in rather than looked up: the use case batches the
// policy reads, and turning one row at a time back into a query would undo that.
// It is nil when the policy row is gone, which is the case an administrator most
// needs to see - the grant exists but its policy no longer does.
func AIPolicyGrantToDTO(
	g *aiexecutor_entities.AIPolicyGrant,
	policyBrief *aiexecutor_dto.AICostPolicyBriefRes,
) *aiexecutor_dto.AIPolicyGrantDTO {
	if g == nil {
		return nil
	}
	return &aiexecutor_dto.AIPolicyGrantDTO{
		ID:             g.ID,
		UserID:         g.UserID,
		AICostPolicy:   policyBrief,
		AICostPolicyID: g.PolicyID,
		EffectiveFrom:  g.EffectiveFrom,
		EffectiveTo:    g.EffectiveTo,
		CreatedAt:      g.CreatedAt,
	}
}

// AICostPolicyBriefToDTO converts a policy row to the short form embedded in a
// grant response.
func AICostPolicyBriefToDTO(policy *aiexecutor_entities.AICostPolicy) *aiexecutor_dto.AICostPolicyBriefRes {
	if policy == nil {
		return nil
	}
	return &aiexecutor_dto.AICostPolicyBriefRes{
		ID:      policy.ID,
		Code:    policy.Code,
		Name:    policy.Name,
		Version: policy.Version,
	}
}

// PricingConfigToDTO converts a pricing row for the internal pricing lookup.
//
// Same field list as AIModelConfigToDTO and a different DTO type, because two
// consumers want two shapes: an administrator's list view and the service-to-
// service pricing call. Kept as two functions rather than one with a flag, so
// adding a field to one cannot silently change the other.
func PricingConfigToDTO(c *aiexecutor_entities.AIModelConfig) *aiexecutor_dto.PricingConfigDTO {
	if c == nil {
		return nil
	}
	return &aiexecutor_dto.PricingConfigDTO{
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
