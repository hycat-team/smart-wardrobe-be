package validator

import (
	"context"
	"strings"
	"testing"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/shared/domain/constants/subscription/plankind"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/internal/shared/domain/constants/subscription/aienforcementmode"

	"github.com/google/uuid"
)

func int64Ptr(v int64) *int64 { return &v }

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

	t.Run("strict policy overflow within 1000 VND tolerance warns and passes", func(t *testing.T) {
		repo := &fakeSubscriptionPlanRepository{
			plans: []*entities.SubscriptionPlan{
				{
					Slug:     "free",
					PlanKind: plankind.DefaultFree,
					IsActive: true,
				},
				{
					Slug:     "premium-monthly",
					PlanKind: plankind.Finite,
					IsActive: true,
				},
			},
		}

		attachTestPolicies(repo.plans)
		repo.plans[1].AICostPolicy.EnforcementMode = aienforcementmode.Strict
		repo.plans[1].AICostPolicy.PeriodDays = 1
		repo.plans[1].AICostPolicy.FreeRouteThresholdBPS = 9600
		repo.plans[1].AICostPolicy.HardCostMicroVND = int64Ptr(25_000_000_000)
		repo.plans[1].AICostPolicy.MaxUnknownPaidRequestsPerDay = 1
		for _, op := range repo.plans[1].AICostPolicy.Operations {
			op.NormalRoute = "paid"
			op.NormalMaxInputTokens = 444_440
			op.NormalMaxOutputTokens = 1
		}

		validator := NewSubscriptionCatalogValidator(repo, &config.Config{
			AI: config.AIServiceConfig{
				Pricing: config.AIPricingConfig{
					USDToVND: "27000",
					Paid: config.AIModelPricingConfig{
						InputUSDPerMillionTokens:  "0.10",
						OutputUSDPerMillionTokens: "0.40",
					},
				},
			},
		})
		err := validator.Validate(ctx)
		if err != nil {
			t.Fatalf("expected validation to pass within tolerance, got error: %v", err)
		}
	})

	t.Run("strict policy overflow above 1000 VND tolerance fails", func(t *testing.T) {
		repo := &fakeSubscriptionPlanRepository{
			plans: []*entities.SubscriptionPlan{
				{
					Slug:     "free",
					PlanKind: plankind.DefaultFree,
					IsActive: true,
				},
				{
					Slug:     "premium-monthly",
					PlanKind: plankind.Finite,
					IsActive: true,
				},
			},
		}

		attachTestPolicies(repo.plans)
		repo.plans[1].AICostPolicy.EnforcementMode = aienforcementmode.Strict
		repo.plans[1].AICostPolicy.PeriodDays = 1
		repo.plans[1].AICostPolicy.FreeRouteThresholdBPS = 9600
		repo.plans[1].AICostPolicy.HardCostMicroVND = int64Ptr(25_000_000_000)
		repo.plans[1].AICostPolicy.MaxUnknownPaidRequestsPerDay = 1
		for _, op := range repo.plans[1].AICostPolicy.Operations {
			op.NormalRoute = "paid"
			op.NormalMaxInputTokens = 814_820
			op.NormalMaxOutputTokens = 1
		}

		validator := NewSubscriptionCatalogValidator(repo, &config.Config{
			AI: config.AIServiceConfig{
				Pricing: config.AIPricingConfig{
					USDToVND: "27000",
					Paid: config.AIModelPricingConfig{
						InputUSDPerMillionTokens:  "0.10",
						OutputUSDPerMillionTokens: "0.40",
					},
				},
			},
		})
		err := validator.Validate(ctx)
		if err == nil {
			t.Fatal("expected validation to fail when overflow exceeds tolerance, got nil")
		}
		if !strings.Contains(err.Error(), "tolerance") {
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
