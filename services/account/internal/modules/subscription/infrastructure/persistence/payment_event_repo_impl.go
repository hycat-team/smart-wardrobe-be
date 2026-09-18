package persistence

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/infrastructure/db"
)

type PaymentEventRepository struct{ db *gorm.DB }

func NewPaymentEventRepository(conn *gorm.DB) repositories.IPaymentEventRepository {
	return &PaymentEventRepository{db: conn}
}
func (r *PaymentEventRepository) current(ctx context.Context) *gorm.DB {
	if tx := db.GetTx(ctx); tx != nil {
		return tx
	}
	return r.db.WithContext(ctx)
}
func (r *PaymentEventRepository) Create(ctx context.Context, event *entities.ProviderPaymentEvent) error {
	return r.current(ctx).Create(event).Error
}
func (r *PaymentEventRepository) GetByReference(ctx context.Context, provider, reference string) (*entities.ProviderPaymentEvent, error) {
	var event entities.ProviderPaymentEvent
	err := r.current(ctx).Where("provider = ? AND provider_reference = ?", provider, reference).First(&event).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &event, err
}
