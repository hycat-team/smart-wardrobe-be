package post

import (
	"testing"

	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/mediatype"
	"smart-wardrobe-be/shared/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mediaConfig(maxImageBytes, maxVideoBytes int64, maxDuration int) *config.Config {
	cfg := testConfig()
	cfg.Community.MaxImageBytes = maxImageBytes
	cfg.Community.MaxVideoBytes = maxVideoBytes
	cfg.Community.MaxVideoDurationSeconds = maxDuration
	return cfg
}

func TestValidateMediaExtension(t *testing.T) {
	t.Run("accepts every allowed image extension", func(t *testing.T) {
		for _, ext := range mediatype.Image.AllowedExtensions() {
			assert.NoError(t, validateMediaExtension("https://res.cloudinary.com/x/image/upload/v1/a."+ext, mediatype.Image), ext)
		}
	})

	t.Run("accepts every allowed video extension", func(t *testing.T) {
		for _, ext := range mediatype.Video.AllowedExtensions() {
			assert.NoError(t, validateMediaExtension("https://res.cloudinary.com/x/video/upload/v1/a."+ext, mediatype.Video), ext)
		}
	})

	t.Run("rejects extension belonging to the other media type", func(t *testing.T) {
		require.Error(t, validateMediaExtension("https://x/a.mp4", mediatype.Image))
		require.Error(t, validateMediaExtension("https://x/a.jpg", mediatype.Video))
	})

	t.Run("rejects unknown extension", func(t *testing.T) {
		require.Error(t, validateMediaExtension("https://x/a.svg", mediatype.Image))
		require.Error(t, validateMediaExtension("https://x/a.heic", mediatype.Image))
		require.Error(t, validateMediaExtension("https://x/a.bmp", mediatype.Image))
		require.Error(t, validateMediaExtension("https://x/a.pdf", mediatype.Video))
	})

	t.Run("accepts url without extension", func(t *testing.T) {
		assert.NoError(t, validateMediaExtension("https://res.cloudinary.com/x/image/upload/v123/abc123", mediatype.Image))
		assert.NoError(t, validateMediaExtension("https://res.cloudinary.com/x/image/upload/f_auto/v123/abc123", mediatype.Image))
	})

	t.Run("is case insensitive", func(t *testing.T) {
		assert.NoError(t, validateMediaExtension("https://x/A.JPG", mediatype.Image))
		assert.NoError(t, validateMediaExtension("https://x/A.MP4", mediatype.Video))
	})

	t.Run("ignores query string and fragment", func(t *testing.T) {
		assert.NoError(t, validateMediaExtension("https://x/a.webp?w=100&h=200", mediatype.Image))
		require.Error(t, validateMediaExtension("https://x/a.svg?w=100", mediatype.Image))
		assert.NoError(t, validateMediaExtension("https://x/a.png#fragment", mediatype.Image))
	})

	t.Run("reads extension after transformation segments", func(t *testing.T) {
		assert.NoError(t, validateMediaExtension("https://res.cloudinary.com/x/image/upload/w_300,h_200,c_fill/v1/a.webp", mediatype.Image))
	})

	t.Run("rejects malformed url", func(t *testing.T) {
		require.Error(t, validateMediaExtension("://not a url", mediatype.Image))
	})
}

func TestValidateMediaSize(t *testing.T) {
	image := int64(10485760)
	video := int64(104857600)

	t.Run("skips when size not declared", func(t *testing.T) {
		assert.NoError(t, validateMediaSize(nil, mediatype.Image, image, video))
		assert.NoError(t, validateMediaSize(nil, mediatype.Video, image, video))
	})

	t.Run("accepts size at limit", func(t *testing.T) {
		assert.NoError(t, validateMediaSize(&image, mediatype.Image, image, video))
		assert.NoError(t, validateMediaSize(&video, mediatype.Video, image, video))
	})

	t.Run("rejects size over limit", func(t *testing.T) {
		over := image + 1
		require.Error(t, validateMediaSize(&over, mediatype.Image, image, video))

		overVideo := video + 1
		require.Error(t, validateMediaSize(&overVideo, mediatype.Video, image, video))
	})

	t.Run("image uses the image limit even when video limit is larger", func(t *testing.T) {
		oversizedForImage := video
		require.Error(t, validateMediaSize(&oversizedForImage, mediatype.Image, image, video))
	})

	t.Run("non-positive limit disables check", func(t *testing.T) {
		big := int64(1) << 40
		assert.NoError(t, validateMediaSize(&big, mediatype.Image, 0, 0))
		assert.NoError(t, validateMediaSize(&big, mediatype.Video, image, 0))
	})
}

