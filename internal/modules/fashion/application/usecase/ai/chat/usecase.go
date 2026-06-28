package chat

import (
	"smart-wardrobe-be/config"
	brand_contract "smart-wardrobe-be/internal/modules/brand/contract"
	uc_interfaces "smart-wardrobe-be/internal/modules/fashion/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/fashion/domain/repositories"
	"smart-wardrobe-be/internal/modules/subscription/contract"
	"smart-wardrobe-be/internal/shared/application/ai"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
)

// WardrobeChatUseCase coordinates AI chat session and message workflows.
type WardrobeChatUseCase struct {
	cfg             *config.Config
	contextRepo     repositories.IConversationalContextRepository
	messageRepo     repositories.IMessageRepository
	wardrobeRepo    repositories.IWardrobeItemRepository
	aiService       ai.IAIService
	userSubContract contract.IUserSubscriptionContract
	userQuotaCtr    contract.IUserQuotaContract
	brandContract   brand_contract.IBrandContract
	uow             shared_repos.IUnitOfWork
}

// NewWardrobeChatUseCase builds the wardrobe AI chat use case.
func NewWardrobeChatUseCase(
	cfg *config.Config,
	contextRepo repositories.IConversationalContextRepository,
	messageRepo repositories.IMessageRepository,
	wardrobeRepo repositories.IWardrobeItemRepository,
	aiService ai.IAIService,
	userSubContract contract.IUserSubscriptionContract,
	userQuotaCtr contract.IUserQuotaContract,
	brandContract brand_contract.IBrandContract,
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
		uow:             uow,
	}
}
