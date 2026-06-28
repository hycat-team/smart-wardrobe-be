package usecase

import (
	"context"
	"testing"
	"time"

	"smart-wardrobe-be/internal/shared/domain/constants/loyaltypointlotstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/loyaltytransactiontype"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type loyaltyLotsTestUOW struct{}

func (u loyaltyLotsTestUOW) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

type loyaltyLotsAccountRepo struct {
	accounts map[uuid.UUID]*entities.LoyaltyAccount
}

func (r *loyaltyLotsAccountRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.LoyaltyAccount, error) {
	return r.accounts[id], nil
}
func (r *loyaltyLotsAccountRepo) GetAll(ctx context.Context) ([]*entities.LoyaltyAccount, error) {
	return nil, nil
}
func (r *loyaltyLotsAccountRepo) Create(ctx context.Context, account *entities.LoyaltyAccount) error {
	r.accounts[account.ID] = account
	return nil
}
func (r *loyaltyLotsAccountRepo) Update(ctx context.Context, account *entities.LoyaltyAccount) error {
	r.accounts[account.ID] = account
	return nil
}
func (r *loyaltyLotsAccountRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (r *loyaltyLotsAccountRepo) GetByBrandCustomerID(ctx context.Context, brandCustomerID uuid.UUID) (*entities.LoyaltyAccount, error) {
	return nil, nil
}
func (r *loyaltyLotsAccountRepo) GetByBrandCustomerIDForUpdate(ctx context.Context, brandCustomerID uuid.UUID) (*entities.LoyaltyAccount, error) {
	for _, account := range r.accounts {
		if account.BrandCustomerID == brandCustomerID {
			return account, nil
		}
	}
	return nil, nil
}
func (r *loyaltyLotsAccountRepo) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*entities.LoyaltyAccount, error) {
	return r.accounts[id], nil
}
func (r *loyaltyLotsAccountRepo) GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.LoyaltyAccount, error) {
	return nil, nil
}

type loyaltyLotsLotRepo struct {
	lots map[uuid.UUID]*entities.LoyaltyPointLot
}

