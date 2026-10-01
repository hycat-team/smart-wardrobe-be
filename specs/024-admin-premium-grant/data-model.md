# Mô hình dữ liệu — 024

**Ngày**: 2026-09-29 | **Đầu vào**: [spec.md](./spec.md) | **Quyết định**: [research.md](./research.md)

Có hai bảng ở hai schema khác nhau liên quan trực tiếp tới tính năng này. Ranh giới sở hữu là điểm cốt lõi của cả spec.

| Bảng | Schema | Chủ sở hữu ghi | Module Go sở hữu entity |
|---|---|---|---|
| `user_subscriptions` | `subscription` | Account | `subscription` |
| `user_subscription_events` | `subscription` | Account | `subscription` |
| `subscription_plans` | `subscription` | Account | `subscription` |
| `ai_policy_grants` | `aiexecutor` | **AI (duy nhất)** | `aiexecutor` |
| `ai_cost_policies` | `aiexecutor` | **AI (duy nhất)** | `aiexecutor` |
| `ai_cost_policy_operations` | `aiexecutor` | **AI (duy nhất)** | `aiexecutor` |
| `ai_usage_events` | `aiexecutor` | AI | `aiexecutor` |
| `ai_usage_period_ledgers` | `aiexecutor` | AI | `aiexecutor` |

Lưu ý: `ai_policy_grants` và `ai_cost_policies` có DDL trong cây migration của **Account** (`services/account/database/migrations/`), dịch vụ AI chưa có cây migration. Đây là nợ kỹ thuật đã biết, ghi ở [research.md](./research.md) R6.

---

## 1. AI Policy Grant — thực thể trung tâm

**Bảng**: `aiexecutor.ai_policy_grants` · **Entity**: `aiexecutor_entities.AIPolicyGrant` (`domain/entities/policy.go:46-59`)

| Trường | Kiểu | Ràng buộc | Ghi bởi ai | Ghi chú |
|---|---|---|---|---|
| `id` | `uuid` | PK, mặc định `gen_random_uuid()` | AI | |
| `user_id` | `uuid` | `NOT NULL`, FK → `identity.users(id)` `ON DELETE CASCADE` | AI | |
| `policy_id` | `uuid` | `NOT NULL`, FK → `ai_cost_policies(id)` `ON DELETE RESTRICT` | AI | `aiCostPolicyID` trong tín hiệu |
| `plan_id` | `uuid` | `NOT NULL`, **không còn FK** | AI | FK đã bị cắt ở `20260801000000` → phía phát phải gửi đúng |
| `plan_code` | `varchar(100)` | `NOT NULL` | AI | `toPlanCode` trong tín hiệu |
| `tier_rank` | `int` | `NOT NULL` | AI | `toTierRank` trong tín hiệu |
| `policy_snapshot` | `jsonb` | `NOT NULL` | AI | **Hai phần** `envelope` + `entitlements` (FR-029). Không cần DDL — JSONB không có lược đồ |
| `effective_from` | `timestamptz` | `NOT NULL` | AI | xem bảng tính mốc bên dưới |
| `effective_to` | `timestamptz` | `CHECK (effective_to IS NULL OR effective_to > effective_from)` | AI | `NULL` = không thời hạn (gói trọn đời, gói miễn phí) |
| `status` | `varchar(20)` | `NOT NULL`, `CHECK IN ('active','future','closed')` | AI | xem bảng trạng thái |
| `source_event_id` | `uuid` | nullable, **không còn FK** | AI | **khoá chống xử lý trùng** (FR-033); cần chỉ mục (R6) |
| `source_deposit_transaction_id` | `uuid` | nullable, không còn FK | AI | đối soát tài chính |

**Chỉ mục**: `ix_user_ai_policy_grants_active (user_id, effective_from, effective_to)` — không duy nhất. Thêm `source_event_id` ở GĐ3.

### Tính mốc bắt đầu — quy tắc cốt lõi của GĐ4

Đây là quy tắc nghiệp vụ đang nằm sai chỗ: nó đang ở `projectAIPolicyGrant` thuộc hệ thống Account, và phải được chuyển nguyên vẹn sang AI trước khi gỡ (FR-039).

| Loại thay đổi | `effective_from` | `status` | `effective_to` | Xử lý bản ghi cũ |
|---|---|---|---|---|
| `ACTIVATED`, `UPGRADED`, `OVERLAID`, `LIFETIME_FALLBACK_RESTORED` | `effectiveAt` của tín hiệu | `active` | `effectiveTo` của tín hiệu (gói không thời hạn → `NULL`) | đóng bản ghi `active`, **xoá** bản ghi `future` |
| `EXTENDED`, kỳ mới **đã bắt đầu** | `effectiveAt` của tín hiệu | `active` | `effectiveTo` mới | cắt đuôi bản ghi cũ về đúng `effectiveFrom` mới |
| `EXTENDED`, kỳ mới **chưa bắt đầu** | `effectiveTo` mới **trừ** `durationDays` | `future` | `effectiveTo` mới | cắt đuôi bản ghi cũ về đúng mốc bắt đầu kia |
| `SUBSCRIPTION_DOWNGRADED_TO_FREE` | — | — | — | đóng mọi bản ghi `active`, **xoá** mọi bản ghi `future` |
| `AUTO_RENEW_UPDATED` | — | — | — | **không làm gì** |

