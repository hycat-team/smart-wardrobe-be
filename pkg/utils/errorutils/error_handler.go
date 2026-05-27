package errorutils

import (
	"errors"
	"net/http"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
)

func MapErrorToProblem(err error) (int, string, string) {
	var appErr *errorcode.ErrorResponse
	if errors.As(err, &appErr) {
		return appErr.Status, appErr.Title, appErr.Detail
	}

	for internalErr, info := range errorcode.InternalErrorMap {
		if errors.Is(err, internalErr) {
			return info.Status, info.Title, info.Detail
		}
	}

	return http.StatusInternalServerError, "Lỗi hệ thống", err.Error()
}
