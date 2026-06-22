// Package retrieval implements candidate retrieval, taxonomy term expansion, and lexical/semantic query rewriting.
package retrieval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/types"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/ai"
)

// LLMRecommendationQueryRewriter thực hiện viết lại truy vấn gợi ý trang phục bằng cách ủy quyền cho dịch vụ LLM/AI.
type LLMRecommendationQueryRewriter struct {
	aiService ai.IAIService
	cfg       *config.Config
	local     LocalRecommendationQueryRewriter
}

// NewLLMRecommendationQueryRewriter khởi tạo thực thể [LLMRecommendationQueryRewriter].
func NewLLMRecommendationQueryRewriter(aiService ai.IAIService, cfg *config.Config) *LLMRecommendationQueryRewriter {
	return &LLMRecommendationQueryRewriter{
		aiService: aiService,
		cfg:       cfg,
		local:     LocalRecommendationQueryRewriter{},
	}
}

type llmRecommendationRetrievalQuery struct {
	SemanticQuery string   `json:"semantic_query"`
	LexicalTerms  []string `json:"lexical_terms"`
	ExcludedTerms []string `json:"excluded_terms"`
	HardFilters   struct {
		Seasonality   []string `json:"seasonality"`
		CategorySlugs []string `json:"category_slugs"`
	} `json:"hard_filters"`
}

// Rewrite gửi yêu cầu viết lại ý định gợi ý trang phục tới dịch vụ AI để có được một truy vấn tinh chỉnh tối ưu hơn.
//
// Hành vi:
// 1. Sinh bản nháp truy vấn cơ bản cục bộ bằng [LocalRecommendationQueryRewriter.Rewrite] làm baseline.
// 2. Xây dựng System Prompt và Payload User Prompt chứa thông tin baseline và danh mục hợp lệ thông qua [buildLLMRecommendationRewriterPrompts].
// 3. Gọi dịch vụ AI tạo văn bản phản hồi dạng JSON.
// 4. Kiểm tra, xác thực và chuẩn hóa kết quả đầu ra của AI qua [validateLLMRecommendationRetrievalQuery].
// 5. Trả về cấu trúc [RecommendationRetrievalQuery] an toàn đã lọc sạch mã độc và giới hạn chiều dài.
//
// Đầu vào mẫu:
//
//	intent: dto.ParsedIntent{SemanticQuery: "Đi tiệc cưới ngoài trời mùa hè"}
//
// Đầu ra mẫu:
//
//	(types.RecommendationRetrievalQuery{RewriterSource: "llm", ...}, nil)
func (r LLMRecommendationQueryRewriter) Rewrite(ctx context.Context, intent dto.ParsedIntent) (types.RecommendationRetrievalQuery, error) {
	if r.aiService == nil {
		return types.RecommendationRetrievalQuery{}, errors.New("llm rewriter ai service is nil")
	}
	localQuery, err := r.local.Rewrite(ctx, intent)
	if err != nil {
		return types.RecommendationRetrievalQuery{}, err
	}

	systemPrompt, userPrompt, err := buildLLMRecommendationRewriterPrompts(intent, localQuery)
	if err != nil {
		return types.RecommendationRetrievalQuery{}, err
	}
	if r.cfg.AI.RewriterPromptMaxCharacters > 0 && len([]rune(userPrompt)) > r.cfg.AI.RewriterPromptMaxCharacters {
		return types.RecommendationRetrievalQuery{}, fmt.Errorf("recommendation rewriter prompt exceeds configured character limit")
	}
	response, err := r.aiService.GenerateChatText(ctx, systemPrompt, userPrompt, ai.TextGenerationOptions{MaxOutputTokens: r.cfg.AI.RewriterMaxOutputTokens})
	if err != nil {
		return types.RecommendationRetrievalQuery{}, err
	}
	return validateLLMRecommendationRetrievalQuery(response, r.cfg)
}

