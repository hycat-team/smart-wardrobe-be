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

## US2 — Gate + Makefile + V3 + rename + docs (T018–T022)

**Ngày ghi**: 2026-10-16 | **Kết quả**: ĐẠT cả 5 task (không tick `tasks.md` theo lệnh điều phối; không chạy gate repo `lint`/`check`/`test-race`).

### T018 — Gate script `shared/tools/migration-ownership-check/` (`main.go` + `main_test.go`)

- **Vị trí/quy ước**: Go tool trong `shared/tools/` theo đúng mẫu `nginx-check`
  (`go run ./shared/tools/<tên>`), portable cả Windows lẫn sh nên không cần
  nhánh OS trong Makefile. Output CLI tiếng Anh.
- **Single source of truth**: script parse bảng §1 (map schema→cây) và đúng
  dòng bullet §2 (danh sách cấm mỗi luật + ngoại lệ `REFERENCES identity.users(`)
  từ `contracts/ownership-contract.md` — không hardcode danh sách schema nào.
  G-AI = mọi schema trong universe §1 trừ `aiexecutor`, không ngoại lệ. Parse
  hỏng (luật rỗng, schema lạ, thiếu cây) → exit 2 fail-closed, không pass lặng.
- **Miễn trừ ông nội**: đúng 13 tên file chính xác (11 Account viết lại + 2
  baseline), skip theo (service, tree, version) — comment trong code trỏ hợp
  đồng §3 (điều khoản chuyển tiếp) + §4 (versioning). Ghi nhận lệch đánh số:
  lệnh điều phối gọi "§4" cho điều khoản miễn trừ, nhưng trong file hợp đồng
  điều khoản chuyển tiếp nằm ở **§3**, §4 là versioning — code trỏ cả hai.
- Quét `services/*/database/migrations/*.sql` theo dòng: phải có từ khóa
  `ALTER/CREATE/DROP` kèm `"<schema>."` mới tính; comment `--` được cắt trước;
  dòng mang ngoại lệ FK user được bỏ qua. Vi phạm in `LUẬT file:dòng: nội dung`,
  exit 1; sạch in `OK`, exit 0.
- Hai bug bắt được nhờ test + chạy thật (đã sửa, có test hồi quy): (1) universe
  bỏ sót `shared`/`public` vì `continue` trước khi gom schema ở dòng owner lạ;
  (2) dòng §3 nhắc "G-ACCT" ghi đè bullet §2 — chỉ nhận dòng bullet `- ` đầu tiên.
- Xanh trên cây thật: `go run ./shared/tools/migration-ownership-check` →
  `OK — 25 migration file(s) scanned, 13 grandfathered version(s) skipped,
  0 violations`, exit 0. (25 = 35 Account−11 + 4 Commerce−1 + 2 AI−1−embed.)

### T019 — Makefile `migration-check` + đấu vào `check:`

- Target mới sau `vet` (`go run ./shared/tools/migration-ownership-check`,
  không nhánh OS), `.PHONY` + dòng help `quality-help` tương ứng.
- `check: fmt-check tidy-check generate-check nginx-check vet migration-check
  test-race build-all`.
- `make migration-check` → gate OK + `Migration ownership check finished.`,
  exit 0, không warning. Phát hiện ngoài lề: dòng echo `make vet` trong
  `quality-help` thiếu tab đầu dòng nên make hiểu nhầm thành định nghĩa target
  `vet` (cảnh báo `overriding recipe`) — đã sửa tab, warning hết.

### T020 — Quickstart V3 đỏ/xanh (không git: chạy trên bản sao temp)

- Lệnh dựng cây tạm (PowerShell, từ root repo):
  `$tmp = Join-Path $env:TEMP "v3-ownership-gate"` (`C:\Users\Duck\AppData\Local\Temp\v3-ownership-gate`);
  tạo `$tmp\specs\030-ddl-ownership-handover\contracts` + 3 cây
  `$tmp\services\<account|commerce|ai>\database\migrations`; copy hợp đồng +
  toàn bộ `*.sql` 3 cây vào temp.
