package item

import (
	"context"
	"testing"
	"time"

	"smart-wardrobe-be/services/commerce/internal/modules/brand/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/repositories"
	fashion_dto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	fashion_contract "smart-wardrobe-be/services/commerce/internal/modules/fashion/presentation/contract"

	"github.com/google/uuid"
)

type retryBrandRepoMock struct {
	repositories.IBrandItemRepository
	item    *entities.BrandItem
	updated bool
	claimFn func(ctx context.Context, itemID uuid.UUID, expected *uuid.UUID, newTaskID uuid.UUID, now time.Time) (*entities.BrandItem, bool, error)
}

func (m *retryBrandRepoMock) GetByID(context.Context, uuid.UUID) (*entities.BrandItem, error) {
	return m.item, nil
}

func (m *retryBrandRepoMock) Update(context.Context, *entities.BrandItem) error {
	m.updated = true
	return nil
}

func (m *retryBrandRepoMock) ClaimManualAnalysisRetry(ctx context.Context, itemID uuid.UUID, expected *uuid.UUID, newTaskID uuid.UUID, now time.Time) (*entities.BrandItem, bool, error) {
	if m.claimFn != nil {
		return m.claimFn(ctx, itemID, expected, newTaskID, now)
	}
	claimed := *m.item
	claimed.Status = branditemstatus.Draft
	claimed.TaskID = &newTaskID
	m.updated = true
	return &claimed, true, nil
}

type retryBrandFashionMock struct {
	fashion_contract.IFashionContract
	fi *fashion_dto.FashionItemContractDTO
}

func (m *retryBrandFashionMock) GetFashionItem(context.Context, uuid.UUID) (*fashion_dto.FashionItemContractDTO, error) {
	return m.fi, nil
}

func (m *retryBrandFashionMock) ResetProcessingState(context.Context, uuid.UUID) error {
	// Mirror the real repo: reset clears the review/error reasons, which is what
	// prevents a rapid repeat retry from passing the draft+reviewReason gate.
	if m.fi != nil {
		m.fi.ReviewReason = nil
		m.fi.ProcessingErrorReason = nil
	}
	return nil
}

func (m *retryBrandFashionMock) IsWearableCategory(context.Context, uuid.UUID) (bool, error) {
	return true, nil
}