Quy tắc cốt lõi của dòng `EXTENDED`: gia hạn áp dụng sớm hay muộn đều phải cấp quyền đúng cho **đoạn lịch vừa mua**, nên mốc bắt đầu là `expiresAtMới - durationDays`, không phải "bây giờ". Nếu mốc đó vẫn còn ở tương lai so với `occurredAt` thì bản ghi ở trạng thái `future`. Nhờ vậy tài khoản không bao giờ rơi vào khoảng trống giữa kỳ cũ và kỳ mới — đây là US2 kịch bản 5.

Quy tắc này cần `durationDays` và `effectiveTo` trong tín hiệu, cả hai đều là trường mới (FR-023).

### Vòng đời trạng thái

```text
                  ┌──────────────────────────────────────┐
                  │                                      │
   tín hiệu ──────┤                                      │
   (EXTENDED,     ▼                                      │
    kỳ mới chưa   future ──(đến effective_from)──► active │
    bắt đầu)      │                                      │
                  │                                      ▼
                  └───────────────────────────────►  closed
                        (bản ghi mới được tạo)      (vĩnh viễn, xoá ở GĐ3b)
```

Chuyển trạng thái chỉ xảy ra ở **bộ xử lý tín hiệu** (FR-043). Đường đọc chỉ trả kết quả (FR-042).

### Bất biến

| # | Bất biến | Cơ chế bảo vệ |
|---|---|---|
| I1 | Mỗi tài khoản có **nhiều nhất** một bản ghi `active` | FR-031 — đóng bản ghi cũ rồi tạo bản ghi mới trong **cùng một** giao dịch |
| I2 | `policy_snapshot` không bao giờ rỗng và luôn có đủ phần `entitlements` cho cả bốn thao tác | FR-030; hiện **8/16** dòng rỗng và 8 dòng còn lại thiếu hẳn nhóm quyền lợi → GĐ3b dựng lại toàn bộ |
| I6 | Giới hạn quyền lợi lấy từ ảnh chụp, trần chi phí lấy từ cấu hình sống | FR-029b. Đây là bất biến phân tách, quyết định bởi người phát triển 29/09/2026 |
| I3 | Không có hai bản ghi cùng `source_event_id` | FR-033 — bỏ qua nếu đã có; chỉ mục chống quét toàn bảng (R6) |
| I4 | Tối đa một bản ghi `active` **đang thực sự hiệu lực** | SC-007; hiện đã đạt (0 vi phạm) |
| I5 | Bản ghi `closed` không bao giờ được mở lại | Xoá vĩnh viễn ở GĐ3b |

**I1 chỉ an toàn khi có giao dịch.** `grant_uc.go:47-51` hiện gọi `expireCurrentGrants` rồi `createGrant` mà không có `uow.Execute` — sập giữa chừng là tài khoản mất sạch mức quyền tính phí, tức mất toàn bộ AI trả phí. `GrantUseCase` phải nhận `IUnitOfWork`.

### Cấu trúc `policy_snapshot` và ranh giới ảnh chụp / cấu hình sống

Cột này ghi hai phần. Điểm cốt lõi: **ảnh chụp quyết định cái khách hàng đã mua, cấu hình sống quyết định cái hệ thống chịu rủi ro** (FR-029b).

```json
{
  "envelope": {
    "eventVersion": 2, "eventType": "EXTENDED",
    "occurredAt": "...", "effectiveAt": "...", "effectiveTo": "...",
    "durationDays": 30, "sourceEventId": "..."
  },
  "entitlements": {
    "chat":    { "maxPaidAttemptsPerDay": 10, "normalMaxInputTokens": 4000, "normalMaxOutputTokens": 600, "reducedMaxInputTokens": 3500, "reducedMaxOutputTokens": 500 },
    "outfit":  { "maxPaidAttemptsPerDay": 30, "normalMaxInputTokens": 6000, "normalMaxOutputTokens": 400, "reducedMaxInputTokens": 5000, "reducedMaxOutputTokens": 350 },
    "summary": { "maxPaidAttemptsPerDay": null, "normalMaxInputTokens": 3000, "normalMaxOutputTokens": 250, "reducedMaxInputTokens": 3000, "reducedMaxOutputTokens": 250 },
    "rewriter":{ "maxPaidAttemptsPerDay": null, "normalMaxInputTokens": 1000, "normalMaxOutputTokens": 250, "reducedMaxInputTokens": 1000, "reducedMaxOutputTokens": 250 }
  }
}
```

Ví dụ trên là ảnh chụp của **gói Premium** ở chế độ `strict`. Ở chế độ này `maxPaidAttemptsPerDay` **không bao giờ được đọc** (FR-005e) — nó chỉ tồn tại để bảo toàn hình dạng dữ liệu, nên giá trị `null` là cách nói dứt khoát "chính sách này không dùng giới hạn số lượt". Hai giá trị `summary` và `rewriter` còn thêm một lý do nữa là không có tác dụng: tuyến thực thi của chúng là `free`.

Bốn thao tác đều phải có mặt trong ảnh chụp: thiếu bất kỳ thao tác nào thì bộ đọc ở T066 coi như không đạt. Ảnh chụp của **gói miễn phí** ở chế độ `observe_only` thì ngược lại — cả bốn thao tác đều có giá trị số, và đây là nơi con số đó thực sự thi hành.

`envelope` phục vụ hai việc: chốt chặn tín hiệu đến sai thứ tự (FR-033 — so `occurredAt` với bản ghi đang hiệu lực) và truy vết nguồn. `entitlements` chỉ chứa **giới hạn**, không chứa `is_enabled` — vì tắt thao tác là công cụ chặn chi phí, không phải thứ khách mua.

