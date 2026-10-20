# Deferred — 026 Sổ Đăng Ký Chiến Dịch Trong Cơ Sở Dữ Liệu

Phát hiện trong lúc chốt feature 026. **Không mục nào là việc của 026**: hoặc thuộc hạ tầng
kiểm thử dùng chung, hoặc thuộc phạm vi mà lần chạy này không được phép sửa.

Không mục nào chặn việc đóng 026. Chúng được ghi ở đây chứ không để lại thành ô
`- [ ]` trong `tasks.md`, vì một ô chưa tick khẳng định *"feature này chưa xong"* — điều vừa là
một khẳng định khác vừa là sai.

- **D-01 làm suy yếu bằng chứng của 10 task đã tick.** Đọc D-01 trước khi tin bất kỳ task nào
  trong danh sách đó. **Cập nhật 04/10/2026: V1+V2 đã chạy thật — xem mục "ĐÃ CHẠY THẬT" bên dưới;
  D-01 vẫn mở nhưng nhỏ hơn nhiều, và nó đã tự báo thêm hai lỗi mà chỉ chạy mới lộ ra.**
- **D-02 đã gỡ** — hợp đồng đã được sửa trong lần chống, xem mục D-02 bên dưới.
- **D-03 là flake hiếm** khi `make test-integration` chạy đồng thời hai gói trên cùng một CSDL.
- **D-05 có liên quan hiến pháp** về hạn mức email của bên thứ ba, và là khoản nợ **của 025**, không
  phải của 026. Đọc D-05 **trước khi chạy bất kỳ kịch bản nhiều tài khoản nào ở chế độ production**.

---

## ĐÃ CHẠY THẬT — V1 + V2 trên dev stack, 04/10/2026

Không phải mục riêng của D-01, mà là **bằng chứng gỡ được một phần D-01**. Ghi ở đây vì nó thay đổi
mức độ tin cậy của các ô `[X]` bên dưới.

- **Môi trường**: dev stack sẵn có — Postgres `smart_wardrobe_db` (user `admin`) chứa sẵn **55 claim**
  phản ánh dữ liệu thật; account service phải **rebuild** (`docker compose … up -d --build account`)
  vì image cũ không bind-mount source. Bằng chứng binary đã đúng: `GET /api/v1/admin/campaigns` trả
  **401** thay vì **404**.
- **Migration được container tự áp lúc boot** (`DATABASE_MIGRATE_ON_START=true`, `/migrations` bake
  trong image) nên **không** cần `make migrate-up svc=account`. Sau đó `max(version_id) = 20261003165500`.
- **Email: 0 email rời khỏi máy.** Theo hiến pháp §"Outbound email volume during verification", kịch bản
  cần nhiều tài khoản thì **phải** tắt đường email: đặt `EMAIL_ENABLED=false` trong `.env` (đã revert
  sau khi chạy, không commit). `SmtpSender.Send` trả `nil` trước khi mở socket, và email worker ack
  rồi bỏ message — nên không dựng phiên SMTP nào. OTP **vẫn** sinh và nằm trong Redis vì
  `redis_otp_service.go` không đọc cờ này.

### Kết quả

| Bước | Kết quả |
|---|---|
| Tạo `verify-026` (quota 2, `startsAt 2026-10-03`) | `version 1`, `status running`, `remainingMain 2` |
| Đăng ký acc#1 → `confirm-otp` | ✅ **ĐƯỢC CẤP** — claim `verify-026` / `premium-monthly` / `grant_bucket = main`; `user_subscriptions.current_plan_code` chuyển `free` → **`premium-monthly`** (tier 1, hết hạn 03/11) |
| Tra cứu campaign sau grant | `grantedMainCount = 1`, `remainingMain = 1`, **`budgetLocked = true`** (đóng băng ngân sách vì đã có claim) |
| `eligibility(acc#1)` | `granted`, `granted = true` |
| `close` (`version 1`) | `closedAt` có giá trị, `version` → **2**, `status = closed` |
| Đăng ký acc#2 | ✅ **0 claim** ở mọi chiến dịch; `current_plan_code` vẫn `free` / tier 0 |

