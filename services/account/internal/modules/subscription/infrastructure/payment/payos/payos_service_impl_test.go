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

func TestGetPaymentLinkInfoMapsProviderStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   payment.ProviderPaymentStatus
	}{
		{"pending", "PENDING", payment.ProviderPending},
		{"processing", "PROCESSING", payment.ProviderProcessing},
		{"processing lowercase and padded", "  processing ", payment.ProviderProcessing},
		{"paid", "PAID", payment.ProviderPaid},
		{"cancelled", "CANCELLED", payment.ProviderCancelled},
		{"expired", "EXPIRED", payment.ProviderExpired},
		{"underpaid", "UNDERPAID", payment.ProviderUnderpaid},
		{"failed", "FAILED", payment.ProviderFailed},
		{"unknown", "SOMETHING_NEW", payment.ProviderUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"code":"00","desc":"success","data":{"id":"link-1","orderCode":123,"amount":5000,"amountPaid":0,"status":"` + tt.status + `"}}`))
			}))
			defer server.Close()

			svc := NewPayOSService(testConfig(server.URL))
			info, err := svc.GetPaymentLinkInfo(context.Background(), 123)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if info.Status != tt.want {
				t.Fatalf("status %q: expected %q, got %q", tt.status, tt.want, info.Status)
			}
		})
	}
}

func TestGetPaymentLinkInfoParsesCancellationDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"00","desc":"success","data":{"id":"link-1","orderCode":123,"amount":5000,"amountPaid":0,"status":"CANCELLED","canceledAt":"2026-10-20T10:00:00Z","cancellationReason":"User requested cancellation"}}`))
	}))
	defer server.Close()

	svc := NewPayOSService(testConfig(server.URL))
	info, err := svc.GetPaymentLinkInfo(context.Background(), 123)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.CancelledAt == nil {
		t.Fatal("expected cancelledAt to be parsed")
	}
	if !info.CancelledAt.Equal(time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected cancelledAt: %v", info.CancelledAt)
	}
	if info.CancellationReason != "User requested cancellation" {
		t.Fatalf("unexpected cancellationReason: %q", info.CancellationReason)
	}
}

func TestGetPaymentLinkInfoReturnsBusinessError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":"02","desc":"Mã thanh toán không tồn tại"}`))
	}))
	defer server.Close()

	svc := NewPayOSService(testConfig(server.URL))
	if _, err := svc.GetPaymentLinkInfo(context.Background(), 123); err == nil {
		t.Fatal("expected a business error for code 02, got nil")
	}
}
