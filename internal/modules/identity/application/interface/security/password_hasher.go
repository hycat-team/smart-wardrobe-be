package security

type IPasswordHasher interface {
	HashPassword(password string) (string, error)
	VerifyPassword(password, hashedPassword string) bool
}