### Hai lỗi mà CHỈ chạy thật mới lộ ra

1. **`quickstart.md` V1 dùng `startsAt = 2020-01-01` — không bao giờ thắng.** FR-040 chọn `starts_at`
   **lớn nhất**; seed `launch-2026-10` có `2026-10-02`. `verify-026` với `2020` sẽ không bao giờ được
   chọn, và acc#1 sẽ bị đánh giá với hàng seed. **Đã sửa** thành `2026-10-03` kèm cảnh báo. Cùng loại
   lỗi với cái T081 đã sửa trong test — nhưng nằm trong quy trình kiểm chứng, nên không ai thấy.
2. **Bước 4 của V1 xanh vì lý do sai.** Khi `verify-026` đóng, nó bị `ListOpen` loại và đường cấp gói
   **rơi xuống** `launch-2026-10`. Ở DB này chiến dịch đó **đã hết quota** (`quota_main`), nên bước 4 xanh
   **không phải nhờ `close`**. Điều kiện đã sửa thành **campaign-scoped**: "0 claim từ `verify-026`".
3. **Đường tra cứu theo mã không dùng chung `evaluate()`** — nó trả `outside_window` cho chiến dịch đã
   đóng, không phải `campaign_sleeping`. Sửa ở D-04, `spec.md` §11.3 và `quickstart.md` V2.

### Điều kiện tiên quyết mà D-01 vẫn chưa đóng

Một chiến dịch không bao giờ bị xoá được (registry không có `Delete`, đúng thiết kế), nên sau lần chạy
này DB dev **vĩnh viễn** có: `verify-026` ở trạng thái `closed`, và hai tài khoản
`verify026acct1@example.com` / `verify026acct2@example.com` (acc#1 còn giữ `premium-monthly` từ
`verify-026`). Không dọn được bằng API.

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

- **TRẠNG THÁI (2026-10-20, đợt CI-database — RESOLVED cho tập bằng chứng campaign)**:
  job `db-integration` trong `.github/workflows/ci.yml` dựng Postgres (`pgvector/pgvector:pg17`),
  nạp `database/init/00..03`, chạy `make migrate-up` cho account/commerce/ai rồi
  `make test-db-integration`. Bằng chứng trên CI (run `39368352014`, commit `40d79671`):
  - `account/database/migrations` — 1.074s
  - `account/.../subscription/infrastructure/persistence` — 2.048s
  - `account/.../subscription/application/usecase/campaign` — 2.279s
  - `account/.../subscription/application/usecase/webhook` — 1.287s
  - `ai/.../aiexecutor/application/usecase/grant` — 3.113s
  Cả 5 gói `ok`, **không có dòng `skipping`** nào trong log job: từ đây chúng chạy thật trên mỗi PR,
  không còn `SKIP`. Lần chạy đầu còn lộ ra ba lỗi tiềm ẩn đã sửa trong cùng đợt: schema-guard
  expectations lệch sau 026 (`campaign_code`), fixture code chứa `_` vi phạm
  `chk_campaigns_code_format`, và cleanup của rollback test không đánh dấu version đã gỡ nên
  goose không dựng lại bảng. Phần còn lại của D-01 (package `usecase/subscription` vẫn
  `AutoMigrate`) được tách sang D-06.

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

- **TRẠNG THÁI (2026-10-20, đợt CI-database — RESOLVED)**: chọn đúng phương án tạm thời ở trên và
  biến nó thành mặc định: `make test-db-integration` chạy `go test -race -p 1` (ghi lý do ngay
  trong Makefile), và job `db-integration` dùng chính target đó. Hai gói không còn chạy song song
  trên cùng database nên cửa sổ flake biến mất mà không cần `pg_advisory_lock`. Nếu sau này muốn
  bỏ `-p 1`, phải làm helper lock session-level như mô tả ở trên trước.

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

