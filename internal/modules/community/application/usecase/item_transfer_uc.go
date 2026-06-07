package usecase

import (
	uc_interfaces "smart-wardrobe-be/internal/modules/community/application/interface/usecase"
	"smart-wardrobe-be/internal/modules/community/domain/repositories"
	identity_contract "smart-wardrobe-be/internal/modules/identity/contract"
	wardrobe_contract "smart-wardrobe-be/internal/modules/wardrobe/contract"
	shared_repos "smart-wardrobe-be/internal/shared/domain/repositories"
)

type ItemTransferUseCase struct {
	postRepo            repositories.IPostRepository
	postItemRepo        repositories.IPostItemRepository
	transferRequestRepo repositories.ITransferRequestRepository
	identityCtr         identity_contract.IUserContract
	wardrobeCtr         wardrobe_contract.IWardrobeContract
	uow                 shared_repos.IUnitOfWork
}

func NewItemTransferUseCase(
	postRepo repositories.IPostRepository,
	postItemRepo repositories.IPostItemRepository,
	transferRequestRepo repositories.ITransferRequestRepository,
	identityCtr identity_contract.IUserContract,
	wardrobeCtr wardrobe_contract.IWardrobeContract,
	uow shared_repos.IUnitOfWork,
) uc_interfaces.IItemTransferUseCase {
	return &ItemTransferUseCase{
		postRepo:            postRepo,
		postItemRepo:        postItemRepo,
		transferRequestRepo: transferRequestRepo,
		identityCtr:         identityCtr,
		wardrobeCtr:         wardrobeCtr,
		uow:                 uow,
	}
}