- Xanh trên cây thật trước khi thử: exit 0 (dòng `OK — 25 ... 0 violations`).
- 3 mẫu vi phạm (mỗi file đúng 1 câu, tên file `99999999_v3_probe_*.sql` không
  thuộc diện miễn trừ):
  - Account: `ALTER TABLE aiexecutor.ai_policy_grants ADD COLUMN v3_probe_acct integer;`
  - Commerce: `ALTER TABLE subscription.subscription_plans ADD COLUMN v3_probe_comm integer;`
  - AI: `CREATE TABLE public.v3_probe_ai (id serial PRIMARY KEY);`
- Đỏ: `go run ./shared/tools/migration-ownership-check -root $tmp` →
  `G-ACCT services/account/.../99999999_v3_probe_acct.sql:2: ...` +
  `G-COMM services/commerce/.../99999999_v3_probe_comm.sql:2: ...` +
  `G-AI services/ai/.../99999999_v3_probe_ai.sql:2: ...` +
  `FAIL — 3 violation(s) in 28 scanned file(s) (13 grandfathered skipped)`,
  exit 1. Đủ cả 3 luật, mỗi luật đúng file + dòng.
- Xanh: xóa 3 file mẫu → cùng lệnh → `OK — 25 ... 0 violations`, exit 0.
- Dọn: `Remove-Item -Recurse $tmp`; `Test-Path $tmp` = False; glob repo
  `**/99999999_*` = 0 file (mẫu chỉ tồn tại trong temp, cây thật nguyên vẹn).
- Không ghi database nào (gate chỉ đọc file), không tạo/không cần drop DB tạm.

### T021 — Đổi tên constraint cũ → `ai_policy_grants_effective_window_check`

- Đã đổi (7 điểm, giữ nguyên nghĩa câu): comment
  `grant_repair_integration_test.go:34`; `docs/operations/shared-reference.md`
  (lệnh grep); `docs/operations/grant-restore-runbook.md` (3 điểm: mô tả
  constraint, nguyên văn lỗi, mục chẩn đoán); `specs/027-grant-repair-unblock/runbook.md`
  (khối trích dẫn); `specs/027-grant-repair-unblock/data-model.md` (bảng ràng buộc).
- Giữ nguyên có chủ ý (dòng lịch sử — xem báo cáo grep cuối batch US2):
  `verification-log.md` + `spec.md` FR-004 + `plan.md` (ghi quá trình rename);
  `baseline-old.sql` (snapshot DB cũ); khối rename `DO` trong baseline AI;
  `deferred.md` D-06 (văn bản vấn đề gốc; T022 chỉ append trạng thái);
  `tasks.md`/`quickstart.md`/`error-classification.md` của spec 027 đã đóng;
  ADR 013 (ADR bất biến, không sửa nội dung cũ).
- Ghi nhận lệch thông tin: lệnh điều phối nêu
  "`grant_error_class_integration_test.go:32` comment" — file hiện tại không
  còn tham chiếu tên cũ ở đó (đã dùng tên riêng
  `ai_policy_grants_error_class_check` + chỉ nhắc D-06), không có gì để đổi.
- Tên bảng/index legacy (`user_ai_policy_grants_*`, trừ đúng CHECK trên) KHÔNG
  thuộc phạm vi T021 — baseline cố ý giữ để khớp DB cũ từng byte.

### T022 — ADR 016 + D-06 resolved

- `docs/decisions/016-ddl-ownership-handover.md` (số tiếp theo sau 015, theo
  cấu trúc ADR 014: Context/Decision/Consequences, thân tiếng Việt): vì sao
  rewrite baseline, luật ông nội, gate, rollback = restore backup. Index trong
  `docs/decisions/README.md` đã thêm dòng 016.
- `specs/027-grant-repair-unblock/deferred.md` D-06: append khối
  `Trạng thái (2026-10-16, spec 030 — RESOLVED)` kèm dấu vết sang spec 030 +
  ADR 016; văn bản vấn đề gốc giữ nguyên.

