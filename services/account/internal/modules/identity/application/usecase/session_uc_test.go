package usecase_test

import (
	"context"
	"errors"
	"testing"

	identitydto "smart-wardrobe-be/services/account/dto/identity"
	identityerrors "smart-wardrobe-be/services/account/internal/modules/identity/application/errors"
	appsecurity "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/security"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/usecase"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/authprovider"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/userstatus"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"
	user_repo "smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/shared/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type loginUserRepo struct {
	user_repo.IUserRepository
	user *entities.User
}

func (r *loginUserRepo) GetByUsernameOrEmail(ctx context.Context, value string) (*entities.User, error) {
	return r.user, nil
}

// exactHasher stands in for bcrypt: these tests care about whether Login
// reached the comparison at all, not about the hashing algorithm.
type exactHasher struct {
	password string
}

func (h *exactHasher) HashPassword(plain string) (string, error) { return plain, nil }

func (h *exactHasher) VerifyPassword(plain, hashed string) bool {
	return h.password == plain && h.password == hashed
}

func newLoginUseCase(user *entities.User, blacklist *fakeBlacklist) *usecase.SessionUseCase {
	uc, ok := usecase.NewSessionUseCase(
		&loginUserRepo{user: user},
		nil,
		nil,
		&exactHasher{password: "correct-horse"},
		blacklist,
		&config.Config{},
		nil,
	).(*usecase.SessionUseCase)
	if !ok {
		panic("NewSessionUseCase did not return *usecase.SessionUseCase")
	}
	return uc
}

func loginReq(password string) identitydto.LoginReq {
	return identitydto.LoginReq{LoginName: "duck@example.com", Password: password}
}

func TestLogin_PasswordNeverSet_GoogleOnly(t *testing.T) {
	uc := newLoginUseCase(&entities.User{
		Provider: authprovider.Google,
		Status:   userstatus.Active,
	}, &fakeBlacklist{})

	_, err := uc.Login(context.Background(), loginReq("anything"))

	require.Error(t, err)
	assert.True(t, errors.Is(err, identityerrors.ErrPasswordNotSet()), "got %v", err)
	assert.False(t, errors.Is(err, identityerrors.ErrInvalidCredentials()))
}

func TestLogin_PasswordNeverSet_LocalAccount(t *testing.T) {
	empty := ""
	uc := newLoginUseCase(&entities.User{
		Provider:     authprovider.Local,
		Status:       userstatus.Active,
		PasswordHash: &empty,
	}, &fakeBlacklist{})

	_, err := uc.Login(context.Background(), loginReq("anything"))

	require.Error(t, err)
	assert.True(t, errors.Is(err, identityerrors.ErrPasswordNotSet()), "got %v", err)
}

func TestLogin_WrongPassword_StaysGeneric(t *testing.T) {
	hash := "correct-horse"
	uc := newLoginUseCase(&entities.User{
		Provider:     authprovider.Local,
		Status:       userstatus.Active,
		PasswordHash: &hash,
	}, &fakeBlacklist{})

	_, err := uc.Login(context.Background(), loginReq("wrong-password"))

	require.Error(t, err)
	assert.True(t, errors.Is(err, identityerrors.ErrInvalidCredentials()), "got %v", err)
	assert.False(t, errors.Is(err, identityerrors.ErrPasswordNotSet()))
}

func TestLogin_CorrectPassword_PassesCredentialsCheck(t *testing.T) {
	hash := "correct-horse"
	uc := newLoginUseCase(&entities.User{
		Provider:     authprovider.Local,
		Status:       userstatus.Active,
		PasswordHash: &hash,
	}, &fakeBlacklist{blacklisted: true})

	_, err := uc.Login(context.Background(), loginReq("correct-horse"))

	// A blacklisted account still fails, but the failure is downstream of the
	// credential check, which proves a correct password is no longer rejected
	// as either of the two credential errors.
	require.Error(t, err)
	assert.True(t, errors.Is(err, identityerrors.ErrAccountDisabled()), "got %v", err)
}

var _ appsecurity.IPasswordHasher = (*exactHasher)(nil)
