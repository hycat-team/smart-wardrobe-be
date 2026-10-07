# Nghiên cứu — 027 Sửa tắc hai nguyên nhân chặn vĩnh viễn

**Ngày**: 2026-10-07 | **Đặc tả**: [spec.md](spec.md) | **Kế hoạch**: [plan.md](plan.md)

Mỗi mục ghi lại một câu hỏi mà Technical Context phải trả lời trước khi thiết kế. Không có
mục nào ở lại dạng `NEEDS CLARIFICATION` sau khi tài liệu này viết xong.

---

## R-001 — Nhận diện lỗi vi phạm ràng buộc dữ liệu qua GORM

**Câu hỏi**: FR-009 đòi phân loại lỗi ràng buộc `CHECK` là từ chối vĩnh viễn. Làm sao đọc
được SQLSTATE khi lỗi đi lên từ GORM?

**Phát hiện**: `services/ai/go.mod:16` khai `gorm.io/driver/postgres v1.6.3`, thứ chạy trên
`pgx/v5` stdlib, nên lỗi ở tầng cơ sở dữ liệu có kiểu `*pgconn.PgError` từ
`github.com/jackc/pgx/v5/pgconn`, và có trường `SQLState`. `pgx/v5` hiện là phụ thuộc
**gián tiếp** (`// indirect`) trong `services/ai/go.mod:41`; `shared/go.mod:11` đã khai nó
trực tiếp.

Tìm kiếm toàn repo: **không nơi nào** đọc `pgconn.PgError` hay `SQLState` — kể cả
`shared/infrastructure/db/`. Đây là lần đầu trong hệ thống phải phân loại lỗi theo mã
Postgres, nên không có mẫu cứu theo để làm chuẩn.

**Quyết định**: Dùng `errors.As` với `*pgconn.PgError`, và **nâng `pgx/v5` thành phụ thuộc
trực tiếp** trong `services/ai/go.mod`. Không so khớp chuỗi lỗi: thông điệp Postgres đã đổi
qua nhiều phiên bản, còn `SQLState` là mã ổn định do chính PostgreSQL định nghĩa.

Chỉ bắt **`23514` (check_violation)**, không bắt cả lớp `23` (integrity_constraint_violation).

**Lý do**: `23xxx` rộng hơn nhiều so với cái FR-009 mô tả. `23505` (unique_violation) do
`GetGrantBySourceEventID` kiểm trước rồi, và khi nó vẫn xảy ra thì **đáng** được thử lại —
một cuộc đua duy nhất theo `source_event_id` là tạm thời, đó là tình huống thử lại giải
quyết được. Bắt rộng sẽ biến một cuộc đua thành thư chết vĩnh viễn và làm FR-010 sai.

**Phương án đã cân nhắc**:

- Bắt cả lớp `23` — **loại**. Biến đua duy nhất thành thư chết; vi phạm FR-010.
- So khớp `err.Error()` chứa `"23514"` — **loại**. Nhạy với bản dịch thông điệp Postgres.
- Không phân loại, để `23514` tiếp tục là tạm thời — **loại**. Đó chính là lỗi đang sửa:
  thử lại một vi phạm ràng buộc không bao giờ thành công, hàng đợi quay vô hạn rồi chết
  thư. FR-009 nêu đích danh việc này.

**Rủi ro chưa đóng — mang sang kiểm thử**: GORM có thể bọc lỗi driver bằng `fmt.Errorf`
không dùng `%w`, khiến `errors.As` không bóc được. Nếu xảy ra, phân loại rơi về mặc định
"tạm thời" và ta quay lại đúng lỗi gốc mà không có bất kỳ tín hiệu nào báo trước. Vì vậy
`tasks.md` **bắt buộc** phải có một kiểm thử tích hợp chạy trên Postgres thật, ép vi phạm
`CHECK` thật và chứng minh `errors.As` bóc được xuyên qua GORM. Không được tin vào kiểm thử
đơn vị với `PgError` dựng tay — nó sẽ đỏ ngay cả khi tích hợp hỏng.

---

