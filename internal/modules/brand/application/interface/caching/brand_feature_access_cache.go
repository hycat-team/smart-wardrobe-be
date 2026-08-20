package caching

import (
	"context"

	"github.com/google/uuid"
)

type IBrandFeatureAccessCache interface {
	Get(ctx context.Context, userID, brandID uuid.UUID) ([]string, error)
	Set(ctx context.Context, userID, brandID uuid.UUID, featureCodes []string) error
	Invalidate(ctx context.Context, userID, brandID uuid.UUID) error
	InvalidateByBrand(ctx context.Context, brandID uuid.UUID, userIDs []uuid.UUID) error
}
