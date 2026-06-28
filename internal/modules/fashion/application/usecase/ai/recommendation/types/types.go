// Package types defines shared data structures, interfaces, and constants for the outfit recommendation workflow.
package types

import (
	"context"

	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/outfititemcontext"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

const (
	// CandidateSourceRetrieval chỉ ra ứng viên được lấy thông qua tìm kiếm hỗn hợp (hybrid search).
	CandidateSourceRetrieval = repositories.HybridCandidateSourceHybrid
	// CandidateSourceFallback chỉ ra ứng viên được lấy thông qua tìm kiếm dự phòng (fallback search).
	CandidateSourceFallback = repositories.HybridCandidateSourceFallback
	// CandidateSourceStrictFallback đại diện cho mức độ lọc dự phòng nghiêm ngặt (strict fallback).
	CandidateSourceStrictFallback = "strict-fallback"
	// CandidateSourceRelaxedFallback đại diện cho mức độ lọc dự phòng nới lỏng vừa phải (relaxed fallback).
	CandidateSourceRelaxedFallback = "relaxed-fallback"
	// CandidateSourceGeneralFallback đại diện cho mức độ lọc dự phòng tổng quan/rộng nhất (general fallback).
	CandidateSourceGeneralFallback = "general-fallback"

	// RetrievalTermSourceDictionary chỉ ra từ khóa truy vấn được trích xuất từ các từ điển cục bộ.
	RetrievalTermSourceDictionary = "dictionary"
	// RetrievalTermSourceRaw chỉ ra từ khóa truy vấn thô được trích xuất trực tiếp từ câu truy vấn của người dùng.
	RetrievalTermSourceRaw = "raw"
	// RetrievalTermSourceTaxonomy chỉ ra từ khóa truy vấn được mở rộng từ hệ thống phân loại thời trang (taxonomy).
	RetrievalTermSourceTaxonomy = "taxonomy"
)

// CandidateForPrompt đại diện cho một món đồ ứng viên đã chuẩn bị đầy đủ thông tin (kèm tag thời trang) để đưa vào prompt gửi cho LLM.
type CandidateForPrompt struct {
	Item        *entities.WardrobeItem
	Tags        []string
	ItemContext outfititemcontext.OutfitItemContext
	BrandItem   *entities.BrandItem
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
	Item            *entities.WardrobeItem
	Source          string
	VectorScore     float64
	LexicalScore    float64
	RetrievalScore  float64
	RetrievalRank   int
	RetrievalSource string
	ItemContext     outfititemcontext.OutfitItemContext
	BrandItem       *entities.BrandItem
}

// RankedCandidate đại diện cho một món đồ ứng viên đã được tính điểm và sắp xếp thứ hạng hoàn chỉnh.
type RankedCandidate struct {
	Item          *entities.WardrobeItem
	Score         float64
	Tags          []string
	Source        string
	RetrievalRank int
	ItemContext   outfititemcontext.OutfitItemContext
	BrandItem     *entities.BrandItem
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
	Source       string
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
	Source   string
}

// RecommendationQueryRewriter định nghĩa giao diện (contract) cho các bộ viết lại truy vấn (query rewriter) trong kiến trúc RAG.
type RecommendationQueryRewriter interface {
	// Rewrite viết lại hoặc làm phong phú thêm ý định của người dùng thành cấu trúc truy vấn nâng cao [RecommendationRetrievalQuery].
	Rewrite(ctx context.Context, intent dto.ParsedIntent) (RecommendationRetrievalQuery, error)
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