| Nhóm trường | Đọc từ đâu | Vì sao |
|---|---|---|
| `max_paid_attempts_per_day`, `normal_max_input_tokens`, `normal_max_output_tokens`, `reduced_*` | **Ảnh chụp** `entitlements` | Đây là cái khách mua. Cấu hình trôi không được lấy đi |
| `normal_route`, `reduced_route`, `free_route` | **Sống** | Quyết định định tuyến là vận hành. Lượt trả phí đã đóng băng nên đổi tuyến chỉ làm **giảm** chi phí |
| `hard_cost_micro_vnd`, `enforcement_mode`, `free_route_threshold_bps`, `max_unknown_paid_requests_per_day`, `unknown_hold_minutes`, `period_days`, `is_active`, `is_enabled` | **Sống** | Chống lỗ tài chính và chống lạm dụng. Vận hành phải siết được ngay lập tức với mọi tài khoản |

**Vì sao trần chi phí tuyệt đối không được đóng băng**: 5 tài khoản gói miễn phí có bản ghi `effective_to IS NULL`. Bản ghi đó chỉ được tạo lại khi tài khoản đổi gói. Nếu trần nằm trong ảnh chụp, việc hạ trần **không có tác dụng lên họ** — có thể vĩnh viễn.

**Đánh đổi được chấp nhận**: đổi trường quyền lợi không tự áp dụng cho bản ghi cũ; muốn đẩu xuống tất cả thì chạy công cụ phát lại tín hiệu ở GĐ8 (FR-088, FR-029c). Đây đúng là mẫu "giữ giá cũ mặc định, đẩy mới là hành động tường minh" mà ngành dùng. Trường chống lỗ thì không cần bước này.

### Tình trạng ảnh chụp hiện có — và vì sao phải dựng lại toàn bộ

Đã đọc trực tiếp 8 ảnh chụp không rỗng trên dữ liệu thật. Chúng **không dùng được** với bộ đọc mới:

| Vấn đề | Chi tiết |
|---|---|
| Thiếu hẳn nhóm quyền lợi | Không có `maxPaidAttemptsPerDay`, `normalRoute`, `normalMaxInputTokens` — vì bản ghi gốc chỉ chụp cấp hình ở cấp chính sách, bỏ qua bảng `ai_cost_policy_operations` |
| Trộn hai kiểu ghi | Vừa `Code`/`Version`/`HardCostMicroVND` (kiểu Go) vừa `code`/`version`/`hardCostMicroVnd` (camelCase). 25 khoá trong cùng tập dữ liệu → có **hai** nơi từng ghi bằng cách khác nhau |
| Lệch với cấu hình sống | `maxUnknownPaidRequestsPerDay: 0` trong khi cột thật là `2`; `mode: "OBSERVE_ONLY"` viết hoa trong khi hằng số Go là `observe_only` |

Nếu từng có ai đọc ảnh chụp này để thi hành, chống lạm dụng của tài khoản gói miễn phí sẽ bị kẹt ở 0. Việc mã hiện tại đọc cấu hình sống hóa ra **may mắn**, không phải cố ý.

Hệ quả cho GĐ3b: bước dựng lại phải ghi đè **mọi** bản ghi, không chỉ 8 bản ghi rỗng. Đây là lần chốt giá một lần — ảnh chụp của bản ghi sống lấy theo cấu hình lúc chạy tệp kịch bản, không phải lúc tạo bản ghi, vì dữ liệu tháng 6 không lưu được lịch sử giá. Chấp nhận được vì 7 bản ghi còn lại đều là dữ liệu kiểm thử (FR-092b).

| | Trước | Sau |
|---|---|---|
| Ghi ảnh chụp vào | 8 dòng (`'{}'`) | **16 dòng** (tất cả) |
| Xoá bản ghi chết | 7 dòng | 7 dòng (không đổi) |
| Còn lại | 9 dòng | 7 dòng, tất cả đúng hình dạng |

**Không cần migration DDL.** `policy_snapshot` đã là `jsonb NOT NULL` không có lược đồ; phần đọc sống dùng đúng các cột đã có ở `ai_cost_policies` và `ai_cost_policy_operations`. Toàn bộ việc là mã Go đọc, cộng một lần chạy tệp dọn tay ở GĐ3b.

### Trạng thái dữ liệu hiện tại và việc cần làm ở GĐ3b

| Nhóm | Số dòng | Xử lý |
|---|---|---|
| `active`, thời hạn không xác định, ảnh chụp đầy đủ | 5 | **Giữ nguyên** — sinh từ migration tạo dữ liệu ban đầu, hợp lệ |
| `active`, đã hết thời hạn (`2026-09-11 → 2026-09-22`), ảnh chụp rỗng | 1 | Đóng (FR-092 bước 3) |
| `future`, `effective_from` đã nằm trong quá khứ, ảnh chụp rỗng | 1 | Đóng (FR-092 bước 3) |
| `closed`, đã hết hiệu lực, ảnh chụp rỗng | 6 | Điền ảnh chụp rồi xoá (FR-092 bước 1 rồi bước 2) |
| `closed`, đã hết hiệu lực, ảnh chụp đầy đủ | 1 | Xoá (FR-092 bước 2) |

Thứ tự ba bước trong FR-092 là cố ý: điền ảnh chụp trước để bản ghi còn sống thì giữ được thông tin; xoá bản ghi chết sau; đóng bản ghi hết hạn sai trạng thái ở cuối. Không có bản ghi nào thực sự còn hiệu lực nào bị xoá.

---

## 1b. Hai cổng hạn mức AI — quan hệ giữa chúng

