package admin

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/entities"
	"smart-wardrobe-be/services/account/internal/modules/subscription/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/apperror"

	"github.com/google/uuid"
)

// fakeDepositRepo embeds the full interface and overrides only GetByID: the
// admin use case touches nothing else for these cases.
type fakeDepositRepo struct {
	repositories.IDepositTransactionRepository
	byID *entities.DepositTransaction
}

func (f fakeDepositRepo) GetByID(_ context.Context, _ uuid.UUID) (*entities.DepositTransaction, error) {
	return f.byID, nil
}

// The generic repository reports a miss as (nil, nil); the admin contract
// promises 404 (029 data-model §8). Without the use-case check the handler
// answered 200 with an empty body.
func TestGetTransactionMissMapsToNotFound(t *testing.T) {
	uc := NewAdminBillingUseCase(fakeDepositRepo{byID: nil})

	tx, err := uc.GetTransaction(context.Background(), uuid.New())
	if tx != nil {
		t.Fatalf("expected nil transaction, got %+v", tx)
	}

	var appErr *apperror.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.Error, got %T: %v", err, err)
	}
	if appErr.Status != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", appErr.Status)
	}
}

func TestGetTransactionHitReturnsTheEntity(t *testing.T) {
	want := &entities.DepositTransaction{OrderCode: 4242}
	uc := NewAdminBillingUseCase(fakeDepositRepo{byID: want})

	got, err := uc.GetTransaction(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got != want {
		t.Fatalf("expected the repository entity back, got %+v", got)
	}
}
