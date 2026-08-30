package messaging

// Integration tests cho spec 016 (T021, T026, T037).
//
// Các test này cần hạ tầng thật và MẶC ĐỊNH BỊ SKIP:
//   - RabbitMQ tests: set TEST_RABBITMQ_URL (vd amqp://guest:guest@localhost:5672/)
//     và chạy KHÔNG kèm app (chỉ docker compose infra) để không có consumer
//     nào "ăn" message trước khi assert queue depth. Management API đọc qua
//     TEST_RABBITMQ_MGMT_URL (default http://localhost:15672, guest/guest).
//   - Postgres test (T026): set TEST_DATABASE_URL (vd postgres://user:pass@localhost:5432/db).
//
// Chạy: TEST_RABBITMQ_URL=... TEST_DATABASE_URL=... go test -race -run Integration ./internal/shared/infrastructure/messaging/

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	amqpConnectTimeout = 5 * time.Second
	depthPollInterval  = 500 * time.Millisecond
	depthPollTimeout   = 15 * time.Second
)

func testEnvOrSkip(t *testing.T, key string) string {
	t.Helper()
	if v := os.Getenv(key); v != "" {
		return v
	}
	t.Skipf("SKIP: %s not set (integration test requires real infrastructure)", key)
	return ""
}

// fetchQueueMessages đọc message count của 1 queue qua Management API.
func fetchQueueMessages(t *testing.T, queue string) (int, error) {
	t.Helper()
	base := os.Getenv("TEST_RABBITMQ_MGMT_URL")
	if base == "" {
		base = "http://localhost:15672"
	}
	user := os.Getenv("TEST_RABBITMQ_USER")
	pass := os.Getenv("TEST_RABBITMQ_PASSWORD")
	if user == "" {
		user, pass = "guest", "guest"
	}

	url := fmt.Sprintf("%s/api/queues/%%2F/%s", base, queue)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	req.SetBasicAuth(user, pass)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("management api returned %d", resp.StatusCode)
	}
	var body struct {
		Messages int `json:"messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, err
	}
	return body.Messages, nil
}

func dialTestAMQP(t *testing.T) *amqp.Connection {
	t.Helper()
	conn, err := amqp.DialConfig(testEnvOrSkip(t, "TEST_RABBITMQ_URL"), amqp.Config{Dial: amqp.DefaultDial(amqpConnectTimeout)})
	if err != nil {
		t.Fatalf("connect rabbitmq: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

// TestIntegration_DLQTopologyReachable (T021 - phần DLQ): mọi DLQ trong
// DLQTopology phải tồn tại và đếm được depth qua Management API.
func TestIntegration_DLQTopologyReachable(t *testing.T) {
	conn := dialTestAMQP(t)
	if _, err := conn.Channel(); err != nil {
		t.Fatalf("open channel: %v", err)
	}

	for _, b := range DLQTopology {
		count, err := fetchQueueMessages(t, b.Queue)
		if err != nil {
			t.Errorf("DLQ %s not queryable via management api: %v", b.Queue, err)
			continue
		}
		if count < 0 {
			t.Errorf("DLQ %s has negative message count: %d", b.Queue, count)
		}
	}
}

// TestIntegration_PoisonMessageRoutedToQueue (T021 - phần poison): publish
// poison JSON vào fashion_analyze_queue, assert depth tăng lên, rồi purge
// để test chạy lặp lại được. CHỈ an toàn trên stack test không có app consumer.
func TestIntegration_PoisonMessageRoutedToQueue(t *testing.T) {
	conn := dialTestAMQP(t)
	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	defer ch.Close()

	baseline, err := fetchQueueMessages(t, QueueFashionAnalyze)
	if err != nil {
		t.Fatalf("fetch baseline depth: %v", err)
	}
	err = ch.PublishWithContext(context.Background(),
		ExchangeName, RoutingKeyFashionAnalyze, false, false,
		amqp.Publishing{ContentType: "application/json", Body: []byte("{poison-not-json"), DeliveryMode: amqp.Persistent},
	)
	if err != nil {
		t.Fatalf("publish poison: %v", err)
	}

	if err := waitDepthAtLeast(t, QueueFashionAnalyze, baseline+1); err != nil {
		t.Fatalf("poison message did not arrive in %s: %v", QueueFashionAnalyze, err)
	}

	if _, err := ch.QueuePurge(QueueFashionAnalyze, false); err != nil {
		t.Logf("warn: purge %s failed: %v", QueueFashionAnalyze, err)
	}
}

// TestIntegration_RoutingBurstSeparation (T037): burst 100 wardrobe + 100
// brand events với routing key tương ứng — assert mỗi luồng tới đúng queue
// (wardrobe.event.# → elasticsearch_sync_queue; fashion.event.*.brand →
// brand_fashion_analyzed_queue). Chạy trên stack test KHÔNG có app consumer.
func TestIntegration_RoutingBurstSeparation(t *testing.T) {
	conn := dialTestAMQP(t)
	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	defer ch.Close()

	const burst = 100
	targets := []struct {
		name  string
		key   string
		queue string
	}{
		{"wardrobe", "wardrobe.event.test-burst", QueueElasticsearchSync},
		{"brand", "fashion.event.analyzed.brand.test-burst", QueueBrandFashionAnalyzed},
	}
	for _, target := range targets {
		baseline, err := fetchQueueMessages(t, target.queue)
		if err != nil {
			t.Fatalf("fetch baseline for %s: %v", target.queue, err)
		}
		for i := 0; i < burst; i++ {
			err := ch.PublishWithContext(context.Background(), ExchangeName, target.key,
				false, false,
				amqp.Publishing{ContentType: "application/json", Body: []byte(fmt.Sprintf(`{"burst":%q}`, uuid.NewString())), DeliveryMode: amqp.Persistent},
			)
			if err != nil {
				t.Fatalf("publish %s #%d: %v", target.name, i, err)
			}
		}
		if err := waitDepthAtLeast(t, target.queue, baseline+burst); err != nil {
			t.Fatalf("%s: expected +%d messages in %s: %v", target.name, burst, target.queue, err)
		}
		if _, err := ch.QueuePurge(target.queue, false); err != nil {
			t.Logf("warn: purge %s failed: %v", target.queue, err)
		}
	}
}

// TestIntegration_ClaimConcurrent (T026): 2 "replica" claim đồng thời —
// không ID nào được claim 2 lần nhờ atomic UPDATE...RETURNING + SKIP LOCKED.
func TestIntegration_ClaimConcurrent(t *testing.T) {
	dsn := testEnvOrSkip(t, "TEST_DATABASE_URL")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}

	// Seed 20 records pending.
	for i := 0; i < 20; i++ {
		if err := db.Exec(`INSERT INTO shared.outbox_records (id, topic, payload, status, attempts)
			VALUES (?, 'test.claim.concurrent', '{}', 'pending', 0)`, uuid.NewString()).Error; err != nil {
			t.Fatalf("seed record: %v", err)
		}
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM shared.outbox_records WHERE topic = 'test.claim.concurrent'`)
	})

	var mu sync.Mutex
	claimed := map[string]int{}
	var doubleClaims, total atomic.Int64
	var wg sync.WaitGroup
	for replica := 0; replica < 2; replica++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for round := 0; round < 5; round++ {
				var records []OutboxRecord
				err := db.Transaction(func(tx *gorm.DB) error {
					return tx.Raw(`UPDATE shared.outbox_records SET status = 'processing', updated_at = NOW()
						WHERE id IN (
							SELECT id FROM shared.outbox_records
							WHERE topic = 'test.claim.concurrent' AND status = 'pending'
							ORDER BY created_at ASC LIMIT 2
							FOR UPDATE SKIP LOCKED
						) RETURNING *`).Scan(&records).Error
				})
				if err != nil {
					t.Errorf("claim round %d: %v", round, err)
					return
				}
				mu.Lock()
				for _, rec := range records {
					claimed[rec.ID.String()]++
					if claimed[rec.ID.String()] > 1 {
						doubleClaims.Add(1)
					}
				}
				mu.Unlock()
				total.Add(int64(len(records)))
			}
		}()
	}
	wg.Wait()

	if doubleClaims.Load() > 0 {
		t.Errorf("double claims detected: %d", doubleClaims.Load())
	}
	if total.Load() != 20 {
		t.Errorf("total claimed = %d, want 20", total.Load())
	}
}

func waitDepthAtLeast(t *testing.T, queue string, minDepth int) error {
	t.Helper()
	deadline := time.Now().Add(depthPollTimeout)
	for time.Now().Before(deadline) {
		depth, err := fetchQueueMessages(t, queue)
		if err == nil && depth >= minDepth {
			return nil
		}
		time.Sleep(depthPollInterval)
	}
	depth, err := fetchQueueMessages(t, queue)
	return fmt.Errorf("queue %s depth = %d (err %v), want >= %d after %s", queue, depth, err, minDepth, depthPollTimeout)
}
