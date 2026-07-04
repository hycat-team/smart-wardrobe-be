package grant

import (
	"time"

	"github.com/google/uuid"
)

type PlanChangedPayload struct {
	UserID         string `json:"userId"`
	EventType      string `json:"eventType"`
	ToPlanCode     string `json:"toPlanCode"`
	ToTierRank     int    `json:"toTierRank"`
	AICostPolicyID string `json:"aiCostPolicyId"`
	OccurredAt     string `json:"occurredAt"`
}

func (p PlanChangedPayload) ToDomain() PlanChangedEvent {
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