## R-002 — Mốc cắt và mốc bắt đầu phải là một, không phải hai

**Câu hỏi**: FR-004/FR-005 yêu cầu tính lại thời điểm bắt đầu **bên trong** giao dịch. Nhưng
`grant` được dựng ở `grant_uc.go:207`, **ngoài** giao dịch, và `retirePreviousGrants`
(`grant_uc.go:238`) lại **tự tính lại** cửa sổ từ sự kiện ở `grant_uc.go:279`. Tính lại ở
đâu cho hai bên cùng nhìn một giá trị?

**Phát hiện**: Đây là phát hiện cấu trúc quan trọng nhất của đặc tả này.

`retirePreviousGrants` cho nhánh `EXTENDED` gọi
`TruncateGrantsBefore(..., ComputeGrantWindow(event).EffectiveFrom, ...)`. Nó **tính lại từ
sự kiện**, không nhận mốc cắt từ bên ngoài. Nếu ta tính lại `grant.EffectiveFrom` bên
trong giao dịch mà không sửa `retirePreviousGrants`, mốc cắt và mốc bắt đầu của bản ghi mới
trở thành **hai con số khác nhau**:

- mốc cắt = `effective_from` gốc, chưa dịch;
- mốc bắt đầu = `effective_from` đã dịch 1 µs.

Lệch đúng 1 µs đó là đủ để phá vỡ một trong hai ràng buộc: nếu bản ghi cũ bị cắt **tại** mốc
chưa dịch thì cửa sổ của nó vẫn chạm vào bản ghi mới → chồng lấn, vi phạm FR-008; nếu bản
ghi mới bắt đầu **sau** mốc cắt một µs thì có một khe hở một µs giữa hai bản ghi.

**Quyết định**: Mốc đã tính lại phải là **nguồn duy nhất** cho cả hai bên. Một hàm mới
`resolveGrantWindow(ctx, event)` gọi **một lần** trong giao dịch, sau khi đã giữ khoá, trả
về `GrantWindow` đã chốt; `plan.md` truyền giá trị đó cho cả `grant.EffectiveFrom` lẫn
`retirePreviousGrants`. `retirePreviousGrants` đổi chữ ký để **nhận** mốc cắt thay vì tự
tính lại.

Còn một lý do nữa khiến cách hiện tại không dùng được: hiện tại gọi
`ComputeGrantWindow(event)` lần thứ hai. `ComputeGrantWindow` là hàm thuần
(`grant_window.go:29`), nên gọi hai lần cho cùng một sự kiện cho cùng một kết quả. Hôm nay
đúng, nhưng nó biến "cửa sổ cuối cùng" thành hai giá trị được tính ở hai thời điểm khác
nhau, và đó chính là loại lỗi mà một giao dịch phải tuyệt đối tránh.

**Phương án đã cân nhắc**:

- Chỉ dịch `grant.EffectiveFrom`, giữ nguyên mốc cắt — **loại**. Phá FR-008, tạo chồng lấn
  1 µs.
- Truyền mốc cắt dưới dạng tham số nhưng vẫn để `grant` dựng ngoài giao dịch — **loại**.
  `grant.EffectiveFrom` vẫn phải được gán lại bên trong giao dịch, tức vẫn có hai nơi quyết
  định.
- Giữ nguyên hai lần gọi `ComputeGrantWindow` và chỉ sửa mốc cắt — **loại**. Không giải
  quyết được vấn đề "quyết định phải dựa trên trạng thái dưới khoá" của FR-005.

---

## R-003 — Cột nào đáng tin để dựng sàn thứ tự

**Câu hỏi**: FR-001 phải lùi từ `occurred_at` trong ảnh chụp chính sách sang "thời điểm tạo
của chính bản ghi". Bản ghi có nhiều cột thời gian. Chọn cột nào, và làm sao một cột thời
gian kỹ thuật không thành đường tấn công?