- **Đã gỡ thế nào (lần 1 — suy luận, và SAI một phần)**: orchestrator đã hiệu chỉnh `spec.md` §11.2–11.3 và
  `quickstart.md` V1–V2: phần đảm bảo được giữ nguyên là **0 dòng cấp**. Tập 12 mã `reason` **không đổi**.

- **Đã gỡ thế nào (lần 2 — sau khi đo thật trên dev stack 04/10/2026)**: bản hiệu chỉnh lần 1 còn **một
  câu sai** — nó nói đường tra cứu theo mã vẫn cho `campaign_sleeping`. Đo thực nghiệm thì **không**:

  | Đường | Tình huống | `reason` quan sát được | Nguồn |
  |---|---|---|---|
  | Tra cứu theo mã | `verify-026` đã đóng cưỡng bức | **`outside_window`** | `campaign_query_claims_uc.go:214` |
  | Tra cứu theo mã | `launch-2026-10` còn mở, hết quota | **`quota_main`** | `eligibility = eligible_but_exhausted` |
  | Thông điệp | sổ **rỗng** (chỉ có chiến dịch đã đóng ⇒ `ListOpen` rỗng) | `campaign_not_configured` | `admission_uc.go:81-82` |
  | Thông điệp | sổ **còn** chiến dịch mở khác | rơi xuống chiến dịch đó, lý do **của chiến dịch đó** | `admission_uc.go:94-113` |

  Nguyên nhân gốc của sự lệch: **đường tra cứu theo mã KHÔNG dùng chung `evaluate()`** — nó tự chọn
  ứng viên và gán `outside_window` khi không ra ứng viên nào. Hai đường cùng trả lời "chiến dịch này
  còn cấp được không" nhưng **không** cùng bộ mã. Bản hiệu chỉnh lần 1 đã giả định chúng chung mã.
  Cả `quickstart.md` V2 và `spec.md` §11.3 đã viết lại theo bảng trên.

  `campaign_sleeping` vẫn còn đúng nghĩa và còn được chứng minh, nhưng **chỉ** cho chiến dịch **còn mở**
  đã qua `ends_at` — không phải cho chiến dịch bị đóng cưỡng bức.

- **Không sửa mã, và không thêm kiểm thử khẳng định ngược lại.** Đổi `ListOpen` để giữ chiến
  dịch đã đóng sẽ phá lại chính quyết định mà `spec.md` Câu 6 và `research.md` R9 đã chốt (xem
  `campaign_repo.go:73`), nên đó sẽ là một quyết định thiết kế mới, không phải một sửa lỗi.

---

## D-05 — Production **mặc định gửi email thật**, và template production không cảnh báo

Phát hiện khi chạy V1+V2 và phải tự hỏi: `EMAIL_ENABLED` có mặc định `true` ở production không. Câu trả
lời là **có**, và không tầng nào trong chuỗi cấu hình production tắt nó.

| Tầng | Giá trị | Nguồn |
|---|---|---|
| Loader default | **`true`** | `shared/config/loader.go:63` — `v.SetDefault("email.enabled", true)` |
| `configs/config.yaml` | `enabled: true` (ghi tường minh cho dev) | dòng 137 |
| `configs/config.production.yaml` | **KHÔNG có key `enabled`** trong section `email` | section bắt đầu dòng 130 |
| `.env.production` | **KHÔNG** set `EMAIL_ENABLED` | đã kiểm tra |
| `.env.production.example` | **KHÔNG** có `EMAIL_ENABLED` | đã kiểm tra |
| `.env.production` | **CÓ `EMAIL_SMTP_KEY` thật** (Brevo) | đã kiểm tra |

