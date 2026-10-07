# Hợp đồng: Phân giải cửa sổ mức quyền

**Ngày**: 2026-10-07 | **Kế hoạch**: [plan.md](plan.md) | **Đặc tả**: [spec.md](spec.md)

Hợp đồng giữa tầng ứng dụng và tầng hạ tầng: một hàm trả về cửa sổ mức quyền cuối
cùng. Hàm này **không thuần** — nó đọc bản ghi mới nhất qua repository — nên chỉ được gọi
bên trong giao dịch, sau khi đã giữ khoá tài khoản. Chỉ phần dựng sàn cũ mới thuần. Không có
hợp đồng mạng nào ở đây — đây là hợp đồng **nội bộ giữa các hàm**, ghi ra
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

prior := GetNewestGrantForOrdering(ctx, userID)     // ORDER BY effective_from DESC
nếu prior == nil:
    trả về window                                  // không có sàn nào để dịch

// ---- Sàn cũ: chỉ chống hồi sinh, KHÔNG dùng để dịch ----
staleFloor := prior.CreatedAt
nếu staleFloor là zero hoặc staleFloor > now():
    staleFloor = prior.EffectiveFrom               // cột chịu CHECK, R-003
    GHI LOG cảnh báo FR-018 (user_id, grant_id)

nếu event.OccurredAt <= staleFloor:
    trả về lỗi từ chối vĩnh viễn, mã stale_event   // FR-002

// ---- Sàn mới: tránh chồng lấn, mới là chỗ để dịch ----
orderFloor := max(staleFloor, prior.EffectiveFrom)  // FR-004a

nếu window.EffectiveFrom <= orderFloor:
    shifted := orderFloor + 1µs                     // FR-006
    nếu window.EffectiveTo != nil && shifted >= *window.EffectiveTo:
        trả về lỗi từ chối vĩnh viễn               // FR-017, cửa sổ rỗng
    window.EffectiveFrom = shifted

