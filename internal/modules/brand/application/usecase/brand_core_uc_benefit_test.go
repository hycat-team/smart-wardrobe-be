package usecase

import (
	"context"
	"testing"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	"smart-wardrobe-be/internal/shared/domain/constants/benefit/benefitredemptionstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/benefit/benefitstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/benefit/benefittype"
	"smart-wardrobe-be/internal/shared/domain/constants/benefit/benefitunlocktype"
	"smart-wardrobe-be/internal/shared/domain/constants/brandcustomerstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brandmemberstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brandstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type mockBrandRepo struct {
	brands map[uuid.UUID]*entities.Brand
}

func (m *mockBrandRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.Brand, error) {
	return m.brands[id], nil
}
func (m *mockBrandRepo) GetAll(ctx context.Context) ([]*entities.Brand, error)   { return nil, nil }
func (m *mockBrandRepo) Create(ctx context.Context, brand *entities.Brand) error { return nil }
func (m *mockBrandRepo) Update(ctx context.Context, brand *entities.Brand) error { return nil }
func (m *mockBrandRepo) Delete(ctx context.Context, id uuid.UUID) error          { return nil }
func (m *mockBrandRepo) GetBySlug(ctx context.Context, slug string) (*entities.Brand, error) {
	return nil, nil
}
func (m *mockBrandRepo) GetActive(ctx context.Context) ([]*entities.Brand, error) { return nil, nil }

type mockMemberRepo struct {
	members map[string]*entities.BrandMember
}

func (m *mockMemberRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.BrandMember, error) {
	return nil, nil
}
func (m *mockMemberRepo) GetAll(ctx context.Context) ([]*entities.BrandMember, error) {
	return nil, nil
}
func (m *mockMemberRepo) Create(ctx context.Context, member *entities.BrandMember) error { return nil }
func (m *mockMemberRepo) Update(ctx context.Context, member *entities.BrandMember) error { return nil }
func (m *mockMemberRepo) Delete(ctx context.Context, id uuid.UUID) error                 { return nil }
func (m *mockMemberRepo) GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandMember, error) {
	return m.members[brandID.String()+"_"+userID.String()], nil
}
func (m *mockMemberRepo) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandMember, error) {
	return nil, nil
}

type mockCustomerRepo struct {
	customers map[string]*entities.BrandCustomer
}

func (m *mockCustomerRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.BrandCustomer, error) {
	return nil, nil
}
func (m *mockCustomerRepo) GetAll(ctx context.Context) ([]*entities.BrandCustomer, error) {
	return nil, nil
}
func (m *mockCustomerRepo) Create(ctx context.Context, customer *entities.BrandCustomer) error {
	return nil
}
func (m *mockCustomerRepo) Update(ctx context.Context, customer *entities.BrandCustomer) error {
	return nil
}
func (m *mockCustomerRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockCustomerRepo) GetByBrandAndUser(ctx context.Context, brandID uuid.UUID, userID uuid.UUID) (*entities.BrandCustomer, error) {
	return m.customers[brandID.String()+"_"+userID.String()], nil
}
func (m *mockCustomerRepo) GetByBrandAndPhoneHash(ctx context.Context, brandID uuid.UUID, phoneHash string) (*entities.BrandCustomer, error) {
	return nil, nil
}
func (m *mockCustomerRepo) GetByBrandAndExternalCode(ctx context.Context, brandID uuid.UUID, externalCustomerCode string) (*entities.BrandCustomer, error) {
	return nil, nil
}
func (m *mockCustomerRepo) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandCustomer, error) {
	return nil, nil
}

type mockBenefitRepo struct {
	benefits map[uuid.UUID]*entities.BrandBenefit
}

func (m *mockBenefitRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.BrandBenefit, error) {
	return m.benefits[id], nil
}
func (m *mockBenefitRepo) GetAll(ctx context.Context) ([]*entities.BrandBenefit, error) {
	return nil, nil
}
func (m *mockBenefitRepo) Create(ctx context.Context, benefit *entities.BrandBenefit) error {
	benefit.ID = uuid.New()
	benefit.CreatedAt = time.Now().UTC()
	benefit.UpdatedAt = time.Now().UTC()
	m.benefits[benefit.ID] = benefit
	return nil
}
func (m *mockBenefitRepo) Update(ctx context.Context, benefit *entities.BrandBenefit) error {
	m.benefits[benefit.ID] = benefit
	return nil
}
func (m *mockBenefitRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockBenefitRepo) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandBenefit, error) {
	var list []*entities.BrandBenefit
	for _, b := range m.benefits {
		if b.BrandID == brandID {
			list = append(list, b)
		}
	}
	return list, nil
}
func (m *mockBenefitRepo) GetActiveByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandBenefit, error) {
	var list []*entities.BrandBenefit
	for _, b := range m.benefits {
		if b.BrandID == brandID && b.Status == benefitstatus.Active {
			list = append(list, b)
		}
	}
	return list, nil
}

