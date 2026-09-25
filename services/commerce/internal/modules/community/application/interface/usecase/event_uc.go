package usecase

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
)

type ICommunityEventUseCase interface {
	HandleOutfitDeleted(ctx context.Context, payload dto.OutfitDeletedEventPayload) error
	HandleUserStatusChanged(ctx context.Context, payload dto.UserStatusChangedEventPayload) error
}
