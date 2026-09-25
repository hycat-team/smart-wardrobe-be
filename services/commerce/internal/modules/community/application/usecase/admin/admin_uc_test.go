package admin

import (
	"context"
	"testing"

	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/hiddenreason"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/poststatus"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePostRepo struct {
	repositories.IPostRepository
	post          *entities.Post
	lastStatus    poststatus.PostStatus
	lastModerator *uuid.UUID
}

func (f *fakePostRepo) GetByID(context.Context, uuid.UUID) (*entities.Post, error) {
	return f.post, nil
}

func (f *fakePostRepo) UpdateModerationStatus(_ context.Context, _ uuid.UUID, status poststatus.PostStatus, moderatedBy *uuid.UUID, _ *hiddenreason.HiddenReason) error {
	f.lastStatus = status
	f.lastModerator = moderatedBy
	return nil
}

type fakePublisher struct {
	topic  string
	called bool
}

func (f *fakePublisher) Publish(_ context.Context, topic string, _ any) error {
	f.topic = topic
	f.called = true
	return nil
}

func newAdminUC(post *entities.Post) (*AdminUseCase, *fakePostRepo, *fakePublisher) {
	repo := &fakePostRepo{post: post}
	pub := &fakePublisher{}
	return &AdminUseCase{cfg: &config.Config{}, postRepo: repo, eventPublisher: pub}, repo, pub
}

func TestRestorePost(t *testing.T) {
	adminID := uuid.New()

	t.Run("rejects restoring author-deleted post", func(t *testing.T) {
		post := &entities.Post{Status: poststatus.Deleted, ModeratedBy: nil}
		post.ID = uuid.New()
		uc, _, _ := newAdminUC(post)
		err := uc.RestorePost(context.Background(), adminID, post.ID)
		require.Error(t, err)
	})

	t.Run("restores admin-deleted post", func(t *testing.T) {
		mod := uuid.New()
		post := &entities.Post{Status: poststatus.Deleted, ModeratedBy: &mod}
		post.ID = uuid.New()
		uc, repo, pub := newAdminUC(post)
		require.NoError(t, uc.RestorePost(context.Background(), adminID, post.ID))
		assert.Equal(t, poststatus.Published, repo.lastStatus)
		assert.Nil(t, repo.lastModerator)
		assert.True(t, pub.called)
		assert.Equal(t, eventconstants.TopicAuditLog, pub.topic)
	})
}

func TestHidePostPublishesAudit(t *testing.T) {
	adminID := uuid.New()
	post := &entities.Post{Status: poststatus.Published}
	post.ID = uuid.New()
	uc, repo, pub := newAdminUC(post)
	require.NoError(t, uc.HidePost(context.Background(), adminID, post.ID))
	assert.Equal(t, poststatus.Hidden, repo.lastStatus)
	require.NotNil(t, repo.lastModerator)
	assert.Equal(t, adminID, *repo.lastModerator)
	assert.True(t, pub.called)
}
