package usecase

import (
	"context"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/identity/application/dto"
	identityerrors "smart-wardrobe-be/internal/modules/identity/application/errors"
	"smart-wardrobe-be/internal/modules/identity/application/mapper"
	"smart-wardrobe-be/internal/modules/identity/domain/entities"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/shared/gender"

	"github.com/google/uuid"
)

func (uc *UserUseCase) ChangePassword(ctx context.Context, userID uuid.UUID, input dto.ChangePasswordReq) (bool, error) {
	if input.NewPassword == input.OldPassword {
		return false, identityerrors.ErrNewPasswordSameAsOld()
	}

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if user == nil || user.IsDeleted {
		return false, identityerrors.ErrUserNotFound()
	}

	if strings.Contains(strings.ToLower(input.NewPassword), strings.ToLower(user.Username)) {
		return false, identityerrors.ErrPasswordContainsUsername()
	}

	isValid := uc.passwordHasher.VerifyPassword(input.OldPassword, user.PasswordHash)
	if !isValid {
		return false, identityerrors.ErrInvalidOldPassword()
	}

	newPasswordHash, err := uc.passwordHasher.HashPassword(input.NewPassword)
	if err != nil {
		return false, err
	}

	user.ChangePasswordHash(newPasswordHash)

	if input.LogoutAllDevices {
		err = uc.refreshTokenRepo.RevokeAllByUserID(ctx, userID)
		if err != nil {
			return false, err
		}
	}

	err = uc.userRepo.Update(ctx, user)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (uc *UserUseCase) UpdateProfile(ctx context.Context, userID uuid.UUID, input dto.UpdateProfileReq) (*dto.UserRes, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsDeleted {
		return nil, identityerrors.ErrUserProfileNotFound()
	}

	dob, err := time.Parse(time.DateOnly, input.DateOfBirth)
	if err != nil {
		return nil, identityerrors.ErrInvalidDob()
	}

	genVal := gender.Unknown
	if input.Gender != nil {
		genVal = *input.Gender
	} else if user.Gender != nil {
		genVal = *user.Gender
	}

	user.UpdateProfile(input.FirstName, input.LastName, dob, genVal)
	user.ChangeAddress(input.Address)

	err = uc.userRepo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	sub, err := uc.subscriptionContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		sub = nil
	}

	return mapper.MapToUserRes(user, sub), nil
}

func (uc *UserUseCase) UpdateBodyProfile(ctx context.Context, userID uuid.UUID, input dto.UpdateBodyProfileReq) (*dto.UserRes, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsDeleted {
		return nil, identityerrors.ErrUserProfileNotFound()
	}

	now := time.Now().UTC()
	profile := &entities.BodyProfile{
		HeightCM:       input.HeightCM,
		WeightKG:       input.WeightKG,
		BodyShape:      input.BodyShape,
		VerifiedByUser: input.VerifiedByUser,
		LastUpdatedAt:  &now,
	}

	if input.Measurements != nil {
		profile.Measurements = &entities.BodyMeasurements{
			ChestCM: input.Measurements.ChestCM,
			WaistCM: input.Measurements.WaistCM,
			HipCM:   input.Measurements.HipCM,
		}
	}

	if input.InferredByAI != nil {
		profile.InferredByAI = &entities.InferredBodyProfile{
			BodyShape:       input.InferredByAI.BodyShape,
			ConfidenceScore: input.InferredByAI.ConfidenceScore,
		}
	}

	user.UpdateBodyProfile(profile)
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	sub, err := uc.subscriptionContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		sub = nil
	}

	return mapper.MapToUserRes(user, sub), nil
}

func (uc *UserUseCase) GetStyleProfile(ctx context.Context, userID uuid.UUID) (*dto.UserStyleProfileRes, error) {
	return uc.userRepo.GetStyleProfile(ctx, userID)
}

func (uc *UserUseCase) GetAvatarSignature(ctx context.Context, userID uuid.UUID) (*shared_dto.UploadSignatureResult, error) {
	params := shared_dto.UploadSignatureParams{
		Folder:   uc.cfg.Cloudinary.AvatarFolder,
		PublicID: userID.String(),
	}
	return uc.mediaService.GenerateUploadSignature(ctx, params)
}

func (uc *UserUseCase) UpdateAvatar(ctx context.Context, userID uuid.UUID, input dto.UpdateAvatarReq) (*dto.UserBriefRes, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsDeleted {
		return nil, identityerrors.ErrUserProfileNotFound()
	}

	user.UpdateAvatar(input.AvatarUrl, input.AvatarPublicID)
	err = uc.userRepo.Update(ctx, user)
	if err != nil {
		return nil, err
	}

	return mapper.MapUserBriefToRes(mapper.MapToUserBrief(user)), nil
}

func (uc *UserUseCase) ListUserSessions(ctx context.Context, userID uuid.UUID) ([]*dto.UserSessionItemDTO, error) {
	tokens, err := uc.refreshTokenRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	res := make([]*dto.UserSessionItemDTO, 0, len(tokens))
	for _, t := range tokens {
		if t == nil {
			continue
		}
		res = append(res, &dto.UserSessionItemDTO{
			ID:           t.ID,
			DeviceName:   t.DeviceName,
			DeviceIP:     t.DeviceIP,
			UserAgent:    t.UserAgent,
			LastActiveAt: t.LastActiveAt,
			ExpiresAt:    t.ExpiresAt,
			IsRevoked:    t.IsRevoked,
			CreatedAt:    t.CreatedAt,
		})
	}
	return res, nil
}

// RevokeUserSessions revokes all refresh tokens and blacklists the user's active access tokens
func (uc *UserUseCase) RevokeUserSessions(ctx context.Context, userID uuid.UUID) error {
	if err := uc.refreshTokenRepo.RevokeAllByUserID(ctx, userID); err != nil {
		return err
	}
	accessTokenTTL := time.Minute * time.Duration(uc.cfg.Jwt.AccessExpirationMinutes)
	return uc.tokenBlacklistService.BlacklistUser(ctx, userID, accessTokenTTL)
}
