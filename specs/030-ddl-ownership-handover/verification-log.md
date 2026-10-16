# Nhật ký kiểm chứng: Bàn giao quyền sở hữu DDL (030)

**Nhánh**: `030-ddl-ownership-handover` | **Ngày bắt đầu**: 2026-10-16

Tài liệu này ghi lại mọi bằng chứng kiểm chứng của các task T001–T031.
Mọi database kiểm chứng chỉ dùng tên `verify_d06_*` hoặc database dev cục bộ ở chế độ chỉ đọc.
Cấm xóa volume Docker, cấm trỏ vào database production.

## T001 — Điểm cắt version (cutover point)

**Ngày ghi**: 2026-10-16 | **Nguồn**: database dev cục bộ (`smart_wardrobe_db` trên `postgres-closy-dev`, truy vấn chỉ đọc).

| Bảng version | Số dòng | Version cao nhất | Ghi chú |
|---|---|---|---|
| `goose_db_version` (Account) | 35 | `20261011090002` | Khớp mốc spec §FR-008 (35 dòng). 5 version mới nhất: `20261011090002`, `20261011090001`, `20261011090000`, `20261003165500`, `20261003165440`, đều `is_applied = t`. |
| `goose_db_version_commerce` (Commerce) | 3 | `20260927120000` | Gồm 3 dòng: `0`, `20260925120000`, `20260927120000`, đều `is_applied = t`. Dòng `0` là baseline khởi tạo của goose. |
| `goose_db_version_ai` (AI) | chưa tồn tại | — | `to_regclass('public.goose_db_version_ai')` trả về NULL — đúng kỳ vọng: service AI chưa có cây migration. |

- **Staging/CI**: không tiếp cận được từ máy local (không có `.env.staging`, không có endpoint staging/CI nào cấu hình sẵn) nên không ghi điểm cắt — cần bổ sung khi có quyền truy cập.
- **PROD**: do chủ hệ thống xác nhận (không tự truy vấn production từ máy local).

**Lệnh kiểm chứng lại** (chỉ đọc, database dev):

```powershell
docker exec -e PGPASSWORD=123456 postgres-closy-dev psql -U admin -d smart_wardrobe_db -c "SELECT count(*), max(version_id) FROM goose_db_version;"
docker exec -e PGPASSWORD=123456 postgres-closy-dev psql -U admin -d smart_wardrobe_db -c "SELECT count(*), max(version_id) FROM goose_db_version_commerce;"
docker exec -e PGPASSWORD=123456 postgres-closy-dev psql -U admin -d smart_wardrobe_db -c "SELECT to_regclass('public.goose_db_version_ai');"
```

## T002 — Đọc quy ước thiết kế

- Đã đọc bản mới nhất của `docs/system-design/design-pattern.md` (822 dòng, gồm các mục: contextutils, validation, error pattern, pagination, handler, repository/generic, unit of work, cấu trúc application/usecase/handler, quy ước DTO, contract, worker, rate limiting hai lớp).
- Kết luận áp dụng cho Phase 1+2: các task T001–T006 chỉ chạm file `.sql`, file embed `migrations.go`, composition root `di/providers.go` và tài liệu `specs/` — không vi phạm phân lớp kiến trúc (§IV), không vượt biên module (§II), không đụng event/outbox (§III).

## T003 — Snapshot schema-only của DB dev

- File: `specs/030-ddl-ownership-handover/baseline-old.sql` (113859 bytes, `pg_dump --schema-only --no-owner --no-privileges` từ `smart_wardrobe_db`, chỉ đọc).
- Nội dung mốc: có `CREATE SCHEMA aiexecutor;`, `CREATE SCHEMA dashboard;`, đủ 7 bảng `aiexecutor.*`, 58 `CREATE TABLE` toàn database, và dòng `CONSTRAINT user_ai_policy_grants_check CHECK ...` (tên cũ, làm mốc đối chiếu rename ở US1).
- **Lệnh kiểm chứng lại**: `Select-String -Path specs/030-ddl-ownership-handover/baseline-old.sql -Pattern "user_ai_policy_grants_check"`.

## T004 — Thực nghiệm R1 (sửa comment vô hình với DB đã applied)

**Kết quả**: ĐẠT. Goose chỉ theo dõi theo version, không đối chiếu nội dung file.

