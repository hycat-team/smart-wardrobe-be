# Mô hình dữ liệu — 027 Sửa tắc hai nguyên nhân chặn vĩnh viễn

**Ngày**: 2026-10-07 | **Kế hoạch**: [plan.md](plan.md) | **Đặc tả**: [spec.md](spec.md)

027 **không** thêm bảng, không thêm cột, không đổi kiểu dữ liệu. Thay đổi nằm ở chỗ hệ thống
*đọc* và *tính* trên bảng đã có. Tài liệu này ghi lại những gì thay đổi và những gì không.

---

## 1. Thực thể: `aiexecutor.ai_policy_grants`

Bảng này **không đổi**. Tài liệu ghi lại các cột mà 027 thực sự dùng lại, và điều kiện ràng
buộc mà mọi tính toán phải tôn trọng.

| Cột | Kiểu | Vai trò với 027 |
| --- | --- | --- |
| `user_id` | uuid | Khoá tài khoản cho `pg_advisory_xact_lock` và cho mọi truy vấn theo tài khoản |
| `effective_from` | timestamptz | **Sàn thứ tự dự phòng cuối** (mức 3), chỉ khi cả `occurred_at` lẫn `created_at` không dùng được (FR-004a, R-003) |
| `effective_to` | timestamptz NULL | Ranh giới trên. NULL = đang hiệu lực không thời hạn |
| `created_at` | timestamptz | **Sàn thứ tự mức 2**, dùng khi ảnh chụp không đọc được `occurred_at` và giá trị này không nằm trong tương lai (R-003). Cột `started_at` **không tồn tại** |
| `occurred_at` (trong `policy_snapshot`) | timestamptz | **Sàn thứ tự mức 1, ưu tiên** — thời gian nghiệp vụ của thay đổi đã tạo ra bản ghi (FR-001) |
| `updated_at` | timestamptz | Không dùng làm sàn — đổi mỗi lần cắt bản ghi, không phản ánh thời điểm sự kiện |
| `status` | text | `active` / `future` / `closed`. Mục 4 của kiểm tra toàn vẹn lọc theo cửa sổ hiệu lực và **giữ nguyên** việc không lọc theo trạng thái — mục đó chưa từng lọc trạng thái (FR-014) |
| `policy_snapshot` | jsonb | Nguồn của `occurred_at` — đồng hồ thứ tự **ưu tiên** (FR-001), đọc qua `ExtractOccurredAt`. Khi không đọc được thì lùi về `created_at`, rồi `effective_from` |
| `source_event_id` | uuid | Chống áp dụng trùng; kiểm tra **lại** dưới khoá |

### Ràng buộc phải tôn trọng

| Ràng buộc | Nội dung | Ai giữ |
| --- | --- | --- |
| `CHECK ai_policy_grants_effective_window_check` | `effective_to > effective_from` | **Postgres**. Vi phạm là `23514`, giờ là từ chối vĩnh viễn (FR-009) |
| Quy tắc tầng ứng dụng (FR-034 của spec 024) | Các cửa sổ đang hiệu lực của một tài khoản không chồng lấn | `TruncateGrantsBefore` + `resolveGrantWindow` |

### Bất biến FR-034 không được đẩy xuống schema

Spec 024 FR-034 cấm một ràng buộc `UNIQUE` ở tầng dữ liệu, vì một tài khoản được phép giữ
nhiều dòng `active` **miễn là không chồng lấn**. Vì vậy bất biến không-chồng-lấn buộc phải
được giữ bằng khoá giao dịch ở tầng ứng dụng.

Hệ quả cho 027: FR-004 (mốc bắt đầu mới lớn hơn nghiêm mọi bản ghi trước) là **cách thức
duy nhất** để giữ bất biến đó mà không cần đổi schema. Nó phải được thực thi bằng khoá tài
khoản ở `LockGrantsForAccount` (`ai_policy_grant_writer.go:40`), không phải bằng một ràng
buộc mới.

---

## 2. Cấu trúc: `GrantWindow` sau khi thay đổi

Kiểu `GrantWindow` (`grant_window.go:11`) giữ nguyên ba trường. Cái đổi là **ai sinh ra nó**
và **có bao nhiêu cách sinh ra nó**.

```text
PlanChangedEvent
      │
      ├─► EventGrantWindow(event) ──► GrantWindow  [thuần, không cần DB]
      │         "sự kiện nói rằng cửa sổ nên là gì"
      │
      └─► resolveGrantWindow(ctx, event) ──► GrantWindow | error
                │  [trong giao dịch, sau khoá tài khoản]
                │  "trạng thái nói rằng cửa sổ được phép là gì"
                │
                ├─ EventGrantWindow(event)
                ├─ GetNewestGrantForOrdering(userID)
                ├─ không có bản ghi trước? ──► trả về nguyên cửa sổ gốc
                ├─ dựng sàn: created_at (nếu ≤ now) nếu không thì effective_from
                ├─ OccurredAt ≤ sàn? ──► từ chối vĩnh viễn (FR-002)
                └─ nâng EffectiveFrom lên sàn + 1µs, không vượt quá mức đó (FR-004/FR-006)
```

