package dto

import "time"

// UpdateStyleProfileReq is the partial-update body of PUT /me/style-profile.
// Every field is a pointer so that an absent field means "keep the stored
// value" while an explicitly empty array means "clear the list"
// (contracts/personalization-api.md §1.2).
type UpdateStyleProfileReq struct {
	// Danh sách phong cách yêu thích (mảng rỗng = xoá danh sách)
	Styles *[]string `json:"styles" binding:"omitempty,dive,nfcmax=50" label:"phong cách yêu thích" example:"minimalist,elegant"`
	// Danh sách phong cách muốn tránh (mảng rỗng = xoá danh sách)
	ExcludedStyles *[]string `json:"excludedStyles" binding:"omitempty,dive,nfcmax=50" label:"phong cách muốn tránh" example:"streetwear"`
	// Bật/tắt áp dụng hồ sơ gu vào gợi ý phối đồ
	Enabled *bool `json:"enabled" example:"true"`
}

// StyleProfileRes is the style profile returned by GET and PUT
// /me/style-profile. The embedding vector, its hash and the embedding model are
// internal and never serialized.
type StyleProfileRes struct {
	Styles                   []string `json:"styles"`
	ExcludedStyles           []string `json:"excludedStyles"`
	Enabled                  bool     `json:"enabled"`
	PersonalizationAvailable bool     `json:"personalizationAvailable"`
	HasEmbedding             bool     `json:"hasEmbedding"`
	MaxStyles                int      `json:"maxStyles"`
	// UpdatedAt is the row timestamp; nil when the user has never saved a
	// profile (a default profile is not persisted by GET).
	UpdatedAt *time.Time `json:"updatedAt"`
}

// StyleSuggestionItem is one suggested style label derived from the user's
// wardrobe during cold start (contracts/personalization-api.md §1.4).
type StyleSuggestionItem struct {
	// Giá trị phong cách đã chuẩn hoá
	Value string `json:"value" example:"minimalist"`
	// Số món đang hoạt động mang phong cách này
	ItemCount int `json:"itemCount" example:"12"`
	// Tỷ lệ itemCount / sampleSize, làm tròn 2 chữ số
	Ratio float64 `json:"ratio" example:"0.4"`
}

// StyleSuggestionRes is the cold-start suggestion payload. The informational
// message ("Chưa đủ dữ liệu...") belongs to the response envelope, not here.
type StyleSuggestionRes struct {
	Styles     []StyleSuggestionItem `json:"styles"`
	SampleSize int                   `json:"sampleSize" example:"30"`
}
