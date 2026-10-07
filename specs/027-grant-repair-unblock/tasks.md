# Tasks: 027 Sửa tắc hai nguyên nhân chặn vĩnh viễn

**Input**: Tài liệu thiết kế từ `/specs/027-grant-repair-unblock/`

**Yêu cầu**: `plan.md`, `spec.md`, `research.md`, `data-model.md`,
`contracts/grant-window-resolution.md`, `contracts/error-classification.md`, `quickstart.md`,
`deferred.md`

**Kiểm thử**: CÓ. Đặc tả ghi rõ "Kiểm thử độc lập" cho cả ba user story, và mỗi story có
Acceptance Scenarios để kiểm chứng. Mỗi task kiểm thử dưới đây trỏ tới một hàng cụ thể
trong bảng của [quickstart.md](quickstart.md).

## Format: `[ID] [P?] [Story] Mô tả`

- **[P]**: chạy song song được (khác tệp, không phụ thuộc task chưa xong)
- **[Story]**: user story mà task thuộc về
- Mọi task có đường dẫn tệp chính xác

### Quy ước đường dẫn

Mọi tệp Go nằm trong
`services/ai/internal/modules/aiexecutor/application/usecase/grant/` trừ khi ghi rõ khác.
Viết tắt: `GRANT/` = đường dẫn đó.

> **Cảnh báo xung đột tệp**: US1 và US3 đều sửa `grant_uc.go`. Chúng **không** chạy song
> song được. Xem mục Dependencies.

---

## Phase 1: Setup

**Mục đích**: Xác nhận điểm xuất phát. 027 không thêm gói phụ thuộc mới, không migration,
không endpoint mới — nên phase này nhẹ, và đó là điều đúng.

- [x] T001 Xác nhận đang ở nhánh `027-grant-repair-unblock` và `.specify/feature.json` trỏ tới `specs/027-grant-repair-unblock`
- [x] T002 Ghi lại kết quả `go test -race -count=3` của gói `grant` **trước** khi sửa, làm mốc đối chiếu (`cd services/ai; go test ./internal/modules/aiexecutor/application/usecase/grant/ -race -count=3`)

---

## Phase 2: Foundational (chặn mọi user story)

**Mục đích**: Hai việc mà US1, US2, US3 đều đứng trên.

**Vì sao T003 chặn**: Sau khi thêm `resolveGrantWindow`, sẽ tồn tại hai hàm cùng tính cửa
sổ. `plan.md` Mục 1 nêu đây là Rủi ro 2: `ComputeGrantWindow` và `resolveGrantWindow` trong
cùng một tệp, khác nhau ở chỗ có đọc cơ sở dữ liệu hay không, thì không có cách nào biết
đang dùng loại nào. Đổi tên **trước**, khi chỉ có một hàm — lúc đó không có gì để nhầm.

- [x] T003 Đổi tên `ComputeGrantWindow` thành `EventGrantWindow` trong `GRANT/grant_window.go:29` và cập nhật mọi nơi gọi (`GRANT/grant_uc.go`, `GRANT/grant_window_test.go`, `GRANT/grant_window_renewal_test.go`); chú thích phải nói rõ hàm này **không** đọc cơ sở dữ liệu và chỉ phản ánh ý sự kiện
- [x] T004 Thêm helper dựng bản ghi mức quyền có ảnh chụp chính sách **không đọc được** (`envelope` thiếu) vào tệp kiểm thử dùng chung `GRANT/grant_testhelpers_test.go`, vì US1, US2 và US3 đều cần kịch bản này và không nên viết lại ba lần

**Checkpoint**: Nền tảng xong — bắt đầu user story được.

---

## Phase 3: User Story 1 — Khách hàng đổi gói vẫn được đổi quyền AI (P1) MVP

**Mục đích**: Sự kiện đến muộn được áp dụng thay vì bị từ chối vĩnh viễn, và mốc bắt đầu
của bản ghi mới luôn lớn hơn nghiêm mọi bản ghi trước.

**Kiểm thử độc lập**: Chuẩn bị một tài khoản có bản ghi mức quyền hiện hành thiếu ảnh chụp
chính sách, phát một sự kiện đổi gói hợp lệ, xác nhận quyền AI thay đổi theo gói mới — mà
không cần thao tác sửa dữ liệu nào.

