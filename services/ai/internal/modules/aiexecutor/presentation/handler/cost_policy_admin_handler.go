package handler

import (
	"fmt"

	"smart-wardrobe-be/services/ai/dto/aiexecutor"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/usecase/admin"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	"smart-wardrobe-be/shared/pkg/logger"
	"smart-wardrobe-be/shared/pkg/utils/contextutils"
	"smart-wardrobe-be/shared/pkg/utils/validation"
	shared_pres "smart-wardrobe-be/shared/presentation"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Client-facing text of the gate-2 admin endpoints. Kept as constants at the
// top of the file per docs/system-design/design-pattern.md rather than inlined.
const (
	successAdminListCostPolicies      = "Danh sách chính sách chi phí AI"
	successAdminUpdateCostPolicyOpMsg = "Đã cập nhật giới hạn lượt của thao tác %s"
)

type adminCostPolicyOperationUri struct {
	ID        string `uri:"id" binding:"required" label:"mã cấu hình thao tác"`
	Operation string `uri:"operation" binding:"required,oneof=chat outfit summary rewriter" label:"tên thao tác"`
}

// CostPolicyAdminHandler serves the admin gate-2 endpoints. The routes live
// inside the existing /admin/ai group, so they inherit r.auth.Handle() and
// RolesAuthorize(roleslug.Admin) — no separate authentication path is created
// (024 FR-004b, contracts/admin-cost-policy-api.md §1).
type CostPolicyAdminHandler struct {
	logger logger.Interface
	costUC usecase.IAICostPolicyAdminUseCase
}

func NewCostPolicyAdminHandler(logger logger.Interface, costUC usecase.IAICostPolicyAdminUseCase) *CostPolicyAdminHandler {
	return &CostPolicyAdminHandler{logger: logger, costUC: costUC}
}

// ListCostPolicies
// @Summary Danh sách chính sách chi phí AI (cổng 2)
// @Description Quản trị viên xem cổng 2 của từng gói: chế độ thi hành, trần chi phí và giới hạn lượt của từng thao tác.
// @Tags Admin AI
// @Produce json
// @Success 200 {object} shared_pres.APIResponse{data=[]dto.AICostPolicyDTO}
// @Router /api/v1/admin/ai/cost-policies [get]
func (h *CostPolicyAdminHandler) ListCostPolicies(c *gin.Context) error {
	policies, err := h.costUC.ListCostPolicies(c.Request.Context())
	if err != nil {
		return err
	}
	shared_pres.Success(c, successAdminListCostPolicies, policies)
	return nil
}

// UpdateCostPolicyOperation
// @Summary Cập nhật giới hạn lượt của một thao tác AI (cổng 2)
// @Description Sửa maxPaidAttemptsPerDay. Trường này thuộc nhóm quyền lợi đã đóng băng nên cần phát lại tín hiệu (FR-029c) mới có tác dụng.
// @Tags Admin AI
// @Accept json
// @Produce json
// @Param id path string true "Operation ID" format(uuid)
// @Param operation path string true "Tên thao tác" Enums(outfit, chat, rewriter, summary)
// @Param body body dto.UpdateCostPolicyOperationReq true "Thông tin cập nhật"
// @Success 200 {object} shared_pres.APIResponse{data=dto.UpdateCostPolicyOperationResp}
// @Router /api/v1/admin/ai/cost-policies/{id}/operations/{operation} [put]
func (h *CostPolicyAdminHandler) UpdateCostPolicyOperation(c *gin.Context) error {
	var uri adminCostPolicyOperationUri
	if err := validation.BindURI(c, &uri); err != nil {
		return err
	}
	// A non-UUID id is a caller mistake and must surface as 400, not as a
	// database "invalid input syntax for uuid" wrapped in a 500.
	operationID, err := uuid.Parse(uri.ID)
	if err != nil {
		return err
	}

	var req dto.UpdateCostPolicyOperationReq
	if err := validation.BindJSON(c, &req); err != nil {
		return err
	}

	actorUserID, _ := contextutils.GetUserId(c)
	actor := admin.CostPolicyActor{UserID: actorUserID, RequestIP: c.ClientIP()}

	updated, err := h.costUC.UpdateCostPolicyOperation(
		c.Request.Context(),
		operationID,
		aioperation.AIOperation(uri.Operation),
		req,
		actor,
	)
	if err != nil {
		return err
	}

	shared_pres.Success(c, fmt.Sprintf(successAdminUpdateCostPolicyOpMsg, updated.Operation), updated)
	return nil
}
