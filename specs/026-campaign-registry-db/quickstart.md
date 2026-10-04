# Cẩu Nang Kiểm Chứng — Sổ đăng ký chiến dịch trong CSDL

Cẩu nang chạy tay để chứng minh tính năng hoạt động đúng. Không phải tài liệu triển khai — thiết kế
nằm ở [data-model.md](./data-model.md), [contracts/](./contracts/).

> **Không** chạy `make hygiene-scan` khi thực hiện cẩu nang này. Lệnh đó sinh tệp báo cáo làm rối kho
> mã nguồn, và `AGENTS.md` quy định phải được cho phép rõ ràng mới chạy.

---

## ⛔ CỔNG AN TOÀN #1 — MÔI TRƯỜNG (đọc trước mục 2)

**Tín hiệu phân biệt duy nhất là tên cơ sở dữ liệu so với `.env.production`.** Không dùng địa chỉ
mạng, không dùng loopback.

Lý do đáng ghi lại: cả môi trường phát triển lẫn production đều chạy Docker Compose với
`DB_HOST=postgres`, nên `inet_server_addr()` trả về địa chỉ bridge ở **cả hai** bên. Địa chỉ loopback
chỉ xuất hiện khi `psql` vào container — tách khỏi đúng đường ứng dụng thật kết nối — nên kiểm tra
loopback cho kết quả **sai theo cả hai chiều** (chặn nhầm dev container hoá, và không chứng minh được
mình không ở production).

```powershell
# production dùng tên nào — lấy từ chính tệp biến môi trường của production (tệp THẬT, không phải .example)
Select-String -Path .env.production -Pattern '^DB_NAME='

# môi trường hiện tại đang nối tới tên nào
psql -h <host> -U <user> -d <db> -c "SELECT current_database(), inet_server_addr()"
```

**Điều kiện chặn duy nhất: `current_database()` KHÔNG được trùng với `DB_NAME` trong `.env.production`.**

Trong kho này, 025 ghi nhận production là `closy_db` và dev cục bộ là `smart_wardrobe_db`;
`.env.production.example` ghi `DB_NAME=closy`. Vì tệp mẫu và tệp thật có thể khác, **tệp thật là
nguồn quyết định**. Nếu hai tên trùng nhau thì **dừng** — V4 chạy migration và V1/V2/V3/V6 xoá dữ liệu.

### Phân loại kịch bản

| Kịch bản | Chạm dữ liệu thật? | Ghi chú |
|---|---|---|
| V1 — mở/đóng không deploy | CÓ — tạo tài khoản + cấp gói + email | Bật/tắt email theo cổng #2 |
| V2 — chiến dịch đã đóng không cấp | CÓ — tạo tài khoản | Tắt email |
| V3 — bất biến ngân sách | CÓ — ghi sổ + cố tình `UPDATE` | Chỉ trên CSDL kiểm thử |
| V4 — trung thực migration | CÓ — migration/rollback | **Không** trên production; dùng bản sao |
| V5 — hợp đồng API đọc | KHÔNG — chỉ đọc | An toàn |
| V6 — quét bù nhận chiến dịch mới | CÓ — tạo tài khoản | Tắt email |
| V7 — đọc nhật ký kiểm toán | CÓ — ghi chiến dịch/lượt cấp qua API | Không đăng ký tài khoản mới nên không gửi email; **phải chờ consumer** |

---

## ⛔ CỔNG AN TOÀN #2 — GIỚI HẠN EMAIL

**Mỗi lượt cấp gói sinh MỘT email qua Brevo. Brevo có trần theo ngày và không nạp lại trong ngày.**

Quy tắc (đã nằm trong hiến phán §Governance):

1. Mọi kịch bản tạo **từ hai tài khoản trở lên** MUST đặt `email.enabled: false` trong
   `configs/config.yaml`, hoặc `EMAIL_ENABLED=false` khi chạy dịch vụ. **Mặc định là `true`**, nên
   phải chủ động tắt. Cấu hình không nạp nóng — cần khởi động lại dịch vụ.
2. Tối đa **một** email được gửi trong toàn bộ lượt kiểm chứng, dành cho kịch bản kiểm nội dung nếu có.
3. Không chạy lại V1/V2/V6 "cho chắc" khi hạn mức còn nguyên.

