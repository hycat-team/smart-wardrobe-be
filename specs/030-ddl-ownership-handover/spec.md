# Đặc tả tính năng: Bàn giao quyền sở hữu DDL (baseline rewrite)

**Feature Branch**: `030-ddl-ownership-handover`

**Created**: 2026-10-16

**Status**: Draft

**Input**: Bàn giao quyền sở hữu DDL triệt để (baseline rewrite) cho 027/D-06 mở rộng — 7 bảng `aiexecutor.*` sang service AI, 3 bảng `dashboard.*` sang service Commerce, kèm gate lint, rename constraint mồ côi, và quy trình cutover an toàn cho production đang có user thật.

## Clarifications

### Session 2026-10-16

- Q: Khi cutover rename constraint trên production đang có user thật, traffic app ở chế độ nào? → A: Read-only window — ĐÃ THAY THẾ bởi quyết định full-stop ở bullet 4 bên dưới (chuẩn duy nhất hiện hành).
- Q: Môi trường bị quên (staging bỏ xó, laptop dev, backup cũ restore lên) xử lý thế nào? → A: Tự hội tụ (boot bản mới là tự hội tụ nhờ migration idempotent, không cần can thiệp tay).
- Q: Lệnh ngừng bắn migration mới (FR-011) thực thi bằng cách nào? → A: Quy ước thủ công (quy ước nhóm + reviewer kiểm tra tay, không thêm check máy cho lệnh ngừng bắn tạm thời).
- Q: Cutover production ai thực hiện, ở chế độ traffic nào? → A: Chủ hệ thống thao tác tay theo runbook trong cửa sổ full-stop 1–2h không có user truy cập (do owner tự quyết thời điểm); không cần cơ chế chặn read-only tự động hay CI chặn gì cả.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Triển khai phiên bản mới không còn hai service cùng sửa một schema (Priority: P1)

Người vận hành triển khai các service độc lập (Account, Commerce, AI) mà không lo migration của service này ghi đè hoặc xung đột schema do service khác sở hữu. Mỗi nhóm bảng chỉ có đúng một service được phép thay đổi cấu trúc, ở mọi môi trường.

**Why this priority**: Đây là nguyên nhân gốc của D-06 — bảng thuộc schema của service AI nhưng lịch sử tạo/sửa nó nằm ở service Account. Chỉ cần một lần hai bên cùng sửa là production đang có user thật sẽ kẹt migrate lúc boot.

**Independent Test**: Triển khai đồng thời 3 service lên một database trắng và một database đã có dữ liệu — cả hai đều boot xanh, không service nào báo thiếu bảng hay xung đột version.

**Acceptance Scenarios**:

1. **Given** database trắng chưa có gì, **When** khởi động đồng thời cả 3 service, **Then** mọi bảng được tạo đầy đủ, service nào thiếu điều kiện tiên quyết thì báo lỗi rõ ràng và tự thử lại cho tới khi thành công thay vì bỏ qua im lặng.
2. **Given** database đang chạy chứa dữ liệu thật, **When** triển khai phiên bản bàn giao, **Then** không mất hay đổi 1 dòng dữ liệu nào; kiểm tra cấu trúc trước/sau chỉ khác đúng tên constraint đã thống nhất đổi.
3. **Given** đã bàn giao xong, **When** một lập trình viên vô tình viết thay đổi cấu trúc `aiexecutor` vào cây migration của Account, **Then** kiểm tra tự động báo đỏ trước khi được merge.
4. **Given** database trắng chưa có gì, **When** cả 3 service khởi động và migrate cùng lúc, **Then** lock phiên advisory serialize toàn bộ DDL (không báo bảng đã tồn tại, không deadlock, không xung đột schema) và cả 3 service đều boot xanh.

---

### User Story 2 - Nhìn vào lịch sử migration là biết schema thuộc về ai (Priority: P1)

Chủ sản phẩm và DBA mở cây migration của từng service là thấy ngay service nào sở hữu nhóm bảng nào, không còn tình trạng "bảng của AI nhưng file tạo nằm ở Account" hay "cột dashboard do Account thêm".

**Why this priority**: Mọi quyết định index, constraint, audit sau này (ví dụ index truy vấn theo kỳ của 029/D-01) đều cần biết viết vào đúng service chủ. Không có ownership rõ ràng thì mỗi lần sửa schema là một lần đánh cược.

