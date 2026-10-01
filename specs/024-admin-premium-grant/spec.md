# Feature Specification: Một chủ sở hữu mức quyền tính phí AI và công cụ cấp gói Premium cho quản trị viên

**Feature Branch**: `024-admin-premium-grant`

**Created**: 2026-09-29

**Status**: Draft

**Input**: User description: "trước khi tạo spec 024 hãy kiểm tra codebase lại lần nữa để đảm bảo không còn hiểu lầm nào — Thêm 1 chức năng cho admin, cấp tài khoản premium monthly cho 1 user với user id hoặc username hoặc email, admin truyền 1 trong 3; api nhận batch user và slug cần cho từng user" (bổ sung: gộp toàn bộ phần sửa chủ sở hữu mức quyền tính phí AI vào cùng spec này, mọi thứ thuộc 024)

## Thuật ngữ: tên trong văn xuôi và mã trong cơ sở dữ liệu
Tài liệu này dùng **tiếng Việt trong văn xuôi** nhưng **mã thật trong cơ sở dữ liệu** khi nói về giá trị cụ thể. Đây là bảng tra cứu nhanh, đọc một lần là đủ.
| Cách viết trong văn xuôi | Mã thật trong cơ sở dữ liệu | Ở đâu | Ghi chú |
|---|---|---|---|
| gói Premium | `premium-monthly` | `subscription.subscription_plans.slug` | **không** phải `premium` và **không** phải `premium-default` |
| gói miễn phí | `free` | `subscription.subscription_plans.slug` | — |
| chính sách Premium | `premium-default` | `aiexecutor.ai_cost_policies.code` | đây là **mã chính sách**, không phải plan slug |
| chính sách miễn phí | `free-default` | `aiexecutor.ai_cost_policies.code` | — |
> **Cảnh báo nhầm lẫn thường gặp nhất.** `premium-monthly` và `premium-default` là **hai mã khác nhau thuộc hai bảng khác nhau thuộc hai dịch vụ khác nhau**. Không được viết `premium-monthly` vào cột `code` của bảng chính sách, và không được viết `premium-default` vào trường `planSlug` của yêu cầu cấp gói.
Hệ thống **chỉ có hai plan**, đã xác nhận bằng truy vấn `tier_rank > 1 OR plan_kind > 1` trả về **0 dòng** ngày 29/09/2026. Hai cổng đã nối với nhau bằng khóa ngoại `subscription.subscription_plans.ai_cost_policy_id`, nên có thể đối chiếu hai cổng chỉ bằng cách đọc theo quan hệ này (xem FR-004).
## Bối cảnh

### Hệ thống có hai tầng độc lập

| Tầng | Nội dung | Lưu giữ ở đâu |
|---|---|---|
| **Tầng thuê bao** | Gói hiện tại, thời hạn, hạn mức AI dùng/ngày | Bảng gói đăng ký + bảng đăng ký người dùng |
| **Tầng quyền tính phí AI** | Tính năng AI nào chạy trên nhà cung cấp trả phí, giới hạn lượt, trần chi phí | Bảng mức quyền tính phí |

Tầng quyền tính phí quyết định: thao tác trò chuyện và gợi ý phối đồ có được gọi lên mô hình trả phí không, tối đa bao nhiêu lượt mỗi ngày, và tối đa chi bao nhiêu tiền trong một chu kỳ. Tầng thuê bao chỉ quyết định người dùng *được quyền* gì, không quyết định *được chạy* gì.

Hai tầng này nằm ở **hai hệ thống khác nhau** (Account và AI), nên bất kỳ thay đổi gói nào cũng phải được thông báo qua hàng đỗi sự kiện — đúng theo hiến pháp dự án: *across service boundaries, asynchronous side effects MUST use RabbitMQ with the transactional outbox*.

### Vấn đề 1 — Có hai chủ sở hữu ghi cùng một bảng, một đường đã hỏng

Hiện trạng kiểm chứng trên dữ liệu thực tế:

| Đường ghi | Cách ghi | Tình trạng |
|---|---|---|
| **Đồng bộ, từ hệ thống Account** | Ghi lệnh SQL thẳng vào bảng mức quyền tính phí, ngay trong cùng giao dịch với việc ghi lịch sử thay đổi gói | **Vi phạm hiến pháp.** Hệ thống Account đang ghi vào bảng thuộc hệ thống AI. Vi phạm kiểu này trình biên dịch không bắt được vì nó là lệnh SQL thô, không phải lời gọi hàm. |
| **Bất đồng bộ, qua hàng đỗi** | Hệ thống Account phát tín hiệu, hệ thống AI nhận và tạo bản ghi | **Hỏng.** Bản ghi được tạo ra thiếu mã định danh gói và thiếu ảnh chụp chính sách bắt buộc, nên ghi vào cơ sở dữ liệu thất bại. Nhật ký vận hành không ghi nhận bất kỳ lần ghi nào. |

Hệ quả của hai chủ sở hữu cùng tồn tại: bản ghi của cùng một tài khoản bị ghi nhiều lần với hai quy tắc khác nhau, và không có quy tắc chống trùng nào ở tầng dữ liệu — chỉ có một chỉ mục **không duy nhất** trên `(user_id, effective_from, effective_to)`, nên cơ sở dữ liệu không ngăn được hai chủ sở hữu ghi chồng nhau.

Kết quả kiểm chứng trên dữ liệu thật (16 bản ghi) cho thấy tình trạng này chưa đến mức trùng hiệu lực, nhưng vẫn hỏng dữ liệu:

| Người dùng | Số bản ghi | Trạng thái | Vấn đề phát hiện |
|---|---|---|---|
| 5 tài khoản gói miễn phí | 1 mỗi người | `active`, thời hạn không xác định | Hợp lệ — sinh ra từ migration tạo dữ liệu ban đầu, **không** phải từ hai chủ sở hữu |
| `ad11ad11-ad11-ad11-ad11-ad11ad11ad12` (Premium) | 9 | 7 `closed`, 1 `active`, 1 `future` | 6 bản ghi `closed` và cả bản ghi `active`/`future` đều có **ảnh chụp chính sách rỗng** |

Hai kết luận quan trọng được rút ra từ số liệu trên, và chúng **thay đổi phạm vi công việc**:

- **Không có tài khoản nào có từ hai mức quyền tính phí trả phí cùng hiệu lực** — 7 bản ghi `closed` là lịch sử nối tiếp nhau, không chồng lấn. Vì vậy tiêu chí SC-007 hiện đã đạt, và bước dọn dữ liệu ở GĐ3b **không cần** gộp bản ghi trùng.
- **Tài khoản Premium hiện KHÔNG có mức quyền tính phí đang hiệu lực**: bản ghi `active` có thời hạn `2026-09-11 → 2026-09-22` đã kết thúc trước ngày kiểm tra, và bản ghi `future` có `effective_from = 2026-09-22` đã nằm trong quá khứ nhưng trạng thái chưa được nâng lên `active`. Bộ dò hiện tại **không lọc theo trạng thái** nên vẫn trả về bản ghi `future` — tức là hành vi đang đúng một cách tình cờ, không phải do thiết kế.
- **8 trên 16 bản ghi có ảnh chụp chính sách rỗng** (`'{}'::jsonb`), nên SC-009 hiện **không** đạt. Nguyên nhân: đường ghi đồng bộ luôn ghi `'{}'::jsonb` một cách cố định, và đường bất đồng bộ không tạo được bản ghi nào vì thiếu trường bắt buộc.

### Vấn đề 2 — Bản ghi lịch sử thay đổi gói không đồng đều

Bản ghi lịch sử thay đổi gói là nguồn phát tín hiệu. Nhưng có ba đường thay đổi gói **không sinh bản ghi lịch sử**:

| Đường | Hệ quả |
|---|---|
| Cấp gói thủ công cho một tài khoản | Tài khoản có gói Premium nhưng **không** có mức quyền tính phí ⇒ dùng AI trả phí bị từ chối |
| Thu hồi gói thủ công | Không chấm dứt mức quyền tính phí cũ ⇒ tài khoản đã bị thu hồi vẫn tiêu hạn mức gói cũ |
| Cấp phát gói mặc định cho tài khoản chưa có gói | Tạo bản ghi gói "trần" không kèm lịch sử ⇒ **6 tài khoản đã xác minh là có bản ghi gói nhưng không có mức quyền tính phí** (không phải bản ghi bị đóng, mà là **không hề có bản ghi**) |

### Vấn đề 3 — Bộ dò tự xoá dữ liệu trong lúc đọc

Khi bộ dò phát hiện mã định danh gói trong bản ghi mức quyền không khớp gói hiện tại, nó **đóng và xoá bản ghi ngay trong lúc tra cứu**. Một sai lệch nhỏ về dữ liệu biến thành mất dữ liệu vĩnh viễn, không tự phục hồi, và thao tác ghi diễn ra trong một đường vốn lẽ ra chỉ đọc.

### Vấn đề 4 — Thiếu khoảng đệm khi tín hiệu đến trễ

Bộ dò hiện tại **trả lỗi cứng** khi không tìm thấy bản ghi mức quyền. Khi chuyển sang cơ chế bất đồng bộ đúng đắn, sẽ tồn tại vài giây tài khoản đã có gói Premium nhưng mức quyền tính phí chưa tới. Nếu giữ nguyên hành vi trả lỗi, người dùng sẽ gặp lỗi **ngay sau khi thanh toán xong** — tệ hơn cả trạng thái hiện tại.

### Vấn đề 5 — Thiếu công cụ cấp quyền thủ công

Thao tác cấp gói hiện có chỉ nhận mã định danh tài khoản, chỉ nhận mã định danh gói, mỗi lần một tài khoản. Quản trị viên không có cách xử lý các trường hợp ngoại lệ: người dùng báo lỗi không nhận được quà, người dùng hợp lệ không thuộc diện tự động, khách hàng quan trọng cần ưu đãi riêng.

**Và chính thao tác này hiện không hoạt động** vì nó thuộc một trong ba đường nêu ở Vấn đề 2.

### Quy mô ảnh hưởng đã đo

Đo lại trực tiếp trên cơ sở dữ liệu đang chạy:

| Nhóm | Số tài khoản | Thao tác AI trả phí |
|---|---|---|
| Có mức quyền tính phí **đang hiệu lực** | 5 | Chạy được, hạn mức gói miễn phí |
| Có mức quyền tính phí nhưng **đã hết hiệu lực** | 1 (Premium) | **Lỗi** — cần chạy lại đường sự kiện |
| Có bản ghi gói nhưng **không** có bản ghi mức quyền tính phí | 6 | **Lỗi** |
| **Chưa** có bản ghi gói | 68 | **Lỗi** |
| Tổng tài khoản | 82 | 75 tài khoản không dùng được thao tác AI trả phí |

Trong 68 tài khoản chưa có bản ghi gói, phần lớn là dữ liệu kiểm thử; hai tài khoản đăng ký bằng Google là tài khoản thật.

**Cần nói rõ để tránh hiểu sai:** luồng thanh toán qua cổng *đã từng* tạo ra mức quyền tính phí hợp lệ, nhờ đường ghi đồng bộ — đó là lý do gốc khiến lỗi này bị bỏ qua. Nhưng tại thời điểm đo, tài khoản Premium duy nhất **không còn** mức quyền tính phí đang hiệu lực, vì bản ghi của họ đã hết thời hạn mà không có tín hiệu nào sinh ra bản ghi kế tiếp. Nói cách khác: đường ghi đồng bộ **không** tự nó tạo ra mức quyền tính phí bền vững; nó chỉ tạo ra một mức quyền tính phí đúng một lần, rồi im lặng. Đây chính là lý do US2 được xếp P1: nếu gỡ đường đó mà chưa có đường thay thế, luồng hiện đang chạy sẽ hỏng ngay lần gia hạn kế tiếp.

### Có hai cổng hạn mức AI độc lập, và chúng đang lệch nhau

Đây là điểm dễ hiểu nhầm nhất của hệ thống. Mỗi thao tác AI bị giới hạn bởi **hai cổng tách biệt**, ở hai bảng khác nhau, bởi hai dịch vụ khác nhau:

| | Cổng 1 — hạn mức người dùng | Cổng 2 — hạn mức chi phí AI |
|---|---|---|
| Cấu hình | `subscription_plans.ai_outfit_daily_quota`, `ai_chat_daily_quota` | `ai_cost_policy_operations.max_paid_attempts_per_day` |
| Bộ đếm | `user_daily_quotas` | đếm trong `ai_usage_events` |
| Thi hành ở | Commerce | hệ thống AI |
| Vượt hạn mức thì | **Chặn yêu cầu**, trả lỗi cho người dùng | **Hạ xuống mô hình miễn phí**, vẫn chạy |
| Cổng nào chặt hơn | thường là cổng này | thường là cổng kia |

| | Cổng 1 (người dùng thấy) | Cổng 2 — `observe_only` | Cổng 2 — `strict` | Mục tiêu đã chốt |
|---|---|---|---|---|
| Gợi ý phối đồ | 15 | 30 | **không dùng** | **30** |
| Trò chuyện | 20 | 10 | **không dùng** | **10** |
**Cột `strict` là cột quan trọng nhất và dễ bỏ sót nhất.** Kiểm chứng trên mã cho thấy `max_paid_attempts_per_day` chỉ được đọc **đúng một chỗ**, ở nhánh `observe_only` (`ai_cost_repo_impl.go:90-96`). Nhánh `strict` — tức là mọi chính sách có trần chi phí, gồm cả Premium — **không hề tham chiếu trường này**; nó chỉ so `actual_cost + reserved_cost` với ngưỡng trong sổ tính phí. Nơi quyết định tuyến cũng không dùng nó: `route := plan.Route` lấy thẳng từ kế hoạch thực thi.

Dữ liệu thật xác nhận: một tài khoản `premium-monthly` thực hiện **50 lượt chat trả phí trong một ngày**, trong khi cấu hình ghi `10`. Lượt thứ 11 vẫn trả phí vì tổng tiền (khoảng 2.600đ) chưa chạm trần 25.000đ.

> **Kết luận: trường `max_paid_attempts_per_day` của `premium-default` là cấu hình chết.** Đây là lý do nó tồn tại song song với cổng 1 mà không ai nhận ra — ai cũng đọc bảng cấu hình rồi kết luận "AI có hạn 10 chat/ngày".

Điều này **đảo ngược** kết luận mà ba lượt phân tích trước đưa ra. Không có lỗi "khách thấy 20 mà dùng được 10" ở Premium: khách thấy 20 và dùng được đủ 20, chỉ bị giới hạn bởi ngân sách chung của cả tháng. Cổng 1 là nơi duy nhất thi hành mục tiêu 30/10 cho Premium.

Gói miễn phí thì ngược lại và đây là nơi còn cấu hình cần sửa: người phát triển đã chốt **5 lượt gợi ý phối đồ và 3 lượt trò chuyện mỗi ngày, không giới hạn trần cứng**, và đặt cổng 2 là `outfit` = `4`, `chat` = `3`. Lệch một lượt ở outfit là **cố ý** — xem FR-005b: hệ thống phải ghi nhận sự lệch đó trong tệp kiểm chứng FR-090, và **không** thêm bất kỳ thông điệp nào cho người dùng về phía cổng 2 — kể cả khi lượt thứ 5 bị hạ xuống mô hình miễn phí. (Các điều kiện vận hành còn lại của FR-005d là chuyện khác: cơ chế này chỉ dùng được vì gói miễn phí ở chế độ `observe_only`.)

Vì số lượt nằm trong ảnh chụp đóng băng (FR-029b), việc này cần tệp kịch bản **và** phát lại tín hiệu, không chỉ sửa cấu hình.

### Hệ quả tài chính của mục tiêu đã chốt

Dùng giá model trong cơ sở dữ liệu (`usd_to_vnd = 27500`):

