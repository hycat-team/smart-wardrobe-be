package persistence

import (
	"context"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WalletStatementRepository struct {
	*shared_repos.GenericRepository[entities.WalletStatement, uuid.UUID]
}

func NewWalletStatementRepository(dbConn *gorm.DB) repositories.IWalletStatementRepository {
	relations := []string{}
	return &WalletStatementRepository{
		GenericRepository: shared_repos.NewGenericRepository[entities.WalletStatement, uuid.UUID](dbConn, relations),
	}
}

func (r *WalletStatementRepository) GetByUserID(ctx context.Context, userID uuid.UUID, pagination shared_dto.PaginationQuery) ([]*entities.WalletStatement, error) {
	var statements []*entities.WalletStatement
	query := r.GetDB(ctx).Model(&entities.WalletStatement{}).Where("user_id = ?", userID)
	db := shared_repos.ApplyPagination(query, pagination)
	err := db.Order("created_at desc").Find(&statements).Error
	if err != nil {
		return nil, err
	}
	return statements, nil
}

func (r *WalletStatementRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.GetDB(ctx).Model(&entities.WalletStatement{}).Where("user_id = ?", userID).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}
