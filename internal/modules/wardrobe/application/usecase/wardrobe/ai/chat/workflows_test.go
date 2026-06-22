package chat

import (
	"context"
	"errors"
	"testing"

	"smart-wardrobe-be/config"
	sub_contract "smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"

	"github.com/google/uuid"
)

// fakeContextRepo implements IConversationalContextRepository for testing
type fakeContextRepo struct {
	repositories.IConversationalContextRepository
	session *entities.ConversationalContext
	updated bool
	deleted bool
}

func (f *fakeContextRepo) GetByID(ctx context.Context, id uuid.UUID) (*entities.ConversationalContext, error) {
	if f.session != nil && f.session.ID == id {
		return f.session, nil
	}
	return nil, nil
}

func (f *fakeContextRepo) Update(ctx context.Context, entity *entities.ConversationalContext) error {
	f.updated = true
	return nil
}

func (f *fakeContextRepo) Delete(ctx context.Context, id uuid.UUID) error {
	f.deleted = true
	return nil
}

// fakeMessageRepo implements IMessageRepository for testing
type fakeMessageRepo struct {
	repositories.IMessageRepository
	savedMessages []*entities.Message
}

func (f *fakeMessageRepo) Create(ctx context.Context, entity *entities.Message) error {
	f.savedMessages = append(f.savedMessages, entity)
	return nil
}

func (f *fakeMessageRepo) GetRecentByContextID(ctx context.Context, contextID uuid.UUID, limit int) ([]*entities.Message, error) {
	return nil, nil
}

func (f *fakeMessageRepo) CountUnsummarizedByContextID(ctx context.Context, contextID uuid.UUID) (int64, error) {
	return 0, nil
}

// fakeUserSubContract implements IUserSubscriptionContract for testing
type fakeUserSubContract struct {
	sub_contract.IUserSubscriptionContract
}

func (f fakeUserSubContract) GetUserSubscriptionOverview(ctx context.Context, userID uuid.UUID) (*sub_contract.UserSubscriptionOverviewDTO, error) {
	return &sub_contract.UserSubscriptionOverviewDTO{
		MaxWardrobeItems: 100,
	}, nil
}

// fakeUserQuotaContract implements IUserQuotaContract for testing
type fakeUserQuotaContract struct {
	sub_contract.IUserQuotaContract
	chatQuotaUpdated bool
}

func (f *fakeUserQuotaContract) GetAndResetDailyQuota(ctx context.Context, userID uuid.UUID) (*sub_contract.UserSubscriptionDTO, error) {
	return &sub_contract.UserSubscriptionDTO{
		AiChatDailyQuota: 10,
		AiUsageCount:     2,
	}, nil
}

func (f *fakeUserQuotaContract) UpdateAiChatQuota(ctx context.Context, userID uuid.UUID, count int) error {
	f.chatQuotaUpdated = true
	return nil
}

// fakeUnitOfWork implements IUnitOfWork for testing
type fakeUnitOfWork struct {
	shared_repos.IUnitOfWork
}

func (f fakeUnitOfWork) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// fakeAIService implements IAIService for testing
type fakeAIService struct {
	textChan chan string
	errChan  chan error
}

func (f fakeAIService) AnalyzeImage(ctx context.Context, imageUrl string, prompt string) (string, error) {
	return "", nil
}

func (f fakeAIService) GenerateEmbeddings(ctx context.Context, chunks []string) ([][]float32, error) {
	return nil, nil
}

func (f fakeAIService) GenerateChatText(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	return "", nil
}

func (f fakeAIService) GenerateRecommendationText(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	return "", nil
}

func (f fakeAIService) GenerateChatTextStream(ctx context.Context, systemPrompt string, userPrompt string) (<-chan string, <-chan error) {
	return f.textChan, f.errChan
}

