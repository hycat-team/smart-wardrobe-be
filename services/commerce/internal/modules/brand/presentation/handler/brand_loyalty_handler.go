package handler

import (
	usecase_interfaces "smart-wardrobe-be/services/commerce/internal/modules/brand/application/interface/usecase"
)

// BrandLoyaltyHandler serves brand loyalty HTTP endpoints across member,
// points, program, benefit and claim token concerns; the method set lives in
// the brand_loyalty_*_handler.go files of this package.
type BrandLoyaltyHandler struct {
	brandUC   usecase_interfaces.IBrandUseCase
	loyaltyUC usecase_interfaces.IBrandLoyaltyUseCase
	benefitUC usecase_interfaces.IBrandBenefitUseCase
	claimUC   usecase_interfaces.IBrandClaimUseCase
}

func NewBrandLoyaltyHandler(
	brandUC usecase_interfaces.IBrandUseCase,
	loyaltyUC usecase_interfaces.IBrandLoyaltyUseCase,
	benefitUC usecase_interfaces.IBrandBenefitUseCase,
	claimUC usecase_interfaces.IBrandClaimUseCase,
) *BrandLoyaltyHandler {
	return &BrandLoyaltyHandler{
		brandUC:   brandUC,
		loyaltyUC: loyaltyUC,
		benefitUC: benefitUC,
		claimUC:   claimUC,
	}
}
