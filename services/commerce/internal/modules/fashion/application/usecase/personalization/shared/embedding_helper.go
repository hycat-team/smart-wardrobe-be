package shared

import (
	"strings"

	"smart-wardrobe-be/shared/pkg/utils/sliceutils"
	"smart-wardrobe-be/shared/pkg/utils/stringutils"
)

// BlendVectors returns normalize((1-weight)·normalize(query) + weight·normalize(style)),
// the soft personalisation blend of design decision D4. Both inputs are
// normalised first and the blended result is normalised again, so the output is
// always a unit vector. Mismatched lengths (the stored columns are vector(768))
// are truncated/padded to the longer input before blending.
//
// An empty style vector returns the normalised query unchanged (personalisation
// degraded to text/vector-only), an empty query returns the normalised style
// vector, and two empty vectors return nil. Weight is clamped to [0, 1].
func BlendVectors(query, style []float32, weight float64) []float32 {
	queryVector := sliceutils.NormalizeVector(query)
	styleVector := sliceutils.NormalizeVector(style)

	if len(queryVector) == 0 && len(styleVector) == 0 {
		return nil
	}
	if len(styleVector) == 0 {
		return queryVector
	}
	if len(queryVector) == 0 {
		return styleVector
	}

	if weight < 0 {
		weight = 0
	} else if weight > 1 {
		weight = 1
	}

	targetLength := len(queryVector)
	if len(styleVector) > targetLength {
		targetLength = len(styleVector)
	}
	queryVector = sliceutils.AdjustVectorLength(queryVector, targetLength)
	styleVector = sliceutils.AdjustVectorLength(styleVector, targetLength)

	blended := make([]float32, targetLength)
	for i := range blended {
		blended[i] = float32((1-weight)*float64(queryVector[i]) + weight*float64(styleVector[i]))
	}
	return sliceutils.NormalizeVector(blended)
}

// MeanVectors returns the L2-normalised arithmetic mean of the non-empty input
// vectors, or nil when every vector is empty. Shorter vectors are padded to the
// longest one so a single odd dimension cannot make the result ragged.
func MeanVectors(vectors [][]float32) []float32 {
	nonEmpty := make([][]float32, 0, len(vectors))
	targetLength := 0
	for _, vector := range vectors {
		if len(vector) == 0 {
			continue
		}
		nonEmpty = append(nonEmpty, vector)
		if len(vector) > targetLength {
			targetLength = len(vector)
		}
	}
	if len(nonEmpty) == 0 {
		return nil
	}

	sum := make([]float64, targetLength)
	for _, vector := range nonEmpty {
		adjusted := sliceutils.AdjustVectorLength(vector, targetLength)
		for i, value := range adjusted {
			sum[i] += float64(value)
		}
	}

	mean := make([]float32, targetLength)
	for i := range mean {
		mean[i] = float32(sum[i] / float64(len(nonEmpty)))
	}
	return sliceutils.NormalizeVector(mean)
}

// HashStyleProfile returns a deterministic hash of everything that affects the
// embedded style text: the normalised style lists plus the embedding model. A
// profile whose hash is unchanged can skip the embedding provider call
// (design decision D3). Reordering the same labels does not change the hash.
func HashStyleProfile(styles, excludedStyles []string, embeddingModel string) string {
	text := BuildStyleEmbeddingText(styles, excludedStyles)
	return stringutils.HashSHA256(text + "|" + strings.TrimSpace(embeddingModel))
}
