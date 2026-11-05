package worker

import (
	"context"
	"errors"
	"testing"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/observability/workerlog"

	"github.com/google/uuid"
)

// The wardrobe analyze worker used to compute the configured backoff and then
// ignore it: the retry was republished on the spot, so all three attempts were
// spent within ~5s. Cloudinary needs 10-24s to generate the background-removal
// derivative the vision provider downloads, so every fresh upload failed with
// "analysis_temporary_error". A transient failure must now go to the delay queue
// matching its attempt.
func TestHandleJobFailureSchedulesRetryOnTheMatchingDelayQueue(t *testing.T) {
	cases := []struct {
		retryCount int
		wantTopic  string
	}{
		{0, eventconstants.TopicFashionAnalyzeRetry1},
		{1, eventconstants.TopicFashionAnalyzeRetry2},
		{2, eventconstants.TopicFashionAnalyzeRetry3},
	}

	for _, tc := range cases {
		fashionItemID := uuid.New()
		var publishedTopics []string

		uc := &FashionWorkerUseCase{
			cfg: &config.Config{Wardrobe: config.WardrobeProcessing{MaxRetryCount: 3}},
			fashionItemRepo: &mockFashionItemRepoForWorker{
				incrementRetryCountFn: func(ctx context.Context, itemID uuid.UUID) error {
					return nil
				},
			},
			eventPublisher: &mockEventPublisherForWorker{
				publishFn: func(ctx context.Context, topic string, payload any) error {
					publishedTopics = append(publishedTopics, topic)
					return nil
				},
			},
			uow: &mockUOWForWorker{
				executeFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
					return fn(ctx)
				},
			},
			logger: &mockLogger{},
		}

		job := dto.FashionAnalyzeJobDTO{
			TaskID:        uuid.NewString(),
			ItemID:        uuid.New(),
			FashionItemID: &fashionItemID,
			RetryCount:    tc.retryCount,
		}
		run := workerlog.New("fashion_analyze_item", "test")

		uc.handleJobFailure(context.Background(), job, errors.New("failed to download image, status code: 423"), run)

		if len(publishedTopics) != 1 {
			t.Fatalf("retryCount %d: published %d events (%v), want exactly 1",
				tc.retryCount, len(publishedTopics), publishedTopics)
		}
		if publishedTopics[0] != tc.wantTopic {
			t.Errorf("retryCount %d: published to %q, want %q (the delay queue for that attempt)",
				tc.retryCount, publishedTopics[0], tc.wantTopic)
		}
	}
}

// A retry budget that is already spent must finalize the item instead of
// scheduling a fourth attempt.
func TestHandleJobFailureFinalizesWhenRetryBudgetIsSpent(t *testing.T) {
	fashionItemID := uuid.New()
	var publishedTopics []string
	failedMarked := false

	uc := &FashionWorkerUseCase{
		cfg: &config.Config{Wardrobe: config.WardrobeProcessing{MaxRetryCount: 3}},
		fashionItemRepo: &mockFashionItemRepoForWorker{
			markProcessingFailedFn: func(ctx context.Context, itemID uuid.UUID, processingVersion int, reason string, reviewReason *string) (bool, error) {
				failedMarked = true
				return true, nil
			},
		},
		eventPublisher: &mockEventPublisherForWorker{
			publishFn: func(ctx context.Context, topic string, payload any) error {
				publishedTopics = append(publishedTopics, topic)
				return nil
			},
		},
		uow: &mockUOWForWorker{
			executeFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
				return fn(ctx)
			},
		},
		logger: &mockLogger{},
	}

	job := dto.FashionAnalyzeJobDTO{
		TaskID:        uuid.NewString(),
		ItemID:        uuid.New(),
		FashionItemID: &fashionItemID,
		RetryCount:    3,
	}

	uc.handleJobFailure(context.Background(), job, errors.New("failed to download image, status code: 423"),
		workerlog.New("fashion_analyze_item", "test"))

	if !failedMarked {
		t.Fatal("expected MarkProcessingFailed for an exhausted retry budget")
	}
	if len(publishedTopics) != 1 || publishedTopics[0] != eventconstants.TopicFashionFailedWardrobe {
		t.Fatalf("published %v, want only %s", publishedTopics, eventconstants.TopicFashionFailedWardrobe)
	}
}

// A non-transient failure must never enter the retry ladder: retrying a 404 or a
// "no fashion item" answer forever is exactly what FR-033 forbids.
func TestHandleJobFailureDoesNotRetryPermanentErrors(t *testing.T) {
	fashionItemID := uuid.New()
	var publishedTopics []string

	uc := &FashionWorkerUseCase{
		cfg: &config.Config{Wardrobe: config.WardrobeProcessing{MaxRetryCount: 3}},
		fashionItemRepo: &mockFashionItemRepoForWorker{
			markProcessingFailedFn: func(ctx context.Context, itemID uuid.UUID, processingVersion int, reason string, reviewReason *string) (bool, error) {
				return true, nil
			},
		},
		eventPublisher: &mockEventPublisherForWorker{
			publishFn: func(ctx context.Context, topic string, payload any) error {
				publishedTopics = append(publishedTopics, topic)
				return nil
			},
		},
		uow: &mockUOWForWorker{
			executeFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
				return fn(ctx)
			},
		},
		logger: &mockLogger{},
	}

	job := dto.FashionAnalyzeJobDTO{
		TaskID:        uuid.NewString(),
		ItemID:        uuid.New(),
		FashionItemID: &fashionItemID,
	}

	uc.handleJobFailure(context.Background(), job, errors.New("google vision: invalid image payload"),
		workerlog.New("fashion_analyze_item", "test"))

	for _, topic := range publishedTopics {
		if topic == eventconstants.TopicFashionAnalyzeRetry1 ||
			topic == eventconstants.TopicFashionAnalyzeRetry2 ||
			topic == eventconstants.TopicFashionAnalyzeRetry3 {
			t.Fatalf("permanent error was scheduled on a retry queue: %v", publishedTopics)
		}
	}
}

func TestAnalyzeRetryTopicFallsBackToTheLongestDelay(t *testing.T) {
	// If max_retry_count is raised later, extra attempts reuse the longest delay.
	// A longer wait is safe; a shorter one is what broke production.
	for _, retryCount := range []int{4, 5, 10} {
		if got := analyzeRetryTopic(retryCount); got != eventconstants.TopicFashionAnalyzeRetry3 {
			t.Errorf("analyzeRetryTopic(%d) = %q, want %q",
				retryCount, got, eventconstants.TopicFashionAnalyzeRetry3)
		}
	}
}
