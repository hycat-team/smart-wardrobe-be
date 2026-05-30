package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/internal/shared/infrastructure/messaging"
	"smart-wardrobe-be/internal/shared/infrastructure/search"
	"smart-wardrobe-be/pkg/logger"

	"go.uber.org/zap"
)

type ElasticsearchSyncWorker struct {
	rabbitmqClient messaging.IRabbitMQClient
	esClient       *search.ElasticsearchClient
	wardrobeRepo   repositories.IWardrobeItemRepository
	logger         logger.Interface
}

func NewElasticsearchSyncWorker(
	rabbitmqClient messaging.IRabbitMQClient,
	esClient *search.ElasticsearchClient,
	wardrobeRepo repositories.IWardrobeItemRepository,
	l logger.Interface,
) *ElasticsearchSyncWorker {
	w := &ElasticsearchSyncWorker{
		rabbitmqClient: rabbitmqClient,
		esClient:       esClient,
		wardrobeRepo:   wardrobeRepo,
		logger:         l,
	}

	// Đồng bộ lần đầu: đẩy toàn bộ System Catalog Items từ PostgreSQL lên Elasticsearch
	go w.initialSync()

	// Khởi chạy 2 goroutine tiêu thụ sự kiện đồng bộ bất đồng bộ
	for range 2 {
		go w.startConsume()
	}

	return w
}

func (w *ElasticsearchSyncWorker) startConsume() {
	deliveries, err := w.rabbitmqClient.Consume(messaging.QueueElasticsearchSync)
	if err != nil {
		w.logger.Error("[ElasticsearchSyncWorker] Failed to start consuming from RabbitMQ queue "+messaging.QueueElasticsearchSync, zap.Error(err))
		return
	}

	w.logger.Info("[ElasticsearchSyncWorker] Worker started listening to RabbitMQ queue elasticsearch_sync_queue...")

	for d := range deliveries {
		var event dto.WardrobeEventPayload
		err := json.Unmarshal(d.Body, &event)
		if err != nil {
			w.logger.Error("[ElasticsearchSyncWorker] Failed to unmarshal sync event payload", zap.Error(err))
			_ = d.Nack(false, false)
			continue
		}

		ctx := context.Background()
		err = w.processSyncEvent(ctx, event)
		if err != nil {
			w.logger.Error("[ElasticsearchSyncWorker] Sync event execution failed",
				zap.String("itemId", event.ItemID.String()),
				zap.String("action", event.Action),
				zap.Error(err),
			)
			_ = d.Nack(false, false)
		} else {
			_ = d.Ack(false)
		}
	}
}

func (w *ElasticsearchSyncWorker) processSyncEvent(ctx context.Context, event dto.WardrobeEventPayload) error {
	switch event.Action {
	case "created", "updated":
		item, err := w.wardrobeRepo.GetByID(ctx, event.ItemID)
		if err != nil {
			return err
		}
		if item == nil {
			return fmt.Errorf("item not found in database for indexing")
		}

		// Chỉ đồng bộ các wardrobe item của hệ thống (SystemCatalogItem = 1) vào Elasticsearch
		if item.ItemType != 1 {
			w.logger.Info("[ElasticsearchSyncWorker] Skipping user item indexing, only system catalog items are synced to Elasticsearch",
				zap.String("itemId", item.ID.String()),
			)
			return nil
		}

		doc := buildItemDocument(item)
		return w.esClient.IndexDocument(ctx, "wardrobe_items", item.ID.String(), doc)

	case "deleted":
		return w.esClient.DeleteDocument(ctx, "wardrobe_items", event.ItemID.String())
	}

	return nil
}

// initialSync đẩy toàn bộ System Catalog Items từ PostgreSQL lên Elasticsearch khi app khởi chạy
func (w *ElasticsearchSyncWorker) initialSync() {
	ctx := context.Background()

	items, err := w.wardrobeRepo.GetSystemCatalogItems(ctx)
	if err != nil {
		w.logger.Error("[ElasticsearchSyncWorker] Failed to fetch system catalog items for initial sync", zap.Error(err))
		return
	}

	if len(items) == 0 {
		w.logger.Info("[ElasticsearchSyncWorker] No system catalog items found in PostgreSQL, skipping initial sync")
		return
	}

	successCount := 0
	for _, item := range items {
		doc := buildItemDocument(item)
		if err := w.esClient.IndexDocument(ctx, "wardrobe_items", item.ID.String(), doc); err != nil {
			w.logger.Error("[ElasticsearchSyncWorker] Failed to index system catalog item during initial sync",
				zap.String("itemId", item.ID.String()),
				zap.Error(err),
			)
			continue
		}
		successCount++
	}

	w.logger.Info("[ElasticsearchSyncWorker] Initial sync completed",
		zap.Int("total", len(items)),
		zap.Int("indexed", successCount),
	)
}

// buildItemDocument chuyển đổi entity WardrobeItem sang document map cho Elasticsearch
func buildItemDocument(item *entities.WardrobeItem) map[string]interface{} {
	return map[string]interface{}{
		"id":        item.ID.String(),
		"user_id":   item.UserID.String(),
		"item_type": int(item.ItemType),
		"category": map[string]interface{}{
			"id":   item.CategoryID.String(),
			"name": item.Category.Name,
			"slug": item.Category.Slug,
		},
		"image_url":      item.ImageUrl,
		"image_public_id": item.ImagePublicID,
		"color":          getStringValue(item.Color),
		"style":          getStringValue(item.Style),
		"material":       getStringValue(item.Material),
		"pattern":        getStringValue(item.Pattern),
		"fit":            getStringValue(item.Fit),
		"seasonality":    getStringValue(item.Seasonality),
		"description":    getStringValue(item.Description),
		"status":         item.Status,
		"created_at":     item.CreatedAt,
	}
}

func getStringValue(val *string) string {
	if val == nil {
		return ""
	}
	return *val
}
