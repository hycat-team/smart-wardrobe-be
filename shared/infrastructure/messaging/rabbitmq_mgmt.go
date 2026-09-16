package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"smart-wardrobe-be/shared/pkg/logger"

	"go.uber.org/zap"
)

// mgmtQueue là subset field của /api/queues trong RabbitMQ Management API
// mà ta cần cho gauges queue_depth/dlq_depth.
type mgmtQueue struct {
	Name     string `json:"name"`
	Messages int    `json:"messages"`
}

// FetchQueueDepths gọi RabbitMQ Management HTTP API, trả về tổng message
// trong các queue chính (không phải DLQ) và tổng message trong các DLQ.
// Lỗi trả về không fatal — caller chỉ log warn và giữ gauge cũ.
func FetchQueueDepths(ctx context.Context, host string, mgmtPort int, user, password string, l logger.Interface) (queueDepth, dlqDepth int64, err error) {
	url := fmt.Sprintf("http://%s:%d/api/queues", host, mgmtPort)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, err
	}
	req.SetBasicAuth(user, password)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("management api returned %d", resp.StatusCode)
	}

	var queues []mgmtQueue
	if err := json.NewDecoder(resp.Body).Decode(&queues); err != nil {
		return 0, 0, err
	}

	for _, q := range queues {
		if len(q.Name) > 4 && q.Name[len(q.Name)-4:] == ".dlq" {
			dlqDepth += int64(q.Messages)
		} else {
			queueDepth += int64(q.Messages)
		}
	}
	l.Debug("[MgmtAPI] Fetched queue depths",
		zap.Int64("queue_depth", queueDepth),
		zap.Int64("dlq_depth", dlqDepth),
	)
	return queueDepth, dlqDepth, nil
}
