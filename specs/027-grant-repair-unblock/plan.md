# Kế hoạch triển khai: 027 Sửa tắc hai nguyên nhân chặn vĩnh viễn

**Nhánh**: `027-grant-repair-unblock` | **Ngày**: 2026-10-07 | **Đặc tả**: [spec.md](spec.md)

**Đầu vào**: Đặc tả tính năng từ `/specs/027-grant-repair-unblock/spec.md`
**Nghiên cứu**: [research.md](research.md) — mọi `NEEDS CLARIFICATION` đã được giải quyết

## Tóm tắt

Hai nguyên nhân khiến 18/28 tài khoản production không được sửa vĩnh viễn khi chạy đường
khôi phục ngày 2026-10-07, cần sửa ở phía dịch vụ AI:

1. **`rejectStaleEvent` từ chối vĩnh viễn.** Khi ảnh chụp chính sách không đọc được
   `occurred_at`, hệ thống lùi về `created_at` của bản ghi — mốc bằng đúng thời điểm sự kiện
   được ghi — rồi thấy sự kiện "không mới hơn" và từ chối. Khoá chết, không chết tạm thời.
2. **Vi phạm `CHECK` bị xếp nhầm là tạm thời.** Khi `effective_to == effective_from`, Postgres
   trả `23514`, và mã đó đi vào nhóm thử lại — thử vô hạn một vi phạm ràng buộc không bao giờ
   tự lành.

Cách sửa: sửa cách đọc mốc thứ tự, nới lỏng bất biến thứ tự thành **dịch mốc bắt đầu 1 µs**
thay vì từ chối, và phân loại lại `23514` thành từ chối vĩnh viễn với mã lý do riêng.

Phần sửa dữ liệu vận hành **không** nằm trong kế hoạch này — nó dùng đường hành chính
`POST /api/v1/admin/subscription-grants` và chỉ chạy được **sau khi** kế hoạch này triển
khai. Thao tác vận hành nằm ở [runbook.md](runbook.md).

Phát hiện cấu trúc quan trọng nhất, từ R-002: FR-005 **không thể thực hiện bằng cách vá một
chỗ**. `retirePreviousGrants` tự tính lại cửa sổ từ sự kiện (`grant_uc.go:279`), trong khi
`grant` được dựng ngoài giao dịch (`grant_uc.go:207`). Nếu tính lại ở một chỗ, mốc cắt và
mốc bắt đầu lệch nhau 1 µs — đủ để tạo chồng lấn hoặc khe hở. Phải có một nguồn sự thật.

## Bối cảnh kỹ thuật

**Ngôn ngữ/Phiên bản**: Go 1.25 (xem `go.mod` từng module)

**Phụ thuộc chính**: `gorm.io/driver/postgres v1.6.3` (chạy trên `pgx/v5` stdlib);
`github.com/jackc/pgx/v5` sẽ được **nâng từ phụ thuộc gián tiếp lên trực tiếp** trong
`services/ai/go.mod` để đọc `*pgconn.PgError` (xem R-001)

**Lưu trữ**: PostgreSQL 17. Bảng `aiexecutor.ai_policy_grants`, thuộc sở hữu riêng của dịch
vụ AI

**Kiểm thử**: `go test -race -count=3`; kiểm thử hợp đồng cho phân loại lỗi cần **Postgres
thật** (xem Rủi ro 1)

**Nền tảng đích**: Dịch vụ Go chạy trong container trên VPS Linux

**Loại dự án**: Dịch vụ web vi mô dịch vụ, tiêu thụ hàng đợi RabbitMQ

**Mục tiêu hiệu năng**: Không thay đổi. Mọi thao tác bổ sung nằm trong giao dịch đang có
sẵn và dùng chính các truy vấn đã được gọi

