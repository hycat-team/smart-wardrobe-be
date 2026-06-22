package recommendation

import (
	"context"

	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/synthesis"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/types"

	"github.com/google/uuid"
)

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
) (*dto.RecommendedOutfitRes, error) {
	return synthesis.GenerateOutfitRecommendation(ctx, uc.aiService, userID, candidates, input, uc.cfg)
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
//	quotaDTO: &contract.UserSubscriptionDTO{AiOutfitDailyQuota: 10, OutfitRecommendCount: 2}
//
// Đầu ra mẫu:
//
//	Cập nhật trường [RemainingQuota] của finalRes thành 7 (hoặc 8 nếu trừ trực tiếp từ quotaDTO cũ).
func (uc *OutfitRecommendationUseCase) updateQuotaAndConstructResponse(
	ctx context.Context,
	userID uuid.UUID,
	finalRes *dto.RecommendedOutfitRes,
	quotaDTO *contract.UserSubscriptionDTO,
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
