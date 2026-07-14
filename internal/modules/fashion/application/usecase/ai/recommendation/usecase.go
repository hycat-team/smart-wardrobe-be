package recommendation

import (
	"smart-wardrobe-be/internal/config"
	aiexecutor_contract "smart-wardrobe-be/internal/modules/aiexecutor/presentation/contract"
	brand_contract "smart-wardrobe-be/internal/modules/brand/presentation/contract"
	"smart-wardrobe-be/internal/modules/fashion/application/dto"
	uc_interfaces "smart-wardrobe-be/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/fashion/application/usecase/ai/recommendation/parser"
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	"smart-wardrobe-be/internal/modules/subscription/presentation/contract"
	wardrobe_contract "smart-wardrobe-be/internal/modules/wardrobe/presentation/contract"
	"smart-wardrobe-be/internal/pkg/logger"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
)

// OutfitRecommendationUseCase điều phối các luồng nghiệp vụ gợi ý trang phục bằng AI.
type OutfitRecommendationUseCase struct {
	cfg              *config.Config
	logger           logger.Interface
	wardrobeContract wardrobe_contract.IWardrobeContract
	fashionItemRepo  repositories.IFashionItemRepository
	aiService        aiexecutor_contract.IAIOrchestrator
	userSubContract  contract.IUserSubscriptionContract
	userQuotaCtr     contract.IUserQuotaContract
	brandContract    brand_contract.IBrandContract
	uow              shared_repos.IUnitOfWork
	nlpParser        *parser.LocalNLPParser
}

// NewOutfitRecommendationUseCase khởi tạo một thực thể mới của [OutfitRecommendationUseCase] với các cấu hình và repository phụ thuộc cần thiết.
func NewOutfitRecommendationUseCase(
	cfg *config.Config,
	l logger.Interface,
	wardrobeContract wardrobe_contract.IWardrobeContract,
	fashionItemRepo repositories.IFashionItemRepository,
	aiService aiexecutor_contract.IAIOrchestrator,
	userSubContract contract.IUserSubscriptionContract,
	userQuotaCtr contract.IUserQuotaContract,
	brandContract brand_contract.IBrandContract,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IOutfitRecommendationUseCase {
	_ = dto.ParsedIntent{} // import check
	return &OutfitRecommendationUseCase{
		cfg:              cfg,
		logger:           l,
		wardrobeContract: wardrobeContract,
		fashionItemRepo:  fashionItemRepo,
		aiService:        aiService,
		userSubContract:  userSubContract,
		userQuotaCtr:     userQuotaCtr,
		brandContract:    brandContract,
		uow:              uow,
		nlpParser:        parser.NewLocalNLPParser(),
	}
}
