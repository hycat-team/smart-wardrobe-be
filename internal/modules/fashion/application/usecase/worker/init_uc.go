package worker

import (
	"context"
	"fmt"

	"smart-wardrobe-be/internal/config"
	brand_contract "smart-wardrobe-be/internal/modules/brand/presentation/contract"
	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/fashion/domain/constants/fashionitemtype"
	"smart-wardrobe-be/internal/modules/fashion/domain/entities"
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	wardrobe_contract "smart-wardrobe-be/internal/modules/wardrobe/presentation/contract"
	"smart-wardrobe-be/internal/pkg/logger"
	"smart-wardrobe-be/internal/shared/application/constants/eventconstants"
	"smart-wardrobe-be/internal/shared/application/event"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/internal/shared/observability/workerlog"

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
	uow              shared_repos.IUnitOfWork
}

func NewFashionInitUseCase(
	cfg *config.Config,
	logger logger.Interface,
	fashionItemRepo repositories.IFashionItemRepository,
	eventPublisher event.IEventPublisher,
	wardrobeContract wardrobe_contract.IWardrobeContract,
	brandContract brand_contract.IBrandContract,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IFashionInitUseCase {
	return &FashionInitUseCase{
		cfg:              cfg,
		logger:           logger,
		fashionItemRepo:  fashionItemRepo,
		eventPublisher:   eventPublisher,
		wardrobeContract: wardrobeContract,
		brandContract:    brandContract,
		uow:              uow,
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

		if job.ItemType == string(fashionitemtype.Brand) {
			if err := uc.brandContract.SetFashionItemID(txCtx, job.ItemID, fashionItemID); err != nil {
				return fmt.Errorf("failed to set fashion item id on brand item: %w", err)
			}
		} else {
			if err := uc.wardrobeContract.SetFashionItemID(txCtx, job.ItemID, fashionItemID); err != nil {
				return fmt.Errorf("failed to set fashion item id on wardrobe item: %w", err)
			}
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
