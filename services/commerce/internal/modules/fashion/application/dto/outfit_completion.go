package dto

import (
	season "smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/constants/season"

	"github.com/google/uuid"
)

// CompleteOutfitReq is the body of POST /ai/outfit-completions (feature 033,
// US7; contracts/personalization-api.md §2.3).
//
// The client names 1-10 fashion items it already owns plus the optional outfit
// context. The endpoint only answers which roles the set is missing and which
// catalog items could fill them: it never runs the LLM, never synthesizes an
// outfit and never consumes the daily outfit quota (FR-024).
type CompleteOutfitReq struct {
	// Danh sách mã trang phục người dùng đã sở hữu (1-10 món)
	FashionItemIDs []uuid.UUID `json:"fashionItemIds" binding:"required,min=1,max=10" label:"danh sách trang phục" example:"3f1c2a9e-0e0b-4a52-9f7e-8b6a1a2c3d4e"`
	// Dịp phối đồ (Gợi ý: casual, work, date, party, sport, hoặc nhập dịp tùy ý)
	Occasion *string `json:"occasion" example:"work"`
	// Phong cách hướng tới (Gợi ý: minimalist, vintage, streetwear, preppy, sporty, elegant, hoặc nhập phong cách tùy ý)
	StyleTarget *string `json:"styleTarget" example:"minimalist"`
	// Mùa phối đồ
	// @enums spring,summer,autumn,winter,all
	Season *season.Season `json:"season" swaggertype:"string" enums:"spring,summer,autumn,winter,all"`
	// Thời tiết hiện tại (Gợi ý: hot, cold, warm, cool, rainy, hoặc nhập thời tiết cụ thể)
	Weather *string `json:"weather" example:"hot"`
	// Tông màu phối đồ (Gợi ý: light, dark, pastel, earthy, neon... hoặc nhập tông màu tùy ý)
	ColorTone *string `json:"colorTone" example:"light"`
	// Ghi chú thêm bằng tay (free text)
	Details *string `json:"details" binding:"omitempty,nfcmax=1000"`
}

// CompleteOutfitRes is the response of POST /ai/outfit-completions. Suggestions
// is always present: an empty array means the set already has a complete
// structure or no catalog item fits (FR-025). The entries reuse
// [OutfitSuggestionRes] so the client renders completions exactly like the
// additive suggestions of the recommendation response.
type CompleteOutfitRes struct {
	Suggestions []OutfitSuggestionRes `json:"suggestions"`
}
