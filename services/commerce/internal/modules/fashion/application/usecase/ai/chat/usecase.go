package chat

import (
	"smart-wardrobe-be/services/account/contract/subscription"
	aiexecutor_contract "smart-wardrobe-be/services/ai/contract/aiexecutor"
	brand_contract "smart-wardrobe-be/services/commerce/internal/modules/brand/presentation/contract"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/fashion/domain/repositories"
	wardrobe_contract "smart-wardrobe-be/services/commerce/internal/modules/wardrobe/presentation/contract"
	"smart-wardrobe-be/shared/config"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
)

// WardrobeChatUseCase coordinates AI chat session and message workflows.
type WardrobeChatUseCase struct {
	cfg              *config.Config
	contextRepo      repositories.IConversationalContextRepository
	messageRepo      repositories.IMessageRepository
	fashionItemRepo  repositories.IFashionItemRepository
	aiService        aiexecutor_contract.IAIOrchestrator
	userSubContract  contract.IUserSubscriptionContract
	userQuotaCtr     contract.IUserQuotaContract
	brandContract    brand_contract.IBrandContract
	wardrobeContract wardrobe_contract.IWardrobeContract
	uow              shared_repos.IUnitOfWork
}

// NewWardrobeChatUseCase builds the wardrobe AI chat use case.
func NewWardrobeChatUseCase(
	cfg *config.Config,
	contextRepo repositories.IConversationalContextRepository,
	messageRepo repositories.IMessageRepository,
	fashionItemRepo repositories.IFashionItemRepository,
	aiService aiexecutor_contract.IAIOrchestrator,
	userSubContract contract.IUserSubscriptionContract,
	userQuotaCtr contract.IUserQuotaContract,
	brandContract brand_contract.IBrandContract,
	wardrobeContract wardrobe_contract.IWardrobeContract,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IWardrobeChatUseCase {
	return &WardrobeChatUseCase{
		cfg:              cfg,
		contextRepo:      contextRepo,
		messageRepo:      messageRepo,
		fashionItemRepo:  fashionItemRepo,
		aiService:        aiService,
		userSubContract:  userSubContract,
		userQuotaCtr:     userQuotaCtr,
		brandContract:    brandContract,
		wardrobeContract: wardrobeContract,
		uow:              uow,
	}
}
