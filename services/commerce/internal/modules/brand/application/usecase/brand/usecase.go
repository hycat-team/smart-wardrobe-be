package brand

import (
	identity_contract "smart-wardrobe-be/services/account/contract/identity"
	brand_caching "smart-wardrobe-be/services/commerce/internal/modules/brand/application/interface/caching"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/repositories"
	"smart-wardrobe-be/shared/application/media"
	"smart-wardrobe-be/shared/config"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
)

type BrandUseCase struct {
	brandRepo             repositories.IBrandRepository
	memberRepo            repositories.IBrandMemberRepository
	customerRepo          repositories.IBrandCustomerRepository
	feedbackRepo          repositories.IDigitalSampleResponseRepository
	benefitRedemptionRepo repositories.IBenefitRedemptionRepository
	userBriefContract     identity_contract.IUserBriefContract
	mediaService          media.IMediaService
	uow                   shared_repos.IUnitOfWork
	cfg                   *config.Config
	membershipCache       brand_caching.IBrandMembershipCache
}

func NewBrandUseCase(
	brandRepo repositories.IBrandRepository,
	memberRepo repositories.IBrandMemberRepository,
	customerRepo repositories.IBrandCustomerRepository,
	feedbackRepo repositories.IDigitalSampleResponseRepository,
	benefitRedemptionRepo repositories.IBenefitRedemptionRepository,
	userBriefContract identity_contract.IUserBriefContract,
	mediaService media.IMediaService,
	uow shared_repos.IUnitOfWork,
	cfg *config.Config,
	membershipCache brand_caching.IBrandMembershipCache,
) uc_interfaces.IBrandUseCase {
	return &BrandUseCase{
		brandRepo:             brandRepo,
		memberRepo:            memberRepo,
		customerRepo:          customerRepo,
		feedbackRepo:          feedbackRepo,
		benefitRedemptionRepo: benefitRedemptionRepo,
		userBriefContract:     userBriefContract,
		mediaService:          mediaService,
		uow:                   uow,
		cfg:                   cfg,
		membershipCache:       membershipCache,
	}
}
