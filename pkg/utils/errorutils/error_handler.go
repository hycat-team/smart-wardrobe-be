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

// WrapError checks if the input error is already a system *errorcode.ErrorResponse.
// If it is, it returns the error as is.
// If not, it wraps the error in a NewInternalError.
func WrapError(err error, fallbackMsg ...string) error {
	if err == nil {
		return nil
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
