package handler

import (
	"context"
	"fmt"
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	wardrobeerrors "smart-wardrobe-be/internal/modules/wardrobe/application/errors"
	usecase_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	shared_pres "smart-wardrobe-be/internal/shared/presentation"
	"smart-wardrobe-be/pkg/utils/contextutils"
	"smart-wardrobe-be/pkg/utils/streamutils"
	"smart-wardrobe-be/pkg/utils/validation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var _ shared_dto.PaginationQuery

const (
	msgAiRecommendOutfitSuccess    = "Tạo gợi ý phối đồ thành công"
	msgAiCreateChatSessionSuccess  = "Tạo cuộc trò chuyện thành công"
	msgAiGetChatSessionsSuccess    = "Lấy danh sách cuộc trò chuyện thành công"
	msgAiGetChatMessagesSuccess    = "Lấy lịch sử tin nhắn thành công"
	msgAiArchiveChatSessionSuccess = "Lưu trữ cuộc trò chuyện thành công"
	msgAiDeleteChatSessionSuccess  = "Xoá cuộc trò chuyện thành công"
	msgAiUpdateChatSessionSuccess  = "Cập nhật cuộc trò chuyện thành công"
)

type WardrobeAIHandler struct {
	recommendationUseCase usecase_interfaces.IOutfitRecommendationUseCase
	chatUseCase           usecase_interfaces.IWardrobeChatUseCase
}

func NewWardrobeAIHandler(
	recommendationUseCase usecase_interfaces.IOutfitRecommendationUseCase,
	chatUseCase usecase_interfaces.IWardrobeChatUseCase,
) *WardrobeAIHandler {
	return &WardrobeAIHandler{
		recommendationUseCase: recommendationUseCase,
		chatUseCase:           chatUseCase,
	}
}

// RecommendOutfit recommendations for outfits based on user wardrobe
// @Summary Gợi ý phối đồ từ tủ đồ
// @Description Nhận gợi ý phối đồ từ các trang phục có sẵn trong tủ đồ của người dùng dựa trên dịp, thời tiết và phong cách.
// @Description
// @Description Các trường trong Request Body:
// @Description - occasion (Dịp phối đồ, gợi ý: casual, work, date, party, sport,...)
// @Description - styleTarget (Phong cách hướng tới, gợi ý: minimalist, vintage, streetwear, preppy, sporty, elegant,...)
// @Description - season (Mùa phối đồ, enum: spring, summer, autumn, winter, all)
// @Description - weather (Thời tiết hiện tại, gợi ý: hot, cold, warm, cool, rainy,...)
// @Description - colorTone (Tông màu phối đồ, gợi ý: light, dark, pastel, earthy, neon,...)
// @Description - details (Ghi chú thêm bằng tay - tự do)
// @Tags Wardrobe AI
// @Accept json
// @Produce json
// @Param body body dto.RecommendOutfitReq true "Yêu cầu gợi ý phối đồ"
// @Success 200 {object} shared_pres.APIResponse{data=dto.RecommendedOutfitRes} "Gợi ý phối đồ thành công"
// @Router /api/v1/ai/outfit-recommendations [post]
func (h *WardrobeAIHandler) RecommendOutfit(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input dto.RecommendOutfitReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.recommendationUseCase.RecommendOutfit(c.Request.Context(), userID, input)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgAiRecommendOutfitSuccess, response)
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
func (h *WardrobeAIHandler) CreateChatSession(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	var input dto.CreateChatSessionReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.chatUseCase.CreateChatSession(c.Request.Context(), userID, input.Title)
	if err != nil {
		return err
	}

	shared_pres.Created(c, msgAiCreateChatSessionSuccess, response)
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
func (h *WardrobeAIHandler) GetChatSessions(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	response, err := h.chatUseCase.GetChatSessions(c.Request.Context(), userID)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgAiGetChatSessionsSuccess, response)
	return nil
}

// GetChatMessages list chat messages
// @Summary Lấy lịch sử tin nhắn AI
// @Description Lấy toàn bộ các tin nhắn trong một phiên trò chuyện cụ thể (phân trang)
// @Tags Wardrobe AI
// @Accept json
// @Produce json
// @Param contextID path string true "ID cuộc trò chuyện"
// @Param query query dto.GetChatMessagesQueryReq false "Bộ lọc phân trang lịch sử tin nhắn"
// @Success 200 {object} shared_pres.APIResponse{data=shared_dto.PaginationResult[dto.ChatMessageRes]} "Lấy lịch sử tin nhắn thành công"
// @Router /api/v1/ai/chat/sessions/{contextID}/messages [get]
func (h *WardrobeAIHandler) GetChatMessages(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	contextID, err := uuid.Parse(c.Param("contextID"))
	if err != nil {
		return wardrobeerrors.ErrInvalidChatIDFormat()
	}

	var query dto.GetChatMessagesQueryReq
	if err := validation.BindQuery(c, &query); err != nil {
		return err
	}

	response, err := h.chatUseCase.GetChatMessages(c.Request.Context(), userID, contextID, query)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgAiGetChatMessagesSuccess, response)
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
func (h *WardrobeAIHandler) ArchiveChatSession(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	contextID, err := uuid.Parse(c.Param("contextID"))
	if err != nil {
		return wardrobeerrors.ErrInvalidChatIDFormat()
	}

	if err := h.chatUseCase.ArchiveChatSession(c.Request.Context(), userID, contextID); err != nil {
		return err
	}

	shared_pres.Success(c, msgAiArchiveChatSessionSuccess, nil)
	return nil
}

// DeleteChatSession delete chat session
// @Summary Xóa cuộc trò chuyện AI
// @Description Xóa vĩnh viễn cuộc trò chuyện với stylist AI và toàn bộ lịch sử tin nhắn liên quan
// @Tags Wardrobe AI
// @Accept json
// @Produce json
// @Param contextID path string true "ID cuộc trò chuyện"
// @Success 200 {object} shared_pres.APIResponse "Xóa cuộc trò chuyện thành công"
// @Router /api/v1/ai/chat/sessions/{contextID} [delete]
func (h *WardrobeAIHandler) DeleteChatSession(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	contextID, err := uuid.Parse(c.Param("contextID"))
	if err != nil {
		return wardrobeerrors.ErrInvalidChatIDFormat()
	}

	if err := h.chatUseCase.DeleteChatSession(c.Request.Context(), userID, contextID); err != nil {
		return err
	}

	shared_pres.Success(c, msgAiDeleteChatSessionSuccess, nil)
	return nil
}

// UpdateChatSession update chat session
// @Summary Cập nhật thông tin cuộc trò chuyện AI
// @Description Cập nhật thông tin chi tiết của phiên trò chuyện với stylist AI (ví dụ: đổi tiêu đề)
// @Tags Wardrobe AI
// @Accept json
// @Produce json
// @Param contextID path string true "ID cuộc trò chuyện"
// @Param body body dto.UpdateChatSessionReq true "Thông tin cập nhật cuộc trò chuyện"
// @Success 200 {object} shared_pres.APIResponse{data=dto.ChatSessionRes} "Cập nhật cuộc trò chuyện thành công"
// @Router /api/v1/ai/chat/sessions/{contextID} [patch]
func (h *WardrobeAIHandler) UpdateChatSession(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	contextID, err := uuid.Parse(c.Param("contextID"))
	if err != nil {
		return wardrobeerrors.ErrInvalidChatIDFormat()
	}

	var input dto.UpdateChatSessionReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	response, err := h.chatUseCase.UpdateChatSession(c.Request.Context(), userID, contextID, input)
	if err != nil {
		return err
	}

	shared_pres.Success(c, msgAiUpdateChatSessionSuccess, response)
	return nil
}


// StreamChatMessage stream chat response
// @Summary Nhắn tin với stylist AI (Stream SSE)
// @Description Gửi tin nhắn cho stylist AI và nhận phản hồi dạng stream sự kiện (Server-Sent Events).
// @Description Nếu mô hình AI phát hiện người dùng yêu cầu phối đồ từ tủ đồ cá nhân, nó sẽ thêm token '[ACTION:REDIRECT_OUTFIT]' vào cuối phản hồi stream.
// @Description CHÚ Ý: Token '[ACTION:REDIRECT_OUTFIT]' có thể bị phân mảnh (split) thành nhiều chunk nhỏ khi truyền tải stream (ví dụ: chunk 1 nhận '[ACTION:RE', chunk 2 nhận 'DIRECT_OUTFIT]').
// @Description Frontend cần tích luỹ toàn bộ chuỗi (accumulated string) hoặc ghép các chunk lại trước khi kiểm tra sự tồn tại của token này để hiển thị nút/card điều hướng sang tính năng Phối đồ chuyên dụng, thay vì chỉ kiểm tra đơn lẻ trên từng chunk nhận được.
// @Tags Wardrobe AI
// @Accept json
// @Produce text/event-stream
// @Param contextID path string true "ID cuộc trò chuyện"
// @Param body body dto.SendChatMessageReq true "Nội dung tin nhắn gửi đi"
// @Success 200 {string} string "Stream Server-Sent Events (SSE) phản hồi từ AI"
// @Router /api/v1/ai/chat/sessions/{contextID}/messages/stream [post]
func (h *WardrobeAIHandler) StreamChatMessage(c *gin.Context) error {
	userID, err := contextutils.GetUserId(c)
	if err != nil {
		return err
	}

	contextID, err := uuid.Parse(c.Param("contextID"))
	if err != nil {
		return wardrobeerrors.ErrInvalidChatIDFormat()
	}

	var input dto.SendChatMessageReq
	if err := validation.BindJSON(c, &input); err != nil {
		return err
	}

	streamCtx, cancelStream := context.WithCancel(c.Request.Context())
	defer cancelStream()

	textChan, commitFn, err := h.chatUseCase.ProcessChatMessageStream(streamCtx, userID, contextID, input.Content)
	if err != nil {
		return err
	}

	flusher, err := shared_pres.InitSSE(c)
	if err != nil {
		return err
	}

	success := false
	defer func() {
		if err := commitFn(success); err != nil {
			_ = c.Error(err)
		}
	}()

	var accumulated strings.Builder
	for {
		select {
		case <-c.Request.Context().Done():
			return nil
		case text, ok := <-textChan:
			if !ok {
				success = true
				if _, err := fmt.Fprintf(c.Writer, "event: done\ndata: %s\n\n", streamutils.SanitizeSSEData(accumulated.String())); err != nil {
					return err
				}
				flusher.Flush()
				return nil
			}

			accumulated.WriteString(text)
			if _, err := fmt.Fprintf(c.Writer, "event: chunk\ndata: %s\n\n", streamutils.SanitizeSSEData(text)); err != nil {
				return err
			}
			flusher.Flush()
		}
	}
}
