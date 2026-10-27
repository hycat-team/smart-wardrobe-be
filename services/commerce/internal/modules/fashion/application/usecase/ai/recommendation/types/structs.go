package types

import (
	brandDto "smart-wardrobe-be/services/commerce/internal/modules/brand/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/repositories"
	wardrobeDto "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/services/commerce/internal/modules/wardrobe/domain/constants/outfititemcontext"
)

// CandidateForPrompt đại diện cho một món đồ ứng viên đã chuẩn bị đầy đủ thông tin (kèm tag thời trang) để đưa vào prompt gửi cho LLM.
type CandidateForPrompt struct {
	Item        *wardrobeDto.WardrobeItemContractDTO
	Tags        []string
	ItemContext outfititemcontext.OutfitItemContext
	BrandItem   *brandDto.BrandItemContractDTO
}

// RerankStats lưu trữ các thông số thống kê phục vụ cho việc chấm điểm và đa dạng hóa danh sách ứng viên sau khi xếp hạng.
type RerankStats struct {
	MinScore         float64
	MaxScore         float64
	AvgScore         float64
	DiversifiedCount int
}

// CandidateForRanking đại diện cho một món đồ ứng viên đang được xử lý trong pipeline chấm điểm và xếp hạng.
type CandidateForRanking struct {
	Item            *wardrobeDto.WardrobeItemContractDTO
	Source          CandidateSource
	VectorScore     float64
	LexicalScore    float64
	RetrievalScore  float64
	RetrievalRank   int
	RetrievalSource string // Đây là nguồn từ DB (vector, lexical, hybrid), giữ kiểu string để tương thích với repositories.HybridCandidate.RetrievalSource
	ItemContext     outfititemcontext.OutfitItemContext
	BrandItem       *brandDto.BrandItemContractDTO
}

// RankedCandidate đại diện cho một món đồ ứng viên đã được tính điểm và sắp xếp thứ hạng hoàn chỉnh.
type RankedCandidate struct {
	Item          *wardrobeDto.WardrobeItemContractDTO
	Score         float64
	Tags          []string
	Source        CandidateSource
	RetrievalRank int
	ItemContext   outfititemcontext.OutfitItemContext
	BrandItem     *brandDto.BrandItemContractDTO
}

// FallbackCandidateCounts đếm số lượng ứng viên tìm được qua các tầng lọc dự phòng khác nhau (Strict, Relaxed, General).
type FallbackCandidateCounts struct {
	Strict  int
	Relaxed int
	General int
}

// Total trả về tổng số lượng ứng viên thu thập được từ tất cả các tầng dự phòng.
func (c FallbackCandidateCounts) Total() int {
	return c.Strict + c.Relaxed + c.General
}

// PersonalizationContext carries the soft personal style context of one
// recommendation request (033 US1). The zero value means "no personalization":
// kill switch off, no entitlement, no profile, or a disabled profile.
//
// The profile is only ever a soft signal. It blends into the query vector and
// adds score bonuses/penalties; it never hard-filters candidates.
type PersonalizationContext struct {
	// Active reports whether a usable profile was loaded and merged. When
	// false every other field is zero and the pipeline runs as before 033.
	Active bool
	// ExplicitStyles is the declared preferred style list, normalised through
	// the existing taxonomy vocabulary. The request wins when both are present.
	ExplicitStyles []string
	// ExcludedStyles is the declared avoided style list (profile
	// excluded_styles), normalised through the same vocabulary.
	ExcludedStyles []string
	// LearnedStyles is the inferred style list, ordered by decreasing learned
	// weight so the prompt shows the strongest signal first.
	LearnedStyles []string
	// StyleVector is the unit vector blended into the query vector: the
	// explicit embedding merged with the learned embedding. It is nil when the
	// profile has no embedding yet, in which case matching is text-only.
	StyleVector []float32
	// LearnedVector is the unit learned embedding kept separately for
	// observability and prompt context.
	LearnedVector []float32
	// StyleWeight is the effective blend weight: style_blend_weight when the
	// request has no style target, style_blend_weight_with_request otherwise.
	StyleWeight float64
	// LearnedStyleWeight is the effective score weight of a learned-style
	// match, read from personalization.learned_style_weight and already
	// defaulted by the builder.
	LearnedStyleWeight float64
	// HasRequestStyle records whether the request itself declares a style. If
	// so, the request wins and the profile only contributes softly.
	HasRequestStyle bool
}

// HasStyleSignals reports whether the context carries at least one style label
// to apply. Behaviour gates on this rather than Active so the context stays
// usable when it is constructed directly (tests, future callers).
func (pc PersonalizationContext) HasStyleSignals() bool {
	return len(pc.ExplicitStyles) > 0 || len(pc.LearnedStyles) > 0 || len(pc.ExcludedStyles) > 0
}

// RecommendationRetrievalQuery đóng gói các tham số tinh chỉnh dùng để truy vấn tìm kiếm các món đồ ứng viên từ cơ sở dữ liệu.
type RecommendationRetrievalQuery struct {
	SemanticQuery  string
	LexicalTerms   []RetrievalTerm
	ExcludedTerms  []RetrievalTerm
	HardFilters    repositories.RecommendationHardFilters
	RewriterSource string
}

// RetrievalTerm đại diện cho một từ khóa riêng lẻ dùng cho việc so khớp tìm kiếm văn bản (lexical search).
type RetrievalTerm struct {
	Value        string
	Source       RetrievalTermSource
	TargetFields []string
	SourceReason string
}

// KeywordMatch đại diện cho một kết quả khớp từ khóa trong từ điển cục bộ trong pha phân tích ngôn ngữ tự nhiên (NLP parser).
type KeywordMatch struct {
	Category string
	Value    string
	Keyword  string
	Start    int
	End      int
	Source   RetrievalTermSource
}

// LlmOutfitResponse đại diện cho cấu trúc JSON của bộ trang phục gợi ý do mô hình AI trả về.
type LlmOutfitResponse struct {
	Title       string `json:"title"`
	Explanation string `json:"explanation"`
	Items       []struct {
		Role           string   `json:"role"`
		PrimaryID      string   `json:"primary_id"`
		AlternativeIDs []string `json:"alternative_ids"`
	} `json:"items"`
}