**Independent Test**: Liệt kê toàn bộ file migration chạm tới `aiexecutor.*` — 100% nằm ở cây AI; chạm tới `dashboard.*` — 100% thay đổi mới nằm ở cây Commerce; cây Account không còn dòng nào chạm 2 nhóm schema này.

**Acceptance Scenarios**:

1. **Given** cây migration sau bàn giao, **When** rà soát mọi file chạm `aiexecutor.*`, **Then** tất cả đều thuộc cây AI, không còn file nào ở cây Account chứa thay đổi cấu trúc nhóm này.
2. **Given** cây migration sau bàn giao, **When** rà soát mọi thay đổi `dashboard.*` phát sinh sau thời điểm bàn giao, **Then** tất cả đều thuộc cây Commerce.
3. **Given** bảng version của Account, **When** đếm số dòng trước và sau bàn giao, **Then** số dòng không đổi (mốc dev: 35 dòng, version cao nhất giữ nguyên) — lịch sử cũ được giữ nguyên vẹn, không mất dòng nào.

---

### User Story 3 - Cutover production có user thật mà không mất dữ liệu (Priority: P2)

Người vận hành (chính chủ hệ thống, thao tác tay theo runbook) đưa bản bàn giao lên production trong một cửa sổ full-stop 1–2h không có user truy cập (thời điểm do chủ hệ thống tự quyết): có backup đã kiểm chứng khôi phục được, đổi tên constraint diễn ra trong chốc lát có kiểm soát, và nếu bất cứ bước nào hỏng thì quay về bằng bản backup thay vì sửa chữa thủ công trên dữ liệu sống. Không yêu cầu cơ chế chặn read-only tự động.

**Why this priority**: Đổi tên constraint giữ lock độc quyền ngắn hạn trên bảng đang có traffic ghi. Không có cửa sổ + backup + rollback rõ ràng thì một sự cố nhỏ thành mất dữ liệu không đảo ngược được.

**Independent Test**: Diễn tập toàn bộ quy trình trên bản sao khôi phục từ backup (không phải production): đo thời gian giữ lock, xác nhận xong thì schema cuối đồng nhất với database dựng trắng từ lịch sử mới.

**Acceptance Scenarios**:

1. **Given** bản sao production, **When** chạy toàn bộ quy trình bàn giao, **Then** schema cuối đồng nhất 100% ở phạm vi handover với database dựng trắng từ lịch sử mới (10 bảng `aiexecutor`+`dashboard`: từng bảng, cột, index, constraint).
2. **Given** đang trong cửa sổ cutover, **When** bất kỳ bước kiểm tra nào đỏ, **Then** dừng lại và khôi phục từ backup thay vì sửa tiếp trên dữ liệu sống.
3. **Given** đã cutover xong, **When** người dùng đăng ký, mua gói, dùng AI, xem dashboard, **Then** mọi luồng nghiệp vụ chạy bình thường (smoke grant + dashboard xanh).
4. **Given** database dev hiện tại (schema y chang production), **When** chạy toàn bộ quy trình bàn giao trên đó sau khi đã backup, **Then** mọi kiểm tra pass — đây là cổng go/no-go: chỉ khi case này xanh mới được đặt giờ G production.

---

### User Story 4 - Môi trường kiểm chứng không bao giờ đe dọa dữ liệu thật (Priority: P2)

Người kiểm chứng quy trình bàn giao trên database tạm được tạo riêng cho lần kiểm chứng và xóa đúng database tạm đó sau khi xong — không bao giờ xóa volume hay chạm database chứa user thật, ở bất kỳ môi trường nào.

**Why this priority**: Production đang có user thật. Một lệnh dọn dẹp nhầm (xóa volume, xóa nhầm database) là mất dữ liệu không cứu được. Quy tắc an toàn phải nằm trong đặc tả, không thể trông chờ vào trí nhớ người chạy.

**Independent Test**: Đọc toàn bộ tài liệu và kịch bản kiểm chứng của tính năng — không tồn tại lệnh xóa volume hay lệnh trỏ vào database production.

**Acceptance Scenarios**:

1. **Given** cần database trắng để kiểm chứng, **When** làm theo tài liệu, **Then** database trắng là database tạm tên riêng (hoặc stack cách ly riêng), và bước dọn dẹp chỉ xóa đúng database/stack tạm đó.
2. **Given** cần kiểm chứng trên dữ liệu giống production, **When** làm theo tài liệu, **Then** nguồn duy nhất được phép là bản sao khôi phục từ backup, không bao giờ là production trực tiếp.

---

### Edge Cases

