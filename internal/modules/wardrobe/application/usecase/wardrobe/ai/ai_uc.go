package ai

import (
	"context"
	"encoding/json"
	"strings"

	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/ai"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/utils/stringutils"

	"github.com/google/uuid"
)

type LLMOutfitResponse struct {
	Title       string `json:"title"`
	Explanation string `json:"explanation"`
	Items       []struct {
		Role           string   `json:"role"`
		PrimaryID      string   `json:"primary_id"`
		AlternativeIDs []string `json:"alternative_ids"`
	} `json:"items"`
}

type WardrobeAIUseCase struct {
	contextRepo     repositories.IConversationalContextRepository
	messageRepo     repositories.IMessageRepository
	wardrobeRepo    repositories.IWardrobeItemRepository
	categoryRepo    repositories.ICategoryRepository
	aiService       ai.IAIService
	userSubContract contract.IUserSubscriptionContract
	userQuotaCtr    contract.IUserQuotaContract
	uow             shared_repos.IUnitOfWork
}

func NewWardrobeAIUseCase(
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
		contextRepo:     contextRepo,
		messageRepo:     messageRepo,
		wardrobeRepo:    wardrobeRepo,
		categoryRepo:    categoryRepo,
		aiService:       aiService,
		userSubContract: userSubContract,
		userQuotaCtr:    userQuotaCtr,
		uow:             uow,
	}
}

