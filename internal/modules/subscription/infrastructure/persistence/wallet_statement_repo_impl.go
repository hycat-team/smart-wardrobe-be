package persistence

import (
	"context"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WalletStatementRepository struct {
	*shared_repos.GenericRepository[entities.WalletStatement, uuid.UUID]
}

func NewWalletStatementRepository(dbConn *gorm.DB) repositories.IWalletStatementRepository {
	return &WalletStatementRepository{
		GenericRepository: shared_repos.NewGenericRepository[entities.WalletStatement, uuid.UUID](dbConn),
	}
}

func (r *WalletStatementRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.WalletStatement, error) {
	var statements []*entities.WalletStatement
	err := r.GetDB(ctx).Where("user_id = ?", userID).Order("created_at desc").Find(&statements).Error
	if err != nil {
		return nil, err
	}
	return statements, nil
}