## T023 — Rehearsal V2 (restore + handover) + đo lock — DỪNG Ở PHÁT HIỆN DIFF (STOP)

**Ngày ghi**: 2026-10-16 | **Kết quả**: HANDOVER SCOPE ĐẠT, FULL-DB PHÁT HIỆN DIFF NGOÀI PHẠM VI → DỪNG theo lệnh worker (không "fix" data, không chạy V5 trên dev live).

**Đường chạy handover**: goose CLI qua `Makefile` (`make migrate-up svc=<name> DB_NAME=<db>`,
đã fix `-table` ở T039: account → `goose_db_version`, ai → `goose_db_version_ai`,
commerce → `goose_db_version_commerce`). Không boot binary ở T023a (tránh worker/email);
V5 binary dự kiến ở bước sau — CHƯA CHẠY.

### T023a-bước 1 — Backup dev + restore vào `verify_d06_rehearsal`

- `pg_dump -U admin -Fc -f /tmp/verify_d06_rehearsal.dump smart_wardrobe_db`
  → 1527231 bytes; `pg_restore --list` đọc được: 410 dòng TOC / 399 entries
  (backup hợp lệ, KHÔNG dump production — chỉ dev).
- `CREATE DATABASE verify_d06_rehearsal` + `pg_restore -d verify_d06_rehearsal`
  (KHÔNG BAO GIỜ là DB live). Đối chiếu sau restore: `goose_db_version` 35 dòng
  max `20261011090002`; `goose_db_version_commerce` 3 dòng max `20260927120000`;
  `goose_db_version_ai` chưa tồn tại; constraint grants còn tên cũ
  `user_ai_policy_grants_check` — đồng nhất dev trước handover.

### T023a-bước 2 — Đếm dòng pre-handover (54 bảng, 8 schema module)

| Nhóm | Kết quả pre |
|---|---|
| `aiexecutor` (7 bảng) | policies 2, audit 0, operations 8, models 7, grants 41, events 478, ledgers 21 |
| `dashboard` (3 bảng) | admin 8, brand 0, summaries 0 |
| `identity` | users 28, refresh_tokens 302, activity_events 0 |
| `subscription` (13 bảng) | plans 2, user_subs 28, sub_events 19, wallets 11, statements 10, deposits 12, pay_events 14, webhook_inbox 14, quotas 25, renewals 0, campaigns 1, claims 6, sweep_runs 1 |
| `wardrobe`/`fashion` | outfits 26, outfit_items 87, wardrobe_items 408, fashion_items 317, categories 9, messages 68, contexts 19 |
| `brand` (15 bảng) | toàn bộ 0 |
| `community` (6 bảng) | posts 6, comments 11, likes 17, media 5, follows 3, follow_stats 5 |

### T023a-bước 3 — Đo thời gian giữ lock rename (cô lập, ROLLBACK giữ nguyên trạng thái)

- Lệnh (trên `verify_d06_rehearsal`, trước handover):
  `BEGIN; SELECT clock_timestamp(); ALTER TABLE aiexecutor.ai_policy_grants
  RENAME CONSTRAINT user_ai_policy_grants_check TO ai_policy_grants_effective_window_check;
  SELECT clock_timestamp(); ROLLBACK;`
- Kết quả: t0 `15:31:18.412745+00` → t1 `15:31:18.41316+00` ≈ **0.415 ms**
  (metadata-only; sau ROLLBACK tên cũ còn nguyên — handover sau đó vẫn làm rename thật).
- Thời gian goose báo cho cả file baseline trên rehearsal (bao gồm mọi CREATE/seed +
  rename, là cận trên của lock DDL): AI `6.92ms`, Commerce `16.69ms`
  (trên DB trắng V1: AI `16.46ms`, Commerce `7.86ms`).
- **Giá trị `lock_timeout` đề xuất (CHƯA CHỐT — chờ điều phối/V5 xác nhận): 10 giây**
  (>20000 lần thời gian rename đo được, vượt xa thời gian cả file baseline, nhưng đủ
  nhanh để abort khi kẹt lock thật trong cửa sổ full-stop 1–2h).

