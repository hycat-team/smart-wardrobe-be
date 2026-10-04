# Deferred — 026 Sổ Đăng Ký Chiến Dịch Trong Cơ Sở Dữ Liệu

Phát hiện trong lúc chốt feature 026. **Không mục nào là việc của 026**: hoặc thuộc hạ tầng
kiểm thử dùng chung, hoặc thuộc phạm vi mà lần chạy này không được phép sửa.

Không mục nào chặn việc đóng 026. Chúng được ghi ở đây chứ không để lại thành ô
`- [ ]` trong `tasks.md`, vì một ô chưa tick khẳng định *"feature này chưa xong"* — điều vừa là
một khẳng định khác vừa là sai.

- **D-01 làm suy yếu bằng chứng của 10 task đã tick.** Đọc D-01 trước khi tin bất kỳ task nào
  trong danh sách đó.
- **D-02 đã gỡ** — hợp đồng đã được sửa trong lần chống, xem mục D-02 bên dưới.
- **D-03 là flake hiếm** khi `make test-integration` chạy đồng thời hai gói trên cùng một CSDL.

---

## D-01 — Không có hạ tầng nào chạy được quickstart V1–V7 của 026 trong `go test` thường

- **Vấn đề**: toàn bộ bằng chứng 026 cần CSDL thật đều nằm trong các tệp kiểm thử `t.Skip` khi
  `TEST_DATABASE_URL` không được đặt, và `campaign_seed_integration_test.go` còn cần thêm
  `TEST_DATABASE_ALLOW_REGISTRY_ROLLBACK=true` (rollback xoá bảng nên phải được bật tường minh):

  | Tệp kiểm thử | Phủ claim |
  |---|---|
  | `infrastructure/persistence/campaign_repo_integration_test.go` | T022 (trigger If-Match, `FOR SHARE`, `EXISTS`) |
  | `infrastructure/persistence/campaign_schema_guard_integration_test.go` | T027 (không FK, `campaign_claims` không đổi) |
  | `infrastructure/persistence/campaign_guard_integration_test.go` | T057 (trigger bất biến chặn SQL thô) |
  | `infrastructure/persistence/campaign_seed_integration_test.go` | T059 (seed idempotent, rollback) |
  | `usecase/campaign/campaign_db_grant_integration_test.go` | T038 |
  | `usecase/campaign/campaign_db_close_integration_test.go` | T049 |
  | `usecase/campaign/campaign_db_close_granted_integration_test.go` | T047 |
  | `usecase/campaign/campaign_db_time_integration_test.go` | T063 |
  | `usecase/campaign/campaign_db_audit_read_integration_test.go` | T072 |

  Ngoài CSDL, các kịch bản `quickstart.md` còn cần hai thứ mà không kho kiểm thử nào cung cấp:
  **đăng ký tài khoản thật** (để có `identity.users` và một lượt đăng ký đúng mốc thời gian) và
  **chờ consumer outbox chạy xong** trước khi đọc `GET .../audit` (FR-060 ghi nhật ký bất đồng
  bộ). Không có bước dựng nào làm ra một tài khoản qua cả ba cửa tạo tài khoản, nên V1–V7 chỉ
  chạy được bằng tay với một môi trường thật.

