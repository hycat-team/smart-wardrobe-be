// Package retrieval implements candidate retrieval, taxonomy term expansion, and lexical/semantic query rewriting.
package retrieval

import (
	"strings"

	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/types"
)

const (
	taxonomyGroupOccasion  = "occasion"
	taxonomyGroupStyle     = "style"
	taxonomyGroupWeather   = "weather"
	taxonomyGroupSeason    = "season"
	taxonomyGroupColorTone = "color-tone"
	taxonomyGroupCategory  = "category"
	taxonomyGroupExcluded  = "excluded"
)

type recommendationTaxonomyEntry struct {
	Term         string
	TargetFields []string
	SourceReason string
}

var recommendationTaxonomy = map[string]map[string][]recommendationTaxonomyEntry{
	taxonomyGroupOccasion: {
		"casual": taxonomyEntries("occasion:casual", []string{"style", "description"}, "casual", "street", "dao pho", "cafe"),
		"date":   taxonomyEntries("occasion:date", []string{"style", "description"}, "romantic", "elegant", "hen ho"),
		"party":  taxonomyEntries("occasion:party", []string{"style", "description"}, "party", "event", "formal", "elegant"),
		"sport":  taxonomyEntries("occasion:sport", []string{"style", "description"}, "sport", "sporty", "workout", "the thao"),
		"work":   taxonomyEntries("occasion:work", []string{"style", "description"}, "office", "formal", "business", "interview", "van phong", "cong so"),
	},
	taxonomyGroupStyle: {
		"minimalist": taxonomyEntries("style:minimalist", []string{"style", "description"}, "minimalist", "toi gian", "basic", "don gian"),
		"vintage":    taxonomyEntries("style:vintage", []string{"style", "description"}, "vintage", "retro", "classic", "co dien"),
		"streetwear": taxonomyEntries("style:streetwear", []string{"style", "description"}, "streetwear", "duong pho", "ca tinh"),
		"preppy":     taxonomyEntries("style:preppy", []string{"style", "description"}, "preppy", "hoc duong", "sinh vien"),
		"elegant":    taxonomyEntries("style:elegant", []string{"style", "description"}, "elegant", "thanh lich", "sang trong"),
	},
	taxonomyGroupWeather: {
		"cold":  taxonomyEntries("weather:cold", []string{"seasonality", "description", "material"}, "cold", "lanh", "ret", "dong", "mua dong", "winter", "len"),
		"cool":  taxonomyEntries("weather:cool", []string{"seasonality", "description"}, "cool", "mat", "thu", "mua thu", "autumn"),
		"hot":   taxonomyEntries("weather:hot", []string{"seasonality", "description"}, "hot", "nong", "he", "mua he", "summer"),
		"rainy": taxonomyEntries("weather:rainy", []string{"seasonality", "description"}, "rainy", "rain", "mua", "ao mua", "waterproof", "water resistant"),
	},
	taxonomyGroupSeason: {
		"spring": taxonomyEntries("season:spring", []string{"seasonality", "description"}, "spring", "xuan", "mua xuan"),
		"summer": taxonomyEntries("season:summer", []string{"seasonality", "description"}, "summer", "he", "mua he"),
		"autumn": taxonomyEntries("season:autumn", []string{"seasonality", "description"}, "autumn", "thu", "mua thu"),
		"winter": taxonomyEntries("season:winter", []string{"seasonality", "description"}, "winter", "dong", "mua dong", "lanh"),
	},
	taxonomyGroupColorTone: {
		"dark":   taxonomyEntries("color-tone:dark", []string{"color", "description"}, "black", "den", "gray", "grey", "xam", "dark", "tram"),
		"earthy": taxonomyEntries("color-tone:earthy", []string{"color", "description"}, "brown", "nau", "beige", "be", "olive", "reu", "earthy"),
		"light":  taxonomyEntries("color-tone:light", []string{"color", "description"}, "white", "trang", "cream", "kem", "pastel", "light"),
	},
	taxonomyGroupCategory: {
		"ao":       taxonomyEntries("category:ao", []string{"category.slug", "category.name"}, "ao"),
		"quan":     taxonomyEntries("category:quan", []string{"category.slug", "category.name"}, "quan"),
		"mu":       taxonomyEntries("category:mu", []string{"category.slug", "category.name"}, "mu"),
		"giay":     taxonomyEntries("category:giay", []string{"category.slug", "category.name"}, "giay"),
		"phu-kien": taxonomyEntries("category:phu-kien", []string{"category.slug", "category.name"}, "phu-kien", "phu kien"),
		"dam":      taxonomyEntries("category:dam", []string{"category.slug", "category.name"}, "dam", "đầm", "dress", "vay lien", "vay lien than"),
		"chan-vay": taxonomyEntries("category:chan-vay", []string{"category.slug", "category.name"}, "chan-vay", "chan vay", "chân váy", "skirt", "vay ngan", "vay midi"),
		"ao-khoac": taxonomyEntries("category:ao-khoac", []string{"category.slug", "category.name"}, "ao-khoac", "ao khoac"),
		"other":    taxonomyEntries("category:other", []string{"category.slug", "category.name"}, "other", "khac"),
	},
	taxonomyGroupExcluded: {
		"dark":   taxonomyEntries("avoid-color:dark", []string{"color"}, "black", "den", "gray", "grey", "xam", "dark", "tram"),
		"earthy": taxonomyEntries("avoid-color:earthy", []string{"color"}, "brown", "nau", "beige", "be", "olive", "reu", "earthy"),
		"light":  taxonomyEntries("avoid-color:light", []string{"color"}, "white", "trang", "cream", "kem", "pastel", "light"),
	},
}

