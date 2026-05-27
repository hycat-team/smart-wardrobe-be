package otpconstants

import "fmt"

const (
	PurposeForgotPassword = "forgot-password"
	PurposeRegistration   = "registration"
	PurposeTwoFactorAuth   = "2fa"

	KeyValue     = "value"
	KeyAttempts = "attempts"
	KeyCooldown = "cooldown"
	KeyData     = "data"
)

func BuildKey(purpose, keyType, email string) string {
	return fmt.Sprintf("otp:%s:%s:%s", purpose, keyType, email)
}
