package media

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
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