Hệ thống có **hai cổng hạn mức độc lập cho cùng một thao tác AI**, ở hai schema và hai dịch vụ khác nhau. Đây là quan hệ dữ liệu dễ hiểu sai nhất của cả hệ thống.

```
subscription_plans                          aiexecutor.ai_cost_policy_operations
  ai_outfit_daily_quota   ┐                    max_paid_attempts_per_day  (outfit)
  ai_chat_daily_quota     │  KHÔNG có khoá    ────────────────────────────────┐
       │                  │  ngoại liên kết        │                          │
       ▼                  │                       ▼                          │
user_daily_quotas         │              ai_usage_events (đếm paid)            │
  outfit_recommend_count  │                       │                          │
  ai_usage_count          │                       │                          │
       │                  │                       │                          │
       ▼                  └───────────────────────┴──────────────────────────┘
thi hành ở Commerce                        thi hành ở hệ thống AI
  Vượt → CHẶN yêu cầu, trả lỗi              Vượt → HẠ mô hình miễn phí, vẫn chạy
  Người dùng NHÌN THẤY                      Người dùng KHÔNG biết
```

| | Cổng 1 | Cổng 2 |
|---|---|---|
| Bảng cấu hình | `subscription_plans` | `ai_cost_policy_operations` |
| Bảng đếm | `user_daily_quotas` | `ai_usage_events` |
| Dịch vụ thi hành | Commerce | AI |
| Hành vi khi vượt | chặn, trả lỗi | hạ xuống mô hình miễn phí |
| Người dùng thấy? | **có** — hồ sơ và màn hình hạn mức | **không bao giờ** — cơ chế nội bộ, cấm hiển thị bất kỳ giá trị nào cho người dùng (FR-005c) |
| Cách sửa | `PUT /admin/subscriptions/plans/{id}` | route HTTP trong nhóm `/admin/ai` của **hệ thống AI**, dành cho quản trị viên (FR-004b) |
| Quan hệ với ảnh chụp | không liên quan | thuộc nhóm **quyền lợi** → đóng băng trong `entitlements` (FR-029b) |
| **Còn thi hành ở chế độ nào?** | **luôn** — không phụ thuộc chế độ | **chỉ `observe_only`**. Ở `strict` trường `max_paid_attempts_per_day` **không được đọc**; khi đó cột này là cấu hình chết (FR-005e) |

> **Dòng cuối là dòng quan trọng nhất của bảng và dễ bỏ sót nhất.** Ở chế độ `strict` (tức là mọi chính sách có trần chi phí, gồm cả Premium) cổng 2 **không hề giới hạn số lượt/ngày** — nó chỉ so tổng tiền trong sổ tính phí với trần. Đọc bảng này mà bỏ qua dòng cuối sẽ dẫn tới kết luận sai rằng AI giới hạn số lượt/ngày cho gói Premium.



### Danh mục chuẩn: plan slug và policy của cổng 2
Đọc trực tiếp từ DB ngày 29/09/2026 (`subscription.subscription_plans`, `aiexecutor.ai_cost_policies`). **Hệ thống chỉ có hai plan**, đã xác nhận bằng truy vấn `tier_rank > 1 OR plan_kind > 1` trả về **0 dòng**.
| `PlanSlug` (cổng 1) | `ai_cost_policy_id` dẫn tới `code` (cổng 2) | `enforcement_mode` | Quan hệ |
|---|---|---|---|
| `free` | `free-default` | `observe_only` | khóa ngoại `subscription_plans.ai_cost_policy_id` |
| `premium-monthly` | `premium-default` | `strict` | khóa ngoại `subscription_plans.ai_cost_policy_id` |
> **Phân biệt mã slug với mã policy - dễ nhầm nhất.** `premium-monthly` và `free` là **plan slug** ở cổng 1. `premium-default` và `free-default` là **mã chính sách** ở cổng 2. Hai cặp này **không trùng nhau** và **không thay thế cho nhau**. Không được viết `premium` hay `premium-monthly` vào cột `code` của bảng chính sách, và không được viết `premium-default` vào trường `planSlug` của yêu cầu cấp gói.
Cột `subscription_plans.ai_cost_policy_id` **đã tồn tại** - hai cổng đã được nối bằng khóa ngoại, nên FR-004 phải có cách đối chiếu hai cổng thì chỉ cần đọc theo quan hệ này.
### Về gói lifetime - hiện chưa tồn tại
DB không có plan `lifetime`, cũng không có `trial` hay `overlay` như plan độc lập. Vì vậy:
- **Cổng 2 của `lifetime` áp dụng y hệt Premium** - dùng chung chính sách `premium-default` (`strict`, trần chi phí chu kỳ 30 ngày, `max_paid_attempts_per_day = NULL`). Không cần tạo chính sách riêng: trần `30.000đ` là **chu kỳ 30 ngày**, mà `lifetime` mua một lần vẫn nằm trong chu kỳ đó, nên về cơ chế cổng 2 hai gói này giống hệt nhau.
- **Cổng 1 của `lifetime` chưa có giá trị và chưa được chốt.** Đây là cam kết bán ra cho khách nên phải do người phát triển quyết định, không tự đặt. Khi tạo plan `lifetime`, bắt buộc chốt đồng thời `ai_outfit_daily_quota` và `ai_chat_daily_quota`.
- FR-016 và nhánh `LIFETIME_FALLBACK_RESTORED` trong spec là **dự phòng cho tương lai**, dùng được ngay khi plan `lifetime` ra đời, không phải mô tả trạng thái hiện tại.
Giá trị đang chạy với plan slug `premium-monthly`: cổng 1 là outfit **15** / chat **20**; cổng 2 là outfit **30** / chat **10**.

