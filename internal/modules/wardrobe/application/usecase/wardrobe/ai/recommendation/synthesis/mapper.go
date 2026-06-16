// Package synthesis implements response synthesis, LLM prompt assembly, response parsing, and validation.
package synthesis

import (
	"strings"

	"github.com/google/uuid"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/mapper"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/types"
	"smart-wardrobe-be/internal/shared/domain/entities"
)

// MapLLMResponseToGroups matches LLM response ID/role suggestions back to actual candidate wardrobe items.
func MapLLMResponseToGroups(
	candidates []types.CandidateForPrompt,
	llmRes types.LlmOutfitResponse,
) []*dto.RecommendedItemGroup {
	candidateMap := map[uuid.UUID]*entities.WardrobeItem{}
	for _, candidate := range candidates {
		candidateMap[candidate.Item.ID] = candidate.Item
	}

	valid := make([]*dto.RecommendedItemGroup, 0)
	for _, group := range llmRes.Items {
		role := strings.ToLower(group.Role)
		primary := resolvePrimaryCandidate(candidateMap, candidates, role, group.PrimaryID)
		if primary == nil {
			continue
		}

		valid = append(valid, &dto.RecommendedItemGroup{
			Role:         role,
			Primary:      mapper.MapToWardrobeItemRes(primary),
			Alternatives: resolveAlternativeCandidates(candidateMap, candidates, role, primary.ID, group.AlternativeIDs),
		})
	}

	return valid
}

func resolvePrimaryCandidate(
	candidateMap map[uuid.UUID]*entities.WardrobeItem,
	candidates []types.CandidateForPrompt,
	role string,
	primaryID string,
) *entities.WardrobeItem {
	if id, err := uuid.Parse(primaryID); err == nil {
		if item := candidateMap[id]; item != nil {
			return item
		}
	}

	for _, candidate := range candidates {
		if candidate.Item.Category != nil && candidate.Item.Category.Slug == role {
			return candidate.Item
		}
	}

	return nil
}

func resolveAlternativeCandidates(
	candidateMap map[uuid.UUID]*entities.WardrobeItem,
	candidates []types.CandidateForPrompt,
	role string,
	primaryID uuid.UUID,
	alternativeIDs []string,
) []*dto.WardrobeItemRes {
	alternatives := make([]*dto.WardrobeItemRes, 0, 2)
	for _, altID := range alternativeIDs {
		altUUID, err := uuid.Parse(altID)
		if err != nil {
			continue
		}

		altItem := candidateMap[altUUID]
		if altItem == nil || altItem.ID == primaryID || altItem.Category == nil || altItem.Category.Slug != role {
			continue
		}

		alternatives = append(alternatives, mapper.MapToWardrobeItemRes(altItem))
		if len(alternatives) == 2 {
			return alternatives
		}
	}

	for _, candidate := range candidates {
		item := candidate.Item
		if item.ID == primaryID || item.Category == nil || item.Category.Slug != role {
			continue
		}

		exists := false
		for _, alternative := range alternatives {
			if alternative.ID == item.ID {
				exists = true
				break
			}
		}
		if exists {
			continue
		}

		alternatives = append(alternatives, mapper.MapToWardrobeItemRes(item))
		if len(alternatives) == 2 {
			break
		}
	}

	return alternatives
}