**Ràng buộc**:
- Mức dịch mốc bắt đầu ≤ 1 microsecond (FR-006)
- Không có tác dụng phụ ở vận hành bình thường (FR-007)
- Các cửa sổ của một tài khoản không chồng lấn (FR-008, FR-034 của spec 024)
- Bảng mức quyền không nhận ghi từ ngoài dịch vụ AI (FR-013)

**Quy mô**: 8 tài khoản cần sửa trên production; 28 tài khoản trong phạm vi lượt khôi phục

## Kiểm tra Hiến pháp

*GATE: phải qua trước Phase 0. Kiểm tra lại sau Phase 1.*

| Điều lệ | Áp dụng | Kết quả |
| --- | --- | --- |
| **§I** Dịch vụ triển khai độc lập | Bảng `ai_policy_grants` chỉ được dịch vụ AI ghi. FR-013 cấm sửa SQL trực tiếp, và đường sửa là đường hành chính có sự kiện. | **Đạt** |
| **§II** Ranh giới mô-đun nghiêm ngặt | Sửa nằm trong `services/ai/.../aiexecutor`, không đọc bảng của dịch vụ khác. `resolveGrantWindow` là hàm thuần trong tầng ứng dụng, không cần truy cập repository khác. | **Đạt** |
| **§III** Xử lý sự kiện bất đồng bộ | Không thêm hàng đợi, không đổi loại sự kiện. Việc phân loại `23514` thành từ chối vĩnh viễn **giữ** nguyên hành vi không tái phát. | **Đạt** |
| **§IV** Kiến trúc sạch, phụ thuộc tầng một chiều | `grant_window.go` ở tầng ứng dụng, không phụ thuộc `pgx`. Hằng `23514` và kiểm tra `*pgconn.PgError` đặt ở tầng hạ tầng vì đó là nơi duy nhất biết về Postgres. | **Đạt** |
| **§V** Chuẩn hoá giao dịch và xử lý lỗi | FR-005 giữ tính lại trong giao dịch sau khoá tài khoản, nên mọi quyết định đều dựa trên trạng thái dưới khoá. | **Đạt** |
| **§VI** Đường ống AI, chi phí, dự phòng cục bộ | Không có đường dẫn chi phí nào đổi. Việc sửa ảnh chụp chính sách chính là điều kiện để giới hạn chi phí được áp dụng lại. | **Đạt** |
| **§VII** Tuân thủ mẫu thiết kế | Quyết định nới lỏng bất biến thứ tự có hệ quả bảo mật được nâng thành [ADR 014](../../docs/decisions/014-relax-grant-ordering-invariant.md), không chỉ nằm trong đặc tả. | **Đạt** |
| **§VIII** Không phụ thuộc nhà cung cấp | `SQLState` là mã do chính PostgreSQL định nghĩa, không phải thông điệp của phiên bản cụ thể. | **Đạt** |
| **§7** Việc ngoài phạm vi vào `deferred.md` | Bốn phát hiện được ghi ở [deferred.md](deferred.md). | **Đạt** |

Kết luận: không có vi phạm. **Complexity Tracking không cần điền.**

## Rủi ro thiết kế

**Rủi ro 1 — chuỗi lỗi có đi qua GORM không (chưa đóng).** Nếu GORM bọc lỗi driver bằng
`fmt.Errorf` không dùng `%w`, `errors.As` không bóc được `*pgconn.PgError`, việc phân loại rơi
về mặc định "tạm thời", và ta quay lại **đúng** lỗi gốc mà không có tín hiệu nào báo trước.
Vì không nơi nào trong repo từng làm việc này (`shared/infrastructure/db/` không đọc
`PgError` lần nào), đây là điều chưa biết chứ không phải điều chắc.

> `tasks.md` **bắt buộc** phải có kiểm thử tích hợp trên Postgres thật, ép vi phạm `CHECK`
> thật và chứng minh `errors.As` bóc được. Kiểm thử đơn vị với `PgError` dựng tay **không
> đủ** — nó sẽ đỏ ngay cả khi tích hợp hỏng.

