package dto

import (
	season "smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/constants/season"
	shared_dto "smart-wardrobe-be/shared/application/dto"

	"github.com/google/uuid"
)

type RecommendOutfitReq struct {
	// Dịp phối đồ (Gợi ý: casual, work, date, party, sport, hoặc nhập dịp tùy ý)
	Occasion *string `json:"occasion" example:"casual"`
	// Phong cách hướng tới (Gợi ý: minimalist, vintage, streetwear, preppy, sporty, elegant, hoặc nhập phong cách tùy ý)
	StyleTarget *string `json:"styleTarget" example:"minimalist"`
	// Mùa phối đồ
	// @enums spring,summer,autumn,winter,all
	Season *season.Season `json:"season" swaggertype:"string" enums:"spring,summer,autumn,winter,all"`
	// Thời tiết hiện tại (Gợi ý: hot, cold, warm, cool, rainy, hoặc nhập thời tiết cụ thể)
	Weather *string `json:"weather" example:"warm"`
	// Ghi chú thêm bằng tay (free text)
	Details *string `json:"details" binding:"omitempty,nfcmax=1000"`
	// Tông màu phối đồ (Gợi ý: light, dark, pastel, earthy, neon... hoặc nhập tông màu tùy ý)
	ColorTone *string `json:"colorTone" example:"light"`
	// Cho phép phối đồ của brand (tỷ lệ tối đa 30%)
	IncludeBrandItems *bool `json:"includeBrandItems,omitempty" example:"true"`
}

type RecommendedFashionItemBriefRes struct {
	ID       uuid.UUID         `json:"id"`
	Category *CategoryBriefRes `json:"category,omitempty"`
	ImageUrl string            `json:"imageUrl,omitempty"`
	Color    *string           `json:"color,omitempty"`
	ColorHex *string           `json:"colorHex,omitempty"`
	Style    *string           `json:"style,omitempty"`
}

type RecommendedItemRes struct {
	ID          uuid.UUID                       `json:"id"`
	ItemContext string                          `json:"itemContext,omitempty"`
	BrandItem   *shared_dto.BrandItemBriefRes   `json:"brandItem,omitempty"`
	FashionItem *RecommendedFashionItemBriefRes `json:"fashionItem,omitempty"`
}

type RecommendedOutfitRes struct {
	// RecommendationID identifies this recommendation for the feedback endpoint
	// (033 contracts §2.1). Always present.
	RecommendationID uuid.UUID               `json:"recommendationId"`
	Title            string                  `json:"title"`
	Explanation      string                  `json:"explanation"`
	Items            []*RecommendedItemGroup `json:"items"`
	IsFallback       bool                    `json:"isFallback"`
	RemainingQuota   int                     `json:"remainingQuota"`
	// Suggestions lists catalog items that fill a missing or low-quality outfit
	// role (033 US2, contracts §2.1). It is additive: suggestions are NOT part
	// of Items, are NOT counted as outfit items and do NOT consume quota.
	Suggestions []OutfitSuggestionRes `json:"suggestions,omitempty"`
}

// OutfitSuggestionRes is one catalog item proposed to fill a gap in the
// generated outfit (033 US2, contracts §2.1). The item is a system catalog
// fashion item the user does not own yet; saving it later clones it.
type OutfitSuggestionRes struct {
	Role        string                          `json:"role"`
	Reason      string                          `json:"reason"`
	FashionItem *RecommendedFashionItemBriefRes `json:"fashionItem"`
}

type RecommendedItemGroup struct {
	Role         string                `json:"role"`
	Primary      *RecommendedItemRes   `json:"primary"`
	Alternatives []*RecommendedItemRes `json:"alternatives"`
}

// SubmitRecommendationFeedbackReq is the body of
// POST /ai/outfit-recommendations/feedback (033 US6, contracts §2.2). The
// client only sends the id it received and one action; the fashion item ids
// and styles are read from the server-side snapshot and never trusted from the
// client (plan D8).
type SubmitRecommendationFeedbackReq struct {
	// Mã gợi ý nhận được từ response của endpoint gợi ý phối đồ
	RecommendationID uuid.UUID `json:"recommendationId" binding:"required" label:"mã gợi ý" example:"5b0c9f54-7d61-4f7a-9a4e-9f1f7f4f3a21"`
	// Hành động phản hồi đối với bộ đồ được gợi ý
	Action string `json:"action" binding:"required,oneof=like dislike saved" enums:"like,dislike,saved" label:"hành động phản hồi" example:"like"`
}

// SubmitRecommendationFeedbackRes acknowledges a feedback submission. The
// contract only promises `{"accepted": true}` (contracts §2.2).
type SubmitRecommendationFeedbackRes struct {
	Accepted bool `json:"accepted" example:"true"`
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
	// PersonalStyle, LearnedStyle and AvoidStyle come from the user's saved
	// profile (033 US1). They are soft preferences and never overwrite the
	// request-level StyleTarget/ExcludedStyles fields.
	PersonalStyle []string `json:"personalStyle,omitempty"`
	LearnedStyle  []string `json:"learnedStyle,omitempty"`
	AvoidStyle    []string `json:"avoidStyle,omitempty"`
	// LearnedStyleWeight is the score weight applied to LearnedStyle matches,
	// read from personalization.learned_style_weight (033 US1/T031). Zero means
	// "not configured", so the scorer keeps its built-in default.
	LearnedStyleWeight float64 `json:"learnedStyleWeight,omitempty"`
}
