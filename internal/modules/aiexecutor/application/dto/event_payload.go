package dto

// PlanChangedEventPayload is the RabbitMQ message payload published by the
// subscription module when a user's plan changes (consumed by the AI grant worker).
type PlanChangedEventPayload struct {
	UserID         string `json:"userId"`
	EventType      string `json:"eventType"`
	ToPlanCode     string `json:"toPlanCode"`
	ToTierRank     int    `json:"toTierRank"`
	AICostPolicyID string `json:"aiCostPolicyId"`
	OccurredAt     string `json:"occurredAt"`
}
