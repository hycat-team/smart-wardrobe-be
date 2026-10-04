package subscriptionerrors

import (
	"fmt"
	"net/http"

	"smart-wardrobe-be/shared/application/constants/apperror"
)

// 026 (FR-001 / FR-003 / FR-005 / FR-054): lỗi của SỔ ĐĂNG KÝ CHIẾN DỊCH.
//
// Chúng nằm ở tệp riêng chứ không nối thêm vào `errors.go` vì đây là một nhóm
// lỗi mới sinh ra từ một nguồn sự thật mới — bảng `subscription.campaigns` — và
// nhóm đó có hai đặc điểm mà nhóm cũ không có:
//
//   - NÓ TRẢ VỀ CHO NGƯỜI VẬN HÀNH, không chỉ cho kỹ sư. Nhóm 025 dưới đây đã
//     có hai lỗi như vậy; nhóm này là toàn bộ những lỗi mà người gọi là một quản
//     trị viên đang cố sửa một chiến dịch đang chạy. Vì vậy mỗi thông điệp phải nêu
//     CẦN LÀM GÌ, không chỉ nêu triệu chứng.
//   - NÓ PHẢI PHÂN BIỆT ĐƯỢC "SAI" VỚI "HỎNG". 404 nghĩa là mã gõ sai, 500 nghĩa
//     là CSDL hỏng, và hai người đó phải làm hai việc hoàn toàn khác nhau.
//
// `422` và `412` không có nhà xướng sẵn trong `apperror` (chỉ có 400/403/404/409/
// 429/500), nên chúng dựng qua `NewError` với mã HTTP tường minh. Thêm hai nhà
// xướng vào `apperror` cho đúng ba chỗ dùng sẽ là thay đổi hợp đồng chung của cả
// ba dịch vụ, và FR-054 đã chốt BẢNG MÃ chứ không chốt bộ nhà xướng.

// ErrCampaignRegistryReadFailed báo rằng không đọc được sổ chiến dịch (026,
// FR-003/FR-040).
//
// Đây là lỗi của đường CHẠY, không phải của đường ghi: nó xuất hiện ở mỗi lượt cấp
// gói, mỗi lần quét bù và mỗi lần tra cứu, vì sổ là nguồn sự thật duy nhất và được
// đọc ở runtime (FR-001/FR-003) thay vì nạp một lần lúc khởi động.
//
// TẠM THỜI, và đó là điều quan trọng nhất về nó: một lỗi đọc KHÔNG BAO GIỜ được
// thay bằng "không có chiến dịch nào". Một sổ rỗng khi thực ra là do CSDL hỏng sẽ
// biến mọi lượt cấp gói thành `campaign_not_configured`, và người vận hành sẽ đọc
// một hàng đỗi im lặng là "chưa có chiến dịch nào chạy" thay vì là một sự cố.
//
// Sự khác biệt đó có giá: một sự cố CSDL biến mọi lượt đăng ký thành lượt retry.
// Đó là hành vi ĐÚNG — mất một lượt cấp là lỗi, còn cấp sai là lỗi vĩnh viễn và
// im lặng.
func ErrCampaignRegistryReadFailed() *apperror.Error {
	e := apperror.NewInternalError("Không thể tra cứu danh sách chiến dịch. Vui lòng thử lại sau.")
	e.Cause = fmt.Errorf("campaign registry: list campaigns")
	return e
}

// ErrCampaignDuplicateCode báo rằng mã chiến dịch đã tồn tại (FR-054 → 409).
//
// CHỈ dùng khi tạo. Người vận hành gõ mã từ một dòng tin khác (kế hoạch ra mắt,
// email cũ), nên thông điệp nêu lại mã đã nhận và nói rõ việc sửa là đổi mã ở thân
// yêu cầu chứ không đổi mã trên URL — mã là bất biến sau khi tạo (FR-024), nên sửa
// URL sẽ không giúp được gì và chỉ làm người gọi mất thêm một lượt.
func ErrCampaignDuplicateCode(campaignCode string) *apperror.Error {
	e := apperror.NewConflict(fmt.Sprintf(
		"Đã tồn tại chiến dịch với mã %s. Vui lòng chọn một mã khác.", campaignCode))
	e.Cause = fmt.Errorf("campaign %s: duplicate code", campaignCode)
	return e
}

// ErrCampaignInvalidField báo rằng một trường vi phạm bất biến của sổ (FR-054 → 422).
//
// Đây là lỗi "đúng hình dạng nhưng sai nghĩa": yêu cầu đã qua được bước kiểm hình
// dạng (RFC3339 có múi giờ, mã khớp `^[a-z0-9][a-z0-9-]{2,63}$`), nhưng giá trị
// không có nghĩa — `quota < 1`, `reserve < 0`, `endsAt` không sau `startsAt`,
// `planSlug` rỗng, `reason` đóng rỗng.
//
// Nêu TÊN TRƯỜNG là bắt buộc: người vận hành đang sửa một biểu mẫu, và một thông
// báo không nói trường nào thì phải mở từng trường ra thử lại lần nữa.
func ErrCampaignInvalidField(field string, reason string) *apperror.Error {
	e := apperror.NewError(http.StatusUnprocessableEntity, "Dữ liệu không hợp lệ", fmt.Sprintf(
		"Trường %s không hợp lệ: %s.", field, reason))
	e.Cause = fmt.Errorf("campaign registry: invalid field %s: %s", field, reason)
	return e
}

