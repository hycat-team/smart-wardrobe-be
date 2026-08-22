package item

import (
	"context"
	"strings"
	"testing"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/benefit/benefitfeaturecode"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandcustomerstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/branditemtype"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberrole"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandmemberstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/brandstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	fashion_dto "smart-wardrobe-be/internal/modules/fashion/application/dto"
	identity_dto "smart-wardrobe-be/internal/modules/identity/application/dto"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"

	"github.com/google/uuid"
)

type loyaltyLotsTestUOW struct{}

func (u loyaltyLotsTestUOW) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

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
func (m *mockCustomerRepo) CountByJoinedSource(ctx context.Context, brandID uuid.UUID) ([]repositories.CustomerSourceCount, error) {
	return nil, nil
}
func (m *mockCustomerRepo) CountNewByBrandID(ctx context.Context, brandID uuid.UUID, fromDate, toDate time.Time) (int64, error) {
	return 0, nil
}
func (m *mockCustomerRepo) CountClaimedByBrandID(ctx context.Context, brandID uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockCustomerRepo) BatchUpdate(ctx context.Context, entities []*entities.BrandCustomer) error {
	return nil
}
func (m *mockCustomerRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

type mockBrandItemRepo struct {
	items map[uuid.UUID]*entities.BrandItem
}

func (m *mockBrandItemRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.BrandItem, error) {
	if item, exists := m.items[id]; exists {
		return item, nil
	}
	return nil, nil
}
func (m *mockBrandItemRepo) GetAll(ctx context.Context) ([]*entities.BrandItem, error) {
	return nil, nil
}
func (m *mockBrandItemRepo) Create(ctx context.Context, item *entities.BrandItem) error {
	m.items[item.ID] = item
	return nil
}
func (m *mockBrandItemRepo) Update(ctx context.Context, item *entities.BrandItem) error {
	m.items[item.ID] = item
	return nil
}
func (m *mockBrandItemRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.items, id)
	return nil
}
func (m *mockBrandItemRepo) GetByProductCode(ctx context.Context, brandID uuid.UUID, code string) (*entities.BrandItem, error) {
	for _, item := range m.items {
		if item.BrandID == brandID && item.ProductCode != nil && *item.ProductCode == code {
			return item, nil
		}
	}
	return nil, nil
}
func (m *mockBrandItemRepo) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandItem, error) {
	var res []*entities.BrandItem
	for _, item := range m.items {
		if item.BrandID == brandID {
			res = append(res, item)
		}
	}
	return res, nil
}
func (m *mockBrandItemRepo) GetByFashionItemID(ctx context.Context, fashionItemID uuid.UUID) (*entities.BrandItem, error) {
	for _, item := range m.items {
		if item.FashionItemID != nil && *item.FashionItemID == fashionItemID {
			return item, nil
		}
	}
	return nil, nil
}
func (m *mockBrandItemRepo) GetByBrandIDs(ctx context.Context, brandIDs []uuid.UUID) ([]*entities.BrandItem, error) {
	allowed := make(map[uuid.UUID]struct{}, len(brandIDs))
	for _, brandID := range brandIDs {
		allowed[brandID] = struct{}{}
	}
	var res []*entities.BrandItem
	for _, item := range m.items {
		if _, ok := allowed[item.BrandID]; ok {
			res = append(res, item)
		}
	}
	return res, nil
}
func (m *mockBrandItemRepo) GetByFashionItemIDs(ctx context.Context, fashionItemIDs []uuid.UUID) ([]*entities.BrandItem, error) {
	allowed := make(map[uuid.UUID]struct{}, len(fashionItemIDs))
	for _, id := range fashionItemIDs {
		allowed[id] = struct{}{}
	}
	var res []*entities.BrandItem
	for _, item := range m.items {
		if item.FashionItemID != nil {
			if _, ok := allowed[*item.FashionItemID]; ok {
				res = append(res, item)
			}
		}
	}
	return res, nil
}
func (m *mockBrandItemRepo) GetByBrandIDPaginated(ctx context.Context, filter repositories.BrandItemFilter) (*repositories.BrandItemListResult, error) {
	var list []*entities.BrandItem
	for _, item := range m.items {
		if item.BrandID != filter.BrandID {
			continue
		}
		if filter.ItemType != nil && *filter.ItemType != "" && item.ItemType != *filter.ItemType {
			continue
		}
		if filter.Status != nil && *filter.Status != "" && item.Status != *filter.Status {
			continue
		}
		list = append(list, item)
	}
	return &repositories.BrandItemListResult{
		Items:      list,
		TotalCount: int64(len(list)),
	}, nil
}
func (m *mockBrandItemRepo) CountByStatus(ctx context.Context, brandID uuid.UUID) ([]repositories.BrandItemStatusCount, error) {
	return nil, nil
}
func (m *mockBrandItemRepo) CountByType(ctx context.Context, brandID uuid.UUID) ([]repositories.BrandItemTypeCount, error) {
	return nil, nil
}
func (m *mockBrandItemRepo) CountByBrandID(ctx context.Context, brandID uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *mockBrandItemRepo) BatchUpdate(ctx context.Context, entities []*entities.BrandItem) error {
	return nil
}
func (m *mockBrandItemRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

type mockDigitalSampleResponseRepo struct {
	feedbacks map[uuid.UUID]*entities.DigitalSampleResponse
}

func (m *mockDigitalSampleResponseRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.DigitalSampleResponse, error) {
	return nil, nil
}
func (m *mockDigitalSampleResponseRepo) GetAll(ctx context.Context) ([]*entities.DigitalSampleResponse, error) {
	return nil, nil
}
func (m *mockDigitalSampleResponseRepo) Create(ctx context.Context, f *entities.DigitalSampleResponse) error {
	f.ID = uuid.New()
	f.CreatedAt = time.Now().UTC()
	m.feedbacks[f.ID] = f
	return nil
}
func (m *mockDigitalSampleResponseRepo) Update(ctx context.Context, f *entities.DigitalSampleResponse) error {
	return nil
}
func (m *mockDigitalSampleResponseRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *mockDigitalSampleResponseRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*entities.DigitalSampleResponse, error) {
	var res []*entities.DigitalSampleResponse
	for _, f := range m.feedbacks {
		if f.UserID == userID {
			res = append(res, f)
		}
	}
	return res, nil
}
func (m *mockDigitalSampleResponseRepo) GetByBrandItemID(ctx context.Context, brandItemID uuid.UUID) ([]*entities.DigitalSampleResponse, error) {
	var res []*entities.DigitalSampleResponse
	for _, f := range m.feedbacks {
		if f.BrandItemID == brandItemID {
			res = append(res, f)
		}
	}
	return res, nil
}
func (m *mockDigitalSampleResponseRepo) BatchUpdate(ctx context.Context, entities []*entities.DigitalSampleResponse) error {
	return nil
}
func (m *mockDigitalSampleResponseRepo) BatchUpdateMap(ctx context.Context, ids []uuid.UUID, values map[string]any) error {
	return nil
}

type mockUserContract struct{}

func (m *mockUserContract) GetByID(ctx context.Context, id uuid.UUID) (*identity_dto.UserRes, error) {
	return &identity_dto.UserRes{ID: id, Username: "test_user"}, nil
}
func (m *mockUserContract) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]*identity_dto.UserRes, error) {
	return nil, nil
}
func (m *mockUserContract) GetByUsername(ctx context.Context, username string) (*identity_dto.UserRes, error) {
	return nil, nil
}
func (m *mockUserContract) GetByUsernamesOrEmails(ctx context.Context, identifiers []string) ([]*identity_dto.UserRes, error) {
	return nil, nil
}
func (m *mockUserContract) GetStyleProfile(ctx context.Context, userID uuid.UUID) (*identity_dto.UserStyleProfileRes, error) {
	return nil, nil
}

