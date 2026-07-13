package dto

type PlanChangedPayload struct {
	UserID         string `json:"userId"`
	EventType      string `json:"eventType"`
	ToPlanCode     string `json:"toPlanCode"`
	ToTierRank     int    `json:"toTierRank"`
	AICostPolicyID string `json:"aiCostPolicyId"`
	OccurredAt     string `json:"occurredAt"`
}