func (uc *WardrobeAIUseCase) RecommendOutfit(ctx context.Context, userID uuid.UUID, input dto.RecommendOutfitReq) (*dto.RecommendedOutfitRes, error) {
	// Giai đoạn 1: Kiểm tra Quota & Kiểm tra số lượng tối thiểu
	quotaDTO, err := uc.userQuotaCtr.GetAndResetDailyQuota(ctx, userID)
	if err != nil {
		return nil, err
	}
	if quotaDTO.OutfitRecommendCount >= quotaDTO.AiOutfitDailyQuota {
		return nil, subscriptionerrors.ErrAiOutfitQuotaExceeded
	}

	items, err := uc.wardrobeRepo.GetByUserID(ctx, userID, nil)
	if err != nil {
		return nil, err
	}

	activeItems := make([]*entities.WardrobeItem, 0, len(items))
	for _, item := range items {
		if item.Status == wardrobestatus.InWardrobe && !item.IsDeleted {
			activeItems = append(activeItems, item)
		}
	}

	if len(activeItems) < 5 {
		return nil, wardrobeerrors.ErrMinimumWardrobeItemsRequired
	}

	// Giai đoạn 2: Lọc thô ứng viên (Smart Filtering)
	var candidates []*entities.WardrobeItem

	if len(activeItems) <= 15 {
		if input.ColorTone != nil && *input.ColorTone != "" {
			tone := strings.ToLower(*input.ColorTone)
			filtered := make([]*entities.WardrobeItem, 0)
			for _, item := range activeItems {
				if item.ColorLightness != nil {
					l := *item.ColorLightness
					if (tone == "dark" || strings.Contains(tone, "trầm") || strings.Contains(tone, "tối")) && l < 40.0 {
						filtered = append(filtered, item)
					} else if (tone == "light" || strings.Contains(tone, "sáng")) && l >= 60.0 {
						filtered = append(filtered, item)
					}
				}
			}
			if len(filtered) > 0 {
				candidates = filtered
			} else {
				candidates = activeItems
			}
		} else {
			candidates = activeItems
		}
	} else {
		categories, err := uc.categoryRepo.GetAll(ctx)
		if err != nil {
			return nil, err
		}

		catMap := make(map[string]uuid.UUID)
		for _, cat := range categories {
			catMap[cat.Slug] = cat.ID
		}

		targets := []struct {
			slug  string
			limit int
		}{
			{"ao", 5},
			{"quan", 5},
			{"giay", 3},
			{"ao-khoac", 3},
			{"vay", 2},
			{"mu", 1},
			{"phu-kien", 1},
		}

		var queryVector entities.Vector
		useVectorSearch := false
		if input.Details != nil && strings.TrimSpace(*input.Details) != "" {
			vecs, err := uc.aiService.GenerateEmbeddings(ctx, []string{*input.Details})
			if err == nil && len(vecs) > 0 && len(vecs[0]) == 768 {
				queryVector = entities.Vector(vecs[0])
				useVectorSearch = true
			}
		}

		catCandidates := make(map[string][]*entities.WardrobeItem)
		for _, tgt := range targets {
			catID, exists := catMap[tgt.slug]
			if !exists {
				continue
			}

			var catItems []*entities.WardrobeItem
			var catErr error

			if useVectorSearch {
				catItems, catErr = uc.wardrobeRepo.GetSimilarItemsByVectorAndCategory(ctx, userID, catID, queryVector, tgt.limit)
			} else {
				catItems, catErr = uc.wardrobeRepo.GetRecentlyActiveItemsByCategory(ctx, userID, catID, tgt.limit)
			}

			if catErr == nil {
				if input.ColorTone != nil && *input.ColorTone != "" {
					tone := strings.ToLower(*input.ColorTone)
					var filtered []*entities.WardrobeItem
					for _, item := range catItems {
						if item.ColorLightness != nil {
							l := *item.ColorLightness
							if (tone == "dark" || strings.Contains(tone, "trầm") || strings.Contains(tone, "tối")) && l < 40.0 {
								filtered = append(filtered, item)
							} else if (tone == "light" || strings.Contains(tone, "sáng")) && l >= 60.0 {
								filtered = append(filtered, item)
							}
						}
					}
					if len(filtered) > 0 {
						catItems = filtered
					}
				}
				catCandidates[tgt.slug] = catItems
			}
		}

		for _, catItems := range catCandidates {
			candidates = append(candidates, catItems...)
		}

		// Backfill
		if len(candidates) < 20 && len(candidates) < len(activeItems) {
			takenIDs := make(map[uuid.UUID]bool)
			for _, c := range candidates {
				takenIDs[c.ID] = true
			}

			for _, item := range activeItems {
				if !takenIDs[item.ID] {
					if input.ColorTone != nil && *input.ColorTone != "" {
						tone := strings.ToLower(*input.ColorTone)
						if item.ColorLightness != nil {
							l := *item.ColorLightness
							isDarkTone := tone == "dark" || strings.Contains(tone, "trầm") || strings.Contains(tone, "tối")
							isLightTone := tone == "light" || strings.Contains(tone, "sáng")
							if (isDarkTone && l >= 40.0) || (isLightTone && l < 60.0) {
								continue
							}
						}
					}
					candidates = append(candidates, item)
					takenIDs[item.ID] = true
					if len(candidates) >= 20 {
						break
					}
				}
			}
		}

		if len(candidates) > 20 {
			candidates = candidates[:20]
		}
	}

	// Giai đoạn 3: Phối đồ tinh tế (AI & Fallback HSL)
	systemPrompt := "You are a professional AI fashion stylist. Return ONLY a valid JSON payload for the outfit recommendation. Do not include markdown code block formatting."
	userPrompt := buildRecommendationPrompt(candidates, input)

	var groups []*dto.RecommendedItemGroup
	var title string
	var explanation string
	var llmErr error

	responseText, llmErr := uc.aiService.GenerateText(ctx, systemPrompt, userPrompt)

	if llmErr == nil && responseText != "" {
		cleanedJSON := stringutils.CleanJSONMarkdown(responseText)
		var llmRes LLMOutfitResponse
		if err := json.Unmarshal([]byte(cleanedJSON), &llmRes); err == nil {
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

			if len(validGroups) > 0 {
				groups = validGroups
				title = llmRes.Title
				explanation = llmRes.Explanation
			} else {
				llmErr = wardrobeerrors.ErrInvalidOutfitStructure
			}
		} else {
			llmErr = err
		}
	}

	var finalRes *dto.RecommendedOutfitRes
	if llmErr != nil || len(groups) == 0 {
		finalRes = runLocalHSLMatching(candidates, input)
	} else {
		finalRes = &dto.RecommendedOutfitRes{
			Title:       title,
			Explanation: explanation,
			Items:       groups,
			IsFallback:  false,
		}
	}

	// Giai đoạn 4: Trừ Quota sau khi thành công
	if err := uc.userQuotaCtr.UpdateOutfitQuota(ctx, userID, 1); err != nil {
		return nil, err
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

	return finalRes, nil
}
