package sliceutils

import (
	"math"
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

// NormalizeVector returns the L2-normalised (unit length) copy of a vector.
// This is the single home of vector normalisation; embedding helpers must call
// it rather than define a second copy.
//
// An empty vector is returned unchanged. An all-zero vector is returned as a
// copy so the caller can mutate the result safely.
func NormalizeVector(vec []float32) []float32 {
	if len(vec) == 0 {
		return vec
	}

	var sumSquares float64
	for _, val := range vec {
		sumSquares += float64(val) * float64(val)
	}
	if sumSquares == 0 {
		return append([]float32(nil), vec...)
	}

	norm := math.Sqrt(sumSquares)
	res := make([]float32, len(vec))
	for i, val := range vec {
		res[i] = float32(float64(val) / norm)
	}
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

func IndexBy[T any, K comparable](collection []T, keyExtractor func(T) K) map[K]T {
	result := make(map[K]T, len(collection))
	for _, item := range collection {
		key := keyExtractor(item)
		result[key] = item
	}
	return result
}
