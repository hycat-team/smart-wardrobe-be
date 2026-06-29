package validator

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/bootstrap"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/subscription/plankind"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/internal/shared/domain/constants/subscription/aienforcementmode"
)

const strictPolicyExposureWarnToleranceMicroVND int64 = 1_000_000_000 // 1,000 VND

type SubscriptionCatalogValidator struct {
	planRepo repositories.ISubscriptionPlanRepository
	cfg      *config.Config
}

func NewSubscriptionCatalogValidator(planRepo repositories.ISubscriptionPlanRepository, cfg *config.Config) bootstrap.StartupValidator {
	return &SubscriptionCatalogValidator{
		planRepo: planRepo,
		cfg:      cfg,
	}
}

func (v *SubscriptionCatalogValidator) Validate(ctx context.Context) error {
	plans, err := v.planRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to load subscription plans for startup validation: %w", err)
	}

	activeDefaultFreeCount := 0
	for _, p := range plans {
		if p.IsActive && (p.AICostPolicyID == uuid.Nil || p.AICostPolicy == nil) {
			return fmt.Errorf("subscription plan %q has no AI cost policy", p.Slug)
		}
		if p.IsActive && p.AICostPolicy != nil {
			if err := validatePolicyOperations(p.AICostPolicy); err != nil {
				return fmt.Errorf("subscription plan %q AI policy invalid: %w", p.Slug, err)
			}
		}
		if p.PlanKind == plankind.DefaultFree && p.IsActive {
			activeDefaultFreeCount++
		}
		if p.IsActive && p.AICostPolicy != nil && p.AICostPolicy.EnforcementMode == aienforcementmode.Strict {
			if err := v.validateStrictPolicy(p.AICostPolicy); err != nil {
				return fmt.Errorf("subscription plan %q AI policy invalid: %w", p.Slug, err)
			}
		}
	}

	if activeDefaultFreeCount == 0 {
		return fmt.Errorf("missing default-free plan: catalog must contain exactly one active DefaultFree plan")
	}
	if activeDefaultFreeCount > 1 {
		return fmt.Errorf("multiple active default-free plans found: catalog must contain exactly one active DefaultFree plan (found %d)", activeDefaultFreeCount)
	}

	return nil
}

func validatePolicyOperations(policy *entities.AICostPolicy) error {
	configured := map[string]bool{}
	for _, op := range policy.Operations {
		if !op.IsEnabled {
			continue
		}
		configured[op.Operation] = true
		if op.NormalMaxInputTokens <= 0 || op.NormalMaxOutputTokens <= 0 || op.ReducedMaxInputTokens <= 0 || op.ReducedMaxOutputTokens <= 0 || op.MaxPaidAttemptsPerDay <= 0 {
			return fmt.Errorf("operation %s has invalid limits", op.Operation)
		}
	}
	for _, required := range []string{"chat", "outfit", "summary", "rewriter"} {
		if !configured[required] {
			return fmt.Errorf("operation %s is missing", required)
		}
	}
	return nil
}

func (v *SubscriptionCatalogValidator) validateStrictPolicy(policy *entities.AICostPolicy) error {
	if policy.HardCostMicroVND == nil {
		return fmt.Errorf("strict policy hard cost is required")
	}
	in, err := decimal.NewFromString(v.cfg.AI.Pricing.Paid.InputUSDPerMillionTokens)
	if err != nil {
		return err
	}
	out, err := decimal.NewFromString(v.cfg.AI.Pricing.Paid.OutputUSDPerMillionTokens)
	if err != nil {
		return err
	}
	fx, err := decimal.NewFromString(v.cfg.AI.Pricing.USDToVND)
	if err != nil {
		return err
	}
	maxReservation := int64(0)
	operations := map[string]bool{}
	for _, op := range policy.Operations {
		operations[op.Operation] = true
		if op.NormalRoute != "paid" {
			continue
		}
		cost := decimal.NewFromInt(int64(op.NormalMaxInputTokens)).Mul(in).Mul(fx).Add(decimal.NewFromInt(int64(op.NormalMaxOutputTokens)).Mul(out).Mul(fx)).Ceil().IntPart()
		if cost > maxReservation {
			maxReservation = cost
		}
	}
	for _, required := range []string{"chat", "outfit", "summary", "rewriter"} {
		if !operations[required] {
			return fmt.Errorf("operation %s is missing", required)
		}
	}
	threshold := *policy.HardCostMicroVND * int64(policy.FreeRouteThresholdBPS) / 10000
	exposure := threshold + int64(policy.PeriodDays)*int64(policy.MaxUnknownPaidRequestsPerDay)*maxReservation
	if exposure > *policy.HardCostMicroVND {
		excess := exposure - *policy.HardCostMicroVND
		if excess <= strictPolicyExposureWarnToleranceMicroVND {
			fmt.Printf("WARN: strict AI policy exposure exceeded hard cost within tolerance (exposure=%d hard_cost=%d excess=%d tolerance=%d)\n", exposure, *policy.HardCostMicroVND, excess, strictPolicyExposureWarnToleranceMicroVND)
			return nil
		}
		return fmt.Errorf("maximum exposure %d exceeds hard cost %d by %d (tolerance %d)", exposure, *policy.HardCostMicroVND, excess, strictPolicyExposureWarnToleranceMicroVND)
	}
	return nil
}
