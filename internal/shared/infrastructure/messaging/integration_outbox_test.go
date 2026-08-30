package messaging_test

// TestIntegration_OutboxPublishBatch: chèn record
// pending vào DB thật → PublishBatch → assert record chuyển 'published' trong
// DB (chỉ khi broker confirm ack) VÀ message xuất hiện trong queue đích.
// Package riêng (messaging_test) vì cần import shared usecase — tránh cycle.
import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"smart-wardrobe-be/internal/config"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	shared_uc "smart-wardrobe-be/internal/shared/application/usecase"
	"smart-wardrobe-be/internal/shared/infrastructure/messaging"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type silentLogger struct{}

func (silentLogger) Debug(string, ...zap.Field) {}
func (silentLogger) Info(string, ...zap.Field)  {}
func (silentLogger) Warn(string, ...zap.Field)  {}
func (silentLogger) Error(string, ...zap.Field) {}
func (silentLogger) Fatal(string, ...zap.Field) {}

func fetchQueueDepth(t *testing.T, queue string) (int, error) {
	t.Helper()
	base := os.Getenv("TEST_RABBITMQ_MGMT_URL")
	if base == "" {
		base = "http://localhost:15672"
	}
	user, pass := os.Getenv("TEST_RABBITMQ_USER"), os.Getenv("TEST_RABBITMQ_PASSWORD")
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

func testEnvOrSkip2(t *testing.T, key string) string {
	t.Helper()
	if v := os.Getenv(key); v != "" {
		return v
	}
	t.Skipf("SKIP: %s not set (integration test requires real infrastructure)", key)
	return ""
}

func TestIntegration_OutboxPublishBatch(t *testing.T) {
	// DSN cần search_path=shared vì bảng outbox_records nằm ở schema shared.
	dsn := testEnvOrSkip2(t, "TEST_DATABASE_URL") + "&search_path=shared"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}

	user, pass := os.Getenv("TEST_RABBITMQ_USER"), os.Getenv("TEST_RABBITMQ_PASSWORD")
	if user == "" {
		user, pass = "guest", "guest"
	}
	cfg := &config.Config{
		Startup:  config.Startup{RetryAttempt1Seconds: 5, RetryAttempt2Seconds: 5, RetryAttempt3Seconds: 5},
		RabbitMQ: config.RabbitMQ{Host: "127.0.0.1", Port: 5672, User: user, Password: pass, Prefetch: 20},
	}
	var l logger.Interface = silentLogger{}
	client, err := messaging.NewRabbitMQClient(cfg, l, nil)
	if err != nil {
		t.Fatalf("new rabbitmq client: %v", err)
	}
	defer client.Close()

	outboxRepo := messaging.NewOutboxRepository(db)
	uc := shared_uc.NewOutboxUseCase(outboxRepo, client, messaging.NewOutboxGaugeProbe(cfg, l), cfg, l)

	queueBaseline, err := fetchQueueDepth(t, eventconstants.QueueAuditLog)
	if err != nil {
		t.Fatalf("fetch queue baseline: %v", err)
	}
	record := &messaging.OutboxRecord{Topic: "admin.audit.log", Payload: []byte(`{"integration":"test"}`),
		Status: messaging.OutboxStatusPending, Headers: []byte(`{"correlationId":"` + uuid.NewString() + `"}`)}
	if err := outboxRepo.Create(context.Background(), record); err != nil {
		t.Fatalf("create outbox record: %v", err)
	}
	t.Cleanup(func() {
		db.Exec(`DELETE FROM shared.outbox_records WHERE id = ?`, record.ID)
	})

	published, err := uc.PublishBatch(context.Background(), 100, 3, time.Minute)
	if err != nil {
		t.Fatalf("PublishBatch: %v", err)
	}
	if published < 1 {
		t.Fatalf("published = %d, want >= 1", published)
	}

	// Record được đánh dấu published CHỈ sau khi broker confirm.
	var after messaging.OutboxRecord
	if err := db.Where("id = ?", record.ID).First(&after).Error; err != nil {
		t.Fatalf("reload record: %v", err)
	}
	if after.Status != messaging.OutboxStatusPublished || after.PublishedAt == nil {
		t.Errorf("record status = %s (publishedAt %v), want published", after.Status, after.PublishedAt)
	}

	// Message phải thực sự nằm trong queue đích.
	deadline := time.Now().Add(15 * time.Second)
	for {
		depth, err := fetchQueueDepth(t, eventconstants.QueueAuditLog)
		if err == nil && depth >= queueBaseline+1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("message not in %s (depth %d, err %v)", eventconstants.QueueAuditLog, depth, err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Dọn message test khỏi queue qua connection AMQP riêng (autoAck 1 message).
	cleanupConn, err := amqp.Dial(testEnvOrSkip2(t, "TEST_RABBITMQ_URL"))
	if err != nil {
		t.Fatalf("dial cleanup connection: %v", err)
	}
	defer cleanupConn.Close()
	ch, err := cleanupConn.Channel()
	if err != nil {
		t.Fatalf("open cleanup channel: %v", err)
	}
	deliveries, err := ch.Consume(eventconstants.QueueAuditLog, "", true, false, false, false, nil)
	if err != nil {
		t.Fatalf("consume cleanup: %v", err)
	}
	select {
	case d := <-deliveries:
		_ = d
	case <-time.After(15 * time.Second):
		t.Log("warn: no message to clean from queue")
	}
	ch.Close()
}
