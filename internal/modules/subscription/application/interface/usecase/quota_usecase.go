package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/subscription/application/dto"

	"github.com/google/uuid"
)

type IUserQuotaUseCase interface {
	GetAndResetDailyQuota(ctx context.Context, userID uuid.UUID) (*dto.UserSubscriptionDTO, error)
	UpdateOutfitQuota(ctx context.Context, userID uuid.UUID, count int) error
	UpdateAiChatQuota(ctx context.Context, userID uuid.UUID, count int) error
}