### Quy tắc xác định sàn thứ tự

```text
sàn_cũ(grant) = grant.occurred_at      nếu đọc được từ ảnh chụp chính sách   (mức 1, ưu tiên)
              = grant.created_at       nếu created_at đọc được và ≤ now()    (mức 2, FR-003)
              = grant.effective_from   nếu không                            (mức 3, FR-018)

sàn_mới = max(sàn_cũ, grant.effective_from)
```

Ba mức, không phải hai. Mức 1 là thời gian **nghiệp vụ** của thay đổi đã tạo ra bản ghi; mức 2
là thời gian **ghi hàng**. Khi hàng đợi có độ trễ, hai mốc lệch nhau, và lấy mức 2 làm ưu tiên
sẽ từ chối nhầm một sự kiện hợp lệ xảy ra trong khoảng lệch (FR-001). Mức 3 chỉ dùng khi cả
ảnh chụp lẫn `created_at` đều không dùng được.

Hai **sàn** (cũ và mới) khác nhau vì chúng trả lời hai câu hỏi khác nhau: `sàn_cũ` hỏi "sự
kiện này có mới hơn thay đổi vừa áp dụng không" và bảo vệ FR-033 của spec 024; `sàn_mới` hỏi
"mốc bắt đầu mới có nằm sau mọi cửa sổ đã có không" và bảo vệ FR-004/FR-008.

Dùng chung một sàn là lỗi: với bản ghi `tương lai`, `created_at` nằm ở quá khứ còn
`effective_from` nằm ở tương lai, nên sàn lấy từ `created_at` sẽ để bản ghi mới bắt đầu
**trước** bản ghi `tương lai`, và `TruncateGrantsBefore` không cắt được dòng đó.

| Điều kiện | Kết quả | Căn cứ |
| --- | --- | --- |
| Không có bản ghi trước | Cửa sổ gốc, không dịch | Không có bản ghi trước thì không có sàn |
| `OccurredAt` ≤ sàn cũ | **Từ chối vĩnh viễn**, `stale_event` | FR-002 — bảo vệ FR-033 của spec 024 |
| `OccurredAt` > sàn mới | Giữ nguyên cửa sổ gốc từ `EventGrantWindow` (không phải `OccurredAt` — nhánh `EXTENDED` dùng `EffectiveTo − DurationDays`) | FR-007 — đường đi bình thường, không tác dụng phụ |
| Sàn mới + 1µs ≥ `EffectiveTo` | **Từ chối vĩnh viễn** | FR-017 — cửa sổ rỗng sau khi dịch |

### Vì sao một nguồn sự thật là bắt buộc

Cả bản ghi mới **và** điểm cắt bản ghi cũ phải dùng **cùng** `EffectiveFrom`. Nếu tách:

```text
mốc cắt  = OccurredAt              (chưa dịch)
mốc đầu  = sàn + 1µs              (đã dịch)
                  ↑ lệch đúng 1 µs
```

Khoảng lệch 1 µs đó phá một trong hai ràng buộc: bản ghi cũ bị cắt *sau* mốc đầu mới thì
**chồng lấn** (FR-008); bản ghi mới bắt đầu *sau* mốc cắt thì có **khe hở 1 µs** và tài khoản
bị từ chối bởi mọi lệnh gọi AI trong khoảng đó. Vì vậy `resolveGrantWindow` được gọi **một
lần** và kết quả được dùng cho cả hai.

---

## 3. Chuyển trạng thái

Tập trung vào nhánh tạo bản ghi mới (`EXTENDED`), là nhánh mà 027 sửa.

```text
                   ┌─────────────────────────────────────────────┐
                   │ có bản ghi mới nhất?                        │
                   └──────────────┬──────────────────────────────┘
                        không     │        có
                   ┌───────────────┘         └──────────────┐
                   ▼                                       ▼
       giữ nguyên cửa sổ gốc                dựng sàn cũ từ created_at
       từ EventGrantWindow                  (lùi effective_from nếu không tin được)
       (không dịch — FR-007)                              │
                   │                          ┌────────────┴────────────┐
                   │                    OccurredAt ≤ sàn cũ     OccurredAt > sàn cũ
                   │                          │                       │
                   │                          ▼                       ▼
                   │                  TỪ CHỐI VĨNH VIỄN        sàn mới = max(sàn cũ,
                   │                  stale_event                 prior.effective_from)
                   │                  (FR-002)                          │
                   │                                                      ▼
                   │                                      cửa sổ gốc ≤ sàn mới?
                   │                                       không          │ có
                   │                                        ▼            ▼
                   │                              giữ cửa sổ gốc    sàn mới + 1µs
                   │                              (không dịch)              │
                   │                                                          ▼
                   │                                            sàn mới+1µs ≥ EffectiveTo?
                   │                                              có          │ không
                   │                                               ▼          ▼
                   │                                       TỪ CHỐI VĨNH VIỄN  EffectiveFrom =
                   │                                       (FR-017)           sàn mới+1µs
                   │                                               │              │
                   └───────────────────────┬───────────────────────┴──────────────┘
                                           ▼
                        TruncateGrantsBefore(userID, EffectiveFrom)   ← cùng một giá trị
                        + CreateGrant(effective_from = EffectiveFrom)
                        + tất cả trong MỘT giao dịch, sau khoá tài khoản
```

