package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"smart-wardrobe-be/internal/shared/application/constants/apperror"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type testLogger struct{}

func (testLogger) Debug(string, ...zap.Field) {}
func (testLogger) Info(string, ...zap.Field)  {}
func (testLogger) Warn(string, ...zap.Field)  {}
func (testLogger) Error(string, ...zap.Field) {}
func (testLogger) Fatal(string, ...zap.Field) {}

func TestGlobalErrorHandlerUsesOriginStackForAppError(t *testing.T) {
	gin.SetMode(gin.DebugMode)

	router := gin.New()
	router.Use(GlobalErrorHandler(testLogger{}, "development"))
	router.GET("/boom", func(c *gin.Context) {
		c.Error(apperror.NewBadRequest("bad request"))
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	var body struct {
		Status     int      `json:"status"`
		Title      string   `json:"title"`
		Detail     string   `json:"detail"`
		StackTrace []string `json:"stack_trace"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(body.StackTrace) == 0 {
		t.Fatal("expected stack trace in development mode")
	}

	joined := strings.Join(body.StackTrace, "\n")
	if !strings.Contains(joined, "TestGlobalErrorHandlerUsesOriginStackForAppError") {
		t.Fatalf("expected origin stack frame, got %v", body.StackTrace)
	}
	if strings.Contains(joined, "internal/shared/application/constants/apperror") {
		t.Fatalf("expected apperror constructor frames to be filtered, got %v", body.StackTrace)
	}
	if strings.Contains(joined, "middleware.GlobalErrorHandler") {
		t.Fatalf("expected not to fabricate middleware-local stack, got %v", body.StackTrace)
	}
}

func TestGlobalErrorHandlerMapsSentinelError(t *testing.T) {
	gin.SetMode(gin.DebugMode)

	router := gin.New()
	router.Use(GlobalErrorHandler(testLogger{}, "development"))
	router.GET("/unauthorized", func(c *gin.Context) {
		c.Error(apperror.ErrUnauthorized())
	})

	req := httptest.NewRequest(http.MethodGet, "/unauthorized", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestGlobalErrorHandlerOmitsStackInProduction(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(GlobalErrorHandler(testLogger{}, "production"))
	router.GET("/prod", func(c *gin.Context) {
		c.Error(apperror.NewBadRequest("bad request"))
	})

	req := httptest.NewRequest(http.MethodGet, "/prod", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if strings.Contains(rec.Body.String(), "stack_trace") {
		t.Fatalf("expected production response to omit stack_trace, got %s", rec.Body.String())
	}
}

func TestGlobalErrorHandlerReturnsPanicStack(t *testing.T) {
	gin.SetMode(gin.DebugMode)

	router := gin.New()
	router.Use(GlobalErrorHandler(testLogger{}, "development"))
	router.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	if !strings.Contains(rec.Body.String(), "stack_trace") {
		t.Fatalf("expected panic response to include stack trace, got %s", rec.Body.String())
	}
}