- Service AI khởi động trước khi Account tạo xong bảng trên database trắng: migration AI phải báo lỗi rõ ràng và thử lại, không được đánh dấu xong khi chưa làm gì.
- Một môi trường (staging bỏ xó, laptop dev, bản backup cũ) bị quên không nâng theo: khi boot bản mới, các migration hội tụ phải tự chữa lành nhờ tính idempotent thay vì kẹt — không yêu cầu runbook phẫu thuật tay cho từng môi trường sót.
- Đổi tên constraint đúng lúc bảng đang có ghi mới: lock chờ quá ngưỡng phải hủy an toàn và thử lại trong cửa sổ, không treo request của user.
- Công cụ migration đối chiếu nội dung file với lịch sử đã applied (nếu có): phải xác minh hành vi này trước khi viết lại nội dung file cũ, vì cả kế hoạch đứng trên giả định "chỉ track theo version".

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: Hệ thống MUST xác lập service AI là chủ sở hữu duy nhất mọi thay đổi cấu trúc của 7 bảng `aiexecutor` (`ai_cost_policies`, `ai_cost_policy_operations`, `ai_policy_grants`, `ai_usage_period_ledgers`, `ai_usage_events`, `ai_model_configs`, `ai_cost_policy_audit_logs`).
- **FR-002**: Hệ thống MUST xác lập service Commerce là chủ sở hữu mọi thay đổi cấu trúc của 3 bảng `dashboard` (`dashboard_admin_daily_snapshots`, `dashboard_brand_daily_snapshots`, `dashboard_user_wardrobe_summaries`) — nội dung cũ ở cây Account được chuyển sang baseline Commerce, thay đổi mới chỉ nằm ở cây Commerce.
- **FR-003**: Lịch sử migration cũ của Account MUST được giữ nguyên tên file và version (không xóa file, không cắt nội dung làm mất khả năng hoàn nguyên), chỉ viết lại nội dung để gỡ DDL ngoài chủ.
- **FR-004**: Constraint mồ côi mang tên bảng cũ trên bảng grant (`user_ai_policy_grants_check`) MUST được đổi sang tên tường minh (`ai_policy_grants_effective_window_check`) và MUST có đường hoàn nguyên đổi ngược lại.
- **FR-005**: Migration bàn giao của AI/Commerce MUST kiểm tra điều kiện tiên quyết (đủ bảng nguồn) trước khi làm việc; khi thiếu MUST báo lỗi rõ ràng và dừng lại để thử lại, MUST NOT đánh dấu hoàn thành khi chưa làm gì.
- **FR-006**: Mọi bước kiểm chứng trong tính năng này MUST NOT xóa volume Docker và MUST NOT trỏ vào database chứa user thật; database trắng chỉ là database/stack tạm có tên riêng và chỉ xóa đúng đối tượng tạm đó.
- **FR-007**: Cutover production do chủ hệ thống thao tác tay theo runbook, MUST chỉ diễn ra khi đã có bản backup mới nhất và đã kiểm chứng khôi phục được, trong cửa sổ full-stop 1–2h không có user truy cập (thời điểm do chủ hệ thống tự quyết); có giới hạn thời gian chờ lock và giám sát lock; rollback khi hỏng MUST là khôi phục từ backup. Không yêu cầu cơ chế chặn traffic tự động.
- **FR-008**: Bảng version của Account MUST giữ nguyên số dòng sau bàn giao — không mất, không thêm dòng nào từ tính năng này (mốc đo trên dev: 35 dòng, version cao nhất `20261011090002`; môi trường khác đối chiếu với điểm cắt đã ghi ở T001).
- **FR-009**: Hệ thống MUST có kiểm tra tự động chặn 3 trường hợp: thay đổi `aiexecutor`/`dashboard`/schema Commerce trong cây Account; thay đổi `identity`/`subscription`/`aiexecutor` trong cây Commerce (trừ khóa ngoại tới user đã cho phép); thay đổi ngoài `aiexecutor` trong cây AI.
- **FR-010**: Mọi tham chiếu tới tên constraint cũ trong kiểm thử, runbook và tài liệu MUST được cập nhật sang tên mới; mục D-06 trong `deferred.md` của 027 MUST được đánh dấu đã gỡ kèm dấu vết.
- **FR-011**: Trong suốt thời gian thực hiện, MUST cấm tạo migration mới chạm tới `aiexecutor.*` và `dashboard.*` cho tới khi bàn giao xong (lệnh ngừng bắn migration); lệnh này thực thi bằng quy ước nhóm và reviewer kiểm tra tay, không thêm check máy (check máy chỉ áp dụng cho gate lâu dài ở FR-009).

