package ai

import (
	"context"
	"errors"
	"strings"
	"testing"

	subscription_contract "smart-wardrobe-be/internal/modules/subscription/contract"
	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"

	"github.com/google/uuid"
)

// Define fake repositories & services
type fakeContextRepo struct {
	repositories.IConversationalContextRepository
	session *entities.ConversationalContext
}
func (r *fakeContextRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.ConversationalContext, error) {
	return r.session, nil
}
func (r *fakeContextRepo) Update(ctx context.Context, item *entities.ConversationalContext) error {
	r.session = item
	return nil
}

type fakeMessageRepo struct {
	repositories.IMessageRepository
	messages []*entities.Message
}
func (r *fakeMessageRepo) Create(ctx context.Context, item *entities.Message) error {
	r.messages = append(r.messages, item)
	return nil
}
func (r *fakeMessageRepo) GetRecentByContextID(ctx context.Context, contextID uuid.UUID, limit int) ([]*entities.Message, error) {
	return r.messages, nil
}
func (r *fakeMessageRepo) CountByContextID(ctx context.Context, contextID uuid.UUID) (int64, error) {
	return int64(len(r.messages)), nil
}
func (r *fakeMessageRepo) GetOldestByContextID(ctx context.Context, contextID uuid.UUID, limit int) ([]*entities.Message, error) {
	return nil, nil
}
func (r *fakeMessageRepo) DeleteByIDs(ctx context.Context, ids []uuid.UUID) error {
	return nil
}

type fakeWardrobeRepo struct {
	repositories.IWardrobeItemRepository
}
func (r *fakeWardrobeRepo) GetByUserID(ctx context.Context, userID uuid.UUID, categoryID *string) ([]*entities.WardrobeItem, error) {
	return []*entities.WardrobeItem{}, nil
}

type fakeAIService struct {
	textChan chan string
	errChan  chan error
}
func (s *fakeAIService) GenerateTextStream(ctx context.Context, systemPrompt, userPrompt string) (<-chan string, <-chan error) {
	return s.textChan, s.errChan
}
func (s *fakeAIService) GenerateText(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return "", nil
}
func (s *fakeAIService) AnalyzeImage(ctx context.Context, imageUrl string, prompt string) (string, error) {
	return "", nil
}
func (s *fakeAIService) GenerateEmbeddings(ctx context.Context, chunks []string) ([][]float32, error) {
	return nil, nil
}

type fakeUserSubContract struct {
	subscription_contract.IUserSubscriptionContract
}
func (c *fakeUserSubContract) GetUserSubscriptionOverview(ctx context.Context, userID uuid.UUID) (*subscription_contract.UserSubscriptionOverviewDTO, error) {
	return &subscription_contract.UserSubscriptionOverviewDTO{
		MaxWardrobeItems: 100,
	}, nil
}

type fakeUserQuotaContract struct {
	subscription_contract.IUserQuotaContract
	quota      *subscription_contract.UserSubscriptionDTO
	updated    bool
	updateErr  error
}
func (c *fakeUserQuotaContract) GetAndResetDailyQuota(ctx context.Context, userID uuid.UUID) (*subscription_contract.UserSubscriptionDTO, error) {
	return c.quota, nil
}
func (c *fakeUserQuotaContract) UpdateAiChatQuota(ctx context.Context, userID uuid.UUID, count int) error {
	c.updated = true
	return c.updateErr
}

type fakeUOW struct {
	shared_repos.IUnitOfWork
}
func (u *fakeUOW) Execute(ctx context.Context, fn func(txCtx context.Context) error) error {
	return fn(ctx)
}

