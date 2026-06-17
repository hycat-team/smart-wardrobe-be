package persistence

import (
	"context"
	"sort"
	"strings"

	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/modules/wardrobe/infrastructure/constants"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type hybridCandidateRecord struct {
	entities.WardrobeItem
	VectorScore     float64 `gorm:"column:vector_score"`
	LexicalScore    float64 `gorm:"column:lexical_score"`
	RetrievalScore  float64 `gorm:"column:retrieval_score"`
	RetrievalSource string
}

// mergeHybridCandidateRecordsByRRF hợp nhất hai tập bản ghi ứng viên (từ tìm kiếm vector và từ tìm kiếm từ khóa) và xếp hạng lại bằng thuật toán Reciprocal Rank Fusion (RRF).
//
// Hành vi:
//   - 1. Tính toán điểm RRF = 1 / (rrfK + rank) cho mỗi ứng viên từ tập kết quả tìm kiếm vector và lưu vào bảng ánh xạ.
//   - 2. Tiếp tục duyệt qua tập kết quả tìm kiếm từ khóa, tính toán điểm RRF tương tự.
//   - 3. Nếu ứng viên đã tồn tại trong bảng ánh xạ (tức khớp cả vector và lexical), cộng dồn điểm RRF và đánh dấu nguồn là `hybrid`.
//   - 4. Sắp xếp lại danh sách theo điểm RRF giảm dần. Nếu điểm bằng nhau, sắp xếp theo ID của món đồ tăng dần.
//   - 5. Cắt danh sách theo [limit] và trả về.
//
// Đầu vào mẫu:
//
//	vectorRecords: []hybridCandidateRecord{{WardrobeItem: entities.WardrobeItem{ID: uuid1}}}
//	lexicalRecords: []hybridCandidateRecord{{WardrobeItem: entities.WardrobeItem{ID: uuid1}}}
//	rrfK: 60
//	limit: 10
//
// Đầu ra mẫu:
//
//	[]hybridCandidateRecord (chứa uuid1 với RetrievalSource: "hybrid", RetrievalScore: 1/61 + 1/61)
func mergeHybridCandidateRecordsByRRF(
	vectorRecords []hybridCandidateRecord,
	lexicalRecords []hybridCandidateRecord,
	rrfK int,
	limit int,
) []hybridCandidateRecord {
	if rrfK <= 0 {
		rrfK = 30
	}

	mergedByID := map[uuid.UUID]*hybridCandidateRecord{}
	for index, record := range vectorRecords {
		record.RetrievalScore = 1 / float64(rrfK+index+1)
		record.RetrievalSource = repositories.HybridCandidateSourceVector
		mergedByID[record.ID] = &record
	}

	for index, record := range lexicalRecords {
		rrfScore := 1 / float64(rrfK+index+1)
		existing := mergedByID[record.ID]
		if existing == nil {
			record.RetrievalScore = rrfScore
			record.RetrievalSource = repositories.HybridCandidateSourceLexical
			mergedByID[record.ID] = &record
			continue
		}

		existing.LexicalScore = record.LexicalScore
		existing.RetrievalScore += rrfScore
		existing.RetrievalSource = repositories.HybridCandidateSourceHybrid
	}

	merged := make([]hybridCandidateRecord, 0, len(mergedByID))
	for _, record := range mergedByID {
		merged = append(merged, *record)
	}

	sort.Slice(merged, func(i, j int) bool {
		if merged[i].RetrievalScore == merged[j].RetrievalScore {
			return merged[i].ID.String() < merged[j].ID.String()
		}
		return merged[i].RetrievalScore > merged[j].RetrievalScore
	})

	if limit > 0 && len(merged) > limit {
		return merged[:limit]
	}
	return merged
}

// buildHybridCandidates thực hiện truy vấn chi tiết các mối quan hệ (preload categories, v.v.) và chuyển đổi danh sách bản ghi thô [hybridCandidateRecord] thành thực thể hoàn chỉnh [HybridCandidate] phục vụ cho nghiệp vụ.
func (r *WardrobeItemRepository) buildHybridCandidates(
	ctx context.Context,
	records []hybridCandidateRecord,
	source string,
	queryErr error,
) ([]repositories.HybridCandidate, error) {
	if queryErr != nil {
		return nil, queryErr
	}
	if len(records) == 0 {
		return nil, nil
	}

	ids := make([]uuid.UUID, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.ID)
	}

	items, err := r.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	itemByID := make(map[uuid.UUID]*entities.WardrobeItem, len(items))
	for _, item := range items {
		itemByID[item.ID] = item
	}

	candidates := make([]repositories.HybridCandidate, 0, len(records))
	for index, record := range records {
		item := itemByID[record.ID]
		if item == nil {
			item = &record.WardrobeItem
		}
		recordSource := source
		if recordSource == "" {
			recordSource = record.RetrievalSource
		}
		candidates = append(candidates, repositories.HybridCandidate{
			Item:            item,
			VectorScore:     record.VectorScore,
			LexicalScore:    record.LexicalScore,
			RetrievalScore:  record.RetrievalScore,
			RetrievalRank:   index + 1,
			RetrievalSource: recordSource,
		})
	}

	return candidates, nil
}

