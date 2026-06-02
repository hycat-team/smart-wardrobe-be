package errorutils

import (
	"errors"
	"net/http"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
)

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
