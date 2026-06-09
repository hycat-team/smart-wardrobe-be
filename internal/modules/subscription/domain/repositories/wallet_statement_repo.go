package repositories

import (
	"context"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

type IWalletStatementRepository interface {
	shared_repos.IGenericRepository[entities.WalletStatement, uuid.UUID]
	GetByUserID(ctx context.Context, userID uuid.UUID, pagination shared_dto.PaginationQuery) ([]*entities.WalletStatement, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}