**Rủi ro 2 — trùng tên `ComputeGrantWindow`.** Sau khi gộp, có hai cửa sổ cho mỗi sự kiện
với hai ý nghĩa khác nhau: cái gốc từ sự kiện và cái đã chốt dưới khoá. Đặt tên lại để sự
khác biệt hiện ra ở chỗ đọc; xem Mục 1 bên dưới.

**Rủi ro 3 — mã lý do mới thay đổi bộ đếm công khai.** Đã quyết định là tương thích ngược
(R-004), nhưng phải xác nhận bằng chứng, không bằng cảm giác; xem Mục 4.

## Cấu trúc dự án

### Tài liệu (tính năng này)

```text
specs/027-grant-repair-unblock/
├── spec.md                       # Đầu vào của /speckit-clarify
├── plan.md                       # File này
├── research.md                   # Kết quả Phase 0 (R-001..R-006)
├── data-model.md                 # Kết quả Phase 1
├── quickstart.md                 # Kết quả Phase 1
├── runbook.md                    # Thao tác vận hành, CHẠY SAU khi triển khai
├── deferred.md                   # Phát hiện ngoài phạm vi (hiến pháp §7)
├── checklists/
│   └── requirements.md           # Checklist chất lượng của đặc tả
└── contracts/
    ├── grant-window-resolution.md
    └── error-classification.md
```

### Mã nguồn (gốc repo)

```text
services/ai/internal/modules/aiexecutor/
├── application/usecase/grant/
│   ├── grant_uc.go               # Điểm vào sự kiện, giữ giao dịch và khoá
│   ├── grant_window.go           # ComputeGrantWindow, CreatesNewGrant, ClosesPriorGrants
│   ├── grant_error_class.go      # MỚI: phân loại lỗi theo SQLSTATE
│   └── grant_reject_metrics.go   # Bộ đếm từ chối, nơi mã lý do mới xuất hiện
└── infrastructure/persistence/
    └── ai_policy_grant_writer.go # LockGrantsForAccount, TruncateGrantsBefore,
                                   # GetNewestGrantForOrdering, CreateGrant

database/scripts/
└── 07-verify_policy_grants.sql   # Mục 4 đổi phạm vi theo FR-014..FR-016

contracts/
└── subscription-plan-changed-event.md  # Đính chính dòng mô tả stale_event (CHK028)

docs/decisions/
└── 014-relax-grant-ordering-invariant.md  # MỚI
```

**Quyết định cấu trúc**: giữ nguyên bố cục hiện có. Đặc tả này sửa **một** đường đi trong
một mô-đun, không sinh mô-đun mới, nên không có "Lựa chọn" nào để chọn. Thư mục
`contracts/` được đặt **bên trong** thư mục đặc tả, khác với `contracts/` ở gốc repo —
đó là nơi hợp đồng sự kiện đã tồn tại; hai thư mục này phục vụ hai tầng khác nhau.

## Quyết định thiết kế

### Mục 1 — `ComputeGrantWindow` tách thành hai hàm, và vì sao

FR-004/FR-005 yêu cầu tính lại mốc bắt đầu bên trong giao dịch. `ComputeGrantWindow`
(`grant_window.go:29`) là hàm thuần lấy thời điểm sự kiện — không nhận trạng thái cơ sở dữ
liệu, không thể làm việc đó.

Đổi tên phần dẫn xuất từ sự kiện thành `EventGrantWindow(event)`, giữ hàm thuần. Thêm hàm
mới `resolveGrantWindow(ctx, event) (GrantWindow, error)` trong cùng tầng, làm ba việc theo
thứ tự:

1. gọi `EventGrantWindow` lấy cửa sổ gốc;
2. đọc bản ghi mới nhất qua `GetNewestGrantForOrdering`, và **nếu không có** thì trả về cửa
   sổ gốc ngay — không có bản ghi trước thì không có sàn thứ tự;
