package post

import (
	"context"
	"strings"
	"testing"

	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/feedtype"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/mediatype"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/posttype"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/repositories"
	"smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sortOrderPtr(v int16) *int16 { return &v }

func testConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Community.MaxMediaPerPost = 10
	cfg.Community.MaxImageBytes = 10485760
	cfg.Community.MaxVideoBytes = 104857600
	cfg.Community.MaxVideoDurationSeconds = 60
	cfg.Community.MaxTitleLength = 150
	cfg.Community.MaxContentLength = 5000
	cfg.Community.MaxCommentLength = 1000
	return cfg
}

func TestValidatePostInput(t *testing.T) {
	uc := &PostUseCase{cfg: testConfig()}

	t.Run("rejects invalid post type", func(t *testing.T) {
		err := uc.validatePostInput(posttype.PostType("story"), nil, "hi", nil)
		require.Error(t, err)
	})

	t.Run("rejects empty body", func(t *testing.T) {
		err := uc.validatePostInput(posttype.Media, nil, "   ", nil)
		require.Error(t, err)
	})

	t.Run("allows media without content", func(t *testing.T) {
		media := []dto.PostMediaReq{{MediaType: "image", MediaURL: "https://x/y.jpg", SortOrder: sortOrderPtr(1)}}
		assert.NoError(t, uc.validatePostInput(posttype.Media, nil, "", media))
	})

	t.Run("allows any valid int16 sort order", func(t *testing.T) {
		media := []dto.PostMediaReq{
			{MediaType: "image", MediaURL: "https://x/0.jpg", SortOrder: sortOrderPtr(0)},
			{MediaType: "image", MediaURL: "https://x/neg.jpg", SortOrder: sortOrderPtr(-5)},
			{MediaType: "image", MediaURL: "https://x/max.jpg", SortOrder: sortOrderPtr(32767)},
		}
		assert.NoError(t, uc.validatePostInput(posttype.Media, nil, "x", media))
	})

	t.Run("rejects missing sort order", func(t *testing.T) {
		media := []dto.PostMediaReq{{MediaType: "image", MediaURL: "https://x/y.jpg"}}
		err := uc.validatePostInput(posttype.Media, nil, "x", media)
		require.Error(t, err)
	})

	t.Run("rejects duplicate sort order", func(t *testing.T) {
		media := []dto.PostMediaReq{
			{MediaType: "image", MediaURL: "https://x/1.jpg", SortOrder: sortOrderPtr(2)},
			{MediaType: "image", MediaURL: "https://x/2.jpg", SortOrder: sortOrderPtr(2)},
		}
		err := uc.validatePostInput(posttype.Media, nil, "x", media)
		require.Error(t, err)
	})

	t.Run("rejects too many media", func(t *testing.T) {
		media := make([]dto.PostMediaReq, 11)
		for i := range media {
			media[i] = dto.PostMediaReq{MediaType: "image", MediaURL: "https://x/y.jpg", SortOrder: sortOrderPtr(int16(i))}
		}
		err := uc.validatePostInput(posttype.Media, nil, "x", media)
		require.Error(t, err)
	})

	t.Run("rejects title over limit", func(t *testing.T) {
		long := strings.Repeat("a", 151)
		err := uc.validatePostInput(posttype.Media, &long, "x", nil)
		require.Error(t, err)
	})

	t.Run("rejects invalid media type", func(t *testing.T) {
		media := []dto.PostMediaReq{{MediaType: "gif", MediaURL: "https://x/y.gif"}}
		err := uc.validatePostInput(posttype.Media, nil, "x", media)
		require.Error(t, err)
	})
}

// feedCapturingPostRepo records the filter passed to ListFeed. It embeds the
// interface so only ListFeed needs a real implementation.
type feedCapturingPostRepo struct {
	repositories.IPostRepository
	captured repositories.PostFilter
}

func (f *feedCapturingPostRepo) ListFeed(_ context.Context, filter repositories.PostFilter) (*repositories.PostListResult, error) {
	f.captured = filter
	return &repositories.PostListResult{}, nil
}