Mục tiêu đã chốt (29/09/2026) cho plan slug `premium-monthly`: cổng 1 thành **30 outfit / 10 chat**. Cổng 2 **không cần đặt theo mục tiêu** vì chế độ `strict` không đọc trường này - nên được đặt thành `NULL` (xem hai dòng dưới).

Plan slug `free`: cổng 1 là 5 outfit / 3 chat, cổng 2 đang là 3 / 3. Chat khớp; **outfit lệch** - cổng 2 chặn ở 3 trong khi người dùng thấy 5. Đây là lệch lệch thật, không phải chuyển thuế, nhánh `observe_only` vẫn đếm lượt trả phí trong ngày rồi chuyển sang mô hình miễn phí khi vượt (`ai_cost_repo_impl.go:90-96`, `cost/usecase.go:124-125`). Người phát triển đã chốt: miễn phí **5 outfit / 3 chat, không trần cứng**, và đặt cổng 2 `outfit` = **`4`** - lệch một lượt lệ **cố ý**, nên bắt buộc phải báo trước cho người dùng (FR-005d).

Cột `max_paid_attempts_per_day` của `premium-default` được đặt thành **`NULL`** vì chế độ `strict` không đọc nó (FR-005e). Vì vậy cột phải cho phép NULL, và `NULL` ở chế độ `observe_only` là **cấu hình sai** — phải ghi cảnh báo và áp tầng suy giảm 2 chứ không âm thầm bỏ qua.

Với mục tiêu **Premium** đó, ngân sách chu kỳ cạn sau **12–15 ngày** — số lượt theo ngày là **tốc độ** chứ không phải hạn mứn cam kết. Xem FR-005c. Gói miễn phí không có ngân sách nên câu này không áp dụng cho nó. Với mục tiêu 4 outfit + 3 chat đã chốt, một tài khoản dùng hết lượng tốn khoảng **11.600đ mỗi tháng** (4 × 58đ + 3 × 52đ, × 30 ngày) — tăng so với mức 7.900đ tính trên 3/3, vẫn nằm trong mức đã được chấp nhận. Đây là lý do gói miễn phí **không** đặt trần cứng: `hard_cost_micro_vnd` của nó là `NULL`.

**Vì sao không có khoá ngoại liên kết**: hai bảng ở hai schema, thuộc hai chủ sở hữu khác nhau. Hiến pháp §I cấm nối cơ sở dữ liệu xuyên dịch vụ, nên hai cổng **không thể** tự đồng bộ. Hệ quả trực tiếp: cổng 1 có thể là 20 trong khi cổng 2 là 10, và không có gì báo lỗi. FR-004 buộc phải có cách đối chiếu hai cổng, FR-005b buộc cổng chặn phải là cổng người dùng nhìn thấy.

**Quan hệ với `ai_policy_grants`**: cổng 2 đi qua bản ghi mức quyền tính phí (đọc `entitlements.maxPaidAttemptsPerDay` trong ảnh chụp theo FR-029b), cổng 1 đi qua hợp đồng gói của hệ thống Account và **không** đi qua bản ghi mức quyền tính phí. Vì vận hành sửa cổng 2 phải phát lại tín hiệu (FR-029c), còn sửa cổng 1 thì không cần.

---

## 2. AI Cost Policy — cấu hình mà AI sở hữu

**Bảng**: `aiexecutor.ai_cost_policies` · **Entity**: `AICostPolicy`

| Trường | Kiểu | Ghi chú |
|---|---|---|
| `id` | `uuid` | PK |
| `code` | `varchar` | `NOT NULL`. Giá trị đang có: `free-default`, `premium-default`. **Khoá tìm chính sách mặc định** (R8) |
| `version` | `bigint` | `NOT NULL` |
| `name` | `varchar` | `NOT NULL` |
| `enforcement_mode` | `varchar` | `observe_only` \| `strict` \| `free_only`. `free_only` **chưa** được xử lý trong `Reserve` |
| `period_days` | `int` | `NOT NULL` = 30 |
| `hard_cost_micro_vnd` | `bigint` | nullable. `NULL` = không trần |
| `compact_threshold_bps` | `int` | `NOT NULL` = 8000. **Trường chết** — không nơi nào đọc |
| `free_route_threshold_bps` | `int` | `NOT NULL` = 9200 |
| `unknown_hold_minutes` | `int` | `NOT NULL` = 1440 |
| `max_unknown_paid_requests_per_day` | `int` | `NOT NULL` = 2 |
| `is_active` | `bool` | `NOT NULL` = true. **Trường chết** — không nơi nào đọc; sẽ dùng làm cổng cho tầng 1 của FR-007 |

Không có `tier_rank`, nên **không** tự suy ra được "chính sách mặc định là cái nào". Vì vậy R8 chọn tìm theo `code` lấy từ cấu hình `ai.default_cost_policy_code`.

**Bản ghi con**: `ai_cost_policy_operations` — `operation ∈ {chat, outfit, summary, rewriter}` (ràng buộc `CHECK`; không có `vision`/`embedding`, đúng vì hai thao tác đó bỏ qua mức quyền tính phí).

### Cấu hình đang áp dụng và mục tiêu GĐ0

| | `free-default` | `premium-default` |
|---|---|---|
| `enforcement_mode` | `observe_only` | `strict` |
| `hard_cost_micro_vnd` | `NULL` | 25.000.000.000 → **30.000.000.000** |
| `period_days` | 30 | 30 |
| `normal_route` (chat / outfit) | **paid** | **paid** |
| Token chat | 3000 / 500 | 4000 / 600 |
| Token outfit | 5000 / 300 | 6000 / 400 |
| Lượt trả phí mỗi ngày | chat 3, outfit 3 | chat 10, outfit 30 |