### T023a-bước 4 — Chạy handover trên rehearsal

- `make migrate-up svc=account DB_NAME=verify_d06_rehearsal`
  → `goose: no migrations to run. current version: 20261011090002` (Up ~0, đúng kỳ vọng).
- `make migrate-up svc=ai DB_NAME=verify_d06_rehearsal`
  → `OK 20261016213000_adopt_aiexecutor_ownership.sql (6.92ms)`,
  `successfully migrated database to version: 20261016213000`.
- `make migrate-up svc=commerce DB_NAME=verify_d06_rehearsal`
  → `OK 20261016213001_adopt_dashboard_ownership.sql (16.69ms)`,
  `successfully migrated database to version: 20261016213001`.
- Không lỗi bảng đã tồn tại, không deadlock.
- Assert sau handover: `goose_db_version` **35 dòng** max `20261011090002` (không đổi);
  `goose_db_version_commerce` **4 dòng** max `20261016213001`;
  `goose_db_version_ai` **2 dòng** (0 + baseline) max `20261016213000`;
  constraint grants chỉ còn `ai_policy_grants_effective_window_check` (tên cũ biến mất,
  không trùng); đủ 7 bảng `aiexecutor` + 3 bảng `dashboard`.

### T023a-bước 5 — Đếm dòng post-handover: BẰNG NHAU TUYỆT ĐỐI

- Toàn bộ 54 bảng trên (ai 2/0/8/7/41/478/21, dashboard 8/0/0, users 28,
  refresh_tokens 302, subscription 2/28/19/11/10/12/14/14/25/0/1/6/1,
  wardrobe/fashion 26/87/408/317/9/68/19, brand toàn 0, community 6/11/17/5/3/5)
  — **khớp từng dòng với pre-handover, 0 mất mát**.

### T023a-bước 6 — Dựng DB trắng V1 (`verify_d06_v1ref`) + diff schema V1 vs V2

- V1: `CREATE DATABASE verify_d06_v1ref`, nạp `00-init→01-schema→02-indexes→03-seed`
  từ `/docker-entrypoint-initdb.d/` (xanh), rồi `make migrate-up` account (35/35 xanh,
  gồm các file viết lại T033–T037) → ai (baseline `16.46ms`) → commerce
  (2 cũ + baseline `7.86ms`). Assert V1: 35 / 4 / 2, đủ 7+3 bảng, cũ 0 / mới 1.
- So sánh catalog `aiexecutor`+`dashboard` V1 vs V2 (rehearsal post-handover):
  bảng **10/10 đồng nhất**, cột **156/156 đồng nhất**, constraint (tên+loại)
  **48/48 đồng nhất**, index **27/27 đồng nhất**; định nghĩa CHECK chỉ khác cách
  render cast của pg (`ARRAY['strict'::text...]` vs `ARRAY[('strict'::varchar)::text...]`,
  tập giá trị enforce giống hệt — cosmetic do version deparse, không phải semantic).
- **DIFF THÔ `pg_dump --schema-only` V1 vs V2: CÓ 4 điểm khác THẬT, toàn bộ nằm ở
  schema `subscription` (ngoài phạm vi handover), nguyên văn**:
  1. `subscription.subscription_plans.plan_kind` + `tier_rank`: V1 có `DEFAULT 0`,
     V2 (và dev live) KHÔNG có default.
  2. `CONSTRAINT deposit_transactions_gateway_reference_key UNIQUE (gateway_reference)`
     trên `subscription.deposit_transactions`: V2 (và dev live) CÓ, V1 KHÔNG có.
  3. `CREATE INDEX idx_webhook_inbox_processing ON subscription.provider_webhook_inbox ...`:
     V1 CÓ, V2 (và dev live) KHÔNG có.
  4. `subscription.user_subscriptions.started_at`: V1 có `DEFAULT now()`,
     V2 (và dev live) KHÔNG có default.
  (Các diff còn lại là cosmetic: token `\restrict` của pg_dump, ngắt dòng thân function
  `unaccent`, thứ tự phần tử trong `ANY (ARRAY[0,3,4,5])` vs `ANY (ARRAY[3,0,4,5])` —
  tương đương ngữ nghĩa.)
