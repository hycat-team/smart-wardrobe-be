package media

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/application/media"
)

type CloudinaryService struct {
	cfg *config.Config
}

func NewCloudinaryService(cfg *config.Config) media.IMediaService {
	return &CloudinaryService{
		cfg: cfg,
	}
}

func (s *CloudinaryService) GenerateUploadSignature(ctx context.Context, params dto.UploadSignatureParams) (*dto.UploadSignatureResult, error) {
	timestamp := time.Now().Unix()

	paramsMap := map[string]string{
		"timestamp": fmt.Sprintf("%d", timestamp),
		"folder":    params.Folder,
	}

	if params.PublicID != "" {
		paramsMap["public_id"] = params.PublicID
		paramsMap["overwrite"] = "true"
	}

	var keys []string
	for k := range paramsMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, paramsMap[k]))
	}
	queryString := strings.Join(parts, "&")

	stringToSign := fmt.Sprintf("%s%s", queryString, s.cfg.Cloudinary.ApiSecret)

	hash := sha1.New()
	hash.Write([]byte(stringToSign))
	signature := hex.EncodeToString(hash.Sum(nil))

	return &dto.UploadSignatureResult{
		Signature: signature,
		Timestamp: timestamp,
		ApiKey:    s.cfg.Cloudinary.ApiKey,
		PublicID:  params.PublicID,
		Folder:    params.Folder,
	}, nil
}

func (s *CloudinaryService) DeleteImage(ctx context.Context, publicID string) error {
	if publicID == "" {
		return nil
	}

	timestamp := time.Now().Unix()
	stringToSign := fmt.Sprintf("public_id=%s&timestamp=%d%s", publicID, timestamp, s.cfg.Cloudinary.ApiSecret)

	hash := sha1.New()
	hash.Write([]byte(stringToSign))
	signature := hex.EncodeToString(hash.Sum(nil))

	formData := url.Values{}
	formData.Set("public_id", publicID)
	formData.Set("timestamp", fmt.Sprintf("%d", timestamp))
	formData.Set("api_key", s.cfg.Cloudinary.ApiKey)
	formData.Set("signature", signature)

	destroyURL := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/destroy", s.cfg.Cloudinary.CloudName)
	req, err := http.NewRequestWithContext(ctx, "POST", destroyURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create destroy request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	cli := &http.Client{Timeout: 10 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call Cloudinary destroy API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Cloudinary destroy API error (HTTP %d): %s", resp.StatusCode, string(respBytes))
	}

	return nil
}
