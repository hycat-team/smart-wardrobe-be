package usecase_test

import (
	"context"
	"testing"

	"smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/usecase"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"
	user_repo "smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/domain/constants/roleslug"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockUserRepo struct {
	mock.Mock
	user_repo.IUserRepository
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}

func (m *mockUserRepo) Update(ctx context.Context, user *entities.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func TestUserUseCase_UpdateUserRole(t *testing.T) {
	ctx := context.Background()
	adminID := uuid.New()
	targetUserID := uuid.New()

	t.Run("fails when updating self role", func(t *testing.T) {
		uc := usecase.NewUserUseCase(nil, nil, nil, nil, nil, nil, &config.Config{}, nil, nil)
		_, err := uc.UpdateUserRole(ctx, adminID, adminID, dto.UpdateUserRoleReq{RoleSlug: roleslug.User})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Bạn không thể tự thay đổi vai trò tài khoản của chính mình.")
	})

	t.Run("fails when promoting to admin", func(t *testing.T) {
		mockRepo := new(mockUserRepo)
		targetUser := &entities.User{
			RoleSlug: roleslug.User,
		}
		targetUser.ID = targetUserID

		mockRepo.On("GetByID", ctx, targetUserID).Return(targetUser, nil)

		uc := usecase.NewUserUseCase(mockRepo, nil, nil, nil, nil, nil, &config.Config{}, nil, nil)
		_, err := uc.UpdateUserRole(ctx, adminID, targetUserID, dto.UpdateUserRoleReq{RoleSlug: roleslug.Admin})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Chỉ tài khoản admin seed mới được quyền phong hoặc tạo tài khoản admin.")
	})

	t.Run("fails when target user is already admin", func(t *testing.T) {
		mockRepo := new(mockUserRepo)
		targetUser := &entities.User{
			RoleSlug: roleslug.Admin,
		}
		targetUser.ID = targetUserID

		mockRepo.On("GetByID", ctx, targetUserID).Return(targetUser, nil)

		uc := usecase.NewUserUseCase(mockRepo, nil, nil, nil, nil, nil, &config.Config{}, nil, nil)
		_, err := uc.UpdateUserRole(ctx, adminID, targetUserID, dto.UpdateUserRoleReq{RoleSlug: roleslug.User})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Không thể thay đổi vai trò của tài khoản quản trị viên khác.")
	})
}
