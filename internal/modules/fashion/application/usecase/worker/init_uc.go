package worker

import (
	"context"
	"fmt"

	"smart-wardrobe-be/config"
	brand_contract "smart-wardrobe-be/internal/modules/brand/contract"
	uc_interfaces "smart-wardrobe-be/internal/modules/fashion/application/interface/usecase"
	fashion_contract "smart-wardrobe-be/internal/modules/fashion/contract"
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	wardrobe_contract "smart-wardrobe-be/internal/modules/wardrobe/contract"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/shared/domain/entities"
	"smart-wardrobe-be/internal/shared/observability/workerlog"
	"smart-wardrobe-be/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type FashionInitUseCase struct {
	cfg              *config.Config
	logger           logger.Interface
	fashionItemRepo  repositories.IFashionItemRepository
	eventPublisher   event.IEventPublisher
	wardrobeContract wardrobe_contract.IWardrobeContract
	brandContract    brand_contract.IBrandContract
}

func NewFashionInitUseCase(
	cfg *config.Config,
	logger logger.Interface,
	fashionItemRepo repositories.IFashionItemRepository,
	eventPublisher event.IEventPublisher,
	wardrobeContract wardrobe_contract.IWardrobeContract,
	brandContract brand_contract.IBrandContract,
) uc_interfaces.IFashionInitUseCase {
	return &FashionInitUseCase{
		cfg:              cfg,
		logger:           logger,
		fashionItemRepo:  fashionItemRepo,
		eventPublisher:   eventPublisher,
		wardrobeContract: wardrobeContract,
		brandContract:    brandContract,
	}
}

func (uc *FashionInitUseCase) ProcessInitJob(ctx context.Context, job fashion_contract.FashionAnalyzeJobDTO, run *workerlog.Run) error {
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
	if err := uc.fashionItemRepo.Create(ctx, fashionItem); err != nil {
		return fmt.Errorf("failed to create fashion item: %w", err)
	}

	if job.ItemType == "brand" {
		if err := uc.brandContract.SetFashionItemID(ctx, job.ItemID, fashionItemID); err != nil {
			return fmt.Errorf("failed to set fashion item id on brand item: %w", err)
		}
	} else {
		if err := uc.wardrobeContract.SetFashionItemID(ctx, job.ItemID, fashionItemID); err != nil {
			return fmt.Errorf("failed to set fashion item id on wardrobe item: %w", err)
		}
	}

	// Publish to analyze queue for AI processing
	analyzeJob := job
	analyzeJob.FashionItemID = &fashionItemID
	_ = uc.eventPublisher.Publish(ctx, "fashion.event.analyze", analyzeJob)

	run.AddSuccess(1)
	run.ChildInfo(uc.logger, "Fashion item created and forwarded to analyze queue",
		zap.String("itemId", job.ItemID.String()),
		zap.String("fashionItemId", fashionItemID.String()),
	)
	return nil
}