// ErrCampaignBudgetLocked báo rằng ngân sách của chiến dịch đã có lượt cấp nên bị
// khoá (FR-020/FR-021/FR-022 → 422).
//
// Điều kiện bất biến là CÓ LƯỢT CẤP, không phải "đã tới starts_at": một chiến dịch
// chưa cấp cho ai thì sửa `quota`/`reserve`/`plan_slug` hoàn toàn an toàn, vì chưa có
// dòng lịch sử nào để diễn giải lại (spec.md Câu 3). Một chiến dịch đã qua
// `starts_at` mà chưa cấp cho ai thì vẫn sửa được, và khoá oan trường hợp đó là một
// kiểu lỗi người vận hành sẽ không bao giờ tự tìm ra nguyên nhân.
//
// Thông điệp phải nêu LỐI RA chứ không chỉ nêu lỗi: "đã cấp, hãy đóng và mở mã
// mới" là thứ duy nhất người vận hành làm được, vì đóng là một chiều (FR-034) và mở
// lại mã cũ sẽ cấp gói cho những tài khoản tạo trong khoảng thời gian đã đóng.
func ErrCampaignBudgetLocked(campaignCode string) *apperror.Error {
	e := apperror.NewError(http.StatusUnprocessableEntity, "Dữ liệu không hợp lệ", fmt.Sprintf(
		"Chiến dịch %s đã cấp gói cho ít nhất một tài khoản, nên hạn mức (quota), "+
			"lượng dự phòng (reserve) và mã gói (planSlug) không thể thay đổi nữa. "+
			"Vui lòng đóng chiến dịch này và mở một mã chiến dịch mới.", campaignCode))
	e.Cause = fmt.Errorf("campaign %s: budget locked by existing claims", campaignCode)
	return e
}

// ErrCampaignVersionConflict báo rằng phiên bản mong đợi đã lệch với dữ liệu
// (FR-055 → 412).
//
// Đây là câu trả lời cho hai quản trị viên cùng sửa một chiến dịch. Thông điệp nói
// rõ PHẢI LÀM GÌ (tải lại rồi sửa lại) thay vì chỉ nói "lệch phiên bản", vì nếu
// không thì người thứ hai sẽ thử lại với đúng số phiên bản cũ và nhận 412 lần nữa —
// rồi kết luận hệ thống hỏng.
func ErrCampaignVersionConflict(campaignCode string) *apperror.Error {
	e := apperror.NewError(http.StatusPreconditionFailed, "Dữ liệu đã thay đổi", fmt.Sprintf(
		"Chiến dịch %s đã được thay đổi bởi thao tác khác. Vui lòng tải lại thông tin "+
			"chiến dịch rồi thực hiện lại thay đổi.", campaignCode))
	e.Cause = fmt.Errorf("campaign %s: version conflict", campaignCode)
	return e
}

// ErrCampaignRegistryWriteFailed báo rằng không ghi được dòng chiến dịch (026).
//
// Lỗi hạ tầng, không phải lỗi người gọi: yêu cầu đã qua hết kiểm hình dạng và kiểm
// bất biến, nên thất bại ở đây là cơ sở dữ liệu. Nói sai loại sẽ mời người vận
// hành "sửa lại đầu vào" trong khi đầu vào của họ đã đúng (data-model.md §2.1).
//
// Mã chiến dịch đi vào `Cause` chứ không vào `Message`: thông bệp trả về cho client,
// và mã chiến dịch là dữ liệu vận hành chứ không phải thông báo lỗi.
func ErrCampaignRegistryWriteFailed(campaignCode string, cause error) *apperror.Error {
	e := apperror.NewInternalError("Không thể lưu thông tin chiến dịch. Vui lòng thử lại sau.")
	e.Cause = fmt.Errorf("campaign %s: registry write: %w", campaignCode, cause)
	return e
}

// ErrCampaignAuditListFailed báo rằng không đọc được nhật ký kiểm toán của một
// chiến dịch (FR-063).
//
// Nổi lên chứ không trả trang rỗng: nhật ký rỗng đọc như "chiến dịch này chưa từng
// có ai thay đổi", và đó là một kết luận sai — người vận hành sẽ kết luận cơ chế
// kiểm toán không hoạt động thay vì báo lỗi hạ tầng.
func ErrCampaignAuditListFailed(campaignCode string) *apperror.Error {
	e := apperror.NewInternalError("Không thể tra cứu nhật ký kiểm toán của chiến dịch. Vui lòng thử lại sau.")
	e.Cause = fmt.Errorf("campaign %s: list audit trail", campaignCode)
	return e
}

