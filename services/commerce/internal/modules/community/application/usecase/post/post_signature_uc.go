package post

import (
	"context"
	"strings"

	shared_dto "smart-wardrobe-be/shared/application/dto"
)

func (uc *PostUseCase) GetUploadSignature(ctx context.Context, resourceType string) (*shared_dto.UploadSignatureResult, error) {
	rt := strings.ToLower(strings.TrimSpace(resourceType))
	if rt != "video" {
		rt = "image"
	}
	folder := uc.cfg.Cloudinary.PostFolder
	if folder == "" {
		folder = "smart_wardrobe/posts"
	}
	return uc.mediaService.GenerateUploadSignature(ctx, shared_dto.UploadSignatureParams{
		Folder:       folder,
		ResourceType: rt,
	})
}
