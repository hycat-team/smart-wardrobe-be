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

type fakeCommentRepo struct {
	repositories.IPostCommentRepository
	byID    map[uuid.UUID]*entities.PostComment
	updated []*entities.PostComment
}

func (f *fakeCommentRepo) GetByID(_ context.Context, id uuid.UUID) (*entities.PostComment, error) {
	return f.byID[id], nil
}

func (f *fakeCommentRepo) Update(_ context.Context, comment *entities.PostComment) error {
	f.updated = append(f.updated, comment)
	return nil
}

func TestResolveCommentParent(t *testing.T) {
	postID := uuid.New()
	root := &entities.PostComment{PostID: postID, Status: poststatus.Published}
	root.ID = uuid.New()
	level1 := &entities.PostComment{PostID: postID, Status: poststatus.Published, ParentCommentID: &root.ID, RootCommentID: &root.ID}
	level1.ID = uuid.New()

	uc := &InteractionUseCase{
		commentRepo: &fakeCommentRepo{byID: map[uuid.UUID]*entities.PostComment{
			root.ID:   root,
			level1.ID: level1,
		}},
	}

	t.Run("root comment has no parent", func(t *testing.T) {
		parentID, rootID, err := uc.resolveCommentParent(context.Background(), postID, nil)
		require.NoError(t, err)
		assert.Nil(t, parentID)
		assert.Nil(t, rootID)
	})

	t.Run("reply to root attaches to root", func(t *testing.T) {
		parentID, rootID, err := uc.resolveCommentParent(context.Background(), postID, &root.ID)
		require.NoError(t, err)
		require.NotNil(t, parentID)
		require.NotNil(t, rootID)
		assert.Equal(t, root.ID, *parentID)
		assert.Equal(t, root.ID, *rootID)
	})

	t.Run("reply to level 1 is reattached to root (max 2 levels)", func(t *testing.T) {
		parentID, rootID, err := uc.resolveCommentParent(context.Background(), postID, &level1.ID)
		require.NoError(t, err)
		require.NotNil(t, parentID)
		require.NotNil(t, rootID)
		assert.Equal(t, root.ID, *parentID)
		assert.Equal(t, root.ID, *rootID)
	})

	t.Run("rejects parent from another post", func(t *testing.T) {
		other := &entities.PostComment{PostID: uuid.New(), Status: poststatus.Published}
		other.ID = uuid.New()
		uc.commentRepo = &fakeCommentRepo{byID: map[uuid.UUID]*entities.PostComment{other.ID: other}}
		_, _, err := uc.resolveCommentParent(context.Background(), postID, &other.ID)
		require.Error(t, err)
	})
}

func TestDeleteCommentDecrementsCounter(t *testing.T) {
	userID := uuid.New()
	post := &entities.Post{Status: poststatus.Published}
	post.ID = uuid.New()
	post.PublicID = "post-public-id"

	comment := &entities.PostComment{PostID: post.ID, UserID: userID, Status: poststatus.Published}
	comment.ID = uuid.New()

	newUC := func() (*InteractionUseCase, *fakeCommentRepo, *fakePostRepoSpy) {
		commentRepo := &fakeCommentRepo{byID: map[uuid.UUID]*entities.PostComment{comment.ID: comment}}
		postRepo := &fakePostRepoSpy{post: post}
		return &InteractionUseCase{
			cfg:         &config.Config{},
			uow:         fakeInteractionUoW{},
			postRepo:    postRepo,
			commentRepo: commentRepo,
		}, commentRepo, postRepo
	}

	t.Run("owner delete soft-deletes and decrements once", func(t *testing.T) {
		uc, commentRepo, postRepo := newUC()

		err := uc.DeleteComment(context.Background(), userID, post.PublicID, comment.ID)

		require.NoError(t, err)
		require.Len(t, commentRepo.updated, 1)
		assert.Equal(t, poststatus.Deleted, commentRepo.updated[0].Status)
		assert.Equal(t, []int{-1}, postRepo.commentDeltas)
	})

	t.Run("non-owner delete does not touch the counter", func(t *testing.T) {
		uc, commentRepo, postRepo := newUC()

		err := uc.DeleteComment(context.Background(), uuid.New(), post.PublicID, comment.ID)

		require.Error(t, err)
		assert.Empty(t, commentRepo.updated)
		assert.Empty(t, postRepo.commentDeltas)
	})
}