func TestGetFeedFilter(t *testing.T) {
	viewer := uuid.New()

	t.Run("following passes the viewer as the only follow source", func(t *testing.T) {
		repo := &feedCapturingPostRepo{}
		uc := &PostUseCase{cfg: testConfig(), postRepo: repo}

		_, err := uc.GetFeed(context.Background(), &viewer, dto.GetFeedQueryReq{Type: "following"})
		require.NoError(t, err)
		assert.Equal(t, feedtype.Following, repo.captured.Type)
		require.NotNil(t, repo.captured.FollowingOf)
		assert.Equal(t, viewer, *repo.captured.FollowingOf)
	})

	t.Run("following without a viewer is unauthorized", func(t *testing.T) {
		uc := &PostUseCase{cfg: testConfig(), postRepo: &feedCapturingPostRepo{}}
		_, err := uc.GetFeed(context.Background(), nil, dto.GetFeedQueryReq{Type: "following"})
		require.Error(t, err)
	})

	t.Run("explore never carries a follow source", func(t *testing.T) {
		repo := &feedCapturingPostRepo{}
		uc := &PostUseCase{cfg: testConfig(), postRepo: repo}

		_, err := uc.GetFeed(context.Background(), &viewer, dto.GetFeedQueryReq{})
		require.NoError(t, err)
		assert.Equal(t, feedtype.Explore, repo.captured.Type)
		assert.Nil(t, repo.captured.FollowingOf)
	})
}

func TestBuildPostMediaPreservesOrder(t *testing.T) {
	postID := uuid.New()
	medias := buildPostMedia(postID, []dto.PostMediaReq{
		{MediaType: "image", MediaURL: "https://x/1.jpg", SortOrder: sortOrderPtr(2)},
		{MediaType: "video", MediaURL: "https://x/2.mp4", SortOrder: sortOrderPtr(0)},
	})
	require.Len(t, medias, 2)
	assert.Equal(t, postID, medias[0].PostID)
	assert.Equal(t, mediatype.Image, medias[0].MediaType)
	assert.Equal(t, int16(2), medias[0].SortOrder)
	assert.Equal(t, mediatype.Video, medias[1].MediaType)
	assert.NotEqual(t, uuid.Nil, medias[0].ID)
}

type fakeUpdatePostRepo struct {
	repositories.IPostRepository
	post          *entities.Post
	editableCalls int
	saveCalls     int
}

func (f *fakeUpdatePostRepo) GetByPublicID(context.Context, string) (*entities.Post, error) {
	return f.post, nil
}

func (f *fakeUpdatePostRepo) UpdateEditableFields(context.Context, uuid.UUID, *string, string, *uuid.UUID) error {
	f.editableCalls++
	return nil
}

func (f *fakeUpdatePostRepo) Update(context.Context, *entities.Post) error {
	f.saveCalls++
	return nil
}

type fakeMediaRepo struct {
	repositories.IPostMediaRepository
	deleteCalls int
}

func (f *fakeMediaRepo) DeleteByPostID(context.Context, uuid.UUID) error {
	f.deleteCalls++
	return nil
}

type fakeLikeRepoPost struct {
	repositories.IPostLikeRepository
}

func (f *fakeLikeRepoPost) Exists(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return false, nil
}

type fakePostUoW struct{}

func (fakePostUoW) Execute(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func TestUpdatePostDoesNotClobberCounters(t *testing.T) {
	userID := uuid.New()
	oldTitle := "old"
	post := &entities.Post{
		PostType:     posttype.Media,
		Title:        &oldTitle,
		Content:      "old content",
		LikeCount:    7,
		CommentCount: 3,
	}
	post.ID = uuid.New()
	post.UserID = userID
	post.PublicID = "post-public-id"

	postRepo := &fakeUpdatePostRepo{post: post}
	mediaRepo := &fakeMediaRepo{}
	uc := &PostUseCase{
		cfg:       testConfig(),
		uow:       fakePostUoW{},
		postRepo:  postRepo,
		mediaRepo: mediaRepo,
		likeRepo:  &fakeLikeRepoPost{},
	}

	newTitle := "new"
	_, err := uc.UpdatePost(context.Background(), userID, post.PublicID, dto.UpdatePostReq{
		Title:   &newTitle,
		Content: "new content",
	})

	require.NoError(t, err)
	assert.Equal(t, 1, postRepo.editableCalls)
	assert.Equal(t, 0, postRepo.saveCalls)
	assert.Equal(t, 1, mediaRepo.deleteCalls)
	assert.Equal(t, 7, post.LikeCount)
	assert.Equal(t, 3, post.CommentCount)
}
