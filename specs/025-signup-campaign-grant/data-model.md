# Mô Hình Dữ Liệu — Cơ chế Chiến Dịch Tặng Gói

**Đầu vào**: [spec.md](./spec.md) · [plan.md](./plan.md) · [research.md](./research.md)

**Nguyên tắc đặt tên**: mọi thứ ở đây mang tên **chiến dịch** chung, không mang tên buổi ra mắt. Lý do
ở R12 và FR-090: cơ chế này được dùng lại nhiều lần, và tên theo buổi ra mắt sẽ thành nói dối ngay
từ chiến dịch thứ hai.

---

## 1. Tổng Quan Quan Hệ

```text
┌─────────────────────────────────────────────────────────────────────┐
│ CẤU HÌNH (tệp YAML, không phải bảng)                                │
│   campaigns[]: code, plan_slug, quota, reserve, starts_at, ends_at  │
└──────────────┬──────────────────────────────────────────────────────┘
               │ tra cứu khi khởi động + khi xử lý
               ▼
┌────────────────────────────────┐        ┌──────────────────────────────────┐
│ identity.users                 │        │ subscription.campaign_claims     │
│  id                           │◄───────┤  user_id          UNIQUE        │
│  created_at                   │        │  campaign_code                   │
│  created_by_admin  ◄─ MỚI     │        │  plan_slug                       │
│  provider                     │        │  plan_id                         │
│  email_verified                │        │  grant_bucket        ◄─ MỚI       │
└────────────────────────────────┘        │  fills_main_quota  ◄─ MỚI       │
         ▲                                │  granted_at                      │
         ▲                                │  expires_at                      │
         │ đọc qua hợp đồng              │  registered_at  (chụp thời điểm   │
         │ (module subscription KHÔNG     │                   tạo tài khoản)  │
         │  được đọc bảng này)            └──────────────────────────────────┘
         │                                            │
         │ phát thông điện                            │ cấp gói qua đường có sẵn
         ▼                                            ▼
┌────────────────────────────────┐        ┌──────────────────────────────────┐
│ shared.outbox_records          │        │ subscription.user_subscriptions  │
│  topic = identity.user.created │        │  current_plan_code               │
│  topic = subscription.plan.*   │───────►│  expires_at, is_auto_renew_enabled│
└────────────────────────────────┘        └──────────────────────────────────┘
                                                │
                                                │ phát subscription.plan.changed
                                                ▼
                                      ┌──────────────────────────────────┐
                                      │ aiexecutor.ai_policy_grants      │
                                      │  (dịch vụ AI — KHÔNG bị sửa)     │
                                      └──────────────────────────────────┘

                    worker quét bù ghi              tra cứu đọc
                                                ┌──────────────────────────────────┐
                                                │ subscription.                    │
                                                │   campaign_sweep_runs            │
                                                │  campaign_code (khoá)            │
                                                │  last_sweep_at, degraded,        │
                                                │  degraded_reason                 │
                                                └──────────────────────────────────┘
  ```

**Bảng bị thay đổi lược đồ**: đúng **3** bảng.
**Bảng chỉ đọc**: `subscription.user_subscriptions`, `subscription.subscription_plans`.
**Bảng của dịch vụ khác**: `aiexecutor.*` — không đụng tới (FR-097, R7).

---

## 2. `identity.users` — bổ sung một cột

### 2.1 Vì sao cần

FR-050 yêu cầu bỏ qua tài khoản do quản trị viên tạo, và cơ chế quét bù (FR-079) đọc từ cơ sở dữ
liệu. Nhưng **BẰNG CHỨNG** cho thấy bảng này không lưu thông tin đó:

`entities.User` (`identity/domain/entities/user.go:15-35`) không có trường nào ghi ai đã tạo. Hai
đường tạo tài khoản dựng ra hai hàng giống hệt nhau về cấu trúc. Không có cột thì cơ chế quét bù
**không phân biệt được** tài khoản kiểm thử của quản trị viên với tài khoản khách hàng, và sẽ cấp
quà cho cả hai — vi phạm FR-050.

Chi tiết và các phương án đã loại: R4.

### 2.2 Thay đổi

```sql
-- 025: FR-050 — nguồn gốc tạo tài khoản.
--
-- WHY THIS EXISTS. `identity.users` records nothing about who created the row, so
-- an administrator-created account is structurally indistinguishable from a
-- self-serve one. The campaign exclusion rule (FR-050) can only be evaluated on
-- the event path, where the publishing code knows which flow it is in -- and the
-- catch-up sweep (FR-079), which reads the database, could not evaluate it at
-- all. It would have granted the promotional plan to the administrator's own
-- test accounts.
--
-- This is a permanent property of the account, not a campaign field: it is
-- equally correct for reporting, support and abuse analysis. Precedent: 024
-- carried RoleSlug on the identity contract for the same reason
-- (services/account/dto/identity/contract_dto.go:19-24).
--
-- NOT DERIVED FROM email_verified. That is a different fact: an administrator
-- may create an account that is already verified, and every self-serve account
-- is verified. Deriving one from the other fails silently.

ALTER TABLE identity.users
    ADD COLUMN created_by_admin BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN identity.users.created_by_admin IS
    'True only when the account was created through the administrator endpoint. Set at exactly one write site; never written from configuration.';
```