### Key Entities

- **Bản đồ sở hữu schema**: quy ước service nào sở hữu nhóm bảng nào (Account: `identity`/`subscription`; Commerce: `wardrobe`/`fashion`/`brand`/`dashboard`/`community`; AI: `aiexecutor`; hạ tầng chung: `shared`/`public`).
- **Lịch sử migration**: chuỗi versioned thay đổi cấu trúc của mỗi service, là bản ghi sự thật để dựng database trắng và hoàn nguyên.
- **Bảng version (sổ applied)**: sổ ghi version đã chạy của từng service trên từng database; là căn cứ để migration không chạy lại và để nghiệm thu "không mất dòng nào".
- **Migration hội tụ**: thay đổi cấu trúc viết theo dạng idempotent (chạy lại nhiều lần vẫn cùng kết quả) để database cũ (dựng từ nội dung cũ) và database mới (dựng từ nội dung mới) gặp nhau ở cùng schema cuối.
- **Cửa sổ cutover**: khoảng thời gian bảo trì đã duyệt để áp dụng thay đổi giữ lock trên production, kèm backup, giám sát và đường rollback.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% file migration chạm `aiexecutor.*` nằm ở cây AI và 100% thay đổi `dashboard.*` mới nằm ở cây Commerce (rà soát toàn bộ cây migration, 0 ngoại lệ).
- **SC-002**: Schema cuối của 2 đường kiểm chứng (database trắng dựng từ lịch sử mới vs bản sao production sau hội tụ) đồng nhất 100% ở phạm vi handover (10 bảng `aiexecutor`+`dashboard`: bảng/cột/index/constraint); drift ngoài phạm vi (nếu có) được ghi `deferred.md`, không chặn hội tụ.
- **SC-003**: Không còn tham chiếu tới tên constraint cũ trong toàn bộ mã nguồn, kiểm thử, runbook và tài liệu (tìm kiếm toàn repo trả về 0 kết quả, trừ dòng lịch sử trong tài liệu bàn giao).
- **SC-004**: Bảng version của Account giữ nguyên số dòng trước/sau (mốc dev: 35 dòng; môi trường khác theo điểm cắt T001); kiểm tra tự động chặn được 3 mẫu migration vi phạm (mỗi luật có 1 mẫu thử đỏ).
- **SC-005**: Cutover production hoàn thành trong cửa sổ đã duyệt với 0 mất mát dữ liệu (đếm dòng các bảng nghiệp vụ trước/sau bằng nhau) và 100% luồng smoke (đăng ký, mua gói, dùng AI, xem dashboard) xanh sau cutover.
- **SC-006**: Toàn bộ tài liệu và kịch bản của tính năng không chứa lệnh xóa volume hay lệnh trỏ vào database production (rà soát bằng tìm kiếm từ khóa, 0 kết quả).
- **SC-007**: Rehearsal trên database dev hiện tại pass 100% kiểm tra (schema đồng nhất, đếm dòng bằng nhau, boot xanh, smoke xanh) — đây là điều kiện bắt buộc trước giờ G production.

## Assumptions

- Số liệu baseline (35 dòng version Account, 3 dòng Commerce, AI chưa có bảng version, đủ 7 bảng `aiexecutor` + 6 bảng `community`) được đo thật trên dev ngày 2026-10-16; môi trường khác có thể lệch và phải chốt "điểm cắt version" riêng trước khi làm.
- Giả định công cụ migration chỉ theo dõi theo version, không đối chiếu nội dung file với lịch sử đã applied — điều này phải được xác minh bằng thực nghiệm trước khi viết lại nội dung file cũ; nếu sai thì toàn bộ phương án đổi sang phẫu thuật bảng version.
- Mọi môi trường (prod, staging, dev, CI) được nâng lên cùng 1 version Account mới nhất trước khi bắt đầu; môi trường nào chưa tới phải nâng trước.
- Từ "triệt để" nghĩa là cây Account sạch DDL ngoài chủ + quyền viết tương lai thuộc đúng service + gate chống tái phạm; không bao gồm việc tách database vật lý (vẫn 1 database dùng chung, tách theo schema).
- Tính năng này không thay đổi hành vi nghiệp vụ, API, hay DTO client-facing nên không cần frontend-guide (ghi nhận rõ trong `tasks.md` thay vì bỏ trống).