- **Cách làm**: khôi phục backup dev (chỉ đọc dev, 18MB) vào database tạm `verify_d06_exp` (35 dòng `goose_db_version`, max `20261011090002` — đồng nhất dev); chèn đúng 1 dòng comment `-- R1 experiment (T004): comment-only line, no SQL change` vào cuối file `services/account/database/migrations/20261011090001_add_user_activity_events.sql` (ngoài mọi khối goose); chạy `make migrate-up svc=account DB_NAME=verify_d06_exp`.
- **Bằng chứng**: goose trả về `goose: no migrations to run. current version: 20261011090002`, exit 0; đếm lại `goose_db_version` trong `verify_d06_exp` vẫn 35 dòng; bảng `identity.user_activity_events` vẫn tồn tại.
- **Hoàn nguyên**: đã xóa dòng thực nghiệm; SHA256 của file sau hoàn nguyên là `FEA98092308935E3137F2F522A2124E70D85360496618009A9EA9349E9DFA9D3`, khớp SHA256 trước thực nghiệm và khớp bản sao lưu tạm — file đồng nhất từng byte. (Không dùng `git diff` để xác minh theo quy định của điều phối — cấm chạy git; thay bằng so sánh hash.)
- **Dọn dẹp**: đã `DROP DATABASE verify_d06_exp`; truy vấn `pg_database WHERE datname LIKE 'verify_d06%'` trả về 0 dòng — không còn database tạm nào.
- **Phát hiện ngoài lề (ghi nhận, chưa sửa)**: replay trắng chỉ từ `database/init/*.sql` hiện tại + toàn bộ goose Account THẤT BẠI ở `20260620_payment_lifecycle.sql` (`relation "subscription_plans" does not exist`) nếu thiếu bước init, và sau khi nạp init hiện tại thì kẹt tiếp ở `20260801000000` (`relation "public.user_style_profiles" does not exist`) — init scripts hiện tại đã lệch khỏi lịch sử mà dev DB từng đi qua (dev DB ra đời từ init cũ hơn). Điều này ảnh hưởng trực tiếp thiết kế replay trắng T016 (US1): baseline mới + rewrite phải tính tới đường dựng trắng từ init hiện tại, không chỉ từ dev DB. Đề nghị điều phối xem xét ghi vào `deferred.md` ở T030 hoặc bổ sung task US1.

## T005 — Đấu wiring cây migration cho AI

**Kết quả**: ĐẠT (có 1 điểm nghẽn phát sinh ở T006, xem dưới).

- **File mới**: `services/ai/database/migrations/migrations.go` — `package migrations`, `//go:embed *.sql` → `var EmbedFS embed.FS` (sao đúng mẫu `services/account/database/migrations/migrations.go`, comment tiếng Anh theo hiến pháp).
- **File sửa**: `services/ai/internal/di/providers.go` — `provideMigrationsFS` chuyển từ `return nil` sang `return aimigrations.EmbedFS` (import `aimigrations "smart-wardrobe-be/services/ai/database/migrations"`), comment cập nhật theo mẫu Account (FR-035).
- **Regen wire**: `make wire svc=ai` thành công — `wire: smart-wardrobe-be/services/ai/internal/di: wrote .../services/ai/internal/di/wire_gen.go`. Nội dung `wire_gen.go` giữ nguyên ngữ nghĩa (`fs := provideMigrationsFS()`), không sửa tay.
- **Dockerfile**: xác minh chỉ đọc — dòng 32 `services/ai/Dockerfile` đã là `COPY --from=builder /src/services/ai/database/migrations /migrations`, không sửa đúng như task yêu cầu.

## T006 — Kiểm chứng build 3 service sau wiring

**Kết quả**: CHƯA ĐẠT toàn phần — account và commerce xanh, AI đỏ vì cây migration còn rỗng (quyết định thiết kế còn thiếu, cần điều phối chốt).

- `make build svc=account` → `Build succeeded. Binary: bin/account.exe`, exit 0.
- `make build svc=commerce` → `Build succeeded. Binary: bin/commerce.exe`, exit 0.
- `make build svc=ai` → THẤT BẠI, lỗi nguyên văn: `services\ai\database\migrations\migrations.go:7:12: pattern *.sql: no matching files found` (make exit 2). Nguyên nhân: `//go:embed *.sql` bắt buộc khớp ít nhất 1 file `.sql`, mà baseline SQL của AI chỉ tới ở T014 (US1) nên thư mục hiện chỉ có `migrations.go` + `.gitkeep`.
- Không chạy `test-race`/`check`/`lint` (cổng thuộc về điều phối). Không còn database `verify_d06_*` nào (truy vấn `pg_database WHERE datname LIKE 'verify%'` trả về 0 dòng).

