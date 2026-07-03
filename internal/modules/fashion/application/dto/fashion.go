package dto

import (
	"encoding/json"
	"fmt"
	"strings"
)

type FashionMetadataResult struct {
	CategorySlug string `json:"category_slug"`
	Color        string `json:"color"`
	ColorHex     string `json:"color_hex"`
	Style        string `json:"style"`
	Material     string `json:"material"`
	Pattern      string `json:"pattern"`
	Fit          string `json:"fit"`
	Seasonality  string `json:"seasonality"`
	Description  string `json:"description"`
	IsSingleItem bool   `json:"is_single_item"`
	ReviewReason string `json:"review_reason"`
}

func (f *FashionMetadataResult) UnmarshalJSON(data []byte) error {
	type Alias FashionMetadataResult
	aux := struct {
		IsSingleItem any `json:"is_single_item"`
		*Alias
	}{
		Alias: (*Alias)(f),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	switch v := aux.IsSingleItem.(type) {
	case bool:
		f.IsSingleItem = v
	case string:
		switch strings.ToLower(v) {
		case "true", "1", "yes":
			f.IsSingleItem = true
		case "false", "0", "no", "":
			f.IsSingleItem = false
		default:
			return fmt.Errorf("invalid boolean string for is_single_item: %s", v)
		}
	case nil:
		f.IsSingleItem = false
	default:
		return fmt.Errorf("unexpected type for is_single_item: %T", v)
	}
	return nil
}

type AICategoryRef struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}
