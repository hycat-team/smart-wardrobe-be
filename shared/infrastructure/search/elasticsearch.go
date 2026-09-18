package search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/config"
	"smart-wardrobe-be/shared/pkg/logger"

	"go.uber.org/zap"
)

// errSearchDisabled is returned by Search when the engine is off via config.
// Read paths treat any search error as a database-fallback trigger (FR-002).
var errSearchDisabled = errors.New("elasticsearch is disabled by configuration")

type ElasticsearchClient struct {
	cfg        *config.Config
	logger     logger.Interface
	httpClient *http.Client
	isHealthy  int32
	stopChan   chan struct{}
	enabled    bool
}

func NewElasticsearchClient(cfg *config.Config, l logger.Interface) *ElasticsearchClient {
	client := &ElasticsearchClient{
		cfg:    cfg,
		logger: l,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		isHealthy: 1,
		stopChan:  make(chan struct{}),
		enabled:   cfg.Elasticsearch.Enabled,
	}

	// FR-001: when disabled, never start the health goroutine and report
	// unhealthy so sync paths skip without retrying (FR-003).
	if !client.enabled {
		atomic.StoreInt32(&client.isHealthy, 0)
		return client
	}

	go client.startHealthCheck()

	return client
}

// Enabled reports whether the engine is switched on via configuration.
func (c *ElasticsearchClient) Enabled() bool {
	return c.enabled
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

		// The ES response body stays in the log above; the client-facing
		// error must be static so the raw body never reaches callers.
		return nil, apperror.NewUpstreamUnavailable(fmt.Errorf("elasticsearch error status %d", resp.StatusCode))
	}

	return respBody, nil
}

func (c *ElasticsearchClient) IndexDocument(ctx context.Context, index, id string, doc any) error {
	// FR-001/FR-003: no-op when disabled — never fail, never fill the DLQ.
	if !c.enabled {
		return nil
	}
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
	// FR-001/FR-003: no-op when disabled — never fail, never fill the DLQ.
	if !c.enabled {
		return nil
	}
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

func (c *ElasticsearchClient) IsHealthy() bool {
	return atomic.LoadInt32(&c.isHealthy) == 1
}

func (c *ElasticsearchClient) Stop() {
	close(c.stopChan)
}

func (c *ElasticsearchClient) startHealthCheck() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	c.checkHealth()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.checkHealth()
		}
	}
}

func (c *ElasticsearchClient) checkHealth() {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	fullUrl := c.getAddress()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullUrl, nil)
	if err != nil {
		atomic.StoreInt32(&c.isHealthy, 0)
		return
	}

	if c.cfg.Elasticsearch.User != "" && c.cfg.Elasticsearch.Password != "" {
		req.SetBasicAuth(c.cfg.Elasticsearch.User, c.cfg.Elasticsearch.Password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if atomic.SwapInt32(&c.isHealthy, 0) == 1 {
			c.logger.Warn("Elasticsearch health check failed, marking as UNHEALTHY", zap.Error(err))
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if atomic.SwapInt32(&c.isHealthy, 1) == 0 {
			c.logger.Info("Elasticsearch health check recovered, marking as HEALTHY")
		}
	} else {
		if atomic.SwapInt32(&c.isHealthy, 0) == 1 {
			c.logger.Warn("Elasticsearch health check returned non-OK status", zap.Int("status", resp.StatusCode))
		}
	}
}

func (c *ElasticsearchClient) Search(ctx context.Context, index string, queryBody any) ([]byte, error) {
	// FR-002: disabled engine errors out so read paths fall back to database.
	if !c.enabled {
		return nil, errSearchDisabled
	}
	if !c.IsHealthy() {
		return nil, fmt.Errorf("elasticsearch is currently unhealthy (circuit breaker active)")
	}

	body, err := json.Marshal(queryBody)
	if err != nil {
		return nil, err
	}

	urlPath := fmt.Sprintf("%s/_search", index)
	resp, err := c.doRequest(ctx, http.MethodPost, urlPath, body)
	if err != nil {
		// Instantly mark as unhealthy on request failure (e.g. connection refused, network timeout)
		if atomic.SwapInt32(&c.isHealthy, 0) == 1 {
			c.logger.Warn("Elasticsearch request failed, tripping circuit breaker to UNHEALTHY")
		}
		return nil, err
	}
	return resp, nil
}
