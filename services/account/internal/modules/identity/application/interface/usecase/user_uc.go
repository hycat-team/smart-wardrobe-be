package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/services/account/dto/identity"
	shared_dto "smart-wardrobe-be/shared/application/dto"

	"github.com/google/uuid"
)

type IUserUseCase interface {
	ChangePassword(ctx context.Context, userID uuid.UUID, input dto.ChangePasswordReq) (bool, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, input dto.UpdateProfileReq) (*dto.UserRes, error)
	UpdateBodyProfile(ctx context.Context, userID uuid.UUID, input dto.UpdateBodyProfileReq) (*dto.UserRes, error)
	GetByID(ctx context.Context, userID uuid.UUID) (*dto.UserRes, error)
	GetByIDs(ctx context.Context, userIDs []uuid.UUID) ([]*dto.UserRes, error)
	GetByUsername(ctx context.Context, username string) (*dto.UserRes, error)
	GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*dto.UserRes, error)
	GetAvatarSignature(ctx context.Context, userID uuid.UUID) (*shared_dto.UploadSignatureResult, error)
	UpdateAvatar(ctx context.Context, userID uuid.UUID, input dto.UpdateAvatarReq) (*dto.UserBriefRes, error)
	UpdateUserStatus(ctx context.Context, adminUserID uuid.UUID, targetUserID uuid.UUID, input dto.UpdateUserStatusReq) (*dto.UserRes, error)
	GetUsersForAdmin(ctx context.Context, query dto.GetUsersQueryReq) (*dto.AdminUserListRes, error)
	// GetActiveUserIDs returns every non-deleted account id, unpaginated: it
	// backs the admin dashboard reader, where a page-sized answer would
	// silently drop most of the population.
	GetActiveUserIDs(ctx context.Context) ([]uuid.UUID, error)
	GetAdminUserStats(ctx context.Context, snapshotDate time.Time) (*dto.UserAdminStatsDTO, error)
	GetUserDetail(ctx context.Context, userID uuid.UUID) (*dto.AdminUserDetailRes, error)
	CreateUserByAdmin(ctx context.Context, adminUserID uuid.UUID, req dto.CreateAdminUserReq) (*dto.UserRes, error)
	UpdateUserRole(ctx context.Context, adminUserID uuid.UUID, targetUserID uuid.UUID, req dto.UpdateUserRoleReq) (*dto.UserRes, error)
	DeactivateUser(ctx context.Context, adminUserID uuid.UUID, targetUserID uuid.UUID) (*dto.UserRes, error)
	ListUserSessions(ctx context.Context, userID uuid.UUID) ([]*dto.UserSessionItemDTO, error)
	RevokeUserSessions(ctx context.Context, userID uuid.UUID) error
}
