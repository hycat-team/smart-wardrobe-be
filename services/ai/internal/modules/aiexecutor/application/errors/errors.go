package errors

import (
	"errors"
	"fmt"

	"smart-wardrobe-be/shared/application/constants/apperror"
)

// ErrObserveOnlyMissingAttemptCap is a CONFIGURATION fault, not a runtime
// failure. It is raised when an observe_only policy has a NULL
// max_paid_attempts_per_day.
//
// In observe_only there is no hard cost cap, so the per-day attempt limit is the
// only thing bounding the account. Treating NULL as "no limit" would let the
// free policy spend without any bound and no error would ever surface, so the
// paid route is refused and the caller escalates to fallback tier 2, which
// warns and runs the free model instead (FR-005e).
//
// A sentinel rather than a value error so the caller can branch on it with
// errors.Is and still log a precise message.
var ErrObserveOnlyMissingAttemptCap = errors.New("observe_only policy has no max_paid_attempts_per_day configured")

func ErrPricingConfigNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy cấu hình giá mô hình AI.")
}

func ErrEffectiveToBeforeEffectiveFrom() *apperror.Error {
	return apperror.NewBadRequest("Thời hạn hiệu lực kết thúc phải sau thời điểm bắt đầu hiệu lực.")
}

// ---------------------------------------------------------------------------
// 024 T187 — the error factory boundary for this module, stated once.
//
// Constitution §V requires business and system errors to be built by `apperror`
// factories rather than inline. The classification below is the rule this file
// applies, so the next refactor does not have to re-derive it per call site:
//
//	CLIENT-FACING — the caller supplied something the service refuses, and the
//	caller can read the refusal. → apperror.NewBadRequest / NewNotFound, message
//	in Vietnamese (constitution, Language Boundaries).
//
//	INTERNAL — stored data is missing or corrupt, configuration is absent, or a
//	driver failed. The caller cannot fix any of these by changing their request,
//	so telling them so would only invite them to retry something that will fail
//	identically. → apperror.NewInternalError, message in Vietnamese because it
//	can still be logged to an operator-facing sink.
//
//	KEEP fmt.Errorf — a programming error, i.e. an invariant this package itself
//	violates. Those are the only errors allowed to stay inline, and each one
//	carries a comment saying why.
//
// The distinction that matters in practice: an unclassified `fmt.Errorf` reaching
// the transport becomes a 5xx carrying English text. That is the failure mode
// T187 exists to remove, because it tells a caller who sent a bad value that the
// infrastructure is broken.
// ---------------------------------------------------------------------------

// ErrUnsupportedAIOperation is returned when a caller names an AI operation this
// service does not implement.
//
// This is the caller-facing case that was previously an inline
// `fmt.Errorf("unsupported operation: %s")`: it became a 5xx with an English
// string, so a client that asked for an operation that does not exist was told
// the server had failed. Nothing about it is a server fault — the request is
// simply asking for something the API does not offer.
//
// The operation is echoed back because the fix is to send a different one, and a
// caller who cannot see which value was rejected cannot find it.
func ErrUnsupportedAIOperation(operation string) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf(
		"Thao tác AI không được hỗ trợ: %q. Hãy kiểm tra lại mã thao tác gửi lên.", operation))
}

// ErrCostPolicyUnavailable is raised when a snapshot is requested for a cost
// policy row that was not supplied.
//
// Internal rather than bad request: the policy id arrives on an event the Account
// service published, and an absent row means this service's configuration or the
// event was wrong — not the caller's request. The use case that calls this already
// reports it through its own classified rejection reason.
func ErrCostPolicyUnavailable() *apperror.Error {
	return apperror.NewInternalError("Không đọc được cấu hình chính sách phí AI.")
}

// ErrPolicySnapshotUnreadable is raised when a stored policy snapshot cannot be
// turned back into entitlements.
//
// Internal: the snapshot was written by this service, so an unreadable one is
// corrupt stored data. The reader treats it as "entitlements unavailable" and
// degrades rather than failing the request — which is correct, and is why the
// message only ever reaches a log.
func ErrPolicySnapshotUnreadable() *apperror.Error {
	return apperror.NewInternalError("Không đọc được ảnh chụp chính sách quyền tính phí AI đã lưu.")
}