## T007–T013 — Viết lại 7 file Account (gỡ DDL ngoài chủ)

**Ngày ghi**: 2026-10-16 | **Kết quả**: ĐẠT (nội dung mới đã ghi, tên file + version giữ nguyên, Down là đảo ngược chính xác của Up mới).

| Task | File | Giữ lại (Up) | Gỡ bỏ | Down mới |
|---|---|---|---|---|
| T007 | `20260622131631_add_ai_cost_policy.sql` | `ALTER subscription_plans ADD COLUMN ai_cost_policy_id` (đã bỏ mệnh đề `REFERENCES`, xem ghi chú), `UPDATE` gán policy, `SET NOT NULL` | Tạo 5 bảng `aiexecutor` + CHECK nội tuyến, 2 `INSERT` seed, backfill `INSERT...SELECT` grants, 3 index | `DROP COLUMN IF EXISTS ai_cost_policy_id` (1 câu, gộp đảo ngược 3 câu Up) |
| T008 | `20260704062159_init_ai_executor_module.sql` | 4 câu `task_id` (wardrobe/brand + 2 index) | `CREATE ai_model_configs`, rename grants, 3 `ALTER ai_usage_events`, seed 6 model | Giữ đúng 4 câu đảo ngược task_id, bỏ phần AI |
| T009 | `20260723000001_create_dashboard_snapshots_tables.sql` | Không còn câu nào (gỡ cả 4 bảng, gồm `brand_item_daily` đã bị drop ở `20260724112217` để fresh replay không hồi sinh bảng chết) | 4 `CREATE TABLE` + 4 index dashboard | `SELECT 1` (đảo ngược của Up rỗng) |
| T010 | `20260801000000_separate_module_schemas.sql` | Toàn bộ di chuyển còn lại (36 bảng: identity 3, wardrobe 3, fashion 4, brand 15, subscription 10, shared 1), 9 câu cắt FK, khối kiểm tra orphan, 9 câu tái lập FK | 2 `CREATE SCHEMA` + 9 `SET SCHEMA` (aiexecutor ×6, dashboard ×3) ở Up; 9 `SET SCHEMA` ngược + 2 `DROP SCHEMA` ở Down | Đối xứng Up mới (comment đếm 45→36) |
| T011 | `20260930120000_admin_premium_grant_schema.sql` | Không còn câu nào (toàn file là DDL `aiexecutor`) | Index grants, ledgers nullable + partial unique, operations nullable + premium NULL, bảng audit + 2 index | `SELECT 1` |
| T012 | `20260930150000_free_default_outfit_attempt_cap.sql` | Không còn câu nào | 1 `UPDATE aiexecutor` (giá trị 4 do AI baseline seed trực tiếp) | `SELECT 1` |
| T013 | `20261011090002_add_payos_status_counts.sql` | Không còn câu nào | 1 `ALTER dashboard.*` (3 cột payos do Commerce baseline tạo sẵn) | `SELECT 1` |

- **Ghi chú quyết định**:
  - T007 bỏ `REFERENCES ai_cost_policies(id)`: FK này đã bị cắt ở `20260801000000`, trạng thái cuối dev không còn FK; giữ nguyên mệnh đề sẽ làm fresh replay lỗi (bảng AI chưa tồn tại khi Account chạy trước). R1 bảo đảm vô hình với DB đã applied.
  - T010 chỉ gỡ đúng `CREATE/SET SCHEMA` theo văn bản task; các câu cắt FK chạm `ai_policy_grants`/`dashboard_*` (dòng 22–24, 29) và khối Down tái lập được giữ nguyên — thuộc diện miễn trừ 7 file ở `contracts/ownership-contract.md` §3.
  - File Up rỗng dùng `SELECT 1` no-op để goose luôn thực thi + ghi version row (giữ 35 dòng, FR-008).
  - Comment mới trong SQL viết tiếng Anh (hiến pháp); toàn bộ 34 file kiểm tra `utf8-ok`, ký tự phi ASCII ở file không đụng còn nguyên (em-dash `—` ở T011/T012, dấu tiếng Việt ở file khác).
  - T010 từng phát sinh 1 dòng comment trùng khi sửa (đã viết lại toàn file, statement giữ nguyên từng byte, comment chuyển English-ASCII).