> Khi công tắc tắt, worker email vẫn nhận message và vẫn **ACK** (vứt hàng loạt trong thời gian tắt).
> Đó là cái giá có chủ ý: trả lỗi sẽ khiến RabbitMQ requeue và khi bật lại sẽ phát cả backlog.

---

## 0. Điều kiện tiên quyết

| Cần | Kiểm tra |
|---|---|
| Go 1.26.0 | `go version` |
| PostgreSQL đang chạy | `make migrate-status svc=account` |
| `goose` đã cài | `goose -version`, nếu thiếu thì `make install-goose` |
| RabbitMQ, Redis đang chạy | `make dev` báo lỗi nếu thiếu |

Nền sạch trước khi bắt đầu:

```powershell
make wire && make lint && go test -race ./services/account/... ./shared/config/...
```

**Điều kiện đạt**: mã trả về 0. Đỏ ngay từ đầu thì dừng — lỗi có sẵn không thuộc phạm vi 026.

---

## 1. Chuẩn bị

### 1.1 Chạy migration

```powershell
make migrate-up svc=account
```

Xác minh bảng, ràng buộc, chỉ mục và trigger:

```sql
SELECT column_name, data_type, is_nullable
  FROM information_schema.columns
 WHERE table_schema = 'subscription' AND table_name = 'campaigns'
 ORDER BY ordinal_position;

SELECT conname FROM pg_constraint
 WHERE conrelid = 'subscription.campaigns'::regclass
 ORDER BY conname;

SELECT indexname FROM pg_indexes
 WHERE schemaname = 'subscription' AND tablename = 'campaigns';

SELECT tgname FROM pg_trigger
 WHERE tgrelid = 'subscription.campaigns'::regclass AND NOT tgisinternal;
```

**Điều kiện đạt**:
- Câu 1 trả về 10 cột: `code, plan_slug, quota, reserve, starts_at, ends_at, closed_at, version,
  created_at, updated_at`.
- Câu 2 trả về 6 ràng buộc: `campaigns_pkey`, `chk_campaigns_code_format`,
  `chk_campaigns_plan_slug_not_blank`, `chk_campaigns_quota`, `chk_campaigns_reserve`,
  `chk_campaigns_window`.
- Câu 3 trả về `ix_campaigns_starts_at`.
- Câu 4 trả về `trg_campaigns_immutable_budget`.

### 1.2 Xác minh seed (V4 chuẩn bị)

```sql
SELECT code, plan_slug, quota, reserve, starts_at, ends_at, closed_at, version
  FROM subscription.campaigns WHERE code = 'launch-2026-10';
```

**Điều kiện đạt**: đúng một dòng `premium-monthly / 50 / 5 / 2026-10-02T00:00:00+07:00 / null / null / 1`.

Chạy lại migration để chứng minh idempotent (không lỗi, không nhân đôi dòng):

```powershell
make migrate-up svc=account
```

### 1.3 Gỡ khối `campaigns` khỏi cấu hình

```powershell
Select-String -Path configs\config.yaml, configs\config.production.yaml -Pattern '^campaigns:'
Select-String -Path configs\config.yaml, configs\config.production.yaml -Pattern '^campaign_sweep:'
```

**Điều kiện đạt**: `campaigns:` **không** còn; `campaign_sweep:` **vẫn** còn (một dòng mỗi tệp).

### 1.4 Khởi động

```powershell
make dev
```

**Điều kiện đạt**: không cảnh báo liên quan tới nạp cấu hình chiến dịch; bộ nhận `identity.user.created`
và worker quét bù vẫn lên lịch như 025.

---

## 2. Bảy kịch bản

### V1 — Mở và đóng chiến dịch **không cần deploy** (bắt buộc)

1. Gọi API tạo (thay `<token>` bằng JWT của quản trị viên):

   ```powershell
   $body = @{
     code = 'verify-026'; planSlug = 'premium-monthly'; quota = 2; reserve = 0
     startsAt = '2020-01-01T00:00:00+07:00'; endsAt = $null
   } | ConvertTo-Json
   Invoke-RestMethod -Method Post -Uri 'http://localhost:8080/api/v1/admin/campaigns' `
     -Headers @{ Authorization = 'Bearer <token>' } -ContentType 'application/json' -Body $body
   ```

2. Đăng ký một tài khoản mới qua email. Trong vòng 5 phút, kiểm tra tài khoản ở gói `premium-monthly`.
3. Gọi API đóng kèm lý do (dùng `version` trả về ở bước 1):

   ```powershell
   $close = @{ version = 1; reason = 'Kiểm chứng V1' } | ConvertTo-Json
   Invoke-RestMethod -Method Post -Uri 'http://localhost:8080/api/v1/admin/campaigns/verify-026/close' `
     -Headers @{ Authorization = 'Bearer <token>' } -ContentType 'application/json' -Body $close
   ```

