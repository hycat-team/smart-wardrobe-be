# Đặc tả tính năng: Khả năng quan sát cho admin (nhật ký hoạt động, giao dịch payOS, kỳ dữ liệu dashboard)

**Feature Branch**: `029-admin-observability`

**Created**: 2026-10-15

**Status**: Draft

**Input**: Bổ sung khả năng quan sát cho admin: (1) API xem nhật ký hoạt động người dùng (`user_activity_events`) — chưa tồn tại; (2) chuẩn hoá contract API admin xem giao dịch payOS — đã tồn tại nhưng lỗi DTO; (3) sửa gốc BE các lỗi kỳ dữ liệu dashboard khiến FE không thể tách rõ daily/monthly; (4) bàn giao `frontend-guide.md` gồm hướng dẫn tích hợp các nhóm API mới/thay đổi + đánh giá UX trang dashboard admin của FE.

## Clarifications

### Session 2026-10-15

- Q: Khi FE gửi fromDate/toDate cho endpoint "phân bổ người dùng theo gói", phân bổ mang nghĩa gì? → A: User có gói hiệu lực trong khoảng — cửa sổ gói hiện tại (`started_at` → `expires_at`) giao với [fromDate, toDate], nhóm theo gói hiện tại; gói FREE (không hết hạn) hiện diện mọi kỳ; chấp nhận giới hạn không tái dựng lịch sử đổi gói.
- Q: Panel "đối soát payOS" (số đếm + tổng giá trị thành công) bao gồm loại giao dịch nào? → A: Cả mua gói trực tiếp (`direct_purchase`) lẫn nạp ví (`wallet_topup`) — mọi dòng tiền qua payOS, khớp trang danh sách giao dịch; số đếm sẽ tăng so với hiện tại.
- Q: Panel "đối soát payOS" có nhận fromDate/toDate để tổng hợp theo kỳ không? → A: Có — tổng hợp theo khoảng từ bảng giao dịch (mặc định hôm nay khi không gửi kỳ), thay nguồn ảnh chụp bằng truy vấn theo kỳ.
- Q: Endpoint danh sách giao dịch có được sửa luôn lỗi kỳ dữ liệu (toDate lệch múi giờ làm thiếu ngày cuối; ngày sai bị nuốt im lặng) không? → A: Có — áp chuẩn kỳ chung (bao gồm trọn ngày theo giờ Việt Nam; 400 khi ngày sai định dạng hoặc ngày bắt đầu sau ngày kết thúc); guide ghi before→after.
- Q: Xử lý trạng thái legacy (FailedLegacy) trong contract giao dịch thế nào? → A: Output `failed_legacy` và bộ lọc nhận giá trị này (khứ hồi đủ 11 giá trị); dữ liệu lịch sử giữ nguyên, không migration.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Admin tra cứu nhật ký hoạt động người dùng (Priority: P1)

Quản trị viên mở danh sách hoạt động người dùng (đăng nhập, làm mới phiên), lọc theo người dùng, loại hoạt động và khoảng ngày, và thấy ngay danh tính người dùng (email, tên) trong từng dòng mà không phải tra chéo màn hình khác.

**Why this priority**: Khả năng quan sát này đang thiếu hoàn toàn — hệ thống chỉ có con số DAU tổng hợp, không trả lời được "ai đăng nhập lúc nào" để điều tra sự cố hoặc hỗ trợ người dùng.

**Independent Test**: Phát sinh đăng nhập/làm mới phiên cho ít nhất 2 tài khoản, gọi danh sách không lọc và có lọc — đối chiếu kết quả với dữ liệu đã phát sinh.

**Acceptance Scenarios**:

1. **Given** một người dùng đã đăng nhập trong ngày, **When** admin gọi danh sách hoạt động không lọc, **Then** sự kiện đăng nhập xuất hiện kèm email, tên hiển thị và thời điểm (theo giờ Việt Nam).
2. **Given** có sự kiện của nhiều người dùng với nhiều loại hoạt động, **When** admin lọc theo người dùng + loại hoạt động + khoảng ngày, **Then** kết quả chỉ chứa các sự kiện khớp đủ cả ba điều kiện, sắp xếp mới nhất trước.
3. **Given** admin lọc với loại hoạt động không hợp lệ hoặc khoảng ngày sai (định dạng sai, ngày bắt đầu sau ngày kết thúc), **When** gọi danh sách, **Then** nhận lỗi 400 với thông báo tiếng Việt.
4. **Given** người gọi không có quyền admin, **When** gọi danh sách, **Then** bị từ chối theo chuẩn phân quyền hiện có (401/403).