### Kiểm thử cho User Story 1

> Viết trước, bảo đảm **ĐỎ** trước khi triển khai.

- [x] T005 [P] [US1] Unit test cho `resolveGrantWindow` trong `GRANT/grant_window_resolution_test.go`, đặt tên hàm `TestResolveGrantWindow`: dùng lại `fakeGrantRepo` đã có sẵn tại `GRANT/grant_uc_test.go:30` (nó đã implement `GetNewestGrantForOrdering`) chứ **không** dựng fake thứ hai trong cùng gói. Kiểm: dịch 1 µs **quanh sàn mới** khi sự kiện đến muộn, **không** dịch khi sự kiện mới hơn (FR-007), lùi về `effective_from` cho sàn cũ khi `created_at` ở tương lai hoặc bằng 0 **và** phát dòng cảnh báo FR-018, từ chối khi `OccurredAt ≤ sàn cũ` (FR-002), từ chối khi dịch xong cửa sổ rỗng (FR-017)
- [x] T006 [P] [US1] Kiểm thử tích hợp trong `GRANT/grant_window_resolution_integration_test.go`, tên hàm `TestIntegrationLateEventShiftsPastFutureGrant`, `TestIntegrationStaleEventIsRefusedWithoutClosing`, `TestIntegrationConcurrentEventsLeaveOneActiveGrant`, `TestIntegrationFutureGrantIsTruncated`, `TestIntegrationLateDowngradeClosesPriorAtShiftedCutoff` (dùng `TEST_DATABASE_URL`, theo khuôn `grant_concurrency_integration_test.go:397`): AS1 sự kiện hợp lệ được áp dụng; AS2 sự kiện thực sự cũ bị từ chối và không bản ghi nào bị đóng; AS3 hai sự kiện gần như đồng thời còn đúng **một** bản ghi đang hiệu lực; bản ghi `tương lai` (`effective_from` 30 ngày sau `created_at`) bị cắt chứ không chồng lấn — ca bắt buộc của lỗi H2; **và** một sự kiện `ClosesPriorGrants` đến muộn (EffectiveAt < `effective_from` của bản ghi `active` mới nhất) đóng được bản ghi cũ tại `cutoff` đã dịch, không để lại hai bản ghi `active` — ca bắt buộc của khiếm khuyết `CloseActiveGrants`
- [x] T006b [P] [US1] Test ràng buộc âm của Hiến pháp trong `GRANT/grant_window_resolution_integration_test.go`: `resolveGrantWindow` **không** phát sinh truy vấn tới bảng nào ngoài `aiexecutor.ai_policy_grants` (AC-01), và mã dịch vụ AI không thêm bất kỳ cờ đặc quyền nào (AC-02) — hai điều kiện âm mà `deferred.md` D-05 mới chỉ nêu bằng lời

### Triển khai cho User Story 1

