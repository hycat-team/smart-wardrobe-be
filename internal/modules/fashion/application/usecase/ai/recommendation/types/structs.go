package types

import (
	brandDto "smart-wardrobe-be/internal/modules/brand/application/dto"
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	wardrobeDto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/outfititemcontext"
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
