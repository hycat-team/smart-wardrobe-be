# Mô Hình Dữ Liệu — Sổ đăng ký chiến dịch trong cơ sở dữ liệu

**Đầu vào**: [spec.md](./spec.md) · [plan.md](./plan.md) · [research.md](./research.md)

**Nguyên tắc**: mọi thứ mang tên **chiến dịch** chung, không mang tên buổi ra mắt. Chiến dịch đầu
tiên (`launch-2026-10`) là **dữ liệu seed**, không phải mã nguồn.

---

## 1. Tổng Quan Quan Hệ

```text
┌──────────────────────────────────────────────────────────────────────┐
│ subscription.campaigns  (MỚI — nguồn sự thật DUY NHẤT)               │
│  code PK, plan_slug, quota, reserve, starts_at, ends_at,              │
│  closed_at, version, created_at, updated_at                           │
│  + trigger BEFORE UPDATE: bất biến quota/reserve/plan_slug khi có     │
│    lượt cấp (FR-020/FR-021)                                           │
└───────────────┬──────────────────────────────────────────────────────┘
                │ đọc lúc chạy (KHÔNG lúc nạp cấu hình)
                ▼
┌────────────────────────────────┐        ┌──────────────────────────────┐
│ identity.users                 │        │ subscription.campaign_claims │
│  id                            │◄───────┤  user_id  UNIQUE             │
│  created_at                    │        │  campaign_code  (không FK)   │
│  created_by_admin              │        │  plan_slug, plan_id          │
└────────────────────────────────┘        │  grant_bucket                │
                                          │  fills_main_quota            │
                                          │  registered_at/granted_at    │
                                          └──────────────────────────────┘
                │
                │ phát thông điệp
                ▼
┌────────────────────────────────┐        ┌──────────────────────────────┐
│ shared.outbox_records          │        │ shared.admin_audit_logs      │
│  topic = identity.user.created │        │  target_type = 'campaign'    │
│  topic = audit.log             │───────►│  target_id = NULL            │
└────────────────────────────────┘        │  payload.campaignCode        │
                                          └──────────────────────────────┘
                │
                ▼
┌────────────────────────────────┐
│ subscription.campaign_sweep_runs │  (không đổi lược đồ)
└────────────────────────────────┘
```

**Bảng bị thay đổi lược đồ**: đúng **1** bảng mới (`campaigns`) + 1 hàm/trigger + 1 seed.
**Bảng không đổi**: `campaign_claims`, `campaign_sweep_runs`, `identity.users`,
`subscription_plans`, `user_subscriptions`, `shared.outbox_records`, `shared.admin_audit_logs`.
**Bảng của dịch vụ khác**: `aiexecutor.*` — không đụng tới.

---

## 2. Bảng `subscription.campaigns`

### 2.1 DDL

