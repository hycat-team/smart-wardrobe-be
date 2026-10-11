# Đặc tả tính năng: Chi phí AI thật phục vụ báo cáo lợi nhuận

**Feature Branch**: `028-ai-actual-cost`

**Created**: 2026-10-11

**Status**: Draft

**Input**: Tách ghi nhận chi phí AI thật (actual) khỏi chi phí tính cho user (billed) — chỉ phục vụ báo cáo lợi nhuận, không đổi cách thu tiền user.

## Clarifications

### Session 2026-10-11

- Q: Lượt gọi thiếu giá model thì "đánh dấu để đối soát sau" bằng cách nào? → A: Giữ status Confirmed, ghi error_code='pricing_config_missing', actual=0; admin đối soát qua API drill-down có sẵn, không thêm cột/bảng.
- Q: Nếu nhà cung cấp không trả số token cho tác vụ miễn phí thì ghi actual thế nào? → A: Ghi actual=0, token 0, kèm error_code='usage_unreported'; không ước tính bịa số liệu.
- Q: Đếm token cho embedding: inline hay worker riêng? (quyết định hậu clarify) → A: Inline countTokens trong orchestrator (1 call thêm/request, fail-open về usage_unreported); không làm worker vì countTokens chỉ trả tổng/không chia ngược cho từng event và event không lưu input text.
- Q: Google chỉ thu tiền embedding theo input, nhưng DB đang lưu output=input — seed thủ công chốt thế nào? → A: Theo Google (output=0, input giữ nguyên); giá trị này được điền trong manual-seed.md.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Admin xem lợi nhuận AI đúng chi phí thật (Priority: P1)

Quản trị viên mở báo cáo tổng quan/thống kê AI và thấy chi phí phản ánh đúng số tiền dự án phải trả cho nhà cung cấp mô hình, kể cả các tác vụ đang miễn phí cho người dùng cuối (như phân tích ảnh, tạo embedding). Biên lợi nhuận ròng được tính từ chi phí thật này.

**Why this priority**: Đây là toàn bộ giá trị của tính năng — báo cáo hiện tại hiển thị 0đ cho các tác vụ miễn phí nên lợi nhuận bị phóng đại, dẫn tới quyết định kinh doanh sai.

**Independent Test**: Mở báo cáo thống kê AI sau khi phát sinh lượt dùng miễn phí (phân tích ảnh, embedding, hoặc lượt dùng bị chuyển sang tuyến miễn phí) và đối chiếu tổng chi phí với bảng giá nhà cung cấp — chi phí phải lớn hơn 0 và khớp cách tính theo token.

**Acceptance Scenarios**:

1. **Given** đã phát sinh lượt phân tích ảnh trong ngày, **When** admin mở báo cáo sử dụng AI theo tác vụ, **Then** dòng tương ứng hiển thị lượt dùng, số token và chi phí thật lớn hơn 0.
2. **Given** đã phát sinh lượt tạo embedding trong ngày, **When** admin mở báo cáo, **Then** lượt embedding được ghi nhận kèm mã `usage_unreported` (spike T001 chứng minh `countTokens` không hỗ trợ embedding model nên không có số liệu — không ước tính bịa số theo Q2-A), thay vì 0đ không rõ nguyên nhân như trước.
3. **Given** có doanh thu gói cước và chi phí AI thật trong kỳ, **When** admin mở báo cáo tổng quan, **Then** lợi nhuận ròng bằng doanh thu trừ đi chi phí thật (bao gồm cả phần miễn phí cho user).

---

### User Story 2 - Người dùng cuối không bị ảnh hưởng (Priority: P1)

Người dùng gói miễn phí/trả phí tiếp tục dùng AI đúng hạn mức cũ: lượt miễn phí vẫn miễn phí, lượt vượt hạn mức vẫn bị chuyển tuyến hoặc từ chối đúng như trước. Không có khoản thu mới nào phát sinh từ tính năng này.

**Why this priority**: Ràng buộc bất biến của tính năng — sai ở đây là sự cố thu tiền, nghiêm trọng hơn mọi sai số báo cáo.

**Independent Test**: Thực hiện cùng một kịch bản dùng AI (trong hạn mức, vượt hạn mức, hết lượt trả phí) trước và sau khi triển khai và so sánh kết quả cho phép/từ chối và số dư hạn mức — phải giống hệt nhau.