trả về window
```

### Vì sao phải có hai sàn

Đây là điểm dễ làm sai nhất, nên ghi rõ. Nếu chỉ có một sàn lấy từ `created_at`:

| Kịch bản | `created_at` | `effective_from` | Sàn một | Kết quả |
| --- | --- | --- | --- | --- |
| Bản ghi thường | quá khứ | quá khứ | `created_at` | đúng |
| Bản ghi `tương lai` | quá khứ | **30 ngày nữa** | `created_at` | **sai** |

Ở hàng thứ hai, sàn là `created_at` (quá khứ), nên `sàn + 1µs` vẫn **nhỏ hơn** `effective_from`
của bản ghi `tương lai`. Bản ghi mới bắt đầu **trước** nó, và `TruncateGrantsBefore` — vốn
chỉ chạm vào dòng có `effective_from < cutoff` — **không** cắt được dòng `tương lai`. Kết quả:
hai cửa sổ đang hiệu lực chồng lấn, phá FR-008, và bộ đọc không còn nói được cái nào áp dụng.

Vì vậy sàn dịch phải là `max(staleFloor, prior.EffectiveFrom)`. Sàn cũ vẫn giữ vai trò chống
hồi sinh — nó hỏi "sự kiện này có mới hơn lần ghi bản ghi gần nhất không" — và sàn mới hỏi
"mốc bắt đầu mới có nằm sau mọi cửa sổ đã có không". Hai câu hỏi khác nhau, không thể dùng
một đáp số cho cả hai.

### Bất biến đầu ra

| # | Bất biến | Căn cứ |
| --- | --- | --- |
| G1 | `EffectiveFrom > orderFloor ≥ prior.EffectiveFrom` **luôn** | FR-004, FR-004a |
| G2 | `EffectiveFrom − orderFloor ≤ 1µs` khi có dịch | FR-006 |
| G3 | Khi `EventGrantWindow(event).EffectiveFrom > orderFloor`, đầu ra **bằng** cửa sổ gốc | FR-007 — không tác dụng phụ |
| G4 | `EffectiveTo == nil` hoặc `EffectiveTo > EffectiveFrom` | Không bao giờ vi phạm `CHECK` |
| G5 | Không đọc bảng nào ngoài `aiexecutor.ai_policy_grants` | Hiến pháp §I, §II |
| G6 | Không ghi gì | FR-012 — bỏ qua và từ chối đều không có tác dụng phụ |

G2 đo **độ dư quanh sàn mới**, không phải tổng độ dịch so với mốc sự kiện — xem FR-006. Một
sự kiện đến muộn 5 giây thì tổng dịch là 5 giây 1 µs và G2 vẫn đúng.

### Khi nào trả về lỗi

| Tình huống | Phân loại | Mã lý do |
| --- | --- | --- |
| `OccurredAt ≤ sàn cũ` | `RejectionError` (vĩnh viễn, không requeue) | `stale_event` |
| Dịch xong thì cửa sổ rỗng (FR-017) | `RejectionError` (vĩnh viễn) | `stale_event` |
| Lỗi cơ sở dữ liệu khi đọc `prior` | `RetryableError` (thử lại) | `transient_failure` |

> Cả hai trường hợp từ chối dùng chung `stale_event`. FR-002 yêu cầu giữ đúng hành vi sẵn có
> của nhánh này, nên thêm một mã mới ở đây sẽ phá yêu cầu đó. Trường hợp thứ hai là
> *hệ quả* của việc sự kiện đến quá muộn, không phải một nguyên nhân mới.
>
> **Hai nhánh từ chối này giờ có FR riêng** (FR-002 và FR-017). Nhánh "cửa sổ rỗng" trước đây
> chỉ tồn tại trong hợp đồng này mà không có yêu cầu nào ủng quyền, dù nó có tác dụng phụ với
> khách hàng.

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

**`rejectStaleEvent` bị XOÁ khỏi đường đi.** Nó không còn xuất hiện trong 6 bước trên, và
không được giữ lại ở vị trí cũ (`grant_uc.go:266`) — giữ cả hai nghĩa là chạy kiểm tra thứ
tự hai lần với hai tiêu chí khác nhau, và `rejectStaleEvent` vẫn từ chối vĩnh viễn khi
`ExtractOccurredAt` thất bại, tức là **giữ nguyên đúng lỗi đang sửa**. Phần cảnh báo và bộ
đếm của nhánh từ chối được chuyển vào `resolveGrantWindow` (FR-002, FR-018).

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
| `ClosesPriorGrants(EventType)` | `CloseActiveGrants(userID, cutoff, event.OccurredAt)` | **Có** — `cutoff` là tham số, xem dưới |
| còn lại (`EXTENDED`) | `TruncateGrantsBefore(userID, cutoff, event.OccurredAt)` | **Có** — `cutoff` là tham số, không tự tính lại |

**Cả hai nhánh đóng/cắt đều dùng `cutoff` đã chốt, không nhánh nào tự tính lại.** Đây là điểm
mà bản hợp đồng đầu tiên của 027 viết sai, và nó chỉ lộ ra khi triển khai.

Ban đầu tôi cho rằng nhánh `CloseActiveGrants` giữ nguyên `event.EffectiveAt` là an toàn, vì
FR-004 nói về "bản ghi mới" và nhánh đó không tạo bản ghi mới. Điều đó đúng về *chủ thể* nhưng
sai về *hệ quả*. `CloseActiveGrants` chạy

```text
UPDATE ... SET effective_to = cutoff, status = 'closed'
WHERE user_id = ? AND status = 'active' AND effective_from <= cutoff
```

Với một sự kiện đến muộn, `event.EffectiveAt` có thể **nhỏ hơn** `effective_from` của bản ghi
`active` mới nhất. Khi đó `effective_from <= cutoff` không thoả, bản ghi cũ **không** bị đóng,
trong khi bản ghi mới đã bị dịch lên `orderFloor + 1µs` — tức nằm **sau** nó. Hai bản ghi
`active` cùng lúc, phá FR-008 đúng bằng cơ chế mà `resolveGrantWindow` sinh ra để tránh.

Truyền `cutoff` đã dịch cho cả hai nhánh sửa điều đó: `cutoff = orderFloor + 1µs` mà
`orderFloor ≥ prior.effective_from`, nên `cutoff > prior.effective_from` và bản ghi cũ bị đóng
tại đúng mốc bản ghi mới bắt đầu — cửa sổ chạm nhau, không chồng lấn.

Với hai đường **không** tạo bản ghi mới (`applyRevocation` và `retireOnUnpriceableDowngrade`),
`cutoff = EventGrantWindow(event).EffectiveFrom = event.EffectiveAt`, nên hành vi của chúng
**không đổi** — giá trị truyền vào giống hệt giá trị chúng tự tính trước đây.

**Ba nơi gọi, không phải một.** Đây là điểm dễ sót:

| Nơi gọi | Truyền gì | Vì sao |
| --- | --- | --- |
| `grant_uc.go:269` — đường tạo grant | `window.EffectiveFrom` từ `resolveGrantWindow` | Đường duy nhất có bước chốt cửa sổ dưới khoá |
| `grant_uc.go:413` — đường thu hồi | `EventGrantWindow(event).EffectiveFrom` | Không tạo grant, không dịch cửa sổ |
| `grant_uc.go:500` — hạ cấp không định giá được | `EventGrantWindow(event).EffectiveFrom` | Như trên |

Bỏ sót nơi gọi thứ hai hoặc thứ ba thì **không biên dịch** — nhưng **truyền nhầm mốc** thì
biên dịch vẫn qua và sai hành vi. Vì vậy cả ba phải được sửa có chủ đích, không phải để trình
biên dịch bắt lỗi.

### 4b. Kiểm tra thứ tự dùng chung

`rejectStaleEvent` được gọi ở **hai** nơi: `grant_uc.go:266` (đường tạo grant — sẽ bị thay bằng
`resolveGrantWindow`) và `grant_uc.go:497` (hạ cấp tới một gói không định giá được — **giữ
lại**).

Phần dựng sàn cũ phải là một hàm thuần dùng chung để cả hai đường đều lùi về `created_at`, theo
FR-019. Nếu chỉ sửa đường tạo grant thì đường hạ cấp vẫn dừng vĩnh viễn trên chính dữ liệu
hỏng mà 027 khai là đang sửa — một lần sửa nửa vời, tệ hơn là không sửa gì vì nó tạo cảm giác
đã xong.

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