package persistence

import (
	"context"

	"smart-wardrobe-be/internal/modules/wardrobe/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/entities"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetHybridCandidates thực hiện tìm kiếm hỗn hợp nhiều giai đoạn (hybrid search) kết hợp giữa:
// - Tìm kiếm ngữ nghĩa bằng khoảng cách Cosine (Cosine Similarity) trên vector embedding (sử dụng pgvector).
// - Tìm kiếm từ khóa (Lexical Search) bằng chỉ mục GIN của PostgreSQL tsquery trên các thuộc tính văn bản.
// - Các tùy chọn tìm kiếm dự phòng (fallback) nếu một trong hai phương thức tìm kiếm trên bị trống hoặc lỗi.
//
// Hành vi:
// 1. Khởi tạo câu lệnh SQL cơ sở lọc các wardrobe items đang hoạt động, thuộc sở hữu của userID và chưa bị xóa mềm.
// 2. Sử dụng subquery để lọc theo category slug nếu có trong hard filters.
// 3. Áp dụng bộ lọc seasonality (mùa vụ) cứng nếu có.
// 4. Áp dụng lọc loại trừ các từ khóa thô/mở rộng trong [excludedTerms].
// 5. Nếu có cả vector ngữ nghĩa và từ khóa:
//   - a. Truy vấn top ứng viên theo khoảng cách vector cosine.
//   - b. Truy vấn top ứng viên theo độ tương đồng lexical (ts_rank_cd).
//   - c. Hợp nhất và xếp hạng lại bằng thuật toán RRF qua [mergeHybridCandidateRecordsByRRF].
//
// 6. Nếu chỉ có vector ngữ nghĩa: truy vấn và sắp xếp hoàn toàn bằng khoảng cách vector cosine.
// 7. Nếu chỉ có từ khóa: truy vấn và sắp xếp hoàn toàn bằng điểm FTS ts_rank_cd.
// 8. Nếu không có cả hai: truy vấn dự phòng lấy các món đồ được tạo gần đây nhất.
// 9. Xây dựng danh sách ứng viên hoàn chỉnh qua helper [buildHybridCandidates].
func (r *WardrobeItemRepository) GetHybridCandidates(
	ctx context.Context,
	userID uuid.UUID,
	semanticVector entities.Vector,
	lexicalTerms []string,
	excludedTerms []string,
	hardFilters repositories.RecommendationHardFilters,
	limit int,
	rrfK int,
) ([]repositories.HybridCandidate, error) {
	var records []hybridCandidateRecord
	db := r.GetDB(ctx).Model(&entities.WardrobeItem{}).
		Where("wardrobe_items.user_id = ? AND wardrobe_items.status = ? AND wardrobe_items.is_deleted = ?",
			userID, wardrobestatus.InWardrobe, false)

	if len(hardFilters.Seasonality) > 0 {
		seasonSQL, seasonArgs := buildRecommendationSeasonalityConditionSubquery(hardFilters.Seasonality)
		if seasonSQL != "" {
			db = db.Where(seasonSQL, seasonArgs...)
		}
	}
	if len(hardFilters.CategorySlugs) > 0 {
		db = db.Where("wardrobe_items.fashion_item_id IN (SELECT id FROM fashion_items WHERE category_id IN (SELECT id FROM categories WHERE slug IN ?))", hardFilters.CategorySlugs)
	}
	excludedQuerySQL, excludedQueryArgs := buildRecommendationTSQuerySubquery(excludedTerms)
	if excludedQuerySQL != "" {
		excludedArgs := append([]any{}, excludedQueryArgs...)
		db = db.Where(gorm.Expr("("+excludedQuerySQL+") = false", excludedArgs...))
	}

	hasVector := len(semanticVector) > 0
	includeQuerySQL, includeQueryArgs := buildRecommendationTSQuerySubquery(lexicalTerms)
	hasKeywords := includeQuerySQL != ""

	if hasVector && hasKeywords {
		vectorScoreSQL := "(1.0 - ((SELECT embedding FROM fashion_items WHERE id = wardrobe_items.fashion_item_id) <=> ?))"
		lexicalScoreSQL := "ts_rank_cd(" + recommendationSearchDocumentSubquerySQL() + ", (" + includeQuerySQL + "))"
		var vectorRecords []hybridCandidateRecord
		if err := db.Session(&gorm.Session{}).
			Select("wardrobe_items.*, "+vectorScoreSQL+" AS vector_score", semanticVector).
			Where("(SELECT embedding FROM fashion_items WHERE id = wardrobe_items.fashion_item_id) IS NOT NULL").
			Order(gorm.Expr("(SELECT embedding FROM fashion_items WHERE id = wardrobe_items.fashion_item_id) <=> ?", semanticVector)).
			Order("wardrobe_items.id ASC").
			Limit(limit).
			Find(&vectorRecords).Error; err != nil {
			return nil, err
		}

		var lexicalRecords []hybridCandidateRecord
		whereArgs := append([]any{}, includeQueryArgs...)
		orderArgs := append([]any{}, includeQueryArgs...)
		if err := db.Session(&gorm.Session{}).
			Select("wardrobe_items.*, "+lexicalScoreSQL+" AS lexical_score", includeQueryArgs...).
			Where(gorm.Expr(recommendationSearchDocumentSubquerySQL()+" @@ ("+includeQuerySQL+")", whereArgs...)).
			Order(gorm.Expr(lexicalScoreSQL+" DESC", orderArgs...)).
			Order("wardrobe_items.id ASC").
			Limit(limit).
			Find(&lexicalRecords).Error; err != nil {
			return nil, err
		}

		mergedRecords := mergeHybridCandidateRecordsByRRF(vectorRecords, lexicalRecords, rrfK, limit)
		return r.buildHybridCandidates(ctx, mergedRecords, "", nil)
	} else if hasVector {
		vectorScoreSQL := "(1.0 - ((SELECT embedding FROM fashion_items WHERE id = wardrobe_items.fashion_item_id) <=> ?))"
		err := db.
			Select("wardrobe_items.*, "+vectorScoreSQL+" AS vector_score, "+vectorScoreSQL+" AS retrieval_score", semanticVector, semanticVector).
			Where("(SELECT embedding FROM fashion_items WHERE id = wardrobe_items.fashion_item_id) IS NOT NULL").
			Order(gorm.Expr("(SELECT embedding FROM fashion_items WHERE id = wardrobe_items.fashion_item_id) <=> ?", semanticVector)).
			Order("wardrobe_items.id ASC").
			Limit(limit).
			Find(&records).Error
		return r.buildHybridCandidates(ctx, records, repositories.HybridCandidateSourceVector, err)
	} else if hasKeywords {
		lexicalScoreSQL := "ts_rank_cd(" + recommendationSearchDocumentSubquerySQL() + ", (" + includeQuerySQL + "))"
		whereArgs := append([]any{}, includeQueryArgs...)
		selectArgs := append([]any{}, includeQueryArgs...)
		selectArgs = append(selectArgs, includeQueryArgs...)
		orderArgs := append([]any{}, includeQueryArgs...)
		err := db.
			Select("wardrobe_items.*, "+lexicalScoreSQL+" AS lexical_score, "+lexicalScoreSQL+" AS retrieval_score", selectArgs...).
			Where(gorm.Expr(recommendationSearchDocumentSubquerySQL()+" @@ ("+includeQuerySQL+")", whereArgs...)).
			Order(gorm.Expr(lexicalScoreSQL+" DESC", orderArgs...)).
			Order("wardrobe_items.id ASC").
			Limit(limit).
			Find(&records).Error
		return r.buildHybridCandidates(ctx, records, repositories.HybridCandidateSourceLexical, err)
	}

	err := db.Order("wardrobe_items.created_at DESC").
		Order("wardrobe_items.id ASC").
		Limit(limit).
		Find(&records).Error
	return r.buildHybridCandidates(ctx, records, repositories.HybridCandidateSourceFallback, err)
}