**Acceptance Scenarios**:

1. **Given** tài khoản còn lượt miễn phí, **When** gọi tác vụ miễn phí (phân tích ảnh, embedding), **Then** yêu cầu thành công và hạn mức còn lại không đổi so với trước đây.
2. **Given** tài khoản hết lượt trả phí cho một tác vụ, **When** gọi tiếp tác vụ đó, **Then** hệ thống vẫn chuyển sang tuyến miễn phí hoặc từ chối đúng chính sách cũ, không phát sinh khoản thu mới.
3. **Given** tài khoản gói trả phí trong hạn mức, **When** gọi tác vụ trả phí, **Then** số lượt/hạn mức bị trừ đúng bằng trước đây.

---

### User Story 3 - Vận hành tự rà soát giá toàn bộ model (Priority: P2)

Người vận hành tự chạy một đoạn SQL thủ công để bổ sung/cập nhật giá
toàn bộ model đang dùng theo bảng giá nhà cung cấp mới nhất, mà không cần
deploy code hay migration.

**Why this priority**: DB hiện thiếu giá `gemini-3.5-flash-lite` (đang dùng ở
production); nếu thiếu giá thì chi phí thật không tính được. Làm thủ công để
nhanh và do dev tự chạy.

**Independent Test**: Chạy đoạn SQL trong tài liệu hướng dẫn với giá trị mẫu, chạy lại lần hai với giá khác — cả hai lần đều thành công và giá sau cùng bằng giá của lần chạy thứ hai.

**Acceptance Scenarios**:

1. **Given** model chưa có trong bảng giá, **When** chạy đoạn SQL với giá từ bảng giá nhà cung cấp, **Then** model xuất hiện với đúng giá đã nhập.
2. **Given** model đã có trong bảng giá, **When** chạy lại đoạn SQL với giá mới, **Then** giá được gán đè bằng giá mới, không báo lỗi trùng và không bị bỏ qua.

### Edge Cases

- Lượt dùng miễn phí của model mà nhà cung cấp cho dùng miễn phí thật (giá bằng 0): chi phí thật ghi nhận bằng 0 là đúng, không phải lỗi.
- Lượt gọi thất bại (lỗi nhà cung cấp, không xác minh được mức dùng): không được tính vào chi phí thật.
- Lịch sử cũ có số token bằng 0 (do trước đây không lưu token cho tuyến miễn phí): giữ nguyên, không hồi tố — chỉ ghi đúng từ thời điểm triển khai trở đi.
- Thiếu giá model tại thời điểm gọi: lượt gọi của user vẫn phải thành công theo chính sách cũ (suy giảm về tuyến miễn phí), chi phí thật của lượt đó được để lại để đối soát thay vì chặn user.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Hệ thống MUST ghi nhận chi phí thật của mọi lượt gọi AI thành công dựa trên số token thực tế và giá của model trong plan đã gọi (ngoại lệ đã biết: khi provider fallback thắng, giá tính theo model primary — xem `research.md` R7), kể cả khi lượt đó miễn phí cho người dùng.
- **FR-002**: Hệ thống MUST giữ nguyên mọi quyết định tính tiền user hiện tại: tuyến nào miễn phí thì vẫn miễn phí, hạn mức/quota/cơ chế chuyển tuyến và từ chối không thay đổi.
- **FR-003**: Báo cáo sử dụng AI theo tác vụ MUST hiển thị chi phí thật (bao gồm cả phần miễn phí cho user) cùng lượt dùng và số token.
- **FR-004**: Báo cáo tổng quan/lợi nhuận MUST tính lợi nhuận ròng từ chi phí thật (doanh thu trừ chi phí thật).
- **FR-005** (chi tiết hóa FR-001 cho tuyến free): Hệ thống MUST lưu số token thực tế cho cả các tác vụ đang miễn phí — phân tích ảnh lấy từ `usageMetadata`; embedding không có số liệu từ nhà cung cấp (spike T001) nên MUST ghi actual=0 kèm `error_code='usage_unreported'`; không được ước tính bịa số liệu trong mọi trường hợp thiếu số liệu.
- **FR-006**: Lượt gọi thất bại hoặc không xác minh được mức dùng MUST NOT được tính vào chi phí thật.
- **FR-007**: Khi thiếu giá model (bao gồm cả trường hợp provider/model rỗng), hệ thống MUST vẫn phục vụ user theo chính sách cũ; lượt đó MUST được ghi nhận với status Confirmed, `error_code='pricing_config_missing'`, chi phí thật bằng 0 để admin đối soát qua báo cáo drill-down có sẵn; không thêm cột/bảng mới.
- **FR-008**: Tài liệu của tính năng MUST kèm một đoạn SQL thủ công để bổ sung/cập nhật giá toàn bộ model đang dùng (7 model, xem `manual-seed.md`); đoạn SQL MUST có ngữ nghĩa gán đè dòng đang hiệu lực (`effective_to IS NULL`) theo cặp (nhà cung cấp, tên model) — đã tồn tại thì gán đè giá trị mới, chưa có thì thêm mới (không dùng `ON CONFLICT (provider, model)` vì unique thật là `(provider, model, effective_from)`).
- **FR-009**: Hệ thống MUST có kiểm thử tự động cho: lượt miễn phí sinh chi phí thật lớn hơn 0 nhưng không tính tiền user; lượt trả phí giữ nguyên cách tính như cũ; phân tích ảnh/embedding lưu được token thật.

