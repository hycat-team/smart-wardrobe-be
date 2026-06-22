package validator

import (
	"context"
	"strings"
	"testing"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/shared/domain/constants/plankind"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type fakeSubscriptionPlanRepository struct {
	plans []*entities.SubscriptionPlan
}

func (f *fakeSubscriptionPlanRepository) GetAll(ctx context.Context) ([]*entities.SubscriptionPlan, error) {
	return f.plans, nil
}

func (f *fakeSubscriptionPlanRepository) GetByID(ctx context.Context, id uuid.UUID) (*entities.SubscriptionPlan, error) {
	return nil, nil
}

func (f *fakeSubscriptionPlanRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.SubscriptionPlan, error) {
	return nil, nil
}

func (f *fakeSubscriptionPlanRepository) Create(ctx context.Context, entity *entities.SubscriptionPlan) error {
	return nil
}

func (f *fakeSubscriptionPlanRepository) Update(ctx context.Context, entity *entities.SubscriptionPlan) error {
	return nil
}

func (f *fakeSubscriptionPlanRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (f *fakeSubscriptionPlanRepository) GetDefaultPlan(ctx context.Context) (*entities.SubscriptionPlan, error) {
	return nil, nil
}

func (f *fakeSubscriptionPlanRepository) GetBySlug(ctx context.Context, slug string) (*entities.SubscriptionPlan, error) {
	return nil, nil
}

func TestSubscriptionCatalogValidator(t *testing.T) {
	ctx := context.Background()

	t.Run("exactly one active default-free plan passes", func(t *testing.T) {
		repo := &fakeSubscriptionPlanRepository{
			plans: []*entities.SubscriptionPlan{
				{
					Slug:     "free",
					PlanKind: plankind.DefaultFree,
					IsActive: true,
				},
				{
					Slug:     "premium",
					PlanKind: plankind.Finite,
					IsActive: true,
				},
			},
		}

		attachTestPolicies(repo.plans)
		validator := NewSubscriptionCatalogValidator(repo, &config.Config{})
		err := validator.Validate(ctx)
		if err != nil {
			t.Fatalf("expected validation to pass, got error: %v", err)
		}
	})

	t.Run("zero active default-free plan fails", func(t *testing.T) {
		repo := &fakeSubscriptionPlanRepository{
			plans: []*entities.SubscriptionPlan{
				{
					Slug:     "free-inactive",
					PlanKind: plankind.DefaultFree,
					IsActive: false,
				},
				{
					Slug:     "premium",
					PlanKind: plankind.Finite,
					IsActive: true,
				},
			},
		}

		attachTestPolicies(repo.plans)
		validator := NewSubscriptionCatalogValidator(repo, &config.Config{})
		err := validator.Validate(ctx)
		if err == nil {
			t.Fatal("expected validation to fail, got nil")
		}
		if !strings.Contains(err.Error(), "missing default-free plan") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("multiple active default-free plans fail", func(t *testing.T) {
		repo := &fakeSubscriptionPlanRepository{
			plans: []*entities.SubscriptionPlan{
				{
					Slug:     "free-1",
					PlanKind: plankind.DefaultFree,
					IsActive: true,
				},
				{
					Slug:     "free-2",
					PlanKind: plankind.DefaultFree,
					IsActive: true,
				},
			},
		}

		attachTestPolicies(repo.plans)
		validator := NewSubscriptionCatalogValidator(repo, &config.Config{})
		err := validator.Validate(ctx)
		if err == nil {
			t.Fatal("expected validation to fail, got nil")
		}
		if !strings.Contains(err.Error(), "multiple active default-free plans found") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})
}

func attachTestPolicies(plans []*entities.SubscriptionPlan) {
	for _, p := range plans {
		id := uuid.New()
		p.AICostPolicyID = id
		p.AICostPolicy = &entities.AICostPolicy{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: id}}}
		for _, operation := range []string{"chat", "outfit", "summary", "rewriter"} {
			p.AICostPolicy.Operations = append(p.AICostPolicy.Operations, &entities.AICostPolicyOperation{Operation: operation, IsEnabled: true, NormalMaxInputTokens: 1, NormalMaxOutputTokens: 1, ReducedMaxInputTokens: 1, ReducedMaxOutputTokens: 1, MaxPaidAttemptsPerDay: 1})
		}
	}
}