type fakeUnitOfWork struct{}

func (f fakeUnitOfWork) Execute(ctx context.Context, fn func(txCtx context.Context) error) error {
	return fn(ctx)
}

type mockFashionContract struct{}

func (m *mockFashionContract) GetFashionItem(ctx context.Context, id uuid.UUID) (*fashion_dto.FashionItemRes, error) {
	return &fashion_dto.FashionItemRes{
		ID:       id,
		ImageUrl: "http://example.com/image.jpg",
	}, nil
}
func (m *mockFashionContract) ListFashionItems(ctx context.Context, ids []uuid.UUID) ([]*fashion_dto.FashionItemRes, error) {
	var list []*fashion_dto.FashionItemRes
	for _, id := range ids {
		list = append(list, &fashion_dto.FashionItemRes{ID: id, ImageUrl: "http://example.com/image.jpg"})
	}
	return list, nil
}
func (m *mockFashionContract) ListFashionItemsBrief(ctx context.Context, ids []uuid.UUID) ([]*fashion_dto.FashionItemBriefRes, error) {
	var list []*fashion_dto.FashionItemBriefRes
	for _, id := range ids {
		list = append(list, &fashion_dto.FashionItemBriefRes{ID: id, ImageUrl: "http://example.com/image.jpg"})
	}
	return list, nil
}
func (m *mockFashionContract) GetCategory(ctx context.Context, id uuid.UUID) (*fashion_dto.CategoryRes, error) {
	return nil, nil
}
func (m *mockFashionContract) ListCategories(ctx context.Context, ids []uuid.UUID) ([]*fashion_dto.CategoryRes, error) {
	return nil, nil
}
func (m *mockFashionContract) GetAllCategories(ctx context.Context) ([]*fashion_dto.CategoryRes, error) {
	return nil, nil
}
func (m *mockFashionContract) GetCategoryBySlug(ctx context.Context, slug string) (*fashion_dto.CategoryRes, error) {
	return nil, nil
}
func (m *mockFashionContract) GetCategoryByName(ctx context.Context, name string) (*fashion_dto.CategoryRes, error) {
	return nil, nil
}
func (m *mockFashionContract) CreateCategory(ctx context.Context, input fashion_dto.CreateCategoryInput) (*fashion_dto.CategoryRes, error) {
	return nil, nil
}
func (m *mockFashionContract) UpdateCategory(ctx context.Context, id uuid.UUID, input fashion_dto.UpdateCategoryInput) (*fashion_dto.CategoryRes, error) {
	return nil, nil
}
func (m *mockFashionContract) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *mockFashionContract) GetFashionItemDirect(ctx context.Context, id uuid.UUID) (*fashion_dto.FashionItemRes, error) {
	return nil, nil
}
func (m *mockFashionContract) MapAnalyzeJobToFashionItemEntity(ctx context.Context, job *fashion_dto.FashionAnalyzeJobDTO) (any, error) {
	return nil, nil
}
func (m *mockFashionContract) CreateFashionItemDirect(ctx context.Context, item any) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockFashionContract) CreateFashionItem(ctx context.Context, input fashion_dto.CreateFashionItemInput) error {
	return nil
}
func (m *mockFashionContract) BulkCreateFashionItems(ctx context.Context, inputs []fashion_dto.CreateFashionItemInput) error {
	return nil
}
func (m *mockFashionContract) GetStaleProcessingInfo(ctx context.Context, fashionItemIDs []uuid.UUID) ([]fashion_dto.FashionProcessingInfoDTO, error) {
	return nil, nil
}
func (m *mockFashionContract) SearchFashionItemIDs(ctx context.Context, query *string, categorySlug *string) ([]uuid.UUID, error) {
	return nil, nil
}
func (m *mockFashionContract) GetFashionItemIDsByCategory(ctx context.Context, categoryID uuid.UUID) ([]uuid.UUID, error) {
	return nil, nil
}
func (m *mockFashionContract) ReassignCategoryForCatalogItems(ctx context.Context, fashionItemIDs []uuid.UUID, fromCategoryID uuid.UUID, toCategoryID uuid.UUID) error {
	return nil
}
func (m *mockFashionContract) ResetProcessingState(ctx context.Context, fashionItemID uuid.UUID) error {
	return nil
}
func (m *mockFashionContract) UpdateFashionItemCategory(ctx context.Context, fashionItemID uuid.UUID, categoryID uuid.UUID) error {
	return nil
}

