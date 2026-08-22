# Feature Specification: Audit Response Structs & BriefRes Pattern

**Feature Branch**: `013-briefres-audit`

**Created**: 2026-08-22

**Status**: Draft

**Input**: Audit toàn bộ response structs trong dự án SmartWardrobe Backend, đề xuất BriefRes pattern để tối ưu JSON size, SAU ĐÓ implement refactor các struct/API đã xác định theo mức ưu tiên. Feature 013 bao gồm cả audit (xác định) lẫn implement (thực thi refactor) — giai đoạn tasks sẽ lên kế hoạch implement chi tiết.

## Clarifications

### Session 2026-08-22

- Q: Phạm vi audit có bao gồm admin/portal endpoints không? → A: Quét đầy đủ 9 modules (bao gồm admin/portal), nhưng với admin/portal chỉ ghi nhận "không đề xuất refactor".
- Q: Chính sách backward compatibility khi đề xuất BriefRes? → A: Breaking change — bỏ field cũ, thay bằng BriefRes, thông báo qua frontend-note.md.
- Q: Giai đoạn thực hiện audit — ai làm & khi nào? → A: Thực hiện trong giai đoạn /speckit-plan. Tạo 1 sub-agent cho 1 module, mỗi sub-agent tạo 1 file report riêng trong folder reports/ nằm trong specs/013/.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Xác định danh sách struct cần BriefRes (Priority: P1)

Dev (backend) cần có danh sách đầy đủ các response struct trong toàn dự án bị **overdelivery** (trả quá nhiều field không cần thiết cho context hiển thị), kèm đề xuất fields giữ và fields bỏ.

**Why this priority**: Đây là nền tảng của toàn bộ feature — không có danh sách chính xác thì không thể biết scope refactor.

**Independent Test**: Có thể kiểm tra độc lập bằng cách đối chiếu danh sách struct trong spec với thực tế `application/dto/` của từng module trong `internal/modules/` (admin, aiexecutor, brand, dashboard, fashion, identity, notification, subscription, wardrobe).

**Acceptance Scenarios**:

1. **Given** danh sách các module trong `internal/modules/`, **When** audit response structs, **Then** mỗi module đều được quét và liệt kê các struct có > 8 fields mà context chỉ cần 3-5 fields.
2. **Given** một struct nhúng entity lớn (ví dụ BrandRes, UserRes), **When** đánh giá, **Then** spec chỉ rõ fields cần giữ cho context hiển thị và fields nên bỏ.

---

### User Story 2 - Xác định API trả bare ID thiếu nested data (Priority: P1)

Dev cần biết các API nào trả UUID foreign key mà không có thông tin related entity, buộc frontend phải gọi thêm API để lấy dữ liệu hiển thị — những nơi này nên thay bare ID bằng BriefRes struct.

**Why this priority**: Đây là vấn đề UX/performance (N+1 request từ frontend), cùng mức độ quan trọng với overdelivery.

**Independent Test**: Có thể kiểm tra độc lập bằng cách rà soát các response struct có field kiểu `uuid.UUID` (như `brandId`, `benefitId`, `userId`) mà không kèm struct info đi kèm.

**Acceptance Scenarios**:

1. **Given** một response struct có field `xxxId` (UUID thuần), **When** đánh giá, **Then** spec đánh dấu đó là underdelivery nếu frontend cần hiển thị thông tin related entity nhưng không có sẵn.
2. **Given** một API trả nested struct thiếu field hiển thị quan trọng (ví dụ có user object nhưng thiếu avatarUrl), **When** đánh giá, **Then** spec ghi nhận field cần bổ sung.

---

### User Story 3 - Phân loại ưu tiên & open questions (Priority: P2)

Dev cần spec liệt kê thứ tự ưu tiên refactor (cao/trung bình/thấp) và các open questions cần quyết định trước khi lên plan (backward compat, scope refactor theo module, v.v.).

**Why this priority**: Phân loại giúp dev biết bắt đầu từ đâu; open questions giúp tránh bỏ sót quyết định thiết kế quan trọng ở giai đoạn plan.

**Independent Test**: Kiểm tra spec có ít nhất 3 mức ưu tiên, mục Open Questions rõ ràng (mỗi câu hỏi có context + phương án đề xuất), và folder `specs/013-briefres-audit/reports/` chứa 1 file report cho mỗi module.

**Acceptance Scenarios**:

1. **Given** danh sách các struct/API được đề xuất, **When** phân loại, **Then** mỗi mục đều được gắn mức ưu tiên cao/trung bình/thấp kèm lý do.
2. **Given** spec hoàn chỉnh, **When** đọc phần Open Questions, **Then** dev hiểu rõ các quyết định cần đưa ra trước khi refactor (ví dụ có giữ field cũ cho backward compat hay không).

### User Story 4 - Implement refactor BriefRes theo ưu tiên (Priority: P1)

Dev (backend) thực thi refactor dựa trên kết quả audit: tạo các struct BriefRes mới, trim field thừa, bổ sung nested data thiếu, cập nhật mapper + handler + swagger cho các API đã xác định.