- [x] T007 [US1] Tạo `GRANT/grant_window_resolution.go` với `resolveGrantWindow(ctx, event) (GrantWindow, error)` theo thuật toán ở `contracts/grant-window-resolution.md` §2: gọi `EventGrantWindow`, đọc `GetNewestGrantForOrdering`, dựng **hai sàn** (`sàn cũ` từ `CreatedAt` với nhánh lùi `EffectiveFrom` kèm cảnh báo FR-018; `sàn mới = max(sàn cũ, prior.EffectiveFrom)`), từ chối vĩnh viễn khi `OccurredAt ≤ sàn cũ` hoặc dịch xong cửa sổ rỗng, nâng `EffectiveFrom` lên `sàn mới + 1 µs`
- [x] T008 [US1] Đổi chữ ký `retirePreviousGrants` tại `GRANT/grant_uc.go` để **nhận** `cutoff time.Time` thay vì tự gọi `EventGrantWindow(event).EffectiveFrom`; **cả hai** nhánh đóng/cắt (`CloseActiveGrants` và `TruncateGrantsBefore`) MUST dùng `cutoff`. Bản kế hoạch đầu tiên chỉ đổi nhánh `EXTENDED` và giữ `event.EffectiveAt` cho nhánh `CloseActiveGrants`; điều đó sai — một sự kiện đến muộn có `EffectiveAt` nhỏ hơn `effective_from` của bản ghi `active` mới nhất sẽ không đóng được nó, trong khi bản ghi mới đã bị dịch lên sau nó, tạo chồng lấn (phá FR-008). Xem `contracts/grant-window-resolution.md` §4. Hai đường không tạo bản ghi mới truyền `EventGrantWindow(event).EffectiveFrom`, tình cờ bằng `event.EffectiveAt`, nên hành vi của chúng không đổi
- [x] T009 [US1] Nối vào giao dịch tại `GRANT/grant_uc.go:237-273` theo đúng thứ tự ở `contracts/grant-window-resolution.md` §3: `LockGrantsForAccount` → kiểm tra trùng → `resolveGrantWindow` → gán `grant.EffectiveFrom` → `retirePreviousGrants(cutoff)` → `CreateGrant`. **Xoá hẳn** lời gọi `rejectStaleEvent` ở `GRANT/grant_uc.go:266` — giữ lại nó nghĩa là chạy kiểm tra thứ tự hai lần với hai tiêu chí khác nhau, và `ExtractOccurredAt` vẫn từ chối vĩnh viễn khi thất bại, tức là giữ nguyên đúng lỗi đang sửa
- [x] T010 [US1] Chuyển phần cảnh báo và bộ đếm của nhánh từ chối từ `rejectStaleEvent` sang `resolveGrantWindow`, giữ nguyên nội dung thông điệp và mã `stale_event` (FR-002, FR-003, FR-018), và bảo đảm nhánh từ chối **không ghi gì** (FR-012)
- [x] T010b [US1] Tách phần dựng **sàn cũ** thành hàm thuần dùng chung trong `GRANT/grant_window_resolution.go`, rồi cho `rejectStaleEvent` dùng hàm đó ở lời gọi còn lại tại `GRANT/grant_uc.go:497` (nhánh thu hồi vì hạ cấp tới gói không định giá được). Nhánh này **không** tạo bản ghi mới nên không cần dịch cửa sổ, nhưng nó dùng cùng hàm kiểm tra thứ tự và vẫn dừng vĩnh viễn khi ảnh chụp chính sách không đọc được — FR-019. Kiểm thử: tài khoản có grant hỏng + sự kiện hạ cấp tới gói không định giá được **không** bị từ chối vĩnh viễn
- [x] T010c [US1] Cập nhật hai nơi gọi còn lại của `retirePreviousGrants` — `GRANT/grant_uc.go:413` (đường thu hồi) và `GRANT/grant_uc.go:500` (hạ cấp không định giá được) — để truyền `EventGrantWindow(event).EffectiveFrom`, vì hai đường đó không có bước `resolveGrantWindow` để lấy mốc đã chốt

**Checkpoint**: US1 chạy và kiểm thử được độc lập.

---

## Phụ lục cho Phase 3 — Vì sao T008 và T009 không tách được

`buildGrant` dựng dòng ở `GRANT/grant_uc.go:207`, **ngoài** giao dịch, còn
`retirePreviousGrants` tự tính lại cửa sổ ở `GRANT/grant_uc.go:279`. Hai nơi cùng tính một
giá trị được coi là một. Nếu chỉ làm T009 mà bỏ T008, mốc cắt và mốc bắt đầu lệch nhau đúng
1 µs — đủ để tạo chồng lấn hoặc khe hở mà trong đó tài khoản bị từ chối bởi mọi lệnh gọi
AI. Chi tiết ở `research.md` R-002.

---

## Phase 4: User Story 2 — Quản trị viên sửa được tài khoản hỏng (P1)

**Mục đích**: Đường sửa trả về đúng kết quả thật, và kiểm tra toàn vẹn đo được đúng phạm vi.

**Kiểm thử độc lập**: Chuẩn bị một tài khoản có bản ghi mức quyền hiện hành bắt đầu đúng
vào ngày kỳ cước bắt đầu, chạy đường sửa, xác nhận bản ghi được dựng lại với ảnh chụp chính
sách đọc được.

