package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	subscription_contract "smart-wardrobe-be/services/account/contract/subscription"
	"smart-wardrobe-be/services/account/dto/identity"
	identityerrors "smart-wardrobe-be/services/account/internal/modules/identity/application/errors"
	identityiface "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/identity"
	appsecurity "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/security"
	uc_interfaces "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/services/account/internal/modules/identity/application/vo"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/authprovider"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/constants/userstatus"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/identity/domain/repositories"
	shared_event "smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/domain/constants/roleslug"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Web redirect error codes surfaced to the frontend via ?error=<code>.
const (
	webErrAccessDenied    = "access_denied"
	webErrEmailUnverified = "email_unverified"
	webErrEmailRegistered = "email_registered"
	webErrAccountLinked   = "account_linked"
	webErrAccountDisabled = "account_disabled"
	webErrExchangeFailed  = "exchange_failed"
	webErrServer          = "server_error"
)

type GoogleAuthUseCase struct {
	userRepo              repositories.IUserRepository
	sessionUC             uc_interfaces.ISessionUseCase
	verifier              appsecurity.IGoogleTokenVerifier
	stateStore            identityiface.IOAuthStateStore
	tokenBlacklistService appsecurity.ITokenBlacklistService
	cfg                   *config.Config
	uow                   shared_repos.IUnitOfWork
	// 024 T196: the narrow in-process subscription contract. Not nil-guarded on
	// purpose - a nil contract here would silently skip provisioning and put the
	// account back on the lazy path, which is the defect this task removes.
	provisioningContract subscription_contract.IUserSubscriptionProvisioningContract
	// 025 T027: emits `identity.user.created` inside the transaction that creates
	// the account. Not nil-guarded, for the same reason as above.
	eventPublisher shared_event.IEventPublisher
}

func NewGoogleAuthUseCase(
	userRepo repositories.IUserRepository,
	sessionUC uc_interfaces.ISessionUseCase,
	verifier appsecurity.IGoogleTokenVerifier,
	stateStore identityiface.IOAuthStateStore,
	tokenBlacklistService appsecurity.ITokenBlacklistService,
	cfg *config.Config,
	uow shared_repos.IUnitOfWork,
	provisioningContract subscription_contract.IUserSubscriptionProvisioningContract,
	eventPublisher shared_event.IEventPublisher,
) uc_interfaces.IGoogleAuthUseCase {
	return &GoogleAuthUseCase{
		userRepo:              userRepo,
		sessionUC:             sessionUC,
		verifier:              verifier,
		stateStore:            stateStore,
		tokenBlacklistService: tokenBlacklistService,
		cfg:                   cfg,
		uow:                   uow,
		provisioningContract:  provisioningContract,
		eventPublisher:        eventPublisher,
	}
}

func (uc *GoogleAuthUseCase) LoginWithIDToken(ctx context.Context, input dto.GoogleLoginReq) (*dto.TokenRes, error) {
	ident, _, err := uc.verify(ctx, input.IDToken)
	if err != nil {
		return nil, err
	}

	user, _, err := uc.resolveUser(ctx, ident)
	if err != nil {
		return nil, err
	}

	return uc.sessionUC.IssueSession(ctx, user, uc_interfaces.SessionDevice{
		Name:      input.DeviceName,
		IP:        input.DeviceIP,
		UserAgent: input.UserAgent,
	})
}

func (uc *GoogleAuthUseCase) StartRedirect(ctx context.Context, redirectURL string) (string, error) {
	validated, err := validateGoogleRedirectURL(uc.cfg, redirectURL)
	if err != nil {
		return "", err
	}
	if len(uc.cfg.Google.ClientIDs) == 0 || strings.TrimSpace(uc.cfg.Google.RedirectURI) == "" {
		return "", identityerrors.ErrGoogleConfigMissing()
	}

	state := randomHex(16)
	codeVerifier := oauth2.GenerateVerifier()
	ttl := time.Duration(uc.cfg.Google.StateTTLMinutes) * time.Minute

	if err := uc.stateStore.Save(ctx, state, vo.OAuthState{RedirectURL: validated, CodeVerifier: codeVerifier}, ttl); err != nil {
		return "", err
	}

	return uc.oauthConfig().AuthCodeURL(state, oauth2.S256ChallengeOption(codeVerifier)), nil
}

