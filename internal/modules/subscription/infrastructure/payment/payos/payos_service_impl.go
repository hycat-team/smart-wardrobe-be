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
	"sort"
	"strconv"
	"strings"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
	"smart-wardrobe-be/internal/shared/application/constants/errorcode"
)

const MinPayOSTransactionAmount = 2000.00

type PayOSService struct {
	clientID    string
	apiKey      string
	checksumKey string
	httpClient  *http.Client
}

func NewPayOSService(cfg *config.Config) payment.IPaymentGatewayService {
	return &PayOSService{
		clientID:    cfg.PayOS.ClientID,
		apiKey:      cfg.PayOS.ApiKey,
		checksumKey: cfg.PayOS.ChecksumKey,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *PayOSService) CreateCheckoutSession(ctx context.Context, req *payment.CheckoutSessionReq) (string, error) {
	if s.clientID == "" || s.apiKey == "" || s.checksumKey == "" {
		return "", errors.New("payos credentials are not fully configured in the environment")
	}

	if req.Amount < MinPayOSTransactionAmount {
		return "", errorcode.NewBadRequest(fmt.Sprintf("Số tiền thanh toán tối thiểu qua cổng PayOS là %d VND", int(MinPayOSTransactionAmount)))
	}

	bodyMap := map[string]any{
		"orderCode":   req.OrderCode,
		"amount":      int64(req.Amount),
		"description": req.Description,
		"returnUrl":   req.ReturnUrl,
		"cancelUrl":   req.CancelUrl,
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
		"amount":      int64(req.Amount),
		"description": req.Description,
		"returnUrl":   req.ReturnUrl,
		"cancelUrl":   req.CancelUrl,
		"signature":   signature,
	}

	reqBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return "", fmt.Errorf("failed to serialize checkout payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api-merchant.payos.vn/v2/payment-requests", bytes.NewBuffer(reqBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create http request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-client-id", s.clientID)
	httpReq.Header.Set("x-api-key", s.apiKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("payos returned non-ok status %d: %s", resp.StatusCode, string(respBytes))
	}

	var res struct {
		Code string `json:"code"`
		Desc string `json:"desc"`
		Data struct {
			CheckoutUrl string `json:"checkoutUrl"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBytes, &res); err != nil {
		return "", fmt.Errorf("failed to parse payos response: %w", err)
	}

	if res.Code != "00" {
		return "", fmt.Errorf("payos returned business error code %s: %s", res.Code, res.Desc)
	}

	return res.Data.CheckoutUrl, nil
}

func (s *PayOSService) VerifyWebhook(ctx context.Context, rawBody []byte, signatureHeader string) (map[string]any, error) {
	var payload struct {
		Code      string         `json:"code"`
		Desc      string         `json:"desc"`
		Signature string         `json:"signature"`
		Data      map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	sigToVerify := signatureHeader
	if sigToVerify == "" {
		sigToVerify = payload.Signature
	}

	if sigToVerify == "" {
		return nil, errors.New("missing signature for webhook validation")
	}

	if !s.VerifyWebhookSignature(payload.Data, sigToVerify) {
		return nil, errors.New("invalid payos signature")
	}

	var fullMap map[string]any
	if err := json.Unmarshal(rawBody, &fullMap); err != nil {
		return nil, fmt.Errorf("failed to parse complete webhook map: %w", err)
	}

	return fullMap, nil
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
