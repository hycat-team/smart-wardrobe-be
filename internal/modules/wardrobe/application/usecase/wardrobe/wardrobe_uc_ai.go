package wardrobe

import (
	"context"
	"strings"
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/shared/domain/constants/messagesender"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

func (uc *WardrobeAIUseCase) RecommendOutfit(ctx context.Context, userID uuid.UUID, input dto.RecommendOutfitReq) (*dto.RecommendedOutfitRes, error) {
	if err := uc.userQuotaCtr.UpdateOutfitQuota(ctx, userID, 1); err != nil {
		return nil, err
	}

	items, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
	if err != nil {
		return nil, err
	}

	activeItems := make([]*entities.WardrobeItem, 0, len(items))
	for _, item := range items {
		if item.Status == wardrobestatus.InWardrobe {
			activeItems = append(activeItems, item)
		}
	}

	if len(activeItems) == 0 {
		return nil, wardrobeerrors.ErrNoSuitableItemsForOutfit
	}

	categoryPicked := map[string]bool{}
	groups := make([]*dto.RecommendedItemGroup, 0)
	for _, item := range activeItems {
		categoryName := "Trang phuc"
		categoryKey := "uncategorized"
		if item.Category != nil {
			categoryName = item.Category.Name
			categoryKey = item.Category.Slug
		}
		if categoryPicked[categoryKey] {
			continue
		}

		group := &dto.RecommendedItemGroup{
			Role:         categoryName,
			Primary:      mapper.MapToWardrobeItemRes(item),
			Alternatives: make([]*dto.WardrobeItemRes, 0, 2),
		}
		for _, candidate := range activeItems {
			if candidate.ID == item.ID || candidate.CategoryID == nil || item.CategoryID == nil {
				continue
			}
			if *candidate.CategoryID == *item.CategoryID {
				group.Alternatives = append(group.Alternatives, mapper.MapToWardrobeItemRes(candidate))
			}
			if len(group.Alternatives) == 2 {
				break
			}
		}

		groups = append(groups, group)
		categoryPicked[categoryKey] = true
		if len(groups) == 4 {
			break
		}
	}

	if len(groups) == 0 {
		return nil, wardrobeerrors.ErrNoOutfitsFound
	}

	systemPrompt := "Bạn là stylist thời trang. Hãy viết giải thích ngắn gọn bằng tiếng Việt cho một bộ phối đồ duy nhất được tạo từ tủ đồ sẵn có của người dùng."
	userPrompt := buildRecommendationPrompt(groups, input)
	explanation, err := uc.aiService.GenerateText(ctx, systemPrompt, userPrompt)
	if err != nil || strings.TrimSpace(explanation) == "" {
		explanation = "Bộ phối đồ này ưu tiên sự linh hoạt và khả năng thay thế từng món ngay trên giao diện mà không cần gọi lại backend."
	}

	title := "Gợi ý phối đồ"
	if input.Occasion != nil && strings.TrimSpace(*input.Occasion) != "" {
		title = "Gợi ý phối đồ cho " + strings.TrimSpace(*input.Occasion)
	}

	return &dto.RecommendedOutfitRes{
		Title:       title,
		Explanation: explanation,
		Items:       groups,
	}, nil
}

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

func (uc *WardrobeAIUseCase) GetChatMessages(ctx context.Context, userID uuid.UUID, contextID uuid.UUID) ([]*dto.ChatMessageRes, error) {
	session, err := uc.contextRepo.GetByID(ctx, contextID)
	if err != nil {
		return nil, err
	}
	if session == nil || session.UserID != userID {
		return nil, wardrobeerrors.ErrChatNotFound
	}

	items, err := uc.messageRepo.GetByContextID(ctx, contextID)
	if err != nil {
		return nil, err
	}

	res := make([]*dto.ChatMessageRes, len(items))
	for i, item := range items {
		res[i] = mapChatMessage(item)
	}
	return res, nil
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
		wardrobeItems, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
		if err != nil {
			return nil, nil, err
		}

		systemPrompt := buildChatSystemPrompt(session.ContextSummary, wardrobeItems, recent)
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
		"Bạn là trợ lý có nhiệm vụ tóm tắt hội thoại thời trang bằng tiếng Việt, ngắn gọn và giữ đủ các thông tin gu mặc quan trọng.",
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
