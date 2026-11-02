package repositories

import (
	"context"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

type IUserWalletRepository interface {
	shared_repos.IGenericRepository[entities.UserWallet, uuid.UUID]
	GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserWallet, error)
	GetByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserWallet, error)
	// GetByUserIDs exists because the generic GetByIDs filters on `id`, and a
	// wallet's primary key column is `user_id` — the generic query would fail at
	// runtime rather than compile, so the batch lookup needs its own method.
	GetByUserIDs(ctx context.Context, userIDs []uuid.UUID) ([]*entities.UserWallet, error)
}
