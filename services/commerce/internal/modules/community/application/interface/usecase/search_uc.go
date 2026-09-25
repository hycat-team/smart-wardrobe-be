package usecase

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"

	"github.com/google/uuid"
)

type ISearchUseCase interface {
	Search(ctx context.Context, query dto.SearchQueryReq, viewerUserID *uuid.UUID) (*dto.SearchRes, error)
}