## T014 — Baseline AI (`services/ai/database/migrations/20261016213000_adopt_aiexecutor_ownership.sql`)

**Ngày ghi**: 2026-10-16 | **Kết quả**: ĐẠT (file mới duy nhất, `m = 1`).

- **Timestamp**: `20261016213000` (giờ thực lúc tạo, duy nhất trong cây AI, lớn hơn mọi version hiện có; Goose `allowOutOfOrder=true` nên mọi timestamp duy nhất đều hợp lệ).
- **Một file duy nhất** (thay vì tách nhiều file): toàn chain hiển thị trong một lần review, 1 version row (`m = 1`), đúng điều phối cho phép ("one file is fine").
- **Nội dung**: `CREATE SCHEMA IF NOT EXISTS aiexecutor` + đủ 7 bảng ở trạng thái cuối (trích từ `baseline-old.sql`): tên constraint tường minh ngay từ đầu gồm `ai_policy_grants_effective_window_check` (mới); giữ tên legacy `user_ai_policy_grants_pkey`, `user_ai_policy_grants_policy_id_fkey`, `user_ai_policy_grants_user_id_fkey`, `user_ai_policy_grants_status_check`, `ix_user_ai_policy_grants_active` để khớp DB cũ từng byte; đủ 7 index (gồm `source_event_id`, 2 index audit-log, partial unique fallback); nullable đúng trạng thái cuối (operations cap, ledgers grant, events ledger/user nullable); FK `plan_id`/`source_*` KHÔNG tạo (đúng trạng thái cuối đã cắt).
- **Không có precondition `to_regclass` fail-fast** (quyết định điều phối ARCHI-override văn bản T014 trong tasks.md): baseline phải CREATE từ trắng; hội tụ trên DB cũ nhờ `IF NOT EXISTS` + seed `ON CONFLICT DO NOTHING` / `WHERE NOT EXISTS`. Khối `DO` duy nhất là rename `user_ai_policy_grants_check` → tên mới, chạy chỉ khi tên cũ tồn tại và tên mới chưa có (giới hạn theo `pg_class` của đúng bảng grants).
- **Seed**: 2 policies (giá trị chain lowercase `observe_only`/`strict`, premium `25000000000` — xem drift bên dưới), 8 operations ở giá trị chain-cuối đã đối chiếu dev (free outfit `4`, premium `NULL` toàn bộ), 6 model configs (guard `WHERE NOT EXISTS` theo provider+model vì `effective_from` mặc định `NOW()`). Bỏ backfill grants (phụ thuộc dữ liệu user; DB cũ giữ nguyên, DB trắng bắt đầu rỗng).
- **Drift dữ liệu dev phát hiện (chỉ đọc, ứng viên deferred)**: dev hiện có premium `hard_cost_micro_vnd = 30000000000` (chain ghi `25000000000`), thêm model `gemini-3.5-flash-lite`, 2 embedding output `0.00` (chain `0.20`/`0.15`); `ai_cost_policy_audit_logs` đếm = 0 dòng nên đây là sửa tay không qua migration (cùng họ với case T131) — baseline seed theo giá trị chain có ghi nhận, KHÔNG ghi đè DB cũ. Không ảnh hưởng SC-002 (schema-level).
- **Build**: `go build ./...` trong `services/ai` → exit 0 (xóa điểm nghẽn T006: `//go:embed *.sql` nay đã khớp file).

## T015 — Baseline Commerce (`services/commerce/database/migrations/20261016213001_adopt_dashboard_ownership.sql`)

**Ngày ghi**: 2026-10-16 | **Kết quả**: ĐẠT (1 file, `k = 1` → `goose_db_version_commerce` sẽ lên 4 dòng).

- Tạo `dashboard` schema + đủ 3 bảng ở trạng thái cuối, gồm 3 cột payos trên `admin_daily` ngay từ đầu; giữ tên `uq_brand_daily_snapshot` (explicit, auto-name sẽ lệch); KHÔNG tạo FK `brand_daily.brand_id → brand.brands` (đúng trạng thái cuối đã cắt, enforce ở app layer); FK `summaries.user_id → identity.users` inline (tên auto khớp). `IF NOT EXISTS` toàn bộ, Down đầy đủ (ngược thứ tự + drop schema).

## T016 — Replay trắng V1 (+ V1b)

