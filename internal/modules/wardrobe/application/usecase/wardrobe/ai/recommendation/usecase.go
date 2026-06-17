package recommendation

import (
	"smart-wardrobe-be/config"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/modules/wardrobe/application/dto"
	"smart-wardrobe-be/internal/modules/wardrobe/application/usecase/wardrobe/ai/recommendation/parser"
	uc_interfaces "smart-wardrobe-be/internal/modules/wardrobe/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/wardrobe/domain/repositories"
	"smart-wardrobe-be/internal/shared/application/ai"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
	"smart-wardrobe-be/pkg/logger"
)

// OutfitRecommendationUseCase điều phối các luồng nghiệp vụ gợi ý trang phục bằng AI.
type OutfitRecommendationUseCase struct {
	cfg             *config.Config
	logger          logger.Interface
	wardrobeRepo    repositories.IWardrobeItemRepository
	aiService       ai.IAIService
	userSubContract contract.IUserSubscriptionContract
	userQuotaCtr    contract.IUserQuotaContract
	uow             shared_repos.IUnitOfWork
	nlpParser       *parser.LocalNLPParser
}

// NewOutfitRecommendationUseCase khởi tạo một thực thể mới của [OutfitRecommendationUseCase] với các cấu hình và repository phụ thuộc cần thiết.
func NewOutfitRecommendationUseCase(
	cfg *config.Config,
	l logger.Interface,
	wardrobeRepo repositories.IWardrobeItemRepository,
	aiService ai.IAIService,
	userSubContract contract.IUserSubscriptionContract,
	userQuotaCtr contract.IUserQuotaContract,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IOutfitRecommendationUseCase {
	_ = dto.ParsedIntent{} // import check
	return &OutfitRecommendationUseCase{
		cfg:             cfg,
		logger:          l,
		wardrobeRepo:    wardrobeRepo,
		aiService:       aiService,
		userSubContract: userSubContract,
		userQuotaCtr:    userQuotaCtr,
		uow:             uow,
		nlpParser:       parser.NewLocalNLPParser(),
	}
}