---

### User Story 2 - Admin xem và xử lý giao dịch payOS qua contract dùng được (Priority: P1)

Quản trị viên xem danh sách giao dịch payOS (lọc theo trạng thái, khoảng ngày, phân trang), mở chi tiết một giao dịch và thực hiện các hành động đối soát (thử lại điều tra, đánh dấu đã thanh toán/đã huỷ/tạo thất bại). Mọi phản hồi dùng trường camelCase với trạng thái là chuỗi đọc được.

**Why this priority**: API danh sách đã tồn tại nhưng contract chi tiết bị hỏng (trả entity thô với tên trường Go, trạng thái là số) nên frontend không thể dựng trang; sửa contract là điều kiện tiên quyết để trang giao dịch ra đời.

**Independent Test**: Gọi danh sách + chi tiết trên dữ liệu thật và thực hiện một hành động đối soát; xác nhận JSON camelCase, trạng thái dạng chuỗi, và trạng thái giao dịch thay đổi đúng sau hành động.

**Acceptance Scenarios**:

1. **Given** tồn tại giao dịch ở nhiều trạng thái, **When** admin lọc `status=success`, **Then** chỉ nhận giao dịch thành công và mỗi phần tử có trường trạng thái dạng chuỗi `"success"` (không phải số).
2. **Given** admin mở chi tiết một giao dịch, **Then** phản hồi có đúng bộ trường camelCase như phần tử trong danh sách (không có trường kiểu Go như `OrderCode`, `UserID`), và trạng thái là chuỗi.
3. **Given** giao dịch đang ở trạng thái cần đối soát, **When** admin gọi hành động đối soát phù hợp, **Then** trạng thái giao dịch chuyển đúng và lần gọi danh sách/chi tiết kế tiếp phản ánh trạng thái mới.
4. **Given** có giao dịch phát sinh trong ngày hôm nay (giờ Việt Nam), **When** admin lọc với ngày kết thúc là hôm nay, **Then** mọi giao dịch của cả ngày hôm nay đều được tính (không bị cắt lệch múi giờ); ngày sai định dạng hoặc ngày bắt đầu sau ngày kết thúc trả 400 tiếng Việt.

---

### User Story 3 - Kỳ dữ liệu dashboard nhất quán cho daily/monthly (Priority: P2)

Quản trị viên chọn một kỳ (hôm nay, tháng này, 30 ngày gần nhất hoặc khoảng tuỳ chọn) và mọi panel số liệu theo kỳ — gồm cả đối soát payOS — đều phản ánh đúng kỳ đã chọn; riêng "Tháng này" nghĩa là từ đầu tháng đến hiện tại. Các panel ảnh chụp hiện trạng còn lại (thẻ KPI, phân bổ khi không gửi kỳ) ghi rõ mốc dữ liệu thay vì âm thầm giữ số cũ.

**Why this priority**: Nhiều panel đang bỏ qua bộ lọc hoặc dùng mặc định ngầm (6 tháng cuốn, toàn thời gian) nên admin không giải thích được số liệu — đây là gốc của phàn nàn UX "không chia rõ daily/monthly".

**Independent Test**: Gọi từng endpoint số liệu với hai kỳ khác nhau (tháng này so với 30 ngày gần nhất) và xác nhận số liệu thay đổi tương ứng; gọi không kèm ngày và xác nhận kết quả giữ nguyên như trước (trừ các chỉ số đã ghi nhận thay đổi chủ đích).

**Acceptance Scenarios**:

1. **Given** chọn kỳ "Tháng này" (từ ngày 1 đến hiện tại, giờ Việt Nam), **When** xem doanh thu theo gói và tháng, **Then** dữ liệu trả về chỉ gồm các giao dịch trong kỳ (không còn mặc định 6 tháng cuốn).
2. **Given** chọn một khoảng tuỳ chọn, **When** xem hiệu quả gia hạn, tỷ lệ lỗi AI và đối soát payOS, **Then** cả ba chỉ tính trên các sự kiện/giao dịch phát sinh trong kỳ đó.
3. **Given** chọn một khoảng tuỳ chọn, **When** xem phân bổ người dùng theo gói, **Then** kết quả đếm user có gói hiệu lực tại bất kỳ thời điểm nào trong khoảng đó (cửa sổ gói giao với khoảng lọc), nhóm theo gói hiện tại.
4. **Given** gọi các endpoint số liệu mà không kèm ngày, **Then** kết quả giữ đúng hành vi mặc định trước đây (tương thích ngược), trừ các chỉ số đã sửa ngữ nghĩa ở US4 và panel đối soát payOS.

---

### User Story 4 - Chỉ số đúng ngữ nghĩa: tỷ lệ lỗi AI và tổng giá trị giao dịch thành công (Priority: P2)

Quản trị viên tin được hai con số đang gây hiểu nhầm: tỷ lệ lỗi AI (tử số và mẫu số phải cùng một kỳ, không thể vượt 100%) và tổng giá trị giao dịch thành công (phải là tổng tiền giao dịch thành công thật, không phải doanh thu gói cước).

**Why this priority**: Hai con số sai dẫn tới kết luận vận hành sai; sửa chúng là điều kiện để dashboard dùng được cho quyết định.

**Independent Test**: Đối chiếu trực tiếp với dữ liệu nguồn: đếm lỗi và request trong cùng kỳ; cộng tiền các giao dịch thành công và so với giá trị trả về.

**Acceptance Scenarios**:

1. **Given** trong một kỳ có E lỗi và R yêu cầu AI (E ≤ R), **When** xem tỷ lệ lỗi AI, **Then** tỷ lệ bằng E/R×100, không vượt 100%; khi R = 0 thì tỷ lệ bằng 0.
2. **Given** tổng tiền các giao dịch thành công qua payOS (cả mua gói trực tiếp lẫn nạp ví) là X và doanh thu gói cước cùng kỳ là Y (X ≠ Y), **When** xem đối soát payOS, **Then** trường "tổng giá trị thành công" bằng X.

---

### User Story 5 - Frontend nhận tài liệu bàn giao đầy đủ (Priority: P3)

Lập trình viên frontend dựng hai trang quản trị mới (hoạt động người dùng, giao dịch payOS) và chỉnh dashboard chỉ từ tài liệu bàn giao: đủ endpoint mới/thay đổi với before→after, ví dụ request/response, hướng dẫn tích hợp, và danh sách lỗi UX cần sửa kèm phân loại trách nhiệm.

**Why this priority**: Hiến pháp bắt buộc mọi thay đổi bề mặt client phải có tài liệu bàn giao; đây cũng là kênh duy nhất để các phát hiện UX đến được frontend (spec không sửa code FE).

**Independent Test**: Đối chiếu tài liệu với Swagger/mã nguồn bằng cách đếm (trường, giá trị enum, route, mã HTTP) và dựng thử mapping từ ví dụ JSON trong tài liệu.

**Acceptance Scenarios**:

1. **Given** tài liệu hoàn tất, **When** lập trình viên frontend đọc, **Then** mọi endpoint mới/thay đổi đều có method + path, dấu new/changed, before→after và ví dụ JSON, đủ để triển khai mà không cần đọc mã nguồn backend hay Swagger.
2. **Given** mục đánh giá UX dashboard, **When** lập trình viên frontend đọc, **Then** mỗi lỗi có mô tả, nguyên nhân gốc, và phân loại rõ: backend đã sửa trong 029 / frontend cần làm / ghi nhận deferred.

---

### Edge Cases

