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
	relations := []string{"SubscriptionPlan"}
	return &UserSubscriptionRepository{
		GenericRepository: shared_repos.NewGenericRepository[entities.UserSubscription, uuid.UUID](dbConn, relations),
	}
}

// GetByUserID retrieves active subscription for a specific user
func (r *UserSubscriptionRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	var sub entities.UserSubscription
	query := r.GetQueryWithPreload(ctx)

	err := query.Where("user_id = ? AND is_active = ?", userID, true).First(&sub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func (r *UserSubscriptionRepository) GetByUserIDs(ctx context.Context, userIDs []uuid.UUID) ([]*entities.UserSubscription, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	var subs []*entities.UserSubscription
	err := r.GetQueryWithPreload(ctx).
		Where("user_id IN ? AND is_active = ?", userIDs, true).
		Find(&subs).Error
	if err != nil {
		return nil, err
	}
	return subs, nil
}

func (r *UserSubscriptionRepository) GetByUserIDWithLock(ctx context.Context, userID uuid.UUID) (*entities.UserSubscription, error) {
	var sub entities.UserSubscription
	query := r.GetQueryWithPreload(ctx).Clauses(clause.Locking{Strength: "UPDATE"})

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
	query := r.GetQueryWithPreload(ctx)

	err := query.Where("is_active = ? AND expires_at <= ?", true, now).Find(&expiredSubs).Error
	if err != nil {
		return nil, err
	}
	return expiredSubs, nil
}

func (r *UserSubscriptionRepository) GetActiveExpiredSubscriptionsBatch(ctx context.Context, now time.Time, lastUserID uuid.UUID, lastExpiresAt time.Time, limit int) ([]*entities.UserSubscription, error) {
	if limit <= 0 {
		limit = 100
	}
	var expiredSubs []*entities.UserSubscription
	query := r.GetDB(ctx).Model(&entities.UserSubscription{}).
		Select("user_id", "expires_at").
		Where("is_active = ? AND expires_at IS NOT NULL AND expires_at <= ?", true, now)

	if !lastExpiresAt.IsZero() {
		query = query.Where("expires_at > ? OR (expires_at = ? AND user_id > ?)", lastExpiresAt, lastExpiresAt, lastUserID)
	}

	err := query.Order("expires_at ASC, user_id ASC").Limit(limit).Find(&expiredSubs).Error
	if err != nil {
		return nil, err
	}
	return expiredSubs, nil
}

func (r *UserSubscriptionRepository) GetActiveExpiredSubscriptionByUserIDWithLock(ctx context.Context, userID uuid.UUID, now time.Time) (*entities.UserSubscription, error) {
	var sub entities.UserSubscription
	query := r.GetQueryWithPreload(ctx).
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("user_id = ? AND is_active = ? AND expires_at IS NOT NULL AND expires_at <= ?", userID, true, now)

	err := query.First(&sub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func (r *UserSubscriptionRepository) BulkCreate(ctx context.Context, subs []*entities.UserSubscription) error {
	if len(subs) == 0 {
		return nil
	}
	return r.GetDB(ctx).Create(&subs).Error
}