func TestProcessChatMessageStream_Success(t *testing.T) {
	userID := uuid.New()
	contextID := uuid.New()

	session := &entities.ConversationalContext{
		UserID: userID,
		Title:  "Test Chat",
	}

	cRepo := &fakeContextRepo{session: session}
	mRepo := &fakeMessageRepo{}
	wRepo := &fakeWardrobeRepo{}
	
	// Create AI Service that streams chunks
	aiTextChan := make(chan string, 3)
	aiErrChan := make(chan error, 1)
	aiTextChan <- "Hello "
	aiTextChan <- "world!"
	close(aiTextChan)
	close(aiErrChan)

	aiSvc := &fakeAIService{textChan: aiTextChan, errChan: aiErrChan}
	subCtr := &fakeUserSubContract{}
	quotaCtr := &fakeUserQuotaContract{
		quota: &subscription_contract.UserSubscriptionDTO{
			AiChatDailyQuota: 10,
			AiUsageCount:     5,
		},
	}
	uow := &fakeUOW{}

	uc := &WardrobeAIUseCase{
		contextRepo:     cRepo,
		messageRepo:     mRepo,
		wardrobeRepo:    wRepo,
		aiService:       aiSvc,
		userSubContract: subCtr,
		userQuotaCtr:    quotaCtr,
		uow:             uow,
	}

	stream, commit, err := uc.ProcessChatMessageStream(context.Background(), userID, contextID, "Hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Read stream
	var sb strings.Builder
	for token := range stream {
		sb.WriteString(token)
	}

	if sb.String() != "Hello world!" {
		t.Errorf("expected 'Hello world!', got %q", sb.String())
	}

	// Commit successful stream
	err = commit(true)
	if err != nil {
		t.Fatalf("unexpected commit error: %v", err)
	}

	if !quotaCtr.updated {
		t.Error("expected quota update to be called")
	}

	if len(mRepo.messages) != 2 {
		t.Errorf("expected 2 messages saved, got %d", len(mRepo.messages))
	}
}

func TestProcessChatMessageStream_QuotaExceeded(t *testing.T) {
	userID := uuid.New()
	contextID := uuid.New()

	uc := &WardrobeAIUseCase{
		userQuotaCtr: &fakeUserQuotaContract{
			quota: &subscription_contract.UserSubscriptionDTO{
				AiChatDailyQuota: 10,
				AiUsageCount:     10, // Exceeded!
			},
		},
	}

	_, _, err := uc.ProcessChatMessageStream(context.Background(), userID, contextID, "Hi")
	if !errors.Is(err, subscriptionerrors.ErrAiChatQuotaExceeded) {
		t.Errorf("expected ErrAiChatQuotaExceeded, got %v", err)
	}
}

func TestProcessChatMessageStream_ClientDisconnect(t *testing.T) {
	userID := uuid.New()
	contextID := uuid.New()

	session := &entities.ConversationalContext{
		UserID: userID,
		Title:  "Test Chat",
	}

	cRepo := &fakeContextRepo{session: session}
	mRepo := &fakeMessageRepo{}
	wRepo := &fakeWardrobeRepo{}

	aiTextChan := make(chan string, 1)
	aiErrChan := make(chan error, 1)
	aiSvc := &fakeAIService{textChan: aiTextChan, errChan: aiErrChan}
	quotaCtr := &fakeUserQuotaContract{
		quota: &subscription_contract.UserSubscriptionDTO{
			AiChatDailyQuota: 10,
			AiUsageCount:     5,
		},
	}

	uc := &WardrobeAIUseCase{
		contextRepo:     cRepo,
		messageRepo:     mRepo,
		wardrobeRepo:    wRepo,
		aiService:       aiSvc,
		userSubContract: &fakeUserSubContract{},
		userQuotaCtr:    quotaCtr,
		uow:             &fakeUOW{},
	}

	_, commit, err := uc.ProcessChatMessageStream(context.Background(), userID, contextID, "Hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Client disconnects before reading/finishing stream
	err = commit(false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Quota should NOT be updated, and messages should NOT be saved
	if quotaCtr.updated {
		t.Error("expected quota update NOT to be called on failure")
	}
	if len(mRepo.messages) != 0 {
		t.Errorf("expected 0 messages saved, got %d", len(mRepo.messages))
	}
}
