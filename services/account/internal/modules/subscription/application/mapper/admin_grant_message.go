package mapper

import "fmt"

// MaxIdentifierLength bounds an administrator-supplied identifier (024 FR-062).
//
// It lives here, next to the message that quotes it, so the limit and the
// wording an administrator reads cannot drift apart. It matches the width of
// `identity.users.username` and `.email`.
const MaxIdentifierLength = 255

// MaxPlanSlugLength bounds an administrator-supplied plan slug.
//
// It matches the width of `subscription.subscription_plans.slug`, which is 100 -
// tighter than the identifier limit, and it is checked for the same reason
// FR-062 gives: the slug reaches `WHERE slug = $1`, and a value far longer than
// the column cannot use the index. FR-062 scopes its limit to identifiers only,
// so this bound is an addition, not an interpretation of it.
const MaxPlanSlugLength = 100

// Client-facing text of the admin grant API (024 FR-085).
//
// Kept as constants at the top of the file rather than inlined, per
// docs/system-design/design-pattern.md, and so the wording is reviewable in one
// place. Every message names the specific cause: an administrator told only
// "operation failed" has to guess which of ten rules rejected the line.
const (
	MsgAdminGrantSuccess             = "Đã cấp gói %s cho tài khoản, hiệu lực đến %s."
	MsgAdminGrantNoExpiry            = "Đã cấp gói %s cho tài khoản, không có thời hạn."
	MsgAdminGrantIdentifierBlank     = "Mã định danh tài khoản bị trống."
	MsgAdminGrantUserNotFound        = "Không tìm thấy tài khoản khớp với mã định danh đã cung cấp."
	MsgAdminGrantIdentifierAmbiguous = "Mã định danh khớp với nhiều tài khoản. Hãy dùng UUID hoặc email chính xác hơn."
	MsgAdminGrantPlanNotFound        = "Không tìm thấy gói có mã đã cung cấp."
	// 024 T129. Deliberately says nothing about WHY it failed: the cause is either
	// an operator error (which they would repeat) or an infrastructure fault (which
	// they cannot fix), and guessing wrong in a customer-facing message is worse
	// than saying "not this specific cause".
	MsgAdminGrantFailed             = "Không thể cấp gói cho tài khoản này. Vui lòng thử lại sau hoặc liên hệ quản trị hệ thống."
	MsgAdminGrantPlanInactive       = "Gói đã bị vô hiệu hoá, không thể cấp."
	MsgAdminGrantTargetIsAdmin      = "Không thể cấp gói cho tài khoản quản trị viên."
	MsgAdminGrantUserSoftDeleted    = "Tài khoản đã bị xoá mềm nên không thể cấp gói."
	MsgAdminGrantDuplicateInRequest = "Mã định danh đã xuất hiện trước đó trong cùng danh sách. Chỉ phần tử đầu tiên được xử lý."
	MsgAdminGrantDuplicateRequest   = "Yêu cầu này đã được xử lý với khoá chống gửi lại này. Thời hạn không thay đổi."
	MsgAdminGrantConcurrentModif    = "Tài khoản vừa được đổi gói bởi một thao tác khác. Hãy thử lại."
)

// 024 T178: `MsgAdminGrantInternal` is GONE. Nothing referenced it: the catch-all
// `grant_failed` reason reports `MsgAdminGrantFailed`, and every other failure carries
// a reason-specific message. An unused message is a string a reader assumes is shown
// somewhere, and "Không thể cấp gói cho tài khoản này do lỗi hệ thống" is exactly the
// vague text T129 replaced with reasons an operator can act on.

// Restore tool text (024 FR-089).
const (
	MsgRestoreDryRun           = "Đã chạy thử khôi phục mức quyền tính phí AI. Chưa thay đổi dữ liệu."
	MsgRestoreApplied          = "Đã khôi phục mức quyền tính phí AI."
	MsgRestorePlanNotInCatalog = "Gói trong bản ghi đăng ký không có trong danh mục nên không thể dựng lại mức quyền."
	MsgRestoreExpired          = "Gói đã hết hạn, được xử lý bởi luồng gia hạn hoặc hạ cấp."
	MsgRestorePublishFailed    = "Không thể phát tín hiệu cho tài khoản này."
	MsgRestoreAlreadyRestored  = "Tài khoản này đã được khôi phục ở một lần chạy trước."
)

// RestoreMessage returns the Vietnamese text for one restore reason code.
// A lookup table rather than an inline switch at the call site, so each reason
// has exactly one wording and that wording is edited in one place. An empty
// result means "no message needed", which is the case for a success.
func RestoreMessage(reason string) string {
	switch reason {
	case "plan_not_in_catalog":
		return MsgRestorePlanNotInCatalog
	case "subscription_expired":
		return MsgRestoreExpired
	case "publish_failed":
		return MsgRestorePublishFailed
	case "already_restored":
		return MsgRestoreAlreadyRestored
	default:
		return ""
	}
}

// MsgAdminGrantIdentifierTooLong is built rather than declared as a constant so
// the number it quotes IS the limit. A hand-written "255" here could drift from
// MaxIdentifierLength and an administrator would be told about a limit the
// system does not enforce.
//
// It names the limit explicitly: "quá dài" without a number leaves the
// administrator guessing, and the natural guess is to shorten a valid username
// rather than to find the right account.
var MsgAdminGrantIdentifierTooLong = fmt.Sprintf(
	"Mã định danh dài quá %d ký tự nên không thể tra cứu. Vui lòng dùng UUID, tên đăng nhập hoặc email.",
	MaxIdentifierLength,
)

// MsgAdminGrantPlanSlugTooLong is built rather than declared as a constant so
// the number it quotes IS the limit, exactly as with the identifier message.
//
// It names the limit: an administrator told only "quá dài" has to guess what the
// bound is, and the natural guess - shorten a real slug - makes things worse.
var MsgAdminGrantPlanSlugTooLong = fmt.Sprintf(
	"Mã gói dài quá %d ký tự nên không thể tra cứu. Vui lòng dùng mã gói đúng.",
	MaxPlanSlugLength,
)

// MsgAdminGrantBatchTimeout is FR-076's time budget running out (024 T143).
//
// The seconds are spelled out rather than derived from the use case's constants
// because the mapper package must not import the use case package - that would be
// a cycle, since the use case already imports the mapper for its other messages.
// `TestMsgAdminGrantBatchTimeoutMatchesTheEnforcedBudget` in the use case package
// is what keeps the two copies from drifting.
//
// It tells the administrator to RESUBMIT with the SAME Idempotency-Key: that is
// what makes the already-granted lines no-ops instead of a second set of grants
// (FR-068). Resubmitting with a new key would double-extend every account the
// first attempt already completed.
var MsgAdminGrantBatchTimeout = fmt.Sprintf(
	"Không đủ thời gian xử lý danh sách (giới hạn %d giây cho một phần tử, %d giây cho danh sách). "+
		"Phần tử đã cấp vẫn giữ nguyên. Vui lòng gửi lại với CÙNG khoá Idempotency-Key để tiếp tục.",
	5, 60,
)
