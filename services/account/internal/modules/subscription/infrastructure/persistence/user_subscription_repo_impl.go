package persistence

import (
	"context"
	"errors"
	"time"

	"smart-wardrobe-be/services/account/dto/subscription"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/plankind"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/constants/usersubscriptionstatus"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	shared_repos "smart-wardrobe-be/shared/infrastructure/repositories"

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

	err := query.Where("user_id = ?", userID).First(&sub).Error
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
		Where("user_id IN ?", userIDs).
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

	err := query.Where("current_plan_kind = ? AND expires_at <= ?", plankind.Finite, now).Find(&expiredSubs).Error
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
		Where("current_plan_kind = ? AND expires_at IS NOT NULL AND expires_at <= ?", plankind.Finite, now)

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
		Where("user_id = ? AND current_plan_kind = ? AND expires_at IS NOT NULL AND expires_at <= ?", userID, plankind.Finite, now)

	err := query.First(&sub).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &sub, nil
}

func (r *UserSubscriptionRepository) GetSubscriptionsExpiringWithin(ctx context.Context, from, to time.Time) ([]*entities.UserSubscription, error) {
	var subs []*entities.UserSubscription
	query := r.GetQueryWithPreload(ctx)
	err := query.Where("current_plan_kind = ? AND expires_at IS NOT NULL AND expires_at > ? AND expires_at <= ?", plankind.Finite, from, to).Find(&subs).Error
	if err != nil {
		return nil, err
	}
	return subs, nil
}

func (r *UserSubscriptionRepository) ProvisionDefault(ctx context.Context, sub *entities.UserSubscription) error {
	return r.GetDB(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}}, DoNothing: true}).Create(sub).Error
}

func (r *UserSubscriptionRepository) BulkCreate(ctx context.Context, subs []*entities.UserSubscription) error {
	if len(subs) == 0 {
		return nil
	}
	return r.GetDB(ctx).Create(&subs).Error
}

func (r *UserSubscriptionRepository) GetDistributionByPlanCode(ctx context.Context) ([]dto.SubscriptionDistributionDTO, error) {
	var rows []dto.SubscriptionDistributionDTO
	err := r.GetDB(ctx).
		Model(&entities.UserSubscription{}).
		Select("current_plan_code AS plan_code, COUNT(*) AS user_count").
		Group("current_plan_code").
		Order("user_count DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *UserSubscriptionRepository) ListAdmin(ctx context.Context, filter repositories.AdminUserSubscriptionFilter, page, limit int) ([]*entities.UserSubscription, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	db := r.GetDB(ctx).Model(&entities.UserSubscription{})

	if filter.PlanCode != nil && *filter.PlanCode != "" {
		db = db.Where("current_plan_code = ?", *filter.PlanCode)
	}
	if filter.UserID != nil {
		db = db.Where("user_id = ?", *filter.UserID)
	}
	if filter.Status != nil {
		now := time.Now()
		switch *filter.Status {
		case usersubscriptionstatus.Active:
			db = db.Where("expires_at IS NULL OR expires_at > ?", now)
		case usersubscriptionstatus.Expired:
			db = db.Where("expires_at IS NOT NULL AND expires_at <= ?", now)
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var subs []*entities.UserSubscription
	err := r.GetQueryWithPreload(ctx).
		Where(db).
		Order("started_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&subs).Error
	if err != nil {
		return nil, 0, err
	}

	return subs, total, nil
}