**Phát hiện**: Bản ghi có bốn mốc thời gian: `effective_from` (giá trị nghiệp vụ),
`effective_to`, `created_at` và `updated_at`. **Không có cột `started_at`** — bảng
`aiexecutor.ai_policy_grants` trong `database/scripts/closy-db-schema.sql` không hề có cột
đó, và thực thể `AIPolicyGrant` (`domain/entities/policy.go:53`) kế thừa `CreatedAt` từ
`BaseEntity`.

Điều đáng chú ý hơn: `rejectStaleEvent` (`grant_uc.go:237`) **không** so sánh với bất kỳ cột
thời gian nào của bản ghi. Nó gọi `ExtractOccurredAt(current)`
(`grant_entitlements.go`), đọc `occurred_at` **từ vỏ bọc `envelope` trong
`policy_snapshot`**, và khi đọc không được thì **từ chối vĩnh viễn** — không có đường lùi nào
cả. Đó chính xác là khoá chết.

Trong sự cố production, `created_at` của bản ghi cũ bằng đúng `now()` lúc chạy, còn
`occurred_at` đọc được từ ảnh chụp chính sách của sự kiện lại **trước** `created_at`. Đó là
kịch bản duy nhất dẫn tới `stale_event` mà FR-001 chủ định sửa.

**Quyết định**: Dùng `created_at` làm sàn thứ tự khi nó **không nằm trong tương lai** so
với đồng hồ ứng dụng; nếu không đọc được hoặc nằm trong tương lai thì lùi sang
`effective_from`.

**Lý do về mặt bảo mật**: Hai cột đều nằm trong cùng một bảng nên đều cần quyền ghi để
giả mạo, nhưng chúng không cùng ràng buộc. `effective_from` chịu `CHECK`
`effective_to > effective_from` và phải nằm trong khoảng thời gian khách hàng thực sự
mua. `created_at` là cột kỹ thuật, không có `CHECK` nào, nên có thể bị đẩy tùy ý. Đặt mốc
sàn ở một cột không ràng buộc là biến một giá trị kỹ thuật thành đồng hồ quyền hạn: kẻ nào
đẩy `created_at` tới tương lai sẽ chặn được mọi cấp quyền sau đó của tài khoản đó tới khi
mốc đó trôi qua. Vì vậy `created_at` chỉ được tin khi nó nằm ở quá khứ — đúng cái điều kiện
mà dịch vụ AI tự tạo ra khi ghi một dòng mới.

**Phương án đã cân nhắc**:

- Chỉ dùng `effective_from` — **loại**. Sàn sẽ lệch về quá khứ so với bản ghi vừa ghi, khiến
  sự kiện hợp lệ kế tiếp phải bị dịch mỗi lần; mỗi lần dịch là một lần chạm vào dữ liệu.
- Tin `created_at` vô điều kiện — **loại**. Nêu trên.
- Dùng `updated_at` — **loại**. Nó thay đổi mỗi lần `TruncateGrantsBefore` cắt một bản ghi,
  nên nó **không** phản ánh thời điểm sự kiện được ghi và sẽ dịch sai.

Cả quyết định này được nâng thành [ADR 014](../../docs/decisions/014-relax-grant-ordering-invariant.md)
thay vì để nằm trong `spec.md`, vì nó nới lỏng một bất biến bảo mật và cần người đọc khác
nhìn thấy nó, không chỉ người viết đặc tả.

---

## R-004 — Mã lý do mới có làm đổi hình dạng phản hồi ra ngoài không

**Câu hỏi**: FR-011 yêu cầu mã lý do mới xuất hiện trong bộ đếm từ chối công khai của dịch
vụ AI. Đây có phải thay đổi hình dạng phản hồi mà client phải biết không?

**Phát hiện**: Bộ đếm nằm trong phản hồi sức khoẻ của dịch vụ AI, và khóa là
`grant_rejected_by_reason` — một đối tượng do máy sinh ra, không phải mảng liệt kê đóng.
Nhãn của nó là `string`, không phải kiểu liệt kê do máy sinh. Một khóa mới xuất hiện khi
một từ chối thực sự xảy ra; client đọc theo khoá thì không vỡ.

**Quyết định**: Đây là **thay đổi tương thích ngược**, **không** cần `frontend-guide.md`.

