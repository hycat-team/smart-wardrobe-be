# Feature Specification: Xử lý tech debt cross-module & tách schema theo module

**Feature Branch**: `011-assess-schema-separation`

**Created**: 2026-08-01

**Status**: Draft

**Input**: User description: "Dự án đang là modular monolith nhưng database vẫn chưa tách schema theo module. Trước khi tách schema, cần xem xét các tech debt: cross-module-db-access.md và dashboard-cross-module-queries.md. Feature này sẽ code luôn (không chỉ đánh giá), trong đó đánh giá và nghiên cứu là phase quan trọng trước khi code."

## Clarifications

### Session 2026-08-01

- Q: Xử lý bảng hub `users` (identity) khi tách schema → A: Tách identity ra schema riêng, giữ FK xuyên schema tới `users` (PostgreSQL hỗ trợ FK xuyên schema) — Option A.
- Q: Xử lý các FK xuyên module khác (không phải tới `users`) khi tách schema → A: Cắt toàn bộ FK xuyên module (trừ FK tới `users`), enforce integrity qua contract ở app layer — Option B.
- Q: Tách schema có kèm phân quyền DB-level (GRANT theo schema) không → A: Vẫn dùng 1 connection chung truy cập mọi schema, chỉ tách về mặt tổ chức/ownership — Option A.
- Q: Migration tách schema có cần Down/rollback đầy đủ không → A: Quyết định theo tiêu chí loại migration: (1) NẾU migration chỉ gồm `CREATE SCHEMA`, `ALTER TABLE ... SET SCHEMA`, `ALTER SEQUENCE ... SET SCHEMA`, `ALTER INDEX ... SET SCHEMA`, không thay đổi dữ liệu → VIẾT Down đầy đủ; (2) NẾU migration có chuyển đổi dữ liệu (`UPDATE`, `INSERT INTO ... SELECT`), merge/split bảng, drop cột, recalculate, thay đổi semantics dữ liệu → KHÔNG viết Down, dùng Backup + Restore.
- Q: Xét schema thực tế (`closy_db-schema.sql`, 42 bảng + sequence `deposit_transactions_order_code_seq` đều ở `public`), migration tách schema thuộc nhóm nào → A: Thuộc nhóm (1) — thuần `SET SCHEMA` cho table/sequence/index + `DROP CONSTRAINT`/`ADD CONSTRAINT` (cắt ~9 FK xuyên module, giữ ~30 FK tới `users`), KHÔNG có thao tác đổi dữ liệu → VIẾT Down đầy đủ. Lưu ý khi Down: tái lập FK xuyên module đã cắt phải check dữ liệu không orphan trước khi `ADD CONSTRAINT`.
- Q: Giữ tính atomic transaction giữa wardrobe ↔ fashion sau khi refactor qua contract → A: Chấp nhận eventual consistency — wardrobe ghi `wardrobe_items` xong publish event qua outbox, fashion module tự xử lý update `fashion_items` (event-driven, theo Hiến pháp III) — Option B.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Đánh giá & nghiên cứu trước khi code (Priority: P1)

Developer cần hiểu đúng toàn bộ phạm vi vi phạm cross-module trước khi bắt tay vào code. Việc này gồm: (1) đồng bộ 2 tài liệu tech debt với code thực tế, (2) cập nhật bản đồ schema trong `architecture.md` §VII, (3) tạo tài liệu đánh giá khả thi tách schema (phân loại vi phạm, FK xuyên module, chi phí, rủi ro, thứ tự thực hiện). Kết quả của phase này quyết định thứ tự và cách triển khai 2 sprint code tiếp theo.

**Why this priority**: Mọi quyết định code đều dựa trên hiểu đúng phạm vi. Tài liệu sai/lỗi thời dẫn đến code sai phạm vi, đánh giá sai chi phí.

**Independent Test**: Có thể test độc lập bằng cách đối chiếu 2 tài liệu tech debt + §VII với mã nguồn; mọi mô tả khớp 100% với code hiện tại. Tài liệu đánh giá trả lời đủ 3 câu hỏi: (1) phân loại và ưu tiên vi phạm? (2) FK xuyên module nào là rào cản, cắt được không? (3) kết luận khả thi, chi phí, rủi ro, thứ tự thực hiện?

**Acceptance Scenarios**:

1. **Given** tài liệu `cross-module-db-access.md` hiện chỉ liệt kê JOIN/UPDATE SQL trực tiếp, **When** Developer đối chiếu với `wardrobe_repo.go` và `fashion_item_repo_hybrid.go`, **Then** tài liệu bổ sung đầy đủ vi phạm import domain constants cross-module và ghi nhận `IWardrobeContract` đã có sẵn method phù hợp.
2. **Given** tài liệu `dashboard-cross-module-queries.md` đang liệt kê bảng `brand_conversations` và thiếu một số bảng, **When** Developer đối chiếu với 3 file repo dashboard, **Then** tài liệu gỡ bảng không còn tồn tại, bổ sung bảng thiếu (`user_subscriptions`, `loyalty_accounts`, `loyalty_tiers`, `brand_benefits`) và các vi phạm import domain constants.
3. **Given** §VII hiện liệt kê bảng không tồn tại (`loyalty_points`, `campaigns`, `digital_samples`, `sample_votes`), **When** Developer cập nhật, **Then** các bảng ma được thay bằng bảng mapping thực tế (42 bảng) gán đúng module sở hữu.
4. **Given** tài liệu đánh giá khả thi, **When** Developer đọc, **Then** toàn bộ FK xuyên module được liệt kê rõ ràng: **9 FK xuyên module phi-users cần CẮT** (đi qua contract) + **~30 FK trỏ tới `users` được GIỮ xuyên schema**, đồng thời nêu rõ kết luận khả thi + chi phí + rủi ro + thứ tự thực hiện cho 2 sprint sau.

---

### User Story 2 - Xử lý cross-module DB access wardrobe ↔ fashion (Priority: P1) — Sprint 1

Wardrobe module đang JOIN/UPDATE trực tiếp bảng `fashion_items`/`categories` (12 vị trí trong `wardrobe_repo.go`, gồm 4 UPDATE — hiện trạng; sau refactor: 1 UPDATE reassign category giữ đồng bộ, 3 UPDATE claim/processing state chuyển sang outbox theo FR-010), và fashion module đang query bảng `wardrobe_items` (`GetHybridCandidates`). Sau phase đánh giá, Developer refactor để mọi truy cập đi qua contract, đúng Hiến pháp I & II.

**Why this priority**: Đây là nhóm vi phạm HIGH nhất — gồm cả thao tác GHI (UPDATE) vào bảng module khác. Không xử lý thì không thể tách schema. Là tiền đề bắt buộc cho Sprint 2.

**Independent Test**: Có thể test độc lập bằng `go vet`/scan imports: không còn file nào JOIN/UPDATE bảng của module khác; mọi truy cập qua contract interface. Worker claim/processing vẫn hoạt động đúng (unit test + chạy worker).

**Acceptance Scenarios**:

1. **Given** `wardrobe_repo.go` đang `LEFT JOIN fashion_items` và `UPDATE fashion_items` trực tiếp, **When** refactor xong, **Then** wardrobe module không còn JOIN/UPDATE bảng fashion, mọi thao tác đi qua method contract do fashion module expose.
2. **Given** `fashion_item_repo_hybrid.go` `GetHybridCandidates` đang query bảng `wardrobe_items` trực tiếp, **When** refactor theo phương án đã chốt, **Then** wardrobe module expose `GetUserFashionItemIDs` qua `IWardrobeContract`, fashion module query `fashion_items WHERE id IN (...) ORDER BY embedding <=> ?` trên bảng của mình — tận dụng HNSW index.
3. **Given** các UPDATE processing state nằm trong transaction với update `wardrobe_items` (critical path worker), **When** refactor, **Then** chuyển sang mô hình eventual consistency: wardrobe ghi `wardrobe_items` + publish outbox event trong cùng transaction, fashion consume event và update `fashion_items` bất đồng bộ; riêng đọc stale info + reassign category giữ đồng bộ qua contract.
4. **Given** import domain constants cross-module đã được hiến pháp cho phép, **When** refactor, **Then** module dùng symbol constant (không hardcode giá trị).

---

### User Story 3 - Xử lý dashboard cross-module queries (Priority: P1) — Sprint 1

Dashboard module đang query trực tiếp bảng của 5 module khác (brand, subscription, aiexecutor, wardrobe, fashion) trong 3 file repo (`dashboard_brand_repo.go`, `dashboard_admin_repo.go`, `dashboard_user_repo.go`). Sau phase đánh giá, Developer refactor để dashboard gọi qua contract reader của từng module sở hữu.

**Why this priority**: Tuy read-only (rủi ro thấp) nhưng đây là nhóm vi phạm lớn nhất về số lượng (~18 query). Phải xử lý xong trong Sprint 1 để Sprint 2 tách schema khả thi.

