package event

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"
)

type CommunityEventUseCase struct {
	postRepo repositories.IPostRepository
}

func NewCommunityEventUseCase(postRepo repositories.IPostRepository) uc_interfaces.ICommunityEventUseCase {
	return &CommunityEventUseCase{postRepo: postRepo}
}

// HandleOutfitDeleted cascades an outfit deletion to its outfit posts (FR-005).
func (uc *CommunityEventUseCase) HandleOutfitDeleted(ctx context.Context, payload dto.OutfitDeletedEventPayload) error {
	_, err := uc.postRepo.SoftDeleteByOutfitID(ctx, payload.OutfitID)
	return err
}

// HandleUserStatusChanged hides/restores a user's posts (FR-035).
func (uc *CommunityEventUseCase) HandleUserStatusChanged(ctx context.Context, payload dto.UserStatusChangedEventPayload) error {
	if payload.Active {
		_, err := uc.postRepo.RestoreAllByAuthor(ctx, payload.UserID)
		return err
	}
	_, err := uc.postRepo.HideAllByAuthor(ctx, payload.UserID)
	return err
}
