package validator

import (
	"context"
	"fmt"
	"smart-wardrobe-be/internal/bootstrap"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/plankind"
)

type SubscriptionCatalogValidator struct {
	planRepo repositories.ISubscriptionPlanRepository
}

func NewSubscriptionCatalogValidator(planRepo repositories.ISubscriptionPlanRepository) bootstrap.StartupValidator {
	return &SubscriptionCatalogValidator{
		planRepo: planRepo,
	}
}

func (v *SubscriptionCatalogValidator) Validate(ctx context.Context) error {
	plans, err := v.planRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to load subscription plans for startup validation: %w", err)
	}

	activeDefaultFreeCount := 0
	for _, p := range plans {
		if p.PlanKind == plankind.DefaultFree && p.IsActive {
			activeDefaultFreeCount++
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