type mockRedemptionRepo struct {
	redemptions map[uuid.UUID]*entities.BenefitRedemption
}

func (m *mockRedemptionRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.BenefitRedemption, error) {
	return m.redemptions[id], nil
}
func (m *mockRedemptionRepo) GetAll(ctx context.Context) ([]*entities.BenefitRedemption, error) {
	return nil, nil
}
func (m *mockRedemptionRepo) Create(ctx context.Context, red *entities.BenefitRedemption) error {
	red.ID = uuid.New()
	red.CreatedAt = time.Now().UTC()
	red.UpdatedAt = time.Now().UTC()
	m.redemptions[red.ID] = red
	return nil
}
func (m *mockRedemptionRepo) Update(ctx context.Context, red *entities.BenefitRedemption) error {
	m.redemptions[red.ID] = red
	return nil
}
func (m *mockRedemptionRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockRedemptionRepo) GetByBrandCustomerID(ctx context.Context, brandCustomerID uuid.UUID) ([]*entities.BenefitRedemption, error) {
	return nil, nil
}
func (m *mockRedemptionRepo) GetActiveRedemptionByFeature(ctx context.Context, brandCustomerID uuid.UUID, featureCode string, now time.Time) (*entities.BenefitRedemption, error) {
	for _, r := range m.redemptions {
		if r.BrandCustomerID == brandCustomerID && r.Status == benefitredemptionstatus.Redeemed {
			if r.ExpiresAt == nil || r.ExpiresAt.After(now) {
				return r, nil
			}
		}
	}
	return nil, nil
}

type mockTierRepo struct {
	tiers map[uuid.UUID]*entities.LoyaltyTier
}

func (m *mockTierRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.LoyaltyTier, error) {
	return m.tiers[id], nil
}
func (m *mockTierRepo) GetAll(ctx context.Context) ([]*entities.LoyaltyTier, error) { return nil, nil }
func (m *mockTierRepo) Create(ctx context.Context, tier *entities.LoyaltyTier) error {
	m.tiers[tier.ID] = tier
	return nil
}
func (m *mockTierRepo) Update(ctx context.Context, tier *entities.LoyaltyTier) error { return nil }
func (m *mockTierRepo) Delete(ctx context.Context, id uuid.UUID) error               { return nil }
func (m *mockTierRepo) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.LoyaltyTier, error) {
	return nil, nil
}
func (m *mockTierRepo) GetHighestEligibleBySpend(ctx context.Context, brandID uuid.UUID, totalSpend float64) (*entities.LoyaltyTier, error) {
	return nil, nil
}

func TestCreateBrandBenefit(t *testing.T) {
	brandID := uuid.New()
	userID := uuid.New()

	brandRepo := &mockBrandRepo{brands: map[uuid.UUID]*entities.Brand{
		brandID: {
			Slug:   "closy",
			Name:   "Closy Brand",
			Status: brandstatus.Active,
		},
	}}

	memberRepo := &mockMemberRepo{members: map[string]*entities.BrandMember{
		brandID.String() + "_" + userID.String(): {
			BrandID: brandID,
			UserID:  userID,
			Role:    brandmemberrole.Owner,
			Status:  brandmemberstatus.Active,
		},
	}}

	benefitRepo := &mockBenefitRepo{benefits: make(map[uuid.UUID]*entities.BrandBenefit)}

	uc := &BrandCoreUseCase{
		brandRepo:   brandRepo,
		memberRepo:  memberRepo,
		benefitRepo: benefitRepo,
		tierRepo:    &mockTierRepo{tiers: make(map[uuid.UUID]*entities.LoyaltyTier)},
	}

	featCode := "SAMPLE_MIX_ACCESS"
	input := dto.CreateBrandBenefitReq{
		Name:           "Test Benefit",
		Description:    ptr("Test Desc"),
		BenefitType:    "FEATURE_ACCESS",
		UnlockType:     "POINT_REDEMPTION",
		RequiredPoints: intPtr(100),
		FeatureCode:    &featCode,
		FeatureConfig:  map[string]interface{}{"valid_duration_days": 30},
	}

	res, err := uc.CreateBrandBenefit(context.Background(), userID, brandID, input)
	if err != nil {
		t.Fatalf("Expected nil err, got %v", err)
	}

	if res.Name != "Test Benefit" {
		t.Errorf("Expected Test Benefit, got %s", res.Name)
	}
	if *res.RequiredPoints != 100 {
		t.Errorf("Expected 100 points, got %d", *res.RequiredPoints)
	}
}

