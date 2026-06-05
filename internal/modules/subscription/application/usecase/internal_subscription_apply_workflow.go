package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

// applySubscriptionPlan is an internal workflow to encapsulate the logic
// for applying or extending a subscription plan for a given user.
// It must be executed within a UnitOfWork transaction context (txCtx) to ensure atomicity.
func applySubscriptionPlan(
	txCtx context.Context,
	userSubRepo repositories.IUserSubscriptionRepository,
	userID uuid.UUID,
	plan *entities.SubscriptionPlan,
	now time.Time,
) error {
	// Retrieve the user's current subscription using a database row lock (FOR UPDATE)
	// to prevent race conditions when multiple operations attempt to modify the same subscription concurrently.
	sub, err := userSubRepo.GetByUserIDWithLock(txCtx, userID)
	if err != nil {
		return apperror.NewInternalError("Lỗi khi kiểm tra thông tin gói hội viên hiện tại")
	}

	// Check if this is the first time the user gets a subscription.
	// If no subscription record exists, mark it as new and initialize a new entity structure.
	var isNewSub bool
	if sub == nil {
		isNewSub = true
		sub = &entities.UserSubscription{
			UserID:    userID,
			CreatedAt: now,
		}
	}

	var expiresAt *time.Time

	// Handle the duration logic based on the target subscription plan configuration.
	if plan.DurationDays == nil {
		// If DurationDays is nil, it represents an unlimited plan (e.g., the default Free Tier).
		// Expiration date is set to nil (infinite duration).
		expiresAt = nil
	} else {
		days := *plan.DurationDays
		var t time.Time
		// Calculate the expiration time.
		// - If the user is currently subscribed to the SAME plan, and that subscription is still active,
		//   we extend the existing expiration date by adding the new plan's duration (stacking/accumulating).
		// - If the user is shifting to a DIFFERENT plan, or their previous subscription is inactive/expired,
		//   we discard the previous expiration and start a fresh duration from the current timestamp (now).
		if !isNewSub && sub.IsActive && sub.ExpiresAt != nil && sub.ExpiresAt.After(now) && sub.SubscriptionPlanID == plan.ID {
			t = sub.ExpiresAt.AddDate(0, 0, days)
		} else {
			t = now.AddDate(0, 0, days)
		}
		expiresAt = &t
	}

	// Update the subscription state with the plan details, calculated expiration, and activation flag.
	sub.SubscriptionPlanID = plan.ID
	sub.ExpiresAt = expiresAt
	sub.IsActive = true
	sub.UpdatedAt = now

	// Persist the changes to the database (Create for new, Update for existing).
	if isNewSub {
		if err := userSubRepo.Create(txCtx, sub); err != nil {
			return apperror.NewInternalError("Lỗi khi kích hoạt gói hội viên mới")
		}
	} else {
		if err := userSubRepo.Update(txCtx, sub); err != nil {
			return apperror.NewInternalError("Lỗi khi cập nhật thời hạn gói hội viên")
		}
	}

	return nil
}

