package recommendation

import (
	"context"

	sub_dto "smart-wardrobe-be/services/account/dto/subscription"
	brand_dto "smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	fashioerrors "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/errors"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"
	wardrobeDto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/outfititemcontext"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/ranking"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/retrieval"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/synthesis"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"
	"smart-wardrobe-be/shared/pkg/utils/sliceutils"

	"github.com/google/uuid"
)

// candidateFilterResult is the internal output of the retrieval/ranking stage.
// It carries everything the suggestion builder needs to reuse the exact same
// semantic query, style vector and hard filters as the outfit itself
// (033 US2/T039-T040) without issuing a second embedding call.
type candidateFilterResult struct {
	Candidates      []types.CandidateForPrompt
	FashionItemMap  map[uuid.UUID]*dto.FashionItemContractDTO
	Personalization types.PersonalizationContext
	Pool            []types.CandidateForRanking
	RetrievalQuery  types.RecommendationRetrievalQuery
	QueryVector     entities.Vector
}

// validateAndGetActiveItems kiểm tra hạn ngạch hàng ngày của người dùng và truy vấn các món đồ
// đang hoạt động trong tủ đồ (không bị xóa mềm và có trạng thái là InWardrobe).
//
// Hành vi:
//   - 1. Gọi [GetAndResetDailyQuota] để kiểm tra số lượt gợi ý đã sử dụng. Nếu vượt quá giới hạn [AiOutfitDailyQuota], trả về lỗi [ErrAiOutfitQuotaExceeded].
//   - 2. Lấy danh sách tủ đồ qua [GetByUserID].
//   - 3. Lọc danh sách các món đồ hoạt động dựa theo gói đăng ký (subscription overview) của người dùng để giới hạn số lượng tối đa ([MaxWardrobeItems]).
//   - 4. Nếu số lượng đồ hoạt động ít hơn 5, trả về lỗi [ErrMinimumWardrobeItemsRequired].
//
// Đầu vào mẫu:
//
//	userID: "8e05c317-062e-4b47-ba21-12f5a04f21db"
//
// Đầu ra mẫu:
//
//	([]*wardrobeDto.WardrobeItemRes, *sub_dto.UserSubscriptionDTO, nil)
func (uc *OutfitRecommendationUseCase) validateAndGetActiveItems(
	ctx context.Context,
	userID uuid.UUID,
) ([]*wardrobeDto.WardrobeItemContractDTO, *sub_dto.UserSubscriptionDTO, error) {
	quotaDTO, err := uc.userQuotaCtr.GetAndResetDailyQuota(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	if quotaDTO.OutfitRecommendCount >= quotaDTO.AiOutfitDailyQuota {
		return nil, nil, fashioerrors.ErrAiOutfitQuotaExceeded()
	}

	subOverview, err := uc.userSubContract.GetUserSubscriptionOverview(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	// Lấy active items (InWardrobe + not deleted) qua contract — wardrobe module lọc + giới hạn từ DB,
	// không load toàn bộ wardrobe lên RAM rồi cắt.
	activeItems, err := uc.wardrobeContract.GetActiveWardrobeItemsByUserID(ctx, userID, subOverview.MaxWardrobeItems)
	if err != nil {
		return nil, nil, err
	}

	if len(activeItems) < 5 {
		return nil, nil, fashioerrors.ErrMinimumWardrobeItemsRequired()
	}

	return activeItems, quotaDTO, nil
}

// filterCandidates thực hiện quy trình tìm kiếm lai (hybrid search) và xếp hạng để lọc ra danh sách
// các ứng viên tiềm năng tốt nhất cho prompt gợi ý trang phục.
//
// Hành vi:
//   - 1. Phân tích ý định từ yêu cầu đầu vào thông qua [parseRecommendationIntent] để lấy [ParsedIntent] và xây dựng câu truy vấn ngữ nghĩa.
//   - 2. Tạo vector nhúng (embedding vector) cho câu truy vấn ngữ nghĩa bằng [buildRecommendationQueryVector].
//   - 3. Thực hiện truy vấn lai (kết hợp vector và lexical search) thông qua [GetHybridCandidates] của repository tủ đồ.
//   - 4. Đảm bảo số lượng ứng viên tối thiểu trong bể ứng viên thông qua [EnsureMinimumCandidatePool].
//   - 5. Tính toán điểm số và xếp hạng ứng viên dựa trên độ tương quan ngữ cảnh và tần suất mặc đồ qua [RankCandidates].
//
// Đầu vào mẫu:
//
//	userID: "8e05c317-062e-4b47-ba21-12f5a04f21db"
//	activeItems: []*wardrobe_entities.WardrobeItem{...}
//	input: dto.RecommendOutfitReq{Occasion: pointer to "casual"}
//
// Đầu ra mẫu:
//
//	(*candidateFilterResult, nil)
func (uc *OutfitRecommendationUseCase) filterCandidates(
	ctx context.Context,
	userID uuid.UUID,
	activeItems []*wardrobeDto.WardrobeItemContractDTO,
	input dto.RecommendOutfitReq,
) (*candidateFilterResult, error) {
	intent := uc.parseRecommendationIntent(input)
	// Personalization is loaded and merged here so every downstream stage
	// (query blend, ranking, prompt, HSL fallback) sees the same context.
	// The request-level StyleTarget is never overwritten.
	personalization := uc.loadPersonalizationContext(ctx, userID, intent)
	intent = mergePersonalizationIntoIntent(intent, personalization)
	retrievalQuery := uc.buildRecommendationRetrievalQuery(ctx, userID, intent)
	// Category là sở thích mềm (soft preference), KHÔNG phải bộ lọc cứng: giới hạn
	// pool vào một category sẽ khiến không thể ghép bộ hoàn chỉnh (áo + quần hoặc đầm).
	// Giữ seasonality, bỏ category hard filter.
	retrievalQuery.HardFilters.CategorySlugs = nil
	queryVector := uc.buildRecommendationQueryVector(
		ctx,
		retrievalQuery.SemanticQuery,
		personalization.StyleVector,
		personalization.StyleWeight,
	)

	minimumPool := uc.cfg.RAG.RecommendationMinimumCandidatePool
	maxBrandCandidates := (minimumPool * 30) / 100

	brandCandidatesCount := 0
	rankingCandidates := make([]types.CandidateForRanking, 0, minimumPool)

	// brandFashionMap giữ metadata fashion của brand items để hợp nhất vào fashionItemMap
	// chung (bao phủ toàn bộ active items của user + brand).
	brandFashionMap := map[uuid.UUID]*dto.FashionItemContractDTO{}

	// Fetch Brand items if requested
	if input.IncludeBrandItems != nil && *input.IncludeBrandItems {
		brandItemDTOS, _, err := uc.brandContract.ListEligibleBrandItemsForStyling(ctx, userID, &brand_dto.ListEligibleBrandItemsReq{})
		if err == nil {
			brandFashionIDs := make([]uuid.UUID, 0, len(brandItemDTOS))
			for _, brandItem := range brandItemDTOS {
				if brandItem.FashionItemID != uuid.Nil {
					brandFashionIDs = append(brandFashionIDs, brandItem.FashionItemID)
				}
			}
			brandFashionIDs = sliceutils.UniqueUUIDs(brandFashionIDs)

			if len(brandFashionIDs) > 0 {
				fashionItems, err := uc.fashionItemRepo.GetByIDs(ctx, brandFashionIDs)
				if err == nil {
					for _, fi := range fashionItems {
						brandFashionMap[fi.ID] = fashionItemEntityToContractDTO(fi)
					}
				}
			}

			type brandCandidate struct {
				brandItem   *brand_dto.BrandItemContractDTO
				fashionItem *dto.FashionItemContractDTO
				category    string
			}
			candidatesByCategory := map[string][]brandCandidate{}
			for _, brandItem := range brandItemDTOS {
				fashionItem := brandFashionMap[brandItem.FashionItemID]
				if fashionItem == nil || fashionItem.Category == nil {
					continue
				}
				slug := fashionItem.Category.Slug
				candidatesByCategory[slug] = append(candidatesByCategory[slug], brandCandidate{
					brandItem: brandItem, fashionItem: fashionItem, category: slug,
				})
			}

			// Ưu tiên lấy "ao" (top) trước, sau đó round-robin các category khác
			seenCategory := map[string]bool{}
			addBrandCandidate := func(slug string) bool {
				items := candidatesByCategory[slug]
				if len(items) == 0 || seenCategory[slug] {
					return false
				}
				seenCategory[slug] = true
				cand := items[0]
				candidatesByCategory[slug] = items[1:]

				mockWardrobeItem := &wardrobeDto.WardrobeItemContractDTO{
					ID:            cand.brandItem.FashionItemID,
					FashionItemID: &cand.brandItem.FashionItemID,
				}
				rankingCandidates = append(rankingCandidates, types.CandidateForRanking{
					Item:            mockWardrobeItem,
					Source:          types.CandidateSourceRetrieval,
					VectorScore:     0.95,
					LexicalScore:    0.95,
					RetrievalScore:  0.95,
					RetrievalRank:   brandCandidatesCount + 1,
					RetrievalSource: string(types.CandidateSourceRetrieval),
					ItemContext:     outfititemcontext.BrandItem,
					BrandItem:       cand.brandItem,
				})
				brandCandidatesCount++
				return true
			}

			// Vòng 1: ưu tiên "ao" (top)
			if addBrandCandidate("ao") && brandCandidatesCount >= maxBrandCandidates {
				goto afterBrandCandidates
			}
			// Vòng 2: 1 item từ mỗi category còn lại (ưu tiên ao-khoac, giay, quan)
			for _, slug := range []string{"ao-khoac", "giay", "quan", "chan-vay", "dam", "phu-kien", "mu"} {
				if addBrandCandidate(slug) && brandCandidatesCount >= maxBrandCandidates {
					goto afterBrandCandidates
				}
			}
			// Vòng 3: round-robin lấy thêm từ các category còn item
			for brandCandidatesCount < maxBrandCandidates {
				added := false
				for _, items := range candidatesByCategory {
					if len(items) > 0 {
						cand := items[0]
						candidatesByCategory[cand.category] = items[1:]
						// reuse addBrandCandidate but skip seen
						if seenCategory[cand.category] {
							// Already seen, take additional item from this category
							mockWardrobeItem := &wardrobeDto.WardrobeItemContractDTO{
								ID:            cand.brandItem.FashionItemID,
								FashionItemID: &cand.brandItem.FashionItemID,
							}
							rankingCandidates = append(rankingCandidates, types.CandidateForRanking{
								Item:            mockWardrobeItem,
								Source:          types.CandidateSourceRetrieval,
								VectorScore:     0.95,
								LexicalScore:    0.95,
								RetrievalScore:  0.95,
								RetrievalRank:   brandCandidatesCount + 1,
								RetrievalSource: string(types.CandidateSourceRetrieval),
								ItemContext:     outfititemcontext.BrandItem,
								BrandItem:       cand.brandItem,
							})
							brandCandidatesCount++
							added = true
						}
					}
					if brandCandidatesCount >= maxBrandCandidates {
						break
					}
				}
				if !added {
					break
				}
			}
		afterBrandCandidates:
		}
	}

	// Fetch Wardrobe items
	wardrobeLimit := minimumPool - brandCandidatesCount

	// Rút fashion_item_ids từ chính activeItems (đã lọc + cắt theo giới hạn gói ở validateAndGetActiveItems)
	// để đảm bảo tập query vector search khớp chính xác với tập wardrobe items khả dụng — không lệch.
	userFashionItemIDs := make([]uuid.UUID, 0, len(activeItems))
	for _, item := range activeItems {
		if item.FashionItemID != nil {
			userFashionItemIDs = append(userFashionItemIDs, *item.FashionItemID)
		}
	}
	userFashionItemIDs = sliceutils.UniqueUUIDs(userFashionItemIDs)
	filteredItems := make([]*wardrobeDto.WardrobeItemContractDTO, 0, len(activeItems))
	for _, item := range activeItems {
		if item.FashionItemID != nil {
			filteredItems = append(filteredItems, item)
		}
	}
	activeByFashionID := sliceutils.IndexBy(filteredItems, func(item *wardrobeDto.WardrobeItemContractDTO) uuid.UUID { return *item.FashionItemID })

	candidates, err := uc.fashionItemRepo.GetHybridCandidates(
		ctx,
		userFashionItemIDs,
		queryVector,
		retrieval.ExtractTermStrings(retrievalQuery.LexicalTerms),
		retrieval.ExtractTermStrings(retrievalQuery.ExcludedTerms),
		retrievalQuery.HardFilters,
		wardrobeLimit,
		uc.cfg.RAG.RrfKParameter,
	)
	if err == nil {
		for _, candidate := range candidates {
			item := activeByFashionID[candidate.ID]
			if item == nil {
				continue
			}
			rankingCandidates = append(rankingCandidates, types.CandidateForRanking{
				Item:            item,
				Source:          types.CandidateSource(candidate.RetrievalSource),
				VectorScore:     candidate.VectorScore,
				LexicalScore:    candidate.LexicalScore,
				RetrievalScore:  candidate.RetrievalScore,
				RetrievalRank:   candidate.RetrievalRank,
				RetrievalSource: candidate.RetrievalSource,
				ItemContext:     outfititemcontext.UserWardrobe,
			})
		}
	}

	// Dựng metadata fashion cho TOÀN BỘ active items của user (không chỉ các candidate
	// mà hybrid search trả về), rồi hợp nhất brand items. Nhờ đó bước fallback và
	// outfit-structure guarantee luôn suy ra được wearing role và áp được filter.
	fashionItemMap := map[uuid.UUID]*dto.FashionItemContractDTO{}
	if len(userFashionItemIDs) > 0 {
		fashionItems, err := uc.fashionItemRepo.GetByIDs(ctx, userFashionItemIDs)
		if err == nil {
			for _, fi := range fashionItems {
				fashionItemMap[fi.ID] = fashionItemEntityToContractDTO(fi)
			}
		}
	}
	for id, fi := range brandFashionMap {
		fashionItemMap[id] = fi
	}

	rankingCandidates, _ = ranking.EnsureMinimumCandidatePool(
		rankingCandidates,
		activeItems,
		retrievalQuery,
		minimumPool,
		fashionItemMap,
	)

	// Ensure all candidates have a context set (fallback candidates will be USER_WARDROBE)
	for i := range rankingCandidates {
		if rankingCandidates[i].ItemContext == "" {
			rankingCandidates[i].ItemContext = outfititemcontext.UserWardrobe
		}
	}

	ranked, _ := ranking.RankCandidates(
		rankingCandidates,
		intent,
		retrievalQuery,
		uc.cfg.RAG.RecentlyWornPenaltyDays,
		uc.cfg.RAG.LongUnwornBonusDays,
		minimumPool,
		fashionItemMap,
	)

	if !ranking.HasValidOutfitStructurePrompts(ranked, fashionItemMap) {
		return nil, fashioerrors.ErrInsufficientOutfitComponents()
	}

	return &candidateFilterResult{
		Candidates:      ranked,
		FashionItemMap:  fashionItemMap,
		Personalization: personalization,
		Pool:            rankingCandidates,
		RetrievalQuery:  retrievalQuery,
		QueryVector:     queryVector,
	}, nil
}

// generateOutfitRecommendation gọi dịch vụ AI/LLM để tổng hợp và sinh ra gợi ý trang phục từ danh sách ứng viên đã chọn lọc.
//
// Đầu vào mẫu:
//
//	candidates: []types.CandidateForPrompt{...}
//	input: dto.RecommendOutfitReq{Occasion: pointer to "party"}
//
// Đầu ra mẫu:
//
//	(*dto.RecommendedOutfitRes, nil) chứa thông tin phối đồ và lý do chi tiết từ AI.
func (uc *OutfitRecommendationUseCase) generateOutfitRecommendation(
	ctx context.Context,
	userID uuid.UUID,
	candidates []types.CandidateForPrompt,
	input dto.RecommendOutfitReq,
	fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO,
	personalization types.PersonalizationContext,
) (*dto.RecommendedOutfitRes, error) {
	return synthesis.GenerateOutfitRecommendation(ctx, uc.aiService, userID, candidates, input, uc.cfg, fashionItemMap, personalization)
}

// updateQuotaAndConstructResponse thực hiện trừ đi 1 lượt sử dụng trong quota hàng ngày của người dùng,
// sau đó tính toán và cập nhật số lượt còn lại vào kết quả trả về Front-End.
//
// Hành vi:
// Hàm sẽ thực hiện gọi [UpdateOutfitQuota] để tăng số lần đã sử dụng của người dùng lên 1. Sau đó, nó thử lấy lại
// quota mới nhất thông qua [GetAndResetDailyQuota] để tính toán hạn ngạch còn lại chính xác. Nếu việc lấy quota mới
// thất bại, hàm sẽ tự động dùng thông tin quota cũ và trừ đi 1 làm phương án dự phòng.
//
// Đầu vào mẫu:
//
//	finalRes: &dto.RecommendedOutfitRes{}
//	quotaDTO: &sub_dto.UserSubscriptionDTO{AiOutfitDailyQuota: 10, OutfitRecommendCount: 2}
//
// Đầu ra mẫu:
//
//	Cập nhật trường [RemainingQuota] của finalRes thành 7 (hoặc 8 nếu trừ trực tiếp từ quotaDTO cũ).
func (uc *OutfitRecommendationUseCase) updateQuotaAndConstructResponse(
	ctx context.Context,
	userID uuid.UUID,
	finalRes *dto.RecommendedOutfitRes,
	quotaDTO *sub_dto.UserSubscriptionDTO,
) error {
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

// fashionItemEntityToContractDTO chuyển entity FashionItem (đã preload Category) sang
// contract DTO dùng trong pipeline gợi ý phối đồ.
func fashionItemEntityToContractDTO(fi *entities.FashionItem) *dto.FashionItemContractDTO {
	if fi == nil {
		return nil
	}
	var category *dto.CategoryBriefRes
	if fi.Category != nil {
		category = &dto.CategoryBriefRes{
			ID:   fi.Category.ID,
			Name: fi.Category.Name,
			Slug: fi.Category.Slug,
		}
	}
	return &dto.FashionItemContractDTO{
		ID:              fi.ID,
		Category:        category,
		ImageUrl:        fi.ImageUrl,
		ImagePublicID:   fi.ImagePublicID,
		Color:           fi.Color,
		ColorHex:        fi.ColorHex,
		ColorHue:        fi.ColorHue,
		ColorSaturation: fi.ColorSaturation,
		ColorLightness:  fi.ColorLightness,
		Style:           fi.Style,
		Material:        fi.Material,
		Pattern:         fi.Pattern,
		Fit:             fi.Fit,
		Seasonality:     fi.Seasonality,
		Description:     fi.Description,
		CreatedAt:       fi.CreatedAt,
		UpdatedAt:       fi.UpdatedAt,
	}
}