func TestProcessChatMessageStream_HappyPath(t *testing.T) {
	sessionID := uuid.New()
	userID := uuid.New()
	session := &entities.ConversationalContext{
		AuditableEntity: entities.AuditableEntity{
			BaseEntity: entities.BaseEntity{
				ID: sessionID,
			},
		},
		UserID: userID,
		Title:  "Test Session",
	}

	contextRepo := &fakeContextRepo{session: session}
	messageRepo := &fakeMessageRepo{}
	quotaCtr := &fakeUserQuotaContract{}
	uow := &fakeUnitOfWork{}

	textChan := make(chan string, 2)
	errChan := make(chan error, 1)
	textChan <- "Hello"
	textChan <- " World!"
	close(textChan)
	close(errChan)

	aiService := fakeAIService{textChan: textChan, errChan: errChan}

	uc := &WardrobeChatUseCase{
		cfg:             &config.Config{},
		contextRepo:     contextRepo,
		messageRepo:     messageRepo,
		aiService:       aiService,
		userSubContract: fakeUserSubContract{},
		userQuotaCtr:    quotaCtr,
		uow:             uow,
	}

	stream, commitFn, err := uc.ProcessChatMessageStream(context.Background(), userID, sessionID, "Hi Stylist")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that Commit 1 (User Message) has run immediately
	if len(messageRepo.savedMessages) != 1 {
		t.Fatalf("expected 1 user message saved, got %d", len(messageRepo.savedMessages))
	}
	userMsg := messageRepo.savedMessages[0]
	if userMsg.Content != "Hi Stylist" || userMsg.Sender != "user" {
		t.Errorf("saved user message mismatch: %+v", userMsg)
	}

	// Read stream output
	var output string
	for chunk := range stream {
		output += chunk
	}
	if output != "Hello World!" {
		t.Errorf("expected stream output 'Hello World!', got %q", output)
	}

	// Verify that AI message is NOT yet saved before commitFn is called
	if len(messageRepo.savedMessages) != 1 {
		t.Fatalf("expected AI message to not be saved yet, got %d messages", len(messageRepo.savedMessages))
	}

	// Commit function call (Commit 2)
	err = commitFn(true)
	if err != nil {
		t.Fatalf("unexpected error on commitFn: %v", err)
	}

	// Verify AI message was saved
	if len(messageRepo.savedMessages) != 2 {
		t.Fatalf("expected 2 messages saved after commit, got %d", len(messageRepo.savedMessages))
	}
	aiMsg := messageRepo.savedMessages[1]
	if aiMsg.Content != "Hello World!" || aiMsg.Sender != "ai" {
		t.Errorf("saved AI message mismatch: %+v", aiMsg)
	}

	// Verify quota was consumed
	if !quotaCtr.chatQuotaUpdated {
		t.Error("expected chat quota to be updated/consumed")
	}
}

func TestProcessChatMessageStream_FailurePath(t *testing.T) {
	sessionID := uuid.New()
	userID := uuid.New()
	session := &entities.ConversationalContext{
		AuditableEntity: entities.AuditableEntity{
			BaseEntity: entities.BaseEntity{
				ID: sessionID,
			},
		},
		UserID: userID,
		Title:  "Test Session",
	}

	contextRepo := &fakeContextRepo{session: session}
	messageRepo := &fakeMessageRepo{}
	quotaCtr := &fakeUserQuotaContract{}
	uow := &fakeUnitOfWork{}

	textChan := make(chan string, 1)
	errChan := make(chan error, 1)
	textChan <- "Failed text"
	close(textChan)
	errChan <- errors.New("timeout error")
	close(errChan)

	aiService := fakeAIService{textChan: textChan, errChan: errChan}

	uc := &WardrobeChatUseCase{
		cfg:             &config.Config{},
		contextRepo:     contextRepo,
		messageRepo:     messageRepo,
		aiService:       aiService,
		userSubContract: fakeUserSubContract{},
		userQuotaCtr:    quotaCtr,
		uow:             uow,
	}

	_, commitFn, err := uc.ProcessChatMessageStream(context.Background(), userID, sessionID, "Hi Stylist")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that Commit 1 (User Message) has run immediately
	if len(messageRepo.savedMessages) != 1 {
		t.Fatalf("expected 1 user message saved, got %d", len(messageRepo.savedMessages))
	}
	userMsg := messageRepo.savedMessages[0]
	if userMsg.Content != "Hi Stylist" || userMsg.Sender != "user" {
		t.Errorf("saved user message mismatch: %+v", userMsg)
	}

	// Call commitFn with success = false (simulating failure / abort)
	err = commitFn(false)
	if err != nil {
		t.Fatalf("unexpected error on commitFn: %v", err)
	}

	// Verify AI message was NOT saved, but user message remains saved
	if len(messageRepo.savedMessages) != 1 {
		t.Fatalf("expected only user message saved after failed run, got %d messages", len(messageRepo.savedMessages))
	}

	// Verify quota was NOT consumed
	if quotaCtr.chatQuotaUpdated {
		t.Error("expected chat quota NOT to be consumed on failure")
	}
}