- **Chứng minh pre-existing (không do handover gây ra)**: cả 4 điểm trên đều có trạng
  thái V2 == dev-live (truy vấn trực tiếp `smart_wardrobe_db`, chỉ đọc), trong khi các
  file handover chứa 0 DDL `subscription.*` (gate T018/T020 đã chứng minh cây Account
  sạch + 2 baseline chỉ chạm `aiexecutor`/`dashboard`). Tức replay trắng từ
  init-hiện-tại + history-mới KHÔNG tái hiện byte-for-byte shape `subscription` của dev
  (drift có từ trước, ngoài phạm vi spec 030 — KỂ CẢ drift `user_style_profiles` đã vá
  ở T038 thì 4 điểm này vẫn còn).

### T023 — QUYẾT ĐỊNH STOP (tuân lệnh worker: diff schema → STOP, không "fix" data)

- **V5 trên DB dev live: KHÔNG CHẠY** (chưa backup-v5, chưa boot binary vào dev, chưa
  smoke). Dev giữ nguyên trạng thái T001: 35 / 3 / chưa có bảng AI, tên constraint cũ
  còn nguyên (xác minh lại sau dọn dẹp — xem dưới).
- **T024 (cutover-runbook.md): CHƯA VIẾT** — runbook cho cutover production không được
  xuất bản trên rehearsal đỏ (thiếu V5 xanh + smoke + ngưỡng abort đã kiểm chứng).
- **T025 (Down trên rehearsal): CHƯA CHẠY** — dừng cùng batch; ghi nhận phân tích tĩnh:
  `Down` của 2 baseline là `DROP TABLE IF EXISTS` toàn bộ 7+3 bảng (KHÔNG rename ngược),
  nên trên DB hội tụ (có dữ liệu) `Down` sẽ XÓA dữ liệu chứ không "tên cũ trở lại" như
  văn bản T025 mô tả — mâu thuẫn spec-vs-code cần điều phối chốt (rollback thật theo
  FR-007 là restore backup, không phải Down).
- **Dọn dẹp**: đã `DROP DATABASE verify_d06_rehearsal` + `verify_d06_v1ref`, xóa
  `/tmp/verify_d06_rehearsal.dump` + 2 file schema dump trong container;
  `pg_database LIKE 'verify_d06%'` = 0 dòng. Không `down -v`, không xóa volume,
  không chạm production, không ghi gì vào dev (dev chỉ đọc suốt batch trừ lệnh DROP/CREATE
  DB tạm).
- Xác minh dev nguyên trạng sau batch: `goose_db_version` 35/`20261011090002`,
  `goose_db_version_commerce` 3/`20260927120000`, `goose_db_version_ai` chưa tồn tại,
  5 constraint grants vẫn tên cũ (`user_ai_policy_grants_check`, ...).
- **Cần điều phối quyết định**: (a) chấp nhận 4 diff `subscription` là drift ngoài phạm
  vi (ghi `deferred.md` ở T030) rồi cho chạy V5; hay (b) điều tra drift trước giờ G;
  (c) chốt lại kỳ vọng T025 (Down vs restore-backup) và (d) chốt `lock_timeout = 10s`.

## Quyết định ràng buộc của điều phối (Phase 5 retry, 2026-10-16)

Worker tuân thủ nguyên văn, không quyết định lại, không điều tra lại:

1. **4 diff schema `subscription` (T023-bước 6) là drift có sẵn, ngoài phạm vi,
   được CHẤP NHẬN**: câu lệnh định nghĩa chúng nằm trong file không đụng tới
   `20260620_payment_lifecycle.sql` (Up dòng 1–265, Down dòng 267–370), các file
   viết lại không bao giờ chạm chúng; V2 == dev-live ở cả 4 điểm. KHÔNG sửa,
   KHÔNG truy diff thêm. Ghi nhận chấp thuận tại đây; T030 file vào `deferred.md`
   nguyên văn 4 mục dưới đây:
   - (i) `subscription.subscription_plans.plan_kind` + `tier_rank`: V1 có
     `DEFAULT 0`, V2 (và dev live) KHÔNG có default.
   - (ii) `CONSTRAINT deposit_transactions_gateway_reference_key UNIQUE
     (gateway_reference)` trên `subscription.deposit_transactions`: V2 (và dev
     live) CÓ, V1 KHÔNG có.
   - (iii) `CREATE INDEX idx_webhook_inbox_processing ON
     subscription.provider_webhook_inbox ...`: V1 CÓ, V2 (và dev live) KHÔNG có.
   - (iv) `subscription.user_subscriptions.started_at`: V1 có `DEFAULT now()`,
     V2 (và dev live) KHÔNG có default.
2. **`lock_timeout = 10 giây` ĐÃ DUYỆT** (rename đo 0.415ms, cả file ~17ms).
   Dùng làm ngưỡng abort trong runbook.
3. **T025-revised**: CHỈ là vòng Down/Up trên database DỰNG TRẮNG (DROP an toàn
   ở đó). Rollback trên database có dữ liệu = restore từ backup, không bao giờ
   Down. Thực hiện đúng văn bản T025 mới.
4. **Quy tắc mail của hiến pháp**: MỌI lần boot binary đều đặt
   `EMAIL_ENABLED=false`. Không ngoại lệ.

## T025-revised — Vòng Down/Up trên database trắng `verify_d06_downdb`

**Ngày ghi**: 2026-10-16 | **Kết quả**: ĐẠT (vòng Down/Up khép kín trên DB trắng).

- Dựng trắng: `CREATE DATABASE verify_d06_downdb`, nạp `00-init→03-seed` xanh,
  `make migrate-up` account (toàn chain tới `20261011090002`) → ai (baseline
  `20261016213000`, 37.87ms) → commerce (2 cũ + baseline `20261016213001`).
  Assert trước Down: `aiexecutor` 7 bảng, `dashboard` 3 bảng, versions **35 / 4 / 2**.
- `make migrate-down svc=commerce` → `OK 20261016213001 (18.36ms)`;
  `make migrate-down svc=ai` → `OK 20261016213000 (10.23ms)`.
- Verify sau Down: 0 bảng trong `aiexecutor`/`dashboard`
  (`to_regclass` cả 2 bảng mẫu = NULL), 0 schema còn lại;
  `goose_db_version` **35 dòng** max `20261011090002` (Account nguyên vẹn);
  `goose_db_version_commerce` về **3 dòng** max `20260927120000` (dòng baseline mất);
  `goose_db_version_ai` còn **1 dòng** max `0` (dòng baseline mất, còn dòng `0`).
- `Up` lại xanh: ai `OK (17.6ms)` → commerce `OK (18.5ms)`; assert: 7+3 bảng trở
  lại, versions **35 / 4 / 2**, chỉ còn `ai_policy_grants_effective_window_check`.
- Ghi nhận đúng như phân tích ở T023: `Down` 2 baseline là `DROP TABLE IF EXISTS`
  toàn bộ 7+3 bảng (KHÔNG rename ngược) — trên DB có dữ liệu sẽ XÓA dữ liệu;
  vì vậy rollback thật theo FR-007 là restore backup, runbook §6 ghi rõ.
- Dọn dẹp: đã `DROP DATABASE verify_d06_downdb`; không `down -v`, không xóa
  volume, không chạm production/dev-live (DB tạm duy nhất).

## T023-hoàn tất — Rehearsal V5 trên database dev live (cổng go/no-go SC-007)

**Ngày ghi**: 2026-10-16 | **Kết quả**: ĐẠT 100% (đủ điều kiện đặt giờ G).

Bằng chứng V2 restore-rehearsal (ghi ở mục T023 trên) còn nguyên:
handover scope 10/10 đồng nhất, đếm dòng 54 bảng bằng nhau tuyệt đối,
versions 35/4/2 — KHÔNG chạy lại, đi thẳng V5 theo lệnh điều phối.