`normal_route = paid` ở gói miễn phí là lý do R8 phải **giữ sổ tính phí** thay vì bỏ qua: suy giảm mà không qua bước đặt trước là rò tiền không kiểm soát. Vì vậy tầng 1 vẫn ghi dòng sổ tính phí `FOR UPDATE`, chỉ là dòng đó có `grant_id = NULL` vì tài khoản chưa có bản ghi mức quyền tính phí. Xem research.md R8.

---

## 2b. Nhật ký chỉnh sách cổng 2 — ai sửa, sửa cái gì

**Bảng**: `aiexecutor.ai_cost_policy_audit_logs` · **Entity**: `AICostPolicyAuditLog` (tệp `domain/entities/ai_cost_policy_audit_log.go`, tạo ở T011) · **Bảng tạo ở** T020

Đây là bảng **duy nhất** trong tính năng này ghi lại *ai đã sửa cấu hình*, tách biệt với `user_subscription_events` (ghi *gói đổi thế nào*) và với `ai_policy_grants` (ghi *tài khoản được bao nhiêu quyền*).

| Trường | Kiểu | Ghi chú |
|---|---|---|
| `id` | `uuid` | PK |
| `actor_user_id` | `uuid` | quản trị viên đã thao tác, lấy từ `contextutils.GetUserId(c)` |
| `changed_at` | `timestamptz` | thời điểm thay đổi |
| `request_ip` | `varchar` | `c.ClientIP()`, phục vụ điều tra |
| `policy_code` | `varchar` | `free-default` hoặc `premium-default` |
| `operation` | `varchar` | `chat`, `outfit`, `summary`, `rewriter` |
| `field_name` | `varchar` | hiện chỉ có `maxPaidAttemptsPerDay` |
| `old_value` | `varchar` nullable | vắng mặt = trước đó không có giá trị, **không** phải giá trị rỗng |
| `new_value` | `varchar` nullable | vắng mặt = đã đặt thành `NULL` |
| `action` | `varchar` | theo danh mục `adminaudit.AuditAction`, hiện dùng `ai.cost_policy_update` |

Chỉ mục: theo `actor_user_id` (tra cứu lịch sử theo quản trị viên) và theo `changed_at` (sắp theo thời gian).

**Vì sao hai cột value phải nullable**: một thay đổi có thể đưa một trường từ *chưa có giá trị* sang *có giá trị*, hoặc từ *có* sang `NULL`. Nếu dùng chuỗi rỗng cho cả hai trường hợp thì mất thông tin không thể phân biệt về sau — nhật ký kiểm toán mà không phân biệt được "xóa" với "đặt rỗng" thì vô nghĩa.

**Thuộc sở hữu của AI, tuyệt đối không dùng `shared.admin_audit_logs`**: bảng đó thuộc mô-đun `admin` của Account. Ghi vào nó từ dịch vụ AI là vi phạm §I, đúng loại vi phạm mà tính năng này đang xoá (xem `ai_policy_grants` ở §1). Muốn bản ghi này xuất hiện ở màn hình kiểm toán tập trung của Account thì phải đi qua outbox — xem §5.

---

## 3. User Subscription — gói hiện tại

**Bảng**: `subscription.user_subscriptions` · **Entity**: `UserSubscription` (`domain/entities/user_subscription.go:13-33`)

| Trường | Kiểu | Ràng buộc | Ghi chú |
|---|---|---|---|
| `user_id` | `uuid` | **PK** | không có khoá ngoại tới `users` |
| `subscription_plan_id` | `uuid` | `NOT NULL` | |
| `current_plan_code` | `varchar` | `NOT NULL` | dùng cho đường đọc không cần nối bảng |
| `current_tier_rank` | `int` | `NOT NULL` | |
| `current_plan_kind` | `int` | `NOT NULL` | `0` = free, `1` = hữu hạn, `2` = trọn đời |
| `current_benefit_snapshot` | `jsonb` | `NOT NULL` | FR-071 — chụp quyền lợi tại thời điểm cấp |
| `started_at` | `timestamptz` | `NOT NULL` | |
| `expires_at` | `timestamptz` | nullable | `NULL` = không thời hạn |
| `fallback_plan_id` / `_code` / `_tier_rank` / `_kind` / `_snapshot` | | nullable | dùng cho gói trọn đời bị gói hữu hạn phủ |
| `is_auto_renew_enabled` | `bool` | `NOT NULL` | |
| `last_deposit_transaction_id` | `uuid` | nullable | |
| **`version`** | **`bigint`** | `NOT NULL` | **chống lạc phiên bản** (R10, FR-069) |

`current_plan_code` tồn tại chính là để tránh nối bảng khi đọc — nhưng AI thì **không** được đọc bảng này, nên tín hiệu vẫn phải mang `toPlanCode`.

### Quy tắc tính thời hạn — dùng chung cho mọi nguồn

Đã tồn tại trong `internal_subscription_apply_workflow.go` và FR-072 buộc dùng chung:

| Tình huống | `expires_at` mới |
|---|---|
| Gói không thời hạn (`duration_days IS NULL`) | `NULL` |
| Chưa có bản ghi gói | `now + durationDays` |
| Đang đúng gói đó, thời hạn còn hiệu lực | `expires_at_cũ + durationDays` — **cộng dồn** |
| Gói khác, hoặc thời hạn đã hết | `now + durationDays` |