// fakeEmptyWardrobeRepo implements IWardrobeItemRepository for testing
type fakeEmptyWardrobeRepo struct {
	repositories.IWardrobeItemRepository
}

func (f fakeEmptyWardrobeRepo) GetByUserID(ctx context.Context, userID uuid.UUID, categorySlug *string) ([]*entities.WardrobeItem, error) {
	return nil, nil
}

func TestProcessChatMessageStream_WardrobeKeyword(t *testing.T) {
	sessionID := uuid.New()
	userID := uuid.New()
	session := &entities.ConversationalContext{
		AuditableEntity: entities.AuditableEntity{
			BaseEntity: entities.BaseEntity{
				ID: sessionID,
			},
		},
		UserID: userID,
		Title:  "Test Session",
	}

	contextRepo := &fakeContextRepo{session: session}
	messageRepo := &fakeMessageRepo{}
	quotaCtr := &fakeUserQuotaContract{}
	uow := &fakeUnitOfWork{}

	textChan := make(chan string)
	errChan := make(chan error)
	close(textChan)
	close(errChan)

	aiService := fakeAIService{textChan: textChan, errChan: errChan}

	uc := &WardrobeChatUseCase{
		cfg:             &config.Config{},
		contextRepo:     contextRepo,
		messageRepo:     messageRepo,
		wardrobeRepo:    fakeEmptyWardrobeRepo{},
		aiService:       aiService,
		userSubContract: fakeUserSubContract{},
		userQuotaCtr:    quotaCtr,
		uow:             uow,
	}

	// User query contains keyword "tủ đồ" which triggers wardrobe retrieval
	_, _, err := uc.ProcessChatMessageStream(context.Background(), userID, sessionID, "cho tôi xem tủ đồ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(messageRepo.savedMessages) != 1 {
		t.Fatalf("expected 1 user message saved, got %d", len(messageRepo.savedMessages))
	}
}

func TestDeleteChatSession_Success(t *testing.T) {
	sessionID := uuid.New()
	userID := uuid.New()
	session := &entities.ConversationalContext{
		AuditableEntity: entities.AuditableEntity{
			BaseEntity: entities.BaseEntity{
				ID: sessionID,
			},
		},
		UserID: userID,
		Title:  "Delete Me",
	}

	contextRepo := &fakeContextRepo{session: session}
	uc := &WardrobeChatUseCase{
		contextRepo: contextRepo,
	}

	err := uc.DeleteChatSession(context.Background(), userID, sessionID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !contextRepo.deleted {
		t.Error("expected session to be deleted")
	}
}

func TestDeleteChatSession_NotFound(t *testing.T) {
	sessionID := uuid.New()
	userID := uuid.New()

	contextRepo := &fakeContextRepo{session: nil}
	uc := &WardrobeChatUseCase{
		contextRepo: contextRepo,
	}

	err := uc.DeleteChatSession(context.Background(), userID, sessionID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpdateChatSession_Success(t *testing.T) {
	sessionID := uuid.New()
	userID := uuid.New()
	session := &entities.ConversationalContext{
		AuditableEntity: entities.AuditableEntity{
			BaseEntity: entities.BaseEntity{
				ID: sessionID,
			},
		},
		UserID: userID,
		Title:  "Old Title",
	}

	contextRepo := &fakeContextRepo{session: session}
	uc := &WardrobeChatUseCase{
		contextRepo: contextRepo,
	}

	newTitle := "New Title"
	res, err := uc.UpdateChatSession(context.Background(), userID, sessionID, dto.UpdateChatSessionReq{
		Title: &newTitle,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Title != "New Title" {
		t.Errorf("expected title to be updated, got %s", res.Title)
	}
	if !contextRepo.updated {
		t.Error("expected contextRepo.Update to be called")
	}
}

func TestUpdateChatSession_NotFound(t *testing.T) {
	sessionID := uuid.New()
	userID := uuid.New()

	contextRepo := &fakeContextRepo{session: nil}
	uc := &WardrobeChatUseCase{
		contextRepo: contextRepo,
	}

	newTitle := "New Title"
	_, err := uc.UpdateChatSession(context.Background(), userID, sessionID, dto.UpdateChatSessionReq{
		Title: &newTitle,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

