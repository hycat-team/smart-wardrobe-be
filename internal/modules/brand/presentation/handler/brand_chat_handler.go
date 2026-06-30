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
	chatUC usecase_interfaces.IBrandChatUseCase
}

func NewBrandChatHandler(chatUC usecase_interfaces.IBrandChatUseCase) *BrandChatHandler {
	return &BrandChatHandler{chatUC: chatUC}
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
	res, err := h.chatUC.GetUserConversation(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandChatGetUserConversationSuccess, res)
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
	res, err := h.chatUC.SendUserMessage(c.Request.Context(), userID, brandID, input)
	if err != nil {
		return err
	}
	shared_pres.Created(c, msgBrandChatSendUserMessageSuccess, res)
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
	res, err := h.chatUC.ListBrandConversations(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandChatListBrandConversationsSuccess, res)
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
	res, err := h.chatUC.ListConversationMessages(c.Request.Context(), userID, brandID, conversationID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandChatListConversationMessagesSuccess, res)
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
	res, err := h.chatUC.SendStaffMessage(c.Request.Context(), userID, brandID, conversationID, input)
	if err != nil {
		return err
	}
	shared_pres.Created(c, msgBrandChatSendStaffMessageSuccess, res)
	return nil
}

// MarkUserConversationRead marks the current user's brand conversation as read.
// @Summary Đánh dấu đã đọc hội thoại brand (User)
// @Tags Brand Chat
// @Produce json
// @Param brandId path string true "ID brand"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandConversationRes}
// @Router /api/v1/brands/{brandId}/conversation/read [post]
func (h *BrandChatHandler) MarkUserConversationRead(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return err
	}
	res, err := h.chatUC.MarkUserConversationRead(c.Request.Context(), userID, brandID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandChatMarkConversationReadSuccess, res)
	return nil
}

// MarkStaffConversationRead marks a brand conversation as read for staff.
// @Summary Đánh dấu đã đọc hội thoại brand (Staff)
// @Tags Brand Chat
// @Produce json
// @Param brandId path string true "ID brand"
// @Param conversationId path string true "ID hội thoại"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandConversationRes}
// @Router /api/v1/brand-portal/brands/{brandId}/conversations/{conversationId}/read [post]
func (h *BrandChatHandler) MarkStaffConversationRead(c *gin.Context) error {
	userID, brandID, conversationID, err := parseStaffConversationParams(c)
	if err != nil {
		return err
	}
	res, err := h.chatUC.MarkStaffConversationRead(c.Request.Context(), userID, brandID, conversationID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandChatMarkConversationReadSuccess, res)
	return nil
}

// CloseConversation closes a brand conversation.
// @Summary Đóng hội thoại brand (Staff)
// @Tags Brand Chat
// @Produce json
// @Param brandId path string true "ID brand"
// @Param conversationId path string true "ID hội thoại"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandConversationRes}
// @Router /api/v1/brand-portal/brands/{brandId}/conversations/{conversationId}/close [post]
func (h *BrandChatHandler) CloseConversation(c *gin.Context) error {
	userID, brandID, conversationID, err := parseStaffConversationParams(c)
	if err != nil {
		return err
	}
	res, err := h.chatUC.CloseBrandConversation(c.Request.Context(), userID, brandID, conversationID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandChatCloseConversationSuccess, res)
	return nil
}

// ReopenConversation reopens a brand conversation.
// @Summary Mở lại hội thoại brand (Staff)
// @Tags Brand Chat
// @Produce json
// @Param brandId path string true "ID brand"
// @Param conversationId path string true "ID hội thoại"
// @Success 200 {object} shared_pres.APIResponse{data=dto.BrandConversationRes}
// @Router /api/v1/brand-portal/brands/{brandId}/conversations/{conversationId}/reopen [post]
func (h *BrandChatHandler) ReopenConversation(c *gin.Context) error {
	userID, brandID, conversationID, err := parseStaffConversationParams(c)
	if err != nil {
		return err
	}
	res, err := h.chatUC.ReopenBrandConversation(c.Request.Context(), userID, brandID, conversationID)
	if err != nil {
		return err
	}
	shared_pres.Success(c, msgBrandChatReopenConversationSuccess, res)
	return nil
}

func parseStaffConversationParams(c *gin.Context) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}
	brandID, err := uuid.Parse(c.Param("brandId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}
	conversationID, err := uuid.Parse(c.Param("conversationId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}
	return userID, brandID, conversationID, nil
}