- Người dùng đã bị vô hiệu hoá: sự kiện hoạt động cũ vẫn hiển thị bình thường (dữ liệu lịch sử không bị ẩn); người dùng bị xoá cứng thì sự kiện cũng bị xoá theo ràng buộc dữ liệu hiện có.
- Bộ lọc không khớp bất kỳ sự kiện nào: trả danh sách rỗng kèm metadata phân trang, không lỗi.
- Giới hạn trang vượt trần: tự chuẩn hoá theo quy ước phân trang hiện có (tối đa 100 bản ghi/trang).
- Giao dịch tồn tại ở trạng thái cũ không nằm trong tập giá trị lọc hiện hành (trạng thái legacy): biểu diễn chuỗi là `failed_legacy`, không được rơi về số; giá trị này cũng dùng lại được làm đầu vào bộ lọc.
- Giao dịch không tồn tại: trả lỗi không tìm thấy theo chuẩn hiện có (404).
- Kỳ dữ liệu: ngày bắt đầu sau ngày kết thúc hoặc sai định dạng → lỗi 400 tiếng Việt; kỳ không có dữ liệu → trả số 0/danh sách rỗng, không lỗi.
- Kỳ "Tháng này" rơi vào ngày mùng 1: khoảng ngày bắt đầu bằng ngày kết thúc, vẫn hoạt động bình thường.
- Múi giờ: mọi kỳ tính theo giờ Việt Nam, khoảng ngày bao gồm trọn cả ngày bắt đầu và ngày kết thúc.
- Tỷ lệ lỗi khi mẫu số bằng 0: trả 0%, không chia cho 0.

## Requirements *(mandatory)*

### Functional Requirements

**Nhóm A — API nhật ký hoạt động người dùng (mới)**

- **FR-001**: Hệ thống MUST cung cấp API admin `GET /api/v1/admin/activity-events` trả danh sách phân trang các sự kiện hoạt động; mỗi phần tử gồm: mã sự kiện, mã người dùng, loại hoạt động, thời điểm phát sinh, thời điểm ghi nhận, kèm danh tính người dùng (email, tên hiển thị).
- **FR-002**: API MUST hỗ trợ lọc theo: người dùng, loại hoạt động (tập giá trị hiện có: `login`, `refresh`), khoảng ngày (`fromDate`/`toDate`, định dạng `YYYY-MM-DD`, bao gồm trọn ngày); phân trang chuẩn (mặc định 20, tối đa 100 bản ghi/trang).
- **FR-003**: API MUST chỉ cho vai trò admin truy cập (401/403 theo chuẩn middleware hiện có); bộ lọc sai định dạng hoặc khoảng ngày không hợp lệ MUST trả 400 với thông báo tiếng Việt.
- **FR-004**: Kết quả MUST sắp xếp mới nhất trước; mọi trường JSON MUST camelCase; thời gian MUST ở định dạng ISO 8601.
- **FR-005**: API MUST chỉ đọc — không tạo, sửa hay xoá sự kiện (bảng giữ nguyên tính chất append-only).
- **FR-006**: API MUST có tài liệu trong `docs/api/identity/` và Swagger annotation đầy đủ.

**Nhóm B — Chuẩn hoá contract API giao dịch payOS (đã tồn tại)**

- **FR-007**: `GET /api/v1/admin/subscriptions/transactions/{id}` MUST trả DTO camelCase với đúng bộ trường như phần tử trong danh sách (`DepositTransactionRes`), KHÔNG trả entity thô (tên trường kiểu Go).
- **FR-008**: Trường trạng thái giao dịch trong mọi phản hồi admin của giao dịch MUST là chuỗi. Tập giá trị xuất ra gồm 10 giá trị đang được bộ lọc chấp nhận (`pending`, `success`, `creating`, `reconciliation_required`, `reconciling`, `creation_failed`, `cancelled`, `expired`, `investigation_required`, `refunded`) cộng `failed_legacy` cho trạng thái legacy (tổng 11 giá trị); mọi giá trị xuất ra MUST dùng lại được làm đầu vào bộ lọc (khứ hồi một vòng) — bộ lọc bổ sung chấp nhận `failed_legacy`.
- **FR-009**: Các endpoint hành động đối soát (thử lại điều tra, đánh dấu đã thanh toán/đã huỷ/tạo thất bại) MUST giữ nguyên đường dẫn và hành vi nghiệp vụ; nếu phản hồi có chứa trạng thái giao dịch thì cũng theo FR-008.
- **FR-010**: Bộ lọc trạng thái và phân trang của danh sách giao dịch MUST không đổi. Khoảng ngày của danh sách MUST theo chuẩn kỳ chung của tính năng (FR-014, FR-015): bao gồm trọn ngày theo giờ Việt Nam; trả 400 khi ngày sai định dạng hoặc ngày bắt đầu sau ngày kết thúc. Tài liệu `docs/api/subscription/admin-billing-api.md` và Swagger MUST được cập nhật khớp contract mới, gồm before→after của kỳ lọc.