### V5-bước 1 — Backup dev đầy đủ + xác minh đọc được

- `pg_dump -U admin -Fc -f /tmp/verify_d06_v5_dev_before.dump smart_wardrobe_db`
  → **1527231 bytes** trong container; bản sao thứ hai trên host
  (`opencode\verify_d06_v5_dev_before.dump`, 1527233 bytes — chênh 2 bytes
  header copy, nội dung đồng nhất).
- `pg_restore --list` đọc được: **399 TOC entries** (khớp mốc T023a), gồm đủ
  TABLE + TABLE DATA + CONSTRAINT + INDEX + FK toàn database — backup hợp lệ,
  đường về sẵn sàng (KHÔNG dump production — chỉ dev).

### V5-bước 2 — Số liệu pre-handover trên dev (55 bảng, 8 schema module)

`ai`: policies 2, audit 0, operations 8, models 7, grants 41, events 478,
ledgers 21; `dashboard`: 8/0/0; `identity`: users 28, refresh_tokens 302,
activity_events 0; `subscription`: plans 2, user_subs 28, sub_events 19,
wallets 11, statements 10, deposits 12, pay_events 14, webhook_inbox 14,
quotas 25, renewals 0, campaigns 1, claims 6, sweep_runs 1; `wardrobe`/`fashion`:
outfits 26, outfit_items 87, wardrobe_items 408, fashion_items 317, categories 9,
messages 68, contexts 19; `brand` 15 bảng toàn 0; `community`: 6/11/17/5/3/5.
(Khớp từng dòng với pre-handover T023a — dev không drift dữ liệu từ đó.)

### V5-bước 3 — Deploy 3 binary mới (EMAIL_ENABLED=false) vào dev live

- `make build-all` xanh (nhúng migrations mới nhất).
- Boot tuần tự Account(:8111) → AI(:8113) → Commerce(:8112) từ `bin/*.exe`,
  env như compose dev (`DATABASE_MIGRATE_ON_START=true`, bảng version
  per-service, `INGRESS_*_URL` trỏ localhost 8111/8112/8113) + bắt buộc
  `EMAIL_ENABLED=false`. Log ra file `v5-<svc>.log` ngoài repo.
- Cả 3: `Running database migrations...` → `Database migrations applied
  successfully.` → `service listening` (account 22:47:00, ai 22:49:49,
  commerce 22:52:00 — serialize qua advisory lock, 0 lỗi bảng đã tồn tại,
  0 deadlock, 0 migration-failed). Health `/api/v1/health` **200 cả 3**.

### V5-bước 4 — Kiểm tra sau handover (toàn bộ xanh)

- Versions: `goose_db_version` **35** max `20261011090002` (không đổi);
  `goose_db_version_commerce` **4** max `20261016213001`;
  `goose_db_version_ai` **2** max `20261016213000`.
- Constraint grant: chỉ còn `ai_policy_grants_effective_window_check`,
  tên cũ biến mất, không trùng.
- Đếm dòng post-handover (55 bảng): **bằng nhau tuyệt đối với pre**
  (ai 2/0/8/7/41/478/21, dashboard 8/0/0, users 28, refresh 302,
  subscription 2/28/19/11/10/12/14/14/25/0/1/6/1, wardrobe/fashion
  26/87/408/317/9/68/19, brand toàn 0, community 6/11/17/5/3/5) —
  **0 mất mát (SC-005)**.
- Schema handover-scope dev-post vs DB trắng V1 (`verify_d06_v1ref` dựng mới
  từ init + history mới): bảng **10/10**, cột **156/156**, constraint **48/48**,
  index **27/27** — đồng nhất ở mức catalog; `pg_dump --schema-only`
  2 schema chỉ còn token `\restrict` và khác biệt render cast cosmetic của pg
  (tập giá trị enforce giống hệt — cùng hiện tượng đã ghi ở T023a).
  **SC-002 trên dev: ĐẠT** (4 diff `subscription` ngoài phạm vi đã được chấp
  nhận ở trên, không chặn).

### V5-bước 5 — Smoke V4 trên dev (qua binary mới)