4. Đăng ký một tài khoản mới thứ hai.
5. **Điều kiện đạt**:
   - Bước 2: tài khoản nhận gói.
   - Bước 4: tài khoản **không** nhận gói. **Xem hiệu chỉnh V2 bên dưới** về mã `reason` quan sát
     được trên đường thông điệp.
   - Trong **suốt** quá trình **không** có commit/tag/deploy nào.
   - Từ bước 3, `GET /api/v1/admin/campaigns/verify-026` trả `status = "closed"`, `closedAt` khác null.

### V2 — Chiến dịch đã đóng **không cấp gói cho tài khoản mới** (bắt buộc)

Trên một chiến dịch đã `closed_at IS NOT NULL`:
1. Tạo một tài khoản mới.
2. **Điều kiện đạt**: **0** dòng `campaign_claims` mới cho tài khoản đó; API tra cứu trả
   `eligibility = "not_eligible"`.
3. **Ghi chú về mã `reason`** — đọc trước khi chạy, nếu không sẽ dễ kết luận sai:
   - Trên **đường thông điệp**, với sổ chỉ có chiến dịch đã đóng, mã quan sát được là
     **`campaign_not_configured`**, **không** phải `campaign_sleeping`. Lý do: `ListOpen` lọc
     `closed_at IS NULL`, nên tập ứng viên thức rỗng và **điều kiện 1** trả về trước khi tới kiểm tra
     chiến dịch.
   - Mã `campaign_sleeping` của 025 là mã của **tầng bộ đánh giá** cho một chiến dịch *có mặt* nhưng
     không thức. Muốn quan sát nó trên CSDL thật thì sổ phải còn **một** chiến dịch khác chưa đóng và
     chưa thức — thiết lập mà bản V1 này không dựng.
   - Dù bất kỳ mã nào, **điều kiện đạt cốt lõi là 0 dòng cấp**. Nếu bạn cần `campaign_sleeping` thì hãy
     kiểm tra bằng `TestEvaluatorExcludesForceClosedCampaigns` thay vì bằng quickstart.
   - Lý do lịch sử: bản soạn đầu của V2 yêu cầu `campaign_sleeping`; yêu cầu đó không quan sát được và
     đã được hiệu chỉnh ở `spec.md` §11.3.

```sql
SELECT count(*) FROM subscription.campaign_claims
 WHERE campaign_code = 'verify-026' AND user_id = '<id-tài-khoản-V2>';
-- kỳ vọng: 0
```

### V3 — Bất biến `quota`/`reserve`/`plan_slug` (FR-020/FR-021)

1. Tạo chiến dịch `verify-026-lock` (quota 2); sửa `quota` khi **chưa** cấp ⇒ `200`.
2. Cấp cho một tài khoản; gọi API `PATCH` đổi `quota` ⇒ nhận `422`.
3. Cố tình sửa thẳng trong CSDL:

   ```sql
   UPDATE subscription.campaigns SET quota = quota + 1 WHERE code = 'verify-026-lock';
   ```

   **Điều kiện đạt**: trigger nâng ngoại lệ (`check_violation`), giá trị không đổi.

### V4 — Trung thực của migration

Trên **bản sao** CSDL có dữ liệu `launch-2026-10`:
1. Chụp output 4 API đọc (`/campaigns`, `/campaigns/launch-2026-10`, `.../claims`, `.../eligibility/<id>`).
2. Chạy `make migrate-up svc=account` hai lần.
3. So output: số đã cấp, `watermarkAt`, `status` **giống hệt** trước migration.
4. `campaign_claims` không đổi:

   ```sql
   SELECT count(*) FROM subscription.campaign_claims WHERE campaign_code = 'launch-2026-10';
   ```

5. Rollback thử — **`goose down` chỉ lùi MỘT phiên bản mỗi lần**, nên phải chạy **hai lần** (một lần cho
   migration seed, một lần cho migration lược đồ):

   ```powershell
   make migrate-down svc=account
   make migrate-down svc=account
   ```

   **Điều kiện đạt**: bảng `campaigns` biến mất; `campaign_claims` vẫn còn nguyên số dòng.

