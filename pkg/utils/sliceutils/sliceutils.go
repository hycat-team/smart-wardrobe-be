package sliceutils

import (
	"slices"
	"strings"

	"github.com/google/uuid"
)

// AdjustVectorLength resizes a float32 vector to the target length by truncation or padding.
func AdjustVectorLength(vec []float32, targetLength int) []float32 {
	if len(vec) == targetLength {
		return vec
	}
	if len(vec) > targetLength {
		return vec[:targetLength]
	}
	res := make([]float32, targetLength)
	copy(res, vec)
	return res
}

// UniqueAndSortStrings returns a deduplicated and sorted slice of strings.
func UniqueAndSortStrings(slice []string) []string {
	keys := make(map[string]bool, len(slice))
	var list []string
	for _, entry := range slice {
		if !keys[entry] {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	slices.Sort(list)
	return list
}

// AppendUniqueStringCaseInsensitive appends a value to a slice if it is not already present (case-insensitive).
func AppendUniqueStringCaseInsensitive(slice []string, val string) []string {
	for _, s := range slice {
		if strings.EqualFold(s, val) {
			return slice
		}
	}
	return append(slice, val)
}

func UniqueUUIDs(ids []uuid.UUID) []uuid.UUID {
	if len(ids) == 0 {
		return nil
	}

	seen := make(map[uuid.UUID]struct{}, len(ids))
	result := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}

	return result
}
