package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/dto/identity"
	identityerrors "smart-wardrobe-be/services/account/internal/modules/identity/application/errors"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/mapper"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/authprovider"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/userstatus"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	shared_dto "smart-wardrobe-be/shared/application/dto"
	"smart-wardrobe-be/shared/domain/constants/roleslug"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// userStatusChangedEvent is published on topic identity.user.status.changed so
// other services (community) can hide/restore a user's public content (FR-035).
type userStatusChangedEvent struct {
	UserID uuid.UUID `json:"userId"`
	Active bool      `json:"active"`
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

	if uc.eventPublisher != nil {
		payload := userStatusChangedEvent{UserID: targetUserID, Active: input.Status == userstatus.Active}
		if err := uc.eventPublisher.Publish(ctx, eventconstants.TopicUserStatusChanged, payload); err != nil {
			uc.logger.Warn("failed to publish user status changed event", zap.String("user_id", targetUserID.String()), zap.Error(err))
		}
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
	// Check role: only "user" is permitted unless special admin seed creation
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
		Username:      req.Username,
		Email:         req.Email,
		PasswordHash:  &hash,
		Provider:      authprovider.Local,
		EmailVerified: false,
		FirstName:     &req.FirstName,
		LastName:      req.LastName,
		DateOfBirth:   dob,
		Gender:        req.Gender,
		RoleSlug:      role,
		Status:        userstatus.Active,
	}

	// 024 T196 / FR-013: the account and its default entitlement are ONE
	// transaction. SC-006 asks for a paid-plan grant to exist within 60 seconds
	// of `POST /admin/users` with no subscription read in between, and this is
	// the path that missed it most: an administrator-created account got the free
	// plan late and silently, so its AI cost could not be reconciled against any
	// record.
	//
	// One `uow.Execute` wraps both writes because `user_subscriptions.user_id` is
	// a foreign key onto `identity.users(id)`: a second transaction would run on a
	// connection that cannot see the uncommitted user row, so the entitlement
	// would fail the FK check - or, worse, attach itself to a stale row and leave
	// a live grant with no account behind it.
	err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.userRepo.Create(txCtx, newUser); err != nil {
			return err
		}
		return uc.provisioningContract.EnsureDefaultSubscription(txCtx, newUser.ID)
	})
	if err != nil {
		// 024 T197: the unit of work above is also where the default entitlement
		// is provisioned, so its failure is an account-creation failure and has to
		// leave this module as one. Returning it raw made `POST /admin/users`
		// answer 404 with a message about the plan catalogue, which reads as
		// "this account does not exist" for an operation that was creating one.
		//
		// 024 T209: the factory no longer interpolates the cause into the client
		// facing message either. What it names is the KIND of failure, which is
		// what tells the administrator which part of the setup is broken; the raw
		// error stays on `Cause` for the log.
		return nil, identityerrors.ErrAccountCreationFailedWithCause(err)
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

	// Regular admins cannot promote anyone to admin
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
