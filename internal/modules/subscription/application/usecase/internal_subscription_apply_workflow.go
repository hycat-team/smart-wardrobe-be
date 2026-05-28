package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

// applySubscriptionPlan is an internal workflow to encapsulate the logic
// for applying or extending a subscription plan for a given user.
// It must be executed within a UnitOfWork transaction context (txCtx).
func applySubscriptionPlan(
	txCtx context.Context,
	userSubRepo repositories.IUserSubscriptionRepository,
	userID uuid.UUID,
	plan *entities.SubscriptionPlan,
	now time.Time,
) error {
	sub, err := userSubRepo.GetByUserIDWithLock(txCtx, userID)
	if err != nil {
		return errorcode.NewInternalError("Lỗi khi kiểm tra thông tin gói hội viên hiện tại")
	}

	var isNewSub bool
	if sub == nil {
		isNewSub = true
		sub = &entities.UserSubscription{
			UserID:    userID,
			CreatedAt: now,
		}
	}

	var expiresAt *time.Time

	if plan.DurationDays == nil {
		// Gói không giới hạn thời gian (ví dụ: hạng Free)
		expiresAt = nil
	} else {
		days := *plan.DurationDays
		var t time.Time
		// Kiểm tra phân hạng và tính toán thời gian hết hạn cho gói hội viên
		// Nếu trùng khớp gói đang hoạt động thì thời hạn được cộng dồn kéo dài
		// Nếu thay đổi sang gói khác thì thời hạn cũ sẽ bị hủy và tính mới từ thời điểm hiện tại
		if !isNewSub && sub.IsActive && sub.ExpiresAt != nil && sub.ExpiresAt.After(now) && sub.SubscriptionPlanID == plan.ID {
			t = sub.ExpiresAt.AddDate(0, 0, days)
		} else {
			t = now.AddDate(0, 0, days)
		}
		expiresAt = &t
	}

	sub.SubscriptionPlanID = plan.ID
	sub.ExpiresAt = expiresAt
	sub.IsActive = true
	sub.UpdatedAt = now

	if isNewSub {
		if err := userSubRepo.Create(txCtx, sub); err != nil {
			return errorcode.NewInternalError("Lỗi khi kích hoạt gói hội viên mới")
		}
	} else {
		if err := userSubRepo.Update(txCtx, sub); err != nil {
			return errorcode.NewInternalError("Lỗi khi cập nhật thời hạn gói hội viên")
		}
	}

	return nil
}
