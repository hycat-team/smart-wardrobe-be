package recommendation

import "smart-wardrobe-be/internal/shared/domain/entities"

type CandidateForPrompt struct {
	Item *entities.WardrobeItem
	Tags []string
}

type RankedCandidate struct {
	Item  *entities.WardrobeItem
	Score float64
	Tags  []string
}

type llmOutfitResponse struct {
	Title       string `json:"title"`
	Explanation string `json:"explanation"`
	Items       []struct {
		Role           string   `json:"role"`
		PrimaryID      string   `json:"primary_id"`
		AlternativeIDs []string `json:"alternative_ids"`
	} `json:"items"`
}