// buildRecommendationTSQueryOR sinh chuỗi truy vấn tsquery của PostgreSQL nối với nhau bằng toán tử "OR" (||) từ danh sách các từ khóa văn bản, đồng thời chuẩn hóa Lowercase và loại bỏ dấu tiếng Việt (unaccent).
func buildRecommendationTSQueryOR(terms []string) (string, []any) {
	var parts []string
	args := make([]any, 0, len(terms))
	seen := map[string]bool{}
	for _, term := range terms {
		term = strings.TrimSpace(strings.ToLower(term))
		if term == "" || seen[term] {
			continue
		}
		seen[term] = true
		parts = append(parts, "plainto_tsquery('simple', immutable_unaccent(lower(?)))")
		args = append(args, term)
	}
	if len(parts) == 0 {
		return "", nil
	}
	return strings.Join(parts, " || "), args
}

// buildRecommendationSeasonalityCondition xây dựng điều kiện SQL (SQL Where condition) để lọc các món đồ có mùa vụ (seasonality) phù hợp với mùa yêu cầu hoặc hỗ trợ tất cả các mùa (all-season).
func buildRecommendationSeasonalityCondition(seasons []string) (string, []any) {
	aliases := recommendationSeasonalityAliases(seasons)
	if len(aliases) == 0 {
		return "", nil
	}

	seasonalitySQL := "immutable_unaccent(lower(coalesce(wardrobe_items.seasonality, '')))"
	conditions := []string{
		"wardrobe_items.seasonality IS NULL",
		"btrim(wardrobe_items.seasonality) = ''",
	}
	args := make([]any, 0, len(aliases)+len(constants.RecommendationAllSeasonAliases))
	for _, alias := range aliases {
		conditions = append(conditions, seasonalitySQL+" LIKE ?")
		args = append(args, "%"+alias+"%")
	}
	for _, alias := range constants.RecommendationAllSeasonAliases {
		conditions = append(conditions, seasonalitySQL+" LIKE ?")
		args = append(args, "%"+alias+"%")
	}

	return "(" + strings.Join(conditions, " OR ") + ")", args
}

// recommendationSeasonalityAliases chuyển đổi danh sách các mùa thô thành tập hợp các từ khóa mùa tương đương/đồng nghĩa mở rộng (ví dụ: "he" -> "he", "summer", "mua he").
func recommendationSeasonalityAliases(seasons []string) []string {
	seen := map[string]bool{}
	var aliases []string
	for _, season := range seasons {
		season = strings.TrimSpace(strings.ToLower(season))
		if season == "" {
			continue
		}
		values := constants.RecommendationSeasonAliases[season]
		if len(values) == 0 {
			values = []string{season}
		}
		for _, value := range values {
			value = strings.TrimSpace(strings.ToLower(value))
			if value == "" || seen[value] {
				continue
			}
			seen[value] = true
			aliases = append(aliases, value)
		}
	}
	sort.Strings(aliases)
	return aliases
}

// recommendationSearchDocumentSQL định nghĩa cấu trúc tsvector của PostgreSQL bằng cách ghép nối tất cả các thuộc tính văn bản của tủ đồ (color, style, description, v.v.) để tìm kiếm toàn văn (FTS).
func recommendationSearchDocumentSQL(includeCategory bool) string {
	fields := []string{
		"wardrobe_items.color",
		"wardrobe_items.style",
		"wardrobe_items.material",
		"wardrobe_items.pattern",
		"wardrobe_items.fit",
		"wardrobe_items.seasonality",
		"wardrobe_items.description",
	}
	if includeCategory {
		fields = append([]string{
			"recommendation_categories.slug",
			"recommendation_categories.name",
		}, fields...)
	}

	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		parts = append(parts, "coalesce("+field+", '')")
	}

	return "to_tsvector('simple', immutable_unaccent(lower(" + strings.Join(parts, " || ' ' || ") + ")))"
}
