package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/identity/application/dto"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"

	"github.com/google/uuid"
)

type IUserUseCase interface {
	ChangePassword(ctx context.Context, userID uuid.UUID, input dto.ChangePasswordReq) (bool, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, input dto.UpdateProfileReq) (*dto.UserRes, error)
	UpdateBodyProfile(ctx context.Context, userID uuid.UUID, input dto.UpdateBodyProfileReq) (*dto.UserRes, error)
	GetByID(ctx context.Context, userID uuid.UUID) (*dto.UserRes, error)
	GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*dto.UserRes, error)
	GetBriefByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*dto.UserBriefRes, error)
	GetByUsername(ctx context.Context, username string) (*dto.UserRes, error)
	GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*dto.UserRes, error)
	GetStyleProfile(ctx context.Context, userID uuid.UUID) (*dto.UserStyleProfileRes, error)
	GetAvatarSignature(ctx context.Context, userID uuid.UUID) (*shared_dto.UploadSignatureResult, error)
	UpdateAvatar(ctx context.Context, userID uuid.UUID, input dto.UpdateAvatarReq) (*dto.UserRes, error)
	UpdateUserStatus(ctx context.Context, adminUserID uuid.UUID, targetUserID uuid.UUID, input dto.UpdateUserStatusReq) (*dto.UserRes, error)
	GetUsersForAdmin(ctx context.Context, query dto.GetUsersQueryReq) (*dto.AdminUserListRes, error)
	GetAdminUserStats(ctx context.Context, snapshotDate time.Time) (*dto.UserAdminStatsDTO, error)
	GetUserDetail(ctx context.Context, userID uuid.UUID) (*dto.AdminUserDetailRes, error)
	CreateUserByAdmin(ctx context.Context, adminUserID uuid.UUID, req dto.CreateAdminUserReq) (*dto.UserRes, error)
	UpdateUserRole(ctx context.Context, adminUserID uuid.UUID, targetUserID uuid.UUID, req dto.UpdateUserRoleReq) (*dto.UserRes, error)
	DeactivateUser(ctx context.Context, adminUserID uuid.UUID, targetUserID uuid.UUID) (*dto.UserRes, error)
	ListUserSessions(ctx context.Context, userID uuid.UUID) ([]*dto.UserSessionItemDTO, error)
	RevokeUserSessions(ctx context.Context, userID uuid.UUID) error
}
