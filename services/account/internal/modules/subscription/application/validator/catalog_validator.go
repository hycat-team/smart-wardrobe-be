package validator

import (
	"context"
	"fmt"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/bootstrap"
	"smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
)

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
		if p.IsActive && p.AICostPolicyID == uuid.Nil {
			return fmt.Errorf("subscription plan %q has no AI cost policy", p.Slug)
		}
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