| | Chat/lượt | Outfit/lượt | Chi phí/ngày | Ngày còn ngân sách | Còn lại dùng mô hình miễn phí |
|---|---|---|---|---|---|
| **Mục tiêu — cả hai cổng đều 30 outfit / 10 chat** | 41–52đ | 47–58đ | **1.820–2.260đ** | **12–15 ngày** | 15–18 ngày |
| Cổng 1 **đang chạy** — 15 outfit / 20 chat (chỉ mới hiển thị, chưa ai dùng con số này để chặn) | 41–52đ | 47–58đ | 1.525–1.910đ | 14–18 ngày | 12–16 ngày |
| Cổng 2 **đang chạy** — 30 outfit / 10 chat | 41–52đ | 47–58đ | 1.820–2.260đ | 12–15 ngày | 15–18 ngày |

> **Đọc bảng này thế nào cho đúng.** Cột "Ngày còn ngân sách" tính trên ngân sách **27.600đ**, không phải trần 30.000đ — xem dòng dưới. Con số này chỉ có ý nghĩa với **cổng 2**, vì cổng 1 chỉ đếm lượt chứ không có trần tiền; hàng "Cổng 1 đang chạy" ở đây chỉ để so sánh chi phí, không phải để dự đoán hành vi khi thi hành.
>
> **Cổng 2 hiện tại đã trùng đúng mục tiêu** (30/10), nên việc đổi cổng 2 sang `NULL` ở T020 là thu hồi một cấu hình chết chứ không làm đổi hành vi. Ngược lại, **cổng 1 mới thật sự đổi**, từ 15/20 xuống 30/10 — đó là thay đổi người dùng nhìn thấy.

Ngân sách cạn ở mức **27.600đ** chứ không phải 30.000đ, vì ngưỡng hạ mô hình là 92% trần. Nói cách khác: với mục tiêu đã chốt, một tài khoản Premium có **khoảng 12–15 ngày** mô hình trả phí rồi chuyển sang mô hình miễn phí tới hết chu kỳ. Vì thế con số theo ngày **không phải hạn mứn cam kết** mà là tốc độ — xem FR-005c.

Chi phí tối đa không đổi: trần 30.000đ vẫn chi phối. Với chiến dịch 50 tài khoản của SC-022, mức tối đa là 50 × 27.600đ = **1,38 triệuđ**, thấp hơn hạn 3,4 triệuđ. Về doanh thu, mỗi tài khoản Premium trả 59.000đ nhưng tối đa chỉ tiêu 27.600đ vào AI — khoảng **47% doanh thu** chảy vào AI; đây là đánh đổi đã được chốt từ trước và trần 30.000đ là thứ giữ cho mô hình không lỗ.

### Giá trị cấu hình đang áp dụng

| | Gói miễn phí | Gói Premium |
|---|---|---|
| Giá | 0đ | **59.000đ** |
| Thời hạn | không xác định | 30 ngày |
| Cổng 1 — số lượt người dùng thấy | 5 outfit / 3 chat → **giữ nguyên** | 15 outfit / 20 chat → **đổi thành 30 / 10** |
| Cổng 2 — số lượt/ngày | outfit 3 → **4**, chat 3 (**có tác dụng**) | **NULL** (cấu hình chết, xem ghi chú bên dưới) |
| Chế độ thực thi | `observe_only`, **không trần** (`hard_cost_micro_vnd` = NULL) | `strict`, trần **25.000đ** mỗi chu kỳ 30 ngày → **30.000đ** |
| Hạn mức token mỗi lượt | chat 3000/500, outfit 5000/300 | chat 4000/600, outfit 6000/400 |

> **Cái tên `observe_only` dễ hiểu sai.** Nó chỉ có nghĩa là **không thi hành trần chi phí**, không phải không thi hành gì. Kiểm chứng trên mã: nhánh `observe_only` vẫn đếm số lượt trả phí trong ngày và chỉ nhận khi còn dưới `max_paid_attempts_per_day` (`ai_cost_repo_impl.go:90-96`), rồi `Prepare` chuyển sang `PrepareFree` khi không được nhận (`cost/usecase.go:124-125`). Nghĩa là gói miễn phí **vẫn bị giới hạn số lượt/ngày** — đây chính là lý do cổng 2 lệch ở outfit là một lỗi thật, không phải chuyện hình thức.

> Hai giá trị `rewriter: 15` và `summary: 20` ở cổng 2 của gói Premium **không có tác dụng**: tuyến thực thi mặc định của cả hai thao tác này là `free`, nên chúng không bao giờ tiêu tốn lượt trả phí. Cần giữ nguyên khi cấu hình để không gây hiểu nhầm cho người sửa cấu hình sau này. (Ở gói miễn phí hai giá trị này là 5 và 5, cũng vô tác dụng theo cùng lý do.)

Gói miễn phí không có trần cứng, đúng như người phát triển yêu cầu. Với mục tiêu đã chốt 4 outfit + 3 chat, một tài khoản dùng hết lượng tốn khoảng **11.600đ mỗi tháng** (4 × 58đ + 3 × 52đ, × 30 ngày) — tăng so với mức 7.900đ tính trên 3/3. Mức này vẫn được chấp nhận là chấp nhận được, không cần thay đổi.

## Phạm vi và thứ tự thực thi

Toàn bộ nội dung spec này thuộc **024**. Phần cấu hình dữ liệu ở GĐ0 và phần công cụ quản trị ở GĐ7 **không** được làm trước các giai đoạn còn lại, vì chúng chỉ có ý nghĩa khi đường cấp quyền đã chạy đúng.

Yêu cầu chức năng được đánh số **theo đúng thứ tự thực thi**. Thứ tự này là bắt buộc, không phải gợi ý.

| Giai đoạn | Nội dung | Phụ thuộc | Tính chất |
|---|---|---|---|
| **GĐ0** | Cấu hình giá trị vận hành | — | Độc lập, làm trước |
| **GĐ1** | Suy giảm mềm khi thiếu mức quyền tính phí | — | **Bắt buộc làm trước mọi thay đổi cấu trúc** |
| **GĐ2** | Mọi thay đổi gói đều sinh lịch sử và tín hiệu | GĐ1 | |
| **GĐ3** | Nội dung tín hiệu đầy đủ | GĐ2 | |
| **GĐ3b** | Dọn bản ghi mức quyền tính phí thừa tồn tại | GĐ3 | Chạy tay bằng tệp kịch bản, **trước khi bắt đầu GĐ4** |
| **GĐ4** | Hệ thống AI ghi bản ghi mức quyền tính phí | GĐ3, GĐ3b | |
| **GĐ5** | Gỡ đường ghi vi phạm hiến pháp | GĐ4 | ⚠️ **Không đảo ngược được** — hoàn tác bằng git |
| **GĐ6** | Đường đọc không được thay đổi dữ liệu | GĐ5 | |
| **GĐ7** | Công cụ cấp gói cho quản trị viên | GĐ5 | Chỉ có ý nghĩa sau GĐ5 |
| **GĐ8** | Khôi phục tài khoản cũ và kiểm chứng | GĐ7 | |

**Vì sao GĐ1 phải trước:** sau khi cắt đường ghi đồng bộ, sẽ có khoảng thời gian mức quyền tính phí chưa tới nơi. Nếu chưa có suy giảm mềm, lỗi xảy ra đúng vào thời điểm người dùng vừa thanh toán. Có suy giảm mềm thì hậu quả của một sai sót ở GĐ5 chỉ là mất trải nghiệm AI trả phí, **không phải mất toàn hệ thống**.

**Vì sao GĐ5 không đảo ngược được:** xoá đường ghi đồng bộ trước khi đường sự kiện chạy được thì toàn bộ tài khoản mất mức quyền tính phí cùng lúc.

**Vì sao có GĐ3b:** sau khi gỡ đường ghi đồng bộ sẽ không còn đường nào tạo ra bản ghi mới, nên mọi dữ liệu sai còn nằm đó sẽ tồn tại vĩnh viễn. Dọn **trước** khi bắt đầu GĐ4 giúp tệp kiểm chứng ở GĐ8 báo cáo sạch ngay từ đầu, tách được lỗi cũ khỏi lỗi phát sinh trong quá trình chuyển đổi. Tệp dọn làm ba việc theo thứ tự đó chính là vì thứ tự này bảo toàn quyền: điền ảnh chụp chính sách trước (bản ghi sống được thì giữ được), đóng và xoá bản ghi chết sau, và đóng bản ghi hết hạn sai trạng thái ở cuối. Tài khoản thiếu hẳn bản ghi thì không thuộc tệp này — đó là việc của công cụ khôi phục ở GĐ8, và FR-006 bảo đảm không ai bị báo lỗi trong lúc chờ.

**Cơ chế an toàn ở GĐ5:** không dùng cờ bật/tắt. Hệ thống **MUST** kiểm chứng đầy đủ trên môi trường phát triển trước khi gộp, và khi gặp sự cố ở môi trường thật thì hoàn tác bằng git. Lý do chọn cách này: FR-006 đã khoanh vùng thiệt hại — nếu đường sự kiện hỏng, mọi tài khoản rơi về chính sách gói miễn phí chứ không mất hệ thống. Trong điều kiện đó, hoàn tác bằng git là đủ và nhanh; cờ tạm thời thì tạo nợ kỹ thuật phải gỡ sau, và kéo dài thời gian tồn tại hai chủ sở hữu — đúng thứ hiến pháp đang cấm.

---

## Clarifications

### Session 2026-09-29

- Q: Khi phát tín hiệu thay đổi gói, loại thay đổi nên lấy từ đâu? → A: **Lấy từ chính bản ghi lịch sử mà hệ thống đã sinh ra** khi áp dụng gói, không viết tay chuỗi loại thay đổi ở nơi phát tín hiệu. Bộ xử lý phía nhận có bảng rẽ nhánh theo loại thay đổi, nếu chuỗi lệch thì tín hiệu bị bỏ qua một cách âm thầm.
- Q: Loại thay đổi nào cần gửi tín hiệu? → A: **Mọi loại**, kể cả gia hạn và hết hạn. Bộ xử lý phía nhận có thể chọn không làm gì với loại không cần thiết, nhưng phía gửi không nên tự quyết định bỏ qua.
- Q: Tín hiệu cần mang theo những trường nào? → A: Mã định danh tài khoản, loại thay đổi, mã gói đích, cấp bậc quyền của gói đích, mã chính sách phí AI, thời điểm xảy ra, mã định danh gói đích, thời điểm có hiệu lực, thời điểm hết hạn, số ngày của gói, mã bản ghi lịch sử nguồn, và mã giao dịch thanh toán nguồn. Bảy trường đầu là thông tin đang gửi; các trường còn lại là thêm mới, bắt buộc vì bên nhận phải tự ghi được một bản ghi hoàn chỉnh mà không cần đọc bảng của bên gửi.
- Q: Tín hiệu cũ đã nằm trong hàng đỗi khi triển khai bản sửa thì xử lý thế nào? → A: Bộ xử lý phía nhận phải chịu được tín hiệu **thiếu** trường mới, coi như trường không xác định và ghi nhận cảnh báo, tuyệt đối không làm hỏng dữ liệu khi đọc trường đó.
- Q: Ảnh chụp chính sách nên lấy từ đâu? → A: **Dựng trong hệ thống AI từ bảng chính sách phí mà nó sở hữu**, không gửi kèm trong tín hiệu. Nhờ vậy không nhân bản dữ liệu sang bên ghi, và ảnh chụp luôn phản ánh cấu hình đang hiệu lực tại thời điểm cấp.
- Q: Khi tài khoản gia hạn, xử lý bản ghi mức quyền tính phí thế nào? → A: **Giữ mô hình hiện tại**: đóng bản ghi cũ, tạo bản ghi mới; khi kỳ mới chưa bắt đầu thì tạo bản ghi ở trạng thái chờ tới. Quy tắc này phải được chuyển nguyên vẹn sang hệ thống AI.
- Q: Khi sửa lỗi bộ dò tự xoá dữ liệu, có nên giữ cơ chế cũ để dọn dữ liệu lệch không? → A: **Không giữ**. Việc dọn dữ liệu là quyền hạn của bộ xử lý tín hiệu, không phải của đường đọc. Đường đọc chỉ được trả kết quả.
- Q: Tín hiệu có cần trường định danh phiên bản không? → A: **Có, tín hiệu MUST mang trường số phiên bản.** Phía nhận MUST từ chối nội dung có phiên bản mới hơn mức nó hiểu, còn phiên bản cũ hơn thì vẫn đọc được với quy tắc trường thiếu coi là chưa xác định. Lý do: lỗi hiện tại chính là hậu quả của việc phía nhận không biết mình đang xử lý nội dung kiểu nào — trường bị thiếu đã tạo ra đúng một bản ghi quyền tính phí hỏng trên dữ liệu thật.
- Q: Khi tài khoản đã trả tiền nhưng thiếu bản ghi mức quyền tính phí thì khắc phục bằng cách nào? → A: **Phát lại tín hiệu với loại thay đổi kích hoạt** cho các tài khoản đó, dùng lại đúng đường xử lý sẵn có, chạy lại nhiều lần được, không cần sửa dữ liệu trực tiếp.
- Q: Tài khoản mới nên nhận bản ghi mức quyền tính phí bằng cách nào? → A: Khi tài khoản được tạo, hệ thống Account cấp phát đầy đủ bản ghi gói và bản ghi lịch sử trong một giao dịch, rồi phát tín hiệu kích hoạt. Không dùng nhánh xử lý riêng.
- Q: Một tài khoản có thể có nhiều bản ghi mức quyền tính phí cùng hiệu lực không? → A: **Không**. Mỗi lần phát tín hiệu, bản ghi cũ phải được đóng trước khi tạo bản ghi mới, và cả hai bước phải là một giao dịch duy nhất.
- Q: Tín hiệu bị giao lại nhiều lần có tạo bản ghi trùng không? → A: **Không.** Bộ xử lý phải bỏ qua khi đã có bản ghi cùng mã bản ghi lịch sử nguồn.
- Q: Có bắt buộc quan sát được đường cấp quyền không? → A: **Có, bắt buộc.** Mỗi tín hiệu bị từ chối hoặc thất bại phải ghi lại mã tài khoản, lý do và thời điểm; phải có bộ đếm số tín hiệu bị từ chối theo từng lý do; phải có cảnh báo khi bộ đếm tăng. Lý do: sau khi sửa, hệ thống bắt buộc từ chối tín hiệu thiếu trường bắt buộc và tín hiệu phiên bản mới hơn mức hiểu — hai trường hợp này nếu không để lại dấu vết thì tài khoản đó sẽ lặng lẽ mất quyền AI, đúng tình trạng hiện tại, và lần này không ai phát hiện được vì không có dữ liệu.
- Q: Tài khoản không thuộc loại thường có được cấp gói không? → A: **Không**, chỉ cấp cho tài khoản người dùng thông thường.
- Q: Khi cấp gói cho tài khoản đang có đúng gói đó và thời hạn còn hiệu lực, thời hạn xử lý thế nào? → A: **Cộng dồn** — cộng thêm đúng số ngày của gói vào thời điểm hết hạn hiện tại. Đây là hành vi gia hạn tự động đang dùng, giữ nguyên để thời hạn luôn nhất quán bất kể nguồn gốc.
- Q: Danh sách cấp gói chứa tài khoản không tồn tại thì xử lý ra sao? → A: **Bỏ qua và báo riêng**, không làm hỏng các mục còn lại.
- Q: Danh sách chứa cùng một tài khoản nhiều lần? → A: **Chỉ xử lý phần tử đầu tiên**, các phần tử sau đánh dấu bỏ qua có lý do, không cộng dồn thời hạn.
- Q: Quản trị viên có phải khai báo loại mã định danh không? → A: **Không**, hệ thống tự nhận diện, kể cả trong danh sách.
- Q: Một chuỗi mã định danh khớp nhiều tài khoản thì sao? → A: **Đánh dấu phần tử đó lỗi** kèm nguyên nhân mơ hồ, không tự chọn.
- Q: Giới hạn số phần tử của một yêu cầu? → A: **Tối thiểu 1, tối đa 500**; vượt 500 thì từ chối toàn bộ yêu cầu; danh sách 0 phần tử thì từ chối.
- Q: Có bao nhiêu giao diện lập trình cho thao tác cấp gói? → A: **Chỉ một giao diện lập trình duy nhất**, nhận danh sách từ 1 đến N tài khoản. Cấp cho một tài khoản chính là danh sách một phần tử, dùng chung giao diện, cùng kết quả trả về và cùng quy tắc kiểm tra.
- Q: Quản trị viên chờ được bao lâu để thấy kết quả? → A: **Danh sách một phần tử phản hồi trong 5 giây; danh sách tối đa 500 phần tử hoàn tất trong 60 giây. Quản trị viên nhận kết quả ngay trong chính yêu cầu, không phải thao tác tra cứu lần hai.** Ngưỡng này tách bạch với ngưỡng có hiệu lực của quyền tính phí AI: quyền tính phí là bất đồng bộ nên chấp nhận chờ vài giây, còn kết quả cấp gói phải thấy ngay vì quản trị viên đang báo lại cho khách hàng.
- Q: Giới hạn tần suất gọi thao tác cấp gói? → A: **Có, tính theo quản trị viên: tối đa 60 yêu cầu/phút, trong đó tối đa 6 yêu cầu/phút có nhiều hơn một phần tử; vượt thì từ chối kèm thông báo phải chờ bao lâu.** Cần trần cứng để một tài khoản quản trị viên bị lộ thông tin đăng nhập không thể phát tán hàng nghìn yêu cầu trong một phút.
- Q: Có cần giao diện web cho công cụ này không? → A: **Không**, nằm ngoài phạm vi; hiện thực hiện qua giao diện lập trình.
- Q: Thao tác cấp gói có cần gửi email thông báo cho từng người dùng không? → A: **Có**, dùng cùng nội dung bất kể danh sách một phần tử hay nhiều phần tử. Việc gửi thất bại không được làm thất bại thao tác cấp quyền.
- Q: Công cụ khôi phục có quét tài khoản gói miễn phí đã tồn tại không? → A: **Có, quét mọi tài khoản thiếu quyền tính phí, không phân biệt gói miễn phí hay gói trả phí.** Lỗi do cách cấp quyền chỉ chạy khi gói thay đổi, nên nó ảnh hưởng tới mọi tài khoản tạo sau mốc thời gian dữ liệu quyền tính phí được đưa vào hệ thống — kể cả người dùng miễn phí chưa từng trả tiền. Công cụ MUST báo cáo rõ số lượng tài khoản đã khôi phục.
- Q: Khi gỡ đường ghi đồng bộ ở GĐ5, dùng cờ bật/tắt để chuyển đường ghi khi gặp sự cố không? → A: **Không dùng cờ.** Bắt buộc kiểm chứng đầy đủ trên môi trường phát triển trước khi gộp; khi gặp sự cố ở môi trường thật thì hoàn tác bằng git. Lý do: suy giảm mềm ở GĐ1 đã khoanh vùng thiệt hại — đường sự kiện hỏng thì mọi tài khoản rơi về chính sách gói miễn phí chứ không mất hệ thống. Trong điều kiện đó hoàn tác bằng git là đủ và nhanh, còn cờ tạm thời tạo nợ kỹ thuật phải gỡ sau và kéo dài thời gian tồn tại hai chủ sở hữu, đúng thứ hiến pháp đang cấm.
- Q: Khi thiếu mức quyền tính phí, hệ thống AI lấy chính sách thay thế từ đâu, và xử lý thế nào nếu chính sách đó cũng không tồn tại? → A: **Ưu tiên hai tầng.** Tầng một: chính sách phí của gói mặc định đang hoạt động. Tầng hai, dùng khi tầng một không dùng được: chế độ chỉ quan sát, không đặt trần chi phí, số lượt trả phí mỗi ngày bằng 0, mọi thao tác trả phí chuyển sang mô hình miễn phí. Lý do: bổ sung tầng an toàn cuối cùng để người dùng luôn có AI dùng được, kể cả khi cấu hình gói mặc định cũng hỏng — nếu không có tầng hai thì FR-006 lại rơi về lỗi cứng, đúng thứ nó sinh ra để tránh. Tầng hai phải phát cảnh báo vận hành vì đó là dấu hiệu cấu hình đang sai.
- Q: Trước khi chuyển sang cơ chế một chủ sở hữu, có cần bước dọn dữ liệu tồn tại riêng không? → A: **Có, một tệp kịch bản dọn riêng, chạy tay trước khi bắt đầu GĐ4.** Chỉ xoá bản ghi mức quyền tính phí đã đóng và không còn hiệu lực, giữ nguyên bản ghi đang hiệu lực và đang chờ tới, có phần báo cáo số lượng sẽ xoá trước khi ghi. Lý do: sau khi gỡ đường ghi đồng bộ sẽ không còn đường nào tạo ra bản ghi trùng nữa, nên nếu không dọn trước thì dữ liệu rác vẫn nằm đó và mọi đối soát về sau đều phải lọc bỏ. Dọn trước giúp tệp kiểm chứng báo cáo sạch ngay từ đầu, tách được lỗi cũ khỏi lỗi mới. Tài khoản thiếu bản ghi mức quyền tính phí là dữ liệu kiểm thử nên không dọn bằng tệp này, mà xử lý bằng công cụ khôi phục ở GĐ8.
- Q: Tài khoản do quản trị viên tạo có được cấp phát mức quyền tính phí AI đầy đủ như tài khoản đăng ký bình thường không? → A: **Có, y hệt tài khoản đăng ký bình thường.** FR-012 và FR-013 được hiểu là tuyệt đối — không có ngoại lệ. Lý do: chỉ cần một đường tạo tài khoản mà thiếu quyền tính phí là mở lại đúng lớp lỗi mà phần còn lại của spec đang xoá: người dùng ở trạng thái thiếu quyền một cách âm thầm. Ngoài ra bỏ giả định cũ trong Assumptions vốn mâu thuẫn với FR-013 — tài khoản quản trị viên tạo hiện nhận gói miễn phí một cách trễ, nên rơi đúng vào nhóm cấp phát mặc định.

