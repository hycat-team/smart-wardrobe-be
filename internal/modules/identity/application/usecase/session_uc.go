package usecase

import (
	"context"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/identity/application/dto"
	"smart-wardrobe-be/internal/modules/identity/application/interface/security"
	uc_interfaces "smart-wardrobe-be/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/identity/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"smart-wardrobe-be/internal/shared/domain/constants/jwttype"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/pkg/utils/jwtutils"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type SessionUseCase struct {
	userRepo              repositories.IUserRepository
	refreshTokenRepo      repositories.IRefreshTokenRepository
	passwordHasher        security.IPasswordHasher
	tokenBlacklistService security.ITokenBlacklistService
	cfg                   *config.Config
}

func NewSessionUseCase(
	userRepo repositories.IUserRepository,
	refreshTokenRepo repositories.IRefreshTokenRepository,
	passwordHasher security.IPasswordHasher,
	tokenBlacklistService security.ITokenBlacklistService,
	cfg *config.Config,
) uc_interfaces.ISessionUseCase {
	return &SessionUseCase{
		userRepo:              userRepo,
		refreshTokenRepo:      refreshTokenRepo,
		passwordHasher:        passwordHasher,
		tokenBlacklistService: tokenBlacklistService,
		cfg:                   cfg,
	}
}

func (uc *SessionUseCase) Login(ctx context.Context, input dto.LoginReq) (*dto.TokenRes, error) {
	user, err := uc.userRepo.GetByUsernameOrEmail(ctx, input.LoginName)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errorcode.NewBadRequest("Sai tài khoản hoặc mật khẩu.")
	}

	isValid := uc.passwordHasher.VerifyPassword(input.Password, user.PasswordHash)
	if !isValid {
		return nil, errorcode.NewBadRequest("Sai tài khoản hoặc mật khẩu.")
	}

	accessExpiry := time.Minute * time.Duration(uc.cfg.Jwt.AccessExpirationMinutes)
	refreshExpiry := time.Hour * 24 * time.Duration(uc.cfg.Jwt.RefreshExpirationDays)

	accessToken, err := jwtutils.GenerateToken(
		user.ID, user.Email, string(user.RoleSlug),
		jwttype.AccessToken,
		uc.cfg.Jwt.Secret, uc.cfg.Jwt.Issuer, uc.cfg.Jwt.Audience,
		accessExpiry,
	)
	if err != nil {
		return nil, errorcode.NewInternalError("Lỗi khi cấp phiên làm việc")
	}

	refreshToken, err := jwtutils.GenerateToken(
		user.ID, user.Email, string(user.RoleSlug),
		jwttype.RefreshToken,
		uc.cfg.Jwt.Secret, uc.cfg.Jwt.Issuer, uc.cfg.Jwt.Audience,
		refreshExpiry,
	)
	if err != nil {
		return nil, errorcode.NewInternalError("Lỗi khi cấp phiên làm việc")
	}

	rt := &entities.RefreshToken{
		UserID:    user.ID,
		Token:     refreshToken,
		ExpiresAt: time.Now().Add(refreshExpiry),
		IsRevoked: false,
	}
	rt.ID = uuid.New()

	err = uc.refreshTokenRepo.Create(ctx, rt)
	if err != nil {
		return nil, err
	}

	return &dto.TokenRes{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (uc *SessionUseCase) RefreshToken(ctx context.Context, input dto.RefreshTokenReq) (*dto.TokenRes, error) {
	claims, err := jwtutils.ValidateToken([]byte(uc.cfg.Jwt.Secret), input.OldRefreshToken, jwttype.RefreshToken)
	if err != nil {
		return nil, errorcode.NewUnauthorized("Phiên làm việc không hợp lệ. Vui lòng đăng nhập lại.")
	}

	oldExpiresAtUtc := claims.ExpiresAt.Time
	remainingTime := time.Until(oldExpiresAtUtc)
	if remainingTime <= 0 {
		return nil, errorcode.NewUnauthorized("Phiên làm việc đã hết hạn. Vui lòng đăng nhập lại.")
	}

	userId, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, errorcode.NewUnauthorized("Phiên làm việc không hợp lệ. Vui lòng đăng nhập lại.")
	}

	user, err := uc.userRepo.GetByID(ctx, userId)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsDeleted {
		return nil, errorcode.NewUnauthorized("Không tìm thấy người dùng này.")
	}

	existingToken, err := uc.refreshTokenRepo.GetByToken(ctx, input.OldRefreshToken)
	if err != nil {
		return nil, err
	}
	if existingToken == nil || existingToken.IsRevoked {
		return nil, errorcode.NewUnauthorized("Phiên làm việc không hợp lệ. Vui lòng đăng nhập lại.")
	}

	accessExpiry := time.Minute * time.Duration(uc.cfg.Jwt.AccessExpirationMinutes)

	newAccessToken, err := jwtutils.GenerateToken(
		user.ID, user.Email, string(user.RoleSlug),
		jwttype.AccessToken,
		uc.cfg.Jwt.Secret, uc.cfg.Jwt.Issuer, uc.cfg.Jwt.Audience,
		accessExpiry,
	)
	if err != nil {
		return nil, errorcode.NewInternalError("Lỗi khi cấp phiên làm việc")
	}

	newRefreshToken, err := jwtutils.GenerateToken(
		user.ID, user.Email, string(user.RoleSlug),
		jwttype.RefreshToken,
		uc.cfg.Jwt.Secret, uc.cfg.Jwt.Issuer, uc.cfg.Jwt.Audience,
		remainingTime,
	)
	if err != nil {
		return nil, errorcode.NewInternalError("Lỗi khi cấp phiên làm việc")
	}

	err = uc.refreshTokenRepo.RevokeToken(ctx, input.OldRefreshToken)
	if err != nil {
		return nil, err
	}

	rt := &entities.RefreshToken{
		UserID:    user.ID,
		Token:     newRefreshToken,
		ExpiresAt: time.Now().Add(remainingTime),
		IsRevoked: false,
	}
	rt.ID = uuid.New()

	err = uc.refreshTokenRepo.Create(ctx, rt)
	if err != nil {
		return nil, err
	}

	return &dto.TokenRes{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (uc *SessionUseCase) Logout(ctx context.Context, input dto.LogoutReq) (bool, error) {
	claims, err := jwtutils.ValidateToken([]byte(uc.cfg.Jwt.Secret), input.RefreshToken, jwttype.RefreshToken)
	if err != nil {
		return false, errorcode.NewUnauthorized("Phiên làm việc không hợp lệ.")
	}

	userId, err := uuid.Parse(claims.Subject)
	if err != nil {
		return false, errorcode.NewUnauthorized("Phiên làm việc không hợp lệ.")
	}

	user, err := uc.userRepo.GetByID(ctx, userId)
	if err != nil {
		return false, err
	}
	if user == nil || user.IsDeleted {
		return false, errorcode.NewUnauthorized("Không tìm thấy người dùng.")
	}

	err = uc.refreshTokenRepo.RevokeToken(ctx, input.RefreshToken)
	if err != nil {
		return false, err
	}

	// Calculate remaining access token lifetime
	tokenHandler := &jwt.Parser{}
	tokenClaims := &jwtutils.CustomClaims{}
	_, _, err = tokenHandler.ParseUnverified(input.AccessToken, tokenClaims)
	if err == nil && tokenClaims.ExpiresAt != nil {
		expTime := tokenClaims.ExpiresAt.Time
		remainingTime := time.Until(expTime)
		if remainingTime > 0 {
			_ = uc.tokenBlacklistService.BlacklistToken(ctx, input.AccessToken, remainingTime)
		}
	}

	return true, nil
}
