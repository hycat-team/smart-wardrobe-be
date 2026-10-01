package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Access to aiexecutor.ai_cost_policies and aiexecutor.ai_cost_policy_operations.
//
// These live in their own file because ai_cost_repo_impl.go is already at the
// code-hygiene warn threshold (500 lines) and this table needs four more access
// paths for the admin gate-2 API (024 FR-004/FR-004b).

// ResolvePolicy finds the grant in force for an account and the live policy
// configuration that governs it.
//
// It is a pure read: no row is created, closed or modified. The previous
// version closed a grant whose plan did not match the one it was asked about,
// which meant a read could destroy an entitlement — and did so permanently,
// because the write it performed was indistinguishable from a legitimate
// revocation (024 FR-042).
func (r *AICostRepository) ResolvePolicy(ctx context.Context, userID uuid.UUID, operation aioperation.AIOperation, now time.Time) (*aiexecutor_entities.AIPolicyGrant, *aiexecutor_entities.AICostPolicy, *aiexecutor_entities.AICostPolicyOperation, error) {
	var grant aiexecutor_entities.AIPolicyGrant
	err := r.GetDB(ctx).
		Where("user_id = ? AND effective_from <= ? AND (effective_to IS NULL OR effective_to > ?)", userID, now, now).
		Order("effective_from DESC").
		First(&grant).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// No grant is not an error here — the caller degrades through the
		// fallback resolver. Returning a sentinel instead of a hard failure is
		// the whole point of GĐ1.
		return nil, nil, nil, nil
	} else if err != nil {
		return nil, nil, nil, err
	}

	policy, err := r.getPolicyBy(ctx, "id = ?", grant.PolicyID)
	if err != nil {
		return nil, nil, nil, err
	}
	if policy == nil {
		return nil, nil, nil, nil
	}
	// 024 T165: a deactivated policy stops granting, exactly like a disabled
	// operation below.
	//
	// FR-029b requires the live is-active flag to be read for enforcement precisely
	// so that disabling takes effect on accounts that ALREADY hold a grant — and
	// those are the accounts a grant alone cannot fix, because a lifetime grant is
	// never rewritten: it is only created when the account changes plan. Without this
	// check the flag was unreachable for exactly the accounts it exists to control,
	// which is why disabling a purchased plan's cost policy changed nothing at all.
	//
	// Returning nil rather than an error hands every caller the "no usable grant"
	// shape it already handles, so they degrade through the fallback tiers instead of
	// failing the request (FR-006). It is the same treatment the fallback resolver
	// gives a deactivated DEFAULT policy (grant_fallback_resolver.go).
	if !policy.IsActive {
		return nil, nil, nil, nil
	}
	for _, op := range policy.Operations {
		if op.Operation == operation && op.IsEnabled {
			return &grant, policy, op, nil
		}
	}
	return nil, nil, nil, nil
}

// GetPolicyByCode resolves a policy by its configuration code.
//
// The code comes from configuration (ai.default_cost_policy_code) rather than
// from a constant in the code: ai_cost_policies has no tier_rank and no
// is_default column, so nothing in the data marks a policy as the default, and
// hardcoding a code would both violate constitution §VIII and make changing the
// default a code change plus a release (research.md R8).
func (r *AICostRepository) GetPolicyByCode(ctx context.Context, code string) (*aiexecutor_entities.AICostPolicy, error) {
	return r.getPolicyBy(ctx, "code = ?", code)
}

// ListPolicies returns every cost policy with its operations preloaded.
func (r *AICostRepository) ListPolicies(ctx context.Context) ([]*aiexecutor_entities.AICostPolicy, error) {
	var policies []*aiexecutor_entities.AICostPolicy
	// GetQueryWithPreload applies the repository's registered "Operations"
	// relation, so each policy arrives with all four operation rows.
	if err := r.GetQueryWithPreload(ctx).Order("code ASC").Find(&policies).Error; err != nil {
		return nil, err
	}
	return policies, nil
}

// GetPolicyOperationByID loads one operation row plus its parent policy.
// The parent is required, not a convenience: the caller must read the
// enforcement mode and the hard cap before it may write the operation, because
// those two decide whether NULL is legal here (FR-005e).
func (r *AICostRepository) GetPolicyOperationByID(ctx context.Context, operationID uuid.UUID) (*aiexecutor_entities.AICostPolicyOperation, *aiexecutor_entities.AICostPolicy, error) {
	var op aiexecutor_entities.AICostPolicyOperation
	if err := r.GetDB(ctx).First(&op, "id = ?", operationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	var policy aiexecutor_entities.AICostPolicy
	if err := r.GetDB(ctx).First(&policy, "id = ?", op.PolicyID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	return &op, &policy, nil
}

// UpdatePolicyOperation persists a changed per-day paid attempt limit. A nil
// limit writes SQL NULL, which is the sanctioned way to record "this policy
// does not use a per-day attempt cap" (FR-005e).
func (r *AICostRepository) UpdatePolicyOperation(ctx context.Context, op *aiexecutor_entities.AICostPolicyOperation) error {
	return r.GetDB(ctx).Save(op).Error
}

// CountGrants returns the total number of grant rows and how many of them have
// effective_to IS NULL.
func (r *AICostRepository) CountGrants(ctx context.Context) (int64, int64, error) {
	db := r.GetDB(ctx).Model(&aiexecutor_entities.AIPolicyGrant{})
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return 0, 0, err
	}
	var openEnded int64
	if err := db.Where("effective_to IS NULL").Count(&openEnded).Error; err != nil {
		return 0, 0, err
	}
	return total, openEnded, nil
}