### 2.3 Quy tắc ghi

| Đường tạo tài khoản | Tệp | `created_by_admin` |
|---|---|---|
| Xác nhận mã đăng ký | `register_uc.go` | không đặt → mặc định `false` |
| Đăng nhập Google tạo mới | `google_auth_uc.go` | không đặt → mặc định `false` |
| Quản trị viên tạo tài khoản | `admin_uc.go` | **đặt `true`** |

Đây là **đúng một** chỗ đặt `true` trong toàn mã nguồn. Một kiểm thử phải chứng minh điều đó bằng
cách tìm kiếm văn bản, vì trình biên dịch không bắt được.

---

## 3. `subscription.campaign_claims` — bảng mới

### 3.1 Vai trò

Bảng này **vừa** là nơi ghi lại lượt cấp quyền lợi, **vừa** là cơ chế chống trùng, **vừa** là nơi
lưu mốc ưu tiên. Ba vai trò dùng chung một bảng là chủ ý: chúng phải khớp nhau trong cùng một
transaction, tách ra là sinh ra hai nguồn sự thật phải hoà giải.

| Vai trò | Cơ chế |
|---|---|
| Chống cấp trùng cho cùng một tài khoản | `UNIQUE (user_id)` — thông báo giao lại nhiều lần hay chạy chiến dịch khác đều bị chặn bởi chính ràng buộc này (FR-025, FR-012) |
| Chống vượt hạn mức | `COUNT(*)` trong transaction đã giữ khoá tư vấn (FR-022, FR-024) |
| Đếm phần chính và phần dự phòng | cột `grant_bucket` — mỗi lượt cấp tự ghi mình thuộc phần nào (FR-022) |
| Lưu mốc ưu tiên | cột `fills_main_quota` + chỉ mục một phần (FR-030) |

**Hai cột `grant_bucket` và `fills_main_quota` làm hai việc khác nhau, không thay thế nhau.**
Đây là điểm dễ sai nhất của toàn bộ lược đồ, nên nói rõ:

| | `grant_bucket` | `fills_main_quota` |
|---|---|---|
| Câu hỏi trả lời | Lượt cấp này lấy từ phần chính hay phần dự phòng? | Lượt cấp này có phải lượt lấp đầy hạn mức chính không? |
| Giá trị | Mọi dòng đều có | Đúng **một** dòng mỗi chiến dịch, các dòng khác là `FALSE` |
| Dùng để | Đếm `grantedMainCount` và `grantedCompensationCount` | Tìm mốc ưu tiên FR-030 và chặn lần chốt thứ hai |
| Nếu thiếu | Không tách được phần dự phòng khỏi phần chính ⇒ dự phòng không dùng được | Vẫn chạy được, nhưng mất FR-030 |

Không dùng `sequence_no` (`1..50` rồi `51..55`) vì phần thuộc về suy ra từ hạn mức **cấu hình**. Khi
`quota` của một chiến dịch đang chạy bị nâng từ 50 lên 60, mọi lượt cấp cũ sẽ bị diễn giải lại thành
phần chính, số liệu tra cứu sai và mốc ưu tiên dịch chuyển. `grant_bucket` đóng băng ý nghĩa tại
thời điểm ghi nên không bị ảnh hưởng. Hệ quả trở thành quy tắc vận hành: **`quota` và `reserve` của
chiến dịch đang chạy phải bất biến**; muốn đổi thì kết thúc chiến dịch cũ và mở mã chiến dịch mới.

Lý do đầy đủ, kèm phương án bị loại: `research.md` R2b.

### 3.2 DDL