**Independent Test**: Có thể test độc lập bằng cách chạy các API dashboard sau refactor; dữ liệu trả về giống hệt trước refactor (snapshot test). Không còn repo dashboard nào query bảng module khác.

**Acceptance Scenarios**:

1. **Given** 3 file repo dashboard đang query trực tiếp bảng module khác, **When** refactor xong, **Then** dashboard chỉ truy cập qua contract reader (`IBrandDashboardReader`, `ISubscriptionDashboardReader`, `IAICostDashboardReader`, `IWardrobeDashboardReader`, `IFashionDashboardReader`).
2. **Given** các contract reader hiện có chưa đủ method cho dashboard, **When** refactor, **Then** thêm method cần thiết vào contract của module sở hữu, dashboard gọi qua contract thay vì query thẳng DB.
3. **Given** fashion module chưa có dashboard reader, **When** refactor, **Then** tạo `IFashionDashboardReader` để phục vụ query `fashion_items`/`categories` của dashboard.

---

### User Story 4 - Tách database schema vật lý theo module (Priority: P2) — Sprint 2

Sau khi Sprint 1 hoàn tất (không còn cross-module DB access), Developer tách schema vật lý: mỗi module có schema PostgreSQL riêng (vd: `wardrobe`, `fashion`, `brand`, `subscription`, ...), cắt các FK xuyên module, cấu hình search_path/DSN/GORM cho phù hợp.

**Why this priority**: Đây là mục tiêu cuối cùng nhưng phụ thuộc toàn bộ vào Sprint 1 — chỉ khả thi khi không còn truy cập DB xuyên module.

**Independent Test**: Có thể test độc lập bằng cách chạy toàn bộ test suite + các luồng nghiệp vụ chính (worker processing, AI recommendation, dashboard, payment) sau khi tách schema; mọi query trỏ đúng schema của module sở hữu.

**Acceptance Scenarios**:

1. **Given** database đang ở schema `public` duy nhất, **When** tách schema xong, **Then** mỗi module sở hữu schema riêng, không còn bảng lẫn trong `public`.
2. **Given** các FK xuyên module đã được liệt kê ở phase đánh giá, **When** tách schema, **Then** FK trỏ tới `users` được giữ (FK xuyên schema), các FK xuyên module còn lại được cắt và integrity được enforce qua contract ở app layer.
3. **Given** migration hiện tại đang chạy trên `public` (42 bảng + sequence `deposit_transactions_order_code_seq`), **When** tách schema, **Then** có migration chuyển đổi thuần `SET SCHEMA` + drop/add constraint (không đổi dữ liệu), có Down đầy đủ để rollback về `public`, không phá migration consolidated hiện tại, config DSN/search_path/GORM cập nhật đúng.
4. **Given** `users` là bảng hub trung tâm, **When** tách schema, **Then** identity được tách ra schema riêng, các module khác giữ FK xuyên schema trỏ tới `users` (PostgreSQL hỗ trợ FK xuyên schema, giữ referential integrity).

---

### Edge Cases

- Bảng đã bị drop (`dashboard_brand_item_daily_snapshots`, các bảng community/resale) — loại khỏi danh sách FK xuyên module.
- Migration consolidated (`20260628000000_closy_b2b2c_rebuild.sql`) chứa nhiều phần Up/Down lồng nhau — rủi ro khi viết migration chuyển đổi schema.
- Import domain constants cross-module được hiến pháp cho phép (exception), nhưng hardcode giá trị vẫn là vi phạm — phân biệt rõ 2 loại.
- `users` là bảng hub bị gần như mọi module tham chiếu — được xử lý bằng cách giữ FK xuyên schema (không cắt, không chuyển UUID thuần).
- Vector similarity search phải tiếp tục dùng HNSW index sau khi tách schema (giữ hiệu năng).
- Worker processing (claim retry, processing state) hiện chạy transaction atomic xuyên bảng — được tái cấu trúc sang eventual consistency (outbox event), chấp nhận độ trễ nhỏ giữa cập nhật `wardrobe_items` và `fashion_items`; cần xử lý race condition khi 2 worker claim cùng item (row lock `FOR UPDATE` trên bảng sở hữu).

## Requirements *(mandatory)*

### Functional Requirements

**Phase 1 — Đánh giá & nghiên cứu:**

