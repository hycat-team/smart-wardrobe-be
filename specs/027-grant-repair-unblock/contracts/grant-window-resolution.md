# Hợp đồng: Phân giải cửa sổ mức quyền

**Ngày**: 2026-10-07 | **Kế hoạch**: [plan.md](plan.md) | **Đặc tả**: [spec.md](spec.md)

Hợp đồng giữa tầng ứng dụng và tầng hạ tầng: một hàm thuần trả về cửa sổ mức quyền cuối
cùng. Không có hợp đồng mạng nào ở đây — đây là hợp đồng **nội bộ giữa các hàm**, ghi ra
để `tasks.md` và kiểm thử có một chuẩn so sánh.

---

## 1. `EventGrantWindow(event) GrantWindow`

Tên mới của `ComputeGrantWindow` (`grant_window.go:29`). Đổi tên vì sau 027 có **hai** loại
cửa sổ khác nhau và tên cũ không phân biệt được (xem Mục 1 của kế hoạch).

**Thuần hoàn toàn.** Không đọc cơ sở dữ liệu, không nhận `context`. Có thể kiểm thử không cần
hạ tầng.

### Hàm vi

```go
func EventGrantWindow(event *PlanChangedEvent) GrantWindow
```

### Hành vi — không đổi so với bản hiện tại

| Điều kiện | `EffectiveFrom` | `EffectiveTo` | `Status` |
| --- | --- | --- | --- |
| `EffectiveTo == nil` | `event.EffectiveAt` | `nil` | `active` |
| `EventType != EXTENDED` hoặc `DurationDays <= 0` | `event.EffectiveAt` | `event.EffectiveTo` | `active` |
| `EXTENDED`, `start ≤ OccurredAt` | `EffectiveTo − DurationDays` | `event.EffectiveTo` | `active` |
| `EXTENDED`, `start > OccurredAt` | `EffectiveTo − DurationDays` | `event.EffectiveTo` | `future` |

Nhánh thứ tư là quan trọng: cấp quyền trước thời điểm bắt đầu là cấp quyền cho thời gian
chưa mua. 027 **không** chạm vào bảng này.

---

## 2. `resolveGrantWindow(ctx, event) (GrantWindow, error)`

Hàm mới. Chỉ gọi **bên trong** giao dịch, **sau** `LockGrantsForAccount`.

```go
func (uc *GrantUseCase) resolveGrantWindow(
    ctx context.Context,
    event *PlanChangedEvent,
) (GrantWindow, error)
```

### Điều kiện tiên quyết — vi phạm thì là lỗi lập trình, không phải lỗi dữ liệu

| Điều kiện | Hệ quả nếu vi phạm |
| --- | --- |
| Đã giữ khoá tài khoản | Hai giao dịch cùng quyết định từ hai trạng thái → chồng lấn |
| Chạy trong cùng giao dịch sẽ ghi bản ghi | Tính ra một cửa sổ rồi bỏ, cửa sổ thật lại khác |

### Thuật toán

```text
window := EventGrantWindow(event)

prior := GetNewestGrantForOrdering(ctx, userID)
nếu prior == nil:
    trả về window                                  // không có sàn nào để dịch

floor := prior.StartedAt
nếu floor là zero hoặc floor > now():
    floor = prior.EffectiveFrom                    // cột chịu CHECK, R-003

nếu event.OccurredAt <= floor:
    trả về lỗi từ chối vĩnh viễn, mã stale_event   // FR-002

nếu window.EffectiveFrom <= floor:
    shifted := floor + 1µs
    nếu window.EffectiveTo != nil && shifted >= *window.EffectiveTo:
        trả về lỗi từ chối vĩnh viễn               // cửa sổ rỗng sau khi dịch
    window.EffectiveFrom = shifted

trả về window
```

### Bất biến đầu ra

| # | Bất biến | Căn cứ |
| --- | --- | --- |
| G1 | `EffectiveFrom > floor` **luôn** | FR-004 |
| G2 | `EffectiveFrom − EventGrantWindow(event).EffectiveFrom ≤ 1µs` | FR-006 |
| G3 | Khi `EventGrantWindow(event).EffectiveFrom > floor`, đầu ra **bằng** cửa sổ gốc | FR-007 — không tác dụng phụ |
| G4 | `EffectiveTo == nil` hoặc `EffectiveTo > EffectiveFrom` | Không bao giờ vi phạm `CHECK` |
| G5 | Không đọc bảng nào ngoài `aiexecutor.ai_policy_grants` | Hiến pháp §I, §II |
| G6 | Không ghi gì | FR-012 — bỏ qua và từ chối đều không có tác dụng phụ |

