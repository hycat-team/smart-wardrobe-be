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
	"maps"
	"net/http"
	"sort"
	"strings"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/application/interface/payment"
)

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

	rawString := fmt.Sprintf("amount=%d&cancelUrl=%s&description=%s&orderCode=%d&returnUrl=%s",
		int(req.Amount), req.CancelUrl, req.Description, req.OrderCode, req.ReturnUrl)

	signature := s.generateHMAC256(rawString)

	payload := map[string]any{
		"orderCode":   req.OrderCode,
		"amount":      int(req.Amount),
		"description": req.Description,
		"cancelUrl":   req.CancelUrl,
		"returnUrl":   req.ReturnUrl,
		"signature":   signature,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to serialize payos payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api-merchant.payos.vn/v2/payment-requests", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create http request for payos: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-client-id", s.clientID)
	httpReq.Header.Set("x-api-key", s.apiKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("payos API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read payos response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("payos API returned status code %d: %s", resp.StatusCode, string(respBytes))
	}

	var res payOSResponse
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

	result := make(map[string]any)
	maps.Copy(result, payload.Data)
	result["webhook_code"] = payload.Code
	result["webhook_desc"] = payload.Desc

	return result, nil
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
		if v == nil {
			parts = append(parts, fmt.Sprintf("%s=", k))
			continue
		}

		var valStr string
		switch val := v.(type) {
		case string:
			valStr = val
		case float64:
			if val == float64(int64(val)) {
				valStr = fmt.Sprintf("%d", int64(val))
			} else {
				valStr = fmt.Sprintf("%g", val)
			}
		case int:
			valStr = fmt.Sprintf("%d", val)
		case int64:
			valStr = fmt.Sprintf("%d", val)
		case bool:
			valStr = fmt.Sprintf("%t", val)
		default:
			valBytes, _ := json.Marshal(val)
			valStr = string(valBytes)
		}
		parts = append(parts, fmt.Sprintf("%s=%s", k, valStr))
	}

	rawString := strings.Join(parts, "&")
	computedSignature := s.generateHMAC256(rawString)

	return computedSignature == expectedSignature
}

func (s *PayOSService) generateHMAC256(data string) string {
	h := hmac.New(sha256.New, []byte(s.checksumKey))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
