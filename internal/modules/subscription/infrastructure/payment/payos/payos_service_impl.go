package payos

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"sort"
	"strconv"
	"strings"
	"time"

	"smart-wardrobe-be/config"
	subscriptionerrors "smart-wardrobe-be/internal/modules/subscription/application/errors"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	"smart-wardrobe-be/internal/shared/domain/constants/shared/currency"
	sharedmoney "smart-wardrobe-be/internal/shared/domain/money"

	"github.com/shopspring/decimal"
)

var minPayOSTransactionAmount = decimal.NewFromInt(2000)

type PayOSService struct {
	clientID       string
	apiKey         string
	checksumKey    string
	expiredMinutes int
	httpClient     *http.Client
}

func NewPayOSService(cfg *config.Config) payment.IPaymentGatewayService {
	return &PayOSService{
		clientID:       cfg.PayOS.ClientID,
		apiKey:         cfg.PayOS.ApiKey,
		checksumKey:    cfg.PayOS.ChecksumKey,
		expiredMinutes: cfg.PayOS.ExpiredMinutes,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *PayOSService) CreateCheckoutSession(ctx context.Context, req *payment.CheckoutSessionReq) (*payment.CheckoutSessionResult, error) {
	if s.clientID == "" || s.apiKey == "" || s.checksumKey == "" {
		return &payment.CheckoutSessionResult{Outcome: payment.OutcomeKnownFailure, ErrorCode: "CONFIGURATION_ERROR"}, errors.New("payos credentials are not fully configured in the environment")
	}

	if req.Amount.LessThan(minPayOSTransactionAmount) {
		return &payment.CheckoutSessionResult{Outcome: payment.OutcomeKnownFailure, ErrorCode: "MINIMUM_AMOUNT"}, subscriptionerrors.ErrPayosMinAmount(minPayOSTransactionAmount.IntPart())
	}
	amountVND, err := sharedmoney.ToMinorUnits(req.Amount, currency.VND)
	if err != nil {
		return &payment.CheckoutSessionResult{Outcome: payment.OutcomeKnownFailure, ErrorCode: "INVALID_AMOUNT"}, subscriptionerrors.ErrPayosMustBeInteger()
	}

	bodyMap := map[string]any{
		"orderCode":   req.OrderCode,
		"amount":      amountVND,
		"description": req.Description,
		"returnUrl":   req.ReturnUrl,
		"cancelUrl":   req.CancelUrl,
	}
	if !req.ExpiresAt.IsZero() {
		bodyMap["expiredAt"] = req.ExpiresAt.Unix()
	}

	var keys []string
	for k := range bodyMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, bodyMap[k]))
	}
	rawStr := strings.Join(parts, "&")

	signature := s.generateHMAC256(rawStr)

	reqPayload := map[string]any{
		"orderCode":   req.OrderCode,
		"amount":      amountVND,
		"description": req.Description,
		"returnUrl":   req.ReturnUrl,
		"cancelUrl":   req.CancelUrl,
		"signature":   signature,
	}
	if s.expiredMinutes > 0 {
		reqPayload["expiredAt"] = bodyMap["expiredAt"]
	}

	reqBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return &payment.CheckoutSessionResult{Outcome: payment.OutcomeKnownFailure, ErrorCode: "SERIALIZATION_ERROR"}, fmt.Errorf("failed to serialize checkout payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api-merchant.payos.vn/v2/payment-requests", bytes.NewBuffer(reqBytes))
	if err != nil {
		return &payment.CheckoutSessionResult{Outcome: payment.OutcomeKnownFailure, ErrorCode: "REQUEST_BUILD_ERROR"}, fmt.Errorf("failed to create http request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-client-id", s.clientID)
	httpReq.Header.Set("x-api-key", s.apiKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return &payment.CheckoutSessionResult{Outcome: payment.OutcomeUnknown, Retryable: true, ErrorCode: "NETWORK_ERROR"}, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &payment.CheckoutSessionResult{Outcome: payment.OutcomeUnknown, Retryable: true, ErrorCode: "RESPONSE_READ_ERROR"}, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		outcome := payment.OutcomeUnknown
		retryable := resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
			outcome = payment.OutcomeKnownFailure
		}
		return &payment.CheckoutSessionResult{Outcome: outcome, Retryable: retryable, ErrorCode: fmt.Sprintf("HTTP_%d", resp.StatusCode)}, fmt.Errorf("payos returned non-ok status %d", resp.StatusCode)
	}

	var res struct {
		Code string `json:"code"`
		Desc string `json:"desc"`
		Data struct {
			CheckoutURL string `json:"checkoutUrl"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBytes, &res); err != nil {
		return &payment.CheckoutSessionResult{Outcome: payment.OutcomeUnknown, Retryable: true, ErrorCode: "MALFORMED_RESPONSE"}, fmt.Errorf("failed to parse payos response: %w", err)
	}

	if res.Code != "00" {
		return &payment.CheckoutSessionResult{Outcome: payment.OutcomeKnownFailure, ErrorCode: res.Code}, fmt.Errorf("payos returned business error code %s: %s", res.Code, res.Desc)
	}

	return &payment.CheckoutSessionResult{CheckoutURL: res.Data.CheckoutURL, Outcome: payment.OutcomeSucceeded}, nil
}

func (s *PayOSService) GetPaymentLinkInfo(ctx context.Context, orderCode int64) (*payment.PaymentLinkInfo, error) {
	var response struct {
		Code string `json:"code"`
		Data struct {
			ID          string          `json:"id"`
			OrderCode   int64           `json:"orderCode"`
			Amount      decimal.Decimal `json:"amount"`
			AmountPaid  decimal.Decimal `json:"amountPaid"`
			Status      string          `json:"status"`
			CheckoutURL string          `json:"checkoutUrl"`
		} `json:"data"`
	}
	if err := s.doJSON(ctx, http.MethodGet, fmt.Sprintf("https://api-merchant.payos.vn/v2/payment-requests/%d", orderCode), nil, &response); err != nil {
		return nil, err
	}
	status := payment.ProviderUnknown
	switch strings.ToUpper(response.Data.Status) {
	case "pending":
		status = payment.ProviderPending
	case "PAID":
		status = payment.ProviderPaid
	case "cancelled":
		status = payment.ProviderCancelled
	}
	return &payment.PaymentLinkInfo{OrderCode: response.Data.OrderCode, PaymentLinkID: response.Data.ID, Amount: response.Data.Amount, AmountPaid: response.Data.AmountPaid, Currency: "vnd", Status: status, CheckoutURL: response.Data.CheckoutURL}, nil
}

func (s *PayOSService) CancelPaymentLink(ctx context.Context, orderCode int64, reason string) error {
	body, _ := json.Marshal(map[string]string{"cancellationReason": reason})
	return s.doJSON(ctx, http.MethodPost, fmt.Sprintf("https://api-merchant.payos.vn/v2/payment-requests/%d/cancel", orderCode), body, nil)
}

func (s *PayOSService) doJSON(ctx context.Context, method, url string, body []byte, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-client-id", s.clientID)
	req.Header.Set("x-api-key", s.apiKey)
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("payos returned status %d", resp.StatusCode)
	}
	if out != nil {
		return json.Unmarshal(data, out)
	}
	return nil
}

func (s *PayOSService) VerifyWebhook(ctx context.Context, rawBody []byte, signatureHeader string) error {
	var payload struct {
		Code      string         `json:"code"`
		Desc      string         `json:"desc"`
		Signature string         `json:"signature"`
		Data      map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return apperror.NewInternalError("Failed to parse PayOS webhook body")
	}

	sigToVerify := signatureHeader
	if sigToVerify == "" {
		sigToVerify = payload.Signature
	}

	if sigToVerify == "" {
		return apperror.NewUnauthorized("Missing payload checksum")
	}

	if !s.VerifyWebhookSignature(payload.Data, sigToVerify) {
		return apperror.NewUnauthorized("Invalid PayOS webhook signature")
	}

	return nil
}

func (s *PayOSService) VerifyWebhookSignature(dataMap map[string]any, expectedSignature string) bool {
	if s.checksumKey == "" {
		return false
	}

	var keys []string
	for k := range dataMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		v := dataMap[k]
		valStr := formatWebhookValue(v)
		parts = append(parts, fmt.Sprintf("%s=%s", k, valStr))
	}

	rawString := strings.Join(parts, "&")
	computedSignature := s.generateHMAC256(rawString)

	return computedSignature == expectedSignature
}

func formatWebhookValue(v any) string {
	if v == nil {
		return ""
	}

	switch val := v.(type) {
	case string:
		if val == "null" || val == "undefined" || val == "NULL" {
			return ""
		}
		return val
	case bool:
		return fmt.Sprintf("%t", val)
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%.0f", val)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case json.Number:
		return val.String()
	case []any:
		sortedSlice := make([]any, len(val))
		for i, item := range val {
			if itemMap, ok := item.(map[string]any); ok {
				sortedSlice[i] = sortMapKeys(itemMap)
			} else {
				sortedSlice[i] = item
			}
		}
		bytes, err := json.Marshal(sortedSlice)
		if err != nil {
			return ""
		}
		return string(bytes)
	case map[string]any:
		sortedMap := sortMapKeys(val)
		bytes, err := json.Marshal(sortedMap)
		if err != nil {
			return ""
		}
		return string(bytes)
	default:
		bytes, err := json.Marshal(val)
		if err != nil {
			return ""
		}
		return string(bytes)
	}
}

func sortMapKeys(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	sorted := make(map[string]any)
	for k, v := range m {
		if innerMap, ok := v.(map[string]any); ok {
			sorted[k] = sortMapKeys(innerMap)
		} else if innerSlice, ok := v.([]any); ok {
			sortedSlice := make([]any, len(innerSlice))
			for i, item := range innerSlice {
				if itemMap, ok := item.(map[string]any); ok {
					sortedSlice[i] = sortMapKeys(itemMap)
				} else {
					sortedSlice[i] = item
				}
			}
			sorted[k] = sortedSlice
		} else {
			sorted[k] = v
		}
	}
	return sorted
}

func (s *PayOSService) generateHMAC256(data string) string {
	h := hmac.New(sha256.New, []byte(s.checksumKey))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
