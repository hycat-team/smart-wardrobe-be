package handler

import (
	"fmt"
	"time"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/streamutils"
	"smart-wardrobe-be/pkg/utils/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RecommendOutfit recommendations for outfits based on user wardrobe
// @Summary Gợi ý phối đồ từ tủ đồ
// @Description Nhận gợi ý phối đồ từ các trang phục có sẵn trong tủ đồ của người dùng dựa trên dịp, thời tiết và phong cách
// @Tags Wardrobe AI
// @Accept json
// @Produce json
// @Param body body dto.RecommendOutfitReq true "Yêu cầu gợi ý phối đồ"
// @Success 200 {object} shared_pres.APIResponse{data=dto.RecommendedOutfitRes} "Gợi ý phối đồ thành công"
// @Router /api/v1/ai/outfit-recommendations [post]
func (h *WardrobeHandler) RecommendOutfit(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input dto.RecommendOutfitReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.wardrobeUseCase.RecommendOutfit(c.Request.Context(), userID, input)
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Tạo gợi ý phối đồ thành công", response)
	return nil
}

// CreateChatSession create new AI chat session
// @Summary Tạo cuộc trò chuyện AI mới
// @Description Khởi tạo một phiên tư vấn phong cách thời trang mới với stylist AI
// @Tags Wardrobe AI
// @Accept json
// @Produce json
// @Param body body dto.CreateChatSessionReq true "Yêu cầu tạo cuộc trò chuyện"
// @Success 201 {object} shared_pres.APIResponse{data=dto.ChatSessionRes} "Tạo cuộc trò chuyện thành công"
// @Router /api/v1/ai/chat/sessions [post]
func (h *WardrobeHandler) CreateChatSession(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input dto.CreateChatSessionReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.wardrobeUseCase.CreateChatSession(c.Request.Context(), userID, input.Title)
	if err != nil {
		return err
	}

	shared_pres.Created(c, "Tạo cuộc trò chuyện thành công", response)
	return nil
}

// GetChatSessions list chat sessions
// @Summary Lấy danh sách cuộc trò chuyện AI
// @Description Lấy tất cả các phiên trò chuyện của người dùng hiện tại với stylist AI
// @Tags Wardrobe AI
// @Accept json
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.ChatSessionRes} "Lấy danh sách cuộc trò chuyện thành công"
// @Router /api/v1/ai/chat/sessions [get]
func (h *WardrobeHandler) GetChatSessions(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	response, err := h.wardrobeUseCase.GetChatSessions(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Lấy danh sách cuộc trò chuyện thành công", response)
	return nil
}

// GetChatMessages list chat messages
// @Summary Lấy lịch sử tin nhắn AI
// @Description Lấy toàn bộ các tin nhắn trong một phiên trò chuyện cụ thể
// @Tags Wardrobe AI
// @Accept json
// @Produce json
// @Param contextID path string true "ID cuộc trò chuyện"
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.ChatMessageRes} "Lấy lịch sử tin nhắn thành công"
// @Router /api/v1/ai/chat/sessions/{contextID}/messages [get]
func (h *WardrobeHandler) GetChatMessages(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	contextID, err := uuid.Parse(c.Param("contextID"))
	if err != nil {
		return apperror.NewBadRequest("Định dạng ID cuộc trò chuyện không hợp lệ.")
	}

	response, err := h.wardrobeUseCase.GetChatMessages(c.Request.Context(), userID, contextID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, "Lấy lịch sử tin nhắn thành công", response)
	return nil
}

// ArchiveChatSession archive chat session
// @Summary Lưu trữ cuộc trò chuyện AI
// @Description Lưu trữ/ẩn cuộc trò chuyện với stylist AI
// @Tags Wardrobe AI
// @Accept json
// @Produce json
// @Param contextID path string true "ID cuộc trò chuyện"
// @Success 200 {object} shared_pres.APIResponse "Lưu trữ cuộc trò chuyện thành công"
// @Router /api/v1/ai/chat/sessions/{contextID}/archive [patch]
func (h *WardrobeHandler) ArchiveChatSession(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	contextID, err := uuid.Parse(c.Param("contextID"))
	if err != nil {
		return apperror.NewBadRequest("Định dạng ID cuộc trò chuyện không hợp lệ.")
	}

	if err := h.wardrobeUseCase.ArchiveChatSession(c.Request.Context(), userID, contextID); err != nil {
		return err
	}

	shared_pres.Success(c, "Lưu trữ cuộc trò chuyện thành công", nil)
	return nil
}

// StreamChatMessage stream chat response
// @Summary Nhắn tin với stylist AI (Stream SSE)
// @Description Gửi tin nhắn cho stylist AI và nhận phản hồi dạng stream sự kiện (Server-Sent Events)
// @Tags Wardrobe AI
// @Accept json
// @Produce text/event-stream
// @Param contextID path string true "ID cuộc trò chuyện"
// @Param body body dto.SendChatMessageReq true "Nội dung tin nhắn gửi đi"
// @Success 200 {string} string "Stream Server-Sent Events (SSE) phản hồi từ AI"
// @Router /api/v1/ai/chat/sessions/{contextID}/messages/stream [post]
func (h *WardrobeHandler) StreamChatMessage(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	contextID, err := uuid.Parse(c.Param("contextID"))
	if err != nil {
		return apperror.NewBadRequest("Định dạng ID cuộc trò chuyện không hợp lệ.")
	}

	var input dto.SendChatMessageReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	_, aiMessage, err := h.wardrobeUseCase.ProcessChatMessage(c.Request.Context(), userID, contextID, input.Content)
	if err != nil {
		return err
	}

	flusher, err := shared_pres.InitSSE(c)
	if err != nil {
		return err
	}

	chunks := streamutils.SplitForStream(aiMessage.Content, 24)
	for _, chunk := range chunks {
		if _, err := fmt.Fprintf(c.Writer, "event: chunk\ndata: %s\n\n", streamutils.SanitizeSSEData(chunk)); err != nil {
			return err
		}
		flusher.Flush()
		time.Sleep(25 * time.Millisecond)
	}

	if _, err := fmt.Fprintf(c.Writer, "event: done\ndata: %s\n\n", streamutils.SanitizeSSEData(aiMessage.Content)); err != nil {
		return err
	}

	flusher.Flush()
	return nil
}

