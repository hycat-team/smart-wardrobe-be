package persistence

import (
	"context"
	"errors"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserWalletRepository struct {
	*shared_repos.GenericRepository[entities.UserWallet, uuid.UUID]
}

func NewUserWalletRepository(dbConn *gorm.DB) repositories.IUserWalletRepository {
	relations := []string{}
	return &UserWalletRepository{
		GenericRepository: shared_repos.NewGenericRepository[entities.UserWallet, uuid.UUID](dbConn, relations),
	}
}

func (r *UserWalletRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserWallet, error) {
	var wallet entities.UserWallet
	err := r.GetDB(ctx).Where("user_id = ?", userID).First(&wallet).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &wallet, nil
}

func (r *UserWalletRepository) GetByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserWallet, error) {
	var wallet entities.UserWallet
	err := r.GetDB(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).
		First(&wallet).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &wallet, nil
}
