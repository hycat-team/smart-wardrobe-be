package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/internal/shared/domain/constants/subscription/aipolicygrantstatus"
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
	if err := db.Preload("SubscriptionPlan.AICostPolicy.Operations").First(&sub, "user_id=?", event.UserID).Error; err != nil {
		return err
	}
	plan := sub.SubscriptionPlan
	if plan == nil || plan.AICostPolicy == nil {
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
		if err := db.Where("user_id=? AND status=?", event.UserID, string(aipolicygrantstatus.Future)).Delete(&entities.UserAIPolicyGrant{}).Error; err != nil {
			return err
		}
		if err := db.Model(&entities.UserAIPolicyGrant{}).Where("user_id=? AND status=? AND effective_from < ?", event.UserID, string(aipolicygrantstatus.Active), event.EffectiveAt).Updates(map[string]any{"status": aipolicygrantstatus.Closed, "effective_to": event.EffectiveAt, "updated_at": event.OccurredAt}).Error; err != nil {
			return err
		}
	} else {
		if err := db.Model(&entities.UserAIPolicyGrant{}).Where("user_id=? AND effective_from < ? AND (effective_to IS NULL OR effective_to > ?)", event.UserID, start, start).Updates(map[string]any{"effective_to": start, "updated_at": event.OccurredAt}).Error; err != nil {
			return err
		}
	}
	snapshot, _ := json.Marshal(plan.AICostPolicy)
	grant := entities.UserAIPolicyGrant{UserID: event.UserID, PolicyID: plan.AICostPolicyID, PlanID: plan.ID, PlanCode: plan.Slug, TierRank: plan.TierRank, PolicySnapshot: entities.JSONDocument(snapshot), EffectiveFrom: start, EffectiveTo: end, Status: status, SourceEventID: &event.ID, SourceDepositTransactionID: event.SourceDepositTransactionID}
	return db.Create(&grant).Error
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
