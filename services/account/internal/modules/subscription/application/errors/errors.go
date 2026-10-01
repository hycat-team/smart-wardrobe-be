package subscriptionerrors

import (
	"fmt"

	"smart-wardrobe-be/shared/application/constants/apperror"
	sharedmoney "smart-wardrobe-be/shared/domain/money"

	"github.com/google/uuid"
)

// Lỗi động (cần định dạng tham số)
func ErrPayosMinAmount(minAmount int64) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Số tiền thanh toán tối thiểu qua cổng PayOS là %d VNĐ.", minAmount))
}

func ErrTransactionNotFound(orderCode int64) *apperror.Error {
	return apperror.NewNotFound(fmt.Sprintf("Không tìm thấy giao dịch thanh toán với mã đơn hàng %d.", orderCode))
}

// Lỗi tĩnh
func ErrPayosMustBeInteger() *apperror.Error {
	return apperror.NewBadRequest("Số tiền thanh toán qua cổng PayOS phải là số tiền nguyên theo đơn vị VND.")
}

func ErrInvalidAmount() *apperror.Error {
	return apperror.NewBadRequest(sharedmoney.ErrInvalidAmount)
}

func ErrVerifySignatureFailed() *apperror.Error {
	return apperror.NewBadRequest("Không thể xác thực chữ ký giao dịch.")
}

func ErrWebhookPayloadMalformed() *apperror.Error {
	return apperror.NewBadRequest("Dữ liệu thông báo giao dịch không đúng định dạng.")
}

func ErrPaymentAmountMismatch() *apperror.Error {
	return apperror.NewBadRequest("Số tiền thanh toán không khớp với số tiền của giao dịch.")
}

func ErrProcessPaymentGatewayFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể xử lý thông tin cổng thanh toán.")
}

func ErrLockTransactionFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể khóa dữ liệu giao dịch.")
}

func ErrCompletePaymentRecordFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể hoàn tất hồ sơ giao dịch thanh toán.")
}

func ErrQueryTransactionFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể truy vấn thông tin giao dịch thanh toán.")
}

func ErrWalletNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy thông tin ví của tài khoản.")
}

func ErrInvalidDepositAmount() *apperror.Error {
	return apperror.NewBadRequest("Số tiền nạp không hợp lệ.")
}

func ErrDepositInitFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể khởi tạo giao dịch nạp tiền.")
}

func ErrQueryWalletBalanceFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể truy vấn thông tin số dư ví.")
}

func ErrWalletInsufficientBalance() *apperror.Error {
	return apperror.NewBadRequest("Số dư ví không đủ để thực hiện giao dịch.")
}

func ErrUserSubscriptionNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy gói hội viên của tài khoản.")
}

func ErrSubscriptionPlanNotFinite() *apperror.Error {
	return apperror.NewBadRequest("Chỉ gói hội viên có thời hạn mới có thể thiết lập tự động gia hạn.")
}

func ErrSubscriptionPlanNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy thông tin gói hội viên.")
}

func ErrDefaultPlanConfigNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy cấu hình cho gói hội viên mặc định.")
}

func ErrDefaultPlanLoadFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể tải thông tin cấu hình gói hội viên mặc định.")
}

func ErrQueryExpiredSubscriptionsFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể truy vấn danh sách gói hội viên hết hạn.")
}

func ErrRequestedPlanNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy gói hội viên được yêu cầu.")
}

func ErrFreePlanDirectPurchase() *apperror.Error {
	return apperror.NewBadRequest("Không thể đăng ký trực tiếp gói hội viên miễn phí")
}

func ErrDirectPurchaseCreateFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể tạo giao dịch thanh toán trực tiếp.")
}

func ErrSearchPlanFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể tìm kiếm thông tin gói cước.")
}

func ErrCurrentSubscriptionLoadFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể tải thông tin gói hội viên hiện tại.")
}

func ErrCurrentSubscriptionNotFound() *apperror.Error {
	return apperror.NewNotFound("Không tìm thấy gói hội viên hiện tại.")
}

func ErrActivateNewSubscriptionFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể kích hoạt gói hội viên mới.")
}

func ErrUpdateSubscriptionExpiryFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể cập nhật thời hạn gói hội viên.")
}

func ErrPlanInactive() *apperror.Error {
	return apperror.NewBadRequest("Gói hội viên này hiện đang tạm dừng hoạt động.")
}

