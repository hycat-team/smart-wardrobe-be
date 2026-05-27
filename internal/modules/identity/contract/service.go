package contract

import (
	"context"

	"github.com/google/uuid"
)

type IIdentityModuleContract interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*PublicUserDTO, error)
	UpdateOutfitQuota(ctx context.Context, userID uuid.UUID, count int, resetDate bool) error
	UpdateAiChatQuota(ctx context.Context, userID uuid.UUID, count int, resetDate bool) error
	ResetDailyQuotas(ctx context.Context, userID uuid.UUID) error
}