**Phụ thuộc US1**: Story này là cùng một đường với US1. Spec nói rõ "Nếu US1 chưa được sửa
thì User Story này vô nghĩa".

### Kiểm thử cho User Story 2

- [ ] T011 [P] [US2] Kiểm thử tích hợp trong `GRANT/grant_repair_integration_test.go`, tên hàm `TestIntegrationRepairClosesOldAndCreatesReadable`, `TestIntegrationRepairDoesNotShiftNormalWindow`, `TestIntegrationRepairStartsAfterAllPrior`: AS1 bản ghi cũ được đóng và bản ghi mới có ảnh chụp đọc được, **không** phát sinh lỗi ràng buộc; AS2 sự kiện mới hơn thì mốc hiệu lực **không** bị dịch; AS3 bản ghi mới bắt đầu **sau** mọi bản ghi trước và không có hai cửa sổ đang hiệu lực chồng lấn

### Triển khai cho User Story 2

Ba task SQL cùng sửa **một tệp**, nên cố ý không đánh dấu `[P]`:

- [ ] T012 [US2] **Thêm** điều kiện lọc theo cửa sổ hiệu lực (`effective_from <= now() AND (effective_to IS NULL OR effective_to > now())`) vào mục 4 trong `database/scripts/07-verify_policy_grants.sql`, và **giữ nguyên** việc không lọc theo trạng thái — mục 4 hiện tại chưa có điều kiện `status` nào để xoá. KHÔNG thêm bộ lọc trạng thái "để cho khớp mô tả": đó chính là điều FR-014/FR-015 cấm, và nó bỏ sót đúng trường hợp bản ghi `đã đóng` nhưng cửa sổ còn hiệu lực — trường hợp bộ đọc vẫn dùng
- [ ] T013 [US2] Thêm truy vấn **thông tin** đếm bản ghi lịch sử có ảnh chụp không đọc được vào `database/scripts/07-verify_policy_grants.sql`, không tính vào kết quả kiểm tra (FR-016)
- [ ] T014 [US2] Cập nhật phần mở đầu trong `database/scripts/07-verify_policy_grants.sql`: **giữ nguyên** tiêu đề "six invariants" vì mục 4 vẫn là một trong sáu mục, nhưng sửa **mô tả phạm vi của mục 4** và các dòng `\echo` của nó cho khớp với việc đã lọc theo cửa sổ hiệu lực; đồng thời sửa dòng ở mục 1 nói `restore` là cách sửa, vì sau sự cố 027 nó là một trong **hai** nguyên nhân
- [ ] T015 [US2] Cập nhật `specs/027-grant-repair-unblock/runbook.md` cho khớp với hành vi đã triển khai, và bổ sung cổng chặn dựa trên bộ đếm `grant_rejected_by_reason` nếu cần

**Checkpoint**: US2 chạy và kiểm thử được độc lập.

---

## Phase 5: User Story 3 — Lỗi tất định được báo đúng một lần (P2)

**Mục đích**: Vi phạm ràng buộc dữ liệu bị báo ngay, đếm một lần, không thử lại — và sự cố
tạm thời thật sự vẫn được thử lại như trước.

**Kiểm thử độc lập**: Gây một vi phạm ràng buộc dữ liệu có kiểm soát, xác nhận nó được báo
đúng một lần, không thử lại, đi thẳng vào hàng đỗi chết.

### Kiểm thử cho User Story 3

> **Rủi ro chưa đóng của cả đợt này nằm ở T016.** Đọc nó trước.

