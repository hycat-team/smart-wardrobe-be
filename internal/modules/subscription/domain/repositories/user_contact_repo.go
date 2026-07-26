package repositories

import (
	"context"
	"github.com/google/uuid"
)

type IUserContactRepository interface {
	GetEmailByUserID(ctx context.Context, userID uuid.UUID) (string, error)
}
