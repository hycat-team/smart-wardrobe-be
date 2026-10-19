# Research: Bàn giao quyền sở hữu DDL (030)

**Ngày**: 2026-10-16 | **Spec**: [spec.md](spec.md)

Mọi quyết định dưới đây đã được xác minh bằng mã nguồn hoặc database thật, không phải suy đoán.

## R1 — Goose có đối chiếu nội dung file với lịch sử applied không? (GIẢ ĐỊNH SỐNG CÒN)

- **Decision**: Không. Goose v3.28.0 chỉ track theo **tập version number**, không checksum/compare nội dung. Viết lại nội dung file cũ (giữ nguyên tên + version) là an toàn với database đang chạy; **xóa file vẫn bị cấm** vì lý do khác (xem dưới).
- **Rationale (bằng chứng)**:
  1. Bảng `goose_db_version` đo thật trên dev chỉ có 4 cột `id, version_id, is_applied, tstamp` — không cột checksum nào để mà đối chiếu.
  2. Quét toàn bộ mã nguồn `pressly/goose v3.28.0` (module cache): **0 kết quả** cho `checksum/Checksum`.
  3. `internal/gooseutil/resolve.go → UpVersions(fsysVersions, dbVersions, ...)`: pending = phép toán tập hợp trên version (`v > dbMaxVersion`, missing = version FS chưa applied mà nhỏ hơn max). Nội dung file không tham gia phép tính.
  4. `RunMigrations` của repo (`shared/infrastructure/db/migration.go:82-89`) dựng provider với `WithAllowOutofOrder(MigrationAllowOutOfOrder)` = `true` trên compose — file convergent timestamp cũ vẫn apply thay vì lỗi.
- **Nuance ghi nhận**: provider `Up` bỏ qua im lặng version DB không còn file tương ứng (không lỗi như legacy path), nhưng `Down` version đó sẽ `ErrVersionNotFound` và database trắng replay sẽ **mất DDL** — nên luật "không xóa file" vẫn giữ nguyên, với lý do là fresh-replay + Down chứ không phải checksum.
- **Alternatives considered**: phẫu thuật bảng version (INSERT/DELETE tay) — loại vì không cần thiết khi content-edit đã vô hình với DB cũ; chỉ dùng nếu R1 sai (nó đúng).

## R2 — Kiểm kê đầy đủ file vi phạm (đo thật, đã chốt đóng băng)

- **Decision**: Danh sách đóng băng để viết lại gồm 11 file Account (7 file gốc + 4 file phát hiện ở T016: `20260627143730`, `20260627225700`, `20260629072235`, `20260724173038` — cùng họ DDL/data `aiexecutor`, R2 vòng T016) + 2–3 file baseline mới (AI, Commerce). Không mở rộng thêm sau mốc này; phát hiện mới khác (nếu có) đi `deferred.md`.
- **Rationale**: grep toàn cây `services/account/database/migrations`:
  - `aiexecutor`: `20260622131631` (tạo 5 bảng + CHECK nội tuyến), `20260704062159` (tạo `ai_model_configs` + rename), `20260801000000` (SET SCHEMA ×6), `20260930120000` (4 DDL mới + tạo `ai_cost_policy_audit_logs`), `20260930150000` (UPDATE data).
  - `dashboard`: `20260723000001` (tạo 3 bảng), `20261011090002` (3 cột payos).
  - Sạch đã xác minh: `20261011090001`/`20261001125023`/`20261011090000` (`identity`), cụm campaign (`subscription`), `20260817090000` (`identity`+`shared`), `20260802090000` (`shared` infra).
  - Commerce sạch: `20260925120000` (`community.*`, đúng chủ), `20260927120000` (`brand.*`, đúng chủ — mẫu đúng).
- **Alternatives considered**: chỉ gỡ file mới (20260930/20261011), giữ lịch sử cũ — loại vì user chọn triệt để baseline.

## R3 — Wiring migration cho service AI

- **Decision**: Thêm `services/ai/database/migrations/migrations.go` (`//go:embed *.sql` → `EmbedFS`, copy mẫu Account) + đổi `provideMigrationsFS` ở `services/ai/internal/di/providers.go:10-13` sang return `EmbedFS` + regen wire. Không đụng `app.go` (đã gọi `RunMigrationsIfEnabled` sẵn) và Dockerfile (đã `COPY .../database/migrations /migrations` ở dòng 32).
- **Rationale**: chính comment trong `providers.go` đã ghi sẵn đường này ("future AI migrations add an embed.go and return it here (FR-035)"). Commerce không cần wiring mới.
- **Alternatives considered**: chạy migration AI bằng job tay/`make migrate-up` — loại vì FR-035 bắt service tự chạy khi start.

## R4 — Boot song song + fail-fast đã là mẫu vận hành hiện có

- **Decision**: Giữ nguyên tắc đã chốt: precondition fail-fast (`RAISE EXCEPTION` rõ ràng) + `restart: always` thử lại; cấm skip im lặng.
- **Rationale**: dev compose đã tuần tự hóa (commerce/ai `depends_on account`), prod song song; migration Commerce `20260927120000` (`ALTER brand.brand_items` — bảng do Account tạo) hôm nay đã sống đúng kiểu fail→restart này trên prod. Handover AI/Commerce chỉ lặp lại mẫu đã ổn định, khác ở message lỗi viết rõ để không ai tưởng là bug.
- **Alternatives considered**: thêm edge `depends_on account` vào prod — loại vì user đã chốt giữ song song.

## R5 — Cơ chế gate lâu dài

- **Decision**: Script kiểm tra mới (quét text: cây Account cấm `aiexecutor.|wardrobe.|fashion.|brand.|dashboard.|community.` trừ `REFERENCES identity.users`; cây Commerce cấm `identity.|subscription.|aiexecutor.` trừ FK user; cây AI chỉ cho `aiexecutor.`) + target Makefile `migration-check`, đấu vào `make check` sau `vet` (tiền lệ: `frontend-guide-check` cũng đấu vào `make check` theo hiến pháp).
- **Rationale**: `AGENTS.md` bắt ưu tiên Makefile target hơn lệnh tay; `make lint` hiện chỉ là fmt-check + vet nên cần target riêng. Lệnh ngừng bắn tạm thời (FR-011) giữ quy ước thủ công theo quyết định clarify — gate máy chỉ cho luật lâu dài (FR-009).
- **Alternatives considered**: check chỉ chạy ở CI workflow — loại vì dev cần thấy đỏ ngay local trước khi push.

## R6 — Baseline mới phải là định nghĩa độc lập đầy đủ, không phải delta

- **Decision**: Vì nội dung cũ ở cây Account bị gỡ DDL ngoài chủ, chain baseline ở cây AI/Commerce phải **tự đứng được**: replay từ trắng ra đúng schema cuối (7 bảng + rename tên mới ngay từ đầu bằng tên constraint tường minh; 3 bảng dashboard + cột payos). Mọi file baseline viết idempotent (`IF NOT EXISTS`, `DO` block check `pg_constraint`) để database cũ boot lên là hội tụ no-op.
- **Rationale**: database trắng sau rewrite không còn nhận DDL từ Account — nếu baseline thiếu 1 cột là trắng thiếu vĩnh viễn, trong khi database cũ vẫn đủ (lệch 2 đường, vỡ SC-002). Convergence audit (diff trắng-mới vs restore-cũ-hội-tụ) là bằng chứng duy nhất chứng minh đủ.
- **Alternatives considered**: baseline chỉ chứa delta so với nội dung cũ đã gỡ — loại vì database trắng không còn "nội dung cũ" để delta lên.
