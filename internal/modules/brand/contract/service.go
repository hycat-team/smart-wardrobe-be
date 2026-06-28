package contract

import (
	"context"

	"github.com/google/uuid"
)

type IBrandContract interface {
	CheckBrandFeatureAccess(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, featureCode string) (bool, error)
	ListEligibleBrandItemsForStyling(ctx context.Context, userID uuid.UUID, filter interface{}) (interface{}, error)
}