- [ ] T016 [US3] Kiểm thử tích hợp trên Postgres thật trong `GRANT/grant_error_class_integration_test.go`, tên các hàm: `TestIsCheckViolationThroughGorm`, `TestUniqueViolationStaysRetryable`, `TestDeadlockStaysRetryable`: **tự thêm** ràng buộc `CHECK (effective_to IS NULL OR effective_to > effective_from)` bằng `ALTER TABLE` — thực thể `AIPolicyGrant` không có thẻ `check:` nên `AutoMigrate` không tạo — rồi ép `23514` thật, ép `23505` thật (chèn hai dòng trùng khoá) và ép deadlock thật (hai transaction lấy `pg_advisory_xact_lock` ngược thứ tự), khẳng định `errors.As` bóc được `*pgconn.PgError` **xuyên qua GORM**. Ba ca này **không** được thay bằng `PgError` dựng tay — nó sẽ đỏ ngay cả khi tích hợp hỏng. Nếu bước này đỏ thì toàn bộ US3 dựng trên giả định chưa kiểm chứng: dừng và báo lại, đừng sửa tiếp
- [ ] T017 [P] [US3] Unit test trong `GRANT/grant_error_class_test.go`, tên hàm `TestIsCheckViolation`: `isCheckViolation(nil)` trả `false`, lỗi bọc nhiều tầng vẫn bóc được, `23505` và `40001` trả `false` (FR-010)
- [ ] T018 [P] [US3] Bổ sung `GRANT/grant_reject_metrics_test.go`, tên hàm `TestConstraintViolationReasonRecorded` và `TestConstraintViolationIsNotRequeued`: mã lý do mới xuất hiện trong `grant_rejected_by_reason`, **đúng một** dòng cảnh báo mỗi lần, dòng đó mang `user_id`, **không** có ngưỡng cảnh báo (FR-011); lỗi trả về là `sharedevent.PermanentError` chứ không phải `RetryableError` và `startConsumerLoop` **không** requeue nó, trong khi `transient_failure` vẫn requeue (SC-004); mọi đường từ chối đều ghi đủ `user_id` + mã lý do + nguyên nhân (SC-006)

### Triển khai cho User Story 3

- [ ] T019 [US3] Tạo `GRANT/grant_error_class.go` với hằng `pgCheckViolationCode = "23514"` và hàm `isCheckViolation(err) bool` dùng `errors.As` với `*pgconn.PgError`; chú thích phải nêu rõ vì sao **chỉ** bắt `23514` chứ không bắt cả lớp `23` (xem `research.md` R-001)
- [ ] T020 [US3] Chạy `cd services/ai; go mod tidy` rồi xác nhận `github.com/jackc/pgx/v5` chuyển từ `// indirect` sang phụ thuộc trực tiếp trong `services/ai/go.mod`
- [ ] T021 [US3] Khai mã lý do `policy_grant_constraint_violation` cạnh năm mã đang có trong `GRANT/grant_uc.go:43-52`
- [ ] T022 [US3] Thêm nhánh phân loại vào khối xử lý lỗi giao dịch tại `GRANT/grant_uc.go:274-284`, đứng **trước** nhánh `transient`; nhánh `*RejectionError` và đường `transient` cũ giữ nguyên
- [ ] T023 [US3] Xác nhận lại trên mã sau thay đổi rằng `GrantMetrics.Snapshot()` vẫn trả `map[string]int64` và `grant_rejected_by_reason` vẫn được lắp trong `internal/bootstrap/router.go`, để **đóng CHK026 bằng bằng chứng**; bằng chứng đã có sẵn ở `quickstart.md` K4 nên việc này chỉ là xác nhận không hồi quy. Nếu phát hiện `Snapshot()` đã đổi sang kiểu đóng thì dừng, mở lại CHK026 và tạo `specs/027-grant-repair-unblock/frontend-guide.md`

**Checkpoint**: Cả ba user story chạy và kiểm thử được độc lập.

---

## Phase 6: Polish & Cross-Cutting Concerns

