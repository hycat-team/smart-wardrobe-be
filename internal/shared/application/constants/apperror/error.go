package apperror

import (
	"errors"
	"net/http"
	"runtime/debug"
)

type ValidationErrorItem struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type Error struct {
	Status     int                   `json:"status"`
	Title      string                `json:"title,omitempty"`
	Message    string                `json:"message,omitempty"`
	Errors     []ValidationErrorItem `json:"errors,omitempty"`
	StackTrace any                   `json:"stackTrace,omitempty"`

	Cause    error  `json:"-"`
	rawStack string `json:"-"`
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return http.StatusText(e.Status)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *Error) Stack() string {
	if e == nil {
		return ""
	}
	return e.rawStack
}

func (e *Error) WithStackTrace(stackTrace any) Error {
	res := *e
	res.StackTrace = stackTrace
	return res
}

func (e *Error) Is(target error) bool {
	if e == nil || target == nil {
		return false
	}
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Status == t.Status && e.Message == t.Message
}

func New(status int, title, message string) *Error {
	return &Error{
		Status:   status,
		Title:    title,
		Message:  message,
		rawStack: string(debug.Stack()),
	}
}

func Wrap(err error, fallbackMsg ...string) error {
	if err == nil {
		return nil
	}

	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}

	msg := "Đã xảy ra lỗi hệ thống"
	if len(fallbackMsg) > 0 && fallbackMsg[0] != "" {
		msg = fallbackMsg[0]
	} else if err.Error() != "" {
		msg = err.Error()
	}

	return &Error{
		Status:   http.StatusInternalServerError,
		Title:    "Lỗi hệ thống",
		Message:  msg,
		Cause:    err,
		rawStack: string(debug.Stack()),
	}
}

func From(err error) *Error {
	if err == nil {
		return nil
	}

	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}

	if err.Error() == "" {
		return NewInternalError("Đã xảy ra lỗi hệ thống")
	}

	wrapped := Wrap(err, err.Error())
	var converted *Error
	if errors.As(wrapped, &converted) {
		return converted
	}

	return NewInternalError(err.Error())
}
