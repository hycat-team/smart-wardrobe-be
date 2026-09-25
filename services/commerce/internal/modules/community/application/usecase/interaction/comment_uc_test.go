package interaction

import (
	"context"
	"testing"

	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/poststatus"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCommentRepo struct {
	repositories.IPostCommentRepository
	byID map[uuid.UUID]*entities.PostComment
}

func (f *fakeCommentRepo) GetByID(_ context.Context, id uuid.UUID) (*entities.PostComment, error) {
	return f.byID[id], nil
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