- **⚠️ Vòng hội tụ phát hiện D-01 đã under-report — và đã sửa, nhưng phần "chưa bật được" vẫn còn.**
  Bản D-01 đầu tiên chỉ nói các kiểm thử "chưa từng chạy". Vòng hội tụ phát hiện điều tệ hơn: các
  kiểm thử đó **nếu chạy sẽ trượt vì một lý do sai**, chứ không phải trượt vì thiếu hạ tầng.
  Cụ thể `baseInstant()` trả về **đúng** mốc `starts_at = 2026-10-02T00:00:00+07:00` của hàng
  seed `launch-2026-10`, còn mọi fixture tạo chiến dịch ở `baseInstant()-24h` — tức **sớm hơn**.
  FR-040 chọn hàng có `starts_at` **lớn nhất**, nên hàng seed sẽ thắng và các khẳng định ở
  `campaign_db_grant_integration_test.go`, `campaign_db_close_integration_test.go`,
  `campaign_db_close_granted_integration_test.go` và `campaign_db_time_integration_test.go`
  sẽ kiểm tra **sai chiến dịch** mà vẫn xanh.

  **[X] T081 đã sửa phần này**: `campaign_db_registry_isolation_test.go` xoá hàng seed trước mỗi
  fixture và **khôi phục nguyên vẹn** ở `t.Cleanup`, đồng thời có tiền điều kiện
  `requireRegistryIsolatedTo` gọi `ListAll` và `t.Fatalf` nếu thấy mã lạ hoặc thiếu mã của tệp;
  thông báo lỗi in **tên tệp** lấy từ `runtime.Caller`. Dùng `ListAll` chứ không `ListOpen`
  vì đó là câu hỏi chặt hơn — một chiến dịch **đã đóng** sót lại vẫn lọt qua `ListOpen`.

  Nghĩa là D-01 **giảm** còn một rủi ro: tiền điều kiện cô lập của chính T081 cũng chỉ chạy *sau
  khi* kết nối CSDL thành công, nên nó cũng chưa từng được thực thi. Bằng chứng cho T081 hiện là
  **đối chiếu tĩnh** (migration + `baseInstant()` + `startsLaterThan` + opener của fixture), không
  phải kết quả chạy.

- **⚠️ Vòng hội tụ lần 2 tìm ra một lỗi thứ hai trong D-01 — và đã sửa.** Phase 11 phát hiện
  `migrationSection(t, name, "Up")` truyền **marker đóng rỗng** cho `betweenMarkers`, nên phần
  `Up` chạy tới **cuối tệp** và nuốt luôn thân phần `Down`. Với migration seed, thân `Down` chính là
  `DELETE FROM subscription.campaigns WHERE code = 'launch-2026-10'`, và
  `TestIntegrationSeedInsertIsIdempotent` **thực thi** chuỗi SQL mà nó nhận được. Nên khi có CSDL,
  kiểm thử đó sẽ chèn hàng, xoá ngay hàng đó bằng chính phần `Down` lẫn vào, đếm được **0** rồi
  **FAIL** — một kiểm thử *không thể xanh trên bất kỳ CSDL nào*, do một helper trông đúng.

  Đây là lớp hỏng giống hệt phần trên: không phải "chưa bật được" mà là "bật lên sẽ hỏng". Nó
  không lọt vào D-01 ở lần ghi đầu vì lúc đó chưa ai đọc kỹ helper.

  **Đã gỡ**: `migrationSection` nay đóng phần `Up` tại `-- +goose Down`. `Down` là phần cuối
  nên để marker đóng rỗng ở đó vẫn vô hại. Phần `Down` giữ nguyên, nên
  `TestIntegrationRegistryDownSectionsDoNotTouchTheClaimLedger` và
  `TestIntegrationSeedDownIsScopedToTheSeededRow` không đổi hành vi. Khoá lại bằng
  `TestTheUpSectionStopsAtTheDownMarker` — kiểm thử **tĩnh**, chạy được không cần CSDL, nên không
  thể trượt cùng lý do với D-01. Nó còn khẳng định phần `Up` **vẫn còn** `INSERT`, để phần kiểm
  tra `DELETE` không trở nên rỗng.

  Đã kiểm chứng bằng phản chứng: đặt lại marker đóng rỗng thì kiểm thử **FAIL** đúng thông báo;
  hoàn nguyên thì xanh.

- **Vì sao ngoài phạm vi 026**: đây là **hạ tầng kiểm thử dùng chung**, không phải logic chiến
  dịch. 026 không sửa entity, không sửa migration, không sửa mã harness nào; nó chỉ là feature
  đầu tiên phải dựng lại đường cấp gói trên CSDL thật. Sửa `openWebhookTestDB` hay viết một
  `openCampaignTestDB` là việc của feature hạ tầng hoặc của feature sau, và đặt nó ở đây sẽ
  biến một khoản nợ của hạ tầng thành một ô chưa tick của 026.

  Lưu ý đây là **cùng họ** với D-01 và D-02 của `specs/025-signup-campaign-grant/deferred.md`:
  cả hai feature đầu tiên chạm CSDL thật đều phát hiện hạ tầng chưa sẵn sàng. Ba phát hiện,
  cùng một nguyên nhân gốc.