### Hệ quả trên dữ liệu production

Áp dụng vào 8 tài khoản có grant hỏng, mỗi tài khoản đi qua đúng một lần:

| Bước | Trạng thái trước | Trạng thái sau |
| --- | --- | --- |
| Bản ghi cũ | `active`, `effective_to` NULL, ảnh chụp không đọc được `envelope` | `closed`, `effective_to = mốc cắt`, **vẫn giữ nguyên ảnh chụp hỏng** |
| Bản ghi mới | không có | `active`, `effective_from = mốc cắt`, ảnh chụp có `envelope` + `entitlements` |

Hai điểm cần nói rõ vì chúng dễ bị hiểu sai:

1. **Bản ghi hỏng không bị xoá hay sửa.** Nó được đóng lại và nằm ngoài cửa sổ hiệu lực. Đó là
   lý do `DELETE future grants` không liên quan ở đây, và lý do FR-033 của spec 024 được
   bảo toàn: mức quyền cũ không bị hồi sinh, nó chỉ ngừng có hiệu lực.
2. **Mục 4 của kiểm tra toàn vẹn về 0 là nhờ lọc lại, không nhờ xoá dữ liệu.** Bản ghi hỏng
   vẫn còn trong bảng và vẫn được đếm ở truy vấn thông tin (FR-016). Đó là chủ ý — xoá đi là
   mất dấu vết của một sự cố đã xảy ra.

---

## 4. Phân loại lỗi

Không có mô hình dữ liệu mới. Đây là bảng tra cứu phân loại, thuần, kiểm thử được.

| Mã Postgres | Nghĩa | Phân loại | Mã lý do | Có thử lại không |
| --- | --- | --- | --- | --- |
| `23514` | `check_violation` | Từ chối vĩnh viễn | `policy_grant_constraint_violation`¹ | **Không** — vi phạm ràng buộc không tự lành |
| `23505` | `unique_violation` | Tạm thời | `transient_failure` | Có — đua duy nhất theo `source_event_id` là tình huống tạm thời |
| `23503` | `foreign_key_violation` | Tạm thời | `transient_failure` | Có |
| `08xxx`, `40xxx`, `57P03` | timeout, deadlock, shutdown | Tạm thời | `transient_failure` | Có |
| còn lại | — | Tạm thời | `transient_failure` | Có, như hiện tại |

¹ Chuỗi mã lý do đúng được chốt ở `tasks.md`, không phải ở tài liệu này. Đặc tả 027 yêu cầu
một mã **riêng** (FR-009) và yêu cầu nó xuất hiện trong bộ đếm công khai (FR-011), nhưng không
chốt tên. Đề xuất trên theo đúng quy ước của năm mã đang có trong `grant_uc.go:43-52`. Vì
đây là khoá của bộ đếm và tên cảnh báo, nó là hợp đồng với vận hành: đổi sau này thì cả
bảng đếm lẫn cảnh báo đều phải đổi theo.

Vì sao **không** bắt cả lớp `23`: `23505` xảy ra khi hai giao dịch cùng thắng đua trên
`source_event_id`. Đó là cuộc đua, thử lại thì một bên thắng — bắt rộng sẽ biến nó thành
thư chết vĩnh viễn và làm FR-010 sai. Chi tiết ở [research.md](research.md) R-001.

---

## 5. Điều KHÔNG thay đổi

Ghi lại tường minh để `tasks.md` không mở rộng phạm vi theo hiểu lầm:

- **Không** migration. Không thêm bảng, cột, chỉ mục hay ràng buộc.
- **Không** đổi schema của `subscription.*`. Dịch vụ Account không được biết gì thêm.
- **Không** thay đổi hợp đồng sự kiện `subscription.plan.changed` — chỉ đính chính mô tả
  trong hợp đồng (CHK028).
- **Không** thay đổi đường khôi phục. Nó vẫn trả `already_restored` vĩnh viễn cho các khoá
  `RESTORE_GRANT:<userID>:<version>` đã ghi.
- **Không** sửa dữ liệu bằng SQL. Đường sửa là `POST /api/v1/admin/subscription-grants`
  với khoá chống trùng mới (FR-013).