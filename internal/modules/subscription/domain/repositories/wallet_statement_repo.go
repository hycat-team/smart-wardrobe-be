package repositories

import (
	"context"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type IWalletStatementRepository interface {
	shared_repos.IGenericRepository[entities.WalletStatement, uuid.UUID]
	GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.WalletStatement, error)
}
