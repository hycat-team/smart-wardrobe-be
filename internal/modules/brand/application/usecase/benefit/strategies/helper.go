package strategies

import (
	"encoding/json"

	"smart-wardrobe-be/internal/modules/brand/domain/entities"
)

func parseValidDurationDays(doc entities.JSONDocument) int {
	if len(doc) == 0 {
		return 0
	}
	var config map[string]any
	if err := json.Unmarshal(doc, &config); err != nil {
		return 0
	}
	if val, ok := config["validDurationDays"]; ok {
		if floatVal, ok := val.(float64); ok {
			return int(floatVal)
		}
	}
	return 0
}
