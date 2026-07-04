package grant

import (
	"context"
	"time"

	aiexecutor_dto "smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aipolicygrantstatus"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/repositories"

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

func NewGrantUseCase(aiCostRepo repositories.IAICostRepository, policyRepo repositories.IPricingConfigRepository) uc_interfaces.IGrantUseCase {
	return &GrantUseCase{aiCostRepo: aiCostRepo, policyRepo: policyRepo}
}

func parsePlanChangedPayload(p aiexecutor_dto.PlanChangedPayload) PlanChangedEvent {
	userID, _ := uuid.Parse(p.UserID)
	policyID, _ := uuid.Parse(p.AICostPolicyID)
	occurredAt, _ := time.Parse(time.RFC3339Nano, p.OccurredAt)
	return PlanChangedEvent{
		UserID:         userID,
		EventType:      p.EventType,
		ToPlanCode:     p.ToPlanCode,
		ToTierRank:     p.ToTierRank,
		AICostPolicyID: policyID,
		OccurredAt:     occurredAt,
	}
}

func (uc *GrantUseCase) HandlePlanChanged(ctx context.Context, payload aiexecutor_dto.PlanChangedPayload) error {
	event := parsePlanChangedPayload(payload)
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