func (uc *GoogleAuthUseCase) HandleCallback(ctx context.Context, code string, state string, providerError string) (*uc_interfaces.GoogleRedirectResult, error) {
	stateData, err := uc.stateStore.Consume(ctx, state)
	if err != nil {
		return nil, err
	}
	if stateData == nil {
		return nil, identityerrors.ErrOAuthStateInvalid()
	}

	result := &uc_interfaces.GoogleRedirectResult{RedirectURL: stateData.RedirectURL}

	if providerError != "" {
		result.ErrorCode = webErrAccessDenied
		return result, nil
	}
	if strings.TrimSpace(code) == "" {
		result.ErrorCode = webErrExchangeFailed
		return result, nil
	}

	token, err := uc.oauthConfig().Exchange(ctx, code, oauth2.VerifierOption(stateData.CodeVerifier))
	if err != nil {
		result.ErrorCode = webErrExchangeFailed
		return result, nil
	}

	rawIDToken, _ := token.Extra("id_token").(string)
	ident, codeErr, err := uc.verify(ctx, rawIDToken)
	if err != nil {
		result.ErrorCode = codeErr
		return result, nil
	}

	user, codeErr, err := uc.resolveUser(ctx, ident)
	if err != nil {
		result.ErrorCode = codeErr
		return result, nil
	}

	tokens, err := uc.sessionUC.IssueSession(ctx, user, uc_interfaces.SessionDevice{})
	if err != nil {
		result.ErrorCode = webErrServer
		return result, nil
	}

	result.Tokens = tokens
	return result, nil
}

// verify validates the ID token and maps adapter errors to Vietnamese apperrors.
// It also returns the web redirect error code for the same failure.
func (uc *GoogleAuthUseCase) verify(ctx context.Context, idToken string) (*appsecurity.GoogleIdentity, string, error) {
	ident, err := uc.verifier.VerifyIDToken(ctx, idToken)
	if err != nil {
		if errors.Is(err, appsecurity.ErrGoogleConfigMissing) {
			return nil, webErrServer, identityerrors.ErrGoogleConfigMissing()
		}
		return nil, webErrExchangeFailed, identityerrors.ErrGoogleTokenInvalid()
	}
	if !ident.EmailVerified {
		return nil, webErrEmailUnverified, identityerrors.ErrGoogleEmailUnverified()
	}
	return ident, "", nil
}

// resolveUser implements the account resolution/auto-link rules (spec R3).
// It returns a web redirect error code alongside the apperror.
func (uc *GoogleAuthUseCase) resolveUser(ctx context.Context, ident *appsecurity.GoogleIdentity) (*entities.User, string, error) {
	existing, err := uc.userRepo.GetByProviderID(ctx, string(authprovider.Google), ident.Subject)
	if err != nil {
		return nil, webErrServer, err
	}
	if existing != nil {
		return uc.eligible(ctx, existing)
	}

	email := strings.ToLower(strings.TrimSpace(ident.Email))
	byEmail, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, webErrServer, err
	}
	if byEmail != nil {
		if byEmail.Provider == authprovider.Google {
			// Same email already bound to a different Google identity.
			return nil, webErrAccountLinked, identityerrors.ErrAccountLinkedOtherGoogle()
		}
		if !byEmail.EmailVerified {
			return nil, webErrEmailRegistered, identityerrors.ErrEmailRegisteredLoginPassword()
		}

		byEmail.LinkGoogleIdentity(ident.Subject)
		if err := uc.userRepo.Update(ctx, byEmail); err != nil {
			return nil, webErrServer, identityerrors.ErrAccountCreationFailed()
		}
		return uc.eligible(ctx, byEmail)
	}

	return uc.createGoogleUser(ctx, ident)
}