- Q: Hạn mức AI của gói Premium cụ thể là bao nhiêu? → A: **30 lượt gợi ý phối đồ và 10 lượt trò chuyện mỗi ngày.** Chốt 29/09/2026. Đây là **quyết định sản phẩm của người phát triển, không phải hệ quả suy ra** — bản nháp đầu tiên ghi 30/20 cũng là một lựa chọn hợp lý, và cả hai đều làm cạn ngân sách chu kỳ (30/20 cạn sớm hơn khoảng 2 ngày). Đừng "suy lại" con số này từ trần chi phí. FR-005, SC-011
- Q: Cổng thứ hai của gói Premium có cần đặt số lượt mỗi ngày không? → A: **Không.** Ở chế độ strict trường max_paid_attempts_per_day không được đọc; nó chỉ so tổng tiền trong sổ tính phí với trần. Người phát triển chốt đặt nó thành **NULL** để không ai đọc bảng cấu hình lại hiểu nhầm. FR-005e
- Q: Gói miễn phí cho phép bao nhiêu, và có trần cứng không? → A: **5 lượt gợi ý phối đồ, 3 lượt trò chuyện, không giới hạn trần cứng.** Cổng thứ hai đặt outfit = **4** — lệch một lượt so với con số người dùng thấy là **quyết định cố ý**, nên hệ thống chỉ ghi nhận sự lệch đó trong tệp kiểm chứng FR-090 và **không** thêm bất kỳ thông điệp nào cho người dùng về phía cổng 2. FR-005b, FR-005d
- Q: Số lượt theo ngày có phải hạn mứn cam kết không? → A: **Không**, nó là tốc độ tiêu thụ trong ngân sách chu kỳ. Với mục tiêu 30/10, ngân sách cạn ở 27.600đ sau khoảng 12–15 ngày. Vì vậy màn hình hạn mức chỉ hiện **ngôn ngữ của cổng 1** — số lượt mỗi ngày — và **không** được hiện trần chi phí chu kỳ, cũng **không** được nhắc tới số tiền hay sự tồn tại của cổng 2. FR-005c
- Q: Ba thay đổi lược đồ có tách thành ba tệp migration không? → A: **Không, gộp làm một tệp** để tránh làm loãng. Đánh đổi được chấp nhận: hoàn tác trở thành all-or-nothing.
- Q: Cổng 1 và cổng 2 khác nhau về bản chất thế nào? → A: **Cổng 1 là quota công khai thật của gói** và là điều hứa với khách. **Cổng 2 là cơ chế chặn ẩn, không công khai cho người dùng.** Vì vậy khi hai cổng lệch nhau thì hệ thống ghi nhận trong tệp kiểm chứng nhưng **không** thông báo gì cho người dùng về phía cổng 2. Quyết định này gỡ bỏ yêu cầu hiển thị trần chi phí và yêu cầu cảnh báo lượt thứ 5 trước đây. FR-005b, FR-005c
- Q: Quản trị viên có sửa được cổng 2 không, hay phải dùng tệp kịch bản cơ sở dữ liệu? → A: **Có, quản trị viên được xem và sửa cổng 2 qua giao diện lập trình**, thay cho tệp kịch bản như plan ban đầu. bắt buộc có ghi nhận ký, cảnh báo khi sửa trường đã đóng băng, từ chối đặt `NULL` ở chế độ `observe_only`, và từ chối chuyển sang `strict` khi chưa có trần chi phí. FR-004b
- Q: Quản trị viên gọi giao diện lập trình quản lý cổng 2 theo đường nào, và xác thực ra sao? → A: **Bảng điều khiển quản trị gọi thẳng vào dịch vụ AI qua HTTP**, đặt trong nhóm route `/admin/ai` sẵn có nên **tự động** hưởng `r.auth.Handle()` và `RolesAuthorize(roleslug.Admin)` — không tạo đường xác thực riêng, không thêm hợp đồng gRPC. Bản ghi kiểm toán nằm ở bảng **của AI**; tuyệt đối không ghi vào `shared.admin_audit_logs` vì bảng đó thuộc mô-đun `admin` của Account. FR-004b
## User Scenarios & Testing *(mandatory)*

### User Story 1 - Hệ thống AI không bao giờ báo lỗi vì thiếu mức quyền tính phí (Priority: P1)

Người dùng vừa đăng ký mở ứng dụng và thử trò chuyện ngay. Thao tác chạy được trong hạn mức gói miễn phí. Vài giây sau, tài khoản được nâng lên gói Premium qua cổng thanh toán; ngay cả trong khoảng thời gian mức quyền tính phí chưa kịp cập nhật, thao tác vẫn chạy được chứ không báo lỗi.

**Why this priority**: Đây là điều kiện tiên quyết cho mọi thay đổi cấu trúc ở GĐ2 đến GĐ6. Nếu chưa có suy giảm mềm, mọi sai sót trong quá trình đổi chủ sở hữu sẽ biểu hiện thành lỗi cho người dùng — thường xuyên hơn và nghiêm trọng hơn cả lỗi hiện tại. Đây là yêu cầu giảm rủi ro, nhưng phải có trước khi đụng vào cấu trúc.

**Independent Test**: Dùng một tài khoản chưa có mức quyền tính phí thực hiện thao tác trò chuyện; xác nhận thao tác chạy được trong hạn mức gói miễn phí thay vì trả lỗi.

**Acceptance Scenarios**:

1. **Given** một tài khoản chưa có bản ghi mức quyền tính phí, **When** tài khoản thực hiện thao tác trò chuyện, **Then** thao tác chạy được trong hạn mức của gói miễn phí, không trả lỗi.
2. **Given** một tài khoản chưa có bản ghi mức quyền tính phí, **When** tài khoản thực hiện gợi ý phối đồ, **Then** thao tác chạy được trong hạn mức của gói miễn phí.
3. **Given** một tài khoản vừa được nâng lên gói Premium nhưng mức quyền tính phí chưa cập nhật, **When** tài khoản thực hiện thao tác AI trả phí, **Then** thao tác vẫn chạy được thay vì báo lỗi.
4. **Given** hệ thống tính phí tạm thời không hoạt động, **When** người dùng thực hiện thao tác AI, **Then** hệ thống suy giảm sang mô hình miễn phí, không báo lỗi.
5. **Given** một tài khoản có mức quyền tính phí hợp lệ, **When** kiểm tra quyền của tài khoản, **Then** kết quả trả về đúng và hệ thống **không** thay đổi bất kỳ **bản ghi mức quyền tính phí** nào trong lúc kiểm tra. Bước đặt trước ở giai đoạn sau vẫn ghi **sổ tính phí** như bình thường — hai việc này khác nhau.

---

### User Story 2 - Khách hàng đã trả tiền dùng được tính năng AI đúng hạn mức đã mua (Priority: P1)

Sau khi mua gói Premium, khách hàng thực hiện thao tác AI trả phí. Hệ thống nhận diện đúng gói của tài khoản, áp dụng đúng hạn mức token, đúng số lượt mỗi ngày và đúng trần chi phí của gói Premium. Khi hết hạn, tài khoản trở về đúng hạn mức gói miễn phí.

**Why this priority**: Đây là tiền thật đã thu. Tài khoản Premium không dùng được AI trả phí là mất đúng giá trị họ đã mua, và trải nghiệm "đã trả tiền nhưng bị từ chối" gây tổn hại niềm tin. Luồng thanh toán hiện đang hoạt động nhờ đường ghi đồng bộ; nếu gỡ đường đó mà đường sự kiện chưa hoàn chỉnh thì luồng này sẽ **hỏng theo chiều ngược lại**, từ hoạt động thành không hoạt động.

**Independent Test**: Cấp gói Premium cho một tài khoản qua cổng thanh toán, thực hiện thao tác AI trả phí, xác nhận áp dụng đúng hạn mức Premium; đợi hết thời hạn, xác nhận tài khoản trở về hạn mức gói miễn phí.

**Acceptance Scenarios**:

1. **Given** một tài khoản hoàn tất mua gói Premium qua cổng thanh toán, **When** kiểm tra mức quyền tính phí, **Then** bản ghi trỏ đúng gói Premium và đúng chính sách phí Premium.
2. **Given** một tài khoản dùng gói Premium, **When** thực hiện thao tác trò chuyện và gợi ý phối đồ, **Then** áp dụng hạn mức token và số lượt của gói Premium, cao hơn gói miễn phí.
3. **Given** một tài khoản dùng gói Premium đã vượt trần chi phí chu kỳ, **When** thực hiện thao tác AI trả phí, **Then** hệ thống tự hạ xuống mô hình miễn phí thay vì tiếp tục tiêu tiền, **và** người dùng **không** được thông báo bất cứ điều gì về phía cổng 2 — số tiền, trần chi phí, hay sự tồn tại của cổng 2; thông điệp gửi tới người dùng chỉ dùng **ngôn ngữ của cổng 1**, tức số lượt mỗi ngày và chất lượng kết quả (FR-005c).
4. **Given** một tài khoản dùng gói Premium, **When** thời hạn gói kết thúc, **Then** tài khoản trở về gói miễn phí và không còn mức quyền tính phí nào của gói Premium còn hiệu lực.
5. **Given** một tài khoản dùng gói Premium được gia hạn tự động thành công, **When** kiểm tra mức quyền tính phí, **Then** mức quyền tính phí hiệu lực liên tục, không bị gián đoạn, và bản ghi chờ tới cho kỳ kế tiếp đã được tạo sẵn.
6. **Given** một tài khoản dùng gói Premium, **When** quản trị viên thu hồi gói, **Then** tài khoản trở về gói miễn phí và quyền tính phí AI của gói Premium chấm dứt.
7. **Given** một tài khoản mua gói bậc cao hơn, **When** kiểm tra mức quyền tính phí, **Then** mức quyền tính phí chuyển sang gói bậc cao hơn và mức cũ chấm dứt.
8. **Given** nhiều tài khoản cùng thay đổi gói trong một khoảng thời gian ngắn, **When** kiểm tra mức quyền tính phí từng tài khoản, **Then** mỗi tài khoản có đúng một mức quyền tính phí đang hiệu lực.
9. **Given** một tài khoản dùng gói miễn phí đã dùng **4** lượt gợi ý phối đồ trả phí trong ngày, **When** thực hiện thao tác thứ 5, **Then** hệ thống dùng mô hình miễn phí cho lượt đó. Sự lệch giữa con số người dùng thấy (`5`) và con số thực thi (`4`) là quyết định vận hành nội bộ theo FR-005d, và người dùng **không** được thông báo bất cứ điều gì về phía cổng 2 (FR-005b, FR-005c).
10. **Given** một tài khoản dùng bất kỳ gói nào, **When** xem màn hình hạn mức AI, **Then** chỉ hiện những gì thuộc **cổng 1** — tuyệt đối **không** chứa số tiền, trần chi phí, ngưỡng hạ mô hình, hay bất kỳ giá trị nào lấy từ cổng 2 (FR-005c). Với gói miễn phí, số lượt hiện là **5** gợi ý phối đồ và **3** trò chuyện; với gói Premium là **30** và **10**.

