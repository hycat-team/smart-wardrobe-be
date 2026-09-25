package usecase_test

import (
	"context"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/dto/identity"
	identityiface "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/identity"
	appsecurity "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/security"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/usecase"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/vo"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/authprovider"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/userstatus"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/domain/constants/roleslug"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func (m *mockUserRepo) GetByProviderID(ctx context.Context, provider string, providerID string) (*entities.User, error) {
	args := m.Called(ctx, provider, providerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*entities.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.User), args.Error(1)
}

func (m *mockUserRepo) IsUsernameExists(ctx context.Context, username string) (bool, error) {
	args := m.Called(ctx, username)
	return args.Bool(0), args.Error(1)
}

func (m *mockUserRepo) Create(ctx context.Context, user *entities.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

type fakeVerifier struct {
	appsecurity.IGoogleTokenVerifier
	ident *appsecurity.GoogleIdentity
	err   error
}

func (f *fakeVerifier) VerifyIDToken(ctx context.Context, idToken string) (*appsecurity.GoogleIdentity, error) {
	return f.ident, f.err
}

type fakeSession struct {
	uc_interfaces.ISessionUseCase
	issued *entities.User
	err    error
}

func (f *fakeSession) IssueSession(ctx context.Context, user *entities.User, device uc_interfaces.SessionDevice) (*dto.TokenRes, error) {
	f.issued = user
	if f.err != nil {
		return nil, f.err
	}
	return &dto.TokenRes{AccessToken: "access", RefreshToken: "refresh"}, nil
}

type fakeStateStore struct {
	identityiface.IOAuthStateStore
	data *vo.OAuthState
}

func (f *fakeStateStore) Save(ctx context.Context, state string, data vo.OAuthState, ttl time.Duration) error {
	return nil
}

func (f *fakeStateStore) Consume(ctx context.Context, state string) (*vo.OAuthState, error) {
	return f.data, nil
}

type fakeBlacklist struct {
	appsecurity.ITokenBlacklistService
	blacklisted bool
}

func (f *fakeBlacklist) IsUserBlacklisted(ctx context.Context, userID uuid.UUID) (bool, error) {
	return f.blacklisted, nil
}

type fakeUoW struct{}

func (f *fakeUoW) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func newGoogleUC(repo *mockUserRepo, verifier *fakeVerifier, session *fakeSession, state *fakeStateStore, blacklist *fakeBlacklist) uc_interfaces.IGoogleAuthUseCase {
	return usecase.NewGoogleAuthUseCase(repo, session, verifier, state, blacklist, &config.Config{}, &fakeUoW{})
}

func identityClaim() *appsecurity.GoogleIdentity {
	return &appsecurity.GoogleIdentity{Subject: "sub-1", Email: "Duck@Gmail.com", EmailVerified: true}
}

func TestGoogleLogin_NewUser(t *testing.T) {
	ctx := context.Background()
	repo := new(mockUserRepo)
	repo.On("GetByProviderID", ctx, "google", "sub-1").Return(nil, nil)
	repo.On("GetByEmail", ctx, "duck@gmail.com").Return(nil, nil)
	repo.On("IsUsernameExists", ctx, mock.Anything).Return(false, nil)
	repo.On("Create", ctx, mock.Anything).Return(nil)

	session := &fakeSession{}
	uc := newGoogleUC(repo, &fakeVerifier{ident: identityClaim()}, session, &fakeStateStore{}, &fakeBlacklist{})

	res, err := uc.LoginWithIDToken(ctx, dto.GoogleLoginReq{IDToken: "token"})
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.NotNil(t, session.issued)
	assert.Equal(t, "duck@gmail.com", session.issued.Email)
	assert.Equal(t, authprovider.Google, session.issued.Provider)
	assert.Equal(t, roleslug.User, session.issued.RoleSlug)
	assert.True(t, session.issued.EmailVerified)
	assert.Equal(t, "duck", session.issued.Username)
}

func TestGoogleLogin_UnverifiedEmail(t *testing.T) {
	ctx := context.Background()
	repo := new(mockUserRepo)
	session := &fakeSession{}
	ident := identityClaim()
	ident.EmailVerified = false
	uc := newGoogleUC(repo, &fakeVerifier{ident: ident}, session, &fakeStateStore{}, &fakeBlacklist{})

	_, err := uc.LoginWithIDToken(ctx, dto.GoogleLoginReq{IDToken: "token"})
	assert.Error(t, err)
	assert.Nil(t, session.issued)
}

func TestGoogleLogin_AutoLinkVerifiedLocal(t *testing.T) {
	ctx := context.Background()
	hash := "hash"
	local := &entities.User{
		Username:      "duck",
		Email:         "duck@gmail.com",
		PasswordHash:  &hash,
		Provider:      authprovider.Local,
		EmailVerified: true,
		RoleSlug:      roleslug.User,
		Status:        userstatus.Active,
	}
	local.ID = uuid.New()

	repo := new(mockUserRepo)
	repo.On("GetByProviderID", ctx, "google", "sub-1").Return(nil, nil)
	repo.On("GetByEmail", ctx, "duck@gmail.com").Return(local, nil)
	repo.On("Update", ctx, local).Return(nil)

	session := &fakeSession{}
	uc := newGoogleUC(repo, &fakeVerifier{ident: identityClaim()}, session, &fakeStateStore{}, &fakeBlacklist{})

	_, err := uc.LoginWithIDToken(ctx, dto.GoogleLoginReq{IDToken: "token"})
	assert.NoError(t, err)
	assert.Equal(t, authprovider.Google, local.Provider)
	assert.NotNil(t, local.ProviderID)
	assert.Equal(t, "sub-1", *local.ProviderID)
	repo.AssertExpectations(t)
}

func TestGoogleLogin_AdminCreatedNotVerified(t *testing.T) {
	ctx := context.Background()
	hash := "hash"
	local := &entities.User{Username: "duck", Email: "duck@gmail.com", PasswordHash: &hash, Provider: authprovider.Local, EmailVerified: false}
	local.ID = uuid.New()

	repo := new(mockUserRepo)
	repo.On("GetByProviderID", ctx, "google", "sub-1").Return(nil, nil)
	repo.On("GetByEmail", ctx, "duck@gmail.com").Return(local, nil)

	uc := newGoogleUC(repo, &fakeVerifier{ident: identityClaim()}, &fakeSession{}, &fakeStateStore{}, &fakeBlacklist{})
	_, err := uc.LoginWithIDToken(ctx, dto.GoogleLoginReq{IDToken: "token"})
	assert.Error(t, err)
}

func TestGoogleLogin_LinkedOtherGoogle(t *testing.T) {
	ctx := context.Background()
	other := "sub-2"
	local := &entities.User{Username: "duck", Email: "duck@gmail.com", Provider: authprovider.Google, ProviderID: &other, EmailVerified: true}
	local.ID = uuid.New()

	repo := new(mockUserRepo)
	repo.On("GetByProviderID", ctx, "google", "sub-1").Return(nil, nil)
	repo.On("GetByEmail", ctx, "duck@gmail.com").Return(local, nil)

	uc := newGoogleUC(repo, &fakeVerifier{ident: identityClaim()}, &fakeSession{}, &fakeStateStore{}, &fakeBlacklist{})
	_, err := uc.LoginWithIDToken(ctx, dto.GoogleLoginReq{IDToken: "token"})
	assert.Error(t, err)
}

func TestGoogleLogin_DisabledAccount(t *testing.T) {
	ctx := context.Background()
	disabled := &entities.User{Provider: authprovider.Google, Status: userstatus.Inactive}
	disabled.ID = uuid.New()

	repo := new(mockUserRepo)
	repo.On("GetByProviderID", ctx, "google", "sub-1").Return(disabled, nil)

	uc := newGoogleUC(repo, &fakeVerifier{ident: identityClaim()}, &fakeSession{}, &fakeStateStore{}, &fakeBlacklist{})
	_, err := uc.LoginWithIDToken(ctx, dto.GoogleLoginReq{IDToken: "token"})
	assert.Error(t, err)
}

func TestGoogleLogin_Blacklisted(t *testing.T) {
	ctx := context.Background()
	active := &entities.User{Provider: authprovider.Google, Status: userstatus.Active, RoleSlug: roleslug.User}
	active.ID = uuid.New()

	repo := new(mockUserRepo)
	repo.On("GetByProviderID", ctx, "google", "sub-1").Return(active, nil)

	uc := newGoogleUC(repo, &fakeVerifier{ident: identityClaim()}, &fakeSession{}, &fakeStateStore{}, &fakeBlacklist{blacklisted: true})
	_, err := uc.LoginWithIDToken(ctx, dto.GoogleLoginReq{IDToken: "token"})
	assert.Error(t, err)
}

func TestGoogleLogin_RaceReFetch(t *testing.T) {
	ctx := context.Background()
	created := &entities.User{Provider: authprovider.Google, Status: userstatus.Active, RoleSlug: roleslug.User}
	created.ID = uuid.New()

	repo := new(mockUserRepo)
	repo.On("GetByProviderID", ctx, "google", "sub-1").Return(nil, nil).Once()
	repo.On("GetByEmail", ctx, "duck@gmail.com").Return(nil, nil)
	repo.On("IsUsernameExists", ctx, mock.Anything).Return(false, nil)
	repo.On("Create", ctx, mock.Anything).Return(assert.AnError)
	repo.On("GetByProviderID", ctx, "google", "sub-1").Return(created, nil).Once()

	session := &fakeSession{}
	uc := newGoogleUC(repo, &fakeVerifier{ident: identityClaim()}, session, &fakeStateStore{}, &fakeBlacklist{})

	_, err := uc.LoginWithIDToken(ctx, dto.GoogleLoginReq{IDToken: "token"})
	assert.NoError(t, err)
	assert.Equal(t, created, session.issued)
}

func googleRedirectCfg() *config.Config {
	return &config.Config{
		Server: config.Server{Env: "vps", FrontEndOrigins: []string{"https://closy.hycat.online"}},
		Google: config.Google{
			ClientIDRaw:     "web-client",
			RedirectURI:     "https://api.example.com/api/v1/auth/google/callback",
			StateTTLMinutes: 10,
			ClientIDs:       []string{"web-client"},
		},
	}
}

func newGoogleUCWithConfig(cfg *config.Config, repo *mockUserRepo, session *fakeSession, state *fakeStateStore) uc_interfaces.IGoogleAuthUseCase {
	return usecase.NewGoogleAuthUseCase(repo, session, &fakeVerifier{}, state, &fakeBlacklist{}, cfg, &fakeUoW{})
}

func TestGoogleStartRedirect_InvalidURL(t *testing.T) {
	ctx := context.Background()
	uc := newGoogleUCWithConfig(googleRedirectCfg(), new(mockUserRepo), &fakeSession{}, &fakeStateStore{})

	_, err := uc.StartRedirect(ctx, "https://evil.example.com/")
	assert.Error(t, err)
}

func TestGoogleStartRedirect_Valid(t *testing.T) {
	ctx := context.Background()
	uc := newGoogleUCWithConfig(googleRedirectCfg(), new(mockUserRepo), &fakeSession{}, &fakeStateStore{})

	authURL, err := uc.StartRedirect(ctx, "https://closy.hycat.online/auth/callback")
	assert.NoError(t, err)
	assert.Contains(t, authURL, "accounts.google.com")
	assert.Contains(t, authURL, "state=")
}

func TestGoogleCallback_InvalidState(t *testing.T) {
	ctx := context.Background()
	uc := newGoogleUCWithConfig(googleRedirectCfg(), new(mockUserRepo), &fakeSession{}, &fakeStateStore{data: nil})

	_, err := uc.HandleCallback(ctx, "code", "unknown-state", "")
	assert.Error(t, err)
}

func TestGoogleCallback_ProviderError(t *testing.T) {
	ctx := context.Background()
	state := &fakeStateStore{data: &vo.OAuthState{RedirectURL: "https://closy.hycat.online/cb", CodeVerifier: "v"}}
	uc := newGoogleUCWithConfig(googleRedirectCfg(), new(mockUserRepo), &fakeSession{}, state)

	result, err := uc.HandleCallback(ctx, "", "state-1", "access_denied")
	assert.NoError(t, err)
	assert.Equal(t, "access_denied", result.ErrorCode)
	assert.Equal(t, "https://closy.hycat.online/cb", result.RedirectURL)
	assert.Nil(t, result.Tokens)
}

func TestGoogleCallback_MissingCode(t *testing.T) {
	ctx := context.Background()
	state := &fakeStateStore{data: &vo.OAuthState{RedirectURL: "https://closy.hycat.online/cb"}}
	uc := newGoogleUCWithConfig(googleRedirectCfg(), new(mockUserRepo), &fakeSession{}, state)

	result, err := uc.HandleCallback(ctx, "", "state-1", "")
	assert.NoError(t, err)
	assert.Equal(t, "exchange_failed", result.ErrorCode)
}