func (m *retryBrandFashionMock) UpdateFashionItemCategory(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

type brandRetryPublisher struct {
	topic   string
	payload any
}

func (p *brandRetryPublisher) Publish(_ context.Context, topic string, payload any) error {
	p.topic = topic
	p.payload = payload
	return nil
}

type brandUoW struct{}

func (brandUoW) Execute(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }

func brandItem(status branditemstatus.BrandItemStatus) *entities.BrandItem {
	fiID := uuid.New()
	item := &entities.BrandItem{
		BrandID:       uuid.New(),
		FashionItemID: &fiID,
		Status:        status,
	}
	item.ID = uuid.New()
	return item
}

func TestRetryBrandItemAnalysis_FailedItemAllowed(t *testing.T) {
	item := brandItem(branditemstatus.Failed)
	fi := &fashion_dto.FashionItemContractDTO{ID: *item.FashionItemID, Category: &fashion_dto.CategoryBriefRes{ID: uuid.New(), Slug: "ao"}}
	repo := &retryBrandRepoMock{item: item}
	pub := &brandRetryPublisher{}
	uc := &BrandItemUseCase{
		brandItemRepo:   repo,
		fashionContract: &retryBrandFashionMock{fi: fi},
		eventPublisher:  pub,
		uow:             brandUoW{},
	}

	res, err := uc.RetryBrandItemAnalysis(context.Background(), uuid.New(), item.BrandID, item.ID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.TaskID == nil {
		t.Fatal("expected a taskId in the retry response")
	}
	if !repo.updated {
		t.Fatal("expected the item to be updated")
	}
	job, ok := pub.payload.(fashion_dto.FashionAnalyzeJobDTO)
	if !ok {
		t.Fatalf("expected FashionAnalyzeJobDTO, got %T", pub.payload)
	}
	if job.ItemType != "brand" || !job.PreserveCategory {
		t.Fatalf("unexpected job: %+v", job)
	}
}

func TestRetryBrandItemAnalysis_ForbiddenStates(t *testing.T) {
	cases := []struct {
		name string
		item *entities.BrandItem
		fi   *fashion_dto.FashionItemContractDTO
	}{
		{"plain draft", brandItem(branditemstatus.Draft), &fashion_dto.FashionItemContractDTO{}},
		{"active", brandItem(branditemstatus.Active), &fashion_dto.FashionItemContractDTO{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc := &BrandItemUseCase{
				brandItemRepo:   &retryBrandRepoMock{item: tc.item},
				fashionContract: &retryBrandFashionMock{fi: tc.fi},
				eventPublisher:  &brandRetryPublisher{},
				uow:             brandUoW{},
			}
			_, err := uc.RetryBrandItemAnalysis(context.Background(), uuid.New(), tc.item.BrandID, tc.item.ID, nil)
			if err == nil || err.Error() != branderrors.ErrBrandRetryForbidden().Error() {
				t.Fatalf("expected ErrBrandRetryForbidden, got %v", err)
			}
		})
	}
}

func TestRetryBrandItemAnalysis_DraftWithReviewReasonAllowed(t *testing.T) {
	item := brandItem(branditemstatus.Draft)
	reason := "Không đủ tự tin"
	fi := &fashion_dto.FashionItemContractDTO{ID: *item.FashionItemID, ReviewReason: &reason}
	uc := &BrandItemUseCase{
		brandItemRepo:   &retryBrandRepoMock{item: item},
		fashionContract: &retryBrandFashionMock{fi: fi},
		eventPublisher:  &brandRetryPublisher{},
		uow:             brandUoW{},
	}
	catID := uuid.New()
	if _, err := uc.RetryBrandItemAnalysis(context.Background(), uuid.New(), item.BrandID, item.ID, &catID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRetryBrandItemAnalysis_DraftAwaitingReviewRequiresCategory(t *testing.T) {
	item := brandItem(branditemstatus.Draft)
	reason := "Không đủ tự tin"
	fi := &fashion_dto.FashionItemContractDTO{ID: *item.FashionItemID, ReviewReason: &reason}
	uc := &BrandItemUseCase{
		brandItemRepo:   &retryBrandRepoMock{item: item},
		fashionContract: &retryBrandFashionMock{fi: fi},
		eventPublisher:  &brandRetryPublisher{},
		uow:             brandUoW{},
	}
	_, err := uc.RetryBrandItemAnalysis(context.Background(), uuid.New(), item.BrandID, item.ID, nil)
	if err == nil || err.Error() != branderrors.ErrBrandRetryCategoryRequired().Error() {
		t.Fatalf("expected ErrBrandRetryCategoryRequired, got %v", err)
	}
}

func TestRetryBrandItemAnalysis_InProgressRejected(t *testing.T) {
	item := brandItem(branditemstatus.Failed)
	repo := &retryBrandRepoMock{
		item: item,
		claimFn: func(context.Context, uuid.UUID, *uuid.UUID, uuid.UUID, time.Time) (*entities.BrandItem, bool, error) {
			return nil, false, nil // someone else already claimed
		},
	}
	uc := &BrandItemUseCase{
		brandItemRepo:   repo,
		fashionContract: &retryBrandFashionMock{fi: &fashion_dto.FashionItemContractDTO{}},
		eventPublisher:  &brandRetryPublisher{},
		uow:             brandUoW{},
	}
	_, err := uc.RetryBrandItemAnalysis(context.Background(), uuid.New(), item.BrandID, item.ID, nil)
	if err == nil || err.Error() != branderrors.ErrBrandRetryInProgress().Error() {
		t.Fatalf("expected ErrBrandRetryInProgress, got %v", err)
	}
}

// A claimed retry clears the fashion review reason, so a rapid repeat retry is
// rejected instead of creating a duplicate analysis session (FR-011).
func TestRetryBrandItemAnalysis_SecondRetryRejectedAfterClaim(t *testing.T) {
	item := brandItem(branditemstatus.Draft)
	reason := "AI không chắc"
	fi := &fashion_dto.FashionItemContractDTO{ID: *item.FashionItemID, ReviewReason: &reason}
	uc := &BrandItemUseCase{
		brandItemRepo:   &retryBrandRepoMock{item: item},
		fashionContract: &retryBrandFashionMock{fi: fi},
		eventPublisher:  &brandRetryPublisher{},
		uow:             brandUoW{},
	}

	catID := uuid.New()
	if _, err := uc.RetryBrandItemAnalysis(context.Background(), uuid.New(), item.BrandID, item.ID, &catID); err != nil {
		t.Fatalf("first retry should succeed: %v", err)
	}
	_, err := uc.RetryBrandItemAnalysis(context.Background(), uuid.New(), item.BrandID, item.ID, nil)
	if err == nil || err.Error() != branderrors.ErrBrandRetryForbidden().Error() {
		t.Fatalf("second retry should be forbidden, got %v", err)
	}
}
