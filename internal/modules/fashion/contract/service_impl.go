package contract

import (
	"context"

	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/shared/domain/entities"

	"github.com/google/uuid"
)

type FashionContract struct {
	fashionItemRepo repositories.IFashionItemRepository
	eventPublisher  event.IEventPublisher
}

func NewFashionContract(fashionItemRepo repositories.IFashionItemRepository, eventPublisher event.IEventPublisher) IFashionContract {
	return &FashionContract{
		fashionItemRepo: fashionItemRepo,
		eventPublisher:  eventPublisher,
	}
}

func (c *FashionContract) CreateFashionItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, itemType string, categoryID *uuid.UUID, imageUrl string, imagePublicID string) (uuid.UUID, error) {
	item := &entities.FashionItem{
		CategoryID:        categoryID,
		ImageUrl:          imageUrl,
		ImagePublicID:     imagePublicID,
		ProcessingVersion: 1,
	}
	err := c.fashionItemRepo.Create(ctx, item)
	if err != nil {
		return uuid.Nil, err
	}

	// Publish async analysis event to RabbitMQ
	job := FashionAnalyzeJobDTO{
		FashionItemID:     item.ID,
		ItemID:            itemID,
		ItemType:          itemType,
		UserID:            userID,
		ImageUrl:          imageUrl,
		ImagePublicID:     imagePublicID,
		CategoryID:        categoryID,
		ProcessingVersion: 1,
		RetryCount:        0,
	}
	_ = c.eventPublisher.Publish(ctx, "fashion.event.analyze_item", job)

	return item.ID, nil
}

func (c *FashionContract) GetFashionItem(ctx context.Context, id uuid.UUID) (*entities.FashionItem, error) {
	return c.fashionItemRepo.GetByID(ctx, id)
}

func (c *FashionContract) ListFashionItems(ctx context.Context, ids []uuid.UUID) ([]*entities.FashionItem, error) {
	return c.fashionItemRepo.GetByIDs(ctx, ids)
}