- **Vì sao mặc định là `true` là CỐ Ý, không phải lỗi**: `config.go:464-467` ghi rõ *"A missing key
  therefore means 'send'; only an explicit `enabled: false` turns sending off"*. Lý do đã được nêu ở
  `loader.go:55-62`: nếu mặc định là `false`, mọi deployment nâng cấp từ config viết trước khi có key
  này sẽ **âm thầm ngừng gửi email**. Đó là thất bại im lặng, nên thiếu key phải nghĩa là "vẫn gửi".

- **Khoảng trống cần lưu ý**: hiến pháp §"Outbound email volume during verification" yêu cầu bulk
  fixtures **không được** chạm relay SMTP thật, và hạn mức *"does not refill"*. Nhưng cảnh báo duy nhất
  nằm ở `configs/config.yaml` — file **dev** (`enabled: true` kèm dòng *"Set it to false before a bulk
  verification run"*). Ở phía production **không có** key này, và `.env.production.example` **không
  nhắc** `EMAIL_ENABLED`. Hệ quả: chạy một kịch bản nhiều tài khoản ở chế độ production mà không biết
  chuyện này sẽ **gửi một email thật cho mỗi tài khoản**, tới địa chỉ thật, và tiêu hết hạn mức.

- **Vì sao ngoài phạm vi 026**: cơ chế `EMAIL_ENABLED` đến từ **025 T113**. Đã kiểm tra diff 12 commit
  của 026: `configs/config.yaml` và `configs/config.production.yaml` chỉ bị sửa ở phần
  `campaigns`/`campaign_sweep`, **không đụng gì tới `email`**. Việc sửa template production là việc
  của feature hạ tầng hoặc của 025, không phải của 026.

- **Gỡ bằng cách nào** (rẻ, và nên làm sớm):
  1. Thêm `enabled: true` **tường minh** cùng comment cảnh báo vào section `email` của
     `configs/config.production.yaml`, để trạng thái được ghi ra thay vì kế thừa ngầm.
  2. Thêm dòng `EMAIL_ENABLED=true` vào `.env.production.example` kèm chú thích một dòng, để người
     vận hành thấy switch mà không phải đọc source.
  3. Cân nhắc một kiểm thử tĩnh khẳng định **mọi** file `.env*` đều có `EMAIL_ENABLED` được nhắc,
     theo đúng hình thức `TestNoEnvironmentVariableCarriesCampaignData` đã làm cho FR-080.

- **Liên quan 026 thế nào**: lần chạy V1+V2 ngày 04/10/2026 đã **tắt** email đúng hiến pháp (0 email rời
  khỏi máy) và đã **revert** `.env` sau đó. Nhưng sự cần thiết phải biết trước mặc định này chính là
  dấu hiệu khoảng trống đang tồn tại — người chạy kịch bản không có cách nào phát hiện nó nếu không
  đọc source.

- **TRẠNG THÁI (2026-10-20, đợt quick-win — RESOLVED)**: đã làm đủ 3 bước gỡ:
  1. `configs/config.production.yaml` khai `enabled: true` tường minh trong section `email` kèm
     comment giải thích "missing key = send" và cách tắt.
  2. `EMAIL_ENABLED=true` (kèm comment tiếng Anh) đã có trong **cả ba** template:
     `.env.production.example`, `.env.example`, `.env.aws.example` — không chỉ file production.
  3. Test tĩnh mới: `TestProductionConfigStatesTheEmailSwitchExplicitly` (production phải khai key)
     và `TestEveryEnvExampleMentionsTheEmailSwitch` (mọi `.env*.example` phải nhắc switch, fail nếu
     không tìm thấy template nào để tránh pass rỗng). `TestEmailEnabledDefaultsToTrueWhenTheKeyIsAbsent`
     được chỉnh để vẫn chứng minh default trên chính file production sau khi thêm key (dùng bản copy
     đã bỏ block `email:`).

---

### D-06 — `usecase/subscription` vẫn tự dựng schema bằng `AutoMigrate`, che khuất schema module

- **Vấn đề**: `services/account/.../application/usecase/subscription/renewal_integration_test.go`
  còn gọi GORM `AutoMigrate`. Vì GORM dùng tên bảng **không có schema** và `search_path` bắt đầu
  bằng `public`, nó tạo `public.subscription_plans`, `public.user_subscriptions`… ngay cả khi
  `subscription.*` đã tồn tại. Từ đó `public` **che khuất** các bảng thật cho mọi package chạy sau
  trong cùng database: truy vấn không định danh schema rơi vào bản `public` rỗng, nên
  `GetDefaultPlan` trả `nil` và webhook test đỏ dù dữ liệu đúng.

- **Vì sao ngoài phạm vi đợt này**: đây đúng lớp nợ của `025/D-01` (test tự viết schema) nhưng ở
  package khác, và cách sửa không nhỏ: phải chuyển các test đó sang schema dựng bằng migration,
  hoặc cho package một database trống riêng. Đợt CI-database chỉ mở đường chạy bằng chứng; nó
  không có nhiệm vụ chuyển đổi từng harness.

- **Đã làm gì để không chặn**: `make test-db-integration` chạy danh sách package tường minh
  (migrations, persistence, campaign, webhook, ai/grant) thay vì `./...`, và ghi lý do ngay trong
  Makefile. Package bị loại vẫn **chưa từng** chạy trên CI trước đây, nên không có hồi quy — chỉ là
  bằng chứng của nó vẫn còn thiếu.

- **Gỡ bằng cách nào**: chọn một trong hai:
  1. Chuyển `renewal_integration_test.go` (và bất kỳ harness nào còn `AutoMigrate` trong
     `usecase/subscription`) sang yêu cầu database dựng bằng `database/init/*.sql` + migrations,
     như `openWebhookTestDB` đã làm ở `025/D-01`; hoặc
  2. Cho riêng package đó một database trống (`TEST_DATABASE_URL` thứ hai) và thêm biến môi trường
     vào Makefile/CI. Cách này giữ nguyên hành vi test nhưng phải quản lý hai schema.
  Sau khi xong, đổi target về `./...` và xoá ghi chú loại trừ.

- **Ảnh hưởng tới task đã tick**: `026/D-01` (đóng một phần — bằng chứng CSDL đã chạy cho
  campaign/webhook/migrations, chưa gồm package này); `026/D-03` (đóng bằng `-p 1`, nhưng chỉ có
  hiệu lực trên tập package đang chạy).

- **TRẠNG THÁI (2026-10-20 — RESOLVED, chọn hướng 1)**: `openSubscriptionTestDB` không còn
  `AutoMigrate`; nó yêu cầu schema do `database/init/*.sql` + migrations dựng và fail-fast nếu
  không (cùng mẫu `openWebhookTestDB`), kèm chuẩn hoá `search_path` trong DSN. Bốn lỗi fixture mà
  việc bỏ `AutoMigrate` làm lộ ra đã sửa trong cùng đợt:
  1. `cleanupSubscriptionTables` đổi từ `WHERE 1 = 1` sang scope theo hai user fixture + ba slug
     fixture (nếu không, nó xoá luôn `subscription_plans` seed mà package khác cần).
  2. Fixture user được chèn tường minh vào `identity.users` (FK `user_subscriptions.user_id`);
     `uuid.New()` trần bị từ chối trên schema thật.
  3. Bốn test tạo gói `DefaultFree` đang active → chuyển sang dùng gói `free` seed sẵn
     (`ux_single_active_default_free_plan` chỉ cho một gói).
  4. Các gói fixture chỉ cần slug được đặt `PlanKind: Finite` để không rơi vào ràng buộc trên.
  `make test-db-integration` nay chạy account với `./...`; toàn bộ suite (kể cả
  `usecase/subscription`) xanh trên database dựng bằng init + migrations. Job `db-integration`
  của CI phủ lại từ commit này.