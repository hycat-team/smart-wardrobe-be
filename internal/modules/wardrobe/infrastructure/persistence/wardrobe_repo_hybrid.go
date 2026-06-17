package persistence

import (
	"context"

	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/constants/wardrobestatus"
	"smart-wardrobe-be/internal/shared/domain/entities"

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
// 2. Joins bảng categories để lọc theo category slug nếu có trong hard filters.
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
	// Base query filtering active, non-deleted wardrobe items owned by the specified user
	db := r.GetDB(ctx).Model(&entities.WardrobeItem{}).
		Where("wardrobe_items.user_id = ? AND wardrobe_items.status = ? AND wardrobe_items.is_deleted = ?",
			userID, wardrobestatus.InWardrobe, false)
	db = db.Joins("LEFT JOIN categories recommendation_categories ON recommendation_categories.id = wardrobe_items.category_id")

	searchVectorSQL := recommendationSearchDocumentSQL(false)
	if len(hardFilters.Seasonality) > 0 {
		seasonSQL, seasonArgs := buildRecommendationSeasonalityCondition(hardFilters.Seasonality)
		if seasonSQL != "" {
			db = db.Where(seasonSQL, seasonArgs...)
		}
	}
	if len(hardFilters.CategorySlugs) > 0 {
		db = db.Where("recommendation_categories.slug IN ?", hardFilters.CategorySlugs)
	}
	excludedQuerySQL, excludedQueryArgs := buildRecommendationTSQueryOR(excludedTerms)
	if excludedQuerySQL != "" {
		excludedArgs := append([]any{}, excludedQueryArgs...)
		db = db.Where(gorm.Expr("("+searchVectorSQL+" @@ ("+excludedQuerySQL+")) = false", excludedArgs...))
	}

	hasVector := len(semanticVector) > 0
	includeQuerySQL, includeQueryArgs := buildRecommendationTSQueryOR(lexicalTerms)
	hasKeywords := includeQuerySQL != ""

	// Case 1: Both vector and keyword search parameters are available (RRF hybrid search)
	if hasVector && hasKeywords {
		vectorScoreSQL := "(1.0 - (wardrobe_items.embedding <=> ?))"
		lexicalScoreSQL := "ts_rank_cd(" + searchVectorSQL + ", (" + includeQuerySQL + "))"
		var vectorRecords []hybridCandidateRecord
		if err := db.Session(&gorm.Session{}).
			Select("wardrobe_items.*, "+vectorScoreSQL+" AS vector_score", semanticVector).
			Where("wardrobe_items.embedding IS NOT NULL").
			Order(gorm.Expr("wardrobe_items.embedding <=> ?", semanticVector)).
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
			Where(gorm.Expr(searchVectorSQL+" @@ ("+includeQuerySQL+")", whereArgs...)).
			Order(gorm.Expr(lexicalScoreSQL+" DESC", orderArgs...)).
			Order("wardrobe_items.id ASC").
			Limit(limit).
			Find(&lexicalRecords).Error; err != nil {
			return nil, err
		}

		mergedRecords := mergeHybridCandidateRecordsByRRF(vectorRecords, lexicalRecords, rrfK, limit)
		return r.buildHybridCandidates(ctx, mergedRecords, "", nil)
	} else if hasVector {
		// Case 2: Only vector search is available (pure semantic retrieval ordered by closest cosine distance)
		vectorScoreSQL := "(1.0 - (wardrobe_items.embedding <=> ?))"
		err := db.
			Select("wardrobe_items.*, "+vectorScoreSQL+" AS vector_score, "+vectorScoreSQL+" AS retrieval_score", semanticVector, semanticVector).
			Where("wardrobe_items.embedding IS NOT NULL").
			Order(gorm.Expr("wardrobe_items.embedding <=> ?", semanticVector)).
			Order("wardrobe_items.id ASC").
			Limit(limit).
			Find(&records).Error
		return r.buildHybridCandidates(ctx, records, repositories.HybridCandidateSourceVector, err)
	} else if hasKeywords {
		// Case 3: Only keywords are available (pure lexical retrieval utilizing GIN index matching with @@)
		lexicalScoreSQL := "ts_rank_cd(" + searchVectorSQL + ", (" + includeQuerySQL + "))"
		whereArgs := append([]any{}, includeQueryArgs...)
		selectArgs := append([]any{}, includeQueryArgs...)
		selectArgs = append(selectArgs, includeQueryArgs...)
		orderArgs := append([]any{}, includeQueryArgs...)
		err := db.
			Select("wardrobe_items.*, "+lexicalScoreSQL+" AS lexical_score, "+lexicalScoreSQL+" AS retrieval_score", selectArgs...).
			Where(gorm.Expr(searchVectorSQL+" @@ ("+includeQuerySQL+")", whereArgs...)).
			Order(gorm.Expr("ts_rank_cd("+searchVectorSQL+", ("+includeQuerySQL+")) DESC", orderArgs...)).
			Order("wardrobe_items.id ASC").
			Limit(limit).
			Find(&records).Error
		return r.buildHybridCandidates(ctx, records, repositories.HybridCandidateSourceLexical, err)
	}

	// Case 4: Fallback retrieval (default to most recently created items)
	err := db.Order("wardrobe_items.created_at DESC").
		Order("wardrobe_items.id ASC").
		Limit(limit).
		Find(&records).Error
	return r.buildHybridCandidates(ctx, records, repositories.HybridCandidateSourceFallback, err)
}
