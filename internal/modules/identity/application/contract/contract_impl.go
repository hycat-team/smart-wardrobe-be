package contract

import (
	"context"
	"errors"
	"smart-wardrobe-be/internal/modules/identity/contract"
	"smart-wardrobe-be/internal/modules/identity/domain/repositories"

	"github.com/google/uuid"
)

// IdentityModuleContractImpl implements external identity lookup functions
type IdentityModuleContractImpl struct {
	userRepo repositories.IUserRepository
}

// NewIdentityModuleContractImpl creates a new IdentityModuleContractImpl instance
func NewIdentityModuleContractImpl(repo repositories.IUserRepository) contract.IIdentityModuleContract {
	return &IdentityModuleContractImpl{
		userRepo: repo,
	}
}

// GetUserByID retrieves core user metadata for external module contexts
func (impl *IdentityModuleContractImpl) GetUserByID(ctx context.Context, id uuid.UUID) (*contract.PublicUserDTO, error) {
	user, err := impl.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsDeleted {
		return nil, errors.New("user not found or deleted")
	}

	return &contract.PublicUserDTO{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		RoleSlug: user.RoleSlug,
	}, nil
}
