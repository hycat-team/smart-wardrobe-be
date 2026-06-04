package handler

import (
	"smart-wardrobe-be/internal/modules/community/application/dto"
	usecase_interfaces "smart-wardrobe-be/internal/modules/community/application/interface/usecase"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ItemTransferHandler struct {
	transferUC usecase_interfaces.IItemTransferUseCase
}

func NewItemTransferHandler(transferUC usecase_interfaces.IItemTransferUseCase) *ItemTransferHandler {
	return &ItemTransferHandler{transferUC: transferUC}
}

// MarkPostItemSold mark a post item as sold to another user
// @Summary Đánh dấu món đồ đã bán
// @Description Đánh dấu trang phục đã được bán cho một người dùng khác và kích hoạt trạng thái bàn giao (transfer)
// @Tags Community
// @Accept json
// @Produce json
// @Param postItemID path string true "ID chi tiết món đồ trong bài post"
// @Param body body dto.UpdatePostItemsBuyerReq true "Thông tin người mua"
// @Success 200 {object} shared_pres.APIResponse "Đánh dấu đã bán thành công"
// @Router /api/v1/transfers/{postItemID}/mark-sold [post]
func (h *ItemTransferHandler) MarkPostItemSold(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	postItemID, err := uuid.Parse(c.Param("postItemID"))
	if err != nil {
		return err
	}

	var input dto.UpdatePostItemsBuyerReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	if err := h.transferUC.MarkPostItemSold(c.Request.Context(), userID, postItemID, input.BuyerUserID); err != nil {
		return err
	}

	shared_pres.Success(c, "Đánh dấu đã bán thành công", nil)
	return nil
}

// GetPendingTransfers get pending transfer items of user
// @Summary Danh sách trang phục đang chờ nhận bàn giao
// @Description Lấy danh sách các trang phục do người khác đánh dấu bán cho bạn đang chờ xác nhận nhận về tủ đồ
// @Tags Community
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.PostItemRes} "Lấy danh sách đang chờ nhận thành công"
// @Router /api/v1/transfers/pending [get]
func (h *ItemTransferHandler) GetPendingTransfers(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	response, err := h.transferUC.GetPendingTransfers(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Lấy danh sách đang chờ nhận thành công", response)
	return nil
}

// AcceptTransfer accept item transfer into user's wardrobe
// @Summary Chấp nhận nhận bàn giao trang phục
// @Description Đồng ý nhận trang phục đã mua về tủ đồ cá nhân (Trang phục sẽ chuyển quyền sở hữu)
// @Tags Community
// @Produce json
// @Param postItemID path string true "ID chi tiết món đồ trong bài post"
// @Success 200 {object} shared_pres.APIResponse{data=dto.PostItemRes} "Nhận món đồ vào tủ thành công"
// @Router /api/v1/transfers/{postItemID}/accept [post]
func (h *ItemTransferHandler) AcceptTransfer(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	postItemID, err := uuid.Parse(c.Param("postItemID"))
	if err != nil {
		return err
	}

	response, err := h.transferUC.AcceptTransfer(c.Request.Context(), userID, postItemID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Nhận món đồ vào tủ thành công", response)
	return nil
}

// DeclineTransfer decline item transfer
// @Summary Từ chối nhận bàn giao trang phục
// @Description Từ chối nhận bàn giao trang phục mua từ bài đăng cộng đồng
// @Tags Community
// @Produce json
// @Param postItemID path string true "ID chi tiết món đồ trong bài post"
// @Success 200 {object} shared_pres.APIResponse "Từ chối nhận món đồ thành công"
// @Router /api/v1/transfers/{postItemID}/decline [post]
func (h *ItemTransferHandler) DeclineTransfer(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	postItemID, err := uuid.Parse(c.Param("postItemID"))
	if err != nil {
		return err
	}

	if err := h.transferUC.DeclineTransfer(c.Request.Context(), userID, postItemID); err != nil {
		return err
	}

	shared_pres.Success(c, "Từ chối nhận món đồ thành công", nil)
	return nil
}
