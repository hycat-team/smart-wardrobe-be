package loyalty

import (
	"context"
	"testing"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/benefit/benefitstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/benefit/benefittype"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/benefit/benefitunlocktype"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/loyaltypointlotstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/loyaltytransactiontype"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/repositories"
	"smart-wardrobe-be/shared/observability/workerlog"

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
func (r *loyaltyLotsAccountRepo) GetByBrandCustomerIDs(ctx context.Context, brandCustomerIDs []uuid.UUID) ([]*entities.LoyaltyAccount, error) {
	var accounts []*entities.LoyaltyAccount
	for _, brandCustomerID := range brandCustomerIDs {
		for _, account := range r.accounts {
			if account.BrandCustomerID == brandCustomerID {
				accounts = append(accounts, account)
			}
		}
	}
	return accounts, nil
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
func (r *loyaltyLotsAccountRepo) CountByTier(ctx context.Context, brandID uuid.UUID) ([]repositories.TierMemberCount, error) {
	return nil, nil
}
func (r *loyaltyLotsAccountRepo) CountBySpendSegment(ctx context.Context, brandID uuid.UUID) ([]repositories.SpendSegmentCount, error) {
	return nil, nil
}
func (r *loyaltyLotsAccountRepo) BatchUpdate(ctx context.Context, entities []*entities.LoyaltyAccount) error {
	return nil
}
func (r *loyaltyLotsAccountRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}
func (r *loyaltyLotsAccountRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.LoyaltyAccount, error) {
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
func (r *loyaltyLotsLotRepo) ListByAccountID(ctx context.Context, loyaltyAccountID uuid.UUID, status *loyaltypointlotstatus.LoyaltyPointLotStatus, expiresAt *time.Time, page int, limit int) ([]*entities.LoyaltyPointLot, error) {
	var lots []*entities.LoyaltyPointLot
	for _, lot := range r.lots {
		if lot.LoyaltyAccountID != loyaltyAccountID {
			continue
		}
		if status != nil && lot.Status != *status {
			continue
		}
		if expiresAt != nil && (lot.ExpiresAt == nil || !lot.ExpiresAt.Before(*expiresAt)) {
			continue
		}
		lots = append(lots, lot)
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
func (r *loyaltyLotsLotRepo) SumExpiringPointsByMonth(ctx context.Context, brandID uuid.UUID, status loyaltypointlotstatus.LoyaltyPointLotStatus) ([]repositories.MonthPointSum, error) {
	return nil, nil
}
func (r *loyaltyLotsLotRepo) BatchUpdate(ctx context.Context, entities []*entities.LoyaltyPointLot) error {
	return nil
}
func (r *loyaltyLotsLotRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}
func (r *loyaltyLotsLotRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.LoyaltyPointLot, error) {
	return nil, nil
}
func (r *loyaltyLotsLotRepo) GetNearestExpiringActiveLots(ctx context.Context, loyaltyAccountIDs []uuid.UUID, now time.Time) (map[uuid.UUID]*entities.LoyaltyPointLot, error) {
	return nil, nil
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
func (r *loyaltyLotsTxRepo) GetByLoyaltyAccountIDPaginated(ctx context.Context, filter repositories.LoyaltyTransactionFilter) (*repositories.LoyaltyTransactionListResult, error) {
	return nil, nil
}
func (r *loyaltyLotsTxRepo) BatchUpdate(ctx context.Context, entities []*entities.LoyaltyPointTransaction) error {
	return nil
}
func (r *loyaltyLotsTxRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}
func (r *loyaltyLotsTxRepo) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*entities.LoyaltyPointTransaction, error) {
	return nil, nil
}

func newLoyaltyLotsTestUseCase(account *entities.LoyaltyAccount, lots ...*entities.LoyaltyPointLot) (*BrandLoyaltyUseCase, *loyaltyLotsAccountRepo, *loyaltyLotsLotRepo, *loyaltyLotsTxRepo) {
	accountRepo := &loyaltyLotsAccountRepo{accounts: map[uuid.UUID]*entities.LoyaltyAccount{account.ID: account}}
	lotRepo := &loyaltyLotsLotRepo{lots: map[uuid.UUID]*entities.LoyaltyPointLot{}}
	for _, lot := range lots {
		lotRepo.lots[lot.ID] = lot
	}
	txRepo := &loyaltyLotsTxRepo{}
	uc := &BrandLoyaltyUseCase{
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

	expiredPoints, err := uc.ProcessExpiredLoyaltyPointLots(context.Background(), workerlog.New("test", "test"), now, 100)
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

	expiredPoints, err = uc.ProcessExpiredLoyaltyPointLots(context.Background(), workerlog.New("test", "test"), now, 100)
	if err != nil {
		t.Fatal(err)
	}
	if expiredPoints != 0 || accountRepo.accounts[account.ID].CurrentPoints != 30 || len(txRepo.transactions) != 1 {
		t.Fatalf("expected idempotent second run, got expired=%d balance=%d tx=%d", expiredPoints, accountRepo.accounts[account.ID].CurrentPoints, len(txRepo.transactions))
	}
}

type tierTestBrandRepo struct {
	repositories.IBrandRepository
	brand *entities.Brand
}

func (r tierTestBrandRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.Brand, error) {
	return r.brand, nil
}

type tierTestTierRepo struct {
	repositories.ILoyaltyTierRepository
	tiers []*entities.LoyaltyTier
}

func (r tierTestTierRepo) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.LoyaltyTier, error) {
	return r.tiers, nil
}

func (r tierTestTierRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.LoyaltyTier, error) {
	for _, tier := range r.tiers {
		if tier.ID == id {
			return tier, nil
		}
	}
	return nil, nil
}

type tierTestBenefitRepo struct {
	repositories.IBrandBenefitRepository
	benefits []*entities.BrandBenefit
}

func (r tierTestBenefitRepo) GetActiveByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandBenefit, error) {
	return r.benefits, nil
}

func (r tierTestBenefitRepo) CountActiveByTierIDs(ctx context.Context, brandID uuid.UUID, tierIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	counts := make(map[uuid.UUID]int64, len(tierIDs))
	for _, benefit := range r.benefits {
		if benefit.BrandID != brandID || benefit.Status != benefitstatus.Active || benefit.RequiredTierID == nil {
			continue
		}
		for _, tierID := range tierIDs {
			if *benefit.RequiredTierID == tierID {
				counts[tierID]++
			}
		}
	}
	return counts, nil
}
func newTierTestBenefit(name string, requiredTier *entities.LoyaltyTier) *entities.BrandBenefit {
	return &entities.BrandBenefit{
		Name:           name,
		UnlockType:     benefitunlocktype.TierPrivilege,
		BenefitType:    benefittype.Voucher,
		RequiredTierID: &requiredTier.ID,
		RequiredTier:   requiredTier,
	}
}

func buildTierTestUseCase(brandStatus brandstatus.BrandStatus, tiers []*entities.LoyaltyTier, benefits []*entities.BrandBenefit) *BrandLoyaltyUseCase {
	return &BrandLoyaltyUseCase{
		brandRepo:   tierTestBrandRepo{brand: &entities.Brand{Status: brandStatus}},
		tierRepo:    tierTestTierRepo{tiers: tiers},
		benefitRepo: tierTestBenefitRepo{benefits: benefits},
	}
}

func TestCumulativeTierBenefits(t *testing.T) {
	brandID := uuid.New()
	bronze := &entities.LoyaltyTier{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}}, BrandID: brandID, Name: "Bronze", Rank: 1}
	silver := &entities.LoyaltyTier{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}}, BrandID: brandID, Name: "Silver", Rank: 2}
	gold := &entities.LoyaltyTier{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}}, BrandID: brandID, Name: "Gold", Rank: 3}

	benefits := []*entities.BrandBenefit{
		newTierTestBenefit("bronze-perk", bronze),
		newTierTestBenefit("silver-perk", silver),
		newTierTestBenefit("gold-perk", gold),
	}

	if got := len(cumulativeTierBenefits(benefits, bronze.Rank)); got != 1 {
		t.Fatalf("expected 1 benefit at rank 1, got %d", got)
	}
	if got := len(cumulativeTierBenefits(benefits, silver.Rank)); got != 2 {
		t.Fatalf("expected 2 benefits at rank 2, got %d", got)
	}
	if got := len(cumulativeTierBenefits(benefits, gold.Rank)); got != 3 {
		t.Fatalf("expected 3 benefits at rank 3, got %d", got)
	}
}