3. dựng sàn từ `created_at` nếu nó không nằm trong tương lai so với `now()`, nếu không thì
   từ `effective_from` (R-003); từ chối vĩnh viễn nếu `event.OccurredAt` không mới hơn sàn
   (FR-002); nếu còn lại thì nâng `EffectiveFrom` lên `sàn + 1 µs`, không vượt quá mức đó.

Sau đó `grant.EffectiveFrom` được gán bằng `window.EffectiveFrom` và `retirePreviousGrants`
**nhận** mốc cắt đó làm tham số.

Tên gọi có chủ đích: `EventGrantWindow` là cái *sự kiện nói*; `resolveGrantWindow` là cái
*trạng thái nói*. Rủi ro 2 nêu trên là lý do không được để cả hai tên chung gốc — đọc
`ComputeGrantWindow` ở một chỗ và `resolveGrantWindow` ở chỗ khác trong cùng một tệp thì
người đọc không có cách nào biết mình đang dùng loại nào.

### Mục 2 — `retirePreviousGrants` đổi chữ ký

Hiện tại (`grant_uc.go:238`) nó tự gọi `ComputeGrantWindow(event).EffectiveFrom`. Đổi thành
nhận `cutoff time.Time`. Đây là thay đổi cấu trúc **bắt buộc**, không phải tuỳ chọn: nếu
không, mốc cắt và mốc bắt đầu lệch nhau (R-002).

Nhánh `CloseActiveGrants` (thu hồi) giữ nguyên — FR-004 nói về "bản ghi mức quyền mới", và
nhánh đó không tạo bản ghi mới.

### Mục 3 — Phân loại `23514` ở đâu

Đặt trong tệp mới `grant_error_class.go`, cùng tầng ứng dụng với `transient`/`reject`, vì
đó là nơi phân loại đã diễn ra. Hàm `isCheckViolation(err) bool` dùng `errors.As` với
`*pgconn.PgError` và so `SQLState == "23514"`.

Hằng `23514` khai trong tệp đó kèm chú thích nói rõ **vì sao chỉ bắt `23514` chứ không bắt
cả lớp `23`** — lý do nằm ở R-001 và rất dễ bị "sửa cho gọn" thành `strings.HasPrefix(code,
"23")` bởi một người đọc sau không đọc nghiên cứu.

Trong xử lý lỗi của giao dịch (`grant_uc.go:282`), thêm nhánh phân loại **trước** nhánh
`transient`, và chỉ khi `isCheckViolation` mới ghi vào mã lý do mới; còn lại giữ nguyên
đường `transient` cũ (FR-010).

### Mục 4 — Mã lý do mới: tương thích ngược, có bằng chứng

R-004 kết luận **không** cần `frontend-guide.md`: nhãn của `grant_rejected_by_reason` là
`string` tự do, không phải kiểu liệt kê đóng, nên khóa mới không làm vỡ client đọc theo
khoá.

`tasks.md` có một mục xác minh đúng điều đó bằng chứng trước khi đóng CHK026. Nếu kiểm tra
cho thấy nhãn là kiểu đóng hoặc có bảng giá trị được sinh từ một danh sách cố định, thì kết
luận R-004 sai, CHK026 mở lại và `frontend-guide.md` trở thành bắt buộc.

### Mục 5 — Sửa phạm vi mục 4 của kiểm tra toàn vẹn

Ba thay đổi, theo FR-014..FR-016:

- **Lọc theo cửa sổ hiệu lực** (`effective_from <= now() AND (effective_to IS NULL OR
  effective_to > now())`) và **bỏ hẳn** điều kiện lọc theo trạng thái, khớp đúng điều kiện
  bộ đọc dùng (`ai_policy_grant_writer.go:90-96`);
- tách một truy vấn **thông tin** đếm bản ghi lịch sử có ảnh chụp không đọc được, không tính
  vào kết quả kiểm tra;
