package usecase

import (
	brandContract "smart-wardrobe-be/internal/modules/brand/presentation/contract"
	uc_interfaces "smart-wardrobe-be/internal/modules/dashboard/application/interface/usecase"
	dashboardRepos "smart-wardrobe-be/internal/modules/dashboard/domain/repositories"
)

type DynamicQueryUseCase struct {
	dashboardRepo dashboardRepos.IDashboardRepository
	brandReader   brandContract.IBrandDashboardReader
}

func NewDynamicQueryUseCase(
	dashboardRepo dashboardRepos.IDashboardRepository,
	brandReader brandContract.IBrandDashboardReader,
) uc_interfaces.IDynamicQueryUseCase {
	return &DynamicQueryUseCase{
		dashboardRepo: dashboardRepo,
		brandReader:   brandReader,
	}
}
