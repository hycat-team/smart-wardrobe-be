package persistence

import (
	"context"
	"errors"

	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserContactRepository struct {
	db *gorm.DB
}

func NewUserContactRepository(db *gorm.DB) repositories.IUserContactRepository {
	return &UserContactRepository{db: db}
}

func (r *UserContactRepository) GetEmailByUserID(ctx context.Context, userID uuid.UUID) (string, error) {
	var email string
	err := r.db.WithContext(ctx).
		Model(&struct{}{}).
		Table("users").
		Select("email").
		Where("id = ?", userID).
		Scan(&email).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return email, nil
}