- [ ] T024 [P] Trong `specs/024-admin-premium-grant/contracts/subscription-plan-changed-event.md`: sửa dòng mô tả sai phạm vi `stale_event` (CHK028), **thêm** một hàng cho `policy_grant_constraint_violation` vào bảng mã lý do tối thiểu vì FR-011 bắt buộc nó xuất hiện trong bộ đếm công khai, và sửa luôn hàng `plan_not_in_catalog` đã bị 024 T159 đổi thành `ai_cost_policy_not_found`
- [ ] T025 Tick mục **Frontend Integration Guide**: 027 không đổi/thêm/xoá HTTP endpoint, không đổi event payload, không đổi DTO client-facing — chỉ thêm một khoá trong đối tượng sức khoẻ nội bộ, và T023 đã xác nhận khoá đó là `string` tự do nên không phá client. **Không** tick nếu T023 phát hiện `Snapshot()` đã đổi sang kiểu đóng
- [ ] T026 [P] Tick mục **Phát hiện ngoài phạm vi**: `specs/027-grant-repair-unblock/deferred.md` đã ghi 6 mục (D-01..D-06) với **vấn đề**, **vì sao ngoài phạm vi**, **gỡ bằng cách nào**; D-03 và D-06 ghi rõ chúng làm suy yếu FR-014/FR-015 và FR-009, và nêu đích danh task bị ảnh hưởng
- [ ] T027 Chạy `make lint` và `go test -race -count=3` trong `services/ai`; đối chiếu với mốc T002
- [ ] T028 Chạy `make check` đầy đủ — bắt buộc trước khi push hoặc tag
- [ ] T029 Cập nhật `specs/027-grant-repair-unblock/quickstart.md` nếu số đếm K6 lệch so với kết quả thật sau khi chạy
- [ ] T030 Đọc lại `specs/027-grant-repair-unblock/quickstart.md`, `runbook.md` và cả hai tệp trong `contracts/`, đối chiếu **từng mệnh đề** với mã nguồn, và **ghi lại số đã đối chiếu** (bao nhiêu hàm, bao nhiêu cột, bao nhiêu mã lý do, bao nhiêu lệnh kiểm chứng đã chạy thật). Một lượt đọc lại không sinh ra con số thì lượt đó chưa xảy ra
- [ ] T031 Xác minh FR-013: **không** có câu lệnh SQL nào ghi vào `aiexecutor.ai_policy_grants` trong đợt này — đường sửa là `POST /api/v1/admin/subscription-grants` sinh sự kiện; và trong mã nguồn, **không** có thành phần nào ngoài dịch vụ AI ghi bảng này. Ghi lại bằng chứng (đường dẫn tệp và truy vấn `grep`/`Select-String`) vào báo cáo cuối, vì đây là yêu cầu dạng **phủ định** — không có gì để chạy, nên bằng chứng phải là điều đã chứng minh là không tồn tại
- [ ] T032 Xác nhận AC-03: tập loại sự kiện, hợp đồng sự kiện và nội dung ảnh chụp chính sách **không** đổi. Đối chiếu `git diff` chỉ chạm các tệp trong danh sách của 027, và nêu rõ trong báo cáo nếu có tệp hợp đồng nào bị sửa (T024 sửa mô tả trong hợp đồng 024, và đó là đính chính mô tả, không phải đổi cấu trúc)

---

## Dependencies & Execution Order

### Phụ thuộc giữa các phase

- **Phase 1 (Setup)**: không phụ thuộc gì
- **Phase 2 (Foundational)**: phụ thuộc Phase 1 — **chặn mọi user story**
- **Phase 3 (US1)**: phụ thuộc Phase 2
- **Phase 4 (US2)**: phụ thuộc Phase 2 **và US1** — cùng một đường mã, không tách rời được
- **Phase 5 (US3)**: phụ thuộc Phase 2; **không** phụ thuộc US1/US2 về mặt nghiệp vụ
- **Phase 6 (Polish)**: phụ thuộc các story đã chọn triển khai

### Phụ thuộc giữa các user story

- **US1 (P1)**: bắt đầu sau Phase 2. Không phụ thuộc story nào
- **US2 (P1)**: bắt đầu sau US1. Đặc tả nói thẳng "nếu US1 chưa được sửa thì User Story này
  vô nghĩa"
- **US3 (P2)**: bắt đầu sau Phase 2. Độc lập về nghiệp vụ

### Xung đột tệp — cảnh báo thật, không phải hình thức

US1 và US3 cùng sửa `GRANT/grant_uc.go`: US1 ở khối giao dịch (T009), US3 ở khối xử lý
lỗi (T022) và khối hằng (T021). Hai vùng đó cách nhau, nhưng **cùng một tệp**, nên:

> **US1 và US3 KHÔNG chạy song song.** Chạy tuần tự, hoặc ghép lại rồi tách tệp nếu muốn
> song song thật.

### Trong mỗi user story