**Nhóm C — Kỳ dữ liệu dashboard nhất quán và chỉ số đúng ngữ nghĩa**

- **FR-011**: Các endpoint số liệu theo kỳ MUST nhận `fromDate`/`toDate` và áp dụng đúng khi được gửi: doanh thu theo gói và tháng, hiệu quả gia hạn, phân bổ người dùng theo gói, tỷ lệ lỗi AI, đối soát payOS. Hai endpoint đã nhận kỳ từ trước (biên lợi nhuận AI, sử dụng AI theo tác vụ) MUST cũng diễn giải kỳ theo chuẩn chung FR-014.
- **FR-012**: Khi không gửi kỳ, các endpoint số liệu MUST giữ hành vi mặc định hiện tại (tương thích ngược): doanh thu theo gói và tháng giữ mặc định 6 tháng gần nhất (biên ngày tính theo FR-014 — thay đổi chủ đích về múi giờ, ngữ nghĩa khoảng không đổi); hiệu quả gia hạn và phân bổ giữ nguyên như trước; đối soát payOS mặc định hôm nay (giờ Việt Nam). Ngoại lệ chủ đích: tỷ lệ lỗi AI chuyển mặc định sang cùng kỳ 30 ngày cho cả tử số lẫn phân nhóm (FR-016); đối soát payOS đổi phạm vi loại giao dịch và nguồn tính theo kỳ (FR-017).
- **FR-013**: Khi kỳ được gửi cho phân bổ người dùng theo gói, phân bổ MUST đếm user có gói hiệu lực trong khoảng [fromDate, toDate] — cửa sổ gói hiện tại (`started_at` đến `expires_at`; gói không hết hạn tính là còn mãi) giao với khoảng lọc — nhóm theo gói hiện tại. Khi không gửi kỳ, giữ nguyên nghĩa hiện trạng (người dùng đang có gói hiệu lực). Giới hạn đã chấp nhận: lịch sử đổi gói không được tái dựng, user tính theo gói hiện tại.
- **FR-014**: Mọi kỳ MUST tính theo múi giờ Việt Nam; khoảng ngày bao gồm trọn cả ngày bắt đầu và ngày kết thúc; cách diễn giải kỳ MUST nhất quán giữa mọi endpoint.
- **FR-015**: Các endpoint nhận kỳ MUST trả 400 với thông báo tiếng Việt khi định dạng ngày sai hoặc ngày bắt đầu sau ngày kết thúc.
- **FR-016**: Tỷ lệ lỗi AI MUST tính tử số và mẫu số trên cùng một kỳ (mặc định 30 ngày gần nhất khi không gửi kỳ); danh sách phân nhóm lỗi MUST cũng giới hạn trong cùng kỳ đó (tổng lỗi bằng tổng các nhóm); khi mẫu số bằng 0 MUST trả 0%.
- **FR-017**: Phạm vi "đối soát payOS" (cả số đếm lẫn tổng giá trị) MUST bao gồm mọi loại giao dịch nạp qua payOS — mua gói trực tiếp (`direct_purchase`) và nạp ví (`wallet_topup`) — trong kỳ được chọn (mặc định hôm nay khi không gửi kỳ). "Tổng giá trị giao dịch thành công" MUST bằng tổng số tiền các giao dịch thành công trên đúng phạm vi đó; KHÔNG được dùng doanh thu gói cước thay thế (doanh thu subscription giữ nguyên phạm vi mua gói trực tiếp); khi không có giao dịch thành công MUST trả 0. Ảnh chụp dashboard dùng chung truy vấn này nên các số đếm payOS trong snapshot mới (kể cả trường `payOSSuccessCount` của thẻ KPI) mang cùng phạm vi hai loại — shape không đổi, giá trị đổi chủ đích.