```sql
-- 025: FR-022/FR-024/FR-025/FR-030 — campaign claim ledger.
--
-- WHY THIS EXISTS. A campaign is a data-driven promotion (FR-002), so the
-- "first N accounts" rule needs a durable counter. The counter and the grant
-- must commit together, which is why they are the same row rather than a count
-- in one table and a claim in another.
--
-- WHY user_id IS UNIQUE ALONE, NOT (campaign_code, user_id). FR-012 gives an
-- account at most one participation across the whole campaign mechanism, so a
-- later campaign must skip an account that an earlier one already granted. A
-- composite key would let that account collect a second promotional plan, and
-- each grant would add to its term. campaign_code is kept for attribution and
-- for per-campaign counting, not as part of the key.
--
-- WHY grant_bucket IS A COLUMN HERE, AND WHY NOT sequence_no. The campaign
-- splits its budget into a main part and a reserve part, and the admin query
-- (FR-065) has to report them separately. To count them from a single column
-- you need each row to say which part it came from. Deriving that from
-- sequence_no (rows 1..quota are main, the rest are reserve) would tie the
-- meaning of historical rows to the quota in the YAML file: raising quota from
-- 50 to 60 on a live campaign would silently reclassify every earlier claim,
-- corrupt the admin numbers and move the FR-030 watermark. A frozen enum keeps
-- the past correct no matter how the config changes afterwards.
--
-- WHY fills_main_quota IS SEPARATE AND NOT REPLACED BY grant_bucket. FR-030
-- freezes the priority watermark on the claim that filled the main budget, so
-- later comparison always has the same answer regardless of processing order.
-- That is "the 50th claim", not "a main claim" — 49 other rows are also main.
-- The partial unique index below makes "at most one watermark per campaign" a
-- data-layer guarantee, so two workers running concurrently cannot freeze two
-- different watermarks.
--
-- expires_at IS NOT NULL. subscription.user_subscriptions carries a CHECK that
-- a finite plan (plan_kind = 1) must have an expiry, and the granted plan is
-- finite. Allowing NULL here would let a claim record a grant that could never
-- have been committed.

CREATE TABLE subscription.campaign_claims (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID        NOT NULL
                                  CONSTRAINT fk_campaign_claims_user
                                  REFERENCES identity.users(id) ON DELETE CASCADE,
    campaign_code     VARCHAR(64) NOT NULL,
    plan_slug         VARCHAR(100) NOT NULL,
    plan_id           UUID        NOT NULL,
    -- Chụp thời điểm tạo tài khoản tại lúc cấp, để so sánh với mốc ưu tiên mà
    -- không cần đọc bảng tài khoản (FR-031).
    registered_at     TIMESTAMPTZ NOT NULL,
    expires_at        TIMESTAMPTZ NOT NULL,
    granted_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- FR-022: lượt cấp này lấy từ phần chính hay phần dự phòng. Ghi lúc cấp,
    -- không suy ra từ hạn mức cấu hình.
    grant_bucket      VARCHAR(16) NOT NULL
                                  CONSTRAINT chk_campaign_claims_bucket
                                  CHECK (grant_bucket IN ('main', 'reserve')),
    -- FR-030: đúng một dòng mỗi chiến dịch mang TRUE, đó là lượt lấp đầy phần
    -- chính. KHÔNG dùng cột này để đếm phần chính — xem -- WHY ở trên.
    fills_main_quota  BOOLEAN     NOT NULL DEFAULT FALSE,

    CONSTRAINT ux_campaign_claims_user UNIQUE (user_id)
);

-- FR-030: tối đất một mốc ưu tiên cho mỗi chiến dịch, ràng buộc ở tầng dữ liệu.
CREATE UNIQUE INDEX ux_campaign_claims_watermark
    ON subscription.campaign_claims (campaign_code)
    WHERE fills_main_quota;

-- FR-022: mọi phép đếm hạn mức đọc theo (campaign_code, grant_bucket), chỉ mục
-- này làm chúng chỉ quét phần chính hoặc phần dự phòng thay vì toàn bảng.
CREATE INDEX ix_campaign_claims_campaign_bucket
    ON subscription.campaign_claims (campaign_code, grant_bucket);

-- FR-062: bỏ qua có lý do cần tra lại nhanh theo chiến dịch.
CREATE INDEX ix_campaign_claims_campaign_granted
    ON subscription.campaign_claims (campaign_code, granted_at DESC);
```

### 3.3 Cách đọc được cần thiết

| Câu hỏi nghiệp vụ | Truy vấn | Chỉ mục |
|---|---|---|
| Đã cấp bao nhiêu phần chính? | `COUNT(*) WHERE campaign_code = ? AND grant_bucket = 'main'` | `ix_campaign_claims_campaign_bucket` |
| Đã cấp bao nhiêu phần dự phòng? | `COUNT(*) WHERE campaign_code = ? AND grant_bucket = 'reserve'` | `ix_campaign_claims_campaign_bucket` |
| Còn bao nhiêu suất? | `quota + reserve` trừ tổng hai dòng trên | — |
| Mốc ưu tiên là gì? | `SELECT registered_at WHERE campaign_code = ? AND fills_main_quota` | `ux_campaign_claims_watermark` |
| Đã chốt mốc ưu tiên chưa? | `EXISTS(SELECT 1 WHERE campaign_code = ? AND fills_main_quota)` | `ux_campaign_claims_watermark` |
| Tài khoản này đã tham gia chưa? | `SELECT 1 WHERE user_id = ?` | `ux_campaign_claims_user` |
| Danh sách tài khoản đã cấp (có phân trang) | `WHERE campaign_code = ? ORDER BY granted_at DESC` | `ix_campaign_claims_campaign_granted` |