func ErrPendingPaymentExists() *apperror.Error {
	return apperror.NewConflict("Bạn đang có giao dịch thanh toán chờ xử lý. Vui lòng hoàn tất giao dịch cũ hoặc đợi cho đến khi hết hạn.")
}

func ErrPaymentOrderConflict() *apperror.Error {
	return apperror.NewConflict("Không thể khởi tạo liên kết thanh toán do trùng mã đơn hàng với cổng ngân hàng. Vui lòng thử lại.")
}

func ErrAlreadyRegisteredUnlimitedPlan() *apperror.Error {
	return apperror.NewConflict("Bạn đã đăng ký gói hội viên không giới hạn thời gian này.")
}

func ErrSubscriptionStillActive() *apperror.Error {
	return apperror.NewBadRequest("Gói hội viên hiện tại của bạn vẫn còn hiệu lực. Vui lòng tắt chế độ tự động gia hạn nếu không có nhu cầu gia hạn tiếp.")
}

func ErrLinkedPlanNotFound() *apperror.Error {
	return apperror.NewBadRequest("Không tìm thấy thông tin gói hội viên liên kết với giao dịch.")
}

func ErrLoadPlanFailed() *apperror.Error {
	return apperror.NewInternalError("Không thể tải thông tin gói hội viên.")
}

func ErrInvalidManualPaymentResolution() *apperror.Error {
	return apperror.NewBadRequest("Không thể áp dụng quyết định xử lý thủ công vì trạng thái hoặc bằng chứng thanh toán không hợp lệ.")
}

func ErrAiOutfitQuotaExceeded() *apperror.Error {
	return apperror.NewBadRequest("Bạn đã dùng hết lượt tạo trang phục bằng AI trong hôm nay.")
}

func ErrAiChatQuotaExceeded() *apperror.Error {
	return apperror.NewBadRequest("Bạn đã dùng hết lượt trò chuyện với AI Chatbot trong hôm nay.")
}

func ErrPlanSlugAlreadyExists() *apperror.Error {
	return apperror.NewConflict("Mã định danh gói (slug) đã tồn tại trong hệ thống.")
}

// ErrAdminGrantIdempotencyKeyRequired (024 T088, FR-068).
//
// A batch grant has no natural duplicate key of its own: the administrator can
// legitimately grant the same account the same plan twice, so the request cannot
// be deduplicated by its contents. Without a caller-supplied key there is no way
// to tell a retry from a second intentional grant, and a network retry would
// quietly extend the subscription twice.
func ErrAdminGrantIdempotencyKeyRequired() *apperror.Error {
	return apperror.NewBadRequest("Thiếu tiêu đề Idempotency-Key. Vui lòng gửi khoá chống gửi lại để hệ thống nhận ra yêu cầu bị lặp.")
}

// ErrAdminGrantIdempotencyKeyTooLong (024, FR-068 + FR-085).
//
// The key becomes part of the event key `ADMIN_GRANT:{key}:{index}`, stored in a
// `character varying(255)` column. An over-long key used to be accepted by the API and
// only fail on INSERT, where it surfaced per item as `grant_failed` — the reason code
// that tells an administrator the infrastructure failed, for a request they got
// exactly right.
//
// The message gives the limit and tells them the fix, because the limit is a
// consequence of the design rather than an arbitrary policy: stating the number is
// the difference between a caller who shortens the key and one who gives up on the
// feature.
func ErrAdminGrantIdempotencyKeyTooLong(max, actual int) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf(
		"Khoá Idempotency-Key dài %d ký tự, vượt giới hạn %d ký tự. "+
			"Vì khoá này trở thành một phần của mã sự kiện nên bị giới hạn bởi độ dài cột lưu trữ. "+
			"Vui lòng dùng khoá ngắn hơn, ví dụ một UUID.",
		actual, max))
}

// ErrAdminGrantRateLimited (024 FR-078, FR-079).
//
// The limits are per administrator rather than per IP: on a shared office
// connection one administrator could otherwise exhaust the budget of every other
// administrator behind the same address, and a single administrator could keep
// working from a rotating pool of addresses.
func ErrAdminGrantRateLimited(retryAfterSeconds int) *apperror.Error {
	return apperror.NewTooManyRequest(fmt.Sprintf("Bạn đang thao tác quá nhanh. Vui lòng thử lại sau %d giây.", retryAfterSeconds))
}