```sql
-- 026 (FR-010..FR-015, FR-020..FR-024, FR-031, FR-032) — campaign registry.
--
-- WHY THIS EXISTS. Feature 025 made a campaign "data, not code" but kept that
-- data in a YAML file compiled into the image (configs/config*.yaml `campaigns`),
-- so opening one still required a commit, a tag and a deploy. 026 moves the
-- registry into the database, which is the only place it can live and be changed
-- without shipping new binaries.
--
-- WHY THE DB IS THE ONLY SOURCE. quota and reserve decide the hard ceiling. Two
-- sources that disagree make every count wrong against both, and the reconciliation
-- rule between them is exactly where the immutability of a live campaign is broken.
-- The `campaigns` block is therefore deleted from both YAML files.
--
-- WHY NO FOREIGN KEY FROM campaign_claims / campaign_sweep_runs. Those
-- `campaign_code` columns are historical text keys; adding a constraint would
-- require reconstructing attributes for every historical code, which the migration
-- cannot do faithfully, and could block migration on live data. Integrity is kept
-- instead by never deleting a campaign (only closing it): there is no delete path.
--
-- WHY quota/reserve/plan_slug ARE IMMUTABLE ONCE GRANTS EXIST. Each claim froze
-- which budget part it came from (grant_bucket) and the watermark is tied to the
-- quota at fill time. Raising quota on a live campaign would move the watermark and
-- corrupt every count. The trigger below enforces this in the data layer, because
-- application discipline does not bind a hand-written UPDATE.

CREATE TABLE IF NOT EXISTS subscription.campaigns (
    code        VARCHAR(64)  PRIMARY KEY,
    plan_slug   VARCHAR(100) NOT NULL,
    quota       INTEGER      NOT NULL,
    reserve     INTEGER      NOT NULL DEFAULT 0,
    starts_at   TIMESTAMPTZ  NOT NULL,
    ends_at     TIMESTAMPTZ,
    -- NULL = not force-closed. Non-NULL is the operator's closure instant (FR-031).
    closed_at   TIMESTAMPTZ,
    -- Optimistic-concurrency token (FR-055). Incremented on every successful
    -- update/close so a stale PATCH is refused with 412 instead of silently
    -- overwriting another administrator's change.
    version     INTEGER      NOT NULL DEFAULT 1,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),

    -- FR-011: restricted alphabet so a code that ends up in a queue or file name
    -- cannot break path traversal. PRIMARY KEY gives NOT NULL and uniqueness.
    CONSTRAINT chk_campaigns_code_format
        CHECK (code ~ '^[a-z0-9][a-z0-9-]{2,63}$'),
    -- FR-014: plan_slug is non-empty at write time. Whether the plan EXISTS and is
    -- ACTIVE is checked at runtime (FR-014) — loading configuration must not touch
    -- the database, and a wrong slug must never crash a service.
    CONSTRAINT chk_campaigns_plan_slug_not_blank
        CHECK (btrim(plan_slug) <> ''),
    -- FR-012: zero would be a campaign that grants nothing, i.e. meaningless.
    CONSTRAINT chk_campaigns_quota CHECK (quota >= 1),
    CONSTRAINT chk_campaigns_reserve CHECK (reserve >= 0),
    -- FR-013: an open-ended campaign has no ends_at; when set it must be after start.
    CONSTRAINT chk_campaigns_window
        CHECK (ends_at IS NULL OR ends_at > starts_at)
);

-- FR-040: the selection reads awake campaigns newest-first. The table is a handful
-- of rows, so one index on the sort axis is enough.
CREATE INDEX IF NOT EXISTS ix_campaigns_starts_at
    ON subscription.campaigns (starts_at DESC);
```

### 2.2 Trigger bất biến (FR-020/FR-021)

```sql
-- 026 (FR-020/FR-021) — a live campaign's budget is immutable.
--
-- WHY A TRIGGER AND NOT A CHECK. A CHECK is a per-row constraint and cannot see
-- subscription.campaign_claims, so it cannot express "immutable only once a grant
-- exists". The application layer refuses first with a friendly Vietnamese 422; this
-- trigger is the backstop that makes a hand-written SQL UPDATE impossible. It is
-- plain PostgreSQL, no vendor feature (constitution §VIII).

CREATE OR REPLACE FUNCTION subscription.campaigns_guard_immutable_budget()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF (NEW.quota, NEW.reserve, NEW.plan_slug)
           IS DISTINCT FROM (OLD.quota, OLD.reserve, OLD.plan_slug)
       AND EXISTS (
           SELECT 1 FROM subscription.campaign_claims
           WHERE campaign_code = OLD.code
       )
    THEN
        RAISE EXCEPTION
            'campaign % has grants; quota, reserve and plan_slug are immutable (026 FR-020/FR-021)',
            OLD.code
            USING ERRCODE = 'check_violation';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_campaigns_immutable_budget
    BEFORE UPDATE ON subscription.campaigns
    FOR EACH ROW
    EXECUTE FUNCTION subscription.campaigns_guard_immutable_budget();
```

`plan_slug` nằm trong nhóm bất biến vì `campaign_claims.plan_slug` đã chụp gói đã cấp
(`entities/campaign_claim.go:47`); đổi gói giữa chiến dịch sẽ làm sổ mâu thuẫn với cấu hình.

### 2.3 Seed một lần (FR-005, FR-090)

