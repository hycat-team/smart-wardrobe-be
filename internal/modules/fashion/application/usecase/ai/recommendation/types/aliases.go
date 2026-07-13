package types

import (
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
)

type CandidateSource string
type RetrievalTermSource string

const (
	// CandidateSourceRetrieval chỉ ra ứng viên được lấy thông qua tìm kiếm hỗn hợp (hybrid search).
	CandidateSourceRetrieval CandidateSource = CandidateSource(repositories.HybridCandidateSourceHybrid)
	// CandidateSourceFallback chỉ ra ứng viên được lấy thông qua tìm kiếm dự phòng (fallback search).
	CandidateSourceFallback CandidateSource = CandidateSource(repositories.HybridCandidateSourceFallback)
	// CandidateSourceStrictFallback đại diện cho mức độ lọc dự phòng nghiêm ngặt (strict fallback).
	CandidateSourceStrictFallback CandidateSource = "strict-fallback"
	// CandidateSourceRelaxedFallback đại diện cho mức độ lọc dự phòng nới lỏng vừa phải (relaxed fallback).
	CandidateSourceRelaxedFallback CandidateSource = "relaxed-fallback"
	// CandidateSourceGeneralFallback đại diện cho mức độ lọc dự phòng tổng quan/rộng nhất (general fallback).
	CandidateSourceGeneralFallback CandidateSource = "general-fallback"

	// RetrievalTermSourceDictionary chỉ ra từ khóa truy vấn được trích xuất từ các từ điển cục bộ.
	RetrievalTermSourceDictionary RetrievalTermSource = "local-dictionary"
	// RetrievalTermSourceRaw chỉ ra từ khóa truy vấn thô được trích xuất trực tiếp từ câu truy vấn của người dùng.
	RetrievalTermSourceRaw RetrievalTermSource = "raw"
	// RetrievalTermSourceTaxonomy chỉ ra từ khóa truy vấn được mở rộng từ hệ thống phân loại thời trang (taxonomy).
	RetrievalTermSourceTaxonomy RetrievalTermSource = "taxonomy"
)
