package dto

import "github.com/google/uuid"

// Season represents the fashion seasonality
type Season string

const (
	SeasonSpring Season = "spring"
	SeasonSummer Season = "summer"
	SeasonAutumn Season = "autumn"
	SeasonWinter Season = "winter"
	SeasonAll    Season = "all"
)

type RecommendOutfitReq struct {
	// Dịp phối đồ (Gợi ý: casual, work, date, party, sport, hoặc nhập dịp tùy ý)
	Occasion    *string `json:"occasion" example:"casual"`
	// Phong cách hướng tới (Gợi ý: minimalist, vintage, streetwear, preppy, sporty, elegant, hoặc nhập phong cách tùy ý)
	StyleTarget *string `json:"styleTarget" example:"minimalist"`
	// Mùa phối đồ
	// @enums spring,summer,autumn,winter,all
	Season      *Season `json:"season" swaggertype:"string" enums:"spring,summer,autumn,winter,all"`
	// Thời tiết hiện tại (Gợi ý: hot, cold, warm, cool, rainy, hoặc nhập thời tiết cụ thể)
	Weather     *string `json:"weather" example:"warm"`
	// Ghi chú thêm bằng tay (free text)
	Details     *string `json:"details"`
	// Tông màu phối đồ (Gợi ý: light, dark, pastel, earthy, neon... hoặc nhập tông màu tùy ý)
	ColorTone   *string `json:"colorTone" example:"light"`
}

type RecommendedOutfitRes struct {
	Title          string                  `json:"title"`
	Explanation    string                  `json:"explanation"`
	Items          []*RecommendedItemGroup `json:"items"`
	IsFallback     bool                    `json:"isFallback"`
	RemainingQuota int                     `json:"remainingQuota"`
}

type RecommendedItemGroup struct {
	Role         string             `json:"role"`
	Primary      *WardrobeItemRes   `json:"primary"`
	Alternatives []*WardrobeItemRes `json:"alternatives"`
}

type PendingTransferRes struct {
	PostItemID uuid.UUID        `json:"postItemId"`
	Item       *WardrobeItemRes `json:"item"`
	SellerName string           `json:"sellerName"`
}

type ParsedIntent struct {
	SemanticQuery       string   `json:"semantic_query"`
	ExactKeywords       []string `json:"exact_keywords"`
	Occasion            string   `json:"occasion"`
	StyleTarget         []string `json:"style_target"`
	ColorTone           string   `json:"color_tone"`
	PositiveConstraints []string `json:"positive_constraints"`
	NegativeConstraints []string `json:"negative_constraints"`
}

