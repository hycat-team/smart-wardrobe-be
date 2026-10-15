package handler

import (
	"smart-wardrobe-be/services/account/dto/identity"
	"smart-wardrobe-be/shared/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/shared/presentation"

	"github.com/gin-gonic/gin"
)

const successListActivityEvents = "Lấy danh sách hoạt động người dùng thành công"

// ListActivityEvents lấy nhật ký hoạt động người dùng cho Admin
// @Summary Danh sách hoạt động người dùng
// @Description Lấy danh sách phân trang các sự kiện hoạt động (đăng nhập, làm mới phiên) kèm danh tính người dùng. Hỗ trợ lọc theo người dùng, loại hoạt động và khoảng ngày (ngày lịch Việt Nam). Chỉ đọc.
// @Tags Admin User
// @Produce json
// @Param userId query string false "ID người dùng (UUID)"
// @Param activityType query string false "Loại hoạt động: login | refresh"
// @Param fromDate query string false "Từ ngày, bao gồm (YYYY-MM-DD, giờ Việt Nam)"
// @Param toDate query string false "Đến ngày, bao gồm trọn ngày (YYYY-MM-DD, giờ Việt Nam)"
// @Param page query int false "Trang (mặc định 1)"
// @Param limit query int false "Số bản ghi mỗi trang (mặc định 20, tối đa 100)"
// @Success 200 {object} shared_pres.APIResponse{data=dto.ActivityEventListRes} "Lấy danh sách hoạt động thành công"
// @Failure 400 {object} shared_pres.APIResponse
// @Failure 401 {object} shared_pres.APIResponse
// @Failure 403 {object} shared_pres.APIResponse
// @Router /api/v1/admin/activity-events [get]
func (h *AdminHandler) ListActivityEvents(c *gin.Context) error {
	var query dto.GetActivityEventsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}

	response, err := h.userUseCase.ListActivityEvents(c.Request.Context(), query)
	if err != nil {
		return err
	}

	shared_pres.Success(c, successListActivityEvents, response)
	return nil
}
