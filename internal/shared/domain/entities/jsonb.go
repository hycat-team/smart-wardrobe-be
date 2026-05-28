package entities

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type bodyProfile struct {
	Height             float64 `json:"height"`
	Weight             float64 `json:"weight"`
	BodyType           string  `json:"bodyType"`
	FitPreference      string  `json:"fitPreference"`
	SkinTone           string  `json:"skinTone"`
	EstimatedBodyShape string  `json:"estimatedBodyShape"`
	RecommendedSize    string  `json:"recommendedSize"`
	StylingNotes       string  `json:"stylingNotes"`
}

func (b bodyProfile) Value() (driver.Value, error) {
	return json.Marshal(b)
}

func (b *bodyProfile) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}
	return json.Unmarshal(bytes, b)
}
