package recommendation

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/ranking"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/usecase/ai/recommendation/synthesis"
)

// RecommendOutfit thực hiện quy trình gợi ý trang phục dựa trên các món đồ hiện có trong tủ đồ của người dùng và các tùy chọn yêu cầu.
//
// Luồng xử lý chi tiết:
//   - 1. Kiểm tra hạn ngạch (quota) hàng ngày và lấy danh sách các món đồ đang hoạt động (active) của người dùng qua [validateAndGetActiveItems].
//   - 2. Tìm kiếm và xếp hạng các ứng viên phù hợp dựa trên phân tích ý định (NLP intent parser) và tìm kiếm lai (hybrid search) qua [filterCandidates].
//   - 3. Sử dụng dịch vụ AI/LLM để tạo gợi ý trang phục qua [generateOutfitRecommendation].
//   - 4. Nếu AI gặp lỗi hoặc timeout, hệ thống sẽ kích hoạt luồng fallback cục bộ sử dụng thuật toán phối màu HSL qua [RunLocalHSLMatching].
//   - 5. Cập nhật lượt sử dụng hạn ngạch và bổ sung thông tin phản hồi qua [updateQuotaAndConstructResponse].
//
// Đầu vào mẫu:
//
//	userID: "8e05c317-062e-4b47-ba21-12f5a04f21db"
//	input: dto.RecommendOutfitReq{
//	    Occasion: pointer to "work",
//	    StyleTarget: pointer to "minimalist",
//	    Details: pointer to "mặc đi làm ngày mưa lạnh"
//	}
//
// Đầu ra mẫu:
//
//	(*dto.RecommendedOutfitRes, nil) nơi RecommendedOutfitRes chứa danh sách các món đồ được phối kèm mô tả và lý do lựa chọn từ AI hoặc thuật toán HSL fallback.
func (uc *OutfitRecommendationUseCase) RecommendOutfit(
	ctx context.Context,
	userID uuid.UUID,
	input dto.RecommendOutfitReq,
) (*dto.RecommendedOutfitRes, error) {
	// The id is generated once per request so the response can be fed back to
	// the feedback endpoint (033 contracts §2.1, US6).
	recommendationID := uuid.New()

	activeItems, quotaDTO, maxWardrobeItems, err := uc.validateAndGetActiveItems(ctx, userID)
	if err != nil {
		return nil, err
	}

	filterResult, err := uc.filterCandidates(ctx, userID, activeItems, maxWardrobeItems, input)
	if err != nil {
		return nil, err
	}

	finalRes, err := uc.generateOutfitRecommendation(
		ctx,
		userID,
		filterResult.Candidates,
		input,
		filterResult.FashionItemMap,
		filterResult.Personalization,
	)
	if err != nil {
		failureKind, providerHint, promptLen, responsePreview := synthesis.ClassifyFallbackTrace(err)
		uc.logger.Warn("Outfit recommendation AI fallback triggered",
			zap.String("failure_kind", failureKind),
			zap.String("provider_hint", providerHint),
			zap.Error(err),
			zap.String("user_id", userID.String()),
			zap.String("recommendation_id", recommendationID.String()),
			zap.Bool("personalized", filterResult.Personalization.Active),
			zap.Int("candidate_count", len(filterResult.Candidates)),
			zap.Int("prompt_len", promptLen),
			zap.String("response_preview", responsePreview),
		)
		finalRes = ranking.RunLocalHSLMatchingWithPersonalization(
			filterResult.Candidates,
			input,
			filterResult.FashionItemMap,
			filterResult.Personalization,
		)
	}

	if finalRes != nil {
		finalRes.RecommendationID = recommendationID
		// 033 US2/T041: catalog gap-fill suggestions are additive. They are
		// attached after the outfit is final, never merge into Items and never
		// touch the quota below.
		finalRes.Suggestions = uc.buildCatalogSuggestions(ctx, userID, finalRes, filterResult)
		// 033 US3/T056: rotation observability + best-effort last_recommended_at.
		// Both are fail-soft by contract: they log and continue, never blocking
		// the response (FR-054, FR-068).
		uc.recordRotationObservability(
			ctx,
			userID,
			recommendationID,
			finalRes,
			filterResult,
			uc.rotationSettings(),
			time.Now(),
		)
	}

	if err := uc.updateQuotaAndConstructResponse(ctx, userID, finalRes, quotaDTO); err != nil {
		return nil, err
	}

	return finalRes, nil
}