// ErrAdminGrantRateLimitUnavailable is returned when the limiter itself cannot
// answer. A bulk write to customer entitlements is refused rather than performed
// without a limit: the limit is what keeps one administrator from locking the
// feature for everyone.
func ErrAdminGrantRateLimitUnavailable() *apperror.Error {
	return apperror.NewServiceUnavailable("Chưa thể kiểm tra giới hạn tần suất cấp gói. Vui lòng thử lại sau.")
}

// ErrAdminGrantEmptyBatch (024 FR-063).
func ErrAdminGrantEmptyBatch() *apperror.Error {
	return apperror.NewBadRequest("Danh sách cấp gói không được để trống.")
}

// ErrAdminGrantTooManyItems (024 FR-062).
//
// The wording matches contracts/admin-grant-api.md exactly. An administrator comparing
// what they see against the contract should not have to guess which of two similar
// sentences is the real one.
func ErrAdminGrantTooManyItems(max int) *apperror.Error {
	return apperror.NewBadRequest(fmt.Sprintf("Danh sách cấp gói tối đa %d tài khoản mỗi lần.", max))
}

// 024 T129: constitution §V requires business and system errors to be built by
// `apperror` factories rather than inline in the application layer. The three
// below replace the `errors.New` / `fmt.Errorf` calls that used to sit in the
// grant use case.

// ErrAdminGrantResolveUsersFailed wraps a failure of the one-pass identifier
// resolution.
//
// Internal, not bad request: the identifiers were validated as well-formed
// already, so a failure here means the identity contract or the database is
// unavailable, and saying so would invite the caller to "fix" their input.
func ErrAdminGrantResolveUsersFailed(cause error) *apperror.Error {
	e := apperror.NewInternalError("Không thể tra cứu danh sách tài khoản.")
	e.Cause = cause
	return e
}

// ErrAdminGrantRecordAuditFailed wraps a failure to write the audit record.
//
// Deliberately fatal to the whole request rather than to one item: the audit row
// is what makes the grant answerable later (FR-081, SC-021), so continuing after
// it fails would produce a grant nobody can attribute.
func ErrAdminGrantRecordAuditFailed(cause error) *apperror.Error {
	e := apperror.NewInternalError("Không thể ghi nhận ký kiểm toán cho lần cấp gói.")
	e.Cause = cause
	return e
}

// ErrUserDailyQuotaNotFound (024 T148).
//
// Returned when `CreateIfAbsent` reported a conflict but the row is then not
// readable. The conflict proves a row exists, so this cannot happen in practice -
// it is here so the impossible case fails loudly instead of handing back a quota
// object that was never persisted and would reset the customer's daily counters.
func ErrUserDailyQuotaNotFound() *apperror.Error {
	return apperror.NewInternalError("Không đọc được bộ đếm hằng ngày của tài khoản.")
}

// ErrAdminGrantConcurrentModification is the reason an item reports
// `concurrent_modification`.
//
// It is a factory rather than a sentinel so the use case never creates a business
// error inline. Callers compare with `errors.Is` against a fresh instance, which
// works because `apperror` factories return distinct values with the same meaning
// - so the comparison is made on the exported reason code, not on this error.
func ErrAdminGrantConcurrentModification() *apperror.Error {
	return apperror.NewConflict("Tài khoản vừa được thay đổi bởi một thao tác khác.")
}

// ErrProvisioningRequiresCallerTransaction (024 T202).
//
// Returned when eager default provisioning is reached on a context that carries no
// transaction. `EnsureDefaultSubscription` writes three rows - the subscription,
// its history and the outbox record - and takes the handle for all three from the
// context. Without a transaction each write falls back to the base connection and
// commits on its own, which is the exact failure 024 T126 removed on the lazy door:
// an account ends up holding a plan whose event nobody ever heard about, and
// nothing reports an error.
//
// It is also the state the FK would reject if the writes were attempted at all -
// `user_subscriptions.user_id` points at `identity.users(id)`, and the account row
// only exists inside the caller's uncommitted transaction - so failing here turns a
// partial write into one clear error.
//
// The check lives in the module contract, not in the provisioning body: "run inside
// the caller's transaction" is a rule about the ADAPTER that hands the context in,
// and keeping it there means every production door is covered without the business
// logic having to defend itself.
func ErrProvisioningRequiresCallerTransaction() *apperror.Error {
	return apperror.NewInternalError("Không thể cấp phát gói hội viên mặc định vì thiếu giao dịch của yêu cầu tạo tài khoản.")
}

