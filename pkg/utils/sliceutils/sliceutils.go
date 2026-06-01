package sliceutils

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
