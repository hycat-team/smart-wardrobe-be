package post

import (
	"strings"
	"testing"

	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/mediatype"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/posttype"
	"smart-wardrobe-be/shared/config"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sortOrderPtr(v int16) *int16 { return &v }

func testConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Community.MaxMediaPerPost = 10
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
