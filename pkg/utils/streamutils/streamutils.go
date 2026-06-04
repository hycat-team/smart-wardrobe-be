package streamutils

import "fmt"

// SplitForStream chia chuỗi thành các chunk nhỏ để stream
func SplitForStream(content string, chunkSize int) []string {
	if chunkSize <= 0 || len(content) <= chunkSize {
		return []string{content}
	}

	result := make([]string, 0, (len(content)/chunkSize)+1)
	for start := 0; start < len(content); start += chunkSize {
		end := start + chunkSize
		if end > len(content) {
			end = len(content)
		}
		result = append(result, content[start:end])
	}
	return result
}

// SanitizeSSEData escape chuỗi an toàn cho SSE data format
func SanitizeSSEData(content string) string {
	return fmt.Sprintf("%q", content)
}
