package grant

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/repositories"
	"smart-wardrobe-be/internal/modules/subscription/domain/constants/aipolicygrantstatus"

	"github.com/google/uuid"
)

type PlanChangedEvent struct {
	UserID         uuid.UUID
	EventType      string
	ToPlanCode     string
	ToTierRank     int
	AICostPolicyID uuid.UUID
	OccurredAt     time.Time
}

type GrantUseCase struct {
	aiCostRepo repositories.IAICostRepository
	policyRepo repositories.IPricingConfigRepository
}

func NewGrantUseCase(aiCostRepo repositories.IAICostRepository, policyRepo repositories.IPricingConfigRepository) *GrantUseCase {
	return &GrantUseCase{aiCostRepo: aiCostRepo, policyRepo: policyRepo}
}

func (uc *GrantUseCase) HandlePlanChanged(ctx context.Context, payload PlanChangedPayload) error {
	event := payload.ToDomain()
	switch event.EventType {
	case "EXTENDED", "RENEWED":
		return nil
	case "SUBSCRIPTION_DOWNGRADED_TO_FREE":
		return uc.expireCurrentGrants(ctx, event.UserID, event.OccurredAt)
	case "EXPIRED":
		return uc.expireCurrentGrants(ctx, event.UserID, event.OccurredAt)
	}

	if err := uc.expireCurrentGrants(ctx, event.UserID, event.OccurredAt); err != nil {
		return err
	}

	return uc.createGrant(ctx, event)
}

func (uc *GrantUseCase) expireCurrentGrants(ctx context.Context, userID uuid.UUID, now time.Time) error {
	return uc.aiCostRepo.ExpireGrantsByUserID(ctx, userID, now)
}

func (uc *GrantUseCase) createGrant(ctx context.Context, event PlanChangedEvent) error {
	grant := &entities.AIPolicyGrant{
		UserID:        event.UserID,
		PolicyID:      event.AICostPolicyID,
		PlanID:        uuid.Nil,
		PlanCode:      event.ToPlanCode,
		TierRank:      event.ToTierRank,
		EffectiveFrom: event.OccurredAt,
		Status:        aipolicygrantstatus.Active,
	}
	return uc.aiCostRepo.CreateGrant(ctx, grant)
}
