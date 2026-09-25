package handler

import (
	"smart-wardrobe-be/services/commerce/internal/modules/community/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"
	"smart-wardrobe-be/shared/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/shared/presentation"

	"github.com/gin-gonic/gin"
)

const (
	msgToggleFollowSuccess = "Cập nhật theo dõi thành công"
	msgGetFollowsSuccess   = "Lấy danh sách theo dõi thành công"
)

type FollowHandler struct {
	followUC uc_interfaces.IFollowUseCase
}

func NewFollowHandler(followUC uc_interfaces.IFollowUseCase) *FollowHandler {
	return &FollowHandler{followUC: followUC}
}

// ToggleFollow follows or unfollows a user.
// @Summary Theo dõi / Bỏ theo dõi người dùng
// @Description Theo dõi hoặc bỏ theo dõi một người dùng (idempotent, không cho tự theo dõi)
// @Tags Community
// @Accept json
// @Produce json
// @Param username path string true "Tên người dùng"
// @Param body body dto.FollowReq true "Trạng thái theo dõi"
// @Success 200 {object} shared_pres.APIResponse "Cập nhật theo dõi thành công"
// @Router /api/v1/users/{username}/follow [put]
func (h *FollowHandler) ToggleFollow(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	var input dto.FollowReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	if err := h.followUC.ToggleFollow(c.Request.Context(), userID, c.Param("username"), *input.IsFollowing); err != nil {
		return err
	}
	shared_pres.Success(c, msgToggleFollowSuccess, nil)
	return nil
}

// ListFollows lists a user's following/followers.
// @Summary Lấy danh sách theo dõi của người dùng
// @Description Lấy danh sách following/followers (type tùy chọn, bỏ trống trả cả hai kèm relation)
// @Tags Community
// @Produce json
// @Param username path string true "Tên người dùng"
// @Param query query dto.GetFollowsQueryReq false "Bộ lọc danh sách"
// @Success 200 {object} shared_pres.APIResponse{data=dto.FollowUserListRes} "Lấy danh sách theo dõi thành công"
// @Router /api/v1/users/{username}/follows [get]
func (h *FollowHandler) ListFollows(c *gin.Context) error {
	var query dto.GetFollowsQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}
	res, err := h.followUC.ListFollows(c.Request.Context(), c.Param("username"), optionalViewer(c), query)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgGetFollowsSuccess, res)
	return nil
}