- **Gỡ bằng cách nào**: một `make test-integration` hoặc một profile CI có sẵn Postgres, chạy
  `make migrate-up svc=account` trước rồi `go test` với `TEST_DATABASE_URL` và
  `TEST_DATABASE_ALLOW_REGISTRY_ROLLBACK=true`. Riêng quickstart thì cần thêm một bước dựng tài
  khoản dùng chung — nên nằm ở `shared/tools`, không ở gói campaign. Sau khi có, các tệp trên
  chạy **không đổi một dòng**, vì chúng đã được viết đúng từ đầu; chỉ là không ai bật chúng.

- **Ảnh hưởng tới task đã tick**: **`[X] T022`, `[X] T027`, `[X] T038`, `[X] T047`, `[X] T049`,
  `[X] T057`, `[X] T059`, `[X] T063`, `[X] T072`, `[X] T078`, và `[X] T081`.**

  Tất cả mười task đầu đều đã tick, và tất cả đều **có mã** nhưng **chưa từng chạy** trong một lần
  `go test` thường — chúng đã báo `SKIP`. Đọc `[X]` của chúng là đọc "đã viết và biên dịch đúng",
  không phải "đã chạy và xanh". Điều chưa từng được kiểm chứng bằng CSDL thật gồm: hành vi thực
  của trigger `trg_campaigns_immutable_budget`, phép `SELECT … FOR SHARE` của FR-045, phép
  `UPDATE … WHERE version = ?` trả 0 dòng, idempotent của seed, việc rollback giữ nguyên
  `campaign_claims`, và việc nhật ký thật sự xuất hiện sau khi consumer chạy.

  Riêng `[X] T078` còn yếu hơn nữa: nó yêu cầu chạy V5, V6 và nhánh `plan_unavailable` **trên
  CSDL kiểm thử**, và điều đó đã không thực hiện được. Phần T078 *đã* làm là bổ sung các kiểm thử
  đơn vị chứng minh đúng nội dung của V5/V6 bằng fake (`campaign_read_contract_shape_test.go`,
  `campaign_sweep_reads_registry_each_run_test.go`) — đó là bằng chứng thay thế, không phải bằng
  chứng gốc.

---

## D-02 — ĐÃ GỎ: đường dẫn tệp cũ của `CampaignAuditLogRes` trong hợp đồng

- **Vấn đề (đã ghi trước đây)**: `contracts/campaign-registry-api.md` §6.3 ghi
  `CampaignAuditLogRes` nằm ở `campaign_dto.go`, trong khi nó đã được tách sang
  `campaign_audit_dto.go` ở bước chốt (để `campaign_dto.go` từ 578 dòng về dưới ngưỡng cảnh báo
  500 dòng của `code-hygiene.md` §H1). Hợp đồng nói sai, `frontend-guide.md` mới viết dựa trên
  đó — người đọc sẽ tìm nhầm tệp.

- **Đã gỡ thế nào**: orchestrator đã sửa §6.3 sang `campaign_audit_dto.go` và bổ sung `401`/`500`
  vào các bảng lỗi, ngay trước khi commit phase chốt. Mục này được giữ lại thay vì xoá để giữ
  dấu vết: nếu `frontend-guide.md` và hợp đồng lệch nhau trong tương lai thì đây là chỗ đã từng
  sai.

- **Phần T069 vẫn đúng**: `TargetID *uuid.UUID` khai `json:"targetId"` **không** `omitempty`, DTO
  riêng chứ không tái dùng `dto/admin.AdminAuditLogDTO`, và mapper `MapCampaignAuditLogRes`
  vẫn ở `application/mapper/campaign_mapper.go`.

---

## D-03 — Hai gói kiểm thử CSDL chạy song song trên cùng một `TEST_DATABASE_URL`

