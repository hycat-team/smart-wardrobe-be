package usecase

import (
	"context"
)

type IPaymentReconciliationUseCase interface {
	Reconcile(ctx context.Context) error
}