```sql
-- 026 (FR-090) — seed the campaign that is already running on production.
--
-- This is the ONE deliberate exception to FR-005 ("in operation only the admin
-- API writes the table"): a migration INSERT, once, idempotent.
--
-- One-time, idempotent. The values match configs/config.production.yaml exactly, so
-- the switch from configuration to the registry does not change any meaning. It must
-- run BEFORE the new binary: a binary reading a table that does not exist yet would
-- grant nothing to anyone.

INSERT INTO subscription.campaigns (code, plan_slug, quota, reserve, starts_at, ends_at)
VALUES ('launch-2026-10', 'premium-monthly', 50, 5, '2026-10-02T00:00:00+07:00', NULL)
ON CONFLICT (code) DO NOTHING;
```

### 2.4 Migration down (FR-092)

```sql
-- +goose Down (seed migration)
DELETE FROM subscription.campaigns WHERE code = 'launch-2026-10';

-- +goose Down (schema migration), reverse order:
DROP TRIGGER IF EXISTS trg_campaigns_immutable_budget ON subscription.campaigns;
DROP FUNCTION IF EXISTS subscription.campaigns_guard_immutable_budget();
DROP INDEX IF EXISTS subscription.ix_campaigns_starts_at;
DROP TABLE IF EXISTS subscription.campaigns;
```

`campaign_claims` không có FK, nên rollback **không mất** lượt cấp nào.

### 2.5 Thực thể `entities.Campaign` (trong module `subscription`)

| Trường Go | Cột | Ghi chú |
|---|---|---|
| `Code string` | `code` | khoá; bất biến sau khi tạo |
| `PlanSlug string` | `plan_slug` | tham chiếu gói, không sao chép |
| `Quota int` | `quota` | ≥ 1 |
| `Reserve int` | `reserve` | ≥ 0 |
| `StartsAt time.Time` | `starts_at` | `TIMESTAMPTZ` |
| `EndsAt *time.Time` | `ends_at` | nil = không đặt |
| `ClosedAt *time.Time` | `closed_at` | nil = chưa đóng |
| `Version int` | `version` | If-Match |
| `CreatedAt`, `UpdatedAt time.Time` | | vận hành |

`TableName() string { return "campaigns" }`.

---

## 3. Bỏ cấu hình khỏi tệp YAML

| Tệp | Thay đổi |
|---|---|
| `shared/config/config.go` | xoá `Campaigns []CampaignConfig` và kiểu `CampaignConfig`; giữ `CampaignSweepConfig` |
| `shared/config/loader.go` | xoá phân tích/ghi đè `campaigns`; giữ `campaign_sweep` |
| `shared/config/validation.go` | xoá `validateCampaigns`, `validateCampaign`, `campaignCodePattern` |
| `shared/config/campaign_validation_test.go` | xoá; thay bằng kiểm thử CSDL/API |
| `shared/config/campaign_config_files_test.go` | **đảo chiều** khẳng định của 025 T106: khoá `campaigns` phải **vắng mặt** ở cả hai tệp YAML; khoá `campaign_sweep` vẫn phải có |
| `configs/config.yaml` | xoá khối `campaigns` (kể cả `sample-2026-11`); giữ `campaign_sweep` |
| `configs/config.production.yaml` | xoá khối `campaigns`; giữ `campaign_sweep` |
| `.env.example`, `.env.production.example` | xoá ghi chú về khối `campaigns` (comment tiếng Anh) |

**Không** xoá `campaign_sweep`: nhịp quét là núm của cơ chế, không phải dữ liệu chiến dịch, và giữ nó
ở cấu hình khiến nạp cấu hình vẫn không chạm CSDL.

---

## 4. Chọn Chiến Dịch (FR-040, FR-041, FR-045)

### 4.1 Truy vấn "còn thức" (R6)

```sql
SELECT * FROM subscription.campaigns
WHERE closed_at IS NULL
ORDER BY starts_at DESC, code DESC;
```

`code DESC` là tie-break tất định khi hai chiến dịch cùng `starts_at`. Bộ đánh giá thuần sau đó lọc
theo `registered_at` và chọn `starts_at` lớn nhất — giữ nguyên `data-model.md §6.1` của 025.