type mockEventPublisher struct{}

func (m *mockEventPublisher) Publish(ctx context.Context, topic string, event any) error {
	return nil
}

type mockMediaService struct{}

func (m *mockMediaService) GenerateUploadSignature(ctx context.Context, params shared_dto.UploadSignatureParams) (*shared_dto.UploadSignatureResult, error) {
	return nil, nil
}
func (m *mockMediaService) UploadImage(ctx context.Context, file []byte, folder string) (string, string, error) {
	return "", "", nil
}
func (m *mockMediaService) DeleteImage(ctx context.Context, publicID string) error {
	return nil
}

type mockBenefitUseCase struct {
	hasAccess bool
}

func (m *mockBenefitUseCase) CheckBrandFeatureAccess(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, featureCode benefitfeaturecode.BenefitFeatureCode) (bool, error) {
	return m.hasAccess, nil
}
func (m *mockBenefitUseCase) CreateBrandBenefit(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.CreateBrandBenefitReq) (*dto.BrandBenefitRes, error) {
	return nil, nil
}
func (m *mockBenefitUseCase) ListBrandBenefitsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, query dto.ListBrandBenefitsQueryReq) (*dto.BrandBenefitListRes, error) {
	return nil, nil
}
func (m *mockBenefitUseCase) ListActiveBenefitsForUser(ctx context.Context, brandID uuid.UUID, query dto.ListBrandBenefitsQueryReq) (*dto.BrandBenefitListRes, error) {
	return nil, nil
}
func (m *mockBenefitUseCase) GetActiveBenefitForUser(ctx context.Context, userID uuid.UUID, benefitID uuid.UUID) (*dto.BrandBenefitRes, error) {
	return nil, nil
}
func (m *mockBenefitUseCase) ListBenefitRedemptionsForUser(ctx context.Context, userID uuid.UUID, brandID *uuid.UUID, page, limit int) (*dto.BenefitRedemptionListRes, error) {
	return nil, nil
}
func (m *mockBenefitUseCase) RedeemBenefit(ctx context.Context, userID uuid.UUID, benefitID uuid.UUID) (*dto.BenefitRedemptionRes, error) {
	return nil, nil
}
func (m *mockBenefitUseCase) UpdateBenefitStatus(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, benefitID uuid.UUID, status string) (*dto.BrandBenefitRes, error) {
	return nil, nil
}

