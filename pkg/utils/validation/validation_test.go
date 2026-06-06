package validation

import (
	"testing"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

type validationFixture struct {
	Email    string `json:"email" binding:"required,email" label:"email"`
	Password string `json:"password" binding:"required,min=8" label:"mật khẩu"`
}

func TestTranslateValidationErrorReturnsStructuredErrors(t *testing.T) {
	validate, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		t.Fatal("expected gin validator engine")
	}
	err := validate.Struct(validationFixture{})
	if err == nil {
		t.Fatal("expected validation error")
	}

	appErr := TranslateValidationError(err)
	if appErr == nil {
		t.Fatal("expected app error")
	}
	if appErr.Status != 400 {
		t.Fatalf("expected 400, got %d", appErr.Status)
	}
	if len(appErr.Errors) != 2 {
		t.Fatalf("expected 2 validation errors, got %+v", appErr.Errors)
	}
	if appErr.Message != appErr.Errors[0].Message {
		t.Fatalf("expected top-level message to match first validation error, got %q / %q", appErr.Message, appErr.Errors[0].Message)
	}
	if appErr.Errors[0].Field != "email" {
		t.Fatalf("expected first field to be email, got %+v", appErr.Errors[0])
	}
	if appErr.Errors[1].Field != "password" {
		t.Fatalf("expected second field to be password, got %+v", appErr.Errors[1])
	}
}
