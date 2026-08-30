package item

import (
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/domain/constants/branditem/branditemstatus"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	fashion_contract "smart-wardrobe-be/internal/modules/fashion/presentation/contract"
	identity_contract "smart-wardrobe-be/internal/modules/identity/presentation/contract"
	"smart-wardrobe-be/internal/shared/application/event"
	"smart-wardrobe-be/internal/shared/application/media"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
)

var activeStatus = branditemstatus.Active

type BrandItemUseCase struct {
	brandRepo         repositories.IBrandRepository
	memberRepo        repositories.IBrandMemberRepository
	customerRepo      repositories.IBrandCustomerRepository
	brandItemRepo     repositories.IBrandItemRepository
	feedbackRepo      repositories.IDigitalSampleResponseRepository
	userBriefContract identity_contract.IUserBriefContract
	fashionContract   fashion_contract.IFashionContract
	eventPublisher    event.IEventPublisher
	mediaService      media.IMediaService
	benefitUC         uc_interfaces.IBrandBenefitUseCase
	uow               shared_repos.IUnitOfWork
}

func NewBrandItemUseCase(
	brandRepo repositories.IBrandRepository,
	memberRepo repositories.IBrandMemberRepository,
	customerRepo repositories.IBrandCustomerRepository,
	brandItemRepo repositories.IBrandItemRepository,
	feedbackRepo repositories.IDigitalSampleResponseRepository,
	userBriefContract identity_contract.IUserBriefContract,
	fashionContract fashion_contract.IFashionContract,
	eventPublisher event.IEventPublisher,
	mediaService media.IMediaService,
	benefitUC uc_interfaces.IBrandBenefitUseCase,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IBrandItemUseCase {
	return &BrandItemUseCase{
		brandRepo:         brandRepo,
		memberRepo:        memberRepo,
		customerRepo:      customerRepo,
		brandItemRepo:     brandItemRepo,
		feedbackRepo:      feedbackRepo,
		userBriefContract: userBriefContract,
		fashionContract:   fashionContract,
		eventPublisher:    eventPublisher,
		mediaService:      mediaService,
		benefitUC:         benefitUC,
		uow:               uow,
	}
}
