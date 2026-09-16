package repositories

import (
	"context"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
)

type IWalletStatementRepository interface {
	shared_repos.IGenericRepository[entities.WalletStatement, uuid.UUID]
	GetByUserID(ctx context.Context, userID uuid.UUID, pagination shared_dto.PaginationQuery) ([]*entities.WalletStatement, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}
