package handler

import (
	"net/http"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/identity/application/dto"
	identityerrors "smart-wardrobe-be/internal/modules/identity/application/errors"
	usecase_interfaces "smart-wardrobe-be/internal/modules/identity/application/interface/usecase"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/validation"

	"github.com/gin-gonic/gin"
)

// Success messages for AuthHandler
const (
	successRegister                = "Đã nhận thông tin đăng kí. Vui lòng kiểm tra email để lấy OTP xác thực."
	successConfirmRegisterOtp      = "Xác thực tài khoản thành công."
	successLogin                   = "Đăng nhập thành công"
	successLogout                  = "Đăng xuất thành công"
	successRefreshToken            = "Xoay vòng token thành công"
	successForgotPassword          = "Yêu cầu khôi phục mật khẩu thành công. Vui lòng kiểm tra email để lấy OTP xác thực."
	successConfirmForgotPassword   = "Xác thực OTP thành công"
	successResetPassword           = "Đặt lại mật khẩu thành công"
	successResendRegisterOtp        = "Đã gửi lại mã OTP đăng kí thành công. Vui lòng kiểm tra email."
	successResendForgotPasswordOtp = "Đã gửi lại mã OTP khôi phục mật khẩu thành công. Vui lòng kiểm tra email."
)

type AuthHandler struct {
	registerUC usecase_interfaces.IRegisterUseCase
	sessionUC  usecase_interfaces.ISessionUseCase
	recoveryUC usecase_interfaces.IPasswordRecoveryUseCase
	cfg        *config.Config
}

func NewAuthHandler(
	registerUC usecase_interfaces.IRegisterUseCase,
	sessionUC usecase_interfaces.ISessionUseCase,
	recoveryUC usecase_interfaces.IPasswordRecoveryUseCase,
	cfg *config.Config,
) *AuthHandler {
	return &AuthHandler{
		registerUC: registerUC,
		sessionUC:  sessionUC,
		recoveryUC: recoveryUC,
		cfg:        cfg,
	}
}

// Register register a new user account
// @Summary Đăng ký tài khoản
// @Description Đăng ký tài khoản mới cho người dùng và gửi OTP xác thực qua email
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.RegisterReq true "Thông tin đăng ký"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) error {
	var input dto.RegisterReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	_, err := h.registerUC.Register(c.Request.Context(), input)
	if err != nil {
		return err
	}

	shared_pres.Success(c, successRegister, nil)
	return nil
}

// ConfirmRegisterOtp confirm user registration with OTP
// @Summary Xác thực OTP đăng ký
// @Description Xác nhận OTP gửi qua email để kích hoạt tài khoản
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.ConfirmRegisterOtpReq true "Mã OTP xác thực"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/auth/register/confirm-otp [post]
func (h *AuthHandler) ConfirmRegisterOtp(c *gin.Context) error {
	var input dto.ConfirmRegisterOtpReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	_, err := h.registerUC.ConfirmRegisterOtp(c.Request.Context(), input)
	if err != nil {
		return err
	}

	shared_pres.Success(c, successConfirmRegisterOtp, nil)
	return nil
}

// ResendRegisterOtp resend registration verification OTP
// @Summary Gửi lại OTP đăng ký
// @Description Gửi lại mã OTP xác thực đăng ký qua email dựa vào địa chỉ email đã đăng ký trước đó
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.ResendOtpReq true "Email nhận lại OTP"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/auth/register/resend-otp [post]
func (h *AuthHandler) ResendRegisterOtp(c *gin.Context) error {
	var input dto.ResendOtpReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	_, err := h.registerUC.ResendRegisterOtp(c.Request.Context(), input)
	if err != nil {
		return err
	}

	shared_pres.Success(c, successResendRegisterOtp, nil)
	return nil
}


// Login user login
// @Summary Đăng nhập
// @Description Đăng nhập hệ thống bằng username hoặc email và password
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.LoginReq true "Thông tin đăng nhập"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) error {
	var input dto.LoginReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	output, err := h.sessionUC.Login(c.Request.Context(), input)
	if err != nil {
		return err
	}

	// Calculate cookie durations in seconds
	refreshExpirySeconds := int((time.Hour * 24 * time.Duration(h.cfg.Jwt.RefreshExpirationDays)).Seconds())
	accessExpirySeconds := h.cfg.Jwt.AccessExpirationMinutes * 60

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		contextutils.CookieRefreshToken,
		output.RefreshToken,
		refreshExpirySeconds,
		"/",
		"",
		true,
		true,
	)
	c.SetCookie(
		contextutils.CookieAccessToken,
		output.AccessToken,
		accessExpirySeconds,
		"/",
		"",
		true,
		true,
	)

	shared_pres.Success(c, successLogin, nil)
	return nil
}