Hai dòng đầu là cặp đếm bắt buộc và phải luôn dùng cùng nhau. Đếm phần chính **không** được suy ra
từ `NOT fills_main_quota`: cột đó chỉ đánh dấu đúng một dòng, nên phủ định của nó tính cả các lượt
cấp dự phòng và cho ra con số lớn hơn thực tế. Xem `grant_bucket` ở mục 3.1.

Mốc ưu tiên dùng chỉ mục một phần nên truy vấn nó chỉ quét đúng một dòng, không cần thêm chỉ mục.

### 3.4 Ràng buộc toàn vẹn giữa các bảng

| Ràng buộc | Cơ chế bảo vệ |
|---|---|
| Mỗi tài khoản 1 lượt cấp trọn đời cơ chế | `ux_campaign_claims_user` |
| Mỗi chiến dịch ≤ 1 mốc ưu tiên | `ux_campaign_claims_watermark` |
| Lượt cấp chỉ thuộc phần chính hoặc dự phòng | `chk_campaign_claims_bucket` |
| Tổng lượt cấp ≤ `quota + reserve` | Nhánh `grant_bucket` ở mục 6.2: mỗi phần tự giữ trần của nó, nên tổng bị chặn ở tầng luật chứ không cần `CHECK` so với cấu hình |
| Chiến dịch không cấp quá hạn mức | Khoá tư vấn theo transaction + `COUNT` (R3) |
| Lượt cấp không mồ côi khi tài khoản bị xoá | `ON DELETE CASCADE` |
| Mốc ưu tiên không đổi sau khi chốt | Chỉ mục một phần chặn lần chốt thứ hai |

Trần tổng **không** thể biểu diễn thành `CHECK` vì `quota` và `reserve` nằm trong tệp YAML chứ
không nằm trong cơ sở dữ liệu. Một `CHECK` so với hằng số sẽ sai ngay lần đầu tiên cấu hình bị
sửa. Vì vậy trần do luật ở mục 6.2 cùng khoá tư vấn giữ, và mọi phép đếm dùng chính các cột này
để tự đối chiếu.

---

## 4. Cấu hình chiến dịch — hình dạng dữ liệu

Không phải bảng. Nằm trong `configs/config.yaml` và `configs/config.production.yaml`, dưới khoá
`campaigns`. Lý do và quy tắc hình dạng: R10.

```yaml
campaigns:
  - code: launch-2026-10
    plan_slug: premium-monthly
    quota: 50
    reserve: 5
    starts_at: "2026-10-02T00:00:00+07:00"
    ends_at: null            # không bắt buộc (FR-009)
```

### 4.1 Trường và quy tắc kiểm tra

| Trường | Kiểu | Bắt buộc | Quy tắc kiểm tra (FR-010) | Lý do quy tắc |
|---|---|---|---|---|
| `code` | chuỗi | Có | Không rỗng; khớp `^[a-z0-9][a-z0-9-]{2,63}$`; **không trùng** trong danh sách | Mã đi đúng vào cột `campaign_code` và vào nhật ký kiểm toán (FR-038). Ràng buộc định dạng chặn ký tự lạ làm rối tra cứu và chặn truy cấn đường dẫn nếu mã này vô tình được ghép vào tên hàng đỗi |
| `plan_slug` | chuỗi | Có | Kiểm tra **lúc chạy**, không chặn khởi động — xem FR-010 | Gói không hoạt động mà vẫn cấp là cấp thứ không có giá trị thương mại. Nhưng nạp cấu hình không được chạm CSDL, nên chỗ duy nhất tra được là nơi dòng gói vốn được đọc; sai thì bỏ qua với mã `plan_unavailable` |
| `quota` | số nguyên | Có | `>= 1` | FR-021. Bằng 0 nghĩa là chiến dịch không cấp gì, đó là cấu hình vô nghĩa chứ không phải cấu hình sai |
| `reserve` | số nguyên | Có | `>= 0` | FR-021. Cho phép 0 nghĩa là trần cứng bằng đúng `quota` |
| `starts_at` | chuỗi RFC3339 | Có | Có múi giờ tường minh | FR-008. Thiếu múi giờ thì máy chủ UTC chạy lệch 7 giờ (R6) |
| `ends_at` | chuỗi RFC3339 hoặc `null` | Không | Nếu có thì phải **sau** `starts_at` | FR-009, FR-013 |

### 4.1.1 Khoảng thời gian quét bù — nằm ngoài danh sách `campaigns`

FR-083 đòi khoảng thời gian chạy quét bù phải khai báo được. Nó **không** thuộc mảng `campaigns`,
vì lý do kỹ thuật: một worker quét bù chạy một lần cho **toàn bộ** chiến dịch đang hiệu lực, nên
đặt chu kỳ vào từng phần tử sẽ tạo N worker cho N chiến dịch và phải hỏi "ai quét trước".

