package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/infrastructure/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// UserSubscriptionRepository implements database actions for user subscriptions
type UserSubscriptionRepository struct {
	*shared_repos.GenericRepository[entities.UserSubscription, uuid.UUID]
}

// NewUserSubscriptionRepository creates a new instance of subscription repository
func NewUserSubscriptionRepository(dbConn *gorm.DB) repositories.IUserSubscriptionRepository {
	return &UserSubscriptionRepository{
		GenericRepository: shared_repos.NewGenericRepository[entities.UserSubscription, uuid.UUID](dbConn),
	}
}

// GetPreloadRelations returns relation paths to preload by default
func (r *UserSubscriptionRepository) GetPreloadRelations() []string {
	return []string{"SubscriptionPlan"}
}

// GetByUserID retrieves active subscription for a specific user
func (r *UserSubscriptionRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	var sub entities.UserSubscription
	query := r.GetDB(ctx)

	for _, relation := range r.GetPreloadRelations() {
		query = query.Preload(relation)
	}

	err := query.Where("user_id = ?", userID).First(&sub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func (r *UserSubscriptionRepository) GetByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	var sub entities.UserSubscription
	query := r.GetDB(ctx).Clauses(clause.Locking{Strength: "UPDATE"})

	for _, relation := range r.GetPreloadRelations() {
		query = query.Preload(relation)
	}

	err := query.Where("user_id = ?", userID).First(&sub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func (r *UserSubscriptionRepository) GetActiveExpiredSubscriptions(ctx context.Context, now time.Time) ([]*entities.UserSubscription, error) {
	var expiredSubs []*entities.UserSubscription
	query := r.GetDB(ctx)

	for _, relation := range r.GetPreloadRelations() {
		query = query.Preload(relation)
	}

	err := query.Where("is_active = ? AND expires_at <= ?", true, now).Find(&expiredSubs).Error
	if err != nil {
		return nil, err
	}
	return expiredSubs, nil
}