- **FR-001**: Tài liệu `cross-module-db-access.md` MUST được cập nhật liệt kê đầy đủ vị trí vi phạm (JOIN/UPDATE SQL + import domain constants) khớp 100% với code hiện tại.
- **FR-002**: Tài liệu `dashboard-cross-module-queries.md` MUST được cập nhật: gỡ bảng không còn tồn tại (`brand_conversations`), bổ sung bảng thiếu (`user_subscriptions`, `loyalty_accounts`, `loyalty_tiers`, `brand_benefits`) và các vi phạm import domain constants.
- **FR-003**: Mục §VII của `docs/system-design/architecture.md` MUST được thay bằng bảng mapping schema thực tế (42 bảng) theo module sở hữu, không còn bảng ma.
- **FR-004**: Tài liệu đánh giá khả thi MUST được tạo tại `docs/temp/technical-debt/` gồm: phân loại vi phạm, danh sách FK xuyên module + khả năng cắt bỏ, kết luận khả thi + chi phí + rủi ro + thứ tự thực hiện.

**Sprint 1 — Xử lý tech debt:**

- **FR-005**: Wardrobe module MUST không còn JOIN/UPDATE trực tiếp bảng `fashion_items`/`categories`; mọi thao tác qua contract do fashion module expose.
- **FR-006**: Fashion module MUST expose các contract method cho các thao tác **ĐỒNG BỘ còn giữ** do wardrobe gọi: đọc stale processing info (`GetStaleProcessingInfo`) và reassign category (`ReassignCategoryForCatalogItems`). Các thao tác claim retry / cập nhật processing state trên `fashion_items` **KHÔNG đi qua contract đồng bộ** — chuyển sang outbox event do fashion consumer xử lý (phân định theo FR-010).
- **FR-007**: Fashion module MUST không còn query trực tiếp bảng `wardrobe_items`; dùng `GetUserFashionItemIDs` từ `IWardrobeContract` + query `fashion_items WHERE id IN (...) ORDER BY embedding <=> ?`.
- **FR-008**: Dashboard module MUST không còn query trực tiếp bảng của brand/subscription/aiexecutor/wardrobe/fashion; chỉ truy cập qua contract reader.
- **FR-009**: Các contract reader hiện có (`IBrandDashboardReader`, `ISubscriptionDashboardReader`, `IAICostDashboardReader`, `IWardrobeDashboardReader`) MUST được bổ sung method cần thiết; `IFashionDashboardReader` MUST được tạo mới.
- **FR-010**: Các thao tác ghi `fashion_items` hiện nằm trong transaction critical path worker MUST được tái cấu trúc theo hướng eventual consistency: wardrobe ghi `wardrobe_items` xong publish event qua outbox (transactional outbox, Hiến pháp III), fashion module consume event và tự update `fashion_items`; không còn yêu cầu atomic giữa 2 bảng. **Phân định rõ**: claim retry + processing state (mark failed / needs review / complete success) → outbox event; đọc stale info + reassign category → đồng bộ qua contract (FR-006).
- **FR-011**: Import domain constants cross-module MUST dùng symbol constant, không hardcode giá trị.

**Sprint 2 — Tách schema vật lý:**

- **FR-012**: Mỗi module MUST có schema PostgreSQL riêng (gồm cả identity); không còn bảng module trong schema `public` chung.
- **FR-013**: Toàn bộ FK xuyên module MUST được xử lý theo phương án trong tài liệu đánh giá: các FK trỏ tới `users` (hub trung tâm) được GIỮ dưới dạng FK xuyên schema (PostgreSQL hỗ trợ); các FK xuyên module khác (vd: `wardrobe_items→fashion_items`, `brand_items→fashion_items`, `digital_sample_responses→outfits`, `subscription_plans→ai_cost_policies`, `ai_policy_grants→subscription_plans`, dashboard snapshots→`brands`) được CẮT và chuyển sang enforce integrity qua contract ở app layer.
- **FR-014**: Migration chuyển đổi schema MUST tuân theo tiêu chí: migration thuần chuyển schema (chỉ `CREATE SCHEMA` + `SET SCHEMA` cho table/sequence/index + `DROP CONSTRAINT`/`ADD CONSTRAINT`, không đổi dữ liệu) MUST viết Down đầy đủ; migration có chuyển đổi dữ liệu (UPDATE/INSERT...SELECT, merge/split, drop cột, recalculate, đổi semantics) KHÔNG viết Down, thay vào đó phải Backup + Restore an toàn. Áp dụng: migration tách schema của feature này thuộc loại thuần chuyển schema → VIẾT Down đầy đủ (42 bảng + sequence `deposit_transactions_order_code_seq` từ `public` sang schema module). Khi Down tái lập FK xuyên module đã cắt, phải check dữ liệu không orphan trước khi `ADD CONSTRAINT`.
- **FR-015**: Cấu hình DSN/search_path/GORM MUST được cập nhật để mỗi module truy cập đúng schema của mình, đồng thời đảm bảo FK xuyên schema tới `users` vẫn hoạt động; ứng dụng vẫn dùng 1 connection chung (không phân quyền DB-level theo module).

