package errorutils

import (
	"net/http"
	"path/filepath"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"strings"
)

var ignoredStackPaths = []string{
	"smart-wardrobe-be/internal/shared/application/constants/apperror/",
	"/internal/shared/application/constants/apperror/",
}

var ignoredStackFunctions = []string{
	"smart-wardrobe-be/pkg/utils/validation.TranslateValidationError",
}

var noiseCandidateFunctions = []string{
	"smart-wardrobe-be/internal/api/routes.NewEngine.GlobalErrorHandler",
	"smart-wardrobe-be/internal/api/middleware.CORSMiddleware",
	"smart-wardrobe-be/internal/api/routes.NewEngine.GlobalTimeoutMiddleware",
	"smart-wardrobe-be/internal/api/routes.NewEngine.(*RateLimitMiddleware).Handle",
}

type stackFrame struct {
	value            string
	isNoiseCandidate bool
}

// FilterStackTraceArray drops framework/noise frames, deduplicates them,
// and returns one readable line per frame.
func FilterStackTraceArray(rawStack string) []string {
	lines := strings.Split(rawStack, "\n")
	var filteredLines []string
	seenFrames := make(map[string]struct{})
	var frames []stackFrame

	if len(lines) > 0 && strings.HasPrefix(lines[0], "goroutine") {
		filteredLines = append(filteredLines, strings.TrimSpace(lines[0]))
	}

	for i := 1; i < len(lines)-1; i += 2 {
		functionLine := strings.TrimSpace(lines[i])
		fileLine := strings.TrimSpace(lines[i+1])

		if !isProjectFrame(functionLine, fileLine) || shouldIgnoreFrame(functionLine, fileLine) {
			continue
		}

		frame := formatStackFrame(functionLine, fileLine)
		if frame == "" {
			continue
		}
		if _, exists := seenFrames[frame]; exists {
			continue
		}

		seenFrames[frame] = struct{}{}
		frames = append(frames, stackFrame{
			value:            frame,
			isNoiseCandidate: isNoiseCandidateFrame(functionLine),
		})
	}

	for _, frame := range preferredFrames(frames) {
		filteredLines = append(filteredLines, frame.value)
	}

	if len(filteredLines) <= 1 {
		return []string{"No project-level stack trace available."}
	}

	return filteredLines
}

func PrimaryStackFrame(rawStack string) string {
	filtered := FilterStackTraceArray(rawStack)
	if len(filtered) >= 2 {
		return filtered[1]
	}
	return ""
}

func preferredFrames(frames []stackFrame) []stackFrame {
	if len(frames) == 0 {
		return nil
	}

	hasNonNoise := false
	for _, frame := range frames {
		if !frame.isNoiseCandidate {
			hasNonNoise = true
			break
		}
	}

	if !hasNonNoise {
		return frames
	}

	result := make([]stackFrame, 0, len(frames))
	for _, frame := range frames {
		if !frame.isNoiseCandidate {
			result = append(result, frame)
		}
	}
	return result
}

func isProjectFrame(functionLine, fileLine string) bool {
	return strings.Contains(functionLine, "smart-wardrobe-be") || strings.Contains(fileLine, "smart-wardrobe-be")
}

func shouldIgnoreFrame(functionLine, fileLine string) bool {
	for _, ignored := range ignoredStackPaths {
		if strings.Contains(functionLine, ignored) || strings.Contains(fileLine, ignored) {
			return true
		}
	}

	for _, ignored := range ignoredStackFunctions {
		if strings.Contains(functionLine, ignored) {
			return true
		}
	}

	return false
}

func isNoiseCandidateFrame(functionLine string) bool {
	for _, candidate := range noiseCandidateFunctions {
		if strings.Contains(functionLine, candidate) {
			return true
		}
	}
	return false
}

func formatStackFrame(functionLine, fileLine string) string {
	functionName := compactFunctionName(functionLine)
	fileRef := compactFileRef(fileLine)

	if functionName == "" {
		return ""
	}
	if fileRef == "" {
		return functionName
	}

	return functionName + " @ " + fileRef
}

func compactFunctionName(functionLine string) string {
	functionLine = strings.TrimSpace(functionLine)
	lastOpen := strings.LastIndex(functionLine, "(")
	lastClose := strings.LastIndex(functionLine, ")")
	if lastOpen >= 0 && lastClose > lastOpen && lastClose == len(functionLine)-1 {
		return functionLine[:lastOpen]
	}
	return functionLine
}

func compactFileRef(fileLine string) string {
	fileLine = strings.TrimSpace(fileLine)
	fields := strings.Fields(fileLine)
	if len(fields) == 0 {
		return ""
	}

	pathWithLine := fields[0]
	lastColon := strings.LastIndex(pathWithLine, ":")
	if lastColon < 0 || lastColon == len(pathWithLine)-1 {
		return filepath.Base(pathWithLine)
	}

	filePath := pathWithLine[:lastColon]
	lineNo := pathWithLine[lastColon+1:]
	return filepath.Base(filePath) + ":" + lineNo
}

func ToAppError(err error, fallbackMsg ...string) *apperror.Error {
	if err == nil {
		return nil
	}

	if len(fallbackMsg) > 0 {
		if wrapped, ok := apperror.Wrap(err, fallbackMsg...).(*apperror.Error); ok {
			return wrapped
		}
	}

	return apperror.From(err)
}

func MapErrorToProblem(err error) (int, string, string) {
	if err == nil {
		return http.StatusOK, "", ""
	}

	if appErr := ToAppError(err); appErr != nil {
		return appErr.Status, appErr.Title, appErr.Detail
	}

	return http.StatusInternalServerError, "Lỗi hệ thống", err.Error()
}

func WrapError(err error, fallbackMsg ...string) error {
	return apperror.Wrap(err, fallbackMsg...)
}