func TestValidateMediaDuration(t *testing.T) {
	t.Run("skips for image", func(t *testing.T) {
		d := 9999.0
		assert.NoError(t, validateMediaDuration(mediatype.Image, &d, 60))
	})

	t.Run("skips when duration not declared", func(t *testing.T) {
		assert.NoError(t, validateMediaDuration(mediatype.Video, nil, 60))
	})

	t.Run("accepts duration at limit", func(t *testing.T) {
		d := 60.0
		assert.NoError(t, validateMediaDuration(mediatype.Video, &d, 60))
	})

	t.Run("rejects duration over limit", func(t *testing.T) {
		d := 61.0
		require.Error(t, validateMediaDuration(mediatype.Video, &d, 60))
	})

	t.Run("accepts fractional duration reported by cloudinary", func(t *testing.T) {
		d := 3.041267
		assert.NoError(t, validateMediaDuration(mediatype.Video, &d, 60))
	})

	t.Run("rejects fraction above limit", func(t *testing.T) {
		d := 60.5
		require.Error(t, validateMediaDuration(mediatype.Video, &d, 60))
	})

	t.Run("non-positive limit disables check", func(t *testing.T) {
		d := 100000.0
		assert.NoError(t, validateMediaDuration(mediatype.Video, &d, 0))
	})
}

func TestValidateMediaItems(t *testing.T) {
	t.Run("rejects invalid media type", func(t *testing.T) {
		uc := &PostUseCase{cfg: testConfig()}
		media := []dto.PostMediaReq{{MediaType: "gif", MediaURL: "https://x/a.jpg", SortOrder: sortOrderPtr(0)}}
		require.Error(t, uc.validateMediaItems(media))
	})

	t.Run("validates size and duration together", func(t *testing.T) {
		uc := &PostUseCase{cfg: testConfig()}
		bytes := int64(200 * 1024 * 1024)
		duration := 120.0
		media := []dto.PostMediaReq{{
			MediaType:       "video",
			MediaURL:        "https://x/a.mp4",
			SortOrder:       sortOrderPtr(0),
			Bytes:           &bytes,
			DurationSeconds: &duration,
		}}
		require.Error(t, uc.validateMediaItems(media))
	})

	t.Run("non-positive limits skip size and duration", func(t *testing.T) {
		uc := &PostUseCase{cfg: mediaConfig(0, 0, 0)}
		bytes := int64(1) << 40
		duration := 100000.0
		media := []dto.PostMediaReq{{
			MediaType:       "video",
			MediaURL:        "https://x/a.mp4",
			SortOrder:       sortOrderPtr(0),
			Bytes:           &bytes,
			DurationSeconds: &duration,
		}}
		assert.NoError(t, uc.validateMediaItems(media))
	})

	t.Run("still enforces extension when limits disabled", func(t *testing.T) {
		uc := &PostUseCase{cfg: mediaConfig(0, 0, 0)}
		media := []dto.PostMediaReq{{MediaType: "image", MediaURL: "https://x/a.svg", SortOrder: sortOrderPtr(0)}}
		require.Error(t, uc.validateMediaItems(media))
	})

	t.Run("enforces ordering", func(t *testing.T) {
		uc := &PostUseCase{cfg: testConfig()}
		require.Error(t, uc.validateMediaItems([]dto.PostMediaReq{
			{MediaType: "image", MediaURL: "https://x/1.jpg", SortOrder: sortOrderPtr(1)},
			{MediaType: "image", MediaURL: "https://x/2.jpg", SortOrder: sortOrderPtr(1)},
		}))
		require.Error(t, uc.validateMediaItems([]dto.PostMediaReq{
			{MediaType: "image", MediaURL: "https://x/1.jpg"},
		}))
	})
}