// Logout user logout
// @Summary Đăng xuất
// @Description Đăng xuất người dùng, vô hiệu hóa token hiện tại và xóa cookie refresh token & access token
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) error {
	accessToken, err := contextutils.GetAccessToken(c)
	if err != nil {
		return err
	}

	refreshToken, err := c.Cookie(contextutils.CookieRefreshToken)
	if err != nil || len(refreshToken) == 0 {
		return identityerrors.ErrCookieTokenMissing()
	}

	input := dto.LogoutReq{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	_, err = h.sessionUC.Logout(c.Request.Context(), input)
	if err != nil {
		return err
	}

	// Delete client refresh_token and access_token cookies
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		contextutils.CookieRefreshToken,
		"",
		-1,
		"/",
		"",
		true,
		true,
	)
	c.SetCookie(
		contextutils.CookieAccessToken,
		"",
		-1,
		"/",
		"",
		true,
		true,
	)

	shared_pres.Success(c, successLogout, nil)
	return nil
}

// RefreshToken refresh JWT token
// @Summary Xoay vòng token (Refresh Token)
// @Description Sử dụng refresh token trong cookie để lấy access token mới và xoay vòng refresh token
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=map[string]string} "accessToken trong dữ liệu"
// @Router /api/v1/auth/refresh-token [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) error {
	oldRefreshToken, err := c.Cookie(contextutils.CookieRefreshToken)
	if err != nil || len(oldRefreshToken) == 0 {
		return identityerrors.ErrCookieTokenMissing()
	}

	input := dto.RefreshTokenReq{
		OldRefreshToken: oldRefreshToken,
	}

	output, err := h.sessionUC.RefreshToken(c.Request.Context(), input)
	if err != nil {
		return err
	}

	// Calculate cookie durations in seconds
	refreshExpirySeconds := int((time.Hour * 24 * time.Duration(h.cfg.Jwt.RefreshExpirationDays)).Seconds())
	accessExpirySeconds := h.cfg.Jwt.AccessExpirationMinutes * 60

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		contextutils.CookieRefreshToken,
		output.RefreshToken,
		refreshExpirySeconds,
		"/",
		"",
		true,
		true,
	)
	c.SetCookie(
		contextutils.CookieAccessToken,
		output.AccessToken,
		accessExpirySeconds,
		"/",
		"",
		true,
		true,
	)

	shared_pres.Success(c, successRefreshToken, nil)
	return nil
}

// ForgotPassword request password reset OTP
// @Summary Yêu cầu khôi phục mật khẩu
// @Description Gửi mã OTP xác thực khôi phục mật khẩu qua email
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.SendForgotPasswordOtpReq true "Email nhận OTP"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) error {
	var input dto.SendForgotPasswordOtpReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	_, err := h.recoveryUC.SendForgotPasswordOtp(c.Request.Context(), input)
	if err != nil {
		return err
	}

	shared_pres.Success(c, successForgotPassword, nil)
	return nil
}

// ResendForgotPasswordOtp resend forgot password verification OTP
// @Summary Gửi lại OTP khôi phục mật khẩu
// @Description Gửi lại mã OTP xác thực khôi phục mật khẩu qua email dựa vào địa chỉ email đã gửi yêu cầu trước đó
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.ResendOtpReq true "Email nhận lại OTP"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/auth/forgot-password/resend-otp [post]
func (h *AuthHandler) ResendForgotPasswordOtp(c *gin.Context) error {
	var input dto.ResendOtpReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	_, err := h.recoveryUC.ResendForgotPasswordOtp(c.Request.Context(), input)
	if err != nil {
		return err
	}

	shared_pres.Success(c, successResendForgotPasswordOtp, nil)
	return nil
}


// ConfirmForgotPasswordOtp confirm forgot password OTP
// @Summary Xác thực OTP khôi phục mật khẩu
// @Description Xác thực mã OTP khôi phục mật khẩu và lưu token tạm thời vào cookie
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.ConfirmForgotPasswordOtpReq true "Mã OTP và Email"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/auth/forgot-password/confirm-otp [post]
func (h *AuthHandler) ConfirmForgotPasswordOtp(c *gin.Context) error {
	var input dto.ConfirmForgotPasswordOtpReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	resetToken, err := h.recoveryUC.ConfirmForgotPasswordOtp(c.Request.Context(), input)
	if err != nil {
		return err
	}

	expirySeconds := h.cfg.Jwt.ForgotPasswordExpirationMinutes * 60

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		contextutils.CookieForgotPasswordToken,
		resetToken,
		expirySeconds,
		"/",
		"",
		true,
		true,
	)

	shared_pres.Success(c, successConfirmForgotPassword, nil)
	return nil
}

// ResetPassword reset user password
// @Summary Đặt lại mật khẩu
// @Description Sử dụng token tạm thời từ cookie để tiến hành đặt mật khẩu mới
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.ResetPasswordReq true "Mật khẩu mới và mật khẩu xác nhận"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) error {
	var input dto.ResetPasswordReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	resetToken, err := c.Cookie(contextutils.CookieForgotPasswordToken)
	if err != nil || len(resetToken) == 0 {
		return identityerrors.ErrTokenExpiredRecovery()
	}

	_, err = h.recoveryUC.ResetPassword(c.Request.Context(), input, resetToken)
	if err != nil {
		return err
	}

	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(
		contextutils.CookieForgotPasswordToken,
		"",
		-1,
		"/",
		"",
		true,
		true,
	)

	shared_pres.Success(c, successResetPassword, nil)
	return nil
}