---

### User Story 3 - Quản trị viên cấp gói Premium cho một người dùng bằng bất kỳ mã định danh nào (Priority: P2)

Khách hàng báo lỗi không nhận được quà ra mắt. Khách chỉ nhớ email hoặc tên đăng nhập, không biết mã định danh tài khoản. Quản trị viên nhập đúng chuỗi khách cung cấp cùng mã gói, hệ thống tự nhận diện, và cấp gói. Sau đó khách dùng được tính năng AI trả phí ngay với hạn mức Premium, không cần chờ đợi.

Đây là trường hợp đơn giản nhất của thao tác cấp gói: một danh sách chỉ có một phần tử. Quản trị viên không dùng giao diện khác cho trường hợp này.

**Why this priority**: Đây là nhu cầu cấp quyền cốt lõi cho mọi tình huống ngoại lệ. Thao tác cấp gói hiện có **không hoạt động** vì không sinh bản ghi mức quyền tính phí, nên tài khoản được cấp gói vẫn không dùng được AI. Công cụ này chỉ có ý nghĩa sau khi GĐ5 hoàn tất.

**Independent Test**: Cấp gói Premium cho một tài khoản bằng email của họ; xác nhận tài khoản hiển thị gói Premium với thời hạn đúng 1 tháng và dùng được tính năng AI trả phí.

**Acceptance Scenarios**:

1. **Given** một tài khoản đang dùng gói miễn phí, **When** quản trị viên gửi danh sách một phần tử chứa email của tài khoản đó cùng mã gói Premium tháng, **Then** tài khoản chuyển sang gói Premium với thời hạn kết thúc sau 1 tháng kể từ thời điểm cấp và dùng được tính năng AI trả phí.
2. **Given** một tài khoản đang dùng gói miễn phí, **When** quản trị viên thay email bằng tên đăng nhập của tài khoản đó, **Then** kết quả giống kịch bản 1.
3. **Given** một tài khoản đang dùng gói miễn phí, **When** quản trị viên dùng mã định danh tài khoản, **Then** kết quả giống kịch bản 1.
4. **Given** một tài khoản đang dùng gói miễn phí, **When** quản trị viên dùng email viết hoa chữ khác, **Then** hệ thống vẫn nhận diện đúng tài khoản đó.
5. **Given** một tài khoản không tồn tại, **When** phần tử chứa chuỗi không khớp tài khoản nào, **Then** hệ thống báo mục đó lỗi kèm nguyên nhân không tìm thấy tài khoản, và trạng thái gói không bị ảnh hưởng.
6. **Given** một tài khoản đang dùng gói miễn phí, **When** phần tử dùng mã gói không tồn tại trong danh mục, **Then** hệ thống báo mục đó lỗi với nguyên nhân mã gói không hợp lệ, và không có thay đổi nào được ghi nhận.
7. **Given** một tài khoản đang dùng gói miễn phí, **When** phần tử dùng mã của một gói đã bị vô hiệu hoá, **Then** hệ thống báo mục đó lỗi.
8. **Given** một tài khoản quản trị viên, **When** phần tử trỏ tới tài khoản đó, **Then** hệ thống báo mục đó lỗi.
9. **Given** một tài khoản hợp lệ, **When** cấp gói thành công, **Then** thay đổi được ghi nhận kèm danh tính quản trị viên, thời điểm, gói được cấp, mã gói và địa chỉ mạng.
10. **Given** một tài khoản vừa được cấp gói Premium, **When** tài khoản đăng nhập và thực hiện thao tác AI trả phí, **Then** thao tác thành công với hạn mức của gói Premium.
11. **Given** một tài khoản đang dùng gói Premium còn hiệu lực, **When** quản trị viên cấp thêm một chu kỳ Premium, **Then** thời hạn được cộng dồn đúng số ngày của gói và không bị ghi đè.
12. **Given** một danh sách có đúng một phần tử hợp lệ, **When** quản trị viên gửi yêu cầu, **Then** hệ thống xử lý y như với danh sách nhiều phần tử và trả về đúng một kết quả.

---

### User Story 4 - Quản trị viên cấp gói cho nhiều tài khoản cùng lúc (Priority: P2)

Khi chiến dịch ra mắt hoặc khi xử lý một nhóm phản hồi khách hàng, quản trị viên có danh sách dài cần ưu đãi. Quản trị viên nộp cả danh sách vào cùng giao diện lập trình đã dùng cho trường hợp một tài khoản, mỗi người một mã gói riêng, và nhận về báo cáo rõ mục nào thành công, mục nào bị bỏ qua, mục nào lỗi và vì sao.

**Why this priority**: Danh sách một phần tử đã đáp ứng nhu cầu cấp quyền cơ bản. Xử lý cả danh sách là tối ưu hoá quy trình, giá trị rõ rệt khi số lượng tài khoản tăng, nhưng không giải quyết được vấn đề mà trường hợp một tài khoản chưa giải quyết.

**Independent Test**: Nộp một danh sách gồm ba tài khoản hợp lệ và một mã không tồn tại; xác nhận ba tài khoản hợp lệ đều nhận gói và dùng được tính năng AI, mục không tồn tại được báo riêng, và kết quả khớp 4 mục đã gửi.

**Acceptance Scenarios**:

1. **Given** quản trị viên có danh sách nhiều tài khoản, **When** gửi danh sách kèm mã gói riêng cho từng tài khoản, **Then** toàn bộ tài khoản hợp lệ nhận gói tương ứng, dùng được tính năng AI trả phí, và kết quả báo lại từng mục.
2. **Given** danh sách có chứa mã không tồn tại, **When** gửi yêu cầu, **Then** các mục hợp lệ vẫn được cấp gói, mục không tồn tại được đánh dấu lỗi, và yêu cầu vẫn hoàn tất.
3. **Given** danh sách chứa cùng một tài khoản nhiều lần, **When** gửi yêu cầu, **Then** tài khoản đó chỉ được cấp gói một lần, các mục lặp lại được đánh dấu bỏ qua có lý do.
4. **Given** danh sách trộn lẫn cả ba loại mã định danh, **When** gửi yêu cầu, **Then** mỗi mục được nhận diện đúng loại và cấp gói cho đúng tài khoản.
5. **Given** danh sách chứa một mã gói không tồn tại, **When** gửi yêu cầu, **Then** mục đó được đánh dấu lỗi, các mục còn lại vẫn được cấp gói.
6. **Given** danh sách vượt quá giới hạn cho phép, **When** gửi yêu cầu, **Then** hệ thống từ chối toàn bộ yêu cầu và không cấp gói cho mục nào.
7. **Given** danh sách rỗng, **When** gửi yêu cầu, **Then** hệ thống từ chối.
8. **Given** danh sách có mục là tài khoản quản trị viên, **When** gửi yêu cầu, **Then** mục đó được đánh dấu lỗi, các mục còn lại vẫn được cấp gói.

---

### User Story 5 - Quyền lợi của gói chấm dứt đúng lúc (Priority: P3)

Sau khi gói hết hạn hoặc bị thu hồi, tài khoản phải trở về đúng quyền lợi của gói miễn phí, và các tính năng AI trả phí mà gói đó cho phép phải ngừng hoạt động.

**Why this priority**: Điều kiện cần cho niềm tin dài hạn và để bảo đảm doanh thu, nhưng người dùng chỉ cảm nhận được sau khi US2 đã hoàn tất, và lỗi hiện tại không gây thiệt hại cho khách mà ngược lại làm tổn hại doanh thu của chủ hệ thống.

**Independent Test**: Cấp gói Premium cho một tài khoản, cho thời hạn gói đó kết thúc, xác nhận tài khoản trở về hạn mức gói miễn phí và không còn dùng được tính năng AI trả phí của gói Premium.

**Acceptance Scenarios**:

1. **Given** một tài khoản dùng gói Premium, **When** thời hạn gói kết thúc, **Then** tài khoản trở về gói miễn phí và các thao tác AI trả phí vượt hạn mức gói miễn phí bị từ chối.
2. **Given** một tài khoản đã trở về gói miễn phí, **When** kiểm tra mức quyền tính phí, **Then** không còn mức quyền tính phí nào của gói Premium còn hiệu lực.
3. **Given** một tài khoản dùng gói Premium, **When** quản trị viên thu hồi gói, **Then** tài khoản trở về gói miễn phí và quyền tính phí AI của gói Premium chấm dứt.
4. **Given** một tài khoản có mức quyền tính phí cũ bị đóng, **When** có một thay đổi gói mới xảy ra, **Then** không xuất hiện tình trạng không xác định mức quyền tính phí nào đang hiệu lực.

---

### Edge Cases

- Tín hiệu đến lại nhiều lần do hàng đỗi giao lại — có tạo bản ghi mức quyền tính phí trùng không.
- Hai tín hiệu thay đổi gói của cùng một tài khoản đến theo thứ tự ngược nhau.
- Gia hạn tại thời điểm bản ghi cũ chưa được đóng do lỗi trước đó.
- Một tài khoản đã có bản ghi mức quyền tính phí cũ trỏ sang gói đã bị xoá khỏi danh mục.
- Chính sách phí được sửa đổi trong lúc bản ghi mức quyền tính phí đang hiệu lực — bản ghi cũ dùng ảnh chụp cũ hay cấu hình mới. → **Tùy theo nhóm trường**: *giới hạn quyền lợi* dùng ảnh chụp trong bản ghi, nên bản ghi cũ giữ đúng thứ khách đã mua; *trần chi phí và chống lỗ tài chính* dùng cấu hình sống, nên thay đổi có hiệu lực với mọi tài khoản ngay lập tức. Xem FR-029b.
- Một chuỗi mã định danh vừa trùng tên đăng nhập của tài khoản này vừa trùng email của tài khoản khác.
- Tài khoản đã bị xoá mềm xuất hiện trong danh sách cấp gói.
- Cấp gói hữu hạn cho tài khoản đang dùng gói trọn đời.
- Cấp gói cho tài khoản khi mã gói vừa bị vô hiệu hoá giữa lúc chuẩn bị yêu cầu và lúc thực thi.
- Một phần tử trong danh sách cấp gói là chuỗi rỗng hoặc toàn khoảng trắng.
- Một danh sách quy định cùng một tài khoản với nhiều mã gói khác nhau.
- Quản trị viên gửi yêu cầu cấp gói rồi gửi lại lần nữa do mạng lỗi.
- Hai quản trị viên đồng thời cấp gói cho cùng một tài khoản.
- Một tài khoản nhận gói thành công nhưng thông báo qua email không gửi được.
- Hệ thống tính phí tạm thời không liên lạc được khi có tín hiệu thay đổi gói phát đi.
- Yêu cầu cấp gói kéo dài quá thời gian cho phép trước khi trả kết quả — trả kết quả cho những mục đã xử lý xong, hay báo lỗi cho toàn bộ yêu cầu dù một số mục đã được cấp gói.
- Quản trị viên vô tình vượt giới hạn tần suất khi đang cố gắng cấp gói hàng loạt nhiều đợt liên tiếp.

## Requirements *(mandatory)*

### Functional Requirements

Các yêu cầu được đánh số theo thứ tự thực thi đã nêu ở phần *Phạm vi và thứ tự thực thi*.

#### GĐ0 — Cấu hình giá trị vận hành

- **FR-001**: Hệ thống MUST cho phép đặt trần chi phí chu kỳ của gói Premium thông qua một tệp kịch bản cơ sở dữ liệu đặt trong `database/scripts/`, theo cặp quy ước sẵn có: một tệp chỉ kiểm tra và in ra giá trị trước, một tệp thực thi.
- **FR-002**: Tệp kịch bản thực thi ở FR-001 MUST nằm trong một giao dịch, MUST chỉ tác động đúng chính sách phí được chỉ định, MUST có điều kiện chống ghi lặp để chạy nhiều lần không đổi kết quả, và MUST kèm chú thích tiếng Anh giải thích lý do và đơn vị quy đổi.
- **FR-003**: Tệp kịch bản kiểm tra ở FR-001 MUST NOT thay đổi bất kỳ dữ liệu nào.
- **FR-004**: Hệ thống MUST cho phép cập nhật hạn mức AI hằng ngày của một gói thông qua giao diện lập trình có sẵn, và việc cập nhật phải được ghi nhận ký. Giao diện này cập nhật được **cổng 1** (`subscription_plans`). **Cổng 2** (`ai_cost_policy_operations`) là thuộc sở hữu riêng của hệ thống AI, nên giao diện lập trình cho nó phải **nằm trong hệ thống AI**; hệ thống Account MUST NOT ghi vào bảng này. Quyền xem và sửa cổng 2 dành cho **quản trị viên** — xem FR-004b. Vì hai cổng nằm ở hai nơi khác nhau không ràng buộc chằng, hệ thống MUST cung cấp một cách kiểm chứng **đối chiếu hai cổng với nhau** cho từng gói; tệp kiểm chứng ở FR-090 phải báo cáo gói nào có hai cổng lệch nhau.
- **FR-004b**: Quản trị viên MUST có giao diện lập trình để **xem và chỉnh** cổng 2 của từng gói, thay cho cách sửa bằng tệp kịch bản cơ sở dữ liệu. Giao diện này là **HTTP**, nằm trong dịch vụ AI (chủ sở hữu bảng), đặt trong nhóm route `/admin/ai` **sẵn có** nên tự động kế thừa `r.auth.Handle()` và `shared_middleware.RolesAuthorize(roleslug.Admin)`. Hệ thống MUST **không** tạo đường xác thực riêng cho giao diện này, và MUST **không** đặt nó ngoài nhóm `/admin/ai`. Bốn ràng buộc bắt buộc:
  Đây là lời gọi từ **bảng điều khiển web của quản trị viên** vào dịch vụ AI (client đối tác ↔ dịch vụ), **không phải** trao đổi giữa hai dịch vụ backend, nên không thuộc phạm vi ràng buộc gRPC của §II. Cách làm này giống hệt module `brand` của Commerce (route `/admin/brands`) và các route `/admin/ai` sẵn có của chính mô-đun này: khai báo trong `Router.Init` của mô-đun, kế thừa middleware sẵn có, sinh Swagger theo chú thích `@Router` và được Nginx ánh xạ sẵn. Hệ thống MUST NOT tạo thêm một đường xác thực song song cho giao diện này.
  1. Mỗi lần chỉnh sửa qua giao diện lập trình phải **ghi nhận ký** gồm quản trị viên thực hiện, thời điểm, địa chỉ IP, trường cũ và trường mới. Bản ghi phải nằm ở **bảng thuộc sở hữu của hệ thống AI**, tuyệt đối **không** ghi vào `shared.admin_audit_logs` — bảng đó thuộc mô-đun `admin` của Account, và ghi vào đó từ dịch vụ AI là vi phạm §I, đúng loại vi phạm mà tính năng này đang loại bỏ. Bản ghi này **đồng thời** phải được phát đi để xuất hiện ở màn hình kiểm toán tập trung của Account, theo cơ chế outbox quy định ở FR-004c - FR-004e. **Ngoại lệ duy nhất** cho yêu cầu ghi nhận ký là thay đổi dữ liệu trong tệp migration: nó là việc một lần, có vết trong lịch sử mã nguồn và trong `goose_db_version`, nên không bắt buộc ghi vào bảng kiểm toán — nhưng phải được nêu tường minh trong tệp migration và **không được** dùng tệp migration để sửa định kỳ.
  2. Khi sửa `max_paid_attempts_per_day`, phải trả về cho quản trị viên một **cảnh báo bắt buộc**: trường này thuộc nhóm quyền lợi đã đóng băng trong ảnh chụp, nên sửa xong **không** có tác dụng lên bản ghi đang tồn tại cho tới khi chạy phát lại tín hiệu theo FR-029c. Cảnh báo phải nêu rõ còn bao nhiêu bản ghi, trong đó có bao nhiêu bản ghi `effective_to IS NULL` sẽ không bao giờ tự làm mới.
  3. Phải **từ chối** thao tác đặt `NULL` cho một chính sách ở chế độ `observe_only`, kèm lý do theo FR-005e. Chỉ cho phép `NULL` ở chế độ `strict`. Giao diện lập trình phải có cách **tường minh** để yêu cầu đặt `NULL`, vì giá trị `null` trong JSON và trường bị bỏ qua là **không phân biệt được** khi ánh xạ vào con trỏ nullable trong Go — dùng một cờ boolean riêng (`clearMaxPaidAttempts`) để biểu diễn ý định "đặt thành `NULL`". Gửi **cả** giá trị lẫn cờ này trong cùng một yêu cầu là **mâu thuẫn** và phải bị từ chối, vì nếu không thì ý định của người gọi không xác định.
  4. Phải **từ chối** chuyển một chính sách sang `strict` khi `hard_cost_micro_vnd` còn trống, kèm lý do: như vậy sẽ không còn gì chặn cả.