- **Vấn đề**: `make test-integration` chạy `go test ./internal/modules/subscription/...`, tức
  **binary gói `usecase/campaign` và binary gói `infrastructure/persistence` chạy song song** trên
  cùng một cơ sở dữ liệu. Trong lúc các fixture của T081 đang xoá hàng seed `launch-2026-10`,
  `campaign_seed_integration_test.go` ở gói kia có thể đọc sổ và **không thấy hàng đó**, rồi
  `t.Fatal` với thông báo sai.

- **Mức độ**: **flake hiếm**, không phải bộ kiểm thử hỏng. Cửa sổ bằng thời lượng một fixture và
  cần hai kiểm thử trùng nhau. Không xảy ra khi chạy tuần tự (`-p 1`).

- **Vì sao ngoài phạm vi 026**: cách đóng cửa sổ đúng là một khoá `pg_advisory_lock` ở mức
  **session** mà **cả hai** gói cùng nắm. Nhưng khoá session-level phải giữ trên một
  `*sql.Conn` đã ghim, còn các opener hiện trả connection từ pool — giữ khoá trên connection đó
  sẽ **treo cả bộ kiểm thử CSDL về sau**. Sửa thêm opener của gói `persistence` cũng vượt ra
  ngoài bốn task hội tụ. Ghi vào đây thay vì sửa nửa vời.

- **Gỡ bằng cách nào**: một helper CSDL dùng chung ở `shared/tools` cấp
  `pg_advisory_lock` theo session trên một `*sql.Conn` ghim, cùng một API cho cả hai gói. Đây là
  việc của feature hạ tầng kiểm thử. Tạm thời có thể chạy `go test -p 1` khi cần kết quả sạch.

---

## D-04 — Hiệu chỉnh V2: mã `campaign_sleeping` không quan sát được trên đường thông điệp

- **Vấn đề (đã gỡ ở tài liệu, KHÔNG đổi mã)**: `quickstart.md` V2 và `spec.md` §11.2–11.3 bản
  soạn đầu yêu cầu `reason = campaign_sleeping` cho một chiến dịch đã đóng. Với sổ **chỉ có**
  chiến dịch đã đóng, `ListOpen` lọc `closed_at IS NULL` nên tập ứng viên thức rỗng và **điều kiện 1**
  trả `campaign_not_configured` — trên **cả hai** đường (thông điệp và tra cứu, vì
  `campaign_query_claims_uc.go:193` cũng đọc `ListOpen`).

  Mã `campaign_sleeping` chỉ đạt được khi sổ còn **một** chiến dịch khác *chưa đóng* và *chưa
  thức* — thiết lập mà quickstart không dựng. Đây là hệ quả trực tiếp của quyết định đặt **cửa sổ ở
  bộ đánh giá thuần** chứ không ở SQL, **không phải hồi quy và không phải lỗi mã**. Nguồn của quyết
  định: `spec.md` Câu 6 (phần "Phân công"), `research.md` R9, và hợp đồng ghi bằng văn xuôi ở
  `campaign_repo.go:73`.

- **Đã gỡ thế nào**: orchestrator đã hiệu chỉnh `spec.md` §11.2–11.3 và `quickstart.md` V1–V2:
  phần đảm bảo được giữ nguyên là **0 dòng cấp**; mã quan sát được được nêu đúng là
  `campaign_not_configured`, kèm ghi chú trỏ tới nơi `campaign_sleeping` **vẫn** được chứng minh
  (`TestEvaluatorExcludesForceClosedCampaigns`, và đường tra cứu theo mã khi sổ còn chiến dịch
  khác). Tập 12 mã `reason` của 025 **không đổi**.

- **Không sửa mã, và không thêm kiểm thử khẳng định ngược lại.** Đổi `ListOpen` để giữ chiến
  dịch đã đóng sẽ phá lại chính quyết định mà `spec.md` Câu 6 và `research.md` R9 đã chốt (xem
  `campaign_repo.go:73`), nên đó sẽ là một quyết định thiết kế mới, không phải một sửa lỗi.