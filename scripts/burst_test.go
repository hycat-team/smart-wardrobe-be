package main

import (
	"context"
	"fmt"
	"time"

	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/infrastructure/messaging"
)

// burst_test.go — dùng để benchmark burst 100 message (quickstart §6)
// Chạy: go run ./scripts/burst_test.go --count 100 --topic fashion.event.analyze
func main() {
	fmt.Println("Burst test placeholder — dùng Outbox Publish để gửi 100 message")
	fmt.Println("Ví dụ: for i:=0; i<100; i++ { publisher.Publish(ctx, eventconstants.TopicFashionAnalyze, job) }")
	_ = context.Background
	_ = eventconstants.TopicFashionAnalyze
	_ = messaging.ExchangeName
	_ = time.Now
}
