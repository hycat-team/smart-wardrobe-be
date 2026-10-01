// Package mapper holds the entity-to-DTO conversions of the aiexecutor module.
//
// 024 T127. The module had NO mapper package, and its cost-policy administration
// use case assembled DTOs inline. The constitution requires mapper functions to
// live in `application/mapper` and never in use-case files, and the plan assumed
// such a package already existed - so this one is created rather than reused.
//
// It is a package rather than a folder with nothing in it because there is exactly
// one conversion to move today. Creating the directory for a single function and
// leaving it to grow is the cheaper mistake than adding a second copy of the same
// mapping inside a use case later.
package mapper

import (
	aiexecutor_dto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
)

// CostPolicyToDTO converts a cost policy and its operations into the response DTO
// the admin interface serves (FR-004b).
//
// It lives here, not in the use case, so that the shape of what an administrator
// sees over HTTP is decided in one place. The use case decides when to call it.
//
// Gate 2 values (the money cap, the free-route threshold) ARE part of this DTO,
// unlike their Account-side counterpart: this interface is admin-only and the
// whole point of FR-004b is that an administrator can SEE tier-2 configuration.
// It is never served to a customer.
func CostPolicyToDTO(policy *aiexecutor_entities.AICostPolicy) *aiexecutor_dto.AICostPolicyDTO {
	if policy == nil {
		return nil
	}

	ops := make([]aiexecutor_dto.AICostPolicyOperationDTO, 0, len(policy.Operations))
	for _, op := range policy.Operations {
		if dto := AICostPolicyOperationToDTO(op); dto != nil {
			ops = append(ops, *dto)
		}
	}

	return &aiexecutor_dto.AICostPolicyDTO{
		ID:               policy.ID,
		Code:             policy.Code,
		Name:             policy.Name,
		Version:          policy.Version,
		EnforcementMode:  string(policy.EnforcementMode),
		PeriodDays:       policy.PeriodDays,
		HardCostMicroVND: policy.HardCostMicroVND,
		IsActive:         policy.IsActive,
		Operations:       ops,
	}
}

// AICostPolicyOperationToDTO converts one operation row.
//
// Exported because the audit view needs the same operation shape and duplicating
// the field list is how the two drift apart.
func AICostPolicyOperationToDTO(op *aiexecutor_entities.AICostPolicyOperation) *aiexecutor_dto.AICostPolicyOperationDTO {
	if op == nil {
		return nil
	}
	return &aiexecutor_dto.AICostPolicyOperationDTO{
		ID:                    op.ID,
		Operation:             string(op.Operation),
		NormalRoute:           string(op.NormalRoute),
		ReducedRoute:          string(op.ReducedRoute),
		FreeRoute:             string(op.FreeRoute),
		MaxPaidAttemptsPerDay: op.MaxPaidAttemptsPerDay,
		PaidFallbackEnabled:   op.PaidFallbackEnabled,
		IsEnabled:             op.IsEnabled,
	}
}
