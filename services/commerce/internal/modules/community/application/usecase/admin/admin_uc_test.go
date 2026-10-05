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
	moderateCalls int
	recountCalls  int
}

func (f *fakePostRepo) GetByID(context.Context, uuid.UUID) (*entities.Post, error) {
	return f.post, nil
}

func (f *fakePostRepo) UpdateModerationStatus(_ context.Context, _ uuid.UUID, status poststatus.PostStatus, moderatedBy *uuid.UUID, _ *hiddenreason.HiddenReason) error {
	f.lastStatus = status
	f.lastModerator = moderatedBy
	f.moderateCalls++
	return nil
}

func (f *fakePostRepo) RecountCommentCount(context.Context, uuid.UUID) error {
	f.recountCalls++
	return nil
}

type fakeCommentRepo struct {
	repositories.IPostCommentRepository
	comment       *entities.PostComment
	updateCalls   int
	cascadeCalls  int
	cascadeStatus poststatus.PostStatus
}

func (f *fakeCommentRepo) GetByID(context.Context, uuid.UUID) (*entities.PostComment, error) {
	return f.comment, nil
}

func (f *fakeCommentRepo) Update(_ context.Context, comment *entities.PostComment) error {
	f.updateCalls++
	f.comment = comment
	return nil
}

func (f *fakeCommentRepo) UpdateStatusByRoot(_ context.Context, _ uuid.UUID, status poststatus.PostStatus, _ *uuid.UUID) (int64, error) {
	f.cascadeCalls++
	f.cascadeStatus = status
	return 0, nil
}

type fakeAdminUoW struct{}

func (fakeAdminUoW) Execute(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
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
	return &AdminUseCase{cfg: &config.Config{}, uow: fakeAdminUoW{}, postRepo: repo, eventPublisher: pub}, repo, pub
}

func newCommentAdminUC(comment *entities.PostComment) (*AdminUseCase, *fakeCommentRepo, *fakePostRepo, *fakePublisher) {
	commentRepo := &fakeCommentRepo{comment: comment}
	postRepo := &fakePostRepo{}
	pub := &fakePublisher{}
	return &AdminUseCase{
		cfg:            &config.Config{},
		uow:            fakeAdminUoW{},
		postRepo:       postRepo,
		commentRepo:    commentRepo,
		eventPublisher: pub,
	}, commentRepo, postRepo, pub
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

func TestModerateComment(t *testing.T) {
	adminID := uuid.New()

	t.Run("hide recounts comment_count and audits", func(t *testing.T) {
		comment := &entities.PostComment{PostID: uuid.New(), Status: poststatus.Published}
		comment.ID = uuid.New()
		uc, commentRepo, postRepo, pub := newCommentAdminUC(comment)

		require.NoError(t, uc.HideComment(context.Background(), adminID, comment.ID))

		assert.Equal(t, 1, commentRepo.updateCalls)
		assert.Equal(t, poststatus.Hidden, commentRepo.comment.Status)
		require.NotNil(t, commentRepo.comment.ModeratedBy)
		assert.Equal(t, adminID, *commentRepo.comment.ModeratedBy)
		assert.Equal(t, 1, commentRepo.cascadeCalls)
		assert.Equal(t, poststatus.Hidden, commentRepo.cascadeStatus)
		assert.Equal(t, 1, postRepo.recountCalls)
		assert.True(t, pub.called)
	})

	t.Run("delete recounts comment_count", func(t *testing.T) {
		comment := &entities.PostComment{PostID: uuid.New(), Status: poststatus.Published}
		comment.ID = uuid.New()
		uc, commentRepo, postRepo, _ := newCommentAdminUC(comment)

		require.NoError(t, uc.DeleteComment(context.Background(), adminID, comment.ID))

		assert.Equal(t, poststatus.Deleted, commentRepo.comment.Status)
		assert.Equal(t, 1, postRepo.recountCalls)
	})

	t.Run("moderating an author-deleted comment is rejected", func(t *testing.T) {
		comment := &entities.PostComment{PostID: uuid.New(), Status: poststatus.Deleted}
		comment.ID = uuid.New()
		uc, commentRepo, postRepo, _ := newCommentAdminUC(comment)

		require.Error(t, uc.DeleteComment(context.Background(), adminID, comment.ID))

		assert.Equal(t, 0, commentRepo.updateCalls)
		assert.Equal(t, 0, postRepo.recountCalls)
	})
}

func TestRestoreComment(t *testing.T) {
	adminID := uuid.New()
	moderatorID := uuid.New()

	t.Run("restore admin-hidden comment recounts and cascades", func(t *testing.T) {
		comment := &entities.PostComment{PostID: uuid.New(), Status: poststatus.Hidden, ModeratedBy: &moderatorID}
		comment.ID = uuid.New()
		uc, commentRepo, postRepo, _ := newCommentAdminUC(comment)

		require.NoError(t, uc.RestoreComment(context.Background(), adminID, comment.ID))

		assert.Equal(t, poststatus.Published, commentRepo.comment.Status)
		assert.Nil(t, commentRepo.comment.ModeratedBy)
		assert.Equal(t, poststatus.Published, commentRepo.cascadeStatus)
		assert.Equal(t, 1, postRepo.recountCalls)
	})

	t.Run("restore author-deleted comment is rejected", func(t *testing.T) {
		comment := &entities.PostComment{PostID: uuid.New(), Status: poststatus.Deleted}
		comment.ID = uuid.New()
		uc, commentRepo, postRepo, _ := newCommentAdminUC(comment)

		require.Error(t, uc.RestoreComment(context.Background(), adminID, comment.ID))

		assert.Equal(t, 0, commentRepo.updateCalls)
		assert.Equal(t, 0, postRepo.recountCalls)
	})
}

func TestModeratePostRejectsAuthorDeleted(t *testing.T) {
	adminID := uuid.New()
	post := &entities.Post{Status: poststatus.Deleted}
	post.ID = uuid.New()

	t.Run("hide rejects author-deleted post", func(t *testing.T) {
		uc, repo, _ := newAdminUC(post)
		require.Error(t, uc.HidePost(context.Background(), adminID, post.ID))
		assert.Equal(t, 0, repo.moderateCalls)
	})

	t.Run("delete rejects author-deleted post", func(t *testing.T) {
		uc, repo, _ := newAdminUC(post)
		require.Error(t, uc.DeletePost(context.Background(), adminID, post.ID))
		assert.Equal(t, 0, repo.moderateCalls)
	})
}