- **FR-004c**: Mỗi lần quản trị viên chỉnh cổng 2 qua giao diện lập trình phải tạo **hai bản ghi trong cùng một giao dịch** (FR-015): (1) bản ghi kiểm toán tại `aiexecutor.ai_cost_policy_audit_logs` như FR-004b(1), và (2) một bản ghi `shared.outbox_records` với `topic = admin.audit.log`. Hệ thống AI MUST **không** ghi thẳng vào `shared.admin_audit_logs`; bản ghi tại màn hình kiểm toán tập trung phải do **phía nhận** tạo ra từ event. Nếu không tạo được bản ghi outbox thì **phải rollback cả bản ghi kiểm toán** — chấp nhận bản ghi kiểm toán mà không có event sẽ tạo ra trạng thái nửa vời, và người đọc màn hình tập trung sẽ không thấy thay đổi đã xảy ra.
- **FR-004d**: Payload event cho FR-004c phải dùng **đúng** cấu trúc `dto.RecordAuditLogInput` mà Account đã dùng cho `admin.audit.log`: `actorUserId`, `action`, `targetType`, `targetId`, `payload`, `requestIp`. Với thao tác chỉnh cổng 2: `action` phải là `ai.cost_policy_update` và `targetType` phải là `ai_cost_policy` — theo đúng quy ước **ký hiệu chấm** cho `AuditAction` và **snake_case** cho `AuditTargetType` đang có trong `shared/domain/constants/adminaudit/` (`ai.pricing_update`, `ai_model_config`), và phải bổ sung **hai hằng số mới** `AICostPolicyUpdate` cùng `AICostPolicy` vào đúng package đó. Trường `payload` phải chứa `policyCode`, `operation`, `fieldName`, `oldValue`, `newValue` để màn hình kiểm toán hiển thị được **trường nào đổi từ giá trị nào sang giá trị nào**. Dùng lại đúng cấu trúc này là bắt buộc — phát một payload tự chế, hoặc dùng ký hiệu không theo quy ước, sẽ khiến consumer hiện có không đọc được và bản ghi rơi vào DLQ.
- **FR-004e**: Consumer `admin.audit.log` ở phía Account phải xử lý event của FR-004c **chính xác một lần** về mặt tác dụng phụ: khi giao dịch chỉnh cổng 2 bị lặp lại do outbox thử lại, màn hình kiểm toán không được ghi hai dòng cho cùng một thay đổi. Cơ chế chống trùng phải là **kiểm tra trước khi ghi**: trước khi thêm bản ghi, kiểm tra đã có bản ghi trùng **định danh ổn định** của thay đổi chưa (kết hợp `actorUserId` + `policyCode` + `operation` + `fieldName` + `newValue` + mốc thời gian); nếu đã có thì bỏ qua và coi như đã ghi. Vì bảng đích không có ràng buộc duy nhất tự nhiên nên phải chọn cơ chế này, và **không** dùng khoá duy nhất: hai thay đổi hợp lệ có thể trùng tên trường và giá trị nhưng khác thời điểm, khoá duy nhất sẽ nuốt mất thay đổi thứ hai. Giới hạn của cơ chế này phải được ghi rõ trong kiểm thử: hai event đến gần như đồng thời vẫn có thể tạo ra hai dòng, và đây là mức chấp nhận được vì cùng một quản trị viên hiếm khi sửa cùng một trường cùng giá trị trong cùng một khoảnh khắc.

- **FR-005**: Trần chi phí chu kỳ của gói Premium MUST bằng 30.000đ cho mỗi chu kỳ 30 ngày. **Mốc tính chu kỳ**: chu kỳ 30 ngày tính từ `effective_from` của bản ghi mức quyền — tức thời điểm gói có hiệu lực — **không** phải lịch tháng dương. Cụ thể `period_start` và `period_end` của bản ghi sổ chi phí phải trùng khớp chính xác với `effective_from` và `effective_to` của bản ghi mức quyền tương ứng. Quy tắc này đã được kiểm chứng trên dữ liệu thực tế ngày 29/09/2026: bản ghi có hiệu lực `2026-09-22 → 2026-10-22` ứng với `period_start = 2026-09-22`, `period_end = 2026-10-22`, `period_index = 0`. Khi gia hạn, một chu kỳ mới bắt đầu ngay khi chu kỳ cũ kết thúc, kéo dài `period_index`. **Cổng 1** của gói Premium (`premium-monthly`) MUST bằng **30 lượt gợi ý phối đồ** và **10 lượt trò chuyện** mỗi ngày. **Cổng 2** của `premium-default` ở chế độ `strict` là **NULL** cho cả bốn thao tác, và hệ thống **MUST NOT** dùng số lượt/ngày để quyết định bất cứ thao tác nào của chính sách này — chỉ **tổng tiền trong phép tính chi phí chu kỳ** mới được tính, theo FR-005e. Người phát triển đã chốt mục tiêu cổng 1 ngày 29/09/2026. Ghi chú khối kỹ thuật: hai con số **không phải hợp đồng cam kết** - chúng là **tốc độ tiêu thụ trong ngân sách chi phí AI** của cả tháng, nên hệ thống bắt buộc phải nói rõ điều này với người dùng theo FR-005c. *(FR-005e chốt sau và thay thế câu "cổng 2 MUST bằng đúng hai con số đó": ở chế độ `strict`, cột `max_paid_attempts_per_day` là **cấu hình chết** vì nhánh đó chỉ so chi phí với trần, dữ liệu đã đặt NULL cho cả bốn thao tác, và giao diện cập nhật còn từ chối ghi giá trị vào chính sách `strict`. Câu cũ mô tả một cấu hình không tồn tại. Quyết định ngày 29/09/2026 về cổng 1 giữ nguyên toàn bộ — đó là cam kết với khách và là con số duy nhất còn được áp dụng.)*
- **FR-005d**: Gói miễn phí MUST cho phép **5 lượt gợi ý phối đồ** và **3 lượt trò chuyện** mỗi ngày, và MUST **không** áp trần chi phí cứng. Cổng 1 đã đúng sẵn. Cổng 2 đặt `outfit` = `4` và `chat` = `3`; vì số lượt nằm trong ảnh chụp đóng băng nên việc này bắt buộc phải đi kèm phát lại tín hiệu theo FR-029c — sửa cấu hình một mình là **không đủ**. Chốt ngày 29/09/2026. Đây là quyết định vận hành nội bộ: người dùng **không** được thông báo về phía cổng 2 (FR-005b, FR-005c). Hai điều kiện vận hành còn lại: cơ chế này chỉ dùng được vì gói miễn phí ở chế độ `observe_only`, nếu đổi sang `strict` mà không đặt trần chi phí thì sẽ **không còn gì chặn cả** — số lượt sẽ thành cấu hình chết giống Premium.
- **FR-005e**: Hệ thống phải coi `max_paid_attempts_per_day` là **cấu hình chết** với mọi chính sách ở chế độ `strict`, vì nhánh đó chỉ so chi phí với trần. Tệp kiểm chứng ở FR-090 phải báo cáo riêng hai cột này và **không** được dùng giá trị của chế độ `strict` để kết luận về giới hạn thực tế. Người phát triển đã chốt đặt trường này của `premium-default` thành **NULL** để không ai đọc bảng cấu hình lại hiểu nhầm Premium bị giới hạn 10 chat/ngày. Vì vậy: cột MUST cho phép NULL; giá trị NULL MUST được hiểu là "chính sách này không dùng giới hạn số lượt"; và một chính sách ở chế độ `observe_only` mà có NULL là **cấu hình sai** — hệ thống phải ghi cảnh báo và áp tầng suy giảm 2 (0 lượt trả phí) thay vì âm thầm bỏ qua.
- **FR-005b**: Trước khi coi GĐ0 là xong, phải xác định rõ **cổng nào là cổng chặt** với từng thao tác. Cổng 1 là **quota công khai thật** của gói và là điều hứa với khách; cổng 2 là **cơ chế chặn ẩn, không công khai cho người dùng** (FR-005c). Vì vậy khi hai cổng lệch nhau thì hệ thống phải ghi nhận sự lệch đó trong tệp kiểm chứng FR-090, nhưng MUST **không** thêm bất kỳ thông điệp nào cho người dùng về phía cổng 2. Sự lệch giữa cổng 1 và cổng 2 là quyết định vận hành nội bộ, không phải thứ cần công bố. *(Ngoại lệ "phải công bố cho người dùng" trước đây đã bị gỡ theo lựa chọn của người phát triển ngày 29/09/2026.)*
- **FR-005c**: Cổng 2 là **cơ chế chặn nội bộ, không công khai cho người dùng**. Hệ thống MUST NOT hiển thị trần chi phí AI của chu kỳ, ngưỡng hạ mô hình, hay bất kỳ giá trị nào lấy từ cổng 2 cho người dùng. Thông điệp gửi tới người dùng MUST chỉ dùng **ngôn ngữ của cổng 1** — số lượt mỗi ngày và chất lượng kết quả — và MUST NOT nhắc tới số tiền, trần, hay sự tồn tại của cổng 2. *(Yêu cầu hiển thị trần chi phí trước đây đã bị gỡ sau khi người phát triển chốt cổng 2 là nội bộ.)*

#### GĐ1 — Suy giảm mềm khi thiếu mức quyền tính phí

- **FR-006**: Khi không tìm thấy bản ghi mức quyền tính phí cho tài khoản, hệ thống thống phí AI MUST áp dụng chính sách thay thế theo thứ tự ưu tiên nêu ở FR-007 thay vì trả về lỗi.
- **FR-007**: Thứ tự ưu tiên chính sách thay thế: **(1)** chính sách phí của gói mặc định đang hoạt động; **(2)** nếu gói mặc định không tồn tại hoặc đã bị vô hiệu hoá, dùng chế độ chỉ quan sát, **không đặt trần chi phí**, và đặt số lượt trả phí mỗi ngày bằng **0** để mọi thao tác trả phí chuyển sang mô hình miễn phí. Tầng thứ hai MUST tồn tại để bảo đảm người dùng luôn có AI dùng được, kể cả khi cấu hình gói mặc định cũng hỏng.
- **FR-008**: Áp dụng tầng thứ hai ở FR-007 MUST phát cảnh báo vận hành, vì đó là dấu hiệu cấu hình gói mặc định đang sai.
- **FR-009**: Suy giảm ở FR-006 MUST áp dụng cho **mọi** thao tác AI, không riêng thao tác trả phí, vì bản ghi mức quyền tính phí cũng quyết định thao tác nào được chạy trên mô hình miễn phí.
- **FR-010**: Hệ thống MUST ghi nhận mỗi lần áp dụng suy giảm ở FR-006, gồm mã tài khoản, thời điểm, tầng suy giảm đã dùng và lý do, để không trở thành sự cố âm thầm.

#### GĐ2 — Mọi thay đổi gói đều sinh lịch sử và tín hiệu

- **FR-011**: Bản ghi lịch sử thay đổi gói MUST là nguồn duy nhất phát tín hiệu thay đổi gói.
- **FR-012**: Mọi thao tác thay đổi gói của một tài khoản — bao gồm mua qua cổng thanh toán, mua bằng ví, gia hạn tự động, hết thời hạn, cấp thủ công, và thu hồi thủ công — MUST sinh bản ghi lịch sử thay đổi gói. Hệ thống MUST NOT có trường hợp nào thay đổi gói mà không sinh bản ghi lịch sử.
- **FR-013**: Cấp phát gói mặc định cho một tài khoản chưa có bản ghi gói MUST sinh bản ghi lịch sử thay đổi gói, MUST NOT tạo bản ghi gói trần không kèm lịch sử. Điều này áp dụng **giống nhau** cho mọi tài khoản, kể cả tài khoản do quản trị viên tạo — hệ thống MUST NOT có bất kỳ đường tạo tài khoản nào dẫn tới tài khoản thiếu mức quyền tính phí AI.
- **FR-014**: Việc tạo bản ghi gói hàng loạt cho nhiều tài khoản trong một thao tác đọc MUST sinh bản ghi lịch sử cho từng tài khoản, tương tự FR-011.
- **FR-015**: Mỗi lần sinh bản ghi lịch sử thay đổi gói MUST phát tín hiệu thay đổi gói, và việc ghi bản ghi lịch sử cùng ghi bản ghi phân phối tín hiệu phải nằm trong **cùng một giao dịch**.
- **FR-016**: Thao tác cấp gói thủ công và thao tác thu hồi gói thủ công MUST dùng cùng bộ quy tắc chuyển trạng thái với các thao tác mua và gia hạn, MUST sinh bản ghi lịch sử, và MUST phát tín hiệu.
- **FR-017**: Tín hiệu thay đổi gói MUST tồn tại độc lập với việc hệ thống AI có đang hoạt động hay không: khi hệ thống AI tạm thời không liên lạc được, tín hiệu phải được giữ lại và xử lý lại sau, MUST NOT bị mất.
- **FR-018**: Mỗi bản ghi lịch sử thay đổi gói MUST có một định danh duy nhất, và định danh này là khoá chống xử lý trùng ở phía nhận.
- **FR-019**: Loại thay đổi trong tín hiệu MUST được lấy từ bản ghi lịch sử đã sinh ra, MUST NOT được viết tay tại nơi phát tín hiệu.
- **FR-020**: Hệ thống AI MUST có khả năng xử lý tín hiệu cũ thiếu trường mới bằng cách coi trường thiếu là chưa xác định, ghi nhận cảnh báo, và MUST NOT dừng hoạt động vì lý do này.
- **FR-021**: Tín hiệu MUST mang trường số phiên bản nội dung. Hệ thống AI MUST từ chối tín hiệu có phiên bản **mới hơn** mức nó hiểu, ghi nhận cảnh báo, và MUST NOT ghi dữ liệu cho nội dung nó không hiểu.
- **FR-022**: Hệ thống AI MUST xử lý được tín hiệu có phiên bản **cũ hơn** mức tối đa, theo quy tắc trường thiếu ở FR-020.

#### GĐ3 — Nội dung tín hiệu đầy đủ

- **FR-023**: Tín hiệu thay đổi gói MUST chứa các trường: số phiên bản nội dung, mã định danh tài khoản, loại thay đổi, mã định danh gói đích, mã gói dạng chữ, cấp bậc quyền của gói đích, mã chính sách phí AI áp dụng cho gói đích, thời điểm xảy ra, thời điểm có hiệu lực, thời điểm hết hạn của gói, số ngày của gói, định danh bản ghi lịch sử nguồn, và định danh giao dịch thanh toán nguồn.
- **FR-024**: Tín hiệu MUST NOT chứa dữ liệu mà bên nhận có thể tự tra cứu từ bảng nó sở hữu, cụ thể là ảnh chụp chính sách phí.
- **FR-025**: Mở rộng nội dung tín hiệu theo FR-021 MUST giữ nguyên tên và ý nghĩa các trường đang gửi, để tín hiệu cũ vẫn được xử lý đúng.
- **FR-026**: Thời điểm và số ngày trong tín hiệu MUST phản ánh trạng thái gói tại thời điểm phát, để bên nhận tính được mốc bắt đầu, mốc kết thúc và mốc bắt đầu của kỳ chờ tới mà không cần đọc lại trạng thái gói. FR-023 quy định **trường nào** phải có; FR-026 quy định **giá trị của các trường thời gian** phải phản ánh cái gì.

