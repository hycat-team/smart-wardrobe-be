package recommendation

import (
	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/ai"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/logger"
)

// OutfitRecommendationUseCase coordinates outfit recommendation workflows.
type OutfitRecommendationUseCase struct {
	cfg             *config.Config
	logger          logger.Interface
	wardrobeRepo    repositories.IWardrobeItemRepository
	aiService       ai.IAIService
	userSubContract contract.IUserSubscriptionContract
	userQuotaCtr    contract.IUserQuotaContract
	uow             shared_repos.IUnitOfWork
	nlpParser       *LocalNLPParser
}

// NewOutfitRecommendationUseCase builds the outfit recommendation use case.
func NewOutfitRecommendationUseCase(
	cfg *config.Config,
	l logger.Interface,
	wardrobeRepo repositories.IWardrobeItemRepository,
	aiService ai.IAIService,
	userSubContract contract.IUserSubscriptionContract,
	userQuotaCtr contract.IUserQuotaContract,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IOutfitRecommendationUseCase {
	return &OutfitRecommendationUseCase{
		cfg:             cfg,
		logger:          l,
		wardrobeRepo:    wardrobeRepo,
		aiService:       aiService,
		userSubContract: userSubContract,
		userQuotaCtr:    userQuotaCtr,
		uow:             uow,
		nlpParser:       NewLocalNLPParser(),
	}
}
