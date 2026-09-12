// Package synthesis implements response synthesis, LLM prompt assembly, response parsing, and validation.
package synthesis

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"smart-wardrobe-be/internal/config"
	aiexecutor_dto "smart-wardrobe-be/internal/modules/aiexecutor/application/dto"
	aiexecutor_contract "smart-wardrobe-be/internal/modules/aiexecutor/presentation/contract"
	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	fashion_mapper "smart-wardrobe-be/internal/modules/fashion/application/mapper"
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/types"
	"smart-wardrobe-be/internal/pkg/utils/stringutils"

	"github.com/google/uuid"
)

// GenerateOutfitRecommendation điều phối toàn bộ quá trình tạo gợi ý trang phục bằng AI.
//
// Hành vi:
// 1. Dựng Prompt người dùng bằng [BuildRecommendationPrompt] từ danh sách ứng viên và thông tin đầu vào.
// 2. Gửi Prompt hệ thống (Stylist cao cấp, trả về JSON, ngôn ngữ tiếng Việt) và Prompt người dùng tới [aiService.GenerateRecommendationText].
// 3. Nếu dịch vụ AI gặp lỗi hoặc trả về rỗng, tạo lỗi trace dự phòng qua [NewFallbackTraceError].
// 4. Phân tích cú pháp JSON phản hồi từ AI qua [ParseOutfitRecommendationJSON].
// 5. So khớp, đối chiếu danh sách gợi ý của AI với các ứng viên thực tế trong tủ đồ qua [MapLLMResponseToGroups].
// 6. Trả về cấu trúc [RecommendedOutfitRes] hoàn chỉnh cho người dùng.
//
// Đầu vào mẫu:
//
//	candidates: []types.CandidateForPrompt{...}
//	input: dto.RecommendOutfitReq{...}
//
// Đầu ra mẫu:
//
//	(*dto.RecommendedOutfitRes{Title: "Bộ đồ thanh lịch", Explanation: "...", Items: [...]}, nil)
func GenerateOutfitRecommendation(
	ctx context.Context,
	aiService aiexecutor_contract.IAIOrchestrator,
	userID uuid.UUID,
	candidates []types.CandidateForPrompt,
	input dto.RecommendOutfitReq,
	cfg *config.Config,
	fashionItemMap map[uuid.UUID]*dto.FashionItemContractDTO,
) (*dto.RecommendedOutfitRes, error) {
	userPrompt, err := BuildRecommendationPromptWithLimits(candidates, input, PromptLimits{CandidateLimit: cfg.AI.RecommendationPromptCandidateLimit, DescriptionMaxCharacters: cfg.AI.RecommendationDescriptionMaxCharacters, TagsLimit: cfg.AI.RecommendationTagsLimit, PromptMaxCharacters: cfg.AI.RecommendationPromptMaxCharacters}, fashionItemMap)
	if err != nil {
		return nil, err
	}
	responseText, err := aiService.GenerateRecommendationText(
		ctx,
		"You are a senior fashion stylist and wardrobe editor. Recommend the most suitable outfit from the available items, stay faithful to the actual item attributes, and respond with exactly one valid minified JSON object. A valid outfit MUST be either a single fullbody item (dress) OR a combination of at least 1 top and 1 bottom item. The fields title and explanation must be written in natural Vietnamese with proper diacritics. Internal candidate aliases (A1, A2, ...) MUST appear ONLY in primary_id and alternative_ids fields; NEVER write aliases in title or explanation. Candidates tagged with [Fashion Tags: ..., source:brand-item, ...] are brand sample items — prefer including at least one of them in the outfit (as primary or alternative) when they fit the style, occasion, and color harmony.",
		userPrompt,
		aiexecutor_dto.OrchestratorOptions{MaxOutputTokens: cfg.AI.RecommendationMaxOutputTokens, Temperature: 0.1, ResponseMIMEType: "application/json", ResponseSchema: outfitResponseSchema(), UserID: userID, Operation: "outfit"},
	)
	if err != nil {
		return nil, NewFallbackTraceError(
			"provider_error",
			err,
			userPrompt,
			"",
		)
	}

	if responseText == "" {
		return nil, NewFallbackTraceError(
			"empty_response",
			fmt.Errorf("empty response from LLM"),
			userPrompt,
			"",
		)
	}

	llmRes, cleanedResponse, err := ParseOutfitRecommendationJSON(responseText)
	if err != nil {
		return nil, NewFallbackTraceError(
			"invalid_json",
			err,
			userPrompt,
			responseText,
		)
	}
	resolveOutfitAliases(&llmRes, candidates)

	validGroups := fashion_mapper.MapLLMResponseToGroups(candidates, llmRes, fashionItemMap)
	if len(validGroups) == 0 || !HasValidOutfitStructureGroups(validGroups) {
		return nil, NewFallbackTraceError(
			"invalid_outfit_structure",
			fmt.Errorf("AI returned an invalid outfit structure: missing required top or bottom role"),
			userPrompt,
			cleanedResponse,
		)
	}

	return &dto.RecommendedOutfitRes{
		Title:       SanitizeOutfitNarrative(llmRes.Title),
		Explanation: SanitizeOutfitNarrative(llmRes.Explanation),
		Items:       validGroups,
	}, nil
}

