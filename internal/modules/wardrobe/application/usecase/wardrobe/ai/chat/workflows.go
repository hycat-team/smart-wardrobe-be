package chat

import (
	"context"
	"strings"
	"time"

	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/shared"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/messagesender"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type chatSessionContext struct {
	session *entities.ConversationalContext
	recent  []*entities.Message
}

type chatGenerationInput struct {
	systemPrompt string
	userContent  string
}

// CreateChatSession creates a new chat session for the given user.
func (uc *WardrobeChatUseCase) CreateChatSession(ctx context.Context, userID uuid.UUID, title *string) (*dto.ChatSessionRes, error) {
	sessionTitle := "Cuộc trò chuyện mới"
	if title != nil && strings.TrimSpace(*title) != "" {
		sessionTitle = strings.TrimSpace(*title)
	}

	entity := &entities.ConversationalContext{UserID: userID, Title: sessionTitle}
	if err := uc.contextRepo.Create(ctx, entity); err != nil {
		return nil, err
	}
	return mapper.MapChatSession(entity), nil
}

// GetChatSessions returns all chat sessions that belong to the user.
func (uc *WardrobeChatUseCase) GetChatSessions(ctx context.Context, userID uuid.UUID) ([]*dto.ChatSessionRes, error) {
	items, err := uc.contextRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	res := make([]*dto.ChatSessionRes, len(items))
	for i, item := range items {
		res[i] = mapper.MapChatSession(item)
	}
	return res, nil
}

// GetChatMessages returns paginated chat messages for a user-owned session.
func (uc *WardrobeChatUseCase) GetChatMessages(ctx context.Context, userID uuid.UUID, contextID uuid.UUID, query dto.GetChatMessagesQueryReq) (*shared_dto.PaginationResult[*dto.ChatMessageRes], error) {
	session, err := uc.contextRepo.GetByID(ctx, contextID)
	if err != nil {
		return nil, err
	}
	if session == nil || session.UserID != userID {
		return nil, wardrobeerrors.ErrChatNotFound()
	}
	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}

	paginationQuery := shared_dto.PaginationQuery{Page: page, Limit: limit}

	totalItems, err := uc.messageRepo.CountByContextID(ctx, contextID)
	if err != nil {
		return nil, err
	}

	items, err := uc.messageRepo.GetByContextIDPaginated(ctx, contextID, paginationQuery)
	if err != nil {
		return nil, err
	}
	res := make([]*dto.ChatMessageRes, len(items))
	for i, item := range items {
		res[i] = mapper.MapChatMessage(item)
	}
	return &shared_dto.PaginationResult[*dto.ChatMessageRes]{
		Items:    res,
		Metadata: shared_dto.BuildPaginationMetadata(paginationQuery, totalItems),
	}, nil
}

// ArchiveChatSession archives a user-owned chat session without deleting its history.
func (uc *WardrobeChatUseCase) ArchiveChatSession(ctx context.Context, userID uuid.UUID, contextID uuid.UUID) error {
	session, err := uc.contextRepo.GetByID(ctx, contextID)
	if err != nil {
		return err
	}
	if session == nil || session.UserID != userID {
		return wardrobeerrors.ErrChatNotFound()
	}
	session.IsArchived = true
	return uc.contextRepo.Update(ctx, session)
}

// ProcessChatMessageStream streams an AI reply and persists the conversation on successful completion.
func (uc *WardrobeChatUseCase) ProcessChatMessageStream(ctx context.Context, userID uuid.UUID, contextID uuid.UUID, content string) (<-chan string, func(success bool) error, error) {
	sessionCtx, err := uc.loadChatSessionContext(ctx, userID, contextID)
	if err != nil {
		return nil, nil, err
	}

	if err := uc.ensureChatQuotaAvailable(ctx, userID); err != nil {
		return nil, nil, err
	}

	userMessage := &entities.Message{
		ContextID: contextID,
		Sender:    messagesender.User,
		Content:   content,
	}
	if err := uc.persistUserMessage(ctx, sessionCtx.session, userMessage); err != nil {
		return nil, nil, err
	}

	generationInput, err := uc.buildChatGenerationInput(ctx, userID, sessionCtx, content)
	if err != nil {
		return nil, nil, err
	}
	generationInput.userContent = content
	aiTextChan, aiErrChan := uc.aiService.GenerateChatTextStream(ctx, generationInput.systemPrompt, generationInput.userContent)
	return uc.createAIStreamResponse(ctx, userID, sessionCtx.session, aiTextChan, aiErrChan)
}