func TestListEligibleBrandItemsForStyling(t *testing.T) {
	userID := uuid.New()
	brandID1 := uuid.New()
	brandID2 := uuid.New()

	brandRepo := &mockBrandRepo{brands: map[uuid.UUID]*entities.Brand{
		brandID1: {
			Slug:   "closy",
			Name:   "Closy Brand",
			Status: brandstatus.Active,
		},
		brandID2: {
			Slug:   "gucci",
			Name:   "Gucci Brand",
			Status: brandstatus.Active,
		},
	}}

	customerRepo := &mockCustomerRepo{customers: map[string]*entities.BrandCustomer{
		brandID1.String() + "_" + userID.String(): {
			BrandID: brandID1,
			UserID:  &userID,
			Status:  brandcustomerstatus.Active,
			Brand: &entities.Brand{
				Status: brandstatus.Active,
			},
		},
		brandID2.String() + "_" + userID.String(): {
			BrandID: brandID2,
			UserID:  &userID,
			Status:  brandcustomerstatus.Active,
			Brand: &entities.Brand{
				Status: brandstatus.Active,
			},
		},
	}}

	fashionID1 := uuid.New()
	fashionID2 := uuid.New()

	brandItemRepo := &mockBrandItemRepo{items: map[uuid.UUID]*entities.BrandItem{
		uuid.New(): {
			BrandID:       brandID1,
			FashionItemID: &fashionID1,
			ItemType:      branditemtype.Product,
			Status:        branditemstatus.Active,
		},
		uuid.New(): {
			BrandID:       brandID2,
			FashionItemID: &fashionID2,
			ItemType:      branditemtype.Sample,
			Status:        branditemstatus.Active,
		},
	}}

	uc := &BrandItemUseCase{
		brandRepo:       brandRepo,
		customerRepo:    customerRepo,
		brandItemRepo:   brandItemRepo,
		userContract:    &mockUserContract{},
		fashionContract: &mockFashionContract{},
		benefitUC:       &mockBenefitUseCase{hasAccess: true}, // Has access to samples
	}

	dtos, fashionItems, err := uc.ListEligibleBrandItemsForStyling(context.Background(), userID, &dto.ListEligibleBrandItemsReq{})
	if err != nil {
		t.Fatalf("Expected nil err, got %v", err)
	}

	if len(dtos) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(dtos))
	}
	if len(fashionItems) != 2 {
		t.Fatalf("Expected 2 fashion items, got %d", len(fashionItems))
	}

	// Now try without sample access
	uc.benefitUC = &mockBenefitUseCase{hasAccess: false}
	dtos2, fashionItems2, err := uc.ListEligibleBrandItemsForStyling(context.Background(), userID, &dto.ListEligibleBrandItemsReq{})
	if err != nil {
		t.Fatalf("Expected nil err, got %v", err)
	}
	if len(dtos2) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(dtos2))
	}
	if len(fashionItems2) != 1 {
		t.Fatalf("Expected 1 fashion item, got %d", len(fashionItems2))
	}
}

