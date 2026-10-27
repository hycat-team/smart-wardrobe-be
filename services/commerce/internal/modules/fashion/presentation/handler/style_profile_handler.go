package handler

import (
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	usecase_interfaces "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"
	"smart-wardrobe-be/shared/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/shared/presentation"

	"github.com/gin-gonic/gin"
)

const (
	msgStyleProfileGetSuccess    = "Lấy hồ sơ phong cách thành công"
	msgStyleProfileUpdateSuccess = "Cập nhật hồ sơ phong cách thành công"
	msgStyleProfileDeleteSuccess = "Xoá hồ sơ phong cách thành công"
	// msgStyleSuggestionEmpty accompanies the empty payload of a user whose
	// wardrobe has no styled item yet (contracts §1.4: the wording belongs to
	// the envelope, not to `data`).
	msgStyleSuggestionEmpty   = "Chưa đủ dữ liệu để gợi ý phong cách. Hãy thêm món đồ có phong cách vào tủ đồ."
	msgStyleSuggestionSuccess = "Lấy gợi ý phong cách thành công"
)

// StyleProfileHandler exposes the style profile of the authenticated user.
type StyleProfileHandler struct {
	styleProfileUseCase usecase_interfaces.IStyleProfileUseCase
}

func NewStyleProfileHandler(uc usecase_interfaces.IStyleProfileUseCase) *StyleProfileHandler {
	return &StyleProfileHandler{
		styleProfileUseCase: uc,
	}
}

// GetStyleProfile returns the current style profile
// @Summary Lấy hồ sơ phong cách cá nhân
// @Description Trả hồ sơ gu hiện tại của người dùng. Nếu chưa từng lưu, trả giá trị mặc định và không tạo bản ghi.
// @Description Người dùng đã hết quyền lợi gói vẫn xem được hồ sơ, với personalizationAvailable = false.
// @Tags Style Profile
// @Accept json
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=dto.StyleProfileRes} "Lấy hồ sơ phong cách thành công"
// @Router /api/v1/me/style-profile [get]
func (h *StyleProfileHandler) GetStyleProfile(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	response, err := h.styleProfileUseCase.GetProfile(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgStyleProfileGetSuccess, response)
	return nil
}

// UpdateStyleProfile updates the style profile
// @Summary Cập nhật hồ sơ phong cách cá nhân
// @Description Cập nhật một phần hồ sơ gu: trường bỏ trống giữ nguyên giá trị, mảng rỗng xoá danh sách.
// @Description Mỗi danh sách được trim, loại trùng không phân biệt hoa thường, mỗi giá trị tối đa 50 ký tự và tối đa maxStyles giá trị.
// @Description Biểu diễn ngữ nghĩa được tạo lại khi lưu; nếu nhà cung cấp AI lỗi, hồ sơ vẫn được lưu với hasEmbedding = false.
// @Tags Style Profile
// @Accept json
// @Produce json
// @Param body body dto.UpdateStyleProfileReq true "Thông tin hồ sơ phong cách cần cập nhật"
// @Success 200 {object} shared_pres.APIResponse{data=dto.StyleProfileRes} "Cập nhật hồ sơ phong cách thành công"
// @Router /api/v1/me/style-profile [put]
func (h *StyleProfileHandler) UpdateStyleProfile(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input dto.UpdateStyleProfileReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.styleProfileUseCase.UpdateProfile(c.Request.Context(), userID, input)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgStyleProfileUpdateSuccess, response)
	return nil
}

// DeleteStyleProfile deletes the style profile
// @Summary Xoá hồ sơ phong cách cá nhân
// @Description Xoá hồ sơ gu của người dùng hiện tại. Thao tác idempotent: xoá khi chưa có hồ sơ vẫn thành công.
// @Tags Style Profile
// @Accept json
// @Produce json
// @Success 200 {object} shared_pres.APIResponse "Xoá hồ sơ phong cách thành công"
// @Router /api/v1/me/style-profile [delete]
func (h *StyleProfileHandler) DeleteStyleProfile(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	if err := h.styleProfileUseCase.DeleteProfile(c.Request.Context(), userID); err != nil {
		return err
	}

	shared_pres.Success(c, msgStyleProfileDeleteSuccess, nil)
	return nil
}

// GetStyleSuggestions returns cold-start style suggestions derived from the wardrobe
// @Summary Gợi ý phong cách từ tủ đồ hiện có
// @Description Gợi ý gu từ những món đồ đang hoạt động trong tủ (cold start). Không tự áp dụng: người dùng xác nhận bằng PUT /me/style-profile.
// @Description Chưa đủ dữ liệu: data là {"styles": [], "sampleSize": 0} và thông báo nằm ở message của envelope.
// @Description Endpoint read-only: công tắc vận hành personalization.enabled không chặn, nhưng gói không có quyền lợi thì trả 403.
// @Tags Style Profile
// @Accept json
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=dto.StyleSuggestionRes} "Lấy gợi ý phong cách thành công"
// @Router /api/v1/me/style-profile/suggestions [get]
func (h *StyleProfileHandler) GetStyleSuggestions(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	response, err := h.styleProfileUseCase.SuggestStyles(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	message := msgStyleSuggestionSuccess
	if response.SampleSize == 0 {
		message = msgStyleSuggestionEmpty
	}
	shared_pres.Success(c, message, response)
	return nil
}
