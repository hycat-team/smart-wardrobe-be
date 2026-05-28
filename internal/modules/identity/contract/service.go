package contract

import (
	"context"

	"github.com/google/uuid"
)

// IIdentityModuleContract exposes external identity lookups for other modules
type IIdentityModuleContract interface {
	GetUserByID(ctx context.Context, id uuid.UUID) (*PublicUserDTO, error)
}