### Key Entities

- **Lượt dùng AI**: một lần gọi AI, gồm tác vụ (trò chuyện, gợi ý phối đồ, tóm tắt, phân tích ảnh, embedding...), tuyến áp dụng (trả phí/miễn phí), số token thực tế, chi phí thật, chi phí tính cho user.
- **Chi phí thật (actual)**: số tiền dự án phải trả cho nhà cung cấp mô hình cho một lượt dùng — thực thể báo cáo mới cần ghi đúng.
- **Chi phí tính cho user (billed)**: số tiền/hạn mức khấu trừ của user cho một lượt dùng — giữ nguyên logic hiện tại, lượt miễn phí bằng 0.
- **Bảng giá model**: giá đầu vào/đầu ra trên một triệu token và tỉ giá quy đổi của từng cặp (nhà cung cấp, tên model), do vận hành cập nhật thủ công.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% lượt dùng miễn phí phát sinh mới có chi phí thật lớn hơn 0 khi model có giá công bố **và** có số liệu token (phân tích ảnh qua `usageMetadata`; embedding không có số liệu nên mang mã `usage_unreported` thay vì bịa số). Cách đo: truy vấn theo ngày trên môi trường verify — mọi lượt free có giá + có token đều có actual > 0; lượt thiếu giá/thiếu token mang mã đối soát tương ứng.
- **SC-002**: 100% kịch bản quota user (trong hạn mức, vượt hạn mức, hết lượt) cho kết quả cho phép/từ chối và số dư giống hệt trước khi triển khai — không có khoản thu mới.
- **SC-003**: Admin đối chiếu được chi phí trong báo cáo với bảng giá nhà cung cấp theo công thức số token nhân đơn giá trong vòng 5 phút cho bất kỳ dòng tác vụ nào.
- **SC-004**: Đoạn SQL thủ công chạy 2 lần liên tiếp (lần 2 đổi giá) đều thành công và giá cuối bằng giá lần 2, không lỗi trùng lặp.

## Assumptions

- Người dùng cuối không thấy thay đổi nào: không thêm màn hình, không đổi API thanh toán, không đổi gói cước.
- Giá trong bảng giá nội bộ được vận hành nhập tay từ bảng giá công bố của nhà cung cấp và được coi là đúng tại thời điểm nhập.
- Model tự vận hành nội bộ (nếu có) được cấu hình giá bằng 0 và chi phí thật bằng 0 là đúng.
- Dữ liệu lịch sử có token bằng 0 được giữ nguyên, không hồi tố.
- Giá output của model embedding bằng 0 theo cách thu tiền của Google (chỉ thu input); seed thủ công phản ánh đúng điều này thay vì quy ước cũ output=input.
- Tiền tệ báo cáo quy đổi theo tỉ giá cấu hình sẵn trong bảng giá, không cập nhật tỉ giá theo thời gian thực trong phạm vi tính năng này.
