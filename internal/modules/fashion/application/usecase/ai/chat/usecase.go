package chat

import (
	"smart-wardrobe-be/config"
	aiexecutor_contract "smart-wardrobe-be/internal/modules/aiexecutor/contract"
	brand_contract "smart-wardrobe-be/internal/modules/brand/contract"
	uc_interfaces "smart-wardrobe-be/internal/modules/fashion/application/interface/usecase"
	fashion_contract "smart-wardrobe-be/internal/modules/fashion/contract"
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
)

// WardrobeChatUseCase coordinates AI chat session and message workflows.
type WardrobeChatUseCase struct {
	cfg             *config.Config
	contextRepo     repositories.IConversationalContextRepository
	messageRepo     repositories.IMessageRepository
	wardrobeRepo    repositories.IWardrobeItemRepository
	aiService       aiexecutor_contract.IAIOrchestrator
	userSubContract contract.IUserSubscriptionContract
	userQuotaCtr    contract.IUserQuotaContract
	brandContract   brand_contract.IBrandContract
	fashionContract fashion_contract.IFashionContract
	uow             shared_repos.IUnitOfWork
}

// NewWardrobeChatUseCase builds the wardrobe AI chat use case.
func NewWardrobeChatUseCase(
	cfg *config.Config,
	contextRepo repositories.IConversationalContextRepository,
	messageRepo repositories.IMessageRepository,
	wardrobeRepo repositories.IWardrobeItemRepository,
	aiService aiexecutor_contract.IAIOrchestrator,
	userSubContract contract.IUserSubscriptionContract,
	userQuotaCtr contract.IUserQuotaContract,
	brandContract brand_contract.IBrandContract,
	fashionContract fashion_contract.IFashionContract,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IWardrobeChatUseCase {
	return &WardrobeChatUseCase{
		cfg:             cfg,
		contextRepo:     contextRepo,
		messageRepo:     messageRepo,
		wardrobeRepo:    wardrobeRepo,
		aiService:       aiService,
		userSubContract: userSubContract,
		userQuotaCtr:    userQuotaCtr,
		brandContract:   brandContract,
		fashionContract: fashionContract,
		uow:             uow,
	}
}