> **Hiệu chỉnh (sau `/speckit-analyze` vòng 3, khi triển khai).** Bản đầu của mục này đặt **cả hai vế
> cửa sổ theo `now`** vào SQL. Biểu thức đó **loại bỏ** chiến dịch đã hết hạn khỏi tập hàng, nên một
> tài khoản gặp chiến dịch đã kết thúc sẽ thấy tập rỗng và nhận `campaign_not_configured` thay vì
> **`campaign_sleeping`**. Điều đó trái `spec.md` Câu 4 và FR-031, trái `quickstart.md` V2 bước 5, và
> trái hành vi của 025 mà FR-042/FR-043 yêu cầu giữ nguyên.
>
> Vì vậy **SQL chỉ lọc `closed_at IS NULL`**, và vế "còn thức theo `now`" thuộc về **`isAwake` trong bộ
> đánh giá thuần** — đúng như `data-model.md` §4.3 và `research.md` R9 đã nói. Tập ứng viên **kết quả
> y hệt**; chỉ **mã lý do quan sát được** là khác, và mã lý do là thứ hợp đồng quyết định.

| Phần việc | Nơi thực hiện |
|---|---|
| Lọc "chưa bị đóng cưỡng bức" | SQL (`WHERE closed_at IS NULL`) |
| Lọc "còn thức" theo `now` | `isAwake` trong bộ đánh giá thuần |
| Lọc "tài khoản nằm trong cửa sổ" theo `registered_at` | `selectCampaign` trong bộ đánh giá thuần |
| Chọn `starts_at` lớn nhất, tie-break `code DESC` | `selectCampaign` |

Repository có **hai** phương thức đọc: `ListOpen` (chỉ `closed_at IS NULL`) cho **đường cấp gói** và
**đường quét bù**, và `ListAll` cho **API tra cứu** — vì trang quản trị phải hiện được chiến dịch đã
đóng, nếu không `status = "closed"` (FR-032) không bao giờ tới được người đọc.

### 4.2 Đọc lại dưới khoá chia sẻ trong transaction cấp gói (FR-045)

```text
uow.Execute(txCtx):
    LockClaimWindow(campaign.code)                 -- khoá tư vấn (025 R3)
    campaign := GetByCodeForShare(txCtx, code)     -- SELECT ... FOR SHARE
    nếu campaign.closed_at != NULL  -> skip (không ghi)
    mainCount, reserveCount, watermark := ...      -- điều kiện 8
    Create(claim) (ON CONFLICT user_id DO NOTHING) -- điều kiện 9
    applyGrant(...)
```

`FOR SHARE` chặn đóng/sửa trong lúc lượt cấp ghi, mà không chặn các lượt cấp đọc song song. Mọi lượt
cấp đã tuần tự theo khoá tư vấn nên thứ tự khoá là cố định, không deadlock.

### 4.3 "Còn thức" có một định nghĩa

`isAwake(campaign, now)` = `closed_at IS NULL AND starts_at <= now AND (ends_at IS NULL OR now <
ends_at)`. Dùng cho cả điều kiện 2 và tập ứng viên, và khớp với `WHERE` của truy vấn 4.1.

---

## 5. Máy Trạng Thái (6 trạng thái)

Tính lại lúc mỗi lần tra cứu; không lưu (lưu sẽ phải đồng bộ với sổ theo hai đường).

```text
        closed_at IS NOT NULL            ──────────────►  closed   (ưu tiên cao nhất)
                 │
                 ▼
        now >= ends_at (nếu có)          ──────────────►  expired
                 │
        now < starts_at                  ──────────────►  not_started
                 │
        grantedTotal >= quota + reserve   ─────────────►  exhausted
                 │
        grantedMain >= quota             ─────────────►  compensation
                 │
                 ▼
              running
```

| `status` | Nguồn | So với 025 |
|---|---|---|
| `not_started`, `running`, `compensation`, `exhausted`, `expired` | suy ra | không đổi |
| `closed` | `closed_at IS NOT NULL` | **mới**, ưu tiên cao nhất |