Nó là một khối cấp hành động duy nhất, cấp ngay dưới `campaigns`:

```yaml
campaign_sweep:
  interval_seconds: 900     # 15 phút, FR-083
  run_on_start: true        # quét một lần lúc khởi động
```

| Trường | Kiểu | Bắt buộc | Quy tắc kiểm tra | Lý do |
|---|---|---|---|---|
| `interval_seconds` | số nguyên | Có | `>= 60` | Dưới 60 giây thì quét nghẹt hàng đỗi khi có lượng lớn tài khoản chờ. FR-083 |
| `run_on_start` | boolean | Không | Mặc định `true` | Chạy lúc khởi động đóng lại khoảng hở của lần quét cuối trước khi dừng dịch vụ (FR-079, SC-016) |

`interval_seconds` **không** có biến môi trường riêng vì nó đã nằm trong tệp YAML cùng cấu hình
chiến dịch; đổi nó bằng cách sửa YAML, giống hệt cách đổi `quota`. Không tạo đường cấu hình thứ hai
cho cùng một ý nghĩa.

### 4.2 Múi giờ

`starts_at` và `ends_at` phân tích bằng `time.RFC3339` ngay lúc nạp cấu hình. Mọi phép so sánh về sau
dùng `timeutils.GetNow(cfg.Database.TimeZone)`. **Không** dùng `time.Now()` ở bất kỳ đâu trong logic
chiến dịch — hiến phán §VIII cấm phụ thuộc múi giờ của máy chủ.

### 4.3 Trường dẫn xuất — không lưu, tính lúc tra cứu

Giao diện tra cứu (FR-065) trả về các đại lượng này. Tất cả đều **dẫn xuất**, không có trường nào
được lưu — một giá trị lưu sẽ phải được đồng bộ bằng tay và sẽ sai.

| Đại lượng | Nguồn |
|---|---|
| `grantedMainCount` | `COUNT(*)` lọc `grant_bucket = 'main'` |
| `grantedCompensationCount` | `COUNT(*)` lọc `grant_bucket = 'reserve'` |
| `grantedTotalCount` | `grantedMainCount + grantedCompensationCount` |
| `remainingMain` | `max(0, quota - grantedMainCount)` |
| `remainingCompensation` | `max(0, reserve - grantedCompensationCount)` |
| `watermarkAt` | `registered_at` của bản ghi `fills_main_quota`; `null` nghĩa là chưa chốt |
| `status` | xem máy trạng thái mục 5 |

Hai dòng đầu là hai số **độc lập**, không phải một số và phần trừ của nó. Dùng `NOT fills_main_quota`
để suy ra số thứ nhất sẽ lẫn phần dự phòng vào phần chính; xem mục 3.1.

Máy trạng thái mục 5 dùng `grantedMainCount` để phân biệt `running` với `compensation`, nên lỗi ở
đây không chỉ làm sai con số hiển thị mà còn làm chiến dịch báo trạng thái sai.

---

## 5. Máy Trạng Thái Chiến Dịch

Được **tính lại lúc mỗi lần tra cứu** và lúc mỗi lần xử lý thông báo. Không lưu, vì lưu trạng thái
sẽ phải đồng bộ với dữ liệu lượt cấp theo hai đường khác nhau.

```text
                    now < starts_at
        ┌──────────────────────────────────┐
        │          not_started              │
        └───────────────┬──────────────────┘
                        │ now >= starts_at
                        ▼
        ┌──────────────────────────────────┐
   ┌───►│  running                        │◄──── đã cấp < quota
   │    │  (chưa chốt mốc ưu tiên)         │
   │    └───────────────┬──────────────────┘
   │                    │ lượt thứ `quota` được ghi
   │                    ▼
   │    ┌──────────────────────────────────┐
   │┌──►│  compensation                   │◄──── đã cấp (chính+dự phòng) < quota+reserve
   ││   │  (đã chốt mốc ưu tiên)           │
   ││   └───────────────┬──────────────────┘
   ││                   │ đã cấp = quota + reserve
   ││                   ▼
   ││   ┌──────────────────────────────────┐
   │└───│  exhausted                      │  tự ngủ (FR-013)
   │    │  tự ngủ                        │
   │    └──────────────────────────────────┘
   │
   │    ở mọi trạng thái trên, nếu now > ends_at (nếu có khai báo):
   │                    ▼
   │    ┌──────────────────────────────────┐
   └────│  expired                        │  tự ngủ (FR-013)
        │  tự ngủ                        │
        └──────────────────────────────────┘
```

### 5.1 Trạng thái sức khỏe — tách khỏi trạng thái vòng đời

FR-059 đòi phân biệt được ba thứ mà một trạng thái vòng đời không diễn tả nổi: chiến dịch **đang
chạy**, chiến dịch **tạm ngủ**, và chiến dịch **đang chạy nhưng hỏng không cấp được gói**.