// buildLLMRecommendationRewriterPrompts tạo ra prompt hệ thống và dữ liệu JSON cho prompt người dùng gửi lên LLM để thực hiện viết lại truy vấn.
func buildLLMRecommendationRewriterPrompts(intent dto.ParsedIntent, localQuery types.RecommendationRetrievalQuery) (string, string, error) {
	systemPrompt := strings.Join([]string{
		"You rewrite outfit recommendation intent into a compact retrieval query.",
		"Return ONLY one JSON object. Do not wrap it in markdown.",
		"Allowed keys: semantic_query, lexical_terms, excluded_terms, hard_filters.",
		"hard_filters may contain only seasonality and category_slugs arrays.",
		"Do not output SQL, PostgreSQL tsquery syntax, operators, or executable query text.",
		"Use only concise taxonomy-aligned terms likely to appear in wardrobe metadata.",
	}, "\n")

	payload := map[string]any{
		"intent": intent,
		"local_baseline": map[string]any{
			"semantic_query": localQuery.SemanticQuery,
			"lexical_terms":  ExtractTermStrings(localQuery.LexicalTerms),
			"excluded_terms": ExtractTermStrings(localQuery.ExcludedTerms),
			"hard_filters":   localQuery.HardFilters,
		},
		"allowed": map[string]any{
			"seasonality":    []string{"spring", "summer", "autumn", "winter"},
			"category_slugs": RecommendationAllowedCategorySlugs(),
		},
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}
	return systemPrompt, string(bytes), nil
}

// validateLLMRecommendationRetrievalQuery phân tích cú pháp và xác thực tính an toàn của phản hồi JSON thô từ LLM.
//
// Hành vi:
// 1. Phân tính cú pháp chuỗi JSON thành struct nội bộ [llmRecommendationRetrievalQuery].
// 2. Kiểm tra mã độc hoặc SQL Injection/tsquery unsafe syntax qua [containsUnsafeQuerySyntax] đối với trường `semantic_query`.
// 3. Giới hạn độ dài tối đa của semantic query và số lượng từ khóa lexical, excluded theo cấu hình hệ thống.
// 4. Xác thực và chuẩn hóa các danh sách từ khóa tìm kiếm (lexical) và loại trừ (excluded).
// 5. Xác thực các bộ lọc cứng (mùa vụ, danh mục) xem có nằm trong danh sách được phép hay không.
// 6. Chuyển đổi và trả về cấu trúc [RecommendationRetrievalQuery] hoàn chỉnh.
//
// Đầu vào mẫu:
//
//	raw: `{"semantic_query": "đầm dự tiệc mát mẻ", "lexical_terms": ["đầm", "tiệc"], "excluded_terms": ["ấm"], "hard_filters": {"seasonality": ["summer"], "category_slugs": ["ao"]}}`
//
// Đầu ra mẫu:
//
//	(types.RecommendationRetrievalQuery{SemanticQuery: "đầm dự tiệc mát mẻ", ...}, nil)
func validateLLMRecommendationRetrievalQuery(raw string, cfg *config.Config) (types.RecommendationRetrievalQuery, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return types.RecommendationRetrievalQuery{}, errors.New("empty llm rewriter output")
	}
	var parsed llmRecommendationRetrievalQuery
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return types.RecommendationRetrievalQuery{}, fmt.Errorf("invalid llm rewriter json: %w", err)
	}
	if containsUnsafeQuerySyntax(parsed.SemanticQuery) {
		return types.RecommendationRetrievalQuery{}, errors.New("semantic query contains unsafe query syntax")
	}
	semanticQuery := strings.TrimSpace(parsed.SemanticQuery)
	maxSemanticLength := 512
	maxLexicalTerms := 24
	maxExcludedTerms := 24
	if cfg != nil {
		if cfg.RAG.RecommendationRewriterMaxSemanticLength > 0 {
			maxSemanticLength = cfg.RAG.RecommendationRewriterMaxSemanticLength
		}
		if cfg.RAG.RecommendationRewriterMaxLexicalTerms > 0 {
			maxLexicalTerms = cfg.RAG.RecommendationRewriterMaxLexicalTerms
		}
		if cfg.RAG.RecommendationRewriterMaxExcludedTerms > 0 {
			maxExcludedTerms = cfg.RAG.RecommendationRewriterMaxExcludedTerms
		}
	}

	if len(semanticQuery) > maxSemanticLength {
		return types.RecommendationRetrievalQuery{}, fmt.Errorf("semantic query exceeds max length %d", maxSemanticLength)
	}

	lexicalTerms, err := validateLLMTerms(parsed.LexicalTerms, maxLexicalTerms, "lexical_terms")
	if err != nil {
		return types.RecommendationRetrievalQuery{}, err
	}
	excludedTerms, err := validateLLMTerms(parsed.ExcludedTerms, maxExcludedTerms, "excluded_terms")
	if err != nil {
		return types.RecommendationRetrievalQuery{}, err
	}
	seasonality, err := validateLLMSeasonalityFilters(parsed.HardFilters.Seasonality)
	if err != nil {
		return types.RecommendationRetrievalQuery{}, err
	}
	categorySlugs, err := validateLLMCategoryFilters(parsed.HardFilters.CategorySlugs)
	if err != nil {
		return types.RecommendationRetrievalQuery{}, err
	}

	lexicalTermsSlice := make([]types.RetrievalTerm, len(lexicalTerms))
	for i, t := range lexicalTerms {
		lexicalTermsSlice[i] = types.RetrievalTerm{Value: t, Source: types.RetrievalTermSourceRaw}
	}
	excludedTermsSlice := make([]types.RetrievalTerm, len(excludedTerms))
	for i, t := range excludedTerms {
		excludedTermsSlice[i] = types.RetrievalTerm{Value: t, Source: types.RetrievalTermSourceRaw}
	}

	return types.RecommendationRetrievalQuery{
		SemanticQuery:  semanticQuery,
		LexicalTerms:   lexicalTermsSlice,
		ExcludedTerms:  excludedTermsSlice,
		RewriterSource: "llm",
		HardFilters: repositories.RecommendationHardFilters{
			Seasonality:   seasonality,
			CategorySlugs: categorySlugs,
		},
	}, nil
}

