package dto

type FashionMetadataResult struct {
	Color       string `json:"color"`
	Style       string `json:"style"`
	Material    string `json:"material"`
	Pattern     string `json:"pattern"`
	Fit         string `json:"fit"`
	Seasonality string `json:"seasonality"`
	Description string `json:"description"`
}
