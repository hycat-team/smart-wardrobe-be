package usecase

import (
	"context"
	"strings"
	"time"

	"smart-wardrobe-be/internal/config"
	"smart-wardrobe-be/internal/modules/identity/application/dto"
	identityerrors "smart-wardrobe-be/internal/modules/identity/application/errors"
	"smart-wardrobe-be/internal/modules/identity/application/interface/security"
	uc_interfaces "smart-wardrobe-be/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/identity/application/mapper"
	"smart-wardrobe-be/internal/modules/identity/domain/constants/userstatus"
	"smart-wardrobe-be/internal/modules/identity/domain/entities"
	"smart-wardrobe-be/internal/modules/identity/domain/repositories"
	subscription_contract "smart-wardrobe-be/internal/modules/subscription/presentation/contract"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/media"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"
	"smart-wardrobe-be/internal/shared/domain/constants/shared/gender"

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

func (uc *UserUseCase) GetByID(ctx context.Context, userID uuid.UUID) (*dto.UserRes, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsDeleted {
		return nil, identityerrors.ErrUserProfileNotFound()
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
		return nil, identityerrors.ErrUserProfileNotFound()
	}

	sub, err := uc.subscriptionContract.GetUserSubscriptionOverview(ctx, user.ID)
	if err != nil {
		sub = nil
	}

	return mapper.MapToUserRes(user, sub), nil
}

func (uc *UserUseCase) GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*dto.UserRes, error) {
	if len(identifiers) == 0 {
		return nil, nil
	}

	users, err := uc.userRepo.GetByUsernamesOrEmails(ctx, identifiers)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}

	userIDs := make([]uuid.UUID, 0, len(users))
	for _, u := range users {
		if u != nil && !u.IsDeleted {
			userIDs = append(userIDs, u.ID)
		}
	}

	subscriptionsByUserID, err := uc.subscriptionContract.GetUserSubscriptionOverviews(ctx, userIDs)
	if err != nil {
		subscriptionsByUserID = nil
	}

	result := make([]*dto.UserRes, 0, len(users))
	for _, user := range users {
		if user == nil || user.IsDeleted {
			continue
		}
		result = append(result, mapper.MapToUserRes(user, subscriptionsByUserID[user.ID]))
	}

	return result, nil
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
		return nil, identityerrors.ErrUserProfileNotFound()
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
		return nil, identityerrors.ErrSelfStatusUpdate()
	}

	adminUser, err := uc.userRepo.GetByID(ctx, adminUserID)
	if err != nil {
		return nil, err
	}
	if adminUser == nil || adminUser.IsDeleted {
		return nil, identityerrors.ErrAdminAccountNotFound()
	}

	targetUser, err := uc.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if targetUser == nil || targetUser.IsDeleted {
		return nil, identityerrors.ErrUserProfileNotFound()
	}
	if targetUser.RoleSlug != roleslug.User {
		return nil, identityerrors.ErrUserStatusOnly()
	}

	targetUser.Status = input.Status
	if err := uc.userRepo.Update(ctx, targetUser); err != nil {
		return nil, err
	}

	switch input.Status {
	case userstatus.Inactive:
		if err := uc.refreshTokenRepo.RevokeAllByUserID(ctx, targetUserID); err != nil {
			return nil, err
		}

		accessTokenTTL := time.Minute * time.Duration(uc.cfg.Jwt.AccessExpirationMinutes)
		if err := uc.tokenBlacklistService.BlacklistUser(ctx, targetUserID, accessTokenTTL); err != nil {
			return nil, err
		}
	case userstatus.Active:
		if err := uc.tokenBlacklistService.UnblacklistUser(ctx, targetUserID); err != nil {
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

func (uc *UserUseCase) GetAdminUserStats(ctx context.Context, snapshotDate time.Time) (*dto.UserAdminStatsDTO, error) {
	return uc.userRepo.GetAdminUserStats(ctx, snapshotDate)
}

// GetUserDetail retrieves full user information and subscription overview for admin
func (uc *UserUseCase) GetUserDetail(ctx context.Context, userID uuid.UUID) (*dto.AdminUserDetailRes, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsDeleted {
		return nil, identityerrors.ErrUserNotFound()
	}

	sub, err := uc.subscriptionContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		sub = nil
	}

	userRes := mapper.MapToUserRes(user, sub)
	return &dto.AdminUserDetailRes{
		User:         userRes,
		Subscription: sub,
	}, nil
}

// CreateUserByAdmin allows an admin to create a new user account (defaults to user role)
func (uc *UserUseCase) CreateUserByAdmin(ctx context.Context, adminUserID uuid.UUID, req dto.CreateAdminUserReq) (*dto.UserRes, error) {
	// FR-012: Check role: only "user" is permitted unless special admin seed creation
	role := roleslug.User
	if req.RoleSlug != nil {
		if *req.RoleSlug == roleslug.Admin {
			return nil, identityerrors.ErrAdminCannotPromoteAdmin()
		}
		role = *req.RoleSlug
	}

	existingUser, err := uc.userRepo.GetByUsernameOrEmail(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, identityerrors.ErrUsernameExists(req.Username)
	}

	existingEmail, err := uc.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existingEmail != nil {
		return nil, identityerrors.ErrEmailExists(req.Email)
	}

	hash, err := uc.passwordHasher.HashPassword(req.Password)
	if err != nil {
		return nil, identityerrors.ErrAccountCreationFailed()
	}

	var dob *time.Time
	if req.DateOfBirth != nil && *req.DateOfBirth != "" {
		if parsedDob, err := time.Parse("2006-01-02", *req.DateOfBirth); err == nil {
			dob = &parsedDob
		}
	}

	newUser := &entities.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		FirstName:    &req.FirstName,
		LastName:     req.LastName,
		DateOfBirth:  dob,
		Gender:       req.Gender,
		RoleSlug:     role,
		Status:       userstatus.Active,
	}

	if err := uc.userRepo.Create(ctx, newUser); err != nil {
		return nil, err
	}

	return mapper.MapToUserRes(newUser, nil), nil
}