### Khi nào trả về lỗi

| Tình huống | Phân loại | Mã lý do |
| --- | --- | --- |
| `OccurredAt ≤ floor` | `RejectionError` (vĩnh viễn, không requeue) | `stale_event` |
| Dịch xong thì cửa sổ rỗng | `RejectionError` (vĩnh viễn) | `stale_event` |
| Lỗi cơ sở dữ liệu khi đọc `prior` | `RetryableError` (thử lại) | `transient_failure` |

> Cả hai trường hợp từ chối dùng chung `stale_event`. FR-002 yêu cầu giữ đúng hành vi sẵn có
> của nhánh này, nên thêm một mã mới ở đây sẽ phá yêu cầu đó. Trường hợp thứ hai là
> *hệ quả* của việc sự kiện đến quá muộn, không phải một nguyên nhân mới.

---

## 3. Điểm nối vào giao dịch

Hợp đồng về **thứ tự** trong giao dịch — thứ tự này là toàn bộ ý nghĩa của FR-005:

```text
uow.Execute:
  1. LockGrantsForAccount(userID)                    // khoá tài khoản
  2. GetGrantBySourceEventID(sourceEventID)          // kiểm tra trùng dưới khoá
     └─ trùng → alreadyApplied, không ghi gì
  3. window := resolveGrantWindow(ctx, event)        // MỚI: dùng khoá ở bước 1
  4. grant.EffectiveFrom = window.EffectiveFrom      // MỚI
  5. retirePreviousGrants(ctx, event, window.EffectiveFrom)   // ĐỔI CHỮ KÝ
  6. CreateGrant(grant)
```

Sai thứ tự nào cũng hỏng:

| Sai lệch | Hậu quả |
| --- | --- |
| `resolveGrantWindow` **trước** khoá | Hai sự kiện cùng quyết định từ hai trạng thái → hai grant `active` chồng lấn (FR-034 của spec 024) |
| `retirePreviousGrants` tự tính lại cửa sổ | Mốc cắt lệch mốc bắt đầu 1 µs → chồng lấn hoặc khe hở |
| Gán `grant.EffectiveFrom` **trước** khoá | Ghi đè giá trị đã tính dưới khoá bằng giá trị đo trước khoá |

`grant` được dựng ở `grant_uc.go:207`, **ngoài** giao dịch. Điều đó vẫn đúng cho
`buildGrant`: nó chỉ ghép dữ liệu tĩnh từ sự kiện và chính sách. Chỉ `EffectiveFrom` — trường
duy nhất phụ thuộc trạng thái — được gán lại bên trong giao dịch ở bước 4.

---

## 4. Chữ ký `retirePreviousGrants` đổi

```text
CŨ:  retirePreviousGrants(ctx context.Context, event *PlanChangedEvent) error
MỚI: retirePreviousGrants(ctx context.Context, event *PlanChangedEvent, cutoff time.Time) error
```

| Nhánh | Hành vi | Có đổi không |
| --- | --- | --- |
| `DeletesPendingGrants(EventType)` | `DeleteFutureGrants` | Không |
| `ClosesPriorGrants(EventType)` | `CloseActiveGrants(userID, event.EffectiveAt, event.OccurredAt)` | **Không** — nhánh thu hồi không tạo bản ghi mới, nên FR-004 không áp dụng |
| còn lại (`EXTENDED`) | `TruncateGrantsBefore(userID, cutoff, event.OccurredAt)` | **Có** — `cutoff` là tham số, không tự tính lại |

---

## 5. Bất biến giữa các hàm

Những điều `tasks.md` phải giữ được đồng thời, và là điều làm FR-008 đúng:

```text
TruncateGrantsBefore(cutoff = window.EffectiveFrom)
        chạm vào dòng thỏa:  effective_from < cutoff
                              AND (effective_to IS NULL OR effective_to > cutoff)
        ⇒ đặt effective_to := cutoff, và do effective_from < cutoff
          nên CHECK effective_to > effective_from vẫn thoả.

Bản ghi mới: effective_from := cutoff
        ⇒ chỉ chạm từ đúng cutoff trở đi.

⇒ Hai cửa sổ chạm tại cutoff, KHÔNG chồng lấn. Không có khe hở.
```

Đây là lý do `TruncateGrantsBefore` giữ nguyên điều kiện lọc hiện tại
(`ai_policy_grant_writer.go:80`) và lý do 027 không đổi tệp đó.