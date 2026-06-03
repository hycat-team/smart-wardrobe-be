package media

import (
	"context"
	"smart-wardrobe-be/internal/shared/application/dto"
)

type IMediaService interface {
	GenerateUploadSignature(ctx context.Context, params dto.UploadSignatureParams) (*dto.UploadSignatureResult, error)
	DeleteImage(ctx context.Context, publicID string) error
}
