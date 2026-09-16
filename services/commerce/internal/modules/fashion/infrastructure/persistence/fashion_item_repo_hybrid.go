package persistence

import (
	"context"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/repositories"
	shared_entities "smart-wardrobe-be/shared/domain/entities"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetHybridCandidates thực hiện tìm kiếm hỗn hợp nhiều giai đoạn (hybrid search) kết hợp giữa:
// - Tìm kiếm ngữ nghĩa bằng khoảng cách Cosine (Cosine Similarity) trên vector embedding (sử dụng pgvector).
// - Tìm kiếm từ khóa (Lexical Search) bằng chỉ mục GIN của PostgreSQL tsquery trên các thuộc tính văn bản.
// - Các tùy chọn tìm kiếm dự phòng (fallback) nếu một trong hai phương thức tìm kiếm trên bị trống hoặc lỗi.
//
// Signature hiện tại nhận danh sách fashion_item_ids (lấy từ wardrobe module qua contract) và query trực tiếp
// trên bảng fashion_items (module sở hữu) — không JOIN bảng wardrobe_items. Kết quả trả về ID = fashion_item_id
// kèm các điểm số; caller stitch dữ liệu wardrobe (ID/UserID/Price/Status) qua wardrobe contract.
func (r *FashionItemRepository) GetHybridCandidates(
	ctx context.Context,
	fashionItemIDs []uuid.UUID,
	semanticVector []float32,
	lexicalTerms []string,
	excludedTerms []string,
	hardFilters repositories.RecommendationHardFilters,
	limit int,
	rrfK int,
) ([]repositories.HybridCandidate, error) {
	var records []hybridCandidateRecord
	db := r.GetDB(ctx).Table("fashion_items")

	if len(fashionItemIDs) > 0 {
		db = db.Where("fashion_items.id IN ?", fashionItemIDs)
	}

	if len(hardFilters.Seasonality) > 0 {
		seasonSQL, seasonArgs := buildRecommendationSeasonalityConditionWithJoin(hardFilters.Seasonality)
		if seasonSQL != "" {
			db = db.Where(seasonSQL, seasonArgs...)
		}
	}
	if len(hardFilters.CategorySlugs) > 0 {
		db = db.Where("fashion_items.category_id IN (SELECT id FROM categories WHERE slug IN ?)", hardFilters.CategorySlugs)
	}
	excludedQuerySQL, excludedQueryArgs := buildRecommendationTSQuerySubquery(excludedTerms)
	if excludedQuerySQL != "" {
		excludedArgs := append([]any{}, excludedQueryArgs...)
		db = db.Where(gorm.Expr("("+excludedQuerySQL+") = false", excludedArgs...))
	}

	hasVector := len(semanticVector) > 0
	var vec shared_entities.Vector
	if hasVector {
		vec = shared_entities.Vector(semanticVector)
	}

	includeQuerySQL, includeQueryArgs := buildRecommendationTSQuerySubquery(lexicalTerms)
	hasKeywords := includeQuerySQL != ""

	if hasVector && hasKeywords {
		dbWithJoin := db.Session(&gorm.Session{})
		vectorScoreSQL := "(1.0 - (fashion_items.embedding <=> ?))"
		lexicalScoreSQL := "ts_rank_cd(" + recommendationSearchDocumentSQLWithJoin() + ", (" + includeQuerySQL + "))"

		vectorQuery := dbWithJoin.
			Select("fashion_items.id, "+vectorScoreSQL+" AS vector_score, 0.0 AS lexical_score", vec).
			Where("fashion_items.embedding IS NOT NULL").
			Order(gorm.Expr("fashion_items.embedding <=> ?", vec)).
			Limit(limit)

		lexicalQuery := db.Session(&gorm.Session{}).
			Select("fashion_items.id, 0.0 AS vector_score, "+lexicalScoreSQL+" AS lexical_score", includeQueryArgs...).
			Where(gorm.Expr(recommendationSearchDocumentSQLWithJoin()+" @@ ("+includeQuerySQL+")", includeQueryArgs...)).
			Order(gorm.Expr(lexicalScoreSQL+" DESC", includeQueryArgs...)).
			Limit(limit)

		var vectorRecords []hybridCandidateRecord
		if err := vectorQuery.Find(&vectorRecords).Error; err != nil {
			return nil, err
		}

		var lexicalRecords []hybridCandidateRecord
		if err := lexicalQuery.Find(&lexicalRecords).Error; err != nil {
			return nil, err
		}

		mergedRecords := mergeHybridCandidateRecordsByRRF(vectorRecords, lexicalRecords, rrfK, limit)
		return r.buildHybridCandidates(mergedRecords, "", nil)
	} else if hasVector {
		vectorScoreSQL := "(1.0 - (fashion_items.embedding <=> ?))"
		err := db.
			Select("fashion_items.id, "+vectorScoreSQL+" AS vector_score, "+vectorScoreSQL+" AS retrieval_score, 0.0 AS lexical_score", vec, vec).
			Where("fashion_items.embedding IS NOT NULL").
			Order(gorm.Expr("fashion_items.embedding <=> ?", vec)).
			Order("fashion_items.id ASC").
			Limit(limit).
			Find(&records).Error
		return r.buildHybridCandidates(records, repositories.HybridCandidateSourceVector, err)
	} else if hasKeywords {
		lexicalScoreSQL := "ts_rank_cd(" + recommendationSearchDocumentSQLWithJoin() + ", (" + includeQuerySQL + "))"
		whereArgs := append([]any{}, includeQueryArgs...)
		selectArgs := append([]any{}, includeQueryArgs...)
		selectArgs = append(selectArgs, includeQueryArgs...)
		orderArgs := append([]any{}, includeQueryArgs...)
		err := db.
			Select("fashion_items.id, "+lexicalScoreSQL+" AS lexical_score, "+lexicalScoreSQL+" AS retrieval_score, 0.0 AS vector_score", selectArgs...).
			Where(gorm.Expr(recommendationSearchDocumentSQLWithJoin()+" @@ ("+includeQuerySQL+")", whereArgs...)).
			Order(gorm.Expr(lexicalScoreSQL+" DESC", orderArgs...)).
			Order("fashion_items.id ASC").
			Limit(limit).
			Find(&records).Error
		return r.buildHybridCandidates(records, repositories.HybridCandidateSourceLexical, err)
	}

	err := db.Order("fashion_items.created_at DESC").
		Order("fashion_items.id ASC").
		Limit(limit).
		Find(&records).Error
	return r.buildHybridCandidates(records, repositories.HybridCandidateSourceFallback, err)
}