// validateLLMTerms xác thực số lượng từ khóa và kiểm tra các cú pháp truy vấn không an toàn cho danh sách từ khóa do LLM sinh ra.
func validateLLMTerms(terms []string, max int, field string) ([]string, error) {
	if len(terms) > max {
		return nil, fmt.Errorf("%s exceeds max count %d", field, max)
	}
	if slices.ContainsFunc(terms, containsUnsafeQuerySyntax) {
		return nil, fmt.Errorf("%s contains unsafe query syntax", field)
	}
	return NormalizeTermSet(terms), nil
}

// validateLLMSeasonalityFilters kiểm tra xem các mùa được lọc có hợp lệ (nằm trong: spring, summer, autumn, winter) hay không.
func validateLLMSeasonalityFilters(values []string) ([]string, error) {
	allowed := map[string]bool{"spring": true, "summer": true, "autumn": true, "winter": true}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" && !allowed[value] {
			return nil, fmt.Errorf("invalid seasonality hard filter %q", value)
		}
	}
	return NormalizeTermSet(values), nil
}

// validateLLMCategoryFilters kiểm tra xem danh mục được lọc có thuộc danh sách category slugs được hệ thống cho phép hay không.
func validateLLMCategoryFilters(values []string) ([]string, error) {
	allowed := map[string]bool{}
	for _, slug := range RecommendationAllowedCategorySlugs() {
		allowed[slug] = true
	}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" && !allowed[value] {
			return nil, fmt.Errorf("invalid category hard filter %q", value)
		}
	}
	return NormalizeTermSet(values), nil
}

// containsUnsafeQuerySyntax phát hiện các dấu hiệu tấn công SQL Injection hoặc cú pháp PostgreSQL tsquery không an toàn trong chuỗi văn bản.
func containsUnsafeQuerySyntax(value string) bool {
	normalized := strings.ToLower(value)
	unsafeFragments := []string{"@@", "plainto_tsquery", "websearch_to_tsquery", "to_tsquery", "select ", " where ", " from ", "drop ", "delete ", "insert ", "update ", ";", "--"}
	for _, fragment := range unsafeFragments {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}