func (r *loyaltyLotsLotRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.LoyaltyPointLot, error) {
	return r.lots[id], nil
}
func (r *loyaltyLotsLotRepo) GetAll(ctx context.Context) ([]*entities.LoyaltyPointLot, error) {
	return nil, nil
}
func (r *loyaltyLotsLotRepo) Create(ctx context.Context, lot *entities.LoyaltyPointLot) error {
	r.lots[lot.ID] = lot
	return nil
}
func (r *loyaltyLotsLotRepo) Update(ctx context.Context, lot *entities.LoyaltyPointLot) error {
	r.lots[lot.ID] = lot
	return nil
}
func (r *loyaltyLotsLotRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (r *loyaltyLotsLotRepo) ListRedeemableLotsForUpdate(ctx context.Context, loyaltyAccountID uuid.UUID, now time.Time) ([]*entities.LoyaltyPointLot, error) {
	var lots []*entities.LoyaltyPointLot
	for _, lot := range r.lots {
		if lot.LoyaltyAccountID == loyaltyAccountID &&
			lot.Status == loyaltypointlotstatus.Active &&
			lot.RemainingPoints > 0 &&
			(lot.ExpiresAt == nil || lot.ExpiresAt.After(now)) {
			lots = append(lots, lot)
		}
	}
	return lots, nil
}
func (r *loyaltyLotsLotRepo) ListExpiredLotsForUpdate(ctx context.Context, loyaltyAccountID uuid.UUID, now time.Time) ([]*entities.LoyaltyPointLot, error) {
	var lots []*entities.LoyaltyPointLot
	for _, lot := range r.lots {
		if lot.LoyaltyAccountID == loyaltyAccountID &&
			lot.Status == loyaltypointlotstatus.Active &&
			lot.RemainingPoints > 0 &&
			lot.ExpiresAt != nil &&
			!lot.ExpiresAt.After(now) {
			lots = append(lots, lot)
		}
	}
	return lots, nil
}
func (r *loyaltyLotsLotRepo) UpdateLotRemainingAndStatus(ctx context.Context, lotID uuid.UUID, remainingPoints int, status loyaltypointlotstatus.LoyaltyPointLotStatus) error {
	r.lots[lotID].RemainingPoints = remainingPoints
	r.lots[lotID].Status = status
	return nil
}
func (r *loyaltyLotsLotRepo) ListAccountsWithExpiredLots(ctx context.Context, now time.Time, limit int) ([]uuid.UUID, error) {
	seen := map[uuid.UUID]bool{}
	var ids []uuid.UUID
	for _, lot := range r.lots {
		if len(ids) >= limit {
			break
		}
		if lot.Status == loyaltypointlotstatus.Active &&
			lot.RemainingPoints > 0 &&
			lot.ExpiresAt != nil &&
			!lot.ExpiresAt.After(now) &&
			!seen[lot.LoyaltyAccountID] {
			seen[lot.LoyaltyAccountID] = true
			ids = append(ids, lot.LoyaltyAccountID)
		}
	}
	return ids, nil
}
func (r *loyaltyLotsLotRepo) GetNearestExpiringActiveLot(ctx context.Context, loyaltyAccountID uuid.UUID, now time.Time) (*entities.LoyaltyPointLot, error) {
	var nearest *entities.LoyaltyPointLot
	for _, lot := range r.lots {
		if lot.LoyaltyAccountID != loyaltyAccountID ||
			lot.Status != loyaltypointlotstatus.Active ||
			lot.RemainingPoints <= 0 ||
			lot.ExpiresAt == nil ||
			!lot.ExpiresAt.After(now) {
			continue
		}
		if nearest == nil || lot.ExpiresAt.Before(*nearest.ExpiresAt) {
			nearest = lot
		}
	}
	return nearest, nil
}

type loyaltyLotsTxRepo struct {
	transactions []*entities.LoyaltyPointTransaction
}

func (r *loyaltyLotsTxRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.LoyaltyPointTransaction, error) {
	return nil, nil
}
func (r *loyaltyLotsTxRepo) GetAll(ctx context.Context) ([]*entities.LoyaltyPointTransaction, error) {
	return r.transactions, nil
}
func (r *loyaltyLotsTxRepo) Create(ctx context.Context, tx *entities.LoyaltyPointTransaction) error {
	if tx.ID == uuid.Nil {
		tx.ID = uuid.New()
	}
	r.transactions = append(r.transactions, tx)
	return nil
}
func (r *loyaltyLotsTxRepo) Update(ctx context.Context, tx *entities.LoyaltyPointTransaction) error {
	return nil
}
func (r *loyaltyLotsTxRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (r *loyaltyLotsTxRepo) GetByBrandAndIdempotencyKey(ctx context.Context, brandID uuid.UUID, idempotencyKey string) (*entities.LoyaltyPointTransaction, error) {
	return nil, nil
}
func (r *loyaltyLotsTxRepo) GetByLoyaltyAccountID(ctx context.Context, loyaltyAccountID uuid.UUID) ([]*entities.LoyaltyPointTransaction, error) {
	return r.transactions, nil
}

func newLoyaltyLotsTestUseCase(account *entities.LoyaltyAccount, lots ...*entities.LoyaltyPointLot) (*BrandCoreUseCase, *loyaltyLotsAccountRepo, *loyaltyLotsLotRepo, *loyaltyLotsTxRepo) {
	accountRepo := &loyaltyLotsAccountRepo{accounts: map[uuid.UUID]*entities.LoyaltyAccount{account.ID: account}}
	lotRepo := &loyaltyLotsLotRepo{lots: map[uuid.UUID]*entities.LoyaltyPointLot{}}
	for _, lot := range lots {
		lotRepo.lots[lot.ID] = lot
	}
	txRepo := &loyaltyLotsTxRepo{}
	uc := &BrandCoreUseCase{
		accountRepo: accountRepo,
		lotRepo:     lotRepo,
		txRepo:      txRepo,
		uow:         loyaltyLotsTestUOW{},
	}
	return uc, accountRepo, lotRepo, txRepo
}

func TestProcessExpiredLoyaltyPointLotsGroupsExpiredLotsAndIsIdempotent(t *testing.T) {
	now := time.Now().UTC()
	account := &entities.LoyaltyAccount{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}}, BrandID: uuid.New(), BrandCustomerID: uuid.New(), CurrentPoints: 80, LifetimePoints: 100, TotalSpend: 500000}
	expiredAt := now.Add(-time.Hour)
	uc, accountRepo, lotRepo, txRepo := newLoyaltyLotsTestUseCase(account,
		&entities.LoyaltyPointLot{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}}, LoyaltyAccountID: account.ID, RemainingPoints: 30, ExpiresAt: &expiredAt, Status: loyaltypointlotstatus.Active},
		&entities.LoyaltyPointLot{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}}, LoyaltyAccountID: account.ID, RemainingPoints: 20, ExpiresAt: &expiredAt, Status: loyaltypointlotstatus.Active},
	)

	expiredPoints, err := uc.ProcessExpiredLoyaltyPointLots(context.Background(), now, 100)
	if err != nil {
		t.Fatal(err)
	}
	if expiredPoints != 50 || accountRepo.accounts[account.ID].CurrentPoints != 30 {
		t.Fatalf("expected 50 expired and balance 30, got expired=%d balance=%d", expiredPoints, accountRepo.accounts[account.ID].CurrentPoints)
	}
	if len(txRepo.transactions) != 1 || txRepo.transactions[0].TransactionType != loyaltytransactiontype.Expire || txRepo.transactions[0].PointsDelta != -50 {
		t.Fatalf("expected one EXPIRE transaction -50, got %#v", txRepo.transactions)
	}
	for _, lot := range lotRepo.lots {
		if lot.Status != loyaltypointlotstatus.Expired || lot.RemainingPoints != 0 {
			t.Fatalf("expected lot expired with zero remaining, got status=%s remaining=%d", lot.Status, lot.RemainingPoints)
		}
	}

	expiredPoints, err = uc.ProcessExpiredLoyaltyPointLots(context.Background(), now, 100)
	if err != nil {
		t.Fatal(err)
	}
	if expiredPoints != 0 || accountRepo.accounts[account.ID].CurrentPoints != 30 || len(txRepo.transactions) != 1 {
		t.Fatalf("expected idempotent second run, got expired=%d balance=%d tx=%d", expiredPoints, accountRepo.accounts[account.ID].CurrentPoints, len(txRepo.transactions))
	}
}