func TestGetLoyaltyTiersForUserReturnsCumulativeBenefits(t *testing.T) {
	brandID := uuid.New()
	bronze := &entities.LoyaltyTier{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}}, BrandID: brandID, Name: "Bronze", Rank: 1, MinTotalSpend: 0}
	silver := &entities.LoyaltyTier{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}}, BrandID: brandID, Name: "Silver", Rank: 2, MinTotalSpend: 1000000}
	gold := &entities.LoyaltyTier{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}}, BrandID: brandID, Name: "Gold", Rank: 3, MinTotalSpend: 3000000}

	uc := buildTierTestUseCase(
		brandstatus.Active,
		[]*entities.LoyaltyTier{bronze, silver, gold},
		[]*entities.BrandBenefit{
			newTierTestBenefit("bronze-perk", bronze),
			newTierTestBenefit("silver-perk", silver),
			newTierTestBenefit("gold-perk", gold),
		},
	)

	res, err := uc.GetLoyaltyTiersForUser(context.Background(), brandID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 3 {
		t.Fatalf("expected 3 tiers, got %d", len(res))
	}
	for i, expected := range []int{1, 2, 3} {
		if len(res[i].Benefits) != expected {
			t.Fatalf("tier %s: expected %d benefits, got %d", res[i].Name, expected, len(res[i].Benefits))
		}
	}
}

