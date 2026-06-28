package handler

import (
	"smart-wardrobe-be/internal/modules/brand/application/dto"
	usecase_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BrandChatHandler struct {
	brandUC usecase_interfaces.IBrandCoreUseCase
}

func NewBrandChatHandler(brandUC usecase_interfaces.IBrandCoreUseCase) *BrandChatHandler {
	return &BrandChatHandler{brandUC: brandUC}
}

// GetUserConversation returns the user's conversation with a brand.
// @Summary Lấy cuộc hội thoại hiện tại (User)
// @Tags Brand Chat
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandConversationRes}
// @Router /api/v1/brands/{brandId}/conversation [get]
func (h *BrandChatHandler) GetUserConversation(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.brandUC.GetUserConversation(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy thông tin hội thoại thành công", res)
	return nil
}

// SendUserMessage sends a message from a user to a brand.
// @Summary Gửi tin nhắn đến brand (User)
// @Tags Brand Chat
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param body body dto.SendBrandChatMessageReq true "Nội dung tin nhắn"
// @Success 201 {object} shared_pres.APIResponse{data=dto.BrandConversationMessageRes}
// @Router /api/v1/brands/{brandId}/conversation/messages [post]
func (h *BrandChatHandler) SendUserMessage(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	var input dto.SendBrandChatMessageReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.brandUC.SendUserMessage(c.Request.Context(), userID, brandID, input)
	if err != nil {
		return err
	}
	shared_pres.Created(c, "Gửi tin nhắn thành công", res)
	return nil
}

// ListBrandConversations lists conversations for a brand portal.
// @Summary Lấy danh sách các cuộc hội thoại của brand (Staff)
// @Tags Brand Chat
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.BrandConversationRes}
// @Router /api/v1/brand-portal/brands/{brandId}/conversations [get]
func (h *BrandChatHandler) ListBrandConversations(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.brandUC.ListBrandConversations(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy danh sách hội thoại thành công", res)
	return nil
}

// ListConversationMessages lists messages in a conversation.
// @Summary Lấy danh sách tin nhắn trong cuộc hội thoại (Staff)
// @Tags Brand Chat
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param conversationId path string true "ID cuộc hội thoại"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.BrandConversationMessageRes}
// @Router /api/v1/brand-portal/brands/{brandId}/conversations/{conversationId}/messages [get]
func (h *BrandChatHandler) ListConversationMessages(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	conversationID, err := uuid.Parse(c.Param("conversationId"))
	if err != nil {
		return err
	}
	res, err := h.brandUC.ListConversationMessages(c.Request.Context(), userID, brandID, conversationID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, "Lấy danh sách tin nhắn thành công", res)
	return nil
}

// SendStaffMessage sends a message from a staff member to a conversation.
// @Summary Gửi phản hồi của brand staff (Staff)
// @Tags Brand Chat
// @Accept json
// @Produce json
// @Param brandId path string true "ID brand"
// @Param conversationId path string true "ID cuộc hội thoại"
// @Param body body dto.SendBrandChatMessageReq true "Nội dung phản hồi"
// @Success 201 {object} shared_pres.APIResponse{data=dto.BrandConversationMessageRes}
// @Router /api/v1/brand-portal/brands/{brandId}/conversations/{conversationId}/messages [post]
func (h *BrandChatHandler) SendStaffMessage(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	conversationID, err := uuid.Parse(c.Param("conversationId"))
	if err != nil {
		return err
	}
	var input dto.SendBrandChatMessageReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}
	res, err := h.brandUC.SendStaffMessage(c.Request.Context(), userID, brandID, conversationID, input)
	if err != nil {
		return err
	}
	shared_pres.Created(c, "Gửi phản hồi thành công", res)
	return nil
}