**Why this priority**: Đây là pha thực thi — biến kết quả audit thành thay đổi code thực tế, giảm JSON payload và vá lỗi underdelivery/bảo mật.

**Independent Test**: Có thể kiểm tra độc lập bằng cách chạy `make check` (fmt + wire + swagger) và `make test` sau khi implement; đối chiếu từng API trong `contracts/affected-api-contracts.md` với response đề xuất.

**Acceptance Scenarios**:

1. **Given** danh sách BriefRes đề xuất (research.md §3), **When** implement, **Then** mỗi BriefRes mới được tạo tại module sở hữu entity + mapper tương ứng trong `application/mapper/`.
2. **Given** các struct overdelivery/underdelivery đã xác định, **When** implement, **Then** field được trim/thêm theo đúng bảng fields giữ vs bỏ, không ảnh hưởng admin/portal endpoints.
3. **Given** implement hoàn tất, **When** chạy `make check` + `make test`, **Then** không có lỗi build/test, swagger phản ánh struct mới.
4. **Given** toàn bộ API thay đổi đã hoàn tất và verify xong, **When** tạo `frontend-note.md`, **Then** file liệt kê đầy đủ API thay đổi, field cũ → field mới, hướng dẫn frontend sửa.

---

- Struct vừa overdelivery vừa underdelivery (vừa thừa field internal vừa thiếu nested data) — xử lý thế nào khi đề xuất.
- Struct chỉ dùng 1 lần trong 1 endpoint duy nhất — có nên tạo BriefRes riêng hay inline fields?
- Response dùng chung giữa nhiều endpoint với context khác nhau (list vs detail) — có nên tách BriefRes riêng cho từng context?
- Struct đã là BriefRes sẵn (BrandItemBriefRes, LoyaltyTierBriefRes, CategoryBriefRes) — có cần điều chỉnh không?
- Struct phục vụ internal (event messaging, search indexing) có bị ảnh hưởng không?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Spec MUST liệt kê tất cả response struct trong toàn bộ module có trong `internal/modules/`: admin, aiexecutor, brand, dashboard, fashion, identity, notification, subscription, wardrobe.
- **FR-002**: Spec MUST xác định các struct **overdelivery** (quá nhiều field cho context hiển thị) kèm danh sách fields giữ vs fields bỏ.
- **FR-003**: Spec MUST xác định các struct/API **underdelivery** (trả bare ID thiếu nested data) kèm đề xuất BriefRes cần tạo mới.
- **FR-004**: Spec MUST đề xuất tên + danh sách fields cho từng BriefRes mới (nếu có).
- **FR-005**: Spec MUST phân loại từng mục refactor theo ưu tiên: cao / trung bình / thấp, kèm lý do.
- **FR-006**: Spec MUST liệt kê các API endpoint bị ảnh hưởng cho mỗi thay đổi đề xuất (response hiện tại vs đề xuất).
- **FR-007**: Feature 013 gồm 2 pha: (1) **Audit** — xác định vấn đề, đề xuất BriefRes, KHÔNG code ở pha này; (2) **Implement** — tạo các struct BriefRes mới, trim field, cập nhật mapper + handler + swagger theo mức ưu tiên. KHÔNG thay đổi database schema, không thay đổi request DTO — chỉ tập trung vào response DTO + mapper.
- **FR-008**: Spec MUST liệt kê các open questions cần dev quyết định trước khi lên plan, kèm default answer (research.md §5). Policy backward compat đã chốt: **breaking change** — bỏ field cũ, thay bằng BriefRes, thông báo qua frontend-note.md. Nếu dev không phản đối, implement theo default answers để tránh rework.
- **FR-009**: Spec MUST xác định các struct đã là BriefRes (BrandItemBriefRes, LoyaltyTierBriefRes, CategoryBriefRes, v.v.) và đánh giá có cần điều chỉnh hay không.
- **FR-010**: Spec MUST quét ĐẦY ĐỦ 9 modules (bao gồm admin/portal endpoints), nhưng với admin/portal endpoints chỉ ghi nhận trạng thái "không đề xuất refactor" — vẫn giữ full fields cho context quản trị.
- **FR-011**: Tạo file **frontend-note.md** ở CUỐI giai đoạn sau khi giai đoạn thực hiện (implement) và kiểm tra lại (verify/converge) đã hoàn thiện toàn bộ — nghĩa là chỉ khi tất cả API thay đổi đã hoàn tất và xác nhận đúng mới lên frontend-note.md. File này dùng để thông báo danh sách các API bị thay đổi response và hướng dẫn frontend cần sửa gì (cách gọi API, cách render, field cũ → field mới).
- **FR-012**: Việc quét (audit) codebase được thực hiện trong giai đoạn `/speckit-plan`: tạo 1 sub-agent cho 1 module, mỗi sub-agent tạo 1 file report riêng trong folder `specs/013-briefres-audit/reports/`. Kết quả tổng hợp nằm trong plan.md.
- **FR-013**: Implement MUST tạo các struct BriefRes mới (theo research.md §3) tại module sở hữu entity, kèm mapper trong `application/mapper/`, và cập nhật response struct của các API bị ảnh hưởng (contracts/affected-api-contracts.md) theo mức ưu tiên.
- **FR-014**: Implement MUST trim field thừa (overdelivery) và bổ sung nested data (underdelivery) cho các API user/public đã xác định — KHÔNG ảnh hưởng admin/portal endpoints.
- **FR-015**: Implement MUST đảm bảo `make check` + `make test` pass sau mỗi module refactor; swagger được regenerate để phản ánh struct mới.

