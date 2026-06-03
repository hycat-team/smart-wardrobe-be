package jwttype

type JwtType string

const (
	AccessToken        JwtType = "access_token"
	RefreshToken       JwtType = "refresh_token"
	ResetPasswordToken JwtType = "reset_password_token"
)