func TestRedeemBenefit_InsufficientPoints(t *testing.T) {
	brandID := uuid.New()
	userID := uuid.New()
	benefitID := uuid.New()
	custID := uuid.New()

	brandRepo := &mockBrandRepo{brands: map[uuid.UUID]*entities.Brand{
		brandID: {
			Slug:   "closy",
			Name:   "Closy Brand",
			Status: brandstatus.Active,
		},
	}}

	customerRepo := &mockCustomerRepo{customers: map[string]*entities.BrandCustomer{
		brandID.String() + "_" + userID.String(): {
			AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: custID}},
			BrandID:         brandID,
			UserID:          &userID,
			Status:          brandcustomerstatus.Active,
		},
	}}

	benefitRepo := &mockBenefitRepo{benefits: map[uuid.UUID]*entities.BrandBenefit{
		benefitID: {
			BrandID:        brandID,
			Name:           "Test Benefit",
			BenefitType:    benefittype.FeatureAccess,
			UnlockType:     benefitunlocktype.PointRedemption,
			RequiredPoints: intPtr(100),
			Status:         benefitstatus.Active,
		},
	}}

	accountRepo := &loyaltyLotsAccountRepo{accounts: map[uuid.UUID]*entities.LoyaltyAccount{
		uuid.New(): {
			BrandID:         brandID,
			BrandCustomerID: custID,
			CurrentPoints:   50, // Insufficient points
		},
	}}

	uc := &BrandCoreUseCase{
		brandRepo:      brandRepo,
		customerRepo:   customerRepo,
		benefitRepo:    benefitRepo,
		accountRepo:    accountRepo,
		lotRepo:        &loyaltyLotsLotRepo{lots: make(map[uuid.UUID]*entities.LoyaltyPointLot)},
		redemptionRepo: &mockRedemptionRepo{redemptions: make(map[uuid.UUID]*entities.BenefitRedemption)},
		txRepo:         &mockTxRepo{transactions: make(map[uuid.UUID]*entities.LoyaltyPointTransaction)},
		uow:            loyaltyLotsTestUOW{},
	}

	_, err := uc.RedeemBenefit(context.Background(), userID, brandID, benefitID)
	if err == nil {
		t.Fatalf("Expected insufficient points error, got nil")
	}

	if err.Error() != branderrors.ErrInsufficientLoyaltyPoints().Error() {
		t.Errorf("Expected %v, got %v", branderrors.ErrInsufficientLoyaltyPoints(), err)
	}
}

type mockTxRepo struct {
	transactions map[uuid.UUID]*entities.LoyaltyPointTransaction
}

func (m *mockTxRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.LoyaltyPointTransaction, error) {
	return nil, nil
}
func (m *mockTxRepo) GetAll(ctx context.Context) ([]*entities.LoyaltyPointTransaction, error) {
	return nil, nil
}
func (m *mockTxRepo) Create(ctx context.Context, tx *entities.LoyaltyPointTransaction) error {
	m.transactions[tx.ID] = tx
	return nil
}
func (m *mockTxRepo) Update(ctx context.Context, tx *entities.LoyaltyPointTransaction) error {
	return nil
}
func (m *mockTxRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockTxRepo) GetByBrandAndIdempotencyKey(ctx context.Context, brandID uuid.UUID, idempotencyKey string) (*entities.LoyaltyPointTransaction, error) {
	return nil, nil
}
func (m *mockTxRepo) GetByLoyaltyAccountID(ctx context.Context, loyaltyAccountID uuid.UUID) ([]*entities.LoyaltyPointTransaction, error) {
	return nil, nil
}

func ptr[T any](v T) *T {
	return &v
}

func intPtr(v int) *int {
	return &v
}
