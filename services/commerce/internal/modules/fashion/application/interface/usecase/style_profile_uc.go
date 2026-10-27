package usecase

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"

	"github.com/google/uuid"
)

// IStyleProfileUseCase exposes the style profile of the authenticated user
// (feature 033, US1).
type IStyleProfileUseCase interface {
	// GetProfile returns the stored profile or the defaults of a user who has
	// never saved one. It never creates a row and never requires entitlement.
	GetProfile(ctx context.Context, userID uuid.UUID) (*dto.StyleProfileRes, error)
	// UpdateProfile applies a partial update, normalizes the lists, refreshes
	// the style embedding (fail-soft) and returns the stored profile.
	UpdateProfile(ctx context.Context, userID uuid.UUID, input dto.UpdateStyleProfileReq) (*dto.StyleProfileRes, error)
	// DeleteProfile removes the profile row. It is idempotent.
	DeleteProfile(ctx context.Context, userID uuid.UUID) error
}
