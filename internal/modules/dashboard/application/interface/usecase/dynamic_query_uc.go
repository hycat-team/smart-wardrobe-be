package usecase

import (
	"context"

	"smart-wardrobe-be/internal/modules/dashboard/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"

	"github.com/google/uuid"
)

type IDynamicQueryUseCase interface {
	ExecuteDynamicQuery(ctx context.Context, userID uuid.UUID, userRole roleslug.RoleSlug, req *dto.DynamicQueryReq) (*dto.DynamicQueryRes, error)
}
