package post

import (
	"context"
	"testing"

	shared_dto "smart-wardrobe-be/shared/application/dto"
	"smart-wardrobe-be/shared/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubMediaService struct {
	params shared_dto.UploadSignatureParams
}

func (s *stubMediaService) GenerateUploadSignature(_ context.Context, params shared_dto.UploadSignatureParams) (*shared_dto.UploadSignatureResult, error) {
	s.params = params
	return &shared_dto.UploadSignatureResult{
		Signature:      "sig",
		Timestamp:      1,
		ApiKey:         "key",
		Folder:         params.Folder,
		ResourceType:   params.ResourceType,
		AllowedFormats: params.AllowedFormats,
	}, nil
}

func (s *stubMediaService) DeleteImage(_ context.Context, _ string) error { return nil }

func signatureConfig() *config.Config {
	cfg := testConfig()
	cfg.Cloudinary.PostFolder = "closy/posts"
	return cfg
}

func TestGetUploadSignature(t *testing.T) {
	t.Run("defaults to image", func(t *testing.T) {
		stub := &stubMediaService{}
		uc := &PostUseCase{cfg: signatureConfig(), mediaService: stub}

		res, err := uc.GetUploadSignature(context.Background(), "")
		require.NoError(t, err)

		assert.Equal(t, "image", res.ResourceType)
		assert.Equal(t, "image", stub.params.ResourceType)
		assert.Equal(t, "jpg,jpeg,png,webp,avif,gif", res.AllowedFormats)
		assert.Equal(t, "jpg,jpeg,png,webp,avif,gif", stub.params.AllowedFormats)
		assert.Equal(t, "closy/posts", res.Folder)
	})

	t.Run("returns video formats", func(t *testing.T) {
		stub := &stubMediaService{}
		uc := &PostUseCase{cfg: signatureConfig(), mediaService: stub}

		res, err := uc.GetUploadSignature(context.Background(), "video")
		require.NoError(t, err)

		assert.Equal(t, "video", res.ResourceType)
		assert.Equal(t, "mp4,webm,mov,m4v,mkv,avi", res.AllowedFormats)
	})

	// public_id must stay out of the post media signature: a client that echoes
	// it back would break the signature, and signing one public_id would make
	// every media item overwrite the same asset.
	t.Run("never signs a public id", func(t *testing.T) {
		stub := &stubMediaService{}
		uc := &PostUseCase{cfg: signatureConfig(), mediaService: stub}

		res, err := uc.GetUploadSignature(context.Background(), "image")
		require.NoError(t, err)

		assert.Empty(t, stub.params.PublicID)
		assert.Empty(t, res.PublicID)
	})

	t.Run("normalizes case and whitespace", func(t *testing.T) {
		stub := &stubMediaService{}
		uc := &PostUseCase{cfg: signatureConfig(), mediaService: stub}

		res, err := uc.GetUploadSignature(context.Background(), "  VIDEO  ")
		require.NoError(t, err)
		assert.Equal(t, "video", res.ResourceType)

		res, err = uc.GetUploadSignature(context.Background(), "IMAGE")
		require.NoError(t, err)
		assert.Equal(t, "image", res.ResourceType)
	})

	t.Run("rejects unknown resource type", func(t *testing.T) {
		stub := &stubMediaService{}
		uc := &PostUseCase{cfg: signatureConfig(), mediaService: stub}

		for _, rt := range []string{"audio", "gif", "raw", "anything"} {
			_, err := uc.GetUploadSignature(context.Background(), rt)
			require.Error(t, err, rt)
		}
	})

	t.Run("falls back to default folder", func(t *testing.T) {
		stub := &stubMediaService{}
		cfg := testConfig()
		cfg.Cloudinary.PostFolder = ""
		uc := &PostUseCase{cfg: cfg, mediaService: stub}

		res, err := uc.GetUploadSignature(context.Background(), "image")
		require.NoError(t, err)
		assert.Equal(t, "smart_wardrobe/posts", res.Folder)
	})
}
