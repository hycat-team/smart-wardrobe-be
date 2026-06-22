package validation

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

type validationFixture struct {
	Email    string `json:"email" binding:"required,email" label:"email"`
	Password string `json:"password" binding:"required,min=8" label:"mật khẩu"`
}

type nfcFixture struct {
	Content string `json:"content" binding:"required,nfcmax=2"`
}

func TestBindJSONNormalizesNFCBeforePersistingAndCountsNormalizedRunes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"content":"a\u0301b"}`))
	req.Header.Set("Content-Type", "application/json")
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req
	var input nfcFixture
	if err := BindJSON(c, &input); err != nil {
		t.Fatalf("expected decomposed text to fit after NFC normalization: %v", err)
	}
	if input.Content != "áb" {
		t.Fatalf("expected NFC-normalized content, got %q", input.Content)
	}
}

func TestNFCMaxRejectsNormalizedTextOverLimit(t *testing.T) {
	validate := binding.Validator.Engine().(*validator.Validate)
	err := validate.Struct(nfcFixture{Content: "a\u0301bc"})
	if err == nil {
		t.Fatal("expected normalized three-rune content to exceed limit")
	}
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

type complexityFixture struct {
	Password string `json:"password" binding:"required,password_complexity" label:"mật khẩu"`
}

func TestPasswordComplexityValidator(t *testing.T) {
	validate, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		t.Fatal("expected gin validator engine")
	}

	tests := []struct {
		password string
		isValid  bool
	}{
		{"secret", false},
		{"Secret1", false},
		{"Secret123!", true},
		{"SECRET123!", false},
		{"secret123!", false},
		{"Secret!!!", false}, // no number
		{"Secret12", false},  // no special
		{"Ab1!", false},      // under 8 chars
	}

	for _, tc := range tests {
		err := validate.Struct(complexityFixture{Password: tc.password})
		if tc.isValid && err != nil {
			t.Errorf("expected password %q to be valid, but got error: %v", tc.password, err)
		} else if !tc.isValid && err == nil {
			t.Errorf("expected password %q to be invalid, but got no error", tc.password)
		}
	}
}
