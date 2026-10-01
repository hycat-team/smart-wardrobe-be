package admin

import (
	"context"
	"fmt"
	"time"

	aiexecutor_dto "smart-wardrobe-be/services/ai/dto/aiexecutor"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/interface/usecase"
	aiexecutor_mapper "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/application/mapper"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aienforcementmode"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/constants/aioperation"
	aiexecutor_entities "smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/entities"
	"smart-wardrobe-be/services/ai/internal/modules/aiexecutor/domain/repositories"
	"smart-wardrobe-be/shared/application/constants/apperror"
	"smart-wardrobe-be/shared/application/constants/eventconstants"
	sharedevent "smart-wardrobe-be/shared/application/event"
	shared_repos "smart-wardrobe-be/shared/domain/repositories"
	"smart-wardrobe-be/shared/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Cost policy field names as they appear in the audit trail and in the
// admin.audit.log payload. Referenced by name rather than inlined so the two
// records can never disagree.
const (
	costPolicyFieldMaxPaidAttemptsPerDay = "maxPaidAttemptsPerDay"
)

// CostPolicyActor is re-exported from the dto package so existing callers of
// this package keep a single obvious name for the request actor.
type CostPolicyActor = aiexecutor_dto.CostPolicyActor

// AICostPolicyAdminUseCase serves the admin gate-2 API: viewing and editing the
// per-day paid attempt limit that the AI engine actually enforces
// (024 FR-004, FR-004b).
type AICostPolicyAdminUseCase struct {
	aiCostRepo repositories.IAICostRepository
	auditRepo  repositories.IAICostPolicyAuditRepository
	publisher  sharedevent.IEventPublisher
	uow        shared_repos.IUnitOfWork
	logger     logger.Interface
}

func NewAICostPolicyAdminUseCase(
	aiCostRepo repositories.IAICostRepository,
	auditRepo repositories.IAICostPolicyAuditRepository,
	publisher sharedevent.IEventPublisher,
	uow shared_repos.IUnitOfWork,
	logger logger.Interface,
) usecase.IAICostPolicyAdminUseCase {
	return &AICostPolicyAdminUseCase{
		aiCostRepo: aiCostRepo,
		auditRepo:  auditRepo,
		publisher:  publisher,
		uow:        uow,
		logger:     logger,
	}
}

// ListCostPolicies returns every cost policy with its four operations and its
// enforcement mode, so an administrator can see gate 2 without opening a SQL
// client (contracts/admin-cost-policy-api.md §3).
func (uc *AICostPolicyAdminUseCase) ListCostPolicies(ctx context.Context) ([]*aiexecutor_dto.AICostPolicyDTO, error) {
	policies, err := uc.aiCostRepo.ListPolicies(ctx)
	if err != nil {
		return nil, err
	}

	res := make([]*aiexecutor_dto.AICostPolicyDTO, 0, len(policies))
	for _, p := range policies {
		res = append(res, aiexecutor_mapper.CostPolicyToDTO(p))
	}
	return res, nil
}

// UpdateCostPolicyOperation changes the per-day paid attempt limit of one
// operation. Five rejections are mandatory (FR-004b, FR-005e) and none of them may be
// reported as 500 — every one is a caller mistake the administrator can fix:
//
//  1. A value on a policy in strict mode, where the column is never read (T173).
//  2. NULL requested while the policy is in observe_only.
//  3. Switching to strict while the hard cap is empty (defensive: this endpoint
//     edits the operation row, but a policy left in strict with no cap would
//     enforce nothing at all).
//  4. Unknown operation id / unsupported operation -> 404.
//  5. Contradictory body (value AND clearMaxPaidAttempts) -> 400, handled by
//     the handler's binding layer.
//
// The write and the audit record share one transaction (FR-004c): a committed
// configuration change with no audit event — or an audit row with no event —
// is a half state neither screen can reconcile.
func (uc *AICostPolicyAdminUseCase) UpdateCostPolicyOperation(
	ctx context.Context,
	operationID uuid.UUID,
	operation aioperation.AIOperation,
	req aiexecutor_dto.UpdateCostPolicyOperationReq,
	actor CostPolicyActor,
) (*aiexecutor_dto.UpdateCostPolicyOperationResp, error) {
	desired, err := resolveDesiredMaxPaidAttempts(req)
	if err != nil {
		return nil, err
	}

	op, policy, err := uc.aiCostRepo.GetPolicyOperationByID(ctx, operationID)
	if err != nil {
		return nil, err
	}
	if op == nil || policy == nil {
		return nil, apperror.NewNotFound(fmt.Sprintf(
			"Không tìm thấy cấu hình thao tác với mã %s, hoặc thao tác %s không được hỗ trợ.",
			operationID, operation))
	}
	if op.Operation != operation {
		return nil, apperror.NewNotFound(fmt.Sprintf(
			"Cấu hình với mã %s không thuộc thao tác %s.", operationID, operation))
	}
	if err := guardMaxPaidAttemptsWrite(policy, desired); err != nil {
		return nil, err
	}

	oldValue := op.MaxPaidAttemptsPerDay
	if maxPaidAttemptsUnchanged(oldValue, desired) {
		// Nothing to do. Returning a 200 with the current values would hide the
		// fact that no change was recorded, so the caller is told explicitly.
		return nil, apperror.NewConflict("Giá trị giới hạn lượt đã đúng bằng giá trị yêu cầu, không có gì để cập nhật.")
	}

	op.MaxPaidAttemptsPerDay = desired
	if err := uc.uow.Execute(ctx, func(txCtx context.Context) error {
		if err := uc.aiCostRepo.UpdatePolicyOperation(txCtx, op); err != nil {
			return err
		}
		if err := uc.auditRepo.Create(txCtx, buildCostPolicyAuditLog(policy, op, oldValue, desired, actor)); err != nil {
			return err
		}
		// Publishing inside the same transaction is deliberate: if the outbox
		// write fails, the whole thing rolls back and the administrator sees an
		// error instead of a configuration change the central audit screen
		// never learns about.
		return uc.publisher.Publish(txCtx, eventconstants.TopicAuditLog, buildCostPolicyAuditEvent(policy, op, oldValue, desired, actor))
	}); err != nil {
		return nil, err
	}

	// 024 T176: the transaction is already committed, and both required records exist
	// inside it (FR-004c). Failing here used to return the error, so the administrator
	// got a 5xx for a change that HAD been applied — telling them to retry a write that
	// had succeeded, and leaving them unsure whether the audit trail recorded it.
	//
	// Counting is reporting, not part of the write. The count only decorates the
	// warning that tells the administrator a replay is needed, so a failure to read it
	// must not change the outcome of the write. The alternative — pretending there is
	// nothing frozen — is worse: it would tell them no replay is needed when one is.
	//
	// So the change reports success, the missing counts are logged, and the warning
	// still says a replay is required while admitting the numbers are unavailable.
	total, openEnded, countErr := uc.aiCostRepo.CountGrants(ctx)
	if countErr != nil {
		uc.logger.Error("cost policy operation updated, but the affected-grant count could not be read",
			zap.String("policy_code", policy.Code),
			zap.String("operation", string(op.Operation)),
			zap.Error(countErr),
		)
		return &aiexecutor_dto.UpdateCostPolicyOperationResp{
			PolicyCode:            policy.Code,
			Operation:             string(op.Operation),
			MaxPaidAttemptsPerDay: desired,
			Warning:               buildUncountedReplayWarning(),
		}, nil
	}

	return &aiexecutor_dto.UpdateCostPolicyOperationResp{
		PolicyCode:            policy.Code,
		Operation:             string(op.Operation),
		MaxPaidAttemptsPerDay: desired,
		Warning:               buildReplayWarning(total, openEnded),
	}, nil
}

// resolveDesiredMaxPaidAttempts turns the request into the value that should
// end up in the column, or fails when the caller is contradicting themselves.
func resolveDesiredMaxPaidAttempts(req aiexecutor_dto.UpdateCostPolicyOperationReq) (*int, error) {
	hasValue := req.MaxPaidAttemptsPerDay != nil
	hasClear := req.ClearMaxPaidAttempts
	switch {
	case hasValue && hasClear:
		// Without this check the caller's intent is undetermined: NULL and a
		// number are opposite instructions arriving in the same body.
		return nil, apperror.NewBadRequest(
			"Không thể gửi đồng thời maxPaidAttemptsPerDay và clearMaxPaidAttempts. Hãy chọn một trong hai.")
	case hasClear:
		return nil, nil
	case hasValue:
		value := *req.MaxPaidAttemptsPerDay
		if value < 0 {
			return nil, apperror.NewBadRequest("Giới hạn lượt trả phí mỗi ngày không được âm.")
		}
		return &value, nil
	default:
		return nil, apperror.NewBadRequest(
			"Yêu cầu phải có maxPaidAttemptsPerDay, hoặc cờ clearMaxPaidAttempts để đặt thành NULL.")
	}
}

// guardMaxPaidAttemptsWrite rejects the states that would silently break cost
// control (FR-005e, contracts/admin-cost-policy-api.md §5).
//
// 024 T173: a non-NULL value on a STRICT policy is refused. It used to return nil
// immediately for any value, so the endpoint accepted a number into a column that
// strict mode never reads.
//
// FR-005e calls that column DEAD CONFIG for a strict policy and settled on NULL as the
// honest encoding, precisely so that nobody reading the configuration table would
// conclude Premium is limited to N chats a day. Accepting the value quietly undoes
// that: the number is stored, looks authoritative, and enforces nothing. It is worse
// than a no-op, because the operator believes a limit is in force.
//
// Refused rather than warned. A warning that still writes the value leaves the
// misleading number in the table, which is the failure FR-005e describes; and there is
// no legitimate reason to set it, since strict mode compares cost against the cap and
// never reads a daily attempt count.
func guardMaxPaidAttemptsWrite(policy *aiexecutor_entities.AICostPolicy, desired *int) error {
	if desired != nil {
		if policy.EnforcementMode == aienforcementmode.Strict {
			return apperror.NewConflict(
				"Chính sách đang ở chế độ strict, nơi giới hạn lượt trả phí mỗi ngày KHÔNG được đọc — " +
					"chế độ này chỉ so tổng tiền với trần chi phí cứng. Số lượt đặt vào đây sẽ không có tác dụng gì " +
					"và gây hiểu nhầm rằng gói bị giới hạn. Muốn giới hạn lượt, hãy chuyển chính sách sang chế độ observe_only.")
		}
		return nil
	}
	if policy.EnforcementMode == aienforcementmode.ObserveOnly {
		return apperror.NewConflict(
			"Chính sách đang ở chế độ chỉ quan sát (observe_only), nơi giới hạn lượt là cơ chế chặn duy nhất. " +
				"Đặt NULL sẽ tắt hẳn cơ chế đếm lượt, nên chỉ chính sách ở chế độ strict mới được phép đặt NULL.")
	}
	if policy.HardCostMicroVND == nil {
		return apperror.NewConflict(
			"Chính sách chưa có trần chi phí cứng (hard_cost_micro_vnd còn trống). " +
				"Ở chế độ strict, không có trần thì cơ chế chặn sẽ không có tác dụng gì.")
	}
	return nil
}

func maxPaidAttemptsUnchanged(current, desired *int) bool {
	if current == nil || desired == nil {
		return current == nil && desired == nil
	}
	return *current == *desired
}

// buildReplayWarning states the consequence an administrator must know about:
// maxPaidAttemptsPerDay is frozen into the grant snapshot at grant time
// (FR-029b), so editing the column does not change what existing accounts are
// limited to. openGrantCount is the group that will never refresh on its own
// because a row with effective_to IS NULL is only recreated when the account
// changes plan.
func buildReplayWarning(total, openEnded int64) *aiexecutor_dto.CostPolicyReplayWarning {
	return &aiexecutor_dto.CostPolicyReplayWarning{
		ReplayRequired:   true,
		FrozenGrantCount: total,
		OpenGrantCount:   openEnded,
		Message: fmt.Sprintf(
			"Trường này thuộc nhóm quyền lợi đã đóng băng trong ảnh chụp. %d bản ghi mức quyền hiện có không tự làm mới, "+
				"trong đó %d bản ghi effectiveTo = null sẽ không bao giờ tự làm mới. Cần chạy phát lại tín hiệu theo FR-029c.",
			total, openEnded),
	}
}

// buildUncountedReplayWarning is buildReplayWarning for the case where the counts could
// not be read (024 T176).
//
// ReplayRequired stays TRUE and the counts stay ZERO. Zero is the honest value for
// "not read", because the alternative readings are both wrong: false would tell the
// administrator no replay is needed when one is, and omitting the warning entirely
// would drop FR-029c's instruction altogether. The message says which number is
// missing so nobody plans the replay around a fabricated figure.
func buildUncountedReplayWarning() *aiexecutor_dto.CostPolicyReplayWarning {
	return &aiexecutor_dto.CostPolicyReplayWarning{
		ReplayRequired: true,
		Message: "Trường này thuộc nhóm quyền lợi đã đóng băng trong ảnh chụp, nên vẫn cần phát lại " +
			"tín hiệu theo FR-029c. Số bản ghi mức quyền bị đóng băng và số bản ghi effectiveTo = null " +
			"không đọc được do lỗi khi báo cáo, nên hai số này là 0 chứ không phải số thật — " +
			"hãy chạy tệp kiểm chứng FR-090 để lấy con số chính xác.",
	}
}

func buildCostPolicyAuditLog(
	policy *aiexecutor_entities.AICostPolicy,
	op *aiexecutor_entities.AICostPolicyOperation,
	oldValue, newValue *int,
	actor CostPolicyActor,
) *aiexecutor_entities.AICostPolicyAuditLog {
	return &aiexecutor_entities.AICostPolicyAuditLog{
		ActorUserID: actor.UserID,
		ChangedAt:   time.Now().UTC(),
		RequestIP:   actor.RequestIP,
		PolicyCode:  policy.Code,
		Operation:   string(op.Operation),
		FieldName:   costPolicyFieldMaxPaidAttemptsPerDay,
		OldValue:    formatMaxPaidAttempts(oldValue),
		NewValue:    formatMaxPaidAttempts(newValue),
		Action:      string(adminauditAICostPolicyUpdate),
	}
}