#### GĐ4 — Hệ thống AI là chủ sở hữu duy nhất

- **FR-027**: Bảng bản ghi mức quyền tính phí AI MUST chỉ có **một chủ sở hữu duy nhất** là hệ thống AI. Hệ thống khác MUST NOT ghi vào bảng này dưới bất kỳ hình thức nào, kể cả lệnh cơ sở dữ liệu viết thẳng.
- **FR-028**: Hệ thống AI MUST tạo bản ghi mức quyền tính phí từ tín hiệu thay đổi gói, với đầy đủ: mã định danh gói, mã gói dạng chữ, cấp bậc quyền, và ảnh chụp chính sách phí.
- **FR-029**: Ảnh chụp chính sách phí phải gồm **hai phần**. Phần `envelope` lưu thông tin của tín hiệu đã tạo ra bản ghi: số hiệu bản, loại thay đổi, thời điểm xảy ra, thời điểm có hiệu lực, thời điểm hết thạn, số ngày của gói, và định danh bản ghi lịch sử nguồn. Phần `entitlements` lưu **giới hạn quyền lợi** tại thời điểm cấp: số lượt trả phí tối đa mỗi ngày, và giới hạn token vào/ra cho cả tuyến mặc định lẫn tuyến giảm. Cả hai phần phải phản ánh cấu hình **đang hiệu lực** tại thời điểm xử lý tín hiệu, và phải được dựng bởi chính hệ thống AI từ bảng chính sách phí mà nó sở hữu.
- **FR-029b**: Ranh giới giữa ảnh chụp và cấu hình sống phải là **một quy tắc duy nhất**: ảnh chụp quyết định cái khách hàng đã mua, cấu hình sống quyết định cái hệ thống chịu rủi ro. Vì vậy khi thi hành, hệ thống AI MUST đọc `entitlements` trong ảnh chụp cho **số lượt trả phí tối đa mỗi ngày** và **giới hạn token**, nhưng MUST đọc cấu hình **sống** cho **trần chi phí chu kỳ**, **chế độ thực thi**, ngưỡng nén, số yêu cầu trả phí chưa xác định, thời gian giữ yêu cầu chưa xác định, **số ngày của một chu kỳ**, **cờ đang hoạt động của chính sách**, **cờ đang bật của từng thao tác**, và **tuyến thực thi**. Lý do phải tách: giữ hạn mức trong ảnh chụp sẽ **âm thầm lấy đi** thứ khách hàng đã trả tiền; còn giữ trần chi phí trong ảnh chụp sẽ khiến việc hạ trần **không có tác dụng lên tài khoản nào đang có bản ghi** — tệ nhất là bản ghi gói không thời hạn, vốn chỉ được tạo lại khi tài khoản đổi gói, tức là có thể vĩnh viễn.
- **FR-029c**: Hệ thống MUST ghi nhận mỗi lần thay đổi trường **quyền lợi** trong bảng chính sách, và thay đổi phải phải kèm **phát lại tín hiệu khách hàng** cho mọi tài khoản đang bản ghi, kể cả bản ghi cũ giữ nguyên giới hạn cũ. Thứ tự bắt buộc: **(1)** sửa giá trị ở bảng chính sách qua giao diện lập trình của hệ thống AI theo FR-004b, hoặc bằng tệp kịch bản cơ sở dữ liệu khi chưa có giao diện — với tệp kịch bản, phần `-- +goose Up` phải ghi kèm lý do trong chú thích **tiếng Anh**; **(2)** chạy tệp kiểm chứng ở FR-090 để chắc dữ liệu đúng; **(3)** chạy công cụ phát lại tín hiệu ở FR-088 với `dryRun: true` rồi `dryRun: false`. Bước **(3)** bao gồm **cả bốn thao tác** `chat`, `outfit`, `summary` và `rewriter`, và bỏ qua đồng nghĩa mọi tài khoản đang bản ghi vẫn giữ giới hạn cũ. Hệ thống MUST NOT yêu cầu phát lại cho trường **chống lỗ tài chính**, vì những trường đó có hiệu lực ngay lập tức với mọi tài khoản.
- **FR-030**: Ảnh chụp chính sách phí MUST NOT để trống, và phần `entitlements` MUST chứa giới hạn của **tất cả** bốn thao tác `chat`, `outfit`, `summary` và `rewriter`. Ảnh chụp thiếu một trong bốn thao tác coi như không đạt và phải được dựng lại.
- **FR-031**: Hệ thống AI MUST đóng bản ghi mức quyền tính phí đang hiệu lực trước khi tạo bản ghi mới, và hai bước phải là một giao dịch duy nhất sao cho không có thời điểm nào tài khoản không có mức quyền tính phí hiệu lực. Hệ thống AI MUST thực hiện chuỗi kiểm tra-bằng-ghi theo thứ tự chờ-thứ-tự-rồi-ghi **dưới một khoá theo tài khoản giữ trong suốt giao dịch**, và phải kiểm tra lại tình trạng trùng cũng như điều kiện thứ tự ở FR-033 **bên trong** khoá đó. Lý do: hai bước nằm trong cùng giao dịch là chưa đủ nếu quyết định được đưa ra trước khi khoá được giữ — mỗi giao dịch sẽ tự quyết trên một trạng thái mà giao dịch kia đã thay đổi, và khoá sẽ chỉ còn tác dụng trì hoãn việc ghi chứ không còn quyết định kết quả. FR-034 không cho phép ràng buộc duy nhất ở tầng cơ sở dữ liệu, nên tính đúng đắn phải đến từ khoá này.
- **FR-032**: Khi loại thay đổi là gia hạn và kỳ mới chưa bắt đầu, hệ thống AI MUST tạo bản ghi ở trạng thái chờ tới với mốc bắt đầu tính từ thời điểm hết hạn của gói trừ đi số ngày của gói; khi loại thay đổi không phải gia hạn, hệ thống AI MUST xoá các bản ghi đang chờ tới và đóng các bản ghi đang hiệu lực của tài khoản đó.
- **FR-033**: Hệ thống AI MUST bỏ qua tín hiệu mà đã có bản ghi mức quyền tính phí cùng định danh bản ghi lịch sử nguồn, và việc bỏ qua MUST NOT gây tác dụng phụ. Ngoài ra, hệ thống AI MUST từ chối và ghi cảnh báo mọi tín hiệu có thời điểm xảy ra **không mới hơn** thời điểm xảy ra đã ghi trong **bản ghi mới nhất** của tài khoản đó, bất kể bản ghi đó đang hiệu lực, đang chờ tới hay đã đóng. Phạm vi này rộng hơn "bản ghi đang hiệu lực" một cách có chủ đích: một lần gia hạn sớm tạo ra bản ghi **chờ tới** theo FR-032, và bản ghi chờ tới đó vô hình với điều kiện "đang hiệu lực". Nếu chốt chặn chỉ nhìn bản ghi đang hiệu lực thì một tín hiệu cũ đến sau lần gia hạn sẽ không thấy gì để so sánh, cho là hợp lệ, và được ghi thêm — khiến tài khoản có hai bản ghi đang hiệu lực chồng lấn, vi phạm FR-034. Bản ghi **đã đóng** cũng phải nằm trong phạm vi so sánh vì nó vẫn mang thời điểm xảy ra mới nhất từng áp dụng cho tài khoản; bỏ qua nó sẽ để tín hiệu đi trễ hồi sinh mức quyền đã bị thu hồi. Lý do: hai tín hiệu có thể đến sai thứ tự — hàng đỗi giữ thứ tự trong điều kiện bình thường, nhưng một tín hiệu đã vào hàng chết rồi được phát lại sẽ phá thứ tự đó; nếu không có chốt chặn, tín hiệu gia hạn đến sau sẽ hồi sinh mức quyền tính phí đã bị thu hồi. Để chốt chặn được, mỗi bản ghi mức quyền tính phí phải lưu kèm thời điểm xảy ra của tín hiệu đã tạo ra nó, trong ảnh chụp chính sách.
- **FR-034**: Hệ thống MUST bảo đảm mỗi tài khoản có nhiều nhất một bản ghi mức quyền tính phí **đang hiệu lực** tại một thời điểm. Việc bảo đảm thực hiện ở **tầng ứng dụng** theo FR-031, và hệ thống MUST NOT thêm ràng buộc duy nhất ở tầng cơ sở dữ liệu cho cột trạng thái, vì một tài khoản cần giữ nhiều bản ghi `active` theo thứ tự thời gian.
- **FR-035**: Giao diện lập trình tạo bản ghi mức quyền tính phí dành cho quản trị viên phải tạo được bản ghi **đầy đủ mọi trường bắt buộc** — gồm mã định danh gói, mã gói dạng chữ, cấp bậc quyền, ảnh chụp chính sách không rỗng, và trạng thái — **hoặc phải bị gỡ bỏ nếu không làm được**. Hệ thống MUST NOT giữ lại giao diện tạo bản ghi trực tiếp chỉ vì nó đã tồn tại, vì mỗi giao diện như vậy là một chủ sở hữu ghi thứ hai trên bảng của hệ thống AI.
- **FR-036**: Hệ thống AI MUST phát cảnh báo khi nhận tín hiệu chứa `aiCostPolicyId` không tồn tại trong bảng `aiexecutor.ai_cost_policies` thuộc sở hữu của chính dịch vụ AI, thay vì âm thầm bỏ qua. Đối tượng được tra là **chính sách phí AI** mà tín hiệu mang theo, **không** phải danh mục gói: hệ thống AI **MUST NOT** tra danh mục gói, kể cả khi tín hiệu có mã gói dạng chữ `toPlanCode`, vì bảng danh mục gói thuộc miền dữ liệu của dịch vụ Account và đọc bảng đó sẽ vi phạm hiến pháp §I. Vì vậy `toPlanCode` trong tín hiệu MUST chỉ được dùng cho hai việc: ghi vào bản ghi mức quyền tính phí, và kiểm tra nó là **trường bắt buộc** theo FR-023 — tín hiệu thiếu `toPlanCode` bị từ chối với mã lý do `missing_required_field`. Nó MUST **không** được dùng để **tra danh mục gói**: không để xác định chính sách phí, không để hỏi mã gói có tồn tại, và cũng không để suy ra quyền lợi của gói đích. Khi không tìm thấy chính sách, hệ thống AI MUST **từ chối** tín hiệu chứ không được áp dụng với chính sách đoán bừa, và phải để lại dấu vết truy vết được: mã lý do `ai_cost_policy_not_found` trên bộ đếm của FR-087 cùng một dòng log mức cảnh báo mang `reason` và `user_id`. *(Câu trước đây của FR-036 yêu cầu cảnh báo khi "mã gói đích không tồn tại trong danh mục" — một kiểm tra mà dịch vụ AI không và không được thực hiện. Yêu cầu cốt lõi được giữ nguyên: không được âm thầm bỏ qua; chỉ đổi đối tượng được tra từ danh mục gói sang chính sách phí AI. Đây là quyết định của người phát triển, chốt trong T159 và T172.)*

#### GĐ5 — Gỡ đường ghi vi phạm

- **FR-037**: Cơ chế chiếu bản ghi mức quyền tính phí từ hệ thống Account sang bảng thuộc hệ thống AI MUST bị gỡ bỏ hoàn toàn.
- **FR-038**: Sau khi gỡ, bản ghi lịch sử thay đổi gói MUST chỉ ghi bản ghi lịch sử và phát tín hiệu, MUST NOT ghi bất kỳ bản ghi nào thuộc hệ thống AI.
- **FR-039**: Cơ chế chiếu bị gỡ ở FR-037 có chứa quy tắc nghiệp vụ về việc tính mốc bắt đầu của bản ghi chờ tới khi gia hạn; quy tắc đó MUST được chuyển sang hệ thống AI theo FR-032 trước khi cơ chế bị gỡ.
- **FR-040**: Hệ thống MUST **không** có cờ bật/tắt để chuyển giữa hai đường ghi bản ghi mức quyền tính phí. Cơ chế an toàn ở giai đoạn này là kiểm chứng trước khi gộp và hoàn tác bằng git khi gặp sự cố, do FR-006 đã giới hạn thiệt hại ở mức suy giảm về gói miễn phí.
- **FR-041**: Trước khi gỡ cơ chế chiếu ở FR-037, hệ thống MUST được kiểm chứng trên môi trường phát triển rằng sau khi gỡ, mọi loại thay đổi gói đều tạo được bản ghi mức quyền tính phí đầy đủ qua đường sự kiện, và tệp kịch bản kiểm chứng ở FR-088 báo cáo 0 tài khoản có bản ghi gói mà thiếu bản ghi mức quyền tính phí.

#### GĐ6 — Đường đọc không được thay đổi dữ liệu

- **FR-042**: Bộ dò bản ghi mức quyền tính phí MUST NOT thay đổi bất kỳ dữ liệu nào trong lúc tra cứu. Khi phát hiện dữ liệu không khớp, bộ dò MUST trả về kết quả để nơi khác quyết định, MUST NOT tự đóng hoặc xoá bản ghi. Ràng buộc này chỉ áp dụng cho **bản ghi mức quyền tính phí** và bước tra cứu; nó **MUST NOT** được hiểu là cấm ghi **sổ tính phí** ở bước đặt trước — bước đặt trước vẫn phải ghi sổ tính phí khoá dòng `FOR UPDATE` theo hiến pháp §VI, kể cả khi tài khoản đang được suy giảm.
- **FR-043**: Việc đóng một bản ghi mức quyền tính phí MUST thuộc quyền hạn duy nhất của bộ xử lý tín hiệu.
- **FR-044**: Tham số dùng để so khớp mã gói trong lúc tra cứu MUST bị gỡ khỏi giao diện tra cứu, vì sau FR-037 nó không còn ý nghĩa (FR-040 cấm giữ cờ bật/tắt giữa hai đường ghi, nên chỉ còn một đường ghi và không cần tham số để phân biệt); mọi nơi gọi phải được cập nhật cho khớp.

#### GĐ7 — Công cụ cấp gói cho quản trị viên

*Yêu cầu dưới đây áp dụng cho **cùng một thao tác cấp gói duy nhất**, với danh sách từ 1 đến N tài khoản. Danh sách một phần tử là trường hợp đơn giản nhất, không phải một giao diện lập trình riêng.*

*Nhận diện và áp dụng cho một tài khoản*