func TestRedeemLoyaltyPointsExpiresDueLotsBeforeBalanceCheck(t *testing.T) {
	now := time.Now().UTC()
	account := &entities.LoyaltyAccount{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}}, BrandID: uuid.New(), BrandCustomerID: uuid.New(), CurrentPoints: 200}
	expiredAt := now.Add(-time.Hour)
	validAt := now.Add(time.Hour)
	uc, accountRepo, lotRepo, txRepo := newLoyaltyLotsTestUseCase(account,
		&entities.LoyaltyPointLot{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}}, LoyaltyAccountID: account.ID, RemainingPoints: 100, ExpiresAt: &expiredAt, Status: loyaltypointlotstatus.Active},
		&entities.LoyaltyPointLot{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}}, LoyaltyAccountID: account.ID, RemainingPoints: 100, ExpiresAt: &validAt, Status: loyaltypointlotstatus.Active},
	)

	if _, err := uc.redeemLoyaltyPointsFromLots(context.Background(), account.ID, 150, now, nil, nil, nil, nil); err == nil {
		t.Fatal("expected insufficient points after expiry")
	}
	if accountRepo.accounts[account.ID].CurrentPoints != 100 {
		t.Fatalf("expected balance 100 after on-demand expiry, got %d", accountRepo.accounts[account.ID].CurrentPoints)
	}
	if len(txRepo.transactions) != 1 || txRepo.transactions[0].TransactionType != loyaltytransactiontype.Expire {
		t.Fatalf("expected only EXPIRE transaction, got %#v", txRepo.transactions)
	}
	expiredCount := 0
	for _, lot := range lotRepo.lots {
		if lot.Status == loyaltypointlotstatus.Expired {
			expiredCount++
		}
	}
	if expiredCount != 1 {
		t.Fatalf("expected one expired lot, got %d", expiredCount)
	}
}