// UpdateUserRole updates a user's system role (guards against self-modification and elevating to admin)
func (uc *UserUseCase) UpdateUserRole(ctx context.Context, adminUserID uuid.UUID, targetUserID uuid.UUID, req dto.UpdateUserRoleReq) (*dto.UserRes, error) {
	if adminUserID == targetUserID {
		return nil, identityerrors.ErrSelfRoleUpdate()
	}

	targetUser, err := uc.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if targetUser == nil || targetUser.IsDeleted {
		return nil, identityerrors.ErrUserNotFound()
	}

	if targetUser.RoleSlug == roleslug.Admin {
		return nil, identityerrors.ErrCannotModifyAdminRole()
	}

	// FR-012: Regular admins cannot promote anyone to admin
	if req.RoleSlug == roleslug.Admin {
		return nil, identityerrors.ErrAdminCannotPromoteAdmin()
	}

	targetUser.RoleSlug = req.RoleSlug
	if err := uc.userRepo.Update(ctx, targetUser); err != nil {
		return nil, err
	}

	sub, _ := uc.subscriptionContract.GetUserSubscriptionOverview(ctx, targetUserID)
	return mapper.MapToUserRes(targetUser, sub), nil
}

// DeactivateUser performs a soft deactivation and revokes all active sessions/tokens
func (uc *UserUseCase) DeactivateUser(ctx context.Context, adminUserID uuid.UUID, targetUserID uuid.UUID) (*dto.UserRes, error) {
	if adminUserID == targetUserID {
		return nil, identityerrors.ErrSelfDeactivation()
	}

	targetUser, err := uc.userRepo.GetByID(ctx, targetUserID)
	if err != nil {
		return nil, err
	}
	if targetUser == nil || targetUser.IsDeleted {
		return nil, identityerrors.ErrUserNotFound()
	}

	if targetUser.RoleSlug == roleslug.Admin {
		return nil, identityerrors.ErrUserStatusOnly()
	}

	targetUser.Status = userstatus.Inactive
	if err := uc.userRepo.Update(ctx, targetUser); err != nil {
		return nil, err
	}

	_ = uc.refreshTokenRepo.RevokeAllByUserID(ctx, targetUserID)
	accessTokenTTL := time.Minute * time.Duration(uc.cfg.Jwt.AccessExpirationMinutes)
	_ = uc.tokenBlacklistService.BlacklistUser(ctx, targetUserID, accessTokenTTL)

	sub, _ := uc.subscriptionContract.GetUserSubscriptionOverview(ctx, targetUserID)
	return mapper.MapToUserRes(targetUser, sub), nil
}

// ListUserSessions returns the active and past login sessions for a user
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