### V5 — Hợp đồng API đọc không đổi (chỉ đọc)

1. Gọi 4 endpoint đọc; so tập khoá JSON với production.
2. **Điều kiện đạt**: chỉ **thêm** `closedAt`, `budgetLocked`, `version`; tập `reason` vẫn **đúng 12
   mã**; `status` chỉ thêm `closed`; không trường cũ nào bị xoá hay đổi kiểu.

### V6 — Quét bù nhận chiến dịch mới

1. Tạm thời để đường thông điệp không xử lý (hoặc chờ quá chu kỳ quét).
2. Tạo chiến dịch `verify-026-sweep` (`starts_at` trong quá khứ) rồi tạo một tài khoản trong cửa sổ.
3. Chờ một chu kỳ quét (mặc định 15 phút, hoặc hạ `campaign_sweep.interval_seconds`; sàn 60 giây).
4. **Điều kiện đạt**: có lượt cấp cho tài khoản đó, và `campaign_sweep_runs.last_sweep_at` của
   chiến dịch được cập nhật.
5. **Ngay sau khi tạo, trước lần quét đầu tiên**: `lastSweepAt` phải là `null` và `degraded` là `false`
   (FR-072).

### V7 — Đọc nhật ký kiểm toán của một chiến dịch

Kịch bản này **không** đăng ký tài khoản mới, nên không sinh email.

1. Tạo chiến dịch `verify-026-audit` (quota 10, reserve 0, `starts_at` trong quá khứ).
2. Sửa một trường thời gian (`endsAt`), rồi đóng chiến dịch — mỗi bước dùng `version` trả về ở bước
   trước. Ghi lại thời điểm thành công của từng bước.
3. **Chờ** bộ phát hàng đỗi và consumer kiểm toán chạy xong (dòng `admin_audit_logs` là hệ quả bất đồng
   bộ của dòng outbox — FR-060; ngưỡng đo được là 1 phút, SC-006).
4. Đọc nhật ký:

   ```powershell
   Invoke-RestMethod -Uri 'http://localhost:8080/api/v1/admin/campaigns/verify-026-audit/audit' `
     -Headers @{ Authorization = 'Bearer <token>' }
   ```

5. **Điều kiện đạt**:
   - Đúng **một** bản ghi mỗi thao tác, `action` lần lượt `campaign.create`, `campaign.update`,
     `campaign.close`.
   - Mọi bản ghi có `targetType = "campaign"`, `targetId = null`, mã trong `payload.campaignCode`.
   - Bản ghi `campaign.update` có `payload.before` / `payload.after` đúng trường vừa đổi.
   - Bản ghi `campaign.close` có `payload.reason` bằng lý do đã gửi.
   - `actorUserId` là quản trị viên đã thao tác; `createdAt` **sau** thời điểm thao tác.
   - Tra mã không tồn tại ⇒ `404`.

---

## 3. Dọn dẹp

Trên CSDL kiểm thử:

```sql
DELETE FROM subscription.campaign_claims WHERE campaign_code LIKE 'verify-026%';
DELETE FROM subscription.campaign_sweep_runs WHERE campaign_code LIKE 'verify-026%';
DELETE FROM subscription.campaigns WHERE code LIKE 'verify-026%';
```

(Không đụng `launch-2026-10`.)

---

## 4. Bảng đối chiếu nhanh

| Kịch bản | Yêu cầu | Tiêu chí |
|---|---|---|
| V1 | FR-030..FR-034, FR-052 | Có hiệu lực < 1 phút, 0 deploy |
| V2 | FR-031, FR-033 | 0 lượt cấp mới sau đóng |
| V3 | FR-020, FR-021 | `422` qua API; trigger chặn SQL |
| V4 | FR-090..FR-092 | Số liệu không đổi; seed idempotent; rollback an toàn |
| V5 | FR-050, FR-051 | Chỉ thêm trường; 12 mã reason |
| V6 | FR-070..FR-072 | Nhận chiến dịch mới trong 1 chu kỳ; `lastSweepAt` null trước lần quét đầu |
| V7 | FR-060..FR-063 | Nhật ký đúng 1 bản ghi/thao tác, `targetId` null, mã trong payload, before/after, reason |