var recommendationSeasonalityTerms = map[string]bool{
	"spring": true,
	"summer": true,
	"autumn": true,
	"winter": true,
}

// taxonomyEntries xây dựng các phần tử [recommendationTaxonomyEntry] từ danh sách từ khóa và lý do tương ứng.
func taxonomyEntries(reason string, targetFields []string, terms ...string) []recommendationTaxonomyEntry {
	entries := make([]recommendationTaxonomyEntry, 0, len(terms))
	for _, term := range terms {
		entries = append(entries, recommendationTaxonomyEntry{
			Term:         term,
			TargetFields: append([]string(nil), targetFields...),
			SourceReason: reason,
		})
	}
	return entries
}

// ExpandTaxonomyTerms ánh xạ các tham số lọc đơn giản (ví dụ: "work", "sport") sang các từ khóa chi tiết hơn được lưu trữ trong từ điển phân loại (taxonomy dictionary) để cải thiện độ phủ tìm kiếm (recall).
//
// Hành vi:
// 1. Lấy bảng cấu hình từ điển [recommendationTaxonomy] theo nhóm (group) được truyền vào (ví dụ: "occasion").
// 2. Với mỗi giá trị trong danh sách đầu vào ([values]), chuẩn hóa và tìm kiếm các phần tử tương ứng trong cấu hình từ điển.
// 3. Với mỗi phần tử tìm thấy, tạo và thêm một đối tượng [RetrievalTerm] mới với nguồn là `taxonomy`, kèm theo thông tin các trường mục tiêu và lý do mở rộng.
// 4. Trả về danh sách [RetrievalTerm] đã mở rộng.
//
// Đầu vào mẫu:
//
//	group: "style"
//	values: []string{"minimalist"}
//
// Đầu ra mẫu:
//
//	[]types.RetrievalTerm{
//	  {Value: "toi gian", Source: "taxonomy", TargetFields: []string{"style", "description"}, SourceReason: "style:minimalist"},
//	  ...
//	}
func ExpandTaxonomyTerms(group string, values []string) []types.RetrievalTerm {
	var expanded []types.RetrievalTerm
	config := recommendationTaxonomy[group]
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		for _, entry := range config[key] {
			expanded = append(expanded, types.RetrievalTerm{
				Value:        entry.Term,
				Source:       types.RetrievalTermSourceTaxonomy,
				TargetFields: append([]string(nil), entry.TargetFields...),
				SourceReason: entry.SourceReason,
			})
		}
	}
	return expanded
}

// ExpandTaxonomyTermValues mở rộng các tham số lọc thành một danh sách chuỗi thô các từ khóa phân loại tương đương (đã lọc trùng và sắp xếp).
func ExpandTaxonomyTermValues(group string, values []string) []string {
	entries := ExpandTaxonomyTerms(group, values)
	terms := make([]string, 0, len(entries))
	for _, entry := range entries {
		terms = append(terms, entry.Value)
	}
	return NormalizeTermSet(terms)
}
