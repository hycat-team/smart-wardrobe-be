package worker

import (
	"context"
	"fmt"

	"smart-wardrobe-be/services/commerce/internal/modules/fashion/application/dto"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	"smart-wardrobe-be/shared/application/event"
	"smart-wardrobe-be/shared/config"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/observability/workerlog"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type FashionInitUseCase struct {
	cfg             *config.Config
	logger          logger.Interface
	fashionItemRepo repositories.IFashionItemRepository
	eventPublisher  event.IEventPublisher
	uow             shared_repos.IUnitOfWork
}

func NewFashionInitUseCase(
	cfg *config.Config,
	logger logger.Interface,
	fashionItemRepo repositories.IFashionItemRepository,
	eventPublisher event.IEventPublisher,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IFashionInitUseCase {
	return &FashionInitUseCase{
		cfg:             cfg,
		logger:          logger,
		fashionItemRepo: fashionItemRepo,
		eventPublisher:  eventPublisher,
		uow:             uow,
	}
}

func (uc *FashionInitUseCase) ProcessInitJob(ctx context.Context, job dto.FashionAnalyzeJobDTO, run *workerlog.Run) error {
	run.AddTotal(1)

	if job.FashionItemID != nil {
		run.AddSkipped(1)
		run.ChildInfo(uc.logger, "Init job skipped: fashion item already exists",
			zap.String("itemId", job.ItemID.String()),
			zap.String("fashionItemId", job.FashionItemID.String()),
		)
		return nil
	}

	fashionItemID := uuid.New()
	fashionItem := &entities.FashionItem{
		AuditableEntity: entities.AuditableEntity{
			BaseEntity: entities.BaseEntity{ID: fashionItemID},
		},
		ImageUrl:          job.ImageUrl,
		ImagePublicID:     job.ImagePublicID,
		CategoryID:        job.CategoryID,
		ProcessingVersion: job.ProcessingVersion,
	}

	err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.fashionItemRepo.Create(txCtx, fashionItem); err != nil {
			return fmt.Errorf("failed to create fashion item: %w", err)
		}

		createdPayload := dto.FashionItemCreatedEventPayload{
			ItemID:        job.ItemID,
			FashionItemID: fashionItemID,
			ItemType:      job.ItemType,
			UserID:        job.UserID,
		}
		createdTopic := eventconstants.TopicFashionItemCreatedWardrobe
		if job.ItemType == "brand" {
			createdTopic = eventconstants.TopicFashionItemCreatedBrand
		}
		if err := uc.eventPublisher.Publish(txCtx, createdTopic, createdPayload); err != nil {
			return fmt.Errorf("failed to publish fashion item created event: %w", err)
		}

		// Publish to analyze queue for AI processing
		analyzeJob := job
		analyzeJob.FashionItemID = &fashionItemID
		return uc.eventPublisher.Publish(txCtx, eventconstants.TopicFashionAnalyze, analyzeJob)
	})
	if err != nil {
		return err
	}

	run.AddSuccess(1)
	run.ChildInfo(uc.logger, "Fashion item created and forwarded to analyze queue",
		zap.String("itemId", job.ItemID.String()),
		zap.String("fashionItemId", fashionItemID.String()),
	)
	return nil
}
