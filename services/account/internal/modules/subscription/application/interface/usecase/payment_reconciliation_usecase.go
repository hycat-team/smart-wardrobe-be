package usecase

import (
	"context"
	"smart-wardrobe-be/shared/observability/workerlog"
)

type IPaymentReconciliationUseCase interface {
	Reconcile(ctx context.Context, run *workerlog.Run) error
}
