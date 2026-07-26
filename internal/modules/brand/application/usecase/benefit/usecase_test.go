package benefit

import (
	"context"
	"strings"
	"testing"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	branderrors "smart-wardrobe-be/internal/modules/brand/application/errors"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefit/benefitfeaturecode"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefit/benefitredemptionstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefit/benefitstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefit/benefittype"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefit/benefitunlocktype"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandcustomerstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberrole"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/loyaltypointlotstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/loyaltytransactiontype"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"

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
func (m *mockBrandRepo) GetActiveFiltered(ctx context.Context, filter repositories.BrandFilter) (*repositories.BrandListResult, error) {
	activeStatus := brandstatus.Active
	filter.Status = &activeStatus
	return m.GetBrandsForAdmin(ctx, filter)
}
func (m *mockBrandRepo) GetBrandsForAdmin(ctx context.Context, filter repositories.BrandFilter) (*repositories.BrandListResult, error) {
	var list []*entities.Brand
	for _, b := range m.brands {
		if filter.Status != nil && b.Status != *filter.Status {
			continue
		}
		if filter.Query != nil {
			q := strings.ToLower(*filter.Query)
			if !strings.Contains(strings.ToLower(b.Name), q) && !strings.Contains(strings.ToLower(b.Slug), q) {
				continue
			}
		}
		list = append(list, b)
	}
	return &repositories.BrandListResult{
		Brands:     list,
		TotalCount: int64(len(list)),
	}, nil
}
func (m *mockBrandRepo) GetAdminBrandStats(ctx context.Context, snapshotDate time.Time) (*dto.BrandAdminStatsDTO, error) {
	return nil, nil
}
func (m *mockBrandRepo) GetBrandAnalyticsStats(ctx context.Context, brandID uuid.UUID, snapshotDate time.Time) (*dto.BrandAnalyticsStatsDTO, error) {
	return nil, nil
}
func (m *mockBrandRepo) GetSampleLabFeedbacksDrillDown(ctx context.Context, itemID uuid.UUID, page, limit int) ([]dto.SampleFeedbackDrillDownDTO, int64, error) {
	return nil, 0, nil
}
func (m *mockBrandRepo) BatchUpdate(ctx context.Context, entities []*entities.Brand) error {
	return nil
}
func (m *mockBrandRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

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
func (m *mockMemberRepo) GetByBrandAndUserIDs(ctx context.Context, brandID uuid.UUID, userIDs []uuid.UUID) ([]*entities.BrandMember, error) {
	var members []*entities.BrandMember
	for _, userID := range userIDs {
		if member := m.members[brandID.String()+"_"+userID.String()]; member != nil {
			members = append(members, member)
		}
	}
	return members, nil
}
func (m *mockMemberRepo) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandMember, error) {
	return nil, nil
}
func (m *mockMemberRepo) GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.BrandMember, error) {
	var members []*entities.BrandMember
	for _, member := range m.members {
		if member.UserID == userID && member.Status == brandmemberstatus.Active {
			members = append(members, member)
		}
	}
	return members, nil
}
func (m *mockMemberRepo) BulkCreate(ctx context.Context, members []*entities.BrandMember) error {
	return nil
}
func (m *mockMemberRepo) BatchUpdate(ctx context.Context, members []*entities.BrandMember) error {
	return nil
}
func (m *mockMemberRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
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
func (m *mockCustomerRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.BrandCustomer, error) {
	var list []*entities.BrandCustomer
	for _, cust := range m.customers {
		if cust.UserID != nil && *cust.UserID == userID {
			list = append(list, cust)
		}
	}
	return list, nil
}
func (m *mockCustomerRepo) GetByBrandIDPaginated(ctx context.Context, filter repositories.BrandCustomerFilter) (*repositories.BrandCustomerListResult, error) {
	return nil, nil
}
func (m *mockCustomerRepo) CountByBrandID(ctx context.Context, brandID uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockCustomerRepo) CountByBrandIDs(ctx context.Context, brandIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	return map[uuid.UUID]int64{}, nil
}
func (m *mockCustomerRepo) BatchUpdate(ctx context.Context, entities []*entities.BrandCustomer) error {
	return nil
}
func (m *mockCustomerRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
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
func (m *mockBenefitRepo) BatchUpdate(ctx context.Context, entities []*entities.BrandBenefit) error {
	return nil
}
func (m *mockBenefitRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
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
func (m *mockRedemptionRepo) GetByBrandCustomerIDs(ctx context.Context, brandCustomerIDs []uuid.UUID) ([]*entities.BenefitRedemption, error) {
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
func (m *mockRedemptionRepo) GetUserExpiringBenefitsCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockRedemptionRepo) BatchUpdate(ctx context.Context, entities []*entities.BenefitRedemption) error {
	return nil
}
func (m *mockRedemptionRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
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
func (m *mockTierRepo) GetByBrandAndName(ctx context.Context, brandID uuid.UUID, name string) (*entities.LoyaltyTier, error) {
	return nil, nil
}
func (m *mockTierRepo) GetByBrandAndRank(ctx context.Context, brandID uuid.UUID, rank int) (*entities.LoyaltyTier, error) {
	return nil, nil
}
func (m *mockTierRepo) GetHighestEligibleBySpend(ctx context.Context, brandID uuid.UUID, totalSpend float64) (*entities.LoyaltyTier, error) {
	return nil, nil
}
func (m *mockTierRepo) BatchUpdate(ctx context.Context, entities []*entities.LoyaltyTier) error {
	return nil
}
func (m *mockTierRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
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

	uc := &BrandBenefitUseCase{
		brandRepo:   brandRepo,
		memberRepo:  memberRepo,
		benefitRepo: benefitRepo,
		tierRepo:    &mockTierRepo{tiers: make(map[uuid.UUID]*entities.LoyaltyTier)},
	}

	featCode := "sample_mix_access"
	input := dto.CreateBrandBenefitReq{
		Name:           "Test Benefit",
		Description:    ptr("Test Desc"),
		BenefitType:    "feature_access",
		UnlockType:     "point_redemption",
		RequiredPoints: intPtr(100),
		FeatureCode:    &featCode,
		FeatureConfig:  map[string]any{"validDurationDays": 30},
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

func TestCreateBrandBenefitTreatsZeroRequiredPointsAsNil(t *testing.T) {
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

	uc := &BrandBenefitUseCase{
		brandRepo:   brandRepo,
		memberRepo:  memberRepo,
		benefitRepo: benefitRepo,
		tierRepo:    &mockTierRepo{tiers: make(map[uuid.UUID]*entities.LoyaltyTier)},
	}

	featCode := "sample_mix_access"
	input := dto.CreateBrandBenefitReq{
		Name:           "Free Feature",
		BenefitType:    "feature_access",
		UnlockType:     "point_redemption",
		RequiredPoints: intPtr(0),
		FeatureCode:    &featCode,
	}

	res, err := uc.CreateBrandBenefit(context.Background(), userID, brandID, input)
	if err != nil {
		t.Fatalf("Expected nil err, got %v", err)
	}
	if res.RequiredPoints != nil {
		t.Fatalf("Expected nil required points, got %d", *res.RequiredPoints)
	}
}

func TestCreateBrandBenefitRejectsInvalidFeatureCode(t *testing.T) {
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

	uc := &BrandBenefitUseCase{
		brandRepo:   brandRepo,
		memberRepo:  memberRepo,
		benefitRepo: &mockBenefitRepo{benefits: make(map[uuid.UUID]*entities.BrandBenefit)},
		tierRepo:    &mockTierRepo{tiers: make(map[uuid.UUID]*entities.LoyaltyTier)},
	}

	typoFeatureCode := "sample_mix_acess"
	input := dto.CreateBrandBenefitReq{
		Name:           "Broken Feature",
		BenefitType:    "feature_access",
		UnlockType:     "point_redemption",
		RequiredPoints: intPtr(100),
		FeatureCode:    &typoFeatureCode,
	}

	if _, err := uc.CreateBrandBenefit(context.Background(), userID, brandID, input); err == nil {
		t.Fatal("Expected invalid feature code error, got nil")
	}
}

func TestCreateBrandBenefitRequiresFeatureCodeForFeatureAccess(t *testing.T) {
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

	uc := &BrandBenefitUseCase{
		brandRepo:   brandRepo,
		memberRepo:  memberRepo,
		benefitRepo: &mockBenefitRepo{benefits: make(map[uuid.UUID]*entities.BrandBenefit)},
		tierRepo:    &mockTierRepo{tiers: make(map[uuid.UUID]*entities.LoyaltyTier)},
	}

	input := dto.CreateBrandBenefitReq{
		Name:           "Missing Feature Code",
		BenefitType:    "feature_access",
		UnlockType:     "point_redemption",
		RequiredPoints: intPtr(100),
	}

	if _, err := uc.CreateBrandBenefit(context.Background(), userID, brandID, input); err == nil {
		t.Fatal("Expected missing feature code error, got nil")
	}
}

func TestRedeemBenefit_InsufficientPoints(t *testing.T) {
	brandID := uuid.New()
	userID := uuid.New()
	benefitID := uuid.New()
	custID := uuid.New()

	brandRepo := &mockBrandRepo{brands: map[uuid.UUID]*entities.Brand{
		brandID: {
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

	uc := &BrandBenefitUseCase{
		brandRepo:      brandRepo,
		customerRepo:   customerRepo,
		benefitRepo:    benefitRepo,
		accountRepo:    accountRepo,
		lotRepo:        &loyaltyLotsLotRepo{lots: make(map[uuid.UUID]*entities.LoyaltyPointLot)},
		redemptionRepo: &mockRedemptionRepo{redemptions: make(map[uuid.UUID]*entities.BenefitRedemption)},
		txRepo:         &mockTxRepo{transactions: make(map[uuid.UUID]*entities.LoyaltyPointTransaction)},
		uow:            loyaltyLotsTestUOW{},
	}

	_, err := uc.RedeemBenefit(context.Background(), userID, benefitID)
	if err == nil {
		t.Fatalf("Expected insufficient points error, got nil")
	}

	if err.Error() != branderrors.ErrInsufficientLoyaltyPoints().Error() {
		t.Errorf("Expected %v, got %v", branderrors.ErrInsufficientLoyaltyPoints(), err)
	}
}

func TestRedeemBenefitAllowsFreePointRedemption(t *testing.T) {
	brandID := uuid.New()
	userID := uuid.New()
	benefitID := uuid.New()
	custID := uuid.New()

	featCode := benefitfeaturecode.SampleMixAccess
	brandRepo := &mockBrandRepo{brands: map[uuid.UUID]*entities.Brand{
		brandID: {
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
			AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: benefitID}},
			BrandID:         brandID,
			Name:            "Free Sample Access",
			BenefitType:     benefittype.FeatureAccess,
			UnlockType:      benefitunlocktype.PointRedemption,
			FeatureCode:     &featCode,
			Status:          benefitstatus.Active,
		},
	}}
	redemptionRepo := &mockRedemptionRepo{redemptions: make(map[uuid.UUID]*entities.BenefitRedemption)}
	txRepo := &mockTxRepo{transactions: make(map[uuid.UUID]*entities.LoyaltyPointTransaction)}

	uc := &BrandBenefitUseCase{
		brandRepo:      brandRepo,
		customerRepo:   customerRepo,
		benefitRepo:    benefitRepo,
		redemptionRepo: redemptionRepo,
		txRepo:         txRepo,
		uow:            loyaltyLotsTestUOW{},
	}

	res, err := uc.RedeemBenefit(context.Background(), userID, benefitID)
	if err != nil {
		t.Fatalf("Expected nil err, got %v", err)
	}
	if res.PointsSpent != 0 {
		t.Fatalf("Expected 0 points spent, got %d", res.PointsSpent)
	}
	if len(txRepo.transactions) != 0 {
		t.Fatalf("Expected no loyalty point transactions, got %d", len(txRepo.transactions))
	}
	if len(redemptionRepo.redemptions) != 1 {
		t.Fatalf("Expected one redemption, got %d", len(redemptionRepo.redemptions))
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
func (m *mockTxRepo) GetByLoyaltyAccountIDPaginated(ctx context.Context, filter repositories.LoyaltyTransactionFilter) (*repositories.LoyaltyTransactionListResult, error) {
	return nil, nil
}
func (m *mockTxRepo) BatchUpdate(ctx context.Context, entities []*entities.LoyaltyPointTransaction) error {
	return nil
}
func (m *mockTxRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

func ptr[T any](v T) *T {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func newBenefitLotsTestUseCase(account *entities.LoyaltyAccount, lots ...*entities.LoyaltyPointLot) (*BrandBenefitUseCase, *loyaltyLotsAccountRepo, *loyaltyLotsLotRepo, *loyaltyLotsTxRepo) {
	accountRepo := &loyaltyLotsAccountRepo{accounts: map[uuid.UUID]*entities.LoyaltyAccount{account.ID: account}}
	lotRepo := &loyaltyLotsLotRepo{lots: map[uuid.UUID]*entities.LoyaltyPointLot{}}
	for _, lot := range lots {
		lotRepo.lots[lot.ID] = lot
	}
	txRepo := &loyaltyLotsTxRepo{}
	uc := &BrandBenefitUseCase{
		accountRepo: accountRepo,
		lotRepo:     lotRepo,
		txRepo:      txRepo,
		uow:         loyaltyLotsTestUOW{},
	}
	return uc, accountRepo, lotRepo, txRepo
}

func TestRedeemLoyaltyPointsExpiresDueLotsBeforeBalanceCheck(t *testing.T) {
	now := time.Now().UTC()
	account := &entities.LoyaltyAccount{AuditableEntity: entities.AuditableEntity{BaseEntity: entities.BaseEntity{ID: uuid.New()}}, BrandID: uuid.New(), BrandCustomerID: uuid.New(), CurrentPoints: 200}
	expiredAt := now.Add(-time.Hour)
	validAt := now.Add(time.Hour)
	uc, accountRepo, lotRepo, txRepo := newBenefitLotsTestUseCase(account,
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
func (r *loyaltyLotsAccountRepo) BatchUpdate(ctx context.Context, entities []*entities.LoyaltyAccount) error {
	return nil
}
func (r *loyaltyLotsAccountRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
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
func (r *loyaltyLotsLotRepo) BatchUpdate(ctx context.Context, entities []*entities.LoyaltyPointLot) error {
	return nil
}
func (r *loyaltyLotsLotRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
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
