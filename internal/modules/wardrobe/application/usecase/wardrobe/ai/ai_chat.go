package ai

import (
	"context"
	"strings"
	"time"

	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/shared"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/internal/shared/domain/constants/messagesender"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

func (uc *WardrobeAIUseCase) CreateChatSession(ctx context.Context, userID uuid.UUID, title *string) (*dto.ChatSessionRes, error) {
	sessionTitle := "Cuộc trò chuyện mới"
	if title != nil && strings.TrimSpace(*title) != "" {
		sessionTitle = strings.TrimSpace(*title)
	}

	entity := &entities.ConversationalContext{
		UserID:         userID,
		Title:          sessionTitle,
		ContextSummary: "",
		IsArchived:     false,
	}
	if err := uc.contextRepo.Create(ctx, entity); err != nil {
		return nil, err
	}

	return mapChatSession(entity), nil
}

func (uc *WardrobeAIUseCase) GetChatSessions(ctx context.Context, userID uuid.UUID) ([]*dto.ChatSessionRes, error) {
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

func (uc *WardrobeAIUseCase) GetChatMessages(ctx context.Context, userID uuid.UUID, contextID uuid.UUID, query dto.GetChatMessagesQueryReq) (*shared_dto.PaginationResult[*dto.ChatMessageRes], error) {
	session, err := uc.contextRepo.GetByID(ctx, contextID)
	if err != nil {
		return nil, err
	}
	if session == nil || session.UserID != userID {
		return nil, wardrobeerrors.ErrChatNotFound
	}

	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}

	totalItems, err := uc.messageRepo.CountByContextID(ctx, contextID)
	if err != nil {
		return nil, err
	}

	paginationQuery := shared_dto.PaginationQuery{
		Page:  page,
		Limit: limit,
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
		Metadata: shared_dto.BuildPaginationMetadata(query.PaginationQuery, totalItems),
	}, nil
}

func (uc *WardrobeAIUseCase) ArchiveChatSession(ctx context.Context, userID uuid.UUID, contextID uuid.UUID) error {
	session, err := uc.contextRepo.GetByID(ctx, contextID)
	if err != nil {
		return err
	}
	if session == nil || session.UserID != userID {
		return wardrobeerrors.ErrChatNotFound
	}

	session.IsArchived = true
	return uc.contextRepo.Update(ctx, session)
}

func (uc *WardrobeAIUseCase) ProcessChatMessageStream(
	ctx context.Context,
	userID uuid.UUID,
	contextID uuid.UUID,
	content string,
) (<-chan string, func(success bool) error, error) {
	// 1. Soft quota check
	quota, err := uc.userQuotaCtr.GetAndResetDailyQuota(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	if quota.AiUsageCount >= quota.AiChatDailyQuota {
		return nil, nil, subscriptionerrors.ErrAiChatQuotaExceeded
	}

	// 2. Fetch session and verify ownership
	session, err := uc.contextRepo.GetByID(ctx, contextID)
	if err != nil {
		return nil, nil, err
	}
	if session == nil || session.UserID != userID {
		return nil, nil, wardrobeerrors.ErrChatNotFound
	}

	// 3. Check for outfit recommendation intent (static reply fallback)
	if isOutfitIntent(content) {
		textChan := make(chan string, 1)
		textChan <- "Để nhận được gợi ý phối đồ chuẩn xác nhất từ thuật toán của Smart Wardrobe, bạn vui lòng sử dụng chức năng Phối đồ trên màn hình chính."
		close(textChan)

		commitCallback := func(success bool) error {
			if !success {
				return nil
			}

			userMessage := &entities.Message{
				ContextID: contextID,
				Sender:    messagesender.User,
				Content:   content,
			}
			aiMessage := &entities.Message{
				ContextID: contextID,
				Sender:    messagesender.AI,
				Content:   "Để nhận được gợi ý phối đồ chuẩn xác nhất từ thuật toán của Smart Wardrobe, bạn vui lòng sử dụng chức năng Phối đồ trên màn hình chính.",
			}

			return uc.uow.Execute(ctx, func(txCtx context.Context) error {
				if err := uc.userQuotaCtr.UpdateAiChatQuota(txCtx, userID, 1); err != nil {
					return err
				}
				if err := uc.messageRepo.Create(txCtx, userMessage); err != nil {
					return err
				}
				if err := uc.messageRepo.Create(txCtx, aiMessage); err != nil {
					return err
				}
				session.UpdatedAt = time.Now()
				if err := uc.contextRepo.Update(txCtx, session); err != nil {
					return err
				}
				return nil
			})
		}

		return textChan, commitCallback, nil
	}

	// 4. Generate system prompt using history and active wardrobe items
	recent, err := uc.messageRepo.GetRecentByContextID(ctx, contextID, 5)
	if err != nil {
		return nil, nil, err
	}

	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	wardrobeItems, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
	if err != nil {
		return nil, nil, err
	}

	activeWardrobeItems := shared.FilterActiveItems(wardrobeItems, subOverview.MaxWardrobeItems)
	systemPrompt := buildChatSystemPrompt(session.ContextSummary, activeWardrobeItems, recent)

	// 5. Call stream generation in AI Service
	aiTextChan, aiErrChan := uc.aiService.GenerateTextStream(ctx, systemPrompt, content)

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

		userMessage := &entities.Message{
			ContextID: contextID,
			Sender:    messagesender.User,
			Content:   content,
		}
		aiMessage := &entities.Message{
			ContextID: contextID,
			Sender:    messagesender.AI,
			Content:   fullResponse,
		}

		err = uc.uow.Execute(ctx, func(txCtx context.Context) error {
			if err := uc.userQuotaCtr.UpdateAiChatQuota(txCtx, userID, 1); err != nil {
				return err
			}
			if err := uc.messageRepo.Create(txCtx, userMessage); err != nil {
				return err
			}
			if err := uc.messageRepo.Create(txCtx, aiMessage); err != nil {
				return err
			}
			session.UpdatedAt = time.Now()
			if err := uc.contextRepo.Update(txCtx, session); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}

		_ = uc.compressChatContext(ctx, session)
		return nil
	}

	return outTextChan, commitCallback, nil
}

func (uc *WardrobeAIUseCase) ProcessChatMessage(ctx context.Context, userID uuid.UUID, contextID uuid.UUID, content string) (*dto.ChatMessageRes, *dto.ChatMessageRes, error) {
	if err := uc.userQuotaCtr.UpdateAiChatQuota(ctx, userID, 1); err != nil {
		return nil, nil, err
	}

	session, err := uc.contextRepo.GetByID(ctx, contextID)
	if err != nil {
		return nil, nil, err
	}
	if session == nil || session.UserID != userID {
		return nil, nil, wardrobeerrors.ErrChatNotFound
	}

	userMessage := &entities.Message{
		ContextID: contextID,
		Sender:    messagesender.User,
		Content:   content,
	}

	recent, err := uc.messageRepo.GetRecentByContextID(ctx, contextID, 5)
	if err != nil {
		return nil, nil, err
	}

	var responseText string
	if isOutfitIntent(content) {
		responseText = "Để nhận được gợi ý phối đồ chuẩn xác nhất từ thuật toán của Smart Wardrobe, bạn vui lòng sử dụng chức năng Phối đồ trên màn hình chính."
	} else {
		subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
		if err != nil {
			return nil, nil, err
		}

		wardrobeItems, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
		if err != nil {
			return nil, nil, err
		}

		activeWardrobeItems := shared.FilterActiveItems(wardrobeItems, subOverview.MaxWardrobeItems)

		systemPrompt := buildChatSystemPrompt(session.ContextSummary, activeWardrobeItems, recent)
		responseText, err = uc.aiService.GenerateText(ctx, systemPrompt, content)
		if err != nil {
			return nil, nil, err
		}
	}

	aiMessage := &entities.Message{
		ContextID: contextID,
		Sender:    messagesender.AI,
		Content:   responseText,
	}

	createMessages := func(txCtx context.Context) error {
		if err := uc.messageRepo.Create(txCtx, userMessage); err != nil {
			return err
		}

		if err := uc.messageRepo.Create(txCtx, aiMessage); err != nil {
			return err
		}

		session.UpdatedAt = time.Now()
		if err := uc.contextRepo.Update(txCtx, session); err != nil {
			return err
		}

		return nil
	}

	if err = uc.uow.Execute(ctx, createMessages); err != nil {
		return nil, nil, err
	}

	if err = uc.compressChatContext(ctx, session); err != nil {
		return nil, nil, err
	}

	return mapChatMessage(userMessage), mapChatMessage(aiMessage), nil
}

func (uc *WardrobeAIUseCase) compressChatContext(ctx context.Context, session *entities.ConversationalContext) error {
	count, err := uc.messageRepo.CountByContextID(ctx, session.ID)
	if err != nil {
		return err
	}
	if count < 10 {
		return nil
	}

	oldest, err := uc.messageRepo.GetOldestByContextID(ctx, session.ID, 10)
	if err != nil {
		return err
	}
	if len(oldest) < 10 {
		return nil
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

	summary, err := uc.aiService.GenerateText(
		ctx,
		"You are an AI assistant summarizing fashion chat conversations in Vietnamese. Keep it concise and retain key style preferences.",
		builder.String(),
	)
	if err != nil || strings.TrimSpace(summary) == "" {
		summary = builder.String()
	}

	session.ContextSummary = summary

	updateSession := func(txCtx context.Context) error {
		if err := uc.contextRepo.Update(txCtx, session); err != nil {
			return err
		}

		ids := make([]uuid.UUID, len(oldest))
		for i, item := range oldest {
			ids[i] = item.ID
		}

		return uc.messageRepo.DeleteByIDs(txCtx, ids)
	}

	return uc.uow.Execute(ctx, updateSession)
}
