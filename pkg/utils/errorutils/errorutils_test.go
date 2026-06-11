package errorutils

import (
	"errors"
	"strings"
	"testing"

	"smart-wardrobe-be/internal/shared/application/constants/apperror"
)

func TestToAppErrorWrapsPlainError(t *testing.T) {
	appErr := ToAppError(errors.New("plain failure"))

	if appErr == nil {
		t.Fatal("expected app error")
	}
	if appErr.Status != 500 {
		t.Fatalf("expected 500, got %d", appErr.Status)
	}
	if appErr.Stack() == "" {
		t.Fatal("expected stack to be present")
	}
}

func TestMapErrorToProblemMapsSentinel(t *testing.T) {
	status, title, message := MapErrorToProblem(apperror.ErrForbidden())

	if status != 403 {
		t.Fatalf("expected 403, got %d", status)
	}
	if title == "" || message == "" {
		t.Fatalf("expected mapped title/message, got %q / %q", title, message)
	}
}

func TestFilterStackTraceArrayKeepsProjectFrames(t *testing.T) {
	stack := "goroutine 1 [running]:\nsmart-wardrobe-be/pkg/utils/otherutils.TestFilterStackTraceArrayKeepsProjectFrames()\n\t/work/pkg/utils/otherutils/otherutils_test.go:1 +0x1\nruntime.goexit()\n\t/usr/local/go/src/runtime/asm_amd64.s:1693 +0x1\n"

	filtered := FilterStackTraceArray(stack)
	joined := strings.Join(filtered, "\n")

	if !strings.Contains(joined, "smart-wardrobe-be/pkg/utils/otherutils.TestFilterStackTraceArrayKeepsProjectFrames") {
		t.Fatalf("expected project frame to be kept, got %v", filtered)
	}
	if strings.Contains(joined, "\n\t/work/pkg/utils/otherutils/otherutils_test.go:1") {
		t.Fatalf("expected file line to be compacted into one frame, got %v", filtered)
	}
	if strings.Contains(joined, "runtime.goexit") {
		t.Fatalf("expected runtime frame to be filtered, got %v", filtered)
	}
}

func TestFilterStackTraceArrayDropsAppErrorAndDuplicateFrames(t *testing.T) {
	stack := strings.Join([]string{
		"goroutine 1 [running]:",
		"smart-wardrobe-be/internal/shared/application/constants/apperror.New(...)",
		"\t/app/internal/shared/application/constants/apperror/error.go:57",
		"smart-wardrobe-be/internal/shared/application/constants/apperror.ErrUnauthorized()",
		"\t/app/internal/shared/application/constants/apperror/dynamic_errors.go:66 +0x18",
		"smart-wardrobe-be/internal/api/middleware/auth.Handle.func1(0xc0000e0300)",
		"\t/app/internal/api/middleware/auth.go:89 +0x2d7",
		"smart-wardrobe-be/internal/api/middleware/auth.Handle.func1(0xc0000e0300)",
		"\t/app/internal/api/middleware/auth.go:89 +0x2d7",
		"smart-wardrobe-be/internal/api/routes.NewEngine.GlobalErrorHandler.func3(0xc0000e0300)",
		"\t/app/internal/api/middleware/error_handler.go:47 +0xaf",
	}, "\n")

	filtered := FilterStackTraceArray(stack)
	joined := strings.Join(filtered, "\n")

	if strings.Contains(joined, "internal/shared/application/constants/apperror") {
		t.Fatalf("expected apperror frames to be filtered, got %v", filtered)
	}
	if strings.Contains(joined, "error_handler.go") {
		t.Fatalf("expected error handler frame to be filtered, got %v", filtered)
	}
	if count := strings.Count(joined, "auth.go:89"); count != 1 {
		t.Fatalf("expected duplicate frames to be collapsed, got %v", filtered)
	}
}

func TestFilterStackTraceArrayKeepsPublicValidationEntryPoint(t *testing.T) {
	stack := strings.Join([]string{
		"goroutine 86 [running]:",
		"smart-wardrobe-be/pkg/utils/validation.TranslateValidationError({0x18f2ba0, 0xc000535908})",
		"\t/app/pkg/utils/validation/validation.go:102 +0xda5",
		"smart-wardrobe-be/pkg/utils/validation.BindJSON(0xc0003fe300?, {0x1435760?, 0xc0005ab980?})",
		"\t/app/pkg/utils/validation/validation.go:118 +0x65",
		"smart-wardrobe-be/internal/modules/subscription/presentation/handler.(*BillingHandler).CreateWalletTopUp(0xc000408080, 0xc0003fe300)",
		"\t/app/internal/modules/subscription/presentation/handler/billing_handler.go:101 +0x79",
	}, "\n")

	filtered := FilterStackTraceArray(stack)
	joined := strings.Join(filtered, "\n")

	if strings.Contains(joined, "TranslateValidationError") {
		t.Fatalf("expected TranslateValidationError to be filtered, got %v", filtered)
	}
	if !strings.Contains(joined, "BindJSON") {
		t.Fatalf("expected BindJSON to be kept, got %v", filtered)
	}
	if !strings.Contains(joined, "CreateWalletTopUp") {
		t.Fatalf("expected caller frame to be kept, got %v", filtered)
	}
}

func TestFilterStackTraceArrayCompactsFunctionArgsAndFileLine(t *testing.T) {
	stack := strings.Join([]string{
		"goroutine 86 [running]:",
		"smart-wardrobe-be/internal/modules/subscription/presentation/handler.(*BillingHandler).CreateWalletTopUp(0xc000408080, 0xc0003fe300)",
		"\t/app/internal/modules/subscription/presentation/handler/billing_handler.go:101 +0x79",
	}, "\n")

	filtered := FilterStackTraceArray(stack)
	if len(filtered) < 2 {
		t.Fatalf("expected compacted frame, got %v", filtered)
	}
	if filtered[1] != "smart-wardrobe-be/internal/modules/subscription/presentation/handler.(*BillingHandler).CreateWalletTopUp @ billing_handler.go:101" {
		t.Fatalf("unexpected compacted frame: %v", filtered[1])
	}
	if strings.Contains(filtered[1], "0xc000") {
		t.Fatalf("expected arguments to be stripped, got %v", filtered[1])
	}
}

func TestFilterStackTraceArrayKeepsMiddlewareFrameWhenItIsOnlySignal(t *testing.T) {
	stack := strings.Join([]string{
		"goroutine 1 [running]:",
		"smart-wardrobe-be/internal/api/routes.NewEngine.(*RateLimitMiddleware).Handle.func5(0xc0000e0300)",
		"\t/app/internal/api/middleware/ratelimit.go:48 +0x48",
		"smart-wardrobe-be/internal/api/routes.NewEngine.GlobalErrorHandler.func3(0xc0000e0300)",
		"\t/app/internal/api/middleware/error_handler.go:47 +0xaf",
	}, "\n")

	filtered := FilterStackTraceArray(stack)
	joined := strings.Join(filtered, "\n")

	if !strings.Contains(joined, "RateLimitMiddleware") {
		t.Fatalf("expected middleware frame to be kept when it is the only signal, got %v", filtered)
	}
}
