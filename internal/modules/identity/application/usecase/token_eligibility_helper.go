package usecase

import (
	identityerrors "smart-wardrobe-be/internal/modules/identity/application/errors"
	"smart-wardrobe-be/internal/shared/domain/constants/identity/userstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

func ensureUserEligibleForLogin(user *entities.User) error {
	if user == nil {
		return identityerrors.ErrInvalidCredentials()
	}

	if user.IsDeleted || user.Status != userstatus.Active {
		return identityerrors.ErrAccountDisabled()
	}

	return nil
}

func ensureUserEligibleForSession(user *entities.User) error {
	if user == nil || user.IsDeleted {
		return identityerrors.ErrUserNotFoundDetailed()
	}

	if user.Status != userstatus.Active {
		return identityerrors.ErrAccountDisabledAuth()
	}

	return nil
}

func ensureUserEligibleForRecovery(user *entities.User) error {
	if user == nil || user.IsDeleted {
		return identityerrors.ErrEmailNotRegistered()
	}

	if user.Status != userstatus.Active {
		return identityerrors.ErrAccountDisabled()
	}

	return nil
}