| Trường | Kiểu | Nguồn | Ý nghĩa |
|---|---|---|---|
| `status` | xem máy trạng thái | Suy ra từ cấu hình + đếm lượt cấp | Vòng đời |
| `degraded` | boolean | `campaign_sweep_runs.degraded` | Hỏng |
| `degradedReason` | chuỗi | `campaign_sweep_runs.degraded_reason` | Chẩn đoán |
| `lastSweepAt` | thời điểm | `campaign_sweep_runs.last_sweep_at` | Sức khỏe định kỳ |

#### Bảng `subscription.campaign_sweep_runs`

```sql
CREATE TABLE IF NOT EXISTS subscription.campaign_sweep_runs (
    campaign_code   VARCHAR(64) PRIMARY KEY,
    last_sweep_at   TIMESTAMPTZ,
    degraded        BOOLEAN     NOT NULL DEFAULT FALSE,
    degraded_reason TEXT,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Worker quét bù ghi một dòng **cho mỗi chiến dịch** sau mỗi lần quét: `last_sweep_at` cập nhật
mỗi lần quét thành công; khi lần quét gặp lỗi thì `degraded = TRUE` và `degraded_reason` ghi lý
do. Một lần quét thành công xoá `degraded_reason`. API tra cứu đọc thẳng bảng này.

**Vì sao phải nằm trong CSDL chứ không giữ trong bộ nhớ.** Đặc tả cũ giữ cờ trong bộ nhớ worker và
biện minh rằng mất khi khởi động lại là hành vi đúng. Biện minh đó chỉ đúng khi có **một** worker.
Hệ thống này chạy **nhiều phiên bản, mỗi phiên bản một bộ worker riêng** (`spec.md` Bài toán 1), nên:

1. Yêu cầu HTTP rơi vào phiên bản B sẽ đọc cờ của phiên bản B, trong khi lần quét gây lỗi có thể
   do phiên bản A chạy. Cờ sai theo hướng **nguy hiểm nhất**: hỏng thật mà báo lành.
2. Với `APP_MODE=http` thì không có worker nào chạy, `degraded` luôn `false` — đúng cái trạng thái
   giả mà FR-059 sinh ra để chống.

Lưu bảng ghi cũng rẻ: worker quét bù vốn đã ghi vào `campaign_claims`, thêm một dòng theo mã chiến
dịch không thay đổi đặc tính lượt ghi. Đổi lại FR-059 trở thành kiểm chứng được trên hạ tầng thật.

---

## 6. Bảng Quyết Định Điều Kiện Đủ Điều Kiện

Đây là logic mà `admission_uc.go` phải cài đặt nguyên văn. Thứ tự ràng buộc là cố ý — ràng buộc rẻ
chạy trước, ràng buộc cần khoá chạy sau.

| # | Điều kiện | Nguồn dữ liệu | Lý do bỏ qua | Có cần khoá? |
|---|---|---|---|---|
| 1 | Có ít nhất một chiến dịch khai báo không | Cấu hình | `campaign_not_configured` | Không |
| 2 | Có chiến dịch ở trạng thái `running` hoặc `compensation` | Cấu hình + đồng hồ | `campaign_not_started` / `campaign_sleeping` | Không |
| 3 | Tài khoản **không** do quản trị viên tạo | Thông báo, hoặc cột `created_by_admin` khi quét bù | `created_by_admin` | Không |
| 4 | Tài khoản chưa có lượt cấp nào | `campaign_claims` | `already_claimed` | Không — ràng buộc `UNIQUE` lo |
| 5 | Tài khoản đang ở **đúng** gói đó và còn hạn | `user_subscriptions` | `already_on_plan` (FR-018) | Không |
| 6 | Cấm chồng: tài khoản vừa bị hạ về gói miễn phí mà gói cũ chưa dọn xong | `user_subscriptions` | `plan_not_settled` | Không |
| 6a | `plan_slug` của chiến dịch trỏ tới gói tồn tại và đang active | Cấu hình + `subscription_plans` | `plan_unavailable` | Không |
| 7 | Chọn chiến dịch **bắt đầu muộn nhất** mà tài khoản đủ điều kiện | Cấu hình | `outside_window` (FR-011, FR-017) | Không |
| 8 | Chốt hạn mức: còn suất theo vị trí của tài khoản | `campaign_claims` + khoá tư vấn | `quota_main` / `quota_compensation` / `quota_anomaly` | **Có** |
| 9 | Ghi lượt cấp | `campaign_claims` | vi phạm `UNIQUE` — bất thường, xem ghi chú | **Có** |

Bước 8 là bước duy nhất cần khoá, vì đó là bước duy nhất **đọc rồi quyết định rồi ghi**.

**Về điều kiện 6a.** Nạp cấu hình **không** được chạm cơ sở dữ liệu (hiến phán §I, §VIII), nên
không thể chặn khởi động khi `plan_slug` sai. Kiểm tra diễn ra ở nơi dòng gói vốn được đọc — tức lúc
chạy — và ghi lý do `plan_unavailable`. Hệ quả cần nói rõ: **một `plan_slug` sai không làm hỏng dịch
vụ, nó chỉ khiến chiến dịch đó không cấp được gì cho ai**, và người vận hành thấy ngay qua mã này.

**Về điều kiện 9.** Khi ràng buộc `UNIQUE` thực sự kích hoạt, đó không phải một lý do bỏ qua hợp lệ để
báo cho khách hàng mà là dấu hiệu dữ liệu bất thường hoặc đua thao tác — điều kiện 4 đã chặn sớm
bằng `already_claimed` trước khi tới đây. Vì vậy nó **không** có mã `reason` riêng: điều kiện này
được thiết kế để không xảy ra, và nếu xảy ra thì phải điều tra chứ không phải phục vụ trả lời.

### 6.1 Quy tắc chọn chiến dịch (bước 7)

```text
ứng viên = { c : c.still_running(now) ∧ account.created_at >= c.starts_at
                          ∧ (c.ends_at = ∅ ∨ account.created_at < c.ends_at) }
