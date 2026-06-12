package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"smart-wardrobe-be/config"
	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/shared"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/ai"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/utils/stringutils"

	"github.com/google/uuid"
)

type llmOutfitResponse struct {
	Title       string `json:"title"`
	Explanation string `json:"explanation"`
	Items       []struct {
		Role           string   `json:"role"`
		PrimaryID      string   `json:"primary_id"`
		AlternativeIDs []string `json:"alternative_ids"`
	} `json:"items"`
}

type WardrobeAIUseCase struct {
	cfg             *config.Config
	contextRepo     repositories.IConversationalContextRepository
	messageRepo     repositories.IMessageRepository
	wardrobeRepo    repositories.IWardrobeItemRepository
	categoryRepo    repositories.ICategoryRepository
	aiService       ai.IAIService
	userSubContract contract.IUserSubscriptionContract
	userQuotaCtr    contract.IUserQuotaContract
	uow             shared_repos.IUnitOfWork
	nlpParser       *LocalNLPParser
}

func NewWardrobeAIUseCase(
	cfg *config.Config,
	contextRepo repositories.IConversationalContextRepository,
	messageRepo repositories.IMessageRepository,
	wardrobeRepo repositories.IWardrobeItemRepository,
	categoryRepo repositories.ICategoryRepository,
	aiService ai.IAIService,
	userSubContract contract.IUserSubscriptionContract,
	userQuotaCtr contract.IUserQuotaContract,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IWardrobeAIUseCase {
	return &WardrobeAIUseCase{
		cfg:             cfg,
		contextRepo:     contextRepo,
		messageRepo:     messageRepo,
		wardrobeRepo:    wardrobeRepo,
		categoryRepo:    categoryRepo,
		aiService:       aiService,
		userSubContract: userSubContract,
		userQuotaCtr:    userQuotaCtr,
		uow:             uow,
		nlpParser:       NewLocalNLPParser(),
	}
}

func (uc *WardrobeAIUseCase) RecommendOutfit(ctx context.Context, userID uuid.UUID, input dto.RecommendOutfitReq) (*dto.RecommendedOutfitRes, error) {
	// Giai đoạn 1: Kiểm tra Quota & Kiểm tra số lượng tối thiểu
	activeItems, quotaDTO, err := uc.validateAndGetActiveItems(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Giai đoạn 2: Lọc thô ứng viên (Smart Filtering)
	candidates, err := uc.filterCandidates(ctx, userID, activeItems, input)
	if err != nil {
		return nil, err
	}

	// Giai đoạn 3: Phối đồ tinh tế (AI & Fallback HSL)
	finalRes, err := uc.generateOutfitRecommendation(ctx, candidates, input)
	if err != nil {
		finalRes = runLocalHSLMatching(candidates, input)
	}

	// Giai đoạn 4: Trừ Quota sau khi thành công
	if err := uc.updateQuotaAndConstructResponse(ctx, userID, finalRes, quotaDTO); err != nil {
		return nil, err
	}

	return finalRes, nil
}

// validateAndGetActiveItems performs Stage 1: Validation & Active Items Retrieval.
func (uc *WardrobeAIUseCase) validateAndGetActiveItems(ctx context.Context, userID uuid.UUID) ([]*entities.WardrobeItem, *contract.UserSubscriptionDTO, error) {
	quotaDTO, err := uc.userQuotaCtr.GetAndResetDailyQuota(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	if quotaDTO.OutfitRecommendCount >= quotaDTO.AiOutfitDailyQuota {
		return nil, nil, subscriptionerrors.ErrAiOutfitQuotaExceeded
	}

	items, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
	if err != nil {
		return nil, nil, err
	}

	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	items = shared.FilterActiveItems(items, subOverview.MaxWardrobeItems)
	activeItems := make([]*entities.WardrobeItem, 0, len(items))
	for _, item := range items {
		if item.Status == wardrobestatus.InWardrobe && !item.IsDeleted {
			activeItems = append(activeItems, item)
		}
	}

	if len(activeItems) < 5 {
		return nil, nil, wardrobeerrors.ErrMinimumWardrobeItemsRequired
	}

	return activeItems, quotaDTO, nil
}

// filterCandidates thực hiện Giai đoạn 2: Lọc ứng viên bằng Advanced RAG (Local NLP Parser + Hybrid Search + Re-ranking).
func (uc *WardrobeAIUseCase) filterCandidates(ctx context.Context, userID uuid.UUID, activeItems []*entities.WardrobeItem, input dto.RecommendOutfitReq) ([]*entities.WardrobeItem, error) {
	// Chạy bộ phân tách ý định Local NLP Parser từ thông tin chi tiết dạng free-text
	freeText := ""
	if input.Details != nil {
		freeText = *input.Details
	}
	intent := uc.nlpParser.Parse(freeText)

	// Gộp ý định đã phân tách với các bộ lọc rõ ràng được chọn từ giao diện người dùng (nếu có)
	if input.Occasion != nil && *input.Occasion != "" {
		intent.Occasion = *input.Occasion
	}
	if input.ColorTone != nil && *input.ColorTone != "" {
		intent.ColorTone = *input.ColorTone
	}

	// Tạo embedding cho truy vấn với timeout tối đa 2 giây để đảm bảo hệ thống ổn định
	var queryVector entities.Vector
	if intent.SemanticQuery != "" {
		embedCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		vecs, err := uc.aiService.GenerateEmbeddings(embedCtx, []string{intent.SemanticQuery})
		if err == nil && len(vecs) > 0 && len(vecs[0]) == 768 {
			queryVector = entities.Vector(vecs[0])
		}
	}

	// Thực hiện tìm kiếm kết hợp (Hybrid Search) để lấy ra tối đa 40 ứng viên từ cơ sở dữ liệu
	candidates, err := uc.wardrobeRepo.GetHybridCandidates(ctx, userID, queryVector, intent.ExactKeywords, 40)
	if err != nil {
		return nil, err
	}

	// Nếu số lượng ứng viên tìm được ít hơn 20, bù đắp từ danh sách đồ đang hoạt động khác để đảm bảo đủ lựa chọn
	if len(candidates) < 20 && len(candidates) < len(activeItems) {
		takenIDs := make(map[uuid.UUID]bool)
		for _, c := range candidates {
			takenIDs[c.ID] = true
		}
		for _, item := range activeItems {
			if !takenIDs[item.ID] {
				candidates = append(candidates, item)
				takenIDs[item.ID] = true
				if len(candidates) >= 20 {
					break
				}
			}
		}
	}

	type candidateScore struct {
		item  *entities.WardrobeItem
		score float64
		tags  []string
	}

	recentlyWornDays := uc.cfg.RAG.RecentlyWornPenaltyDays
	longUnwornDays := uc.cfg.RAG.LongUnwornBonusDays

	// Chấm điểm Re-ranking dựa trên các quy tắc thời trang (phù hợp phong cách, dịp lễ, thời tiết, tần suất mặc)
	scored := make([]candidateScore, len(candidates))
	for i, item := range candidates {
		score, tags := scoreCandidateItem(item, intent, recentlyWornDays, longUnwornDays)
		scored[i] = candidateScore{
			item:  item,
			score: score,
			tags:  tags,
		}
	}

	// Sắp xếp ứng viên giảm dần theo điểm số phù hợp
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Giới hạn trong khoảng 15-20 ứng viên để tối ưu token truyền cho prompt của LLM Stylist
	limit := min(len(scored), 20)
	if limit < 15 && len(scored) >= 15 {
		limit = 15
	}

	// Tạo danh sách cuối cùng và chèn các tag giải thích vào Description trong bộ nhớ để LLM Stylist hiểu lý do phối
	finalCandidates := make([]*entities.WardrobeItem, limit)
	for i := 0; i < limit; i++ {
		finalCandidates[i] = scored[i].item
		if len(scored[i].tags) > 0 {
			tagsStr := strings.Join(scored[i].tags, ", ")
			desc := ""
			if finalCandidates[i].Description != nil {
				desc = *finalCandidates[i].Description
			}
			newDesc := desc + " [Fashion Tags: " + tagsStr + "]"
			finalCandidates[i].Description = &newDesc
		}
	}

	return finalCandidates, nil
}

// generateOutfitRecommendation performs Stage 3: AI Recommendation & Parsing.
func (uc *WardrobeAIUseCase) generateOutfitRecommendation(ctx context.Context, candidates []*entities.WardrobeItem, input dto.RecommendOutfitReq) (*dto.RecommendedOutfitRes, error) {
	systemPrompt := "You are a professional AI fashion stylist. Return ONLY a valid JSON payload for the outfit recommendation. Do not include markdown code block formatting."
	userPrompt := buildRecommendationPrompt(candidates, input)

	responseText, llmErr := uc.aiService.GenerateText(ctx, systemPrompt, userPrompt)
	if llmErr != nil {
		return nil, llmErr
	}
	if responseText == "" {
		return nil, fmt.Errorf("empty response from LLM")
	}

	cleanedJSON := stringutils.CleanJSONMarkdown(responseText)
	var llmRes llmOutfitResponse
	if err := json.Unmarshal([]byte(cleanedJSON), &llmRes); err != nil {
		return nil, err
	}

	candidateMap := make(map[uuid.UUID]*entities.WardrobeItem)
	for _, c := range candidates {
		candidateMap[c.ID] = c
	}

	validGroups := make([]*dto.RecommendedItemGroup, 0)
	for _, groupItem := range llmRes.Items {
		role := strings.ToLower(groupItem.Role)
		primaryUUID, pErr := uuid.Parse(groupItem.PrimaryID)
		var primaryItem *entities.WardrobeItem
		if pErr == nil {
			primaryItem = candidateMap[primaryUUID]
		}

		if primaryItem == nil {
			for _, c := range candidates {
				if c.Category != nil && c.Category.Slug == role {
					primaryItem = c
					break
				}
			}
		}

		if primaryItem == nil {
			continue
		}

		alternatives := make([]*dto.WardrobeItemRes, 0)
		for _, altID := range groupItem.AlternativeIDs {
			altUUID, altErr := uuid.Parse(altID)
			if altErr == nil {
				if altItem, ok := candidateMap[altUUID]; ok && altItem.Category != nil && altItem.Category.Slug == role && altItem.ID != primaryItem.ID {
					alternatives = append(alternatives, mapper.MapToWardrobeItemRes(altItem))
				}
			}
			if len(alternatives) == 2 {
				break
			}
		}

		if len(alternatives) < 2 {
			for _, c := range candidates {
				if c.ID != primaryItem.ID && c.Category != nil && c.Category.Slug == role {
					alreadyAdded := false
					for _, a := range alternatives {
						if a.ID == c.ID {
							alreadyAdded = true
							break
						}
					}
					if !alreadyAdded {
						alternatives = append(alternatives, mapper.MapToWardrobeItemRes(c))
					}
				}
				if len(alternatives) == 2 {
					break
				}
			}
		}

		validGroups = append(validGroups, &dto.RecommendedItemGroup{
			Role:         role,
			Primary:      mapper.MapToWardrobeItemRes(primaryItem),
			Alternatives: alternatives,
		})
	}

	if len(validGroups) == 0 {
		return nil, wardrobeerrors.ErrInvalidOutfitStructure
	}

	return &dto.RecommendedOutfitRes{
		Title:       llmRes.Title,
		Explanation: llmRes.Explanation,
		Items:       validGroups,
		IsFallback:  false,
	}, nil
}

// updateQuotaAndConstructResponse performs Stage 4: Quota Update & Response Construction.
func (uc *WardrobeAIUseCase) updateQuotaAndConstructResponse(ctx context.Context, userID uuid.UUID, finalRes *dto.RecommendedOutfitRes, quotaDTO *contract.UserSubscriptionDTO) error {
	if err := uc.userQuotaCtr.UpdateOutfitQuota(ctx, userID, 1); err != nil {
		return err
	}

	updatedQuota, err := uc.userQuotaCtr.GetAndResetDailyQuota(ctx, userID)
	if err == nil {
		finalRes.RemainingQuota = updatedQuota.AiOutfitDailyQuota - updatedQuota.OutfitRecommendCount
	} else {
		finalRes.RemainingQuota = quotaDTO.AiOutfitDailyQuota - (quotaDTO.OutfitRecommendCount + 1)
	}

	if finalRes.RemainingQuota < 0 {
		finalRes.RemainingQuota = 0
	}

	return nil
}
