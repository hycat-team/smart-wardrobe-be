package security

import (
	"errors"
	"smart-wardrobe-be/internal/modules/identity/application/interface/security"

	"golang.org/x/crypto/bcrypt"
)

type BcryptPasswordHasher struct{}

func NewBcryptPasswordHasher() security.IPasswordHasher {
	return &BcryptPasswordHasher{}
}

func (h *BcryptPasswordHasher) HashPassword(password string) (string, error) {
	if len(password) == 0 {
		return "", errors.New("vui lòng nhập mật khẩu")
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), 11)
	if err != nil {
		return "", err
	}

	return string(hashedBytes), nil
}

func (h *BcryptPasswordHasher) VerifyPassword(password, hashedPassword string) bool {
	if len(password) == 0 || len(hashedPassword) == 0 {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
