package persistence

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/internal/shared/infrastructure/db"
)

type UserSubscriptionEventRepository struct{ db *gorm.DB }

func NewUserSubscriptionEventRepository(conn *gorm.DB) repositories.IUserSubscriptionEventRepository {
	return &UserSubscriptionEventRepository{db: conn}
}
func (r *UserSubscriptionEventRepository) current(ctx context.Context) *gorm.DB {
	if tx := db.GetTx(ctx); tx != nil {
		return tx
	}
	return r.db.WithContext(ctx)
}
func (r *UserSubscriptionEventRepository) Create(ctx context.Context, event *entities.UserSubscriptionEvent) error {
	return r.current(ctx).Create(event).Error
}
func (r *UserSubscriptionEventRepository) GetByKey(ctx context.Context, key string) (*entities.UserSubscriptionEvent, error) {
	var event entities.UserSubscriptionEvent
	err := r.current(ctx).Where("event_key = ?", key).First(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &event, err
}

type SubscriptionRenewalAttemptRepository struct{ db *gorm.DB }

func NewSubscriptionRenewalAttemptRepository(conn *gorm.DB) repositories.ISubscriptionRenewalAttemptRepository {
	return &SubscriptionRenewalAttemptRepository{db: conn}
}
func (r *SubscriptionRenewalAttemptRepository) current(ctx context.Context) *gorm.DB {
	if tx := db.GetTx(ctx); tx != nil {
		return tx
	}
	return r.db.WithContext(ctx)
}
func (r *SubscriptionRenewalAttemptRepository) Create(ctx context.Context, attempt *entities.SubscriptionRenewalAttempt) error {
	return r.current(ctx).Create(attempt).Error
}
func (r *SubscriptionRenewalAttemptRepository) GetByKey(ctx context.Context, key string) (*entities.SubscriptionRenewalAttempt, error) {
	var attempt entities.SubscriptionRenewalAttempt
	err := r.current(ctx).Where("renewal_attempt_key = ?", key).First(&attempt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &attempt, err
}
func (r *SubscriptionRenewalAttemptRepository) Update(ctx context.Context, attempt *entities.SubscriptionRenewalAttempt) error {
	return r.current(ctx).Save(attempt).Error
}