Cột `version` tăng 1 ở mọi lần áp dụng — đây là mấu chốt của R10.

---

## 4. User Subscription Event — nguồn phát tín hiệu duy nhất

**Bảng**: `subscription.user_subscription_events` · **Entity**: `UserSubscriptionEvent` (`domain/entities/subscription_events.go:11-25`)

| Trường | Kiểu | Ràng buộc | Ánh xạ sang tín hiệu |
|---|---|---|---|
| `id` | `uuid` | PK, `gen_random_uuid()` | `sourceEventId` — khoá chống trùng (FR-033) |
| `event_key` | `varchar(255)` | `NOT NULL` **`UNIQUE`** | không gửi — khoá bất biến nội bộ (R11) |
| `user_id` | `uuid` | `NOT NULL`, FK → `users(id)` `ON DELETE CASCADE` | `userId` |
| `event_type` | `varchar(100)` | `NOT NULL` | `eventType` (FR-019 — lấy từ đây, không viết tay) |
| `from_plan_code` / `from_tier_rank` | | nullable | không gửi |
| `to_plan_code` / `to_tier_rank` | | nullable | `toPlanCode`, `toTierRank` |
| `source_deposit_transaction_id` | `uuid` | nullable, FK → `deposit_transactions(id)` | `sourceDepositTransactionId` |
| `actor_admin_id` | `uuid` | nullable, FK → `users(id)` | không gửi. **Hiện chưa nơi nào gán** → R11 gán ở luồng cấp gói |
| `occurred_at` | `timestamptz` | `NOT NULL` | `occurredAt` |
| `effective_at` | `timestamptz` | `NOT NULL` | `effectiveAt` |
| `metadata` | `jsonb` | nullable | không gửi |

`event_key` `UNIQUE` + `GetByKey` là cơ chế chống trùng sẵn có, đã dùng ở `webhook/purchase_workflow.go`.

**Bảng này không chứa `to_plan_id`.** Đó là lý do tín hiệu phải mở rộng: AI cần `toPlanId` để điền cột `plan_id` bắt buộc, mà bản ghi lịch sử không có trường đó. Sẽ lấy từ `subscription_plans.id` tại thời điểm phát tín hiệu.

### Tập loại thay đổi — 7 giá trị, đóng

Không có loại nào nằm ngoài danh sách này. Loại lạ → bộ xử lý từ chối kèm cảnh báo, không đoán.

| Loại | Nguồn | AI làm gì |
|---|---|---|
| `ACTIVATED` | Kích hoạt gói (PayOS, ví, quản trị viên) | Tạo grant |
| `UPGRADED` | Nâng lên gói bậc cao hơn | Tạo grant |
| `OVERLAID` | Gói trọn đời bị gói hữu hạn phủ | Tạo grant |
| `LIFETIME_FALLBACK_RESTORED` | Khôi phục gói trọn đời | Tạo grant |
| `EXTENDED` | Gia hạn (tự động hoặc thủ công) | Tạo grant, có thể `future` |
| `SUBSCRIPTION_DOWNGRADED_TO_FREE` | Hết hạn, quản trị viên thu hồi | Hạ cấp mọi grant |
| `AUTO_RENEW_UPDATED` | Bật/tắt tự gia hạn | **Không làm gì** |

`RENEWED` và `EXPIRED` — hai chuỗi phía AI đang xử lý — **không tồn tại** và sẽ bị gỡ. Chi tiết ở [research.md](./research.md) R1.

---
## 5. Outbox Record — bảo đảm tín hiệu không mất

**Bảng**: `outbox_records` · ghi qua `IEventPublisher.Publish`

Mỗi lần ghi bản ghi lịch sử cũng ghi một bản ghi outbox **trong cùng giao dịch**. Worker outbox phát lên RabbitMQ rồi đánh dấu đã phát.


| Thuộc tính | Giá trị |
|---|---|
| Chủ đề | `subscription.plan.changed` |
| Khoá định tuyến | `subscription.plan.changed` (trùng chủ đề) |
| Hàng đỗi | `subscription_plan_changed_queue` |
| Loại | classic → **đổi sang quorum**, `x-delivery-limit: 5` (R2) |
| Hàng chết | `subscription_plan_changed_queue.dlq` (quorum, đã tồn tại) |
| Tiền tố | 20 · Số luồng: 1 |
| Chính sách lỗi | `PermanentError` → hàng chết; `RetryableError` → giao lại; lỗi khác → hàng chết |

Cấu hình liên quan: `outbox.retry_max_attempts: 3`, `retry_interval_minutes: 5`, `batch_size: 50`.

### Bảng này phục vụ hai chủ đề, không chỉ một

| Chủ đề | Bản ghi được sinh ra khi | Hàng đỗi | Bên nhận |
|---|---|---|---|
| `subscription.plan.changed` | có bản ghi lịch sử đổi gói | `subscription_plan_changed_queue` | dịch vụ AI (tạo `ai_policy_grants`) |
| `admin.audit.log` | có bản ghi trong `ai_cost_policy_audit_logs` (§2b) | `audit_log_queue` | dịch vụ Account (ghi `shared.admin_audit_logs`) |

Hàng đỗi, khoá định tuyến, hàng chết và consumer của `admin.audit.log` **đã tồn tại sẵn** trong hạ tầng — xem `contracts/admin-cost-policy-api.md` §7.1. Không tạo mới.

