package item

import (
	"context"
	"testing"

	"smart-wardrobe-be/internal/config"
	sub_dto "smart-wardrobe-be/internal/modules/subscription/application/dto"
	sub_contract "smart-wardrobe-be/internal/modules/subscription/presentation/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/media"
	"smart-wardrobe-be/internal/shared/domain/constants/roleslug"

	"github.com/google/uuid"
)

type mockMediaService struct {
	media.IMediaService
	generateUploadSignatureFn func(ctx context.Context, params shared_dto.UploadSignatureParams) (*shared_dto.UploadSignatureResult, error)
}

func (m *mockMediaService) GenerateUploadSignature(ctx context.Context, params shared_dto.UploadSignatureParams) (*shared_dto.UploadSignatureResult, error) {
	if m.generateUploadSignatureFn != nil {
		return m.generateUploadSignatureFn(ctx, params)
	}
	return &shared_dto.UploadSignatureResult{
		Signature: "test_sig",
		Timestamp: 1234567890,
		ApiKey:    "test_key",
		PublicID:  params.PublicID,
		Folder:    params.Folder,
	}, nil
}

type mockSubContract struct {
	sub_contract.IUserSubscriptionContract
	getOverviewFn func(ctx context.Context, userID uuid.UUID) (*sub_dto.UserSubscriptionContractDTO, error)
}

func (m *mockSubContract) GetUserSubscriptionOverview(ctx context.Context, userID uuid.UUID) (*sub_dto.UserSubscriptionContractDTO, error) {
	if m.getOverviewFn != nil {
		return m.getOverviewFn(ctx, userID)
	}
	return &sub_dto.UserSubscriptionContractDTO{
		MaxWardrobeItems: 50,
	}, nil
}

type mockWardrobeRepo struct {
	repositories.IWardrobeItemRepository
	countByUserIDFn func(ctx context.Context, userID uuid.UUID) (int64, error)
}

func (m *mockWardrobeRepo) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	if m.countByUserIDFn != nil {
		return m.countByUserIDFn(ctx, userID)
	}
	return 0, nil
}

func TestGetUploadSignature_Admin(t *testing.T) {
	uc := &WardrobeItemUseCase{
		cfg:          &config.Config{},
		mediaService: &mockMediaService{},
	}

	res, err := uc.GetUploadSignature(context.Background(), uuid.New(), roleslug.Admin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.RemainingSlots != -1 || res.MaxItems != -1 {
		t.Fatalf("expected unlimited slots for admin, got %d / %d", res.RemainingSlots, res.MaxItems)
	}
	if res.Signature != "test_sig" {
		t.Fatalf("expected signature 'test_sig', got %s", res.Signature)
	}
}

func TestGetUploadSignature_User_Success(t *testing.T) {
	uc := &WardrobeItemUseCase{
		cfg:             &config.Config{},
		mediaService:    &mockMediaService{},
		userSubContract: &mockSubContract{},
		wardrobeRepo: &mockWardrobeRepo{
			countByUserIDFn: func(ctx context.Context, userID uuid.UUID) (int64, error) {
				return 10, nil
			},
		},
	}

	res, err := uc.GetUploadSignature(context.Background(), uuid.New(), roleslug.User)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.RemainingSlots != 40 {
		t.Fatalf("expected remaining slots 40, got %d", res.RemainingSlots)
	}
	if res.MaxItems != 50 {
		t.Fatalf("expected max items 50, got %d", res.MaxItems)
	}
	if res.CurrentItems != 10 {
		t.Fatalf("expected current items 10, got %d", res.CurrentItems)
	}
}

func TestGetUploadSignature_User_QuotaExceeded(t *testing.T) {
	uc := &WardrobeItemUseCase{
		cfg:             &config.Config{},
		mediaService:    &mockMediaService{},
		userSubContract: &mockSubContract{},
		wardrobeRepo: &mockWardrobeRepo{
			countByUserIDFn: func(ctx context.Context, userID uuid.UUID) (int64, error) {
				return 50, nil
			},
		},
	}

	res, err := uc.GetUploadSignature(context.Background(), uuid.New(), roleslug.User)
	if err == nil {
		t.Fatal("expected quota exceeded error, got nil")
	}
	if res != nil {
		t.Fatalf("expected nil result on error, got %v", res)
	}
}
