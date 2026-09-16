package loyalty

import (
	identity_contract "smart-wardrobe-be/services/account/contract/identity"
	uc_interfaces "smart-wardrobe-be/services/commerce/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/services/commerce/internal/modules/brand/domain/repositories"
	"smart-wardrobe-be/shared/config"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/pkg/logger"
)

type BrandLoyaltyUseCase struct {
	brandRepo         repositories.IBrandRepository
	memberRepo        repositories.IBrandMemberRepository
	customerRepo      repositories.IBrandCustomerRepository
	userBriefContract identity_contract.IUserBriefContract
	programRepo       repositories.ILoyaltyProgramRepository
	tierRepo          repositories.ILoyaltyTierRepository
	accountRepo       repositories.ILoyaltyAccountRepository
	txRepo            repositories.ILoyaltyPointTransactionRepository
	lotRepo           repositories.ILoyaltyPointLotRepository
	benefitRepo       repositories.IBrandBenefitRepository
	uow               shared_repos.IUnitOfWork
	cfg               *config.Config
	logger            logger.Interface
}

func NewBrandLoyaltyUseCase(
	brandRepo repositories.IBrandRepository,
	memberRepo repositories.IBrandMemberRepository,
	customerRepo repositories.IBrandCustomerRepository,
	userBriefContract identity_contract.IUserBriefContract,
	programRepo repositories.ILoyaltyProgramRepository,
	tierRepo repositories.ILoyaltyTierRepository,
	accountRepo repositories.ILoyaltyAccountRepository,
	txRepo repositories.ILoyaltyPointTransactionRepository,
	lotRepo repositories.ILoyaltyPointLotRepository,
	benefitRepo repositories.IBrandBenefitRepository,
	uow shared_repos.IUnitOfWork,
	cfg *config.Config,
	l logger.Interface,
) uc_interfaces.IBrandLoyaltyUseCase {
	return &BrandLoyaltyUseCase{
		brandRepo:         brandRepo,
		memberRepo:        memberRepo,
		customerRepo:      customerRepo,
		userBriefContract: userBriefContract,
		programRepo:       programRepo,
		tierRepo:          tierRepo,
		accountRepo:       accountRepo,
		txRepo:            txRepo,
		lotRepo:           lotRepo,
		benefitRepo:       benefitRepo,
		uow:               uow,
		cfg:               cfg,
		logger:            l,
	}
}