**Cơ chế giữ hai bản ghi chung một giao dịch** (FR-004c): bọc cả hai lệnh ghi trong `IUnitOfWork.Execute`. Hàm này tiêm transaction GORM vào `ctx` qua `db.InjectTx`, nên `IEventPublisher.Publish` và `IAICostPolicyAuditRepository.Create` **tự động** ghi vào cùng transaction mà không cần truyền tham số `tx` riêng — cả hai đều đi qua `getDB(ctx)` → `db.GetTx(ctx)`.

Nếu không bọc, một trong hai bản ghi có thể nằm xuống còn bản kia thì không — tạo ra trạng thái nửa vời mà không có cách gỡ.

---

## 6. Thực thể dẫn xuất — kết quả cấp gói

Không có bảng. Đây là DTO trả về cho quản trị viên, mỗi phần tử mang đúng một trong ba trạng thái (FR-065).

| Trường | Ý nghĩa |
|---|---|
| `index` | Vị trí trong danh sách đã gửi |
| `userIdentifier` | Đúng chuỗi quản trị viên truyền vào (không phải giá trị đã phân giải) |
| `resolvedUserId` | Chỉ có khi nhận diện thành công |
| `planSlug` | Mã gói đã yêu cầu |
| `status` | `success` \| `skipped` \| `error` |
| `reason` | Mã lý do, chỉ khi không thành công |
| `message` | Thông báo tiếng Việt cho quản trị viên (FR-085) |
| `expiresAt` | Chỉ khi thành công — thời điểm hết hạn **mới** (FR-059) |

Tập mã lý do (mỗi mã ánh xạ tới một hàng trong `apperror` tiếng Việt):

| Mã | Nguồn yêu cầu |
|---|---|
| `user_not_found` | FR-057 |
| `identifier_ambiguous` | FR-048 — một chuỗi khớp nhiều tài khoản, tuyệt đối không tự chọn |
| `plan_not_found` | FR-054 |
| `plan_inactive` | FR-055 |
| `target_is_admin` | FR-056 |
| `user_soft_deleted` | FR-049 |
| `identifier_blank` | phần tử rỗng hoặc toàn khoảng trắng |
| `duplicate_in_request` | FR-066 — trùng trong cùng danh sách |
| `duplicate_request` | FR-068 — gửi lại cùng khoá bất biến |
| `concurrent_modification` | FR-069 — có thao tác khác vừa đổi gói (R10) |
| `identifier_too_long` | FR-062 - định danh dài hơn cột, từ chối trước khi truy vấn |
| `plan_slug_too_long` | FR-062 - mã gọi dài hơn cột, từ chối trước khi truy vấn |
| `grant_failed` | FR-059 - ghi bản ghi thất bại vì lỗi hạ tầng, **không** phải xung đột phiên bản (T129) |
| `batch_timeout` | FR-076 - hết ngân sách thời gian của yêu cầu (T143) |

Toàn bộ lỗi ở cấp **yêu cầu** (ngoài 500 phần tử, danh sách rỗng, vượt giới hạn tần suất) sẽ **từ chối cả yêu cầu** vì không thể xử lý từng phần tử. Đó là ranh giới đúng của FR-062/FR-063/FR-079.

---

## 7. Bảng chuyển đổi vòng đời đầy đủ

Một lần quản trị viên cấp gói, đi qua hai dịch vụ:

```text
  Quản trị viên
      │
      ▼  POST /api/v1/admin/subscription-grants   (danh sách 1..500)
 ┌────────────────────────────────────────────────────────────┐
 │ Account — MỘT giao dịch cho mỗi phần tử                  │
 │  1. nhận diện tài khoản  (UUID | tên đănh nhập | email)   │
 │  2. GetByKey(khoá bất biến)  → đã xử lý? → skipped        │
 │  3. khoá dòng FOR UPDATE trên user_subscriptions          │
 │  4. kiểm tra version không đổi?  → không? → error         │
 │  5. tính expires_at (cộng dồn / mới)                      │
 │  6. ghi user_subscriptions, version++                      │
 │  7. ghi user_subscription_events  (id sinh ở đây)         │
 │  8. ghi outbox_records  ← cùng giao dịch                  │
 │  9. ghi nhật ký kiểm toán  ← cùng giao dịch              │
 └────────────────────────────────────────────────────────────┘
      │                                    │
      │ RabbitMQ                           │ 60 giây
      ▼ (bất đồng bộ)                     │
 ┌────────────────────────────────────────────────────────────┐
 │ AI — bộ xử lý tín hiệu, MỘT giao dịch                    │
 │  1. kiểm tra phiên bản (từ chối nếu mới hơn)             │
 │  2. sourceEventId đã có grant? → bỏ qua                   │
 │  3. đóng bản ghi cũ / cắt đuôi / xoá future              │
 │  4. dựng policy_snapshot từ ai_cost_policies              │
 │  5. ghi ai_policy_grants                                   │
 └────────────────────────────────────────────────────────────┘
      │
      ▼
```

 AI — đường đọc (CHỈ ĐỌC, FR-042)
   1. không thấy grant? → suy giảm tầng 1: lấy chính sách phí của gói
      mặc định theo mã cấu hình, rồi vẫn đi qua sổ tính phí với
      grant = nil (xem research.md R8)
   2. tầng 1 hỏng? → suy giảm tầng 2: chỉ quan sát, không trần,
      0 lượt trả phí mỗi ngày, kèm cảnh báo
   3. thấy grant → dùng đúng hạn mức, số lượt và trần của gói

Bước 3 ở khối AI là **bắt buộc**: không có nó thì sập giữa chừng là mất sạch mức quyền tính phí.