**`reason` không đổi (12 mã).** Chiến dịch đã đóng làm điều kiện 2 trả `campaign_sleeping`, đúng nghĩa
"chiến dịch đã kết thúc" mà 025 đã dùng cho "quá `ends_at`".

---

## 6. Trường Dẫn Xuất (bổ sung của 026)

`summarise` (`campaign_query_uc.go`) đã đọc hai đếm và mốc ưu tiên. 026 thêm:

| Trường | Công thức | Ý nghĩa |
|---|---|---|
| `closedAt` | `campaign.closed_at` | nil = chưa đóng |
| `version` | `campaign.version` | FE gửi lại khi sửa/đóng |
| `budgetLocked` | `(grantedMainCount + grantedCompensationCount) > 0` | khoá `quota`/`reserve`/`plan_slug` |

`budgetLocked` không cần truy vấn thêm: tổng hai đếm chính là "đã cấp ít nhất một lượt" vì mỗi lượt
cấp thuộc đúng một bucket (`chk_campaign_claims_bucket`).

---

## 7. Hợp Đồng Ghi (tóm tắt; chi tiết ở `contracts/campaign-registry-api.md`)

| Endpoint | Thân | Quy tắc |
|---|---|---|
| `POST /api/v1/admin/campaigns` | `code, planSlug, quota, reserve, startsAt, endsAt?` | trùng mã → `409`; hỏng hình dạng → `400`; sai nghĩa → `422`; audit `campaign.create` |
| `PATCH /api/v1/admin/campaigns/{code}` | `version, planSlug?, quota?, reserve?, startsAt?, endsAt?` | `GetByCode` trước (→ `404`), rồi mới so version (→ `412`); `budgetLocked` mà đổi 3 cột ngân sách → `422`; audit `campaign.update` |
| `POST /api/v1/admin/campaigns/{code}/close` | `version, reason` | idempotent nếu đã đóng; `reason` rỗng → `422`; ghi `closed_at`; audit `campaign.close`; `status=closed` |
| `GET /api/v1/admin/campaigns/{code}/audit` | `page, limit` | chỉ đọc; lọc `payload.campaignCode = code` |

Bảng mã lỗi **duy nhất** nằm ở `spec.md` FR-054; bảng trên chỉ tóm tắt và **phải khớp** với nó.

Nhãn JSON camelCase. Thời gian RFC3339 có múi giờ. `version` là số nguyên.

---

## 8. Bảng Ánh Xạ Yêu Cầu → Dữ Liệu

| Yêu cầu | Cơ chế dữ liệu |
|---|---|
| FR-001..FR-005 | Bảng `subscription.campaigns` + seed; khối YAML bị xoá |
| FR-010..FR-015 | `CHECK`/`PRIMARY KEY`/`TIMESTAMPTZ`; `plan_slug` kiểm lúc chạy (`plan_unavailable`) |
| FR-020..FR-024 | Trigger `trg_campaigns_immutable_budget` + tầng ứng dụng; `code` bất biến (không có cột/đường đổi) |
| FR-030..FR-034 | `closed_at`, `status = closed`, một chiều |
| FR-040..FR-045 | Truy vấn 4.1 + chỉ mục `ix_campaigns_starts_at`, tie-break `code DESC`; `FOR SHARE` trong transaction |
| FR-050, FR-051 | DTO thêm `closedAt`/`budgetLocked`/`version`; hằng `CampaignStatusClosed`; 12 mã reason giữ nguyên |
| FR-052..FR-055 | Endpoint ghi + `version` (If-Match) + `412` |
| FR-060..FR-063 | `IAuditLogContract` (outbox, cùng transaction) + `targetType='campaign'` + endpoint đọc nhật ký |
| FR-070..FR-072 | Sweep đọc repository; `campaign_sweep_runs` không đổi |
| FR-080..FR-082 | Không cache; không env mới; comment `.env*` tiếng Anh |
| FR-090..FR-093 | Hai migration + rollback; `frontend-guide.md` |
| FR-100..FR-103 | Không đổi gói/AI khách hàng; không sửa `specs/025-` |
