package chat

import (
	"context"
	"strings"
	"time"

	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
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
	return mapChatSession(entity), nil
}

// GetChatSessions returns all chat sessions that belong to the user.
func (uc *WardrobeChatUseCase) GetChatSessions(ctx context.Context, userID uuid.UUID) ([]*dto.ChatSessionRes, error) {
	items, err := uc.contextRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	res := make([]*dto.ChatSessionRes, len(items))
	for i, item := range items {
		res[i] = mapChatSession(item)
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
		res[i] = mapChatMessage(item)
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
	if isOutfitIntent(content) {
		return uc.createRedirectStreamResponse(), uc.createRedirectCommitCallback(ctx, sessionCtx.session, content), nil
	}
	if err := uc.ensureChatQuotaAvailable(ctx, userID); err != nil {
		return nil, nil, err
	}
	generationInput, err := uc.buildChatGenerationInput(ctx, userID, sessionCtx)
	if err != nil {
		return nil, nil, err
	}
	generationInput.userContent = content
	aiTextChan, aiErrChan := uc.aiService.GenerateChatTextStream(ctx, generationInput.systemPrompt, generationInput.userContent)
	return uc.createAIStreamResponse(ctx, userID, sessionCtx.session, content, aiTextChan, aiErrChan)
}

// ProcessChatMessage generates a synchronous AI reply and persists the conversation.
func (uc *WardrobeChatUseCase) ProcessChatMessage(ctx context.Context, userID uuid.UUID, contextID uuid.UUID, content string) (*dto.ChatMessageRes, *dto.ChatMessageRes, error) {
	sessionCtx, err := uc.loadChatSessionContext(ctx, userID, contextID)
	if err != nil {
		return nil, nil, err
	}
	responseText, shouldConsumeQuota, err := uc.generateChatResponse(ctx, userID, sessionCtx, content)
	if err != nil {
		return nil, nil, err
	}
	userMessage := &entities.Message{ContextID: contextID, Sender: messagesender.User, Content: content}
	aiMessage := &entities.Message{ContextID: contextID, Sender: messagesender.AI, Content: responseText}
	if err = uc.persistChatExchange(ctx, userID, sessionCtx.session, userMessage, aiMessage, shouldConsumeQuota); err != nil {
		return nil, nil, err
	}
	if err = uc.compressChatContext(ctx, sessionCtx.session); err != nil {
		return nil, nil, err
	}
	return mapChatMessage(userMessage), mapChatMessage(aiMessage), nil
}

func (uc *WardrobeChatUseCase) compressChatContext(ctx context.Context, session *entities.ConversationalContext) error {
	count, err := uc.messageRepo.CountByContextID(ctx, session.ID)
	if err != nil || count < 10 {
		return err
	}
	oldest, err := uc.messageRepo.GetOldestByContextID(ctx, session.ID, 10)
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
		return uc.messageRepo.DeleteByIDs(txCtx, ids)
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

func (uc *WardrobeChatUseCase) buildChatGenerationInput(ctx context.Context, userID uuid.UUID, sessionCtx *chatSessionContext) (*chatGenerationInput, error) {
	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, err
	}
	wardrobeItems, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
	if err != nil {
		return nil, err
	}
	activeWardrobeItems := shared.FilterActiveItems(wardrobeItems, subOverview.MaxWardrobeItems)
	return &chatGenerationInput{systemPrompt: buildChatSystemPrompt(sessionCtx.session.ContextSummary, activeWardrobeItems, sessionCtx.recent)}, nil
}

func (uc *WardrobeChatUseCase) createRedirectStreamResponse() <-chan string {
	textChan := make(chan string, 1)
	textChan <- outfitRedirectMessage
	close(textChan)
	return textChan
}

func (uc *WardrobeChatUseCase) createRedirectCommitCallback(ctx context.Context, session *entities.ConversationalContext, content string) func(success bool) error {
	return func(success bool) error {
		if !success {
			return nil
		}
		userMessage := &entities.Message{ContextID: session.ID, Sender: messagesender.User, Content: content}
		aiMessage := &entities.Message{ContextID: session.ID, Sender: messagesender.AI, Content: outfitRedirectMessage}
		return uc.persistChatExchange(ctx, uuid.Nil, session, userMessage, aiMessage, false)
	}
}

func (uc *WardrobeChatUseCase) createAIStreamResponse(ctx context.Context, userID uuid.UUID, session *entities.ConversationalContext, content string, aiTextChan <-chan string, aiErrChan <-chan error) (<-chan string, func(success bool) error, error) {
	outTextChan := make(chan string, 100)
	var fullResponseBuilder strings.Builder
	go func() {
		defer close(outTextChan)
		for t := range aiTextChan {
			fullResponseBuilder.WriteString(t)
			outTextChan <- t
		}
	}()
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
		userMessage := &entities.Message{ContextID: session.ID, Sender: messagesender.User, Content: content}
		aiMessage := &entities.Message{ContextID: session.ID, Sender: messagesender.AI, Content: fullResponse}
		if err := uc.persistChatExchange(ctx, userID, session, userMessage, aiMessage, true); err != nil {
			return err
		}
		_ = uc.compressChatContext(ctx, session)
		return nil
	}
	return outTextChan, commitCallback, nil
}

func (uc *WardrobeChatUseCase) generateChatResponse(ctx context.Context, userID uuid.UUID, sessionCtx *chatSessionContext, content string) (string, bool, error) {
	if isOutfitIntent(content) {
		return outfitRedirectMessage, false, nil
	}
	if err := uc.ensureChatQuotaAvailable(ctx, userID); err != nil {
		return "", false, err
	}
	generationInput, err := uc.buildChatGenerationInput(ctx, userID, sessionCtx)
	if err != nil {
		return "", false, err
	}
	responseText, err := uc.aiService.GenerateChatText(ctx, generationInput.systemPrompt, content)
	return responseText, true, err
}

func (uc *WardrobeChatUseCase) persistChatExchange(ctx context.Context, userID uuid.UUID, session *entities.ConversationalContext, userMessage *entities.Message, aiMessage *entities.Message, shouldConsumeQuota bool) error {
	return uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if shouldConsumeQuota {
			if err := uc.userQuotaCtr.UpdateAiChatQuota(txCtx, userID, 1); err != nil {
				return err
			}
		}
		if err := uc.messageRepo.Create(txCtx, userMessage); err != nil {
			return err
		}
		if err := uc.messageRepo.Create(txCtx, aiMessage); err != nil {
			return err
		}
		session.UpdatedAt = time.Now()
		return uc.contextRepo.Update(txCtx, session)
	})
}
