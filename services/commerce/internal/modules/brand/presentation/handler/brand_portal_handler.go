package handler

import (
	usecase_interfaces "smart-wardrobe-be/services/commerce/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/shared/application/sse"
)

type BrandPortalHandler struct {
	brandUC    usecase_interfaces.IBrandUseCase
	itemUC     usecase_interfaces.IBrandItemUseCase
	sseManager sse.ISSEManager
}

func NewBrandPortalHandler(
	brandUC usecase_interfaces.IBrandUseCase,
	itemUC usecase_interfaces.IBrandItemUseCase,
	sseManager sse.ISSEManager,
) *BrandPortalHandler {
	return &BrandPortalHandler{
		brandUC:    brandUC,
		itemUC:     itemUC,
		sseManager: sseManager,
	}
}
