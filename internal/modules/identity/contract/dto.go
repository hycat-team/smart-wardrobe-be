package contract

import (
	"time"

	"github.com/google/uuid"
)

type PublicUserDTO struct {
	ID                   uuid.UUID
	Username             string
	Email                string
	RoleSlug             string
	SubscriptionPlanID   uuid.UUID
	OutfitRecommendCount int
	AiUsageCount         int
	LastResetDate        time.Time
}
