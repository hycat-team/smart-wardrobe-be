package repositories

import (
	"context"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

// IUserDailyQuotaRepository handles database operations for user daily quotas
type IUserDailyQuotaRepository interface {
	repositories.IGenericRepository[entities.UserDailyQuota, uuid.UUID]
	GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error)
	GetByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserDailyQuota, error)
	// CreateIfAbsent writes the row unless the account already has one, and
	// reports whether it inserted (024 T148).
	//
	// It exists because `user_id` is the primary key and the provisioning path is
	// reached from several request handlers at once. Two concurrent first reads of a
	// brand-new account both see no row and both insert; the second used to fail
	// with a unique violation, which the caller surfaced as HTTP 500 for an account
	// that had in fact just been created correctly.
	//
	// The caller re-reads after a non-insert, so both racers observe the same
	// quota. The contract is one row per account; the insert is what enforces it,
	// not a check that runs before it.
	CreateIfAbsent(ctx context.Context, quota *entities.UserDailyQuota) (inserted bool, err error)
}
