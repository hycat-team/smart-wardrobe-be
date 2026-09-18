package payos

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"smart-wardrobe-be/services/account/internal/modules/subscription/application/interface/payment"
	"smart-wardrobe-be/shared/config"

	"github.com/shopspring/decimal"
)

func testConfig(endpoint string) *config.Config {
	return &config.Config{
		PayOS: config.PayOS{
			ClientID:       "test-client",
			ApiKey:         "test-key",
			ChecksumKey:    "test-checksum",
			Endpoint:       endpoint,
			ExpiredMinutes: 15,
		},
	}
}

func TestCreateCheckoutSessionDuplicateOrderIsRetryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"231","desc":"Đơn thanh toán đã tồn tại"}`))
	}))
	defer server.Close()

	svc := NewPayOSService(testConfig(server.URL))
	result, err := svc.CreateCheckoutSession(context.Background(), &payment.CheckoutSessionReq{
		OrderCode:   123,
		Amount:      decimal.NewFromInt(50000),
		Description: "CLOSY123",
		ReturnUrl:   "https://app.example.com/return",
		CancelUrl:   "https://app.example.com/cancel",
		ExpiresAt:   time.Now().Add(15 * time.Minute),
	})
	if err == nil {
		t.Fatal("expected business error for code 231, got nil")
	}
	if result == nil {
		t.Fatal("expected non-nil result for code 231")
	}
	if result.ErrorCode != payment.ErrCodeDuplicateOrder {
		t.Fatalf("expected error code 231, got %q", result.ErrorCode)
	}
	if result.Outcome != payment.OutcomeKnownFailure || !result.Retryable {
		t.Fatalf("expected known retryable failure, got %+v", result)
	}
	if !payment.IsDuplicateOrderCode(result) {
		t.Fatal("expected IsDuplicateOrderCode to report true for 231")
	}
}

func TestCreateCheckoutSessionOtherBusinessErrorIsNotRetryable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"20","desc":"Internal Server Error."}`))
	}))
	defer server.Close()

	svc := NewPayOSService(testConfig(server.URL))
	result, err := svc.CreateCheckoutSession(context.Background(), &payment.CheckoutSessionReq{
		OrderCode:   456,
		Amount:      decimal.NewFromInt(50000),
		Description: "CLOSY456",
		ReturnUrl:   "https://app.example.com/return",
		CancelUrl:   "https://app.example.com/cancel",
		ExpiresAt:   time.Now().Add(15 * time.Minute),
	})
	if err == nil {
		t.Fatal("expected business error for code 20, got nil")
	}
	if result == nil {
		t.Fatal("expected non-nil result for code 20")
	}
	if result.Retryable {
		t.Fatalf("expected non-retryable failure for code 20, got %+v", result)
	}
	if payment.IsDuplicateOrderCode(result) {
		t.Fatal("expected IsDuplicateOrderCode to report false for code 20")
	}
}