// ---- Nhóm lỗi HÌNH DẠNG (400) của ba endpoint ghi, thêm ở T030 ----
//
// Phase 1 (T003) khai bảy lỗi của sổ đăng ký, và bảy lỗi đó đều là 409/412/422/500:
// chúng mô tả những gì đã đi qua bước kiểm hình dạng rồi mới hỏng. Ba lỗi dưới đây
// là phần còn thiếu của bảng FR-054 — những lỗi phát ra TRƯỚC khi có thể nói gì về
// ngữ nghĩa — và chúng được thêm ở T030, đúng chỗ validator cần tới chúng.
//
// Chúng nằm ở đây chứ không ở `errors.go` vì cùng một lý do như phần trên: đây là
// nhóm lỗi của sổ đăng ký, và nhóm lỗi đó có một đặc điểm mà nhóm 025 không có —
// nó trả lời NGƯỜI VẬN HÀNH đang gõ một biểu mẫu, nên phải nói ra ô nào sai.

// ErrCampaignCodeInvalid báo rằng mã chiến dịch không khớp `^[a-z0-9][a-z0-9-]{2,63}$`
// (FR-054 → 400, FR-011).
//
// Đây là lỗi "HỎNG HÌNH DẠNG": yêu cầu chưa mô tả được một mã chiến dịch, nên không
// có nghĩa nào để kiểm. Nói sai loại sẽ mời người vận hành "kiểm tra lại ngữ nghĩa"
// trong khi ngữ nghĩa chưa tồn tại.
//
// Nêu LUẬT trong thông điệp, vì người gõ mã chiến dịch ở đây là người đang chép từ
// một dòng tin khác (kế hoạch ra mắt, email cũ) chứ không phải người nhớ regex: một
// thông báo chỉ nói "mã không hợp lệ" khiến họ đoán, và đoán sai thường là thêm khoảng
// trắng hoặc dấu gạch dưới — hai thứ luật này cấm.
func ErrCampaignCodeInvalid() *apperror.Error {
	e := apperror.NewBadRequest(
		"Mã chiến dịch không hợp lệ. Mã phải dài 3-64 ký tự, chỉ gồm chữ thường, số và dấu gạch ngang, " +
			"và bắt đầu bằng chữ thường hoặc số.")
	e.Cause = fmt.Errorf("campaign registry: malformed campaign code")
	return e
}

// ErrCampaignTimestampMalformed báo rằng một tham số thời gian không phải là thời điểm
// RFC3339 CÓ MÚI GIỜ (FR-054 → 400, FR-013).
//
// Zone-less là lý do lỗi này tồn tại, không phải thừa. `2026-10-02 00:00:00` không
// mô tả một thời điểm: nó sẽ được đọc theo múi giờ của MÁY CHỦ, lệch bảy giờ ở
// production, và lưu lại một thời điểm không ai gõ. Chiến dịch ra mắt lúc 00:00 thì
// lệch đúng bảy giờ đó là toàn bộ ý nghĩa của thao tác.
//
// Nêu LUẬT kèm VÍ DỤ đúng định dạng, vì lỗi này do người gõ sai định dạng chứ không
// phải do sai ngữ nghĩa — và biểu mẫu của họ chỉ hiện một ví dụ.
func ErrCampaignTimestampMalformed(field string) *apperror.Error {
	e := apperror.NewBadRequest(fmt.Sprintf(
		"Tham số thời gian không hợp lệ: %s phải là một thời điểm RFC3339 có múi giờ, ví dụ 2026-10-02T00:00:00+07:00.",
		field))
	e.Cause = fmt.Errorf("campaign registry: malformed %s timestamp", field)
	return e
}

// ErrCampaignRegistryAuditFailed báo rằng không ghi được dòng kiểm toán cho một thao tác
// ghi sổ đăng ký (FR-060 → 500).
//
// CHẠM TOÀN BỘ GIAO DỊCH, và đó là điểm: `IAuditLogContract.Record` ghi một dòng
// OUTBOX trong cùng transaction với dòng chiến dịch, nên khi nó hỏng thì giao dịch
// rollback và thao tác ghi thất bại. Tiếp tục sau khi nó hỏng sẽ tạo ra một chiến
// dịch mà không ai quy trách nhiệm — đúng thứ FR-061 cấm.
//
// KHÔNG dùng lại `ErrCampaignRecordAuditFailed` của 025: thông điệp của lỗi đó nói
// "lần cấp gói", và một chiến dịch được mở không cấp gói cho ai. Dùng lại nó sẽ khiến
// người vận hành đi tìm một lượt cấp không tồn tại.
func ErrCampaignRegistryAuditFailed(campaignCode string, cause error) *apperror.Error {
	e := apperror.NewInternalError("Không thể ghi nhật ký kiểm toán cho thao tác trên chiến dịch.")
	e.Cause = fmt.Errorf("campaign %s: record registry audit: %w", campaignCode, cause)
	return e
}
