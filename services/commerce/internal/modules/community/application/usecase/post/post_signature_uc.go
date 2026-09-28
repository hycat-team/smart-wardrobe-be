package post

import (
	"context"
	"strings"

	communityerrors "smart-wardrobe-be/services/commerce/internal/modules/community/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/community/domain/constants/mediatype"
	shared_dto "smart-wardrobe-be/shared/application/dto"
)

func (uc *PostUseCase) GetUploadSignature(ctx context.Context, resourceType string) (*shared_dto.UploadSignatureResult, error) {
	rt := strings.ToLower(strings.TrimSpace(resourceType))
	if rt == "" {
		rt = string(mediatype.Image)
	}
	mt := mediatype.MediaType(rt)
	if !mt.IsValid() {
		return nil, communityerrors.ErrInvalidResourceType()
	}

	folder := uc.cfg.Cloudinary.PostFolder
	if folder == "" {
		folder = "smart_wardrobe/posts"
	}
	return uc.mediaService.GenerateUploadSignature(ctx, shared_dto.UploadSignatureParams{
		Folder:         folder,
		ResourceType:   rt,
		AllowedFormats: strings.Join(mt.AllowedExtensions(), ","),
	})
}
