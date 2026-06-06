package usecase

import (
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/internal/shared/domain/constants/userstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

const contactSupportMessage = "Tài khoản đã bị khoá hoặc vô hiệu hoá. Vui lòng liên hệ CSKH."

func ensureUserEligibleForLogin(user *entities.User) error {
	if user == nil {
		return apperror.NewBadRequest("Sai tài khoản hoặc mật khẩu.")
	}

	if user.IsDeleted || user.Status != userstatus.Active {
		return apperror.NewForbidden(contactSupportMessage)
	}

	return nil
}

func ensureUserEligibleForSession(user *entities.User) error {
	if user == nil || user.IsDeleted {
		return apperror.NewUnauthorized("Không tìm thấy người dùng này.")
	}

	if user.Status != userstatus.Active {
		return apperror.NewUnauthorized(contactSupportMessage)
	}

	return nil
}

func ensureUserEligibleForRecovery(user *entities.User) error {
	if user == nil || user.IsDeleted {
		return apperror.NewNotFound("Email này chưa được đăng kí trong hệ thống.")
	}

	if user.Status != userstatus.Active {
		return apperror.NewForbidden(contactSupportMessage)
	}

	return nil
}