chiến dịch = ứng viên có starts_at LỚN NHẤT
```

Hai vế của điều kiện ứng viên dùng `account.created_at`, không dùng "bây giờ". Đó là điểm làm quy
tắc này thuần tuý: **cùng một tài khoản luôn được xếp vào cùng một chiến dịch, bất kể thông báo tới
trước hay sau** (FR-011). Nếu dùng "bây giờ", một thông báo tới chậm sẽ rơi ra ngoài cửa sổ và tài
khoản đó bị loại một cách vô lý.

### 6.2 Quy tắc chốt hạn mức (bước 8)

```text
trong transaction, sau khi đã giữ khoá tư vấn:
  mainCount     = COUNT(*) WHERE campaign_code = ? AND grant_bucket = 'main'
  reserveCount  = COUNT(*) WHERE campaign_code = ? AND grant_bucket = 'reserve'
  watermarkSet  = EXISTS(WHERE campaign_code = ? AND fills_main_quota)
  watermarkAt   = registered_at của dòng fills_main_quota = TRUE

  nếu mainCount < quota:
      → cấp, grant_bucket = 'main'
      → nếu mainCount + 1 == quota VÀ watermarkSet = FALSE:
            đây chính là lượt lấp đầy, đặt fills_main_quota = TRUE
  ngược lại nếu reserveCount < reserve:
      → chỉ cấp nếu watermarkAt CÓ giá trị VÀ account.created_at < watermarkAt
      → cấp, grant_bucket = 'reserve', fills_main_quota = FALSE
ngược lại:
       → bỏ qua, lý do quota_compensation  (dự phòng là TRẦN, nên "hết dự phòng" CHÍNH LÀ chạm trần cứng)
