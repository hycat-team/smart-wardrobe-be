package dto

import (
	shared_dto "smart-wardrobe-be/internal/shared/application/dto"
	"smart-wardrobe-be/internal/shared/domain/constants/brand/brandstatus"
)

type UploadSignatureResult = shared_dto.UploadSignatureResult

type GetBrandsAdminQueryReq struct {
	shared_dto.PaginationQuery
	Status *brandstatus.BrandStatus `form:"status" binding:"omitempty" label:"trạng thái"`
	Query  *string                  `form:"q" binding:"omitempty" label:"từ khóa tìm kiếm"`
}

type GetActiveBrandsQueryReq struct {
	shared_dto.PaginationQuery
	Query *string `form:"q" binding:"omitempty" label:"từ khóa tìm kiếm"`
}

type AdminBrandListRes = shared_dto.PaginationResult[*BrandRes]
type PublicBrandListRes = shared_dto.PaginationResult[*BrandRes]

type GetBrandCustomersQueryReq struct {
	shared_dto.PaginationQuery
	Query  *string `form:"q" binding:"omitempty" label:"từ khóa tìm kiếm"`
	Status *string `form:"status" binding:"omitempty" label:"trạng thái"`
}

type BrandCustomerListRes = shared_dto.PaginationResult[*BrandCustomerRes]

type GetLoyaltyTransactionsQueryReq struct {
	shared_dto.PaginationQuery
}

type LoyaltyTransactionListRes = shared_dto.PaginationResult[*LoyaltyPointTransactionDetailRes]