**Lý do**: Một tài liệu hướng dẫn frontend sinh ra cho một thay đổi không phá vỡ client là
chi phí vô nghĩa và, tệ hơn, là thứ khiến người đọc tin rằng có hợp đồng API ở đây cần được
theo dõi. Nhưng "không cần" phải dựa trên bằng chứng, không phải cảm giác — nên
`plan.md` ghi lập luận này thành văn bản và `tasks.md` có một mục kiểm tra xác nhận nhãn
là `string` tự do, không phải kiểu đóng, trước khi đóng CHK026.

**Phương án đã cân nhắc**:

- Sinh `frontend-guide.md` — **loại**. Lý do ở trên.
- Bỏ qua im lặng, không ghi lập luận — **loại**. CHK026 tồn tại chính vì cần một quyết định
  *có lý do*; im lặng là quyết định mà lần sau không ai dám sửa.

---

## R-005 — Vì sao không dùng lại đường khôi phục, và vì sao không sửa SQL

**Câu hỏi**: Đã có `POST /admin/subscription-grants/restore`. Vì sao 027 không dùng lại?

**Phát hiện**: Khoá sự kiện của đường khôi phục là `RESTORE_GRANT:<userID>:<version>`, và cả
28 tài khoản đã có khoá này trong bảng đã xử lý. Đường đó trả `already_restored` vĩnh viễn.
Sửa dữ liệu bằng SQL vi phạm [hiến pháp §I](../../.specify/memory/constitution.md) (một
chủ sở hữu cho mỗi miền dữ liệu), FR-013 của spec 024 và FR-013 của chính spec 027.

**Quyết định**: Đường dùng cho chính là
`POST /api/v1/admin/subscription-grants` với khoá chống trùng mới. Sự kiện nó sinh ra thuộc loại
`EXTENDED`, mà loại này gọi `TruncateGrantsBefore`
(`ai_policy_grant_writer.go:77`) đặt `effective_to = mốc cắt` trên bản ghi cũ. Bản ghi hỏng
vì thế không còn nằm trong cửa sổ hiệu lực, bản ghi mới mang ảnh chụp đọc được, và hai cửa
sổ chạm nhau mà không chồng lấn.

Cơ chế này **chỉ chạy được sau khi 027 đã triển khai**, vì nó phụ thuộc chính xác vào hành
vi tính lại mà 027 viết. Runbook vận hành nằm ở [runbook.md](runbook.md) và là bước sau
triển khai, không phải một thao tác độc lập.

---

## R-006 — Danh sách phát hiện ngoài phạm vi

Bốn phát hiện được nêu ở mục "Ngoài phạm vi" của đặc tả được ghi vào
[deferred.md](deferred.md) theo [hiến pháp §7](../../.specify/memory/constitution.md):
không có kiểm tra cơ học nào cản một mục ngoài phạm vi, nên nghĩa vụ của người viết là viết
nó ra. Cùng quy ước đó, phát hiện ở CHK028 — dòng mô tả sai phạm vi `stale_event` trong
`contracts/subscription-plan-changed-event.md` — được đính chính ngay trong đợt này vì nó
mô tả sai chính hành vi mà 027 sửa; để một hợp đồng mô tả sai hành vi đang được sửa là loại
nợ mà người đọc hợp đồng sẽ không phát hiện.

---

## Kết luận

Không còn mục nào ở dạng `NEEDS CLARIFICATION`. Hai phát hiện đáng chú ý nhất cho phần thiết
kế:

- **R-002** cho thấy `retirePreviousGrants` và `buildGrant` đang **mỗi cái tự tính lại cửa
  sổ của riêng nó**, nên FR-005 không thể thực hiện bằng cách vá một chỗ. Đây là thay đổi cấu
  trúc nhỏ nhưng bắt buộc.
- **R-001** mang theo rủi ro chưa đóng lớn nhất của đợt này: nếu GORM không bảo toàn chuỗi
  lỗi, việc phân loại sẽ hỏng **âm thầm**. Kiểm thử tích hợp trên Postgres thật là bắt
  buộc, không phải tuỳ chọn.