- **FR-045**: Hệ thống MUST cho phép quản trị viên xác định một tài khoản bằng đúng một trong ba dạng mã định danh: mã định danh tài khoản, tên đăng nhập, hoặc địa chỉ email.
- **FR-046**: Hệ thống MUST tự động nhận diện loại mã định danh quản trị viên truyền vào, không bắt buộc quản trị viên khai báo loại.
- **FR-047**: Hệ thống MUST so khớp tên đăng nhập và địa chỉ email không phân biệt hoa thường.
- **FR-048**: Khi một chuỗi mã định danh khớp nhiều hơn một tài khoản, hệ thống MUST đánh dấu phần tử đó là lỗi kèm nguyên nhân mơ hồ, MUST NOT tự chọn một tài khoản.
- **FR-049**: Hệ thống MUST loại trừ tài khoản đã bị xoá mềm khỏi kết quả nhận diện.
- **FR-050**: Hệ thống MUST cho phép quản trị viên truyền mã gói theo dạng mã định danh gói, không bắt buộc tra cứu trước mã định danh gói.
- **FR-051**: Khi cấp gói cho tài khoản chưa từng có gói, hệ thống MUST tạo bản ghi gói bắt đầu tại thời điểm cấp và kết thúc sau đúng số ngày của gói.
- **FR-052**: Khi cấp gói hữu hạn cho tài khoản đang dùng đúng gói đó và thời hạn còn hiệu lực, hệ thống MUST cộng dồn thời hạn, cộng thêm đúng số ngày của gói vào thời điểm hết hạn hiện tại.
- **FR-053**: Khi cấp gói cho tài khoản đang dùng gói khác, hoặc thời hạn gói hiện tại đã hết, hệ thống MUST bắt đầu thời hạn mới tại thời điểm cấp. Kèm theo đó, hệ thống MUST cập nhật hoặc xoá toàn bộ nhóm trường **dự phòng gói** và cờ **tự gia hạn** cho khớp với gói mới. Lý do: tài khoản đang dùng gói trọn đời có một nhóm trường dự phòng trỏ sang gói hữu hạn; cấp gói hữu hạn khác mà giữ nguyên nhóm cũ sẽ để lại dữ liệu mâu thuẫn, và cờ tự gia hạn còn sót lại có thể phát sinh gia hạn cho một gói mà khách hàng không còn ý định mua.
- **FR-054**: Hệ thống MUST đánh dấu phần tử là lỗi nếu mã gói không tồn tại trong danh mục.
- **FR-055**: Hệ thống MUST đánh dấu phần tử là lỗi nếu gói đã bị vô hiệu hoá.
- **FR-056**: Hệ thống MUST đánh dấu phần tử là lỗi nếu tài khoản đích là tài khoản quản trị viên.
- **FR-057**: Hệ thống MUST đánh dấu phần tử là lỗi với nguyên nhân không tìm thấy tài khoản khi không xác định được tài khoản đích.
- **FR-058**: Khi một phần tử thất bại, hệ thống MUST không ghi nhận bất kỳ thay đổi nào đối với tài khoản của phần tử đó.
- **FR-059**: Kết quả trả về cho quản trị viên MUST nêu rõ mã định danh tài khoản, tên gói, và thời điểm hết hạn mới cho từng phần tử thành công.

*Danh sách từ 1 đến N tài khoản*

- **FR-060**: Hệ thống MUST cung cấp **một và chỉ một** giao diện lập trình cho thao tác cấp gói, nhận một danh sách trong đó mỗi phần tử gồm một mã định danh người dùng và một mã định danh gói riêng. Hệ thống MUST NOT yêu cầu quản trị viên dùng giao diện khác cho trường hợp chỉ cấp cho một tài khoản.
- **FR-061**: Hệ thống MUST cho phép trộn nhiều mã định danh gói khác nhau trong cùng một danh sách.
- **FR-062**: Hệ thống MUST chấp nhận danh sách chỉ định **tối thiểu 1 và tối đa 500 phần tử**; vượt quá 500 phần tử thì phải từ chối toàn bộ yêu cầu, không cấp gói cho bất kỳ phần tử nào. Mỗi phần tử là chuỗi có **tối đa 255 ký tự**; chuỗi dài hơn phải bị từ chối với lý do sai định dạng, không được cắt bớt âm thầm. Giới hạn 255 ký tự này áp dụng cho **cả ba** dạng định danh được chấp nhận ở FR-045 (UUID, tên đăng nhập, địa chỉ email) và phải được kiểm tra **trước** khi tra cứu cơ sở dữ liệu, để một chuỗi khổng lồ không thể làm phình kích thước truy vấn. Tổng dung lượng yêu cầu cũng phải bị giới hạn bởi middleware giới hạn kích thước body đã có sẵn của dịch vụ.
- **FR-063**: Hệ thống MUST từ chối yêu cầu có danh sách 0 phần tử.
- **FR-064**: Yêu cầu phải là thao tác từng phần tử độc lập: một phần tử lỗi MUST NOT làm hỏng các phần tử còn lại. Điều này áp dụng **kể cả** khi phần tử sai ở mức không thể phân tích được — cụ thể, một mã định danh rỗng hoặc toàn khoảng trắng phải được báo thành trạng thái lỗi **riêng cho phần tử đó**, chứ không phải làm hỏng cả yêu cầu. Vì vậy việc kiểm tra phần tử MUST NOT được gắn vào quy tắc kiểm tra ở cấp toàn yêu cầu; nó phải nằm trong tầng ứng dụng để từng phần tử được đánh giá riêng.
- **FR-065**: Hệ thống MUST trả về kết quả riêng cho từng phần tử, phân loại rõ ba trạng thái thành công, bỏ qua và lỗi, kèm lý do tương ứng với từng trạng thái không thành công.
- **FR-066**: Khi cùng một tài khoản xuất hiện nhiều lần trong danh sách, hệ thống MUST chỉ xử lý phần tử đầu tiên và đánh dấu các phần tử sau là bỏ qua có lý do, MUST NOT cộng dồn thời hạn.
- **FR-067**: Các quy tắc ở FR-046, FR-047, FR-052, FR-053, FR-054, FR-055 MUST được áp dụng như trạng thái lỗi riêng cho từng phần tử, thay vì làm hỏng toàn bộ yêu cầu.
- **FR-068**: Hệ thống MUST bảo đảo gửi lại cùng một yêu cầu không làm thời hạn của tài khoản bị cộng dồn thêm lần nữa.
- **FR-069**: Khi hai quản trị viên cùng lúc cấp gói cho cùng một tài khoản, hệ thống MUST bảo đảo chỉ một thao tác thành công và thời hạn không bị cộng dồn kép.

*Quyền lợi và trạng thái sau khi cấp*

- **FR-070**: Sau khi cấp gói, hệ thống MUST ghi nhận toàn bộ quyền lợi của gói, bao gồm hạn mức tính phí AI, chứ không chỉ ghi nhận tên gói.
- **FR-071**: Mọi thay đổi gói MUST ghi nhận mức quyền lợi tại thời điểm thay đổi, để việc theo dõi và đối soát không phụ thuộc cấu hình gói ở thời điểm tra cứu về sau.
- **FR-072**: Hệ thống MUST bảo đảm việc gia hạn tự động và cấp gói thủ công **dùng chung đúng một** quy tắc tính thời hạn — quy tắc quy định ở FR-051, FR-052, FR-053 — để thời hạn của một tài khoản luôn nhất quán bất kể nguồn gốc. FR-072 là ràng buộc **tính nhất quán**; FR-051–FR-053 là quy tắc **tính toán**.
- **FR-073**: Hệ thống MUST bảo đảo tài khoản vừa được cấp gói nhìn thấy trạng thái gói mới ngay khi đăng nhập lại, không bị đọc giá trị cũ đã lưu tạm.
- **FR-074**: Hệ thống MUST đặt thời hạn lưu tạm trạng thái gói ở mức tối đa 60 giây, và việc xác định một tài khoản thuộc gói nào MUST không phụ thuộc vào giá trị trong bộ nhớ đệm.

*Thời gian phản hồi và giới hạn tần suất*

- **FR-075**: Thao tác cấp gói MUST là thao tác **đồng bộ**: quản trị viên nhận được kết quả trong chính yêu cầu đã gửi, MUST NOT phải thao tác tra cứu lần hai.
- **FR-076**: Yêu cầu có danh sách một phần tử MUST trả kết quả trong vòng **5 giây**; yêu cầu có danh sách tối đa 500 phần tử MUST hoàn tất trong vòng **60 giây**. Để ngưỡng 60 giây này khả thi, hạn chót ghi đáp lời của hệ thống phải lớn hơn 60 giây; hệ thống MUST NOT đặt hạn chót ghi đáp lời bằng 60 giây khi phục vụ thao tác này. Nếu yêu cầu vượt thời gian và bị huỷ, hệ thống MUST NOT tự động cấp gói tiếp các phần tử chưa xử lý, và phải coi những phần tử **đã** được cấp là đã hoàn tất — quản trị viên phải gửi lại yêu cầu với **cùng** khoá bất biến của lần trước (FR-068) để không cộng dồn thời hạn hai lần.
- **FR-077**: Ngưỡng thời gian ở FR-076 chỉ áp dụng cho việc ghi nhận kết quả cấp gói, KHÔNG bao gồm thời gian tính phí AI có hiệu lực — phần đó là bất đồng bộ và được đo bằng tiêu chí riêng.
- **FR-078**: Hệ thống MUST giới hạn tần suất gọi thao tác cấp gói theo từng quản trị viên: tối đa **60 yêu cầu/phút**, trong đó tối đa **6 yêu cầu/phút có nhiều hơn một phần tử**.
- **FR-079**: Khi vượt giới hạn tần suất, hệ thống MUST từ chối yêu cầu, MUST thông báo phải chờ bao lâu trước khi thử lại, và MUST NOT cấp gói cho bất kỳ phần tử nào của yêu cầu bị từ chối.

*Kiểm soát truy cập, kiểm toán và thông báo*

- **FR-080**: Hệ thống MUST chỉ cho phép quản trị viên đã đăng nhập thực hiện các thao tác cấp gói.
- **FR-081**: Mỗi thao tác cấp gói phải để lại dấu vết truy vết được: danh tính quản trị viên, tài khoản được cấp gói, mã gói, thời điểm thực hiện và địa chỉ mạng của yêu cầu.
- **FR-082**: Trong yêu cầu cấp gói **nhiều phần tử**, hệ thống MUST ghi nhận ký **riêng cho từng tài khoản được cấp gói thành công**, đồng thời trả về cho quản trị viên danh sách các phần tử bị bỏ qua hoặc lỗi. FR-081 là quy tắc chung về dấu vết; FR-082 là quy tắc **một dòng nhật ký cho mỗi phần tử thành công** trong ngữ cảnh danh sách.
- **FR-083**: Sau khi cấp gói thành công, hệ thống MUST gửi thông báo cho người dùng; nội dung thông báo phải **giống nhau** bất kể phần tử đó nằm trong danh sách một phần tử hay danh sách nhiều phần tử.
- **FR-084**: Việc gửi thông báo thất bại MUST NOT làm thất bại thao tác cấp quyền; thao tác cấp quyền vẫn phải thành công và ghi nhận ký đầy đủ.
- **FR-085**: Mọi thông báo lỗi trả về cho quản trị viên MUST viết bằng tiếng Việt và nêu rõ nguyên nhân cụ thể.

#### GĐ8 — Quan sát, khôi phục và kiểm chứng

- **FR-086**: Mỗi lần bộ xử lý tín hiệu từ chối một tín hiệu hoặc xử lý không thành công, hệ thống MUST ghi lại mã định danh tài khoản, lý do từ chối hoặc lỗi, và thời điểm; việc ghi nhận này MUST NOT phụ thuộc vào việc có thao tác cấp gói nào đi kèm hay không.
- **FR-087**: Hệ thống MUST duy trì bộ đếm số tín hiệu bị từ chối theo từng lý do, và MUST phát cảnh báo khi bộ đếm tăng.
- **FR-088**: Hệ thống MUST cung cấp công cụ để phát lại tín hiệu kích hoạt cho **mọi tài khoản đang có gói nhưng thiếu bản ghi mức quyền tính phí, không phân biệt gói miễn phí hay gói trả phí**; công cụ MUST chạy lại nhiều lần được mà không tạo tác dụng phụ, MUST báo cáo số lượng tài khoản đã khôi phục và số lượng không khôi phục được kèm lý do, và MUST NOT yêu cầu can thiệp trực tiếp vào cơ sở dữ liệu.
- **FR-089**: Thao tác khôi phục ở FR-088 MUST chỉ dành cho quản trị viên và MUST ghi nhận ký.
- **FR-090**: Hệ thống MUST cung cấp một tệp kịch bản kiểm chứng trong `database/scripts/` chạy tay, báo cáo sáu trường hợp: tài khoản có bản ghi gói nhưng thiếu bản ghi mức quyền tính phí; tài khoản có nhiều bản ghi mức quyền tính phí cùng hiệu lực; bản ghi mức quyền tính phí có mã gói không khớp gói hiện tại; bản ghi thiếu phần `entitlements` hoặc thiếu thao tác trong bốn thao tác; bản ghi đã hết hiệu lực nhưng còn trạng thái `active` hoặc `future`; và với mỗi gói, báo **riêng hai cột** theo từng thao tác — `max_paid_attempts_per_day` của cổng 2 **kèm cột chế độ thi hành**. Phép kết luận lệch/không lệch **chỉ áp dụng cho chính sách ở chế độ `observe_only`**, vì đó là chế độ duy nhất thực sự dùng số lượt mỗi ngày; với chế độ `strict` tệp chỉ ghi nhận giá trị và ghi chú là cấu hình chết, **không** được kết luận lệch từ nó (FR-005e). Tệp MUST in ra danh sách gói lệch **kèm lý do nằm trong hay ngoài** danh sách ngoại lệ đã duyệt — danh sách một mục nằm ở **SC-011**, còn quyết định cố ý lệch một lượt ở `outfit` của gói miễn phí nằm ở **FR-005d** — để kết quả `0` luôn kiểm chứng được chứ không phải do bỏ sót.
- **FR-091**: Tệp kịch bản kiểm chứng ở FR-090 MUST NOT thay đổi bất kỳ dữ liệu nào.
- **FR-092**: Hệ thống MUST cung cấp một tệp kịch bản dọn dữ liệu trong `database/scripts/` chạy tay **trước khi bắt đầu GĐ4**. Tệp kịch bản này MUST thực hiện đúng ba việc, theo thứ tự: **(1)** dựng lại ảnh chụp chính sách cho **mọi** bản ghi chứ không chỉ bản ghi đang rỗng, theo đúng hình dạng hai phần ở FR-029, lấy giới hạn từ bảng chính sách phí tại thời điểm chạy, và đặt phần `envelope` mang thời điểm chạy của chính tệp kịch bản; **(2)** đóng mọi bản ghi đã đóng và không còn hiệu lực, rồi xoá vĩnh viễn; **(3)** đóng mọi bản ghi đã hết hiệu lực dù còn để trạng thái `active` hoặc `future`, kể cả bản ghi có thời điểm bắt đầu đã nằm trong quá khứ. Tệp kịch bản MUST NOT xoá bất kỳ bản ghi nào đang thực sự còn hiệu lực.
- **FR-092b**: Việc dựng lại ảnh chụp ở FR-092 phải **ghi đè** mọi bản ghi cũ, không giữ lại ảnh chụp nào đang có, vì ảnh chụp sẽ thiếu hẳn nhóm quyền lợi cố ý lệch với cấu hình sống. Cách dựng lại phải tuân thủ đúng quy trình **phát lại tín hiệu** ở FR-029c — đó là lần chất giới hạn: ảnh chụp của bản ghi sống được dựng theo cấu hình tại thời điểm chạy tệp kịch bản, không phải tại thời điểm tạo bản ghi, vì dữ liệu cũ không lưu được giá trị lịch sử.
- **FR-093**: Tệp kịch bản dọn ở FR-092 MUST có phần báo cáo số lượng bản ghi sẽ bị thay đổi theo từng việc trước khi ghi, MUST nằm trong một giao dịch, và phải chạy lại nhiều lần được mà không đổi kết quả.
- **FR-094**: Sau khi dọn theo FR-092, tệp kịch bản kiểm chứng ở FR-090 phải báo cáo 0 bản ghi mức quyền tính phí thiếu phần `entitlements` hoặc thiếu thao tác trong bốn thao tác, và 0 bản ghi đã đóng không còn hiệu lực, để lỗi cũ tách được khỏi lỗi phát sinh trong quá trình chuyển đổi.

### Key Entities

- **Gói đăng ký (Subscription Plan)**: Danh mục gói có thể cấp. Thuộc tính: mã định danh gói, tên, thời hạn theo ngày, mức giá, hạn mức sử dụng AI, trạng thái hoạt động, cấp bậc quyền, mã chính sách phí AI. Quan hệ: một tài khoản chỉ có một gói hiệu lực tại một thời điểm.

- **Đăng ký của người dùng (User Subscription)**: Gói hiện tại của một tài khoản. Thuộc tính: tài khoản, gói, thời điểm bắt đầu, thời điểm kết thúc, ảnh chụp quyền lợi tại thời điểm cấp, gói dự phòng khi hết hạn, cờ tự gia hạn, số phiên bản. Quan hệ: thuộc về tài khoản, trỏ tới gói.

