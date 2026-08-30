package handler

import (
	usecase_interfaces "smart-wardrobe-be/internal/modules/brand/application/interface/usecase"
	"smart-wardrobe-be/internal/shared/application/sse"
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
