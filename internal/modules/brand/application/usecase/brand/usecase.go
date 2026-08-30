package brand

import (
	"smart-wardrobe-be/internal/config"
	brand_caching "smart-wardrobe-be/internal/modules/brand/application/interface/caching"
	uc_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/brand/domain/repositories"
	identity_contract "smart-wardrobe-be/internal/modules/identity/presentation/contract"
	"smart-wardrobe-be/internal/shared/application/media"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
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