func TestGetLoyaltyTierDetailForUserReturnsCumulativeBenefits(t *testing.T) {
	brandID := uuid.New()
	bronze := &entities.LoyaltyTier{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}}, BrandID: brandID, Name: "Bronze", Rank: 1}
	silver := &entities.LoyaltyTier{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}}, BrandID: brandID, Name: "Silver", Rank: 2}

	uc := buildTierTestUseCase(
		brandstatus.Active,
		[]*entities.LoyaltyTier{bronze, silver},
		[]*entities.BrandBenefit{
			newTierTestBenefit("bronze-perk", bronze),
			newTierTestBenefit("silver-perk", silver),
		},
	)

	res, err := uc.GetLoyaltyTierDetailForUser(context.Background(), brandID, silver.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Benefits) != 2 {
		t.Fatalf("expected 2 cumulative benefits for silver, got %d", len(res.Benefits))
	}
}

func TestGetLoyaltyTiersForStaff_CountsActiveBenefits(t *testing.T) {
	brandID := uuid.New()
	tier := &entities.LoyaltyTier{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}}, BrandID: brandID, Name: "Gold", Rank: 1}
	perk := newTierTestBenefit("gold perk", tier)
	perk.BrandID = brandID
	perk.Status = benefitstatus.Active

	uc := &BrandLoyaltyUseCase{
		tierRepo:    tierTestTierRepo{tiers: []*entities.LoyaltyTier{tier}},
		benefitRepo: tierTestBenefitRepo{benefits: []*entities.BrandBenefit{perk}},
	}

	res, err := uc.GetLoyaltyTiersForStaff(context.Background(), uuid.New(), brandID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 1 || res[0].BenefitsCount != 1 {
		t.Fatalf("expected benefitsCount 1, got %+v", res)
	}
}
