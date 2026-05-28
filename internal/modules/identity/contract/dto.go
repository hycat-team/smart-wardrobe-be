package contract

import (
	"github.com/google/uuid"
)

// PublicUserDTO holds basic identity information for external system interfaces
type PublicUserDTO struct {
	ID       uuid.UUID
	Username string
	Email    string
	RoleSlug string
}
