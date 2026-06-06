package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/shared/application/constants/apperror"
	"smart-wardrobe-be/pkg/logger"

	"go.uber.org/zap"
)

type ElasticsearchClient struct {
	cfg        *config.Config
	logger     logger.Interface
	httpClient *http.Client
}

func NewElasticsearchClient(cfg *config.Config, l logger.Interface) *ElasticsearchClient {
	return &ElasticsearchClient{
		cfg:    cfg,
		logger: l,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *ElasticsearchClient) getAddress() string {
	if len(c.cfg.Elasticsearch.Addresses) > 0 {
		return strings.TrimRight(c.cfg.Elasticsearch.Addresses[0], "/")
	}
	return "http://localhost:9200"
}

func (c *ElasticsearchClient) doRequest(ctx context.Context, method, urlPath string, body []byte) ([]byte, error) {
	fullUrl := fmt.Sprintf("%s/%s", c.getAddress(), strings.TrimLeft(urlPath, "/"))

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullUrl, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.cfg.Elasticsearch.User != "" && c.cfg.Elasticsearch.Password != "" {
		req.SetBasicAuth(c.cfg.Elasticsearch.User, c.cfg.Elasticsearch.Password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Elasticsearch request failed",
			zap.String("method", method),
			zap.String("url", fullUrl),
			zap.Any("error", err),
		)
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		if resp.StatusCode == http.StatusNotFound {
			return nil, apperror.ErrSearchIndexNotFound()
		}

		c.logger.Error("Elasticsearch returned error status",
			zap.Int("status", resp.StatusCode),
			zap.String("response", string(respBody)),
		)

		return nil, apperror.NewError(resp.StatusCode, "Hệ thống gặp sự cố", string(respBody))
	}

	return respBody, nil
}

func (c *ElasticsearchClient) IndexDocument(ctx context.Context, index, id string, doc any) error {
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	urlPath := fmt.Sprintf("%s/_doc/%s", index, id)
	_, err = c.doRequest(ctx, http.MethodPut, urlPath, body)
	if err != nil {
		return err
	}

	c.logger.Info("Successfully indexed document in Elasticsearch",
		zap.String("index", index),
		zap.String("id", id),
	)
	return nil
}

func (c *ElasticsearchClient) DeleteDocument(ctx context.Context, index, id string) error {
	urlPath := fmt.Sprintf("%s/_doc/%s", index, id)
	_, err := c.doRequest(ctx, http.MethodDelete, urlPath, nil)
	if err != nil {
		// Ignore 404 error if document does not exist to let the worker continue running smoothly
		if strings.Contains(err.Error(), "404") {
			c.logger.Info("Document not found in Elasticsearch during deletion, skipping",
				zap.String("index", index),
				zap.String("id", id),
			)
			return nil
		}
		return err
	}

	c.logger.Info("Successfully deleted document from Elasticsearch",
		zap.String("index", index),
		zap.String("id", id),
	)
	return nil
}

func (c *ElasticsearchClient) Search(ctx context.Context, index string, queryBody any) ([]byte, error) {
	body, err := json.Marshal(queryBody)
	if err != nil {
		return nil, err
	}

	urlPath := fmt.Sprintf("%s/_search", index)
	return c.doRequest(ctx, http.MethodPost, urlPath, body)
}