Tài khoản kiểm chứng đánh dấu rõ `v5smoke.20261016@example.com` / `v5smoke01`
(KHÔNG đụng tài khoản người thật):

- Đăng ký → **200**; OTP đọc từ Redis (`otp:registration:value:...`) →
  confirm → **200**; login → **200** (token 423 ký tự).
- `GET /api/v1/subscriptions/plans` → **200**, 2 gói (catalogue mua gói xanh).
- `POST /api/v1/ai/chat/sessions` → **201** (tạo phiên AI xanh, không tốn
  quota provider); `GET` sessions → **200**, count=1.
- `GET /api/v1/admin/dashboard/overview` bằng token user thường → **403**
  (RBAC còn nguyên sau handover; dashboard admin đọc bằng admin nằm ngoài
  smoke vì không đoán mật khẩu seed — ghi nhận trung thực).
- Grant ghi đúng: tài khoản smoke có **2 grants** (`active` + `closed`, do
  worker cấp khi đăng ký) trong `aiexecutor.ai_policy_grants` qua constraint
  tên mới — không lỗi CHECK.
- **0 email gửi ra ngoài**: log worker ghi nguyên văn `Email delivery skipped:
  outbound email is disabled (email.enabled=false)`, `skippedCount=1,
  successCount=0` — công tắc `EMAIL_ENABLED=false` hiệu lực đã chứng minh.
- Delta do chính smoke tạo (ghi riêng, không tính vào đếm handover):
  users 28→29 (+1 smoke), refresh_tokens 302→304 (+2 phiên smoke),
  grants 41→43 (+2 cấp cho smoke), contexts 19→20 (+1 session smoke).
  Tài khoản smoke được giữ lại có đánh dấu để team nhận diện, không xóa tay
  trên dev.

### V5 dọn dẹp + trạng thái dev sau batch

- Đã kill 3 binary host (`account/ai/commerce` không còn process); stack dev
  containers (code cũ) vẫn chạy bình thường trên schema đã hội tụ (tương thích
  ngược: chỉ đổi tên constraint + thêm bảng đã tồn tại).
- Đã `DROP DATABASE verify_d06_v1ref`, xóa file scope-dump tạm trong container;
  `pg_database LIKE 'verify_d06%'` = 0 dòng. **Giữ lại** backup
  `/tmp/verify_d06_v5_dev_before.dump` (+ bản host) tới khi T031 xong.
- Dev sau V5: versions 35/4/2, constraint tên mới, dữ liệu nguyên vẹn +
  delta smoke đã ghi. Không `down -v`, không xóa volume, không chạm production.
- Ghi nhận sự cố harness (không ảnh hưởng kết luận): lệnh `Start-Process`
  với `&` đầu dòng thỉnh thoảng bị wrapper kill (`Unknown: ChildProcess.kill`)
  — chạy lại giống hệt thì qua (gặp 2 lần ở ai/commerce, cả 2 đều xanh ở lần 2).

## T024 — Runbook cắt over (`cutover-runbook.md`)

**Ngày ghi**: 2026-10-16 | **Kết quả**: ĐẠT (file mới, thao tác tay cho chủ hệ thống).

- Nội dung: preconditions (§0: V5-xanh SC-007, backup + verify restore,
  full-stop 1–2h, FR-011 tới T028, `EMAIL_ENABLED=false` mọi boot kiểm chứng,
  `lock_timeout=10s`), backup/restore (§1), số liệu pre (§2), deploy
  Account→AI→Commerce (§3), monitor lock + abort (§4: query `pg_locks`/
  `pg_stat_activity`, ngưỡng 10s, 6 tiêu chí abort — đỏ là dừng + restore,
  không fix-forward), smoke V4 + checklist post-cutover (§5: versions 35/4/2,
  tên constraint mới duy nhất, đếm dòng bằng nhau, smoke xanh),
  rollback = restore (§6 — ghi rõ KHÔNG dùng Down trên DB có dữ liệu),
  môi trường sót tự hội tụ (§7), đóng cửa sổ (§8).
