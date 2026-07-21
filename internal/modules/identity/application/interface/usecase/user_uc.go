package usecase

import (
	"context"
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
	GetByUsername(ctx context.Context, username string) (*dto.UserRes, error)
	GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*dto.UserRes, error)
	GetStyleProfile(ctx context.Context, userID uuid.UUID) (*dto.UserStyleProfileRes, error)
	GetAvatarSignature(ctx context.Context, userID uuid.UUID) (*shared_dto.UploadSignatureResult, error)
	UpdateAvatar(ctx context.Context, userID uuid.UUID, input dto.UpdateAvatarReq) (*dto.UserRes, error)
	UpdateUserStatus(ctx context.Context, adminUserID uuid.UUID, targetUserID uuid.UUID, input dto.UpdateUserStatusReq) (*dto.UserRes, error)
	GetUsersForAdmin(ctx context.Context, query dto.GetUsersQueryReq) (*dto.AdminUserListRes, error)
}