### Key Entities

- **Database schema (PostgreSQL `public`)**: Tập hợp 42 bảng đang hoạt động, mỗi bảng thuộc một module sở hữu (identity, wardrobe, fashion, brand, subscription, aiexecutor, dashboard).
- **Module boundary**: Ràng buộc kiến trúc quy định module chỉ truy cập dữ liệu module khác qua `presentation/contract/`, không JOIN/preload/import entity xuyên module.
- **Foreign Key xuyên module**: Các quan hệ FK vượt biên module (`wardrobe_items→fashion_items`, `brand_items→fashion_items`, `digital_sample_responses→outfits`, `subscription_plans→ai_cost_policies`, `ai_policy_grants→subscription_plans`, dashboard snapshots→brands/users, ...) — rào cản chính khi tách schema.
- **Contract interface**: Các interface trong `presentation/contract/` của từng module (bao gồm `dashboard_reader.go`) — kênh duy nhất cho phép truy cập dữ liệu cross-module.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% vị trí vi phạm cross-module được liệt kê trong 2 tài liệu tech debt khớp với code hiện tại (không còn discrepancy).
- **SC-002**: 0 bảng không tồn tại còn sót trong `architecture.md` §VII sau khi cập nhật.
- **SC-003**: 100% FK xuyên module đang hoạt động được liệt kê và đánh giá khả năng cắt bỏ trong tài liệu đánh giá khả thi.
- **SC-004**: Sau Sprint 1, 0 truy cập DB xuyên module còn lại (kiểm chứng bằng scan code) — `go vet` và test suite pass.
- **SC-005**: Sau Sprint 2, toàn bộ test suite + các luồng nghiệp vụ chính (worker processing, AI recommendation, dashboard, payment) hoạt động đúng trên schema tách.
- **SC-006**: Không có hồi quy hiệu năng vector search (HNSW index vẫn được sử dụng) sau cả 2 sprint.
- **SC-007**: Latency claim retry của worker không tăng quá 100ms so với trước refactor (đo bằng unit/integration test + so sánh trước/sau).

## Assumptions

- Feature được triển khai theo 3 phase: Phase 1 đánh giá/nghiên cứu (làm trước) → Sprint 1 xử lý tech debt → Sprint 2 tách schema vật lý.
- Vector search đã chốt: dùng `GetUserFashionItemIDs` qua contract (không giữ raw query cross-module, không dùng exception hiến pháp).
- Xử lý bảng hub `users` đã chốt: tách identity ra schema riêng, giữ FK xuyên schema tới `users` (không cắt FK users, không dùng UUID thuần).
- Xử lý FK xuyên module đã chốt: cắt toàn bộ FK xuyên module (trừ FK tới `users`), enforce integrity qua contract ở app layer.
- Phân quyền DB-level đã chốt: không phân quyền theo module — vẫn dùng 1 connection chung, tách schema chỉ về mặt tổ chức/ownership.
- Migration rollback đã chốt theo tiêu chí: migration thuần `SET SCHEMA` (không đổi dữ liệu) viết Down; migration có chuyển đổi dữ liệu không viết Down, dùng Backup + Restore. Xét schema thực tế (42 bảng + 1 sequence đều ở `public`, chỉ có SET SCHEMA + drop/add constraint, không đổi dữ liệu) → migration tách schema VIẾT Down đầy đủ.
- Atomicity cross-module đã chốt: không giữ atomic giữa wardrobe ↔ fashion; chuyển sang eventual consistency qua outbox event (theo Hiến pháp III), chấp nhận độ trễ nhỏ. Phân định: claim/processing state → outbox; đọc stale info + reassign category → contract đồng bộ.
- Tài liệu đánh giá khả thi tạm đặt trong `docs/temp/technical-debt/`; ADR chính thức sẽ tạo khi bắt đầu Sprint 2.
- `architecture.md` §VII cần cập nhật; các mục khác (§III, §VI) chỉ cập nhật nếu phát sinh mâu thuẫn trong quá trình code.
- Ngôn ngữ tài liệu trong `specs/` và `docs/` viết bằng tiếng Việt (theo AGENTS.md), trừ tên kỹ thuật/identifier giữ nguyên tiếng Anh.
- Migration hiện tại (Goose) và quy tắc "không sửa `database/init`, mọi thay đổi qua migration mới" vẫn được tuân thủ.
