package persistence

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"smart-wardrobe-be/internal/modules/aiexecutor/domain/constants/aipolicygrantstatus"
	"smart-wardrobe-be/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
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
	db := r.current(ctx)
	if err := db.Create(event).Error; err != nil {
		return err
	}
	return r.projectAIPolicyGrant(db, event)
}

func (r *UserSubscriptionEventRepository) projectAIPolicyGrant(db *gorm.DB, event *entities.UserSubscriptionEvent) error {
	var sub entities.UserSubscription
	if err := db.Preload("SubscriptionPlan").First(&sub, "user_id=?", event.UserID).Error; err != nil {
		return err
	}
	plan := sub.SubscriptionPlan
	if plan == nil {
		return errors.New("subscription plan AI cost policy missing")
	}
	start := event.EffectiveAt
	end := sub.ExpiresAt
	status := aipolicygrantstatus.Active
	if event.EventType == "EXTENDED" && plan.DurationDays != nil && sub.ExpiresAt != nil {
		start = sub.ExpiresAt.AddDate(0, 0, -*plan.DurationDays)
		if start.After(event.OccurredAt) {
			status = aipolicygrantstatus.Future
		}
	}
	if event.EventType != "EXTENDED" {
		if err := db.Exec("DELETE FROM ai_policy_grants WHERE user_id=? AND status=?", event.UserID, string(aipolicygrantstatus.Future)).Error; err != nil {
			return err
		}
		if err := db.Exec("UPDATE ai_policy_grants SET status=?, effective_to=?, updated_at=? WHERE user_id=? AND status=? AND effective_from < ?", string(aipolicygrantstatus.Closed), event.EffectiveAt, event.OccurredAt, event.UserID, string(aipolicygrantstatus.Active), event.EffectiveAt).Error; err != nil {
			return err
		}
	} else {
		if err := db.Exec("UPDATE ai_policy_grants SET effective_to=?, updated_at=? WHERE user_id=? AND effective_from < ? AND (effective_to IS NULL OR effective_to > ?)", start, event.OccurredAt, event.UserID, start, start).Error; err != nil {
			return err
		}
	}
	var policyID uuid.UUID
	if plan.AICostPolicyID != uuid.Nil {
		policyID = plan.AICostPolicyID
	}
	return db.Exec(`INSERT INTO ai_policy_grants (id, created_at, updated_at, user_id, policy_id, plan_id, plan_code, tier_rank, policy_snapshot, effective_from, effective_to, status, source_event_id, source_deposit_transaction_id)
		VALUES (gen_random_uuid(), now(), now(), ?, ?, ?, ?, ?, '{}'::jsonb, ?, ?, ?, ?, ?)`,
		event.UserID, policyID, plan.ID, plan.Slug, plan.TierRank, start, end, string(status), event.ID, event.SourceDepositTransactionID).Error
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