func (uc *WardrobeChatUseCase) compressChatContext(ctx context.Context, session *entities.ConversationalContext) error {
	count, err := uc.messageRepo.CountUnsummarizedByContextID(ctx, session.ID)
	if err != nil || count < 15 {
		return err
	}
	oldest, err := uc.messageRepo.GetOldestUnsummarizedByContextID(ctx, session.ID, 10)
	if err != nil || len(oldest) < 10 {
		return err
	}
	var builder strings.Builder
	if strings.TrimSpace(session.ContextSummary) != "" {
		builder.WriteString(session.ContextSummary)
		builder.WriteString("\n")
	}
	for _, item := range oldest {
		builder.WriteString(string(item.Sender))
		builder.WriteString(": ")
		builder.WriteString(item.Content)
		builder.WriteString("\n")
	}
	summary, err := uc.aiService.GenerateChatText(ctx, "You are an AI assistant summarizing fashion chat conversations in Vietnamese. Keep it concise and retain key style preferences.", builder.String())
	if err != nil || strings.TrimSpace(summary) == "" {
		summary = builder.String()
	}
	session.ContextSummary = summary
	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.contextRepo.Update(txCtx, session); err != nil {
			return err
		}
		ids := make([]uuid.UUID, len(oldest))
		for i, item := range oldest {
			ids[i] = item.ID
		}
		return uc.messageRepo.MarkAsSummarized(txCtx, ids)
	})
}

func (uc *WardrobeChatUseCase) loadChatSessionContext(ctx context.Context, userID uuid.UUID, contextID uuid.UUID) (*chatSessionContext, error) {
	session, err := uc.contextRepo.GetByID(ctx, contextID)
	if err != nil {
		return nil, err
	}
	if session == nil || session.UserID != userID {
		return nil, wardrobeerrors.ErrChatNotFound()
	}
	recent, err := uc.messageRepo.GetRecentByContextID(ctx, contextID, 5)
	if err != nil {
		return nil, err
	}
	return &chatSessionContext{session: session, recent: recent}, nil
}

func (uc *WardrobeChatUseCase) ensureChatQuotaAvailable(ctx context.Context, userID uuid.UUID) error {
	quota, err := uc.userQuotaCtr.GetAndResetDailyQuota(ctx, userID)
	if err != nil {
		return err
	}
	if quota.AiUsageCount >= quota.AiChatDailyQuota {
		return subscriptionerrors.ErrAiChatQuotaExceeded()
	}
	return nil
}

func (uc *WardrobeChatUseCase) buildChatGenerationInput(ctx context.Context, userID uuid.UUID, sessionCtx *chatSessionContext, content string) (*chatGenerationInput, error) {
	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}

	var activeWardrobeItems []*entities.WardrobeItem
	if isWardrobeRelatedQuery(content, sessionCtx.recent) {
		wardrobeItems, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
		if err != nil {
			return nil, err
		}
		activeWardrobeItems = shared.FilterActiveItems(wardrobeItems, subOverview.MaxWardrobeItems)
	}

	return &chatGenerationInput{systemPrompt: buildChatSystemPrompt(sessionCtx.session.ContextSummary, activeWardrobeItems, sessionCtx.recent)}, nil
}

func (uc *WardrobeChatUseCase) createAIStreamResponse(ctx context.Context, userID uuid.UUID, session *entities.ConversationalContext, aiTextChan <-chan string, aiErrChan <-chan error) (<-chan string, func(success bool) error, error) {
	var fullResponseBuilder strings.Builder
	outTextChan := FilterThinkTags(aiTextChan, func(chunk string) {
		fullResponseBuilder.WriteString(chunk)
	})
	commitCallback := func(success bool) error {
		if !success {
			return nil
		}
		select {
		case err, ok := <-aiErrChan:
			if ok && err != nil {
				return err
			}
		default:
		}
		fullResponse := fullResponseBuilder.String()
		if fullResponse == "" {
			return apperror.NewInternalError("Không thể nhận phản hồi từ hệ thống trí tuệ nhân tạo lúc này.")
		}
		aiMessage := &entities.Message{ContextID: session.ID, Sender: messagesender.AI, Content: fullResponse}
		if err := uc.persistAiResponse(ctx, userID, session, aiMessage, true); err != nil {
			return err
		}
		go func() {
			_ = uc.compressChatContext(context.Background(), session)
		}()
		return nil
	}
	return outTextChan, commitCallback, nil
}

func (uc *WardrobeChatUseCase) persistUserMessage(ctx context.Context, session *entities.ConversationalContext, userMessage *entities.Message) error {
	nonCancelledCtx := context.WithoutCancel(ctx)
	return uc.uow.Execute(nonCancelledCtx, func(txCtx context.Context) error {
		if err := uc.messageRepo.Create(txCtx, userMessage); err != nil {
			return err
		}
		session.UpdatedAt = time.Now()
		return uc.contextRepo.Update(txCtx, session)
	})
}

func (uc *WardrobeChatUseCase) persistAiResponse(ctx context.Context, userID uuid.UUID, session *entities.ConversationalContext, aiMessage *entities.Message, shouldConsumeQuota bool) error {
	nonCancelledCtx := context.WithoutCancel(ctx)
	return uc.uow.Execute(nonCancelledCtx, func(txCtx context.Context) error {
		if shouldConsumeQuota {
			if err := uc.userQuotaCtr.UpdateAiChatQuota(txCtx, userID, 1); err != nil {
				return err
			}
		}
		if err := uc.messageRepo.Create(txCtx, aiMessage); err != nil {
			return err
		}
		session.UpdatedAt = time.Now()
		return uc.contextRepo.Update(txCtx, session)
	})
}
