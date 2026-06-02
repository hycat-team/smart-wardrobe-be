package errorutils

import (
	"errors"
	"net/http"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
	"strings"
)

// FilterStackTrace loại bỏ các dòng log của thư viện ngoài, giữ lại dòng code của dự án
func FilterStackTraceArray(rawStack string) []string {
	lines := strings.Split(rawStack, "\n")
	var filteredLines []string

	if len(lines) > 0 && strings.HasPrefix(lines[0], "goroutine") {
		filteredLines = append(filteredLines, lines[0])
	}

	for i := 1; i < len(lines)-1; i += 2 {
		functionLine := lines[i]
		fileLine := lines[i+1]

		if strings.Contains(functionLine, "smart-wardrobe-be") || strings.Contains(fileLine, "smart-wardrobe-be") {
			// Thu gọn khoảng trắng và ký tự tab để hiển thị JSON sạch hơn
			filteredLines = append(filteredLines, strings.TrimSpace(functionLine))
			filteredLines = append(filteredLines, strings.TrimSpace(fileLine))
		}
	}

	if len(filteredLines) <= 1 {
		return []string{"No project-level stack trace available."}
	}

	return filteredLines
}

func MapErrorToProblem(err error) (int, string, string) {
	if err == nil {
		return http.StatusOK, "", ""
	}

	for internalErr, info := range errorcode.InternalErrorMap {
		if errors.Is(err, internalErr) {
			return info.Status, info.Title, info.Detail
		}
	}

	var appErr *errorcode.ErrorResponse
	if errors.As(err, &appErr) {
		return appErr.Status, appErr.Title, appErr.Detail
	}

	return http.StatusInternalServerError, "Lỗi hệ thống", err.Error()
}

func WrapError(err error, fallbackMsg ...string) error {
	if err == nil {
		return nil
	}

	for internalErr := range errorcode.InternalErrorMap {
		if errors.Is(err, internalErr) {
			return err
		}
	}

	var appErr *errorcode.ErrorResponse
	if errors.As(err, &appErr) {
		return err
	}

	msg := "Đã xảy ra lỗi hệ thống"
	if len(fallbackMsg) > 0 && fallbackMsg[0] != "" {
		msg = fallbackMsg[0]
	} else {
		msg = err.Error()
	}

	return errorcode.NewInternalError(msg)
}