func (uc *GoogleAuthUseCase) createGoogleUser(ctx context.Context, ident *appsecurity.GoogleIdentity) (*entities.User, string, error) {
	username, err := uc.generateUsername(ctx, ident.Email)
	if err != nil {
		return nil, webErrServer, err
	}

	subject := ident.Subject
	firstName := nilIfBlank(ident.GivenName)
	lastName := nilIfBlank(ident.FamilyName)
	if firstName == nil && lastName == nil {
		firstName = nilIfBlank(ident.Name)
	}
	user := &entities.User{
		Username:      username,
		Email:         strings.ToLower(strings.TrimSpace(ident.Email)),
		Provider:      authprovider.Google,
		ProviderID:    &subject,
		EmailVerified: true,
		FirstName:     firstName,
		LastName:      lastName,
		RoleSlug:      roleslug.User,
		Status:        userstatus.Active,
	}
	user.ID = uuid.New()
	user.IsDeleted = false

	createFn := func(txCtx context.Context) error {
		if err := uc.userRepo.Create(txCtx, user); err != nil {
			return err
		}
		if err := uc.ensureDefaultSubscription(txCtx, user.ID); err != nil {
			return err
		}
		// 025 T031: inside createFn, therefore inside `uow.Execute` — the event
		// commits with the account or not at all (FR-042).
		//
		// The auto-link door returns above, in resolveUser, and never reaches
		// here: linking a Google identity to an existing account does not create
		// one, so it must not emit an event (contract §7, scenario 4).
		return publishUserCreated(txCtx, uc.eventPublisher, user, false)
	}
	if err := uc.uow.Execute(ctx, createFn); err != nil {
		// Concurrent first login: another request may have created the account.
		if again, ferr := uc.userRepo.GetByProviderID(ctx, string(authprovider.Google), subject); ferr == nil && again != nil {
			return uc.eligible(ctx, again)
		}
		return nil, webErrServer, identityerrors.ErrAccountCreationFailed()
	}

	return uc.eligible(ctx, user)
}

// 024 T196: provision the default plan inside the transaction that just created
// the user row, for the reason spelled out in CreateUserByAdmin - the
// subscription row has a foreign key onto the user, so it cannot be written from
// a second transaction. Idempotent, so a retried first login cannot produce a
// second record.
func (uc *GoogleAuthUseCase) ensureDefaultSubscription(ctx context.Context, userID uuid.UUID) error {
	return uc.provisioningContract.EnsureDefaultSubscription(ctx, userID)
}

// eligible applies the same account-status/blacklist checks as password login.
func (uc *GoogleAuthUseCase) eligible(ctx context.Context, user *entities.User) (*entities.User, string, error) {
	if err := ensureUserEligibleForLogin(user); err != nil {
		return nil, webErrAccountDisabled, err
	}
	if user.ID != uuid.Nil {
		blacklisted, err := uc.tokenBlacklistService.IsUserBlacklisted(ctx, user.ID)
		if err == nil && blacklisted {
			return nil, webErrAccountDisabled, identityerrors.ErrAccountDisabled()
		}
	}
	return user, "", nil
}

func (uc *GoogleAuthUseCase) generateUsername(ctx context.Context, email string) (string, error) {
	base := deriveUsernameBase(email)
	for i := 0; i < 10; i++ {
		candidate := base
		if i > 0 {
			candidate = withUsernameSuffix(base, "_"+randomHex(2))
		}
		exists, err := uc.userRepo.IsUsernameExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}

	for i := 0; i < 5; i++ {
		candidate := "user_" + randomHex(4)
		exists, err := uc.userRepo.IsUsernameExists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
	}

	return "", identityerrors.ErrAccountCreationFailed()
}

// oauthConfig builds the Google OAuth2 config. The first configured client ID is
// used for the web Authorization Code flow (the web client must be listed first).
func (uc *GoogleAuthUseCase) oauthConfig() *oauth2.Config {
	clientID := ""
	if len(uc.cfg.Google.ClientIDs) > 0 {
		clientID = uc.cfg.Google.ClientIDs[0]
	}
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: uc.cfg.Google.ClientSecret,
		RedirectURL:  uc.cfg.Google.RedirectURI,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

var _ uc_interfaces.IGoogleAuthUseCase = (*GoogleAuthUseCase)(nil)