### Key Entities *(include if feature involves data)*

- **Response Struct (DTO)**: Các struct trong `application/dto/` của từng module trong `internal/modules/` — đối tượng chính của audit.
- **BriefRes**: Struct response tối giản đề xuất tạo mới, chỉ chứa fields cần cho context hiển thị (tương tự BrandBriefRes đã có).
- **API Endpoint**: Điểm tiếp xúc frontend-backend, là nơi xác định context sử dụng của từng response struct.
- **Related Entity**: Brand, User, Benefit, Category, Subscription, v.v. — nguồn dữ liệu cho nested BriefRes khi thay bare ID.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Spec phủ được 100% các module có `application/dto/` trong `internal/modules/` (admin, aiexecutor, brand, dashboard, fashion, identity, notification, subscription, wardrobe). Mỗi module có 1 file report riêng trong `specs/013-briefres-audit/reports/`.
- **SC-002**: Mỗi struct được đề xuất đều có danh sách fields giữ vs bỏ cụ thể, không để trống.
- **SC-003**: Mỗi đề xuất đều có mức ưu tiên (cao/trung bình/thấp) và lý do rõ ràng.
- **SC-004**: Implement chỉ tác động response DTO + mapper + handler/swagger — KHÔNG thay đổi database schema, không thay đổi request DTO, không thay đổi entity GORM.
- **SC-005**: Spec có mục Open Questions chứa các quyết định thiết kế quan trọng cần dev trả lời trước giai đoạn plan.
- **SC-006**: File **frontend-note.md** được tạo sau khi toàn bộ API đã implement và verify xong, liệt kê đầy đủ các API bị thay đổi, field cũ → field mới, và hướng dẫn frontend cần sửa trước khi deploy.
- **SC-007**: Implement tạo đủ các BriefRes mới đã đề xuất theo thứ tự ưu tiên (brand → identity/wardrobe → subscription → fashion/dashboard → aiexecutor), không để sót mục ưu tiên cao. Các mục ưu tiên thấp/conditional (B10 ChatSessionBriefRes, rút gọn BrandBriefRes) chỉ thực hiện khi dev chốt.
- **SC-008**: Sau implement, `make check` + `make test` pass; không có endpoint admin/portal nào bị thay đổi response (verify qua git diff + swagger).

## Assumptions

- Audit được thực hiện trên toàn bộ module trong `internal/modules/` (admin, aiexecutor, brand, dashboard, fashion, identity, notification, subscription, wardrobe), có thể tham khảo các báo cáo trước đó tại `docs/temp/` (report-api-brand-fields-audit.md, report-brandloyaltyres-brandres-analysis.md) làm nền tảng.
- Việc quét được thực hiện trong giai đoạn `/speckit-plan` với 1 sub-agent/module; mỗi sub-agent tạo 1 file report trong folder `specs/013-briefres-audit/reports/`.
- Mục **frontend-note.md** được tạo ở CUỐI giai đoạn thực hiện (implement) + kiểm tra lại (verify/converge) đã hoàn thiện toàn bộ API — không tạo ở giai đoạn plan. File này dùng để chuyển cho team frontend, thông báo các API đã thay đổi và những gì frontend cần sửa.
- BrandBriefRes đã tồn tại từ feature 012 — các đề xuất BriefRes mới nên follow cùng naming convention (`<Entity>BriefRes`).
- Feature 013 bao gồm cả audit lẫn implement: giai đoạn plan thực hiện audit (9 sub-agent/module, reports/), giai đoạn tasks lên kế hoạch implement refactor, giai đoạn implement code theo kế hoạch, cuối cùng tạo frontend-note.md.
- Các struct đã là BriefRes sẵn trong codebase sẽ được đánh giá lại nhưng không mặc định thay đổi.
- Chính sách backward compat đã chốt: **breaking change** — field cũ bị bỏ và thay bằng BriefRes; mọi thay đổi response đều được ghi nhận trong frontend-note.md để frontend cập nhật.
- Implement KHÔNG thay đổi database schema, không thay đổi request DTO — chỉ tập trung vào response DTO và mapper (tạo BriefRes, trim field, cập nhật mapper + handler + swagger).
- Phạm vi "toàn bộ API" bao gồm cả các endpoint public lẫn authenticated và admin/portal. Với admin/portal chỉ ghi nhận không refactor (giữ full fields cho context quản trị); ưu tiên implement BriefRes cho các context hiển thị của frontend.