**Nhóm D — Bàn giao frontend và kỷ luật tài liệu**

- **FR-018**: Tính năng MUST ship `specs/029-admin-observability/frontend-guide.md` theo hiến pháp: liệt kê **mọi** endpoint mới/thay đổi (method + path, đánh dấu new/changed, before→after, ví dụ JSON) — gồm cả nhóm dashboard admin đổi tham số kỳ/ngữ nghĩa — kèm hướng dẫn tích hợp trang "Hoạt động người dùng" và trang "Giao dịch payOS" (danh sách + chi tiết + 4 hành động đối soát) — đủ để frontend triển khai mà không cần đọc mã nguồn backend hay Swagger.
- **FR-019**: Tài liệu bàn giao MUST có mục đánh giá UX dashboard admin của frontend, liệt kê tối thiểu các vấn đề đã khảo sát: không tách khu vực daily/monthly; mặc định 30 ngày cuốn và thiếu preset "Hôm nay"/"Tháng này"; bộ lọc ngày chỉ áp dụng một phần panel; nhãn "Doanh thu" và "Tỷ lệ lỗi" trùng nhau nhưng số khác nhau; "DAU hôm nay" lấy từ ảnh chụp cũ; khoảng ngày sai bị thay thế im lặng; bộ lọc bảng sự kiện AI lỗi giá trị mặc định; log gỡ lỗi còn sót; lệch múi giờ khi hiển thị; độ chi tiết biểu đồ cố định "ngày"; trộn ngôn ngữ hiển thị; nhãn cột "Loại" hiển thị sai nội dung; dữ liệu nạp dư không hiển thị. Mỗi vấn đề MUST kèm nguyên nhân gốc và phân loại: backend đã sửa trong 029 / frontend cần làm / deferred.
- **FR-020**: Tài liệu bàn giao MUST được đọc lại và đối chiếu với mã nguồn ở giai đoạn cuối tính năng bằng phép đếm (mọi trường DTO, mọi giá trị enum, mọi route, mọi mã HTTP) theo hiến pháp.
- **FR-021**: Mọi hạng mục cố ý không làm trong tính năng này MUST được ghi vào `specs/029-admin-observability/deferred.md` kèm lý do ngoài phạm vi và điều kiện gỡ.
- **FR-022**: Hệ thống MUST có kiểm thử tự động cho: API nhật ký hoạt động (lọc, phân trang, quyền); contract giao dịch payOS (trạng thái chuỗi, chi tiết camelCase); áp dụng kỳ dữ liệu trên các endpoint mới nhận kỳ; công thức tỷ lệ lỗi cùng kỳ; tổng giá trị giao dịch thành công.

### Key Entities

