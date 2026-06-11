package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/identity/application/dto"
	identityerrors "smart-wardrobe-be/internal/modules/identity/application/errors"
	"smart-wardrobe-be/internal/modules/identity/application/interface/security"
	uc_interfaces "smart-wardrobe-be/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/identity/application/mapper"
	"smart-wardrobe-be/internal/modules/identity/domain/repositories"
	subscription_contract "smart-wardrobe-be/internal/modules/subscription/contract"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/media"
	"smart-wardrobe-be/internal/shared/domain/constants/gender"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	"smart-wardrobe-be/internal/shared/domain/constants/userstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type UserUseCase struct {
	userRepo              repositories.IUserRepository
	passwordHasher        security.IPasswordHasher
	refreshTokenRepo      repositories.IRefreshTokenRepository
	tokenBlacklistService security.ITokenBlacklistService
	subscriptionContract  subscription_contract.IUserSubscriptionContract
	mediaService          media.IMediaService
	cfg                   *config.Config
}

func NewUserUseCase(
	userRepo repositories.IUserRepository,
	passwordHasher security.IPasswordHasher,
	refreshTokenRepo repositories.IRefreshTokenRepository,
	tokenBlacklistService security.ITokenBlacklistService,
	subscriptionContract subscription_contract.IUserSubscriptionContract,
	mediaService media.IMediaService,
	cfg *config.Config,
) uc_interfaces.IUserUseCase {
	return &UserUseCase{
		userRepo:              userRepo,
		passwordHasher:        passwordHasher,
		refreshTokenRepo:      refreshTokenRepo,
		tokenBlacklistService: tokenBlacklistService,
		subscriptionContract:  subscriptionContract,
		mediaService:          mediaService,
		cfg:                   cfg,
	}
}

func (uc *UserUseCase) ChangePassword(ctx context.Context, userID uuid.UUID, input dto.ChangePasswordReq) (bool, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	if user == nil || user.IsDeleted {
		return false, identityerrors.ErrUserNotFound
	}

	isValid := uc.passwordHasher.VerifyPassword(input.OldPassword, user.PasswordHash)
	if !isValid {
		return false, identityerrors.ErrInvalidOldPassword
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
		return nil, identityerrors.ErrUserProfileNotFound
	}

	dob, err := time.Parse(time.DateOnly, input.DateOfBirth)
	if err != nil {
		return nil, identityerrors.ErrInvalidDob
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
		return nil, identityerrors.ErrUserProfileNotFound
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

func (uc *UserUseCase) GetByID(ctx context.Context, userID uuid.UUID) (*dto.UserRes, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsDeleted {
		return nil, identityerrors.ErrUserProfileNotFound
	}

	sub, err := uc.subscriptionContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		sub = nil
	}

	return mapper.MapToUserRes(user, sub), nil
}

func (uc *UserUseCase) GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*dto.UserRes, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}

	users, err := uc.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	usersByID := make(map[uuid.UUID]*entities.User, len(users))
	for _, user := range users {
		if user == nil || user.IsDeleted {
			continue
		}
		usersByID[user.ID] = user
	}

	subscriptionsByUserID, err := uc.subscriptionContract.GetUserSubscriptionOverviews(ctx, userIDs)
	if err != nil {
		subscriptionsByUserID = nil
	}

	result := make([]*dto.UserRes, 0, len(userIDs))
	for _, userID := range userIDs {
		user := usersByID[userID]
		if user == nil {
			continue
		}
		result = append(result, mapper.MapToUserRes(user, subscriptionsByUserID[user.ID]))
	}

	return result, nil
}

func (uc *UserUseCase) GetByUsername(ctx context.Context, username string) (*dto.UserRes, error) {
	user, err := uc.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsDeleted {
		return nil, identityerrors.ErrUserProfileNotFound
	}

	sub, err := uc.subscriptionContract.GetUserSubscriptionOverview(ctx, user.ID)
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

func (uc *UserUseCase) UpdateAvatar(ctx context.Context, userID uuid.UUID, input dto.UpdateAvatarReq) (*dto.UserRes, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsDeleted {
		return nil, identityerrors.ErrUserProfileNotFound
	}

	user.UpdateAvatar(input.AvatarUrl, input.AvatarPublicID)
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

func (uc *UserUseCase) UpdateUserStatus(ctx context.Context, adminUserID uuid.UUID, targetUserID uuid.UUID, input dto.UpdateUserStatusReq) (*dto.UserRes, error) {
	if adminUserID == targetUserID {
		return nil, identityerrors.ErrSelfStatusUpdate
	}

	adminUser, err := uc.userRepo.GetByID(ctx, adminUserID)
	if err != nil {
		return nil, err
	}
	if adminUser == nil || adminUser.IsDeleted {
		return nil, identityerrors.ErrAdminAccountNotFound
	}

	targetUser, err := uc.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if targetUser == nil || targetUser.IsDeleted {
		return nil, identityerrors.ErrUserProfileNotFound
	}
	if targetUser.RoleSlug != roleslug.User {
		return nil, identityerrors.ErrUserStatusOnly
	}

	targetUser.Status = input.Status
	if err := uc.userRepo.Update(ctx, targetUser); err != nil {
		return nil, err
	}

	if input.Status == userstatus.Inactive {
		if err := uc.refreshTokenRepo.RevokeAllByUserID(ctx, targetUserID); err != nil {
			return nil, err
		}

		accessTokenTTL := time.Minute * time.Duration(uc.cfg.Jwt.AccessExpirationMinutes)
		if err := uc.tokenBlacklistService.BlacklistUser(ctx, targetUserID, accessTokenTTL); err != nil {
			return nil, err
		}
	}

	sub, err := uc.subscriptionContract.GetUserSubscriptionOverview(ctx, targetUserID)
	if err != nil {
		sub = nil
	}

	return mapper.MapToUserRes(targetUser, sub), nil
}

func (uc *UserUseCase) GetUsersForAdmin(ctx context.Context, query dto.GetUsersQueryReq) (*dto.AdminUserListRes, error) {
	filter := repositories.UserFilter{
		RoleSlug: query.RoleSlug,
		IsActive: query.IsActive,
		Query:    query.Query,
		Page:     query.Page,
		Limit:    query.Limit,
	}

	result, err := uc.userRepo.GetUsersForAdmin(ctx, filter)
	if err != nil {
		return nil, err
	}

	resUsers := make([]*dto.UserRes, len(result.Users))
	userIDs := make([]uuid.UUID, 0, len(result.Users))
	for _, user := range result.Users {
		userIDs = append(userIDs, user.ID)
	}

	subscriptionsByUserID, err := uc.subscriptionContract.GetUserSubscriptionOverviews(ctx, userIDs)
	if err != nil {
		subscriptionsByUserID = nil
	}

	for idx, user := range result.Users {
		resUsers[idx] = mapper.MapToUserRes(user, subscriptionsByUserID[user.ID])
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	page := query.Page
	if page <= 0 {
		page = 1
	}

	return &shared_dto.PaginationResult[*dto.UserRes]{
		Items:    resUsers,
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, result.TotalCount),
	}, nil
}