```

**Vì sao phải đếm theo `grant_bucket` chứ không theo `NOT fills_main_quota`.** Cột `fills_main_quota`
chỉ đánh dấu **một** dòng — lượt lấp đầy — nên `COUNT(*) WHERE NOT fills_main_quota` bằng
`tổng số lượt cấp trừ 1`, tức là nó **cộng luôn cả phần dự phòng vào phần chính**. Hậu quả kép:

1. Lượt cấp kế tiếp tính `mainCount = 49`, rồi `49 + 1 == 50 == quota` lại đặt
   `fills_main_quota = TRUE`, đụng `ux_campaign_claims_watermark`. Lượt cấp dự phòng **không bao
   giờ dùng được**: nó lỗi vi phạm chỉ mục thay vì rơi sang phần dự phòng.
2. Số liệu tra cứu lệch: cấp 50 chính + 2 dự phòng thì báo 51 chính, 1 dự phòng.

Điều kiện `VÀ watermarkSet = FALSE` trong nhánh chính là chốt an toàn: nếu cột này bị sửa tay hoặc
một chiến dịch cũ được nạp lại với hạn mức khác, quy tắc không được ghi thêm mốc ưu tiên thứ hai và
làm hỏng chỉ mục.

Nhánh dự phòng kiểm tra `watermarkAt` **phải có giá trị** vì phần dự phòng theo định nghĩa chỉ dành
cho tài khoản tạo **trước** mốc ưu tiên (FR-031). Nếu phần chính đã đầy mà mốc chưa chốt thì dữ liệu
đang ở trạng thái bất thường — thay vì đoán, bỏ qua và ghi lý do `quota_anomaly` để tra ra được.

**Trần tổng được bảo đảm bởi cấu trúc, không cần so sánh tổng.** Nhánh chính chỉ chạy khi
`mainCount < quota` nên phần chính không vượt `quota`; nhánh dự phòng chỉ chạy khi
`reserveCount < reserve` nên phần dự phòng không vượt `reserve`. Tổng do đó ≤ `quota + reserve`
mà không cần một phép so sánh tổng nào — và phép so sánh tổng trong bản cũ vốn không đúng vì
`đãCấpBù` ở đó lại tính cả phần chính.

Hệ quả của FR-032: phần dự phòng là **trần**, không phải mục tiêu. Nếu chiến dịch kết thúc mà dự
phòng chưa dùng hết, tổng số lượt cấp **ít hơn** `quota + reserve`, và đó là hành vi đúng.

---

## 7. Thông Điệp Sự Kiện

### 7.1 `identity.user.created`

| Trường | Kiểu Go | Ghi chú |
|---|---|---|
| `userId` | `uuid.UUID` | |
| `provider` | `string` | `local` hoặc `google` |
| `registeredAt` | `time.Time` | `users.created_at` — nơi lưu mốc ưu tiên so sánh |
| `createdByAdmin` | `bool` | Chỉ lấy được từ đường gọi (R4) |

**Không** chứa: email, tên, số điện thoại, mật khẩu (FR-087). Bên nhận thông báo cần email thì đọc
từ cơ sở dữ liệu — cùng một dịch vụ nên không tốn gọi mạng.

Nhãn JSON: `camelCase` (quy ước chung của dự án cho dữ liệu thông điện, trừ khi hợp đồng cũ đã dùng
kiểu khác).

### 7.2 Bản ghi hàng đỗi sự kiện

Dùng chung bảng `shared.outbox_records`. Sự kiện mới tạo **một** dòng, topic
`identity.user.created`, trạng thái `pending`. Bộ phát hàng đỗi hiện có chuyển tiếp và đánh dấu đã
phát; không có đường phát trực tiếp nào được thêm (hiến phán §III).

---

## 8. Thực Thể Không Đổi Lược Đồ Nhưng Bị Đọc

Tài liệu ở đây vì bộ luật chiến dịch phụ thuộc vào chúng; **không** có thay đổi lược đồ nào ở đây.

### 8.1 `subscription.user_subscriptions`

Các trường bộ luật chiến dịch đọc: `current_plan_code`, `current_plan_kind`, `expires_at`,
`is_auto_renew_enabled`, `version`.

Hằng số cấu trúc phụ thuộc:
- Ràng buộc `chk_user_subscription_kind_expiry`: gói có thời hạn bắt buộc có `expires_at` khác rỗng.
  Đây là lý do cột `expires_at` của lượt cấp **không** cho phép rỗng.
- Ràng buộc `chk_user_subscription_auto_renew`: cờ tự động gia hạn bắt buộc tắt với gói không có thời
  hạn.

### 8.2 `subscription.subscription_plans`

Đọc `slug`, `name`, `duration_days`, `ai_cost_policy_id`, `is_active`. Cấu hình chiến dịch **tham
chiếu** gói cước, không sao chép bất kỳ thuộc tính nào. Thời hạn cấp lấy từ `duration_days`
(FR-019) — với `premium-monthly` hiện tại là 30 ngày.

### 8.3 `shared.outbox_records`

Bảng dùng chung, không sửa. Bộ phát sự kiện và bộ phát hàng đỗi hiện có đã xử lý đầy đủ.

---

## 9. Bảng Ánh Xạ Yêu Cầu → Dữ Liệu

Để đối chiếu khi review. Mỗi yêu cầu về dữ liệu phải có mặt ở đây.

| Yêu cầu | Cơ chế dữ liệu |
|---|---|
| FR-021, FR-022 | Cột `quota` + `reserve` trong cấu hình; đếm tách phần bằng cột `grant_bucket` trong transaction có khoá |
| FR-024 | `ux_campaign_claims_user` + khoá tư vấn |
| FR-025 | `ux_campaign_claims_user` |
| FR-028 | Bảng lượt cấp tồn tại qua các lần khởi động lại — không có đường nào xoá nó |
| FR-030, FR-031 | Cột `fills_main_quota` + `ux_campaign_claims_watermark` + cột `registered_at` chụp sẵn |
| FR-032 | Cột `grant_bucket` ghi phần dự phòng là **trần**; không có đường nào ghi `reserve` khi `reserve = 0` |
| FR-033 | Đếm `grant_bucket = 'main'` cho biết đã chạm trần phần chính, tách được khỏi "không đủ điều kiện" khi tra cứu |
| FR-050 | Cột `created_by_admin` |
| FR-059 | Bảng `campaign_sweep_runs` (`degraded`, `degraded_reason`, `last_sweep_at`) — vì sao không giữ trong bộ nhớ ở mục 5.1 |
| FR-065, FR-066 | Bảng lượt cấp + cấu hình |
| FR-087 | Thông điện chỉ có 4 trường, không có dữ liệu nhận diện cá nhân |
| FR-092 | Không có thay đổi lược đồ nào ở tập loại sự kiện |
