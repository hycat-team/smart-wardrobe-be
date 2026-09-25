package event

import (
	"context"
	"testing"

	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePostRepo struct {
	repositories.IPostRepository
	deletedOutfit uuid.UUID
	hiddenAuthor  uuid.UUID
	restoredUser  uuid.UUID
}

func (f *fakePostRepo) SoftDeleteByOutfitID(_ context.Context, outfitID uuid.UUID) (int64, error) {
	f.deletedOutfit = outfitID
	return 1, nil
}

func (f *fakePostRepo) HideAllByAuthor(_ context.Context, authorID uuid.UUID) (int64, error) {
	f.hiddenAuthor = authorID
	return 1, nil
}

func (f *fakePostRepo) RestoreAllByAuthor(_ context.Context, authorID uuid.UUID) (int64, error) {
	f.restoredUser = authorID
	return 1, nil
}

func TestHandleOutfitDeleted(t *testing.T) {
	repo := &fakePostRepo{}
	uc := NewCommunityEventUseCase(repo)
	outfitID := uuid.New()

	require.NoError(t, uc.HandleOutfitDeleted(context.Background(), dto.OutfitDeletedEventPayload{OutfitID: outfitID}))
	assert.Equal(t, outfitID, repo.deletedOutfit)
}

func TestHandleUserStatusChanged(t *testing.T) {
	userID := uuid.New()

	t.Run("inactive hides posts", func(t *testing.T) {
		repo := &fakePostRepo{}
		uc := NewCommunityEventUseCase(repo)
		require.NoError(t, uc.HandleUserStatusChanged(context.Background(), dto.UserStatusChangedEventPayload{UserID: userID, Active: false}))
		assert.Equal(t, userID, repo.hiddenAuthor)
		assert.Equal(t, uuid.Nil, repo.restoredUser)
	})

	t.Run("active restores posts", func(t *testing.T) {
		repo := &fakePostRepo{}
		uc := NewCommunityEventUseCase(repo)
		require.NoError(t, uc.HandleUserStatusChanged(context.Background(), dto.UserStatusChangedEventPayload{UserID: userID, Active: true}))
		assert.Equal(t, userID, repo.restoredUser)
		assert.Equal(t, uuid.Nil, repo.hiddenAuthor)
	})
}
