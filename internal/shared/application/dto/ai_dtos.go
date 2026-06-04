package dto

type FashionMetadataResult struct {
	CategorySlug string `json:"category_slug"`
	Color        string `json:"color"`
	Style        string `json:"style"`
	Material     string `json:"material"`
	Pattern      string `json:"pattern"`
	Fit          string `json:"fit"`
	Seasonality  string `json:"seasonality"`
	Description  string `json:"description"`
}

type AICategoryRef struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type TextGenerationResult struct {
	Content string `json:"content"`
}