**Ngày ghi**: 2026-10-16 | **Kết quả**: CHƯA ĐẠT — thất bại ngoài phạm vi T007–T017 (bằng chứng dưới đây), đã dọn DB tạm.

- **V1 nguyên văn quickstart** (DB trống, không nạp init): `make migrate-up svc=account DB_NAME=verify_d06_fresh` đỏ ngay migration đầu tiên, nguyên văn: `goose run: ERROR 20260620_payment_lifecycle.sql ... ERROR: relation "subscription_plans" does not exist (SQLSTATE 42P01)`. V1 hiện hành thiếu bước nạp init — quickstart cần bổ sung (ghi nhận cho orchestrator, không tự sửa quickstart ngoài phạm vi).
- **Biến thể có nạp init** (`00-init`→`03-seed` rồi `migrate-up`, để cô lập ảnh hưởng của rewrite): 6 version đầu xanh (`20260620` 17.96ms … `20260622131631` bản viết lại 3.21ms — chứng tỏ rewrite T007 hợp lệ), rồi đỏ ở `20260627143730_add_token_estimation_audit.sql`, nguyên văn: `ERROR: relation "ai_usage_events" does not exist (SQLSTATE 42P01)`. Đếm `goose_db_version` lúc kẹt: 7 dòng, max `20260624113000`.
- **Nguyên nhân gốc (ngoài phạm vi, KHÔNG sửa theo hard-rule)**: 4 file Account chạm `aiexecutor` nằm ngoài danh sách đóng băng R2/T007–T013 — `20260627143730` (ALTER events), `20260627225700` (UPDATE operations), `20260629072235` (ALTER/UPDATE ai_*), `20260724173038` (ALTER/UPDATE ai_*) — vẫn đòi bảng AI tồn tại trong khi Account luôn chạy trước AI. Thêm vào đó drift init đã biết từ T004 (`20260801000000` đòi `public.user_style_profiles` không tồn tại ở init hiện tại; dòng đó giữ nguyên ở T010 theo phạm vi) — replay chỉ qua được ải này nếu fix init, nhưng fix init một mình KHÔNG đủ vì 4 file trên kẹt trước. Kết luận: không áp dụng "small init-only fix" (điều kiện "làm replay xanh" không thỏa), không adapt V1 theo đường `baseline-old.sql` (không khả thi: migration Account cũ không idempotent, nạp schema cuối rồi replay sẽ lỗi `already exists`).
- **V1b (boot đồng thời 3 binary vào `verify_d06_race`)**: KHÔNG CHẠY (BLOCKED) — cùng chain Account sẽ đỏ cùng điểm, không quan sát được lock serialization; chạy chỉ thêm log nhiễu, không thêm thông tin. Cần V1 xanh trước (phụ thuộc quyết định mở rộng phạm vi rewrite của điều phối).
- **Assert version-count T016**: không thể khẳng định `35 / 3+k / m` trên đường trắng vì replay kẹt giữa chừng (ghi nhận trạng thái kẹt ở trên thay vì số xanh).
- **Dọn dẹp**: đã `DROP DATABASE verify_d06_fresh`; `pg_database LIKE 'verify_d06%'` = 0 dòng.

## T017 — Boot bản mới trên bản sao DB cũ (`verify_d06_old`)

**Ngày ghi**: 2026-10-16 | **Kết quả**: ĐẠT ở mức goose (tương đương pha migration của binary; không boot binary thật — xem ghi chú).