- **Sự kiện hoạt động người dùng**: một dòng nhật ký append-only ghi nhận người dùng đã đăng nhập hoặc làm mới phiên, gồm người dùng, loại hoạt động, thời điểm phát sinh.
- **Giao dịch nạp tiền**: một lần thanh toán qua payOS cho gói cước, gồm mã đơn, số tiền, trạng thái (11 giá trị nghiệp vụ, trong đó có một giá trị legacy), gói mục tiêu, mốc đối soát; là nguồn dữ liệu của cả trang giao dịch lẫn báo cáo đối soát.
- **Báo cáo đối soát payOS**: bản tổng hợp số lượng giao dịch qua payOS (mọi loại: mua gói trực tiếp + nạp ví) theo nhóm trạng thái và tổng giá trị giao dịch thành công.
- **Kỳ dữ liệu**: khoảng ngày bắt đầu/kết thúc mà một chỉ số được tính, diễn giải theo múi giờ Việt Nam, bao gồm trọn ngày hai đầu.
- **Tài liệu bàn giao frontend**: tài liệu hợp đồng hướng client của tính năng, gồm mọi nhóm API mới/thay đổi (activity-events, giao dịch payOS, dashboard admin), hướng dẫn tích hợp và đánh giá UX dashboard.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Admin tra được toàn bộ lịch sử hoạt động của một người dùng trong một khoảng ngày bằng đúng một lần gọi API có lọc, kết quả kèm danh tính người dùng — không cần tra cứu chéo nguồn nào khác.
- **SC-002**: 100% trường JSON của mọi nhóm API mới/thay đổi (activity-events, giao dịch payOS, dashboard admin) khớp giữa mã nguồn và tài liệu bàn giao (đối chiếu đếm hai chiều); lập trình viên frontend dựng được hai trang quản trị và chỉnh dashboard chỉ từ tài liệu.
- **SC-003**: Với cùng một kỳ được chọn, 100% panel số liệu theo kỳ phản ánh đúng kỳ đó (đổi kỳ → số liệu đổi tương ứng); kỳ "Tháng này" luôn bắt đầu từ ngày mùng 1 theo giờ Việt Nam.
- **SC-004**: Tỷ lệ lỗi AI không bao giờ vượt 100% trong mọi kỳ có dữ liệu; "tổng giá trị giao dịch thành công" khớp tổng tiền các giao dịch thành công trên cùng phạm vi đếm (sai lệch 0 đồng).
- **SC-005**: Không hồi quy: 100% endpoint khi không gửi kỳ giữ nguyên kết quả như trước tính năng, trừ các thay đổi chủ đích đã ghi (tỷ lệ lỗi AI cùng kỳ; panel đối soát payOS đổi phạm vi loại giao dịch và nguồn tính theo kỳ; các số đếm payOS trong snapshot mới/KPI card đổi giá trị theo phạm vi hai loại; kỳ lọc ngày của danh sách giao dịch và biên mặc định của doanh thu theo gói chuẩn hoá theo giờ Việt Nam; 400 khi ngày đảo chiều). Đường dẫn, bộ lọc trạng thái, phân trang và 4 hành động đối soát giữ nguyên về đường dẫn và nghiệp vụ.

## Assumptions

- "Tháng này" nghĩa là từ 00:00 ngày mùng 1 của tháng hiện tại đến thời điểm hiện tại theo giờ Việt Nam; backend không cung cấp preset — frontend tự tính khoảng ngày và gửi lên.
- Phân bổ người dùng theo gói khi có kỳ đếm theo phép giao của cửa sổ gói hiện tại với kỳ lọc ([`started_at`, `expires_at`] ∩ [fromDate, toDate], gói FREE không hết hạn hiện diện mọi kỳ); lịch sử đổi gói không được tái dựng — user chỉ tính theo gói hiện tại. Khi không có kỳ giữ nghĩa hiện trạng.
- Trạng thái legacy của giao dịch (giá trị số 2 trong mã nguồn hiện tại) tồn tại trong dữ liệu cũ; biểu diễn chuỗi đã chốt là `failed_legacy` và bộ lọc chấp nhận giá trị này; dữ liệu lịch sử giữ nguyên, không migration.
- Nhật ký hoạt động hiện chỉ có hai loại (`login`, `refresh`); loại mới trong tương lai sẽ được bổ sung khi phát sinh, ngoài phạm vi tính năng này.
- Hệ thống chỉ có hai vai trò (`admin`, `user`); API mới chỉ dành cho admin, không cần phân quyền chi tiết hơn.
- Danh tính người dùng (email, tên) lấy từ bảng người dùng hiện có; người dùng bị vô hiệu hoá vẫn hiển thị trong lịch sử.
- Repo frontend KHÔNG nằm trong phạm vi sửa code của tính năng này; frontend nhận tài liệu bàn giao để triển khai sau.
- Các thẻ KPI là chỉ số ảnh chụp (snapshot) và phân bổ không kèm kỳ là hiện trạng; việc ghi rõ mốc dữ liệu trên giao diện là việc của frontend, được hướng dẫn trong tài liệu bàn giao.
- Panel đối soát payOS chuyển từ đọc ảnh chụp sang tổng hợp theo khoảng ngày; khi không gửi kỳ mặc định hôm nay theo giờ Việt Nam. Ảnh chụp dashboard vẫn tiếp tục được worker tổng hợp cho các thẻ KPI.
- Không thêm preset kỳ phía backend; backend chỉ nhận khoảng ngày tường minh và giữ nguyên mặc định cũ khi thiếu tham số.