- **Lịch sử thay đổi gói (Subscription Event)**: Bản ghi mọi lần đổi gói; là nguồn phát tín hiệu duy nhất. Thuộc tính: định danh duy nhất, tài khoản, loại thay đổi, gói trước và sau, thời điểm xảy ra, thời điểm có hiệu lực, định danh giao dịch thanh toán nguồn. Quan hệ: thuộc về tài khoản; định danh của bản ghi là khoá chống xử lý trùng ở phía nhận.

- **Mức quyền tính phí AI (AI Policy Grant)**: Mức quyền tính phí AI của một tài khoản, **do hệ thống AI sở hữu duy nhất**. Thuộc tính: tài khoản, chính sách phí, mã định danh gói, mã gói, cấp bậc quyền, ảnh chụp chính sách, thời điểm bắt đầu, thời điểm kết thúc, trạng thái, định danh bản ghi lịch sử nguồn, định danh giao dịch nguồn. Quan hệ: thuộc về tài khoản; tối đa một bản ghi đang hiệu lực tại một thời điểm.

- **Chính sách phí AI (AI Cost Policy)**: Quy tắc chi phí cho một nhóm gói, **thuộc sở hữu của hệ thống AI**. Thuộc tính: mã, phiên bản, chế độ thực thi, trần chi phí mỗi chu kỳ, số ngày một chu kỳ, ngưỡng nén, số lượt trả phí tối đa mỗi ngày. Quan hệ: gói đăng ký trỏ tới một chính sách phí; ảnh chụp chính sách trong mức quyền tính phí được dựng từ bảng này.

- **Cấu hình giá mô hình (AI Model Config)**: Giá mỗi triệu token đầu vào và đầu ra, tỷ giá quy đổi. Quan hệ: dùng để ước tính chi phí mỗi lượt và kiểm tra trần.

- **Bản ghi phân phối sự kiện (Outbox Record)**: Bản ghi sự kiện chờ phát, ghi cùng giao dịch với bản ghi lịch sử. Quan hệ: bảo đảm tín hiệu không mất khi hệ thống AI tạm thời không liên lạc được.

- **Nhật ký kiểm toán quản trị (Admin Audit Log)**: Bản ghi thao tác quản trị. Thuộc tính: quản trị viên, hành động, loại đối tượng, đối tượng, dữ liệu bổ sung, địa chỉ mạng.

## Success Criteria *(mandatory)*

### Measurable Outcomes

*Suy giảm và đường đọc*

- **SC-001**: 0 thao tác AI nào trả về lỗi với lý do thiếu mức quyền tính phí, kể cả với tài khoản chưa có bản ghi mức quyền tính phí.
- **SC-002**: 0 trường hợp bộ dò bản ghi mức quyền tính phí tạo, sửa hoặc xoá bản ghi mức quyền tính phí trong lúc tra cứu; đồng thời 0 trường hợp nào bỏ qua bước đặt trước — mỗi lần gọi thao tác AI trả phí vẫn phải có dòng sổ tính phí tương ứng.
- **SC-003**: 0 thao tác AI nào trả lỗi ngay cả khi gói mặc định cũng không tồn tại hoặc đã bị vô hiệu hoá; mọi thao tác trả phí lúc đó chuyển sang mô hình miễn phí.

*Đồng bộ tín hiệu*

- **SC-004**: 100% thay đổi gói đều sinh bản ghi lịch sử; không còn trường hợp tài khoản có bản ghi gói nhưng không có bản ghi lịch sử thay đổi gói tương ứng.
- **SC-005**: 100% tài khoản từng hoàn tất thanh toán có mức quyền tính phí AI khớp với gói đang mua; không còn tình trạng gói hiển thị đúng nhưng thao tác AI trả phí không dùng được hạn mức gói.
- **SC-006**: 100% tài khoản mới đăng ký có mức quyền tính phí AI hợp lệ trong vòng 60 giây kể từ lúc tạo tài khoản, kể cả tài khoản dùng gói miễn phí và tài khoản do quản trị viên tạo.
- **SC-007**: Không tài khoản nào có từ hai mức quyền tính phí AI trở lên cùng hiệu lực tại một thời điểm.
- **SC-008**: 0 thao tác AI nào bị từ chối trong khoảng thời gian tín hiệu thay đổi gói đang trên đường tới.
- **SC-009**: 100% bản ghi mức quyền tính phí phải có phần `entitlements` đủ bốn thao tác và không bản ghi nào có ảnh chụp rỗng. Sau khi đổi trường chống lỗ tài chính, **0** tài khoản nào còn thi hành trần cũ. Sau khi đổi trường quyền lợi mà chưa phát lại tín hiệu, **100%** bản ghi cũ vẫn giữ đúng giới hạn đã mua.
- **SC-010**: Bảng mức quyền tính phí AI chỉ có một hệ thống duy nhất ghi vào; 0 lệnh ghi chéo hệ thống tồn tại trong mã nguồn.

*Cấu hình vận hành*

- **SC-011**: Trần chi phí chu kỳ của gói Premium là 30.000đ. **Cổng 1** của gói Premium là **30 lượt gợi ý phối đồ** và **10 lượt trò chuyện** mỗi ngày, có hiệu lực trên toàn bộ môi trường. Cổng 2 của gói Premium là `NULL` vì chế độ `strict` không dùng số lượt mỗi ngày (FR-005e), nên nó **không tham gia** phép so sánh hai cổng. Tệp kiểm chứng ở FR-090 phải báo **0** gói lệch **ngoài danh sách ngoại lệ đã được phê duyệt**; danh sách đó hiện có đúng một mục là `outfit` của gói miễn phí (cổng 1 = `5`, cổng 2 = `4`). Số gói lệch ngoài danh sách phải bằng **0**.
- **SC-012**: Tệp kịch bản cấu hình chạy lại nhiều lần không làm thay đổi kết quả.
*Công cụ cấp gói*

- **SC-013**: Quản trị viên cấp được gói Premium cho một tài khoản chỉ bằng cách dán email hoặc tên đăng nhập, không cần tra cứu mã định danh tài khoản trước.
- **SC-014**: 100% tài khoản được cấp gói thành công dùng được tính năng AI trả phí với hạn mức của gói trong vòng 60 giây kể từ lúc cấp.
- **SC-015**: Một yêu cầu cấp gói với 500 tài khoản hoàn tất và báo cáo đủ 500 kết quả trong vòng 60 giây; 95% yêu cầu một phần tử trả kết quả trong vòng 5 giây.
- **SC-016**: Quản trị viên không phải thao tác tra cứu lần hai ở bất kỳ trường hợp nào để biết kết quả cấp gói.
- **SC-017**: Trong một yêu cầu có lẫn tài khoản không tồn tại, 100% tài khoản hợp lệ vẫn được cấp gói, và quản trị viên nhận biết chính xác mục nào lỗi và vì sao.
- **SC-018**: 0 lần cấp quyền trùng lặp khi quản trị viên gửi lại cùng một yêu cầu do lỗi mạng.
- **SC-019**: 0 lần thời hạn bị cộng dồn kép khi hai quản trị viên cùng lúc cấp gói cho một tài khoản.
- **SC-020**: Một tài khoản quản trị viên không thể gửi quá 60 yêu cầu cấp gói trong một phút, và không thể gửi quá 6 yêu cầu nhiều hơn một phần tử trong một phút.
- **SC-021**: 100% thao tác cấp gói và khôi phục quyền tính phí đều truy vết được người thực hiện, thời điểm, gói được cấp và địa chỉ mạng.

*Rủi ro tài chính*

- **SC-022**: Tổng chi phí tính phí AI ước tính của chiến dịch trao quà 1 tháng gói Premium cho 50 tài khoản không vượt 3,4 triệu đ, tính theo mức sử dụng tối đa của từng tài khoản.

*Quan sát và kiểm chứng*

- **SC-023**: 100% tín hiệu bị từ chối đều truy vết được tới mã tài khoản và lý do; khi có tín hiệu bị từ chối, hệ thống phát cảnh báo trước khi người dùng phản ánh.
- **SC-024**: Tín hiệu có số phiên bản lớn hơn mức phía nhận hiểu bị từ chối kèm cảnh báo, và không làm phía nhận dừng hoạt động hay ghi dữ liệu sai.
- **SC-025**: Không có tài khoản nào mất quyền tính phí AI mà không để lại dấu vết điều tra được trong nhật ký.
- **SC-026**: Sau khi chạy công cụ khôi phục, tệp kịch bản kiểm chứng báo cáo 0 tài khoản có bản ghi gói mà thiếu bản ghi mức quyền tính phí, 0 tài khoản có nhiều mức quyền tính phí cùng hiệu lực, và 0 bản ghi mức quyền tính phí có mã gói không khớp gói hiện tại.
- **SC-027**: Số lượt hỗ trợ từ khách hàng nói "đã trả tiền nhưng không dùng được tính năng AI" giảm về 0 sau khi tính năng này được áp dụng. *(Ghi nhận thẳng thắn: tính năng này **không** bảo đảm chỉ số này. Vì cổng 2 là cơ chế nội bộ và lượng thứ 5 của người dùng gói miễn phí bị hạ mô hình **không kèm thông báo** theo lựa chọn của người phát triển ngày 29/09/2026, chỉ số này **dự kiến xấu đi**. Ghi lại ở đây để sau này không ai hiểu nhầm là hồi quy. Nếu cần, hạ cổng 2 gói miễn phí xuống dưới mức cổng 1 là biện pháp khả dụng — xem FR-005d.)*
- **SC-028**: 100% lần quản trị viên chỉnh cổng 2 qua giao diện lập trình đều để lại bản ghi kiểm toán trong `aiexecutor.ai_cost_policy_audit_logs` chứa quản trị viên, thời điểm, địa chỉ IP, giá trị cũ và giá trị mới; **0** bản ghi kiểm toán nào được ghi vào `shared.admin_audit_logs`; 100% lần đặt `NULL` cho chính sách ở chế độ `observe_only` bị từ chối; 100% lần chuyển sang chế độ `strict` khi `hard_cost_micro_vnd` còn trống bị từ chối; và 100% lần sửa `max_paid_attempts_per_day` trả về cảnh báo nêu rõ số bản ghi sẽ không tự làm mới. **FR-004b, FR-005e**
- **SC-029**: Sau khi chạy tệp kịch bản dọn: 0 bản ghi mức quyền tính phí thiếu phần `entitlements` hoặc thiếu thao tác, 0 bản ghi đã đóng không còn hiệu lực, 0 bản ghi đã hết hiệu lực nhưng còn để trạng thái `active` hoặc `future`; đồng thời số bản ghi **đang thực sự còn hiệu lực** không đổi trước và sau khi chạy.
- **SC-030**: 100% lần chỉnh cổng 2 tạo ra **cả hai** bản ghi trong cùng một giao dịch - bản ghi tại `aiexecutor.ai_cost_policy_audit_logs` và bản ghi outbox `admin.audit.log`; **0** trường hợp bản ghi kiểm toán tồn tại mà bản ghi outbox không tồn tại. Khi outbox thử lại **cùng một thay đổi** do lỗi tạm thời, màn hình kiểm toán tập trung không được có **hai dòng** cho cùng một thay đổi. Cơ chế chống trùng là kiểm tra trước khi ghi theo định danh ổn định, nên **giới hạn được thừa nhận**: hai event đến gần như đồng thời vẫn có thể tạo ra hai dòng, và đây là mức chấp nhận được — SC này **không** đòi hỏi bảo đảm tuyệt đối về tính đúng đắn khi có tranh chấp. **FR-004c, FR-004d, FR-004e**

## Assumptions

- Toàn bộ nội dung spec này thuộc **024** và được triển khai theo đúng thứ tự GĐ0 → GĐ8. Các yêu cầu cấu hình vận hành ở GĐ0 và công cụ quản trị ở GĐ7 không thể tách thành đợt triển khai riêng vì chúng chỉ có ý nghĩa khi đường cấp quyền đã chạy đúng.
- Thứ tự triển khai bắt buộc: suy giảm mềm (GĐ1) trước mọi thay đổi cấu trúc; hệ thống AI nhận quyền ghi (GĐ4) trước khi gỡ đường ghi vi phạm (GĐ5).
- Hệ thống quản lý hàng đỗi sự kiện và cơ chế ghi bản ghi phân phối sẵn có đã bảo đảm tín hiệu không mất; phần này chỉ bổ sung nội dung tín hiệu và sửa lỗi quyền tính phí, không thay đổi kiến trúc hàng đỗi.
- Quy trị viên là người dùng đã đăng nhập có quyền quản trị hệ thống; cơ chế phân quyền hiện có được tái sử dụng nguyên vẹn.
- Hệ thống đã có sẵn cơ chế giới hạn tần suất chung cho mọi giao diện lập trình; giới hạn riêng cho thao tác cấp gói là bổ sung, không thay thế giới hạn chung.
- Mặc định số ngày của một gói lấy từ cấu hình gói; gói không có số ngày thì là gói không thời hạn và thời điểm kết thúc để trống.
- So khớp tên đăng nhập và email không phân biệt hoa thường, nhất quán với hành vi hiện có của hệ thống khi kiểm tra trùng lặp.
- Cơ chế cộng dồn thời hạn cho cùng gói đang có hiệu lực đã tồn tại và được dùng cho gia hạn tự động; tính năng này giữ nguyên hành vi đó.
- **Phần hiển thị hạn mức cho khách hàng nằm ngoài kho mã nguồn này.** Kho này là dịch vụ hậu trường Go, không chứa mã ứng dụng khách. Tuy nhiên sau khi người phát triển chốt ngày 29/09/2026 rằng cổng 2 là cơ chế nội bộ, **không có yêu cầu hiển thị nào hướng tới khách hàng còn nằm trong tính năng này** — nên cũng không có phần nào bị bỏ sót ở kho khác. Thứ duy nhất phải bàn giao ra ngoài là câu chữ mô tả hạn mức theo cổng 1, và nó **không chứa** trần chi phí hay bất kỳ giá trị nào của cổng 2 (FR-005c).
- Giao diện quản trị trên web cho các thao tác này chưa nằm trong phạm vi; hiện thực hiện qua **một** giao diện lập trình duy nhất nhận danh sách từ 1 đến N tài khoản.
- Tài khoản do quản trị viên tạo được cấp phát mức quyền tính phí AI y hệt tài khoản đăng ký bình thường, theo FR-013; quản trị viên vẫn có thể cấp gói thủ công cho tài khoản đó nếu đáp ứng điều kiện cho phép.
- Việc gửi email thông báo cho người dùng sau khi được cấp gói là hành vi mong muốn, nhưng thất bại gửi không làm thất bại thao tác cấp quyền.
- Tài khoản thiếu mức quyền tính phí ở thời điểm triển khai là dữ liệu kiểm thử; không cần sửa dữ liệu, chỉ cần công cụ khôi phục để sẵn sàng cho dữ liệu thật.
- Cơ chế trao quà tự động cho 50 tài khoản đầu tiên là một tính năng độc lập, không nằm trong phạm vi spec này, và chỉ dùng chung phần sửa ở GĐ1 đến GĐ6.
- **Chiến lược kiểm thử của tính năng này**: kiểm thử tự động chỉ dành cho **logic thuần** — quyết định chính sách, dựng cảnh báo, dựng và phân tích event, và các trường hợp từ chối — dùng repository giả lập. **Không** có kiểm thử tự động ở tầng cơ sở dữ liệu trong tính năng này, vì toàn dự án hiện **không** có hạ tầng cho việc đó: `services/ai` chỉ có một tệp kiểm thử mapper thuần, và không nơi nào dùng `testcontainers`, `sqlmock`, `sqlite` hay `pgxmock`. Mọi khẳng định về dữ liệu thật — bản ghi kiểm toán có đủ trường, bản ghi outbox có đúng topic, bảng số dữ liệu trước/sau, hệ quả rollback — được xác minh **thủ công bằng `psql`** theo các bước ghi rõ trong `quickstart.md`, kèm số liệu kỳ vọng để so sánh. Đây là quy ước sẵn có của dự án, không phải lựa chọn riêng của tính năng này; nếu sau này dự án bổ sung hạ tầng kiểm thử DB thì các khẳng định này có thể chuyển sang tự động hoá.
