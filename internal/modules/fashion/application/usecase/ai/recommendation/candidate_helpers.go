package recommendation

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/shared"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobe/outfititemcontext"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobe/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/ranking"
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/retrieval"
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/types"
	"smart-wardrobe-be/pkg/utils/sliceutils"
)

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
//	([]*entities.WardrobeItem, *contract.UserSubscriptionDTO, nil)
func (uc *OutfitRecommendationUseCase) validateAndGetActiveItems(
	ctx context.Context,
	userID uuid.UUID,
) ([]*entities.WardrobeItem, *contract.UserSubscriptionDTO, error) {
	quotaDTO, err := uc.userQuotaCtr.GetAndResetDailyQuota(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	if quotaDTO.OutfitRecommendCount >= quotaDTO.AiOutfitDailyQuota {
		return nil, nil, subscriptionerrors.ErrAiOutfitQuotaExceeded()
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
		return nil, nil, wardrobeerrors.ErrMinimumWardrobeItemsRequired()
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
//	activeItems: []*entities.WardrobeItem{...}
//	input: dto.RecommendOutfitReq{Occasion: pointer to "casual"}
//
// Đầu ra mẫu:
//
//	([]types.CandidateForPrompt, nil)
func (uc *OutfitRecommendationUseCase) filterCandidates(
	ctx context.Context,
	userID uuid.UUID,
	activeItems []*entities.WardrobeItem,
	input dto.RecommendOutfitReq,
) ([]types.CandidateForPrompt, error) {
	intent := uc.parseRecommendationIntent(input)
	retrievalQuery := uc.buildRecommendationRetrievalQuery(ctx, userID, intent)
	queryVector := uc.buildRecommendationQueryVector(ctx, retrievalQuery.SemanticQuery)

	minimumPool := uc.cfg.RAG.RecommendationMinimumCandidatePool
	maxBrandCandidates := (minimumPool * 30) / 100

	brandCandidatesCount := 0
	rankingCandidates := make([]types.CandidateForRanking, 0, minimumPool)

	// Fetch Brand items if requested
	if input.IncludeBrandItems != nil && *input.IncludeBrandItems {
		brandItemsRaw, err := uc.brandContract.ListEligibleBrandItemsForStyling(ctx, userID, nil)
		if err == nil {
			if brandItems, ok := brandItemsRaw.([]*entities.BrandItem); ok {
				for _, brandItem := range brandItems {
					if brandCandidatesCount >= maxBrandCandidates {
						break
					}
					if brandItem.FashionItem == nil {
						continue
					}
					mockWardrobeItem := &entities.WardrobeItem{
						AuditableEntity: entities.AuditableEntity{
							BaseEntity: entities.BaseEntity{
								ID: brandItem.FashionItemID,
							},
						},
						FashionItemID: brandItem.FashionItemID,
						FashionItem:   brandItem.FashionItem,
					}
					rankingCandidates = append(rankingCandidates, types.CandidateForRanking{
						Item:            mockWardrobeItem,
						Source:          types.CandidateSourceRetrieval,
						VectorScore:     0.95,
						LexicalScore:    0.95,
						RetrievalScore:  0.95,
						RetrievalRank:   brandCandidatesCount + 1,
						RetrievalSource: types.CandidateSourceRetrieval,
						ItemContext:     outfititemcontext.BrandItem,
						BrandItem:       brandItem,
					})
					brandCandidatesCount++
				}
			}
		}
	}

	// Fetch Wardrobe items
	wardrobeLimit := minimumPool - brandCandidatesCount
	candidates, err := uc.wardrobeRepo.GetHybridCandidates(
		ctx,
		userID,
		queryVector,
		retrieval.ExtractTermStrings(retrievalQuery.LexicalTerms),
		retrieval.ExtractTermStrings(retrievalQuery.ExcludedTerms),
		retrievalQuery.HardFilters,
		wardrobeLimit,
		uc.cfg.RAG.RrfKParameter,
	)
	if err == nil {
		for _, candidate := range candidates {
			if candidate.Item == nil {
				continue
			}
			rankingCandidates = append(rankingCandidates, types.CandidateForRanking{
				Item:            candidate.Item,
				Source:          candidate.RetrievalSource,
				VectorScore:     candidate.VectorScore,
				LexicalScore:    candidate.LexicalScore,
				RetrievalScore:  candidate.RetrievalScore,
				RetrievalRank:   candidate.RetrievalRank,
				RetrievalSource: candidate.RetrievalSource,
				ItemContext:     outfititemcontext.UserWardrobe,
			})
		}
	}

	rankingCandidates, _ = ranking.EnsureMinimumCandidatePool(
		rankingCandidates,
		activeItems,
		retrievalQuery,
		minimumPool,
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
	)

	return ranked, nil
}

// parseRecommendationIntent phân tích ý định từ văn bản tự do của người dùng và hợp nhất với các
// tùy chọn rõ ràng (như dịp, phong cách, thời tiết, mùa) để tạo ra đối tượng [ParsedIntent].
//
// Hành vi:
//  1. Sử dụng bộ phân tích [nlpParser.Parse] trên trường [Details] để trích xuất ý định ngầm định.
//  2. Nếu người dùng chọn trực tiếp các tùy chọn trên UI (như [Occasion], [StyleTarget], [ColorTone], [Weather], [Season]),
//     ghi đè hoặc bổ sung các thuộc tính này vào cấu trúc ý định thu được.
//  3. Loại bỏ các từ khóa bị xung đột qua [RemoveConflictingLexicalTerms].
//  4. Xây dựng chuỗi truy vấn ngữ nghĩa bằng [BuildRecommendationSemanticQuery] làm dữ liệu đầu vào cho mô hình embedding.
//
// Đầu vào mẫu:
//
//	input: dto.RecommendOutfitReq{
//	    Details: pointer to "mặc đi chơi phố năng động",
//	    Occasion: pointer to "casual"
//	}
//
// Đầu ra mẫu:
//
//	dto.ParsedIntent{
//	    Occasion: []string{"casual"},
//	    StyleTarget: []string{"streetwear"},
//	    SemanticQuery: "occasion: casual | style: streetwear | details: mặc đi chơi phố năng động",
//	    ...
//	}
func (uc *OutfitRecommendationUseCase) parseRecommendationIntent(
	input dto.RecommendOutfitReq,
) dto.ParsedIntent {
	freeText := ""
	if input.Details != nil {
		freeText = *input.Details
	}

	intent := uc.nlpParser.Parse(freeText)
	hasExplicitOptions := false
	if input.Occasion != nil && *input.Occasion != "" {
		intent.Occasion = []string{strings.TrimSpace(*input.Occasion)}
		hasExplicitOptions = true
	}
	if input.StyleTarget != nil && *input.StyleTarget != "" {
		intent.StyleTarget = []string{strings.TrimSpace(*input.StyleTarget)}
		hasExplicitOptions = true
	}
	if input.ColorTone != nil && *input.ColorTone != "" {
		intent.ColorTone = []string{strings.TrimSpace(*input.ColorTone)}
		hasExplicitOptions = true
	}
	if input.Weather != nil && *input.Weather != "" {
		weather := strings.TrimSpace(*input.Weather)
		intent.PositiveConstraints = sliceutils.AppendUniqueStringCaseInsensitive(intent.PositiveConstraints, weather)
		hasExplicitOptions = true
	}
	if input.Season != nil && *input.Season != "" && *input.Season != dto.SeasonAll {
		intent.PositiveConstraints = sliceutils.AppendUniqueStringCaseInsensitive(intent.PositiveConstraints, string(*input.Season))
		hasExplicitOptions = true
	}
	intent.LexicalTerms = uc.nlpParser.RemoveConflictingLexicalTerms(intent)

	intent.SemanticQuery = retrieval.BuildRecommendationSemanticQuery(intent, freeText, hasExplicitOptions)

	return intent
}

// buildRecommendationQueryVector sinh vector nhúng (embedding vector) từ chuỗi truy vấn ngữ nghĩa bằng dịch vụ AI.
//
// Hành vi:
// Hàm thiết lập thời hạn timeout từ cấu hình ([RecommendationEmbeddingTimeoutSeconds]) và gọi [GenerateEmbeddings]
// của dịch vụ AI. Nếu xảy ra lỗi hoặc số chiều vector trả về không khớp với cấu hình ([RecommendationEmbeddingDimension]),
// hàm sẽ ghi log cảnh báo và trả về [nil] để hệ thống tiếp tục chạy với tìm kiếm từ khóa (lexical search) thuần túy làm fallback.
//
// Đầu vào mẫu:
//
//	semanticQuery: "occasion: work | color tone: light"
//
// Đầu ra mẫu:
//
//	entities.Vector{0.123, -0.456, ..., 0.009} (độ dài khớp với cấu hình, ví dụ 768 chiều) hoặc nil nếu lỗi.
func (uc *OutfitRecommendationUseCase) buildRecommendationQueryVector(
	ctx context.Context,
	semanticQuery string,
) entities.Vector {
	if semanticQuery == "" {
		uc.logger.Warn("Outfit recommendation embedding skipped",
			zap.String("reason", "empty_semantic_query"),
		)
		return nil
	}

	embedCtx, cancel := context.WithTimeout(ctx, time.Duration(uc.cfg.RAG.RecommendationEmbeddingTimeoutSeconds)*time.Second)
	defer cancel()

	vecs, err := uc.aiService.GenerateEmbeddings(embedCtx, []string{semanticQuery})
	if err != nil {
		uc.logger.Warn("Outfit recommendation embedding failed",
			zap.String("reason", "provider_error"),
			zap.Error(err),
		)
		return nil
	}
	if len(vecs) == 0 {
		uc.logger.Warn("Outfit recommendation embedding failed",
			zap.String("reason", "empty_vector_response"),
		)
		return nil
	}
	if len(vecs[0]) != uc.cfg.RAG.RecommendationEmbeddingDimension {
		uc.logger.Warn("Outfit recommendation embedding failed",
			zap.String("reason", "unexpected_dimension"),
			zap.Int("dimension", len(vecs[0])),
			zap.Int("expected_dimension", uc.cfg.RAG.RecommendationEmbeddingDimension),
		)
		return nil
	}

	return entities.Vector(vecs[0])
}

// buildRecommendationRetrievalQuery chuyển đổi ý định đã phân tích thành truy vấn tìm kiếm nâng cao (RecommendationRetrievalQuery).
//
// Hành vi:
// Nếu tính năng LLM rewriter được bật ([RecommendationLLMRewriterEnabled]), hàm sẽ gửi ý định đến LLM rewriter thông qua
// [LLMRecommendationQueryRewriter.Rewrite] kèm theo timeout. Nếu LLM rewriter xử lý thành công, kết quả trả về sẽ là truy vấn đã được AI tối ưu.
// Nếu tính năng LLM bị tắt hoặc xảy ra lỗi trong quá trình gọi AI, hàm sẽ tự động fallback sang sử dụng [LocalRecommendationQueryRewriter.Rewrite]
// để xây dựng truy vấn dựa trên tập luật tĩnh cục bộ.
//
// Đầu vào mẫu:
//
//	intent: dto.ParsedIntent{Occasion: []string{"work"}}
//
// Đầu ra mẫu:
//
//	types.RecommendationRetrievalQuery{
//	    SemanticQuery: "occasion: work",
//	    LexicalTerms: []types.RetrievalTerm{{Value: "office", Source: "taxonomy"}},
//	    ...
//	}
func (uc *OutfitRecommendationUseCase) buildRecommendationRetrievalQuery(
	ctx context.Context,
	userID uuid.UUID,
	intent dto.ParsedIntent,
) types.RecommendationRetrievalQuery {
	local := retrieval.LocalRecommendationQueryRewriter{}
	if uc.cfg == nil || !uc.cfg.RAG.RecommendationLLMRewriterEnabled {
		query, _ := local.Rewrite(ctx, intent)
		return query
	}

	rewriteCtx, cancel := context.WithTimeout(ctx, time.Duration(uc.cfg.RAG.RecommendationLLMRewriterTimeoutSeconds)*time.Second)
	defer cancel()

	query, err := retrieval.NewLLMRecommendationQueryRewriter(uc.aiService, uc.cfg).WithUserID(userID).Rewrite(rewriteCtx, intent)
	if err == nil {
		if uc.logger != nil {
			uc.logger.Info("Outfit recommendation query rewriter completed",
				zap.String("rewriter_source", "llm"),
				zap.Int("lexical_terms_count", len(query.LexicalTerms)),
				zap.Int("excluded_terms_count", len(query.ExcludedTerms)),
			)
		}
		return query
	}

	if uc.logger != nil {
		uc.logger.Warn("Outfit recommendation query rewriter fallback",
			zap.String("rewriter_source", "local"),
			zap.String("fallback_reason", err.Error()),
		)
	}
	query, _ = local.Rewrite(ctx, intent)
	return query
}

// retrievalSourceCount đếm số lượng ứng viên trong danh sách được trả về từ một nguồn tìm kiếm cụ thể (vector hoặc lexical).
//
// Đầu vào mẫu:
//
//	candidates: []repositories.HybridCandidate{{RetrievalSource: "vector"}, {RetrievalSource: "lexical"}}
//	source: "vector"
//
// Đầu ra mẫu:
//
//	1
func retrievalSourceCount(candidates []repositories.HybridCandidate, source string) int {
	count := 0
	for _, candidate := range candidates {
		if candidate.RetrievalSource == source {
			count++
		}
	}
	return count
}

// recommendationLexicalQueryMode xác định chế độ tìm kiếm được áp dụng dựa trên sự hiện diện của vector truy vấn và từ khóa lexical.
//
// Đầu vào mẫu:
//
//	vector: entities.Vector{0.1, 0.2}
//	lexicalTerms: []string{"ao", "quan"}
//
// Đầu ra mẫu:
//
//	"hybrid" (nếu có cả hai), "vector" (chỉ có vector), "lexical" (chỉ có từ khóa), hoặc "fallback" (không có cả hai).
func recommendationLexicalQueryMode(vector entities.Vector, lexicalTerms []string) string {
	hasVector := len(vector) > 0
	hasLexical := len(lexicalTerms) > 0
	switch {
	case hasVector && hasLexical:
		return "hybrid"
	case hasVector:
		return "vector"
	case hasLexical:
		return "lexical"
	default:
		return "fallback"
	}
}

// missingEmbeddingCount thống kê số lượng món đồ trong danh sách chưa được tạo vector nhúng (embedding).
//
// Đầu vào mẫu:
//
//	items: []*entities.WardrobeItem{{Embedding: []float32{...}}, {Embedding: nil}}
//
// Đầu ra mẫu:
//
//	1
func missingEmbeddingCount(items []*entities.WardrobeItem) int {
	count := 0
	for _, item := range items {
		if item == nil || len(item.FashionEmbedding()) == 0 {
			count++
		}
	}
	return count
}