- cập nhật phần mở đầu "sáu bất biến" vì phạm vi của mục 4 đã thay đổi, và dòng trỏ tới
  `restore` ở mục 1 — sau sự cố 027, `restore` là một trong hai nguyên nhân, nên dòng gợi ý
  nó là "cách sửa" là sai.

### Mục 6 — Đính chính hợp đồng sự kiện

`subscription-plan-changed-event.md` mô tả phạm vi của `stale_event` không khớp với
`rejectStaleEvent` (`grant_uc.go:237`). Sau 027, hai nghĩa từ chối bằng tên đó đã tách bạch,
nên mô tả sai sẽ khiến người đọc hợp đồng hiểu nhầm vĩnh viễn. Sửa ngay trong đợt này
(CHK028).

### Mục 7 — Ranh giới với phần sửa dữ liệu vận hành

Triển khai kế hoạch này **không** sửa dữ liệu. `runbook.md` là bước sau, và lệnh thứ hai
trong đó là cổng chặn "hãy chắc chắn bản sửa đã lên". Việc sửa dữ liệu dùng
`POST /api/v1/admin/subscription-grants` với khoá chống trùng mới — nó sinh sự kiện
`EXTENDED`, gọi `TruncateGrantsBefore` (`ai_policy_grant_writer.go:77`), và chỉ đặt được
`effective_to = mốc cắt` đúng khi hành vi tính lại của Mục 1 đã có.

## Kiểm tra Hiến pháp sau thiết kế

*Re-check sau Phase 1.*

| Điều lệ | Thay đổi thiết kế có làm hỏng không | Kết quả |
| --- | --- | --- |
| **§I** | Không có truy vấn nào mới; không có đường ghi mới. Đường sửa dữ liệu vẫn là đường hành chính có sự kiện. | **Đạt** |
| **§II** | `resolveGrantWindow` chỉ dùng `aiCostRepo`, cùng repository với phần đang sửa. Không đọc bảng của dịch vụ khác. | **Đạt** |
| **§III** | Phân loại `23514` thành vĩnh viễn làm giảm thử lại vô hạn. Không thêm hàng đợi. | **Đạt** |
| **§IV** | `pgconn` chỉ xuất hiện ở tầng hạ tầng. Tầng ứng dụng gọi hàm `isCheckViolation` trả `bool`, không biết Postgres tồn tại. `grant_window.go` vẫn là hàm thuần, không phụ thuộc gì. | **Đạt** |
| **§V** | Tính lại nằm trong giao dịch, sau `LockGrantsForAccount` (`grant_uc.go:238`). Giao dịch và khoá không đổi. | **Đạt** |
| **§VI** | Không có đường chi phí nào đổi. | **Đạt** |
| **§VII** | ADR 014 đã tạo. | **Đạt** |
| **§VIII** | Chỉ dùng `SQLState` và hằng số, không dùng thông điệp lỗi của phiên bản. | **Đạt** |
| **§7** | `deferred.md` đã tạo với 4 mục. | **Đạt** |

Không có vi pham nào được biện minh, nên `Complexity Tracking` vẫn để trống — điều này
đúng với nguyên tắc trong hiến pháp: bảng đó dành cho việc phá vỡ có lý do, không dành cho
việc ghi lại điều đã tuân thủ.

## Điểm nghẽn còn lại

- **Rủi ro 1 chưa đóng** cho tới khi kiểm thử tích hợp chạy trên Postgres thật. Nếu
  `errors.As` không bóc được qua GORM, Mục 3 phải đổi hướng và toàn bộ FR-009..FR-011 bị
  ảnh hưởng.
- **CHK026** chỉ đóng khi Mục 4 có bằng chứng, không phải vì R-004 đã viết ra.
- Phần sửa dữ liệu 8 tài khoản không thể chạy trước khi kế hoạch này triển khai.