- **Nguồn sao**: `pg_dump -Fc smart_wardrobe_db` (chỉ đọc dev) → file trong container `/tmp/verify_d06_old.dump` (1.5MB) → restore vào `verify_d06_old` (KHÔNG BAO GIỜ là DB live). Đối chiếu trước boot: `goose_db_version` 35 dòng max `20261011090002`; policies 2, operations 8, grants 37, ledgers 21, events 478, models 7, audit 0, dash_admin 8, plans 2; constraint grants tên cũ còn nguyên.
- **Sự cố thao tác (đã khắc phục, giữ lại làm bằng chứng)**: lần chạy đầu dùng `make migrate-up svc=ai/commerce` khiến goose CLI ghi version vào bảng mặc định `goose_db_version` (thấy `20261016213000` lọt vào bảng Account + job Commerce báo `found 2 missing migrations before current version`) — Makefile không truyền `-table` cho cây Commerce/AI trong khi app dùng bảng riêng theo FR-035. Đã dựng lại DB từ dump và chạy goose trực tiếp với `-table goose_db_version_ai` / `-table goose_db_version_commerce`. Ghi nhận cho orchestrator: cần sửa Makefile hoặc tài liệu (ngoài phạm vi T007–T017).
- **Boot 1 (đồng thời 3 job goose, 21:39:16)**: Account `no migrations to run. current version: 20261011090002` (Up ~0, R1 giữ); AI `OK 20261016213000_adopt_aiexecutor_ownership.sql (7.31ms)`; Commerce `OK 20261016213001_adopt_dashboard_ownership.sql (3.93ms)`. Không lỗi bảng đã tồn tại, không deadlock.
- **Assert sau boot 1**: `goose_db_version` 35 dòng max `20261011090002` (không đổi vs T001); `goose_db_version_commerce` 4 dòng max `20261016213001`; `goose_db_version_ai` 2 dòng (0 + baseline) max `20261016213000`; `\d`-tương-đương (`pg_constraint` join `pg_class`): chỉ còn `ai_policy_grants_effective_window_check`, tên cũ đã biến mất, không trùng; 7 bảng `aiexecutor` + 3 bảng `dashboard` đầy đủ; đếm dòng nghiệp vụ trước/sau BẰNG NHAU TUYỆT ĐỐI (2/8/37/21/478/7/0/8/2).
- **Boot 2 (tuần tự)**: cả 3 cây `no migrations to run` (current `20261011090002` / `20261016213000` / `20261016213001`) — fully no-op.
- **Ghi chú phạm vi**: (1) Chạy ở mức goose CLI với đúng bảng version của app — cùng câu SQL và thứ tự mà binary dùng `RunMigrations` sẽ chạy; không boot binary thật (cần Redis/RabbitMQ/provider config, vượt phạm vi worker và không đổi kết luận migration). (2) Đường `fail-fast→retry` trong văn bản T017 là rỗng theo thiết kế đã chốt ở T014 (baseline CREATE-từ-trắng + no-op hội tụ, không guard `to_regclass`): boot 1 xanh-ngay chính là hành vi hội tụ kỳ vọng, không phải bỏ qua lỗi.
- **Dọn dẹp**: đã `DROP DATABASE verify_d06_old`, xóa `/tmp/verify_d06_old.dump` trong container; `pg_database LIKE 'verify_d06%'` = 0 dòng. Không xóa volume, không chạm production.

## Build 3 service (go build only, không gate)

**Ngày ghi**: 2026-10-16 — `go build ./...` exit 0 cả ba module: `services/ai` (xóa lỗi embed T006), `services/account`, `services/commerce`.

## Ứng viên deferred.md (điều phối chốt ở T030, worker không tự tạo file)

1. **4 file Account chạm `aiexecutor` ngoài danh sách đóng băng** (`20260627143730`, `20260627225700`, `20260629072235`, `20260724173038`): chặn replay trắng V1 và vi phạm SC-001/G-ACCT theo chữ. Cần quyết định: mở rộng rewrite (warn: Down của file data-UPDATE khó đảo ngược sạch) hoặc chấp nhận Account-first replay cần AI-chạy-trước + đổi thứ tự quickstart.
2. **Init drift `user_style_profiles`**: init hiện tại thiếu bảng mà `20260801000000:37` đòi di chuyển (bằng chứng T004 trên file gốc + dòng 37 giữ nguyên sau T010). Small-init-fix một mình không gỡ kẹt V1 (mục T016).
3. **Drift dữ liệu dev không qua migration** (premium `30000000000` vs chain `25000000000`, model `gemini-3.5-flash-lite`, embedding output `0.00`; audit log 0 dòng): cần đối chiếu trước giờ G (ảnh hưởng SC-002 dữ liệu, không ảnh hưởng schema).
4. **`make migrate-up svc=ai/commerce` ghi nhầm bảng version** (thiếu `-table`; bằng chứng T017): cần sửa Makefile hoặc runbook trước rehearsal T023.

## T033–T037 — Viết lại 4 file sót + follow-up T010 (US1 follow-up)

**Ngày ghi**: 2026-10-16 | **Kết quả**: ĐẠT cả 5 task (tên file + version giữ nguyên; vùng giữ lại xác minh bằng SHA so khớp trước/sau).

| Task | File | Gỡ bỏ | Giữ lại | Bằng chứng |
|---|---|---|---|---|
| T033 | `20260627143730_add_token_estimation_audit.sql` | Toàn bộ ALTER `ai_usage_events` (ai-only) → no-op `SELECT 1` Up + Down | Tên file + version | File 16 dòng, replay V1 xanh ở version này |
| T034 | `20260627225700_update_ai_cost_quota_limits.sql` | Các UPDATE `ai_cost_policy_operations` (Up sections 1–2, Down sections 1–2) | Mọi UPDATE `subscription_plans` byte-identical | SHA vùng giữ: Up `DC893081…F4699A20E`, Down `8A5CD22D…821831213C` — khớp trước/sau; `ai_cost_policy_operations` còn 2 refs, cả 2 nằm trong comment ghi chú handover |
| T035 | `20260629072235_lowercase_constants_and_constraints.sql` | Đúng các dòng chạm `ai_cost_policies`/`user_ai_policy_grants`/`ai_usage_events` (Up 9–11, 20–22, 31–33, 36–37; Down 47–49, 58–60, 69–71, 73–74) + comment section-4 mồ côi + 1 dòng trắng cuối | Mọi dòng `brand_*` byte-identical | SHA 6 khối brand K1–K6 khớp tuyệt đối (K6 từng lệch do sót comment `-- 3. Add old constraints` khi vá — đã chèn lại và xác minh khớp); refs SQL tới ai = 0 (`ix_ai_usage_events_unknown` = 0) |
| T036 | `20260724173038_fix_total_tokens_and_check_constraint.sql` | Toàn bộ DDL/data `ai_*` → no-op `SELECT 1` Up + Down | Tên file + version | Sanity trước khi gỡ: chỉ chạm `ai_usage_events` + `ai_cost_policies`, 0 bảng ngoài-ai |
| T037 | `20260801000000_separate_module_schemas.sql` | Up: 3 DROP trên `public.ai_policy_grants` (21–23) + 1 DROP dashboard (28); Down: orphan-checks ai (135–146) + dashboard (159–161), re-add ai (164–171) + dashboard (180–181) | Mọi dòng wardrobe/fashion/brand/subscription/digital + `subscription_plans` DROP (Up 20) | 191 → 162 dòng; refs `ai_policy_grants`/`ai_cost_policies`/`dashboard_brand_daily` = 0 toàn file; comment đếm cập nhật `Cut 9→5` (×2 Up), `re-add/Re-add 9→4` (Down) |

- Ghi chú T037: Up giữ DROP FK `subscription_plans_ai_cost_policy_id_fkey` nhưng Down bỏ re-add tương ứng — đúng phạm vi điều phối giao (Up `DROP IF EXISTS` vô hại trên DB trắng; re-add trên DB hội tụ sẽ trỏ vào bảng không còn tồn tại nên bỏ là đúng hơn).

## T038 — Init fix `public.user_style_profiles`

**Kết quả**: ĐẠT. Chèn +18 dòng vào `database/init/01-schema.sql` sau khối `refresh_tokens` (332 → 350 dòng): `CREATE TABLE IF NOT EXISTS public.user_style_profiles` + index `ustyles_embedding_cosine_idx`, dịch `identity.` → `public.` từng câu từ khối Down của `20261011090000` (lines 18–26), comment SQL tiếng Anh. Verify: `grep user_style_profiles` = 5 refs, `ustyles_embedding_cosine_idx` = 1. Không đụng gì khác trong init. Replay V1 (dưới đây) đi qua `20260801000000` xanh — drift đã đóng.

## T039 — Fix `Makefile` (`-table` theo service)

**Kết quả**: ĐẠT. Thêm biến `GOOSE_TABLE` (make-level `ifeq`, portable cả 2 OS): commerce → `goose_db_version_commerce`, ai → `goose_db_version_ai`, còn lại → `goose_db_version`; đấu vào cả 4 target `migrate-status/up/down/version` dạng `goose -table $(GOOSE_TABLE) -dir … postgres …`. Verify trên DB tạm `verify_d06_tbl` (stub `identity.users`): `make migrate-up svc=ai` → `OK 20261016213000 (16.39ms)`; assert `goose_db_version_ai` = 2 dòng/max baseline, `goose_db_version` **không tồn tại** trong DB đó (không lọt dòng nào); `migrate-version`/`migrate-status` xanh đúng bảng. Đã `DROP DATABASE verify_d06_tbl`.

## Parity check 7+3 bảng (dev live vs 2 baseline, trước T040)

