package handler

import (
	"net/http"
	"net/url"
	"time"

	"smart-wardrobe-be/services/account/dto/identity"
	identityerrors "smart-wardrobe-be/services/account/internal/modules/identity/application/errors"
	usecase_interfaces "smart-wardrobe-be/services/account/internal/modules/identity/application/interface/usecase"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"
	"smart-wardrobe-be/shared/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/shared/presentation"

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
	successResendRegisterOtp       = "Đã gửi lại mã OTP đăng kí thành công. Vui lòng kiểm tra email."
	successResendForgotPasswordOtp = "Đã gửi lại mã OTP khôi phục mật khẩu thành công. Vui lòng kiểm tra email."
)

type AuthHandler struct {
	registerUC usecase_interfaces.IRegisterUseCase
	sessionUC  usecase_interfaces.ISessionUseCase
	recoveryUC usecase_interfaces.IPasswordRecoveryUseCase
	googleUC   usecase_interfaces.IGoogleAuthUseCase
	cfg        *config.Config
}

func NewAuthHandler(
	registerUC usecase_interfaces.IRegisterUseCase,
	sessionUC usecase_interfaces.ISessionUseCase,
	recoveryUC usecase_interfaces.IPasswordRecoveryUseCase,
	googleUC usecase_interfaces.IGoogleAuthUseCase,
	cfg *config.Config,
) *AuthHandler {
	return &AuthHandler{
		registerUC: registerUC,
		sessionUC:  sessionUC,
		recoveryUC: recoveryUC,
		googleUC:   googleUC,
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
// @Success 200 {object} shared_pres.APIResponse{data=dto.TokenRes} "Đăng nhập thành công, trả token qua JSON body và cookie"
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) error {
	var input dto.LoginReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	if input.DeviceIP == nil && clientIP != "" {
		input.DeviceIP = &clientIP
	}
	if input.UserAgent == nil && userAgent != "" {
		input.UserAgent = &userAgent
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

	shared_pres.Success(c, successLogin, output)
	return nil
}

// GoogleLogin authenticates a user with a Google ID token.
// @Summary Đăng nhập bằng Google (ID token)
// @Description Xác thực Google ID token và cấp phiên làm việc (dành cho mobile và tuỳ chọn web)
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.GoogleLoginReq true "Google ID token"
// @Success 200 {object} shared_pres.APIResponse{data=dto.TokenRes} "Đăng nhập thành công"
// @Router /api/v1/auth/google [post]
func (h *AuthHandler) GoogleLogin(c *gin.Context) error {
	var input dto.GoogleLoginReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	if input.DeviceIP == nil && clientIP != "" {
		input.DeviceIP = &clientIP
	}
	if input.UserAgent == nil && userAgent != "" {
		input.UserAgent = &userAgent
	}

	output, err := h.googleUC.LoginWithIDToken(c.Request.Context(), input)
	if err != nil {
		return err
	}

	h.setAuthCookies(c, output)
	shared_pres.Success(c, successLogin, output)
	return nil
}

// GoogleStart starts the web Authorization Code redirect flow.
// @Summary Bắt đầu đăng nhập Google (web redirect)
// @Description Validate redirectUrl và chuyển hướng sang Google
// @Tags Auth
// @Param redirectUrl query string true "URL quay về sau đăng nhập"
// @Router /api/v1/auth/google [get]
func (h *AuthHandler) GoogleStart(c *gin.Context) error {
	var input dto.GoogleStartReq
	if err := validation.BindQuery(c, &input); err != nil {
		return err
	}

	authURL, err := h.googleUC.StartRedirect(c.Request.Context(), input.RedirectURL)
	if err != nil {
		return err
	}

	c.Redirect(http.StatusFound, authURL)
	return nil
}

// GoogleCallback handles Google's callback for the web redirect flow.
// @Summary Callback đăng nhập Google (web redirect)
// @Description Đổi code, tạo phiên và chuyển hướng về redirectUrl
// @Tags Auth
// @Param code query string false "Authorization code"
// @Param state query string true "OAuth state"
// @Param error query string false "Lỗi do Google trả về"
// @Router /api/v1/auth/google/callback [get]
func (h *AuthHandler) GoogleCallback(c *gin.Context) error {
	result, err := h.googleUC.HandleCallback(
		c.Request.Context(),
		c.Query("code"),
		c.Query("state"),
		c.Query("error"),
	)
	if err != nil {
		return err
	}

	if result.ErrorCode != "" {
		c.Redirect(http.StatusFound, appendQueryError(result.RedirectURL, result.ErrorCode))
		return nil
	}

	h.setAuthCookies(c, result.Tokens)
	c.Redirect(http.StatusFound, result.RedirectURL)
	return nil
}

// setAuthCookies writes the access/refresh HttpOnly cookies for web clients.
// Google login/callback uses SameSite=Lax because the OAuth callback is a
// cross-site top-level navigation; Strict cookies are dropped in that context.
func (h *AuthHandler) setAuthCookies(c *gin.Context, output *dto.TokenRes) {
	if output == nil {
		return
	}
	refreshExpirySeconds := int((time.Hour * 24 * time.Duration(h.cfg.Jwt.RefreshExpirationDays)).Seconds())
	accessExpirySeconds := h.cfg.Jwt.AccessExpirationMinutes * 60

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(contextutils.CookieRefreshToken, output.RefreshToken, refreshExpirySeconds, "/", "", true, true)
	c.SetCookie(contextutils.CookieAccessToken, output.AccessToken, accessExpirySeconds, "/", "", true, true)
}

func appendQueryError(rawURL string, code string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	query := parsed.Query()
	query.Set("error", code)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

// Logout user logout
// @Summary Đăng xuất
// @Description Đăng xuất người dùng, vô hiệu hóa token hiện tại và xóa cookie refresh token & access token
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.LogoutReq false "Refresh token khi không dùng cookie (dành cho Mobile)"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) error {
	accessToken, err := contextutils.GetAccessToken(c)
	if err != nil {
		return err
	}

	refreshToken, err := c.Cookie(contextutils.CookieRefreshToken)
	if err != nil || len(refreshToken) == 0 {
		var req dto.LogoutReq
		if bindErr := c.ShouldBindJSON(&req); bindErr == nil && len(req.RefreshToken) > 0 {
			refreshToken = req.RefreshToken
		}
	}

	if len(refreshToken) == 0 {
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
// @Description Sử dụng refresh token trong cookie hoặc JSON body (nếu không có cookie) để lấy access token mới và xoay vòng refresh token
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.RefreshTokenReq false "Thông tin xoay vòng token khi không dùng cookie (dành cho Mobile)"
// @Success 200 {object} shared_pres.APIResponse{data=dto.TokenRes} "Xoay vòng token thành công. Token mới được trả về qua JSON body và Set-Cookie headers."
// @Router /api/v1/auth/refresh-token [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) error {
	var input dto.RefreshTokenReq
	oldRefreshToken, err := c.Cookie(contextutils.CookieRefreshToken)
	if err == nil && len(oldRefreshToken) > 0 {
		input.OldRefreshToken = oldRefreshToken
	} else {
		if bindErr := c.ShouldBindJSON(&input); bindErr != nil || len(input.OldRefreshToken) == 0 {
			return identityerrors.ErrCookieTokenMissing()
		}
	}

	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	if input.DeviceIP == nil && clientIP != "" {
		input.DeviceIP = &clientIP
	}
	if input.UserAgent == nil && userAgent != "" {
		input.UserAgent = &userAgent
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

	shared_pres.Success(c, successRefreshToken, output)
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
// @Description Xác thực mã OTP khôi phục mật khẩu, lưu token tạm thời vào cookie và trả về trong JSON body (dành cho Mobile)
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.ConfirmForgotPasswordOtpReq true "Mã OTP và Email"
// @Success 200 {object} shared_pres.APIResponse{data=dto.ConfirmForgotPasswordOtpRes}
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

	shared_pres.Success(c, successConfirmForgotPassword, dto.ConfirmForgotPasswordOtpRes{
		ResetToken: resetToken,
	})
	return nil
}

// ResetPassword reset user password
// @Summary Đặt lại mật khẩu
// @Description Sử dụng token tạm thời từ cookie hoặc JSON body (nếu không có cookie) để tiến hành đặt mật khẩu mới
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body dto.ResetPasswordReq true "Mật khẩu mới, xác nhận và resetToken tùy chọn nếu dùng mobile"
// @Success 200 {object} shared_pres.APIResponse
// @Router /api/v1/auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) error {
	var input dto.ResetPasswordReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	resetToken, err := c.Cookie(contextutils.CookieForgotPasswordToken)
	if err != nil || len(resetToken) == 0 {
		if input.ResetToken != nil && len(*input.ResetToken) > 0 {
			resetToken = *input.ResetToken
		} else {
			return identityerrors.ErrTokenExpiredRecovery()
		}
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
