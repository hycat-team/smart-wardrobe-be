package benefit

import (
	"context"
	"time"

	"smart-wardrobe-be/internal/modules/brand/domain/entities"

	"github.com/google/uuid"
)

type UnlockStrategy interface {
	CheckAccess(ctx context.Context, benefit *entities.BrandBenefit, customer *entities.BrandCustomer, account *entities.LoyaltyAccount, now time.Time) (bool, error)
	Redeem(ctx context.Context, benefit *entities.BrandBenefit, customer *entities.BrandCustomer, userID uuid.UUID, now time.Time) (*entities.BenefitRedemption, error)
}
