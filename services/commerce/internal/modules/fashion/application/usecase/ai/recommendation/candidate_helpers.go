package recommendation

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	season "smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/constants/season"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/retrieval"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/types"
	personalization_shared "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/personalization/shared"
	"smart-wardrobe-be/shared/pkg/utils/sliceutils"
)

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
	if input.Season != nil && *input.Season != "" && *input.Season != season.All {
		intent.PositiveConstraints = sliceutils.AppendUniqueStringCaseInsensitive(intent.PositiveConstraints, string(*input.Season))
		hasExplicitOptions = true
	}
	intent.LexicalTerms = uc.nlpParser.RemoveConflictingLexicalTerms(intent)

	intent.SemanticQuery = retrieval.BuildRecommendationSemanticQuery(intent, freeText, hasExplicitOptions)

	return intent
}

// buildRecommendationQueryVector sinh vector nhúng (embedding vector) từ chuỗi truy vấn ngữ nghĩa bằng dịch vụ AI
// và trộn mềm với vector gu cá nhân (033 D4, T030).
//
// Hành vi:
// Hàm thiết lập thời hạn timeout từ cấu hình ([RecommendationEmbeddingTimeoutSeconds]) và gọi [GenerateEmbeddings]
// của dịch vụ AI. Nếu xảy ra lỗi hoặc số chiều vector trả về không khớp với cấu hình ([RecommendationEmbeddingDimension]),
// hàm sẽ ghi log cảnh báo và bỏ phần vector truy vấn.
//
// Khi có style vector: kết quả là normalize((1-w)*normalize(query) + w*normalize(style)) với w là trọng số gu
// hiệu lực. Query rỗng thì dùng style vector; style vector rỗng thì giữ nguyên query (text-only).
//
// Đầu vào mẫu:
//
//	semanticQuery: "occasion: work | color tone: light"
//	styleVector:   []float32{0.123, -0.456, ...} (có thể nil)
//	styleWeight:   0.30
//
// Đầu ra mẫu:
//
//	entities.Vector{0.123, -0.456, ..., 0.009} (độ dài khớp với cấu hình, ví dụ 768 chiều) hoặc nil nếu lỗi.
func (uc *OutfitRecommendationUseCase) buildRecommendationQueryVector(
	ctx context.Context,
	semanticQuery string,
	styleVector []float32,
	styleWeight float64,
) entities.Vector {
	var queryVector []float32
	if semanticQuery == "" {
		uc.logger.Warn("Outfit recommendation embedding skipped",
			zap.String("reason", "empty_semantic_query"),
		)
	} else {
		queryVector = uc.embedRecommendationQuery(ctx, semanticQuery)
	}

	if len(styleVector) == 0 {
		return entities.Vector(queryVector)
	}

	blended := personalization_shared.BlendVectors(queryVector, styleVector, styleWeight)
	return entities.Vector(sliceutils.NormalizeVector(blended))
}

// embedRecommendationQuery calls the embedding provider for the semantic query
// and returns nil (fail-soft) on any error, dimension mismatch or empty answer.
func (uc *OutfitRecommendationUseCase) embedRecommendationQuery(
	ctx context.Context,
	semanticQuery string,
) []float32 {
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

	return vecs[0]
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

	rewriteCtx, cancel := context.WithTimeout(ctx,
		time.Duration(uc.cfg.RAG.RecommendationLLMRewriterTimeoutSeconds)*time.Second,
	)
	defer cancel()

	query, err := retrieval.
		NewLLMRecommendationQueryRewriter(uc.aiService, uc.cfg).
		WithUserID(userID).
		Rewrite(rewriteCtx, intent)
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
