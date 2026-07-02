package usecase

import (
	"context"
	"testing"
	"time"

	"smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/benefit/benefitfeaturecode"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/branditem/branditemstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/branditem/branditemtype"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberrole"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandmemberstatus"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandstatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

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
func (m *mockBrandItemRepo) GetByBrandID(ctx context.Context, brandID uuid.UUID) ([]*entities.BrandItem, error) {
	var res []*entities.BrandItem
	for _, item := range m.items {
		if item.BrandID == brandID {
			res = append(res, item)
		}
	}
	return res, nil
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
func (m *mockBrandItemRepo) GetByProductCode(ctx context.Context, brandID uuid.UUID, code string) (*entities.BrandItem, error) {
	for _, item := range m.items {
		if item.BrandID == brandID && item.ProductCode != nil && *item.ProductCode == code {
			return item, nil
		}
	}
	return nil, nil
}
func (m *mockBrandItemRepo) GetByFashionItemID(ctx context.Context, fashionItemID uuid.UUID) (*entities.BrandItem, error) {
	for _, item := range m.items {
		if item.FashionItemID == fashionItemID {
			return item, nil
		}
	}
	return nil, nil
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

type mockFashionContract struct{}

func (m *mockFashionContract) CreateFashionItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, itemType string, categoryID *uuid.UUID, imageUrl string, imagePublicID string) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockFashionContract) GetFashionItem(ctx context.Context, id uuid.UUID) (*entities.FashionItem, error) {
	return &entities.FashionItem{
		ImageUrl: "http://example.com/image.jpg",
	}, nil
}
func (m *mockFashionContract) ListFashionItems(ctx context.Context, ids []uuid.UUID) ([]*entities.FashionItem, error) {
	return nil, nil
}

type mockBenefitUC struct{}

func (m *mockBenefitUC) CheckBrandFeatureAccess(ctx context.Context, userID uuid.UUID, brandID uuid.UUID, featureCode benefitfeaturecode.BenefitFeatureCode) (bool, error) {
	return true, nil
}
func (m *mockBenefitUC) CreateBrandBenefit(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, input dto.CreateBrandBenefitReq) (*dto.BrandBenefitRes, error) {
	return nil, nil
}
func (m *mockBenefitUC) ListBrandBenefitsForStaff(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandBenefitRes, error) {
	return nil, nil
}
func (m *mockBenefitUC) ListActiveBenefitsForUser(ctx context.Context, userID uuid.UUID, brandID uuid.UUID) ([]*dto.BrandBenefitRes, error) {
	return nil, nil
}
func (m *mockBenefitUC) GetActiveBenefitForUser(ctx context.Context, userID uuid.UUID, benefitID uuid.UUID) (*dto.BrandBenefitRes, error) {
	return nil, nil
}
func (m *mockBenefitUC) ListBenefitRedemptionsForUser(ctx context.Context, userID uuid.UUID) ([]*dto.BenefitRedemptionRes, error) {
	return nil, nil
}
func (m *mockBenefitUC) UpdateBenefitStatus(ctx context.Context, staffUserID uuid.UUID, brandID uuid.UUID, benefitID uuid.UUID, status string) (*dto.BrandBenefitRes, error) {
	return nil, nil
}
func (m *mockBenefitUC) RedeemBenefit(ctx context.Context, userID uuid.UUID, benefitID uuid.UUID) (*dto.BenefitRedemptionRes, error) {
	return nil, nil
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
		benefitUC:       &mockBenefitUC{},
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

func (m *mockBrandItemRepo) GetQueryWithPreload(ctx context.Context) *gormQuery {
	return nil
}

type gormQuery struct{}

func (m *mockBrandItemRepo) GetDB(ctx context.Context) any {
	return nil
}

func (m *mockDigitalSampleResponseRepo) GetQueryWithPreload(ctx context.Context) *gormQuery {
	return nil
}
func (m *mockDigitalSampleResponseRepo) GetDB(ctx context.Context) any {
	return nil
}

func (m *mockBrandItemRepo) FindFirst(ctx context.Context, filter map[string]any) (*entities.BrandItem, error) {
	return nil, nil
}
func (m *mockBrandItemRepo) FindAll(ctx context.Context, filter map[string]any) ([]*entities.BrandItem, error) {
	return nil, nil
}
func (m *mockBrandItemRepo) FindPaged(ctx context.Context, page, limit int, filter map[string]any) ([]*entities.BrandItem, int64, error) {
	return nil, 0, nil
}

func (m *mockDigitalSampleResponseRepo) FindFirst(ctx context.Context, filter map[string]any) (*entities.DigitalSampleResponse, error) {
	return nil, nil
}
func (m *mockDigitalSampleResponseRepo) FindAll(ctx context.Context, filter map[string]any) ([]*entities.DigitalSampleResponse, error) {
	return nil, nil
}
func (m *mockDigitalSampleResponseRepo) FindPaged(ctx context.Context, page, limit int, filter map[string]any) ([]*entities.DigitalSampleResponse, int64, error) {
	return nil, 0, nil
}