func TestBrandItemAndFeedbackFlow(t *testing.T) {
	ctx := context.Background()
	brandID := uuid.New()
	staffUserID := uuid.New()
	customerUserID := uuid.New()

	brandRepo := &mockBrandRepo{
		brands: map[uuid.UUID]*entities.Brand{
			brandID: {
				AuditableEntity: entities.AuditableEntity{
					BaseEntity: entities.BaseEntity{
						ID: brandID,
					},
				},
				Status: brandstatus.Active,
			},
		},
	}
	memberRepo := &mockMemberRepo{
		members: map[string]*entities.BrandMember{
			brandID.String() + "_" + staffUserID.String(): {
				BrandID: brandID,
				UserID:  staffUserID,
				Role:    brandmemberrole.Staff,
				Status:  brandmemberstatus.Active,
			},
		},
	}

	brandItemRepo := &mockBrandItemRepo{items: make(map[uuid.UUID]*entities.BrandItem)}
	feedbackRepo := &mockDigitalSampleResponseRepo{feedbacks: make(map[uuid.UUID]*entities.DigitalSampleResponse)}
	fashionContract := &mockFashionContract{}

	uc := &BrandItemUseCase{
		brandRepo:       brandRepo,
		memberRepo:      memberRepo,
		brandItemRepo:   brandItemRepo,
		feedbackRepo:    feedbackRepo,
		fashionContract: fashionContract,
		benefitUC:       &mockBenefitUseCase{hasAccess: true},
		eventPublisher:  &mockEventPublisher{},
		uow:             fakeUnitOfWork{},
	}

	// 1. Staff creates BrandItem
	productCode := "PC123"
	price := 120.0
	createInput := dto.CreateBrandItemReq{
		CategoryID:    nil,
		ImageUrl:      "http://example.com/img.jpg",
		ImagePublicID: "img_pub",
		ProductCode:   &productCode,
		Name:          "Sample Dress",
		Description:   nil,
		Price:         &price,
		ItemType:      string(branditemtype.Sample),
		Status:        string(branditemstatus.Draft),
	}

	res, err := uc.CreateBrandItem(ctx, staffUserID, brandID, createInput)
	if err != nil {
		t.Fatalf("Failed to create brand item: %v", err)
	}
	if res.Name != "Sample Dress" || res.Status != "draft" {
		t.Errorf("Unexpected created brand item values: %+v", res)
	}

	// 2. Staff updates status to ACTIVE
	updateInput := dto.UpdateBrandItemReq{
		Name:   "Sample Dress v2",
		Price:  &price,
		Status: string(branditemstatus.Active),
	}
	updateRes, err := uc.UpdateBrandItem(ctx, staffUserID, brandID, res.ID, updateInput)
	if err != nil {
		t.Fatalf("Failed to update brand item: %v", err)
	}
	if updateRes.Name != "Sample Dress v2" || updateRes.Status != "active" {
		t.Errorf("Unexpected updated brand item values: %+v", updateRes)
	}

	// 3. User lists brand samples
	query := dto.GetBrandItemsQueryReq{}
	userItems, err := uc.ListBrandSamplesForCustomer(ctx, customerUserID, brandID, query)
	if err != nil {
		t.Fatalf("Failed to list brand samples: %v", err)
	}
	if len(userItems.Items) != 1 {
		t.Errorf("Expected 1 sample for user, got %d", len(userItems.Items))
	}

	// 4. User submits feedback
	voteType := "like"
	feedbackText := "Great quality dress!"
	rating := 5
	feedbackInput := dto.SubmitSampleFeedbackReq{
		OutfitID:     nil,
		VoteType:     &voteType,
		Rating:       &rating,
		FeedbackText: &feedbackText,
	}
	fRes, err := uc.SubmitSampleFeedback(ctx, customerUserID, res.ID, feedbackInput)
	if err != nil {
		t.Fatalf("Failed to submit feedback: %v", err)
	}
	if *fRes.VoteType != "like" || *fRes.FeedbackText != "Great quality dress!" {
		t.Errorf("Unexpected feedback response: %+v", fRes)
	}

	// 5. Staff fetches feedbacks
	feedbacks, err := uc.GetBrandItemFeedbacks(ctx, staffUserID, brandID, res.ID)
	if err != nil {
		t.Fatalf("Failed to fetch feedbacks for staff: %v", err)
	}
	if len(feedbacks) != 1 {
		t.Errorf("Expected 1 feedback, got %d", len(feedbacks))
	}
}
