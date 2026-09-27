package item

import (
	"context"
	"testing"
	"time"

	fashion_dto "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	wardrobeerrors "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/entities"

	"github.com/google/uuid"
)

type retryPublisher struct {
	topic   string
	payload any
}

func (p *retryPublisher) Publish(_ context.Context, topic string, payload any) error {
	p.topic = topic
	p.payload = payload
	return nil
}

func TestRetryWardrobeAnalysis_GuardRejectsInvalidStatus(t *testing.T) {
	userID := uuid.New()
	fashionID := uuid.New()
	item := wardItem(userID, wardrobestatus.InWardrobe, &fashionID)

	repo := &confirmRepoMock{
		getByIDFn: func(context.Context, uuid.UUID) (*entities.WardrobeItem, error) { return item, nil },
	}
	uc := &WardrobeItemUseCase{wardrobeRepo: repo, uow: directUoW{}}

	_, err := uc.RetryWardrobeAnalysis(context.Background(), userID, item.ID, nil)
	assertSameErr(t, err, wardrobeerrors.ErrRetryWardrobeAnalysisForbidden())
}

func TestRetryWardrobeAnalysis_PreservesCategoryAndReturnsTaskID(t *testing.T) {
	userID := uuid.New()
	fashionID := uuid.New()
	catID := uuid.New()
	item := wardItem(userID, wardrobestatus.NeedsReview, &fashionID)

	repo := &confirmRepoMock{
		getByIDFn: func(context.Context, uuid.UUID) (*entities.WardrobeItem, error) { return item, nil },
		claimFn: func(_ context.Context, _, _ uuid.UUID, _ time.Time) (*entities.WardrobeItem, bool, error) {
			claimed := *item
			claimed.Status = wardrobestatus.Processing
			return &claimed, true, nil
		},
	}
	fashion := &confirmFashionMock{
		getFashionItemFn: func(context.Context, uuid.UUID) (*fashion_dto.FashionItemContractDTO, error) {
			return &fashion_dto.FashionItemContractDTO{ID: fashionID, Category: &fashion_dto.CategoryBriefRes{ID: catID, Slug: "ao"}}, nil
		},
	}
	pub := &retryPublisher{}
	uc := &WardrobeItemUseCase{
		wardrobeRepo:    repo,
		fashionContract: fashion,
		eventPublisher:  pub,
		uow:             directUoW{},
	}

	res, err := uc.RetryWardrobeAnalysis(context.Background(), userID, item.ID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.TaskID == nil {
		t.Fatal("expected a taskId in the retry response")
	}

	job, ok := pub.payload.(fashion_dto.FashionAnalyzeJobDTO)
	if !ok {
		t.Fatalf("expected FashionAnalyzeJobDTO payload, got %T", pub.payload)
	}
	if !job.PreserveCategory {
		t.Fatal("expected PreserveCategory=true when the fashion item already has a category")
	}
	if !job.FixedCategory {
		t.Fatal("expected FixedCategory=true for a needs_review retry")
	}
	if job.CategoryID == nil || *job.CategoryID != catID {
		t.Fatalf("expected job categoryId %s, got %v", catID, job.CategoryID)
	}
	if job.TaskID != res.TaskID.String() {
		t.Fatalf("job taskId %q != response taskId %q", job.TaskID, res.TaskID.String())
	}
}

func TestRetryWardrobeAnalysis_NeedsReviewRequiresCategory(t *testing.T) {
	userID := uuid.New()
	fashionID := uuid.New()
	item := wardItem(userID, wardrobestatus.NeedsReview, &fashionID)

	repo := &confirmRepoMock{
		getByIDFn: func(context.Context, uuid.UUID) (*entities.WardrobeItem, error) { return item, nil },
	}
	fashion := &confirmFashionMock{
		getFashionItemFn: func(context.Context, uuid.UUID) (*fashion_dto.FashionItemContractDTO, error) {
			return &fashion_dto.FashionItemContractDTO{ID: fashionID}, nil
		},
	}
	uc := &WardrobeItemUseCase{
		wardrobeRepo:    repo,
		fashionContract: fashion,
		eventPublisher:  &retryPublisher{},
		uow:             directUoW{},
	}

	_, err := uc.RetryWardrobeAnalysis(context.Background(), userID, item.ID, nil)
	assertSameErr(t, err, wardrobeerrors.ErrRetryCategoryRequired())
}

func TestRetryWardrobeAnalysis_NeedsReviewRejectsNonWearableCategory(t *testing.T) {
	userID := uuid.New()
	fashionID := uuid.New()
	item := wardItem(userID, wardrobestatus.NeedsReview, &fashionID)
	requested := uuid.New()

	repo := &confirmRepoMock{
		getByIDFn: func(context.Context, uuid.UUID) (*entities.WardrobeItem, error) { return item, nil },
	}
	fashion := &confirmFashionMock{
		getFashionItemFn: func(context.Context, uuid.UUID) (*fashion_dto.FashionItemContractDTO, error) {
			return &fashion_dto.FashionItemContractDTO{ID: fashionID}, nil
		},
		isWearableCategoryFn: func(context.Context, uuid.UUID) (bool, error) { return false, nil },
	}
	uc := &WardrobeItemUseCase{
		wardrobeRepo:    repo,
		fashionContract: fashion,
		eventPublisher:  &retryPublisher{},
		uow:             directUoW{},
	}

	_, err := uc.RetryWardrobeAnalysis(context.Background(), userID, item.ID, &requested)
	assertSameErr(t, err, wardrobeerrors.ErrRetryCategoryRequired())
}
