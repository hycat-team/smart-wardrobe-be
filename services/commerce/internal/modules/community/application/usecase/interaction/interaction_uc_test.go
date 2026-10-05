package interaction

import (
	"context"
	"testing"

	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/poststatus"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"
	"smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeInteractionUoW struct{}

func (fakeInteractionUoW) Execute(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type fakeLikeRepo struct {
	repositories.IPostLikeRepository
	exists         bool
	deleteAffected int64
	deleteCalls    int
	createCalls    int
}

func (f *fakeLikeRepo) Exists(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return f.exists, nil
}

func (f *fakeLikeRepo) Delete(context.Context, uuid.UUID, uuid.UUID) (int64, error) {
	f.deleteCalls++
	return f.deleteAffected, nil
}

func (f *fakeLikeRepo) Create(context.Context, *entities.PostLike) error {
	f.createCalls++
	return nil
}

type fakePostRepoSpy struct {
	repositories.IPostRepository
	post          *entities.Post
	likeDeltas    []int
	commentDeltas []int
}

func (f *fakePostRepoSpy) GetByPublicID(context.Context, string) (*entities.Post, error) {
	return f.post, nil
}

func (f *fakePostRepoSpy) IncrementLikeCount(_ context.Context, _ uuid.UUID, delta int) error {
	f.likeDeltas = append(f.likeDeltas, delta)
	return nil
}

func (f *fakePostRepoSpy) IncrementCommentCount(_ context.Context, _ uuid.UUID, delta int) error {
	f.commentDeltas = append(f.commentDeltas, delta)
	return nil
}

func TestTogglePostLike(t *testing.T) {
	userID := uuid.New()
	post := &entities.Post{Status: poststatus.Published}
	post.ID = uuid.New()
	post.UserID = userID
	post.PublicID = "post-public-id"

	newUC := func(likeRepo *fakeLikeRepo, postRepo *fakePostRepoSpy) *InteractionUseCase {
		return &InteractionUseCase{
			cfg:      &config.Config{},
			uow:      fakeInteractionUoW{},
			postRepo: postRepo,
			likeRepo: likeRepo,
		}
	}

	t.Run("unlike decrements once", func(t *testing.T) {
		likeRepo := &fakeLikeRepo{exists: true, deleteAffected: 1}
		postRepo := &fakePostRepoSpy{post: post}

		err := newUC(likeRepo, postRepo).TogglePostLike(context.Background(), userID, post.PublicID, false)

		require.NoError(t, err)
		assert.Equal(t, 1, likeRepo.deleteCalls)
		assert.Equal(t, []int{-1}, postRepo.likeDeltas)
	})

	t.Run("unlike that deleted no row does not decrement", func(t *testing.T) {
		likeRepo := &fakeLikeRepo{exists: true, deleteAffected: 0}
		postRepo := &fakePostRepoSpy{post: post}

		err := newUC(likeRepo, postRepo).TogglePostLike(context.Background(), userID, post.PublicID, false)

		require.NoError(t, err)
		assert.Equal(t, 1, likeRepo.deleteCalls)
		assert.Empty(t, postRepo.likeDeltas)
	})

	t.Run("like creates the row and increments", func(t *testing.T) {
		likeRepo := &fakeLikeRepo{exists: false}
		postRepo := &fakePostRepoSpy{post: post}

		err := newUC(likeRepo, postRepo).TogglePostLike(context.Background(), userID, post.PublicID, true)

		require.NoError(t, err)
		assert.Equal(t, 1, likeRepo.createCalls)
		assert.Equal(t, []int{1}, postRepo.likeDeltas)
	})

	t.Run("same state is a no-op", func(t *testing.T) {
		likeRepo := &fakeLikeRepo{exists: true}
		postRepo := &fakePostRepoSpy{post: post}

		err := newUC(likeRepo, postRepo).TogglePostLike(context.Background(), userID, post.PublicID, true)

		require.NoError(t, err)
		assert.Equal(t, 0, likeRepo.createCalls)
		assert.Equal(t, 0, likeRepo.deleteCalls)
		assert.Empty(t, postRepo.likeDeltas)
	})
}