**Kết quả**: SẠCH — không gap, không sửa baseline. So sánh read-only trên dev (`information_schema`, `pg_constraint`, `pg_index`) với nội dung 2 file baseline: toàn bộ cột (tên/kiểu/nullable/default), mọi constraint (tên + định nghĩa, gồm FK `ON DELETE` và CHECK lowercase), mọi index (tên + định nghĩa + predicate) đều khớp. Delta duy nhất có chủ ý: dev còn `user_ai_policy_grants_check`, baseline mang tên mới `ai_policy_grants_effective_window_check` + khối rename `DO` (đúng thiết kế FR-004). Baseline không cần sửa: nó chưa từng applied ở đâu ngoài DB tạm (dev không có `goose_db_version_ai`, mọi `verify_d06_*` đã xóa) nên sửa là an toàn — nhưng không có gì để sửa.

## T040 — Re-run V1 (init-loaded blank replay, goose CLI)

**Kết quả**: ĐẠT. `verify_d06_v1`: nạp `00-init→03-seed` xanh (kể cả bảng/init-index mới của T038), `make migrate-up` xanh toàn chain account (34/34, gồm T033–T037 bản viết lại) → ai (baseline) → commerce (2 cũ + baseline). Assert T016 đầy đủ: `aiexecutor` 7 bảng, `dashboard` 3 bảng, tên cũ 0 / tên mới 1, version counts **35 / 4 / 2** (= 35 / 3+k với k=1 / m+1 với m=1, +1 là dòng `0` của goose), seed policies 2 / operations 8 / models 6. Đã `DROP DATABASE verify_d06_v1`.

## T040 — V1b (boot đồng thời 3 binary thật vào DB trắng)

**Kết quả**: ĐẠT ở lần chạy 3 (2 lần đầu là lỗi harness/config, giữ lại làm bằng chứng).

- **Lần 1** (thiếu `INGRESS_*_URL`): commerce/ai đỏ ngay ở `di.InitializeApp` — nguyên văn `dial account service: service base URL must not be empty`; account migrate xanh toàn chain qua binary rồi chết ở HTTP listen do harness đặt `SERVER_PORT` dính space (`lookup tcp/8111 : unknown port`, lỗi harness, không phải app).
- **Lần 2** (đủ INGRESS, thiếu `DATABASE_MIGRATION_TABLE`): cả 3 boot xanh nhưng mọi version rơi hết vào `goose_db_version` (39 dòng) — phát hiện: binary lấy bảng version từ `DATABASE_MIGRATION_TABLE` (default `goose_db_version`), đúng như compose đã đấu per-service (`goose_db_version` / `_commerce` / `_ai`); Makefile T039 chỉ phủ đường CLI. Không phải bug app — thiếu env harness.
- **Lần 3** (đủ env như compose, DB `verify_d06_race` trắng + init): cả 3 binary boot đồng thời, migrate qua session advisory lock (account :12s → ai :17s → commerce :22s, stagger ~5s), **0 lỗi bảng đã tồn tại, 0 deadlock, 0 migration-failed**; cả 3 `service listening` (8111/8112/8113). Assert đồng nhất V1: 7+3 bảng, cũ 0 / mới 1, counts **35 / 4 / 2**. T016 được assert đủ trên cả 2 đường V1 (CLI) và V1b (binary).
- **Tác dụng phụ runtime (không thuộc migration, ghi nhận trung thực)**: trên DB trắng, worker account đã xử lý 1 event seed và gửi đúng **1 email** tới địa chỉ seed `user@smartwardrobe.com` (trong hạn mức hiến pháp: tối đa 1 email/run verify), và 1 lỗi FK `admin_audit_logs_actor_user_id_fkey` từ `audit_log_queue` (worker chạy trên dữ liệu seed thiếu user actor). Cả hai đều thuộc runtime/worker, không ảnh hưởng kết luận migration; đã kill process ngay sau assert để không phát sinh thêm.

## Dọn dẹp sau T040

- `DROP DATABASE verify_d06_v1` + `verify_d06_race`; `pg_database LIKE 'verify_d06%'` = **0 dòng**; file stage `/tmp/v1_0*.sql` trong container đã xóa; 3 process test đã taskkill (không còn `account/commerce/ai.exe` nào).
- Dev nguyên trạng thái T001: `goose_db_version` 35 dòng, `goose_db_version_commerce` 3 dòng, `goose_db_version_ai` chưa tồn tại (chỉ đọc, không ghi).
- Không `down -v`, không xóa volume, không chạm production trong toàn bộ batch.