- Kiểm thử viết trước và **ĐỎ** trước khi triển khai
- `EventGrantWindow` trước `resolveGrantWindow` (T003 trước T007)
- `resolveGrantWindow` trước khi nối vào giao dịch (T007 trước T009)
- Đổi chữ ký `retirePreviousGrants` trước khi nối (T008 trước T009)
- US3 thứ tự bắt buộc: **T019 → T020 → T016**. T016 import `pgconn` và kiểm thử đường đi
  thật qua GORM, nên `go mod tidy` phải chạy trước; nếu kiểm thử chạy trước T020 thì nó có
  thể đỏ vì phụ thuộc chưa được khai báo, hoặc xanh nhờ phụ thuộc gián tiếp có sẵn — cả hai
  đều không chứng minh được điều gì
- US1 thứ tự bắt buộc: T005/T006/T006b có thể viết song song, nhưng T009 chỉ tick khi T007
  và T008 đã xong

### Cơ hội chạy song song

- Mọi task `[P]` trong Phase 1 và Phase 2
- T005, T006, T006b song song (đều là tệp kiểm thử mới); T011 song song với US1
- T016, T017, T018 song song với nhau, nhưng **cả ba** đứng sau T020
- T024, T026, T029, T031, T032 song song
- **Không** song song: T012→T013→T014 (cùng một tệp SQL), T008→T009 (cùng tệp, có phụ thuộc),
  T023→T025 (phụ thuộc bằng nội dung), US1 với US3 (cùng tệp `grant_uc.go`)

---

## Parallel Example: User Story 1

```text
Task: "Unit test cho resolveGrantWindow trong GRANT/grant_window_resolution_test.go"
Task: "Kiểm thử tích hợp trong GRANT/grant_window_resolution_integration_test.go"
```

## Parallel Example: User Story 3

```text
Task: "Kiểm thử tích hợp trên Postgres thật trong GRANT/grant_error_class_integration_test.go"
Task: "Unit test trong GRANT/grant_error_class_test.go"
Task: "Bổ sung GRANT/grant_reject_metrics_test.go"
```

---

## Chiến lược triển khai

### MVP — chỉ User Story 1

1. Phase 1: Setup (T001, T002)
2. Phase 2: Foundational (T003, T004)
3. Phase 3: US1 (T005..T010)
4. **DỪNG và kiểm chứng**: kiểm thử US1 độc lập
5. Chưa được sửa dữ liệu production ở bước này

### Giao hàng tăng dần

1. Setup + Foundational → nền tảng xong
2. US1 → kiểm thử độc lập → **MVP**
3. US2 → kiểm thử độc lập → đường vận hành dùng được
4. US3 → kiểm thử độc lập → quan sát tốt hơn
5. Triển khai, **rồi mới** chạy `runbook.md`

> Chỉ chạy `runbook.md` sau khi US1 **và** US2 đã triển khai. Đường sửa trong runbook dựa
> vào chính hành vi tính lại mà US1 viết; chạy sớm hơn là sửa dữ liệu bằng bản vá chưa có.

### Chỉ khi có nhiều người

- Sau Phase 2: một người làm US1; US3 có thể chuẩn bị **kiểm thử** song song (T016..T018 là
  tệp mới), nhưng phần triển khai US3 phải chờ US1 xong vì cùng sửa `grant_uc.go`
- US2 chờ US1

---

## Ghi chú

- [P] nghĩa là khác tệp và không phụ thuộc task chưa xong — **không** phải "có thể làm
  song song về mặt ý tưởng"
- Nhãn [Story] để truy vết; mỗi story phải hoàn thành và kiểm thử được **độc lập**
- Kiểm thử phải đỏ trước khi triển khai; test với `PgError` dựng tay **không** thay được
  T016 — nó sẽ đỏ ngay cả khi tích hợp hỏng
- **Không** sửa dữ liệu bằng SQL (FR-013), **không** dùng lại `POST
  /api/v1/admin/subscription-grants/restore` (khoá sự kiện đã ghi vĩnh viễn), **không**
  đẩy bất kỳ commit nào khi chưa được yêu cầu
- Không đánh dấu `[X]` một task nếu chưa chạy lệnh xác minh tương ứng