// 025 T035: lỗi của cơ chế chiến dịch.
//
// Chúng đi kèm `apperror` factory theo hiến phán §V: use case KHÔNG tự dựng lỗi
// nghiệp vụ. Mỗi lỗi dưới đây là một *mã lý do* mà nhật ký và giao diện tra cứu
// (FR-061) đọc lại được, nên thông điệp ở đây phải nói rõ ĐIỀU KIỆN đã hỏng chứ
// không chỉ nêu triệu chứng.

// ErrCampaignLockClaimWindowFailed báo rằng không lấy được khoá tư vấn của một
// chiến dịch.
//
// Đây là lỗi TẠM THỜI, không phải lỗi vĩnh viễn: thông điệp phải được thử lại
// (FR-062). Nếu nó bị coi là vĩnh viễn thì một chiến dịch sẽ mất lượt cấp chỉ vì
// khoá bận — và lượt cấp mất thì không tự quay lại.
func ErrCampaignLockClaimWindowFailed(campaignCode string) *apperror.Error {
	e := apperror.NewInternalError("Không thể lấy khoá hạn mức của chiến dịch. Vui lòng thử lại sau.")
	e.Cause = fmt.Errorf("campaign %s: lock claim window", campaignCode)
	return e
}

// ErrCampaignCountClaimFailed báo rằng không đếm được lượt đã cấp.
//
// Tạm thời: đây là lỗi cơ sở dữ liệu, và thử lại là hành vi đúng. Quan trọng hơn,
// lỗi này phải nổi lên chứ không được thay bằng con số 0 — một phép đếm trả về 0
// khi thực ra thất bại sẽ cấp vượt hạn mức ngay lập tức.
func ErrCampaignCountClaimFailed(campaignCode string, bucket string) *apperror.Error {
	e := apperror.NewInternalError("Không thể kiểm tra số lượt đã cấp của chiến dịch. Vui lòng thử lại sau.")
	e.Cause = fmt.Errorf("campaign %s: count bucket %s", campaignCode, bucket)
	return e
}

// ErrCampaignReadWatermarkFailed báo rằng không đọc được mốc ưu tiên.
//
// Tạm thời, và KHÔNG được thay bằng "chưa chốt mốc". Phần dự phòng chỉ dành cho
// tài khoản tạo TRƯỚC mốc ưu tiên (FR-031), nên coi một lỗi đọc là "chưa có mốc"
// sẽ cấp gói cho những tài khoản không đủ điều kiện — đúng loại hỏng âm thầm mà
// FR-090 cấm.
func ErrCampaignReadWatermarkFailed(campaignCode string) *apperror.Error {
	e := apperror.NewInternalError("Không thể kiểm tra mốc ưu tiên của chiến dịch. Vui lòng thử lại sau.")
	e.Cause = fmt.Errorf("campaign %s: read watermark", campaignCode)
	return e
}

// ErrCampaignWriteClaimFailed báo rằng không ghi được lượt cấp.
//
// Tạm thời: lượt cấp chưa tồn tại thì tài khoản chưa nhận gói, và thử lại là hành
// vi đúng. Quan trọng: lỗi này phải làm hỏng cả giao dịch chứ không chỉ bỏ qua
// lượt cấp — nếu gói đã cấp mà lượt cấp chưa ghi thì một lần quét bù sau đó sẽ
// cấp lại lần nữa.
func ErrCampaignWriteClaimFailed(campaignCode string) *apperror.Error {
	e := apperror.NewInternalError("Không thể ghi nhận lượt cấp của chiến dịch. Vui lòng thử lại sau.")
	e.Cause = fmt.Errorf("campaign %s: write claim", campaignCode)
	return e
}

// ErrCampaignListClaimsFailed báo rằng không đọc được danh sách lượt cấp cho
// giao diện tra cứu của người vận hành (FR-065).
func ErrCampaignListClaimsFailed(campaignCode string) *apperror.Error {
	e := apperror.NewInternalError("Không thể tra cứu danh sách tài khoản đã cấp gói.")
	e.Cause = fmt.Errorf("campaign %s: list claims", campaignCode)
	return e
}

// ErrCampaignAccountsQueryFailed báo rằng không đọc được danh sách tài khoản
// trong cửa sổ chiến dịch qua hợp đồng danh tính (FR-079).
func ErrCampaignAccountsQueryFailed(campaignCode string) *apperror.Error {
	e := apperror.NewInternalError("Không thể tra cứu danh sách tài khoản trong thời gian chiến dịch.")
	e.Cause = fmt.Errorf("campaign %s: query accounts in window", campaignCode)
	return e
}

// ErrCampaignMessageMalformed báo rằng thông điệp tạo tài khoản không đủ
// trường để ra quyết định (FR-062).
//
// Đây là lỗi **VĨNH VIỄN**, không phải lỗi tạm thời: thử lại sẽ cho ra cùng
// một kết quả sai, vì vậy thông điệp phải bị loại vĩnh viễn chứ không requeue.
// Thiếu `userId` hoặc `registeredAt` là lỗi của người phát, và lặp lại nó chỉ
// làm nghẽn hàng đỗi.
func ErrCampaignMessageMalformed(field string) *apperror.Error {
	e := apperror.NewBadRequest("Thông điệp tạo tài khoản không hợp lệ.")
	e.Cause = fmt.Errorf("campaign: user-created payload missing %s", field)
	return e
}

// ErrCampaignReadSubscriptionFailed báo rằng không đọc được gói hiện tại của
// tài khoản khi đánh giá điều kiện đủ điều kiện.
//
// Tạm thời: là lỗi cơ sở dữ liệu, và thử lại là hành vi đúng. Quan trọng: KHÔNG
// được coi là "tài khoản không đủ điều kiện" — một lỗi đọc biến thành lý do bỏ
// qua sẽ âm thầm lấy đi một suất của chiến dịch mà không hề cấp gói.
func ErrCampaignReadSubscriptionFailed(userID uuid.UUID) *apperror.Error {
	e := apperror.NewInternalError("Không thể đọc thông tin gói của tài khoản. Vui lòng thử lại sau.")
	e.Cause = fmt.Errorf("campaign: read subscription for user %s", userID)
	return e
}

// ErrCampaignLoadPlanFailed báo rằng không tải được gói cước mà chiến dịch
// khai báo (FR-010).
func ErrCampaignLoadPlanFailed(planSlug string) *apperror.Error {
	e := apperror.NewInternalError("Không thể tải thông tin gói của chiến dịch. Vui lòng thử lại sau.")
	e.Cause = fmt.Errorf("campaign: load plan %s", planSlug)
	return e
}

// ErrCampaignApplyPlanFailed báo rằng không áp dụng được gói lên tài khoản.
//
// Chạm toàn bộ giao dịch: lượt cấp đã ghi mà gói chưa cấp thì lần quét bù sau sẽ
// cấp lại (FR-039), còn gói đã cấp mà lượt cấp chưa ghi thì thông điệp sau sẽ
// cấp lần nữa.
func ErrCampaignApplyPlanFailed(campaignCode string) *apperror.Error {
	e := apperror.NewInternalError("Không thể cấp gói cho tài khoản trong chiến dịch. Vui lòng thử lại sau.")
	e.Cause = fmt.Errorf("campaign %s: apply plan", campaignCode)
	return e
}

// ErrCampaignPublishPlanChangeFailed báo rằng không phát được sự kiện đổi gói
// sau khi đã áp dụng gói (FR-036, FR-039).
//
// CHẠM TOÀN BỘ GIAO DỊCH, và đây là lý do: dịch vụ AI chỉ biết tài khoản có
// quyền AI mới qua sự kiện này. Gói đã cấp mà sự kiện không phát nghĩa là khách
// trả tiền theo Premium nhưng vẫn ăn hạn mức miễn phí — và không nơi nào báo
// lỗi. Chặn giao dịch là cách duy nhất biến hở đó thành lỗi nhìn thấy được.
func ErrCampaignPublishPlanChangeFailed(campaignCode string) *apperror.Error {
	e := apperror.NewInternalError("Không thể phát thông báo thay đổi gói của chiến dịch. Vui lòng thử lại sau.")
	e.Cause = fmt.Errorf("campaign %s: publish plan change", campaignCode)
	return e
}

// ErrCampaignRecordAuditFailed báo rằng không ghi được nhật ký kiểm toán cho lần
// cấp gói của chiến dịch (FR-038).
//
// Chạm toàn bộ giao dịch: dòng nhật ký là thứ làm cho lần cấp gói trả lời được
// câu hỏi "ai cấp, theo nguồn nào" về sau (FR-094). Tiếp tục sau khi nó thất bại
// là tạo ra một lần cấp gói mà không ai quy được trách nhiệm.
func ErrCampaignRecordAuditFailed(campaignCode string) *apperror.Error {
	e := apperror.NewInternalError("Không thể ghi nhận ký kiểm toán cho lần cấp gói của chiến dịch.")
	e.Cause = fmt.Errorf("campaign %s: record audit", campaignCode)
	return e
}