// HasValidOutfitStructureGroups checks whether outfit groups contain (top AND bottom) OR fullbody.
func HasValidOutfitStructureGroups(groups []*dto.RecommendedItemGroup) bool {
	hasFullbody := false
	hasTop := false
	hasBottom := false
	for _, g := range groups {
		switch g.Role {
		case "fullbody":
			hasFullbody = true
		case "top":
			hasTop = true
		case "bottom":
			hasBottom = true
		}
	}
	return hasFullbody || (hasTop && hasBottom)
}

func outfitResponseSchema() any {
	return map[string]any{"type": "OBJECT", "required": []string{"title", "explanation", "items"}, "properties": map[string]any{"title": map[string]any{"type": "STRING"}, "explanation": map[string]any{"type": "STRING"}, "items": map[string]any{"type": "ARRAY", "items": map[string]any{"type": "OBJECT", "required": []string{"role", "primary_id", "alternative_ids"}, "properties": map[string]any{"role": map[string]any{"type": "STRING"}, "primary_id": map[string]any{"type": "STRING"}, "alternative_ids": map[string]any{"type": "ARRAY", "items": map[string]any{"type": "STRING"}}}}}}}
}

func resolveOutfitAliases(res *types.LlmOutfitResponse, candidates []types.CandidateForPrompt) {
	if res == nil {
		return
	}
	aliases := map[string]string{}
	for i, c := range candidates {
		aliases[fmt.Sprintf("A%d", i+1)] = c.Item.ID.String()
	}
	for i := range res.Items {
		if id, ok := aliases[res.Items[i].PrimaryID]; ok {
			res.Items[i].PrimaryID = id
		}
		for j, id := range res.Items[i].AlternativeIDs {
			if actual, ok := aliases[id]; ok {
				res.Items[i].AlternativeIDs[j] = actual
			}
		}
	}
}

// fallbackTraceError là một kiểu lỗi tùy chỉnh chứa các thông tin vết (trace info) hỗ trợ việc gỡ lỗi hoặc ghi nhận telemetry khi luồng AI bị lỗi và phải dùng HSL fallback.
type fallbackTraceError struct {
	kind            string
	cause           error
	prompt          string
	responsePreview string
}

func (e *fallbackTraceError) Error() string {
	if e == nil || e.cause == nil {
		return ""
	}
	return e.cause.Error()
}

func (e *fallbackTraceError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

// NewFallbackTraceError tạo một đối tượng [fallbackTraceError] mới với các thông tin vết chi tiết của prompt và response.
func NewFallbackTraceError(kind string, cause error, prompt, response string) error {
	return &fallbackTraceError{
		kind:            kind,
		cause:           cause,
		prompt:          prompt,
		responsePreview: stringutils.TruncateLogText(response, 600),
	}
}

// ClassifyFallbackTrace phân loại và phân tích một lỗi trace dự phòng để trích xuất các thông tin thống kê gỡ lỗi và phân loại lỗi do nhà cung cấp API nào (ví dụ: OpenAI, Gemini, Timeout, v.v.).
//
// Hành vi:
// 1. Sử dụng [errors.As] để chuyển đổi sang kiểu lỗi nội bộ [fallbackTraceError] nhằm lấy ra loại lỗi (kind), độ dài prompt, và nội dung phản hồi preview.
// 2. Phân tích chuỗi thông báo lỗi để suy đoán nhà cung cấp (providerHint) như OpenAI, Gemini, context timeout hay cancel.
// 3. Trả về thông tin phân loại chi tiết.
//
// Đầu vào mẫu:
//
//	err: lỗi sinh ra từ GenerateOutfitRecommendation
//
// Đầu ra mẫu:
//
//	(kind: "provider_error", providerHint: "gemini", promptLen: 1540, responsePreview: "...")
func ClassifyFallbackTrace(err error) (kind string, providerHint string, promptLen int, responsePreview string) {
	kind = "unknown"
	providerHint = "unknown"

	var traceErr *fallbackTraceError
	if errors.As(err, &traceErr) {
		if traceErr.kind != "" {
			kind = traceErr.kind
		}
		promptLen = len(traceErr.prompt)
		responsePreview = traceErr.responsePreview
	}

	errText := strings.ToLower(err.Error())
	switch {
	case strings.Contains(errText, "openai"):
		providerHint = "openai"
	case strings.Contains(errText, "google"):
		providerHint = "gemini"
	case errors.Is(err, context.Canceled):
		providerHint = "request_context"
	case errors.Is(err, context.DeadlineExceeded):
		providerHint = "timeout"
	}

	return kind, providerHint, promptLen, responsePreview
}
