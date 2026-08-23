package dto

import (
	wardrobeDto "smart-wardrobe-be/internal/modules/wardrobe/application/dto"

	"github.com/google/uuid"
)

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
	Occasion *string `json:"occasion" example:"casual"`
	// Phong cách hướng tới (Gợi ý: minimalist, vintage, streetwear, preppy, sporty, elegant, hoặc nhập phong cách tùy ý)
	StyleTarget *string `json:"styleTarget" example:"minimalist"`
	// Mùa phối đồ
	// @enums spring,summer,autumn,winter,all
	Season *Season `json:"season" swaggertype:"string" enums:"spring,summer,autumn,winter,all"`
	// Thời tiết hiện tại (Gợi ý: hot, cold, warm, cool, rainy, hoặc nhập thời tiết cụ thể)
	Weather *string `json:"weather" example:"warm"`
	// Ghi chú thêm bằng tay (free text)
	Details *string `json:"details" binding:"omitempty,nfcmax=1000"`
	// Tông màu phối đồ (Gợi ý: light, dark, pastel, earthy, neon... hoặc nhập tông màu tùy ý)
	ColorTone *string `json:"colorTone" example:"light"`
	// Cho phép phối đồ của brand (tỷ lệ tối đa 30%)
	IncludeBrandItems *bool `json:"include_brand_items,omitempty" example:"true"`
}

type RecommendedFashionItemBriefRes struct {
	ID       uuid.UUID                     `json:"id"`
	Category *wardrobeDto.CategoryBriefRes `json:"category,omitempty"`
	ImageUrl string                        `json:"imageUrl,omitempty"`
	Color    *string                       `json:"color,omitempty"`
	ColorHex *string                       `json:"colorHex,omitempty"`
	Style    *string                       `json:"style,omitempty"`
}

type RecommendedItemRes struct {
	ID          uuid.UUID                       `json:"id"`
	ItemContext string                          `json:"itemContext,omitempty"`
	BrandItem   *wardrobeDto.BrandItemBriefRes  `json:"brandItem,omitempty"`
	FashionItem *RecommendedFashionItemBriefRes `json:"fashionItem,omitempty"`
}

type RecommendedOutfitRes struct {
	Title          string                  `json:"title"`
	Explanation    string                  `json:"explanation"`
	Items          []*RecommendedItemGroup `json:"items"`
	IsFallback     bool                    `json:"isFallback"`
	RemainingQuota int                     `json:"remainingQuota"`
}

type RecommendedItemGroup struct {
	Role         string                `json:"role"`
	Primary      *RecommendedItemRes   `json:"primary"`
	Alternatives []*RecommendedItemRes `json:"alternatives"`
}

type ParsedIntent struct {
	SemanticQuery       string   `json:"semanticQuery"`
	LexicalTerms        []string `json:"lexicalTerms"`
	Occasion            []string `json:"occasion"`
	StyleTarget         []string `json:"styleTarget"`
	ColorTone           []string `json:"colorTone"`
	PositiveConstraints []string `json:"positiveConstraints"`
	NegativeConstraints []string `json:"negativeConstraints"`
	ExcludedStyles      []string `json:"excludedStyles"`
	ExcludedColorTones  []string `json:"excludedColorTones"`
	ExcludedWeather     []string `json:"excludedWeather"`
}
