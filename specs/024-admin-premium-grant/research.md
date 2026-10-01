# Nghiên cứu & Quyết định thiết kế — 024

**Ngày**: 2026-09-29 | **Đầu vào**: [spec.md](./spec.md) (94 FR, 28 SC) | **Đầu ra**: [plan.md](./plan.md)

Mọi khẳng định dưới đây đã được kiểm chứng bằng **đọc mã nguồn** và **truy vấn trực tiếp cơ sở dữ liệu đang chạy** (PostgreSQL 17.10, container `postgres-closy-dev`, CSDL `smart_wardrobe_db`). Không có mục nào còn `NEEDS CLARIFICATION`.

---

## R1 — Loại thay đổi gói dùng chung qua gói hằng số trong `shared/`

**Vấn đề**: phía phát tín hiệu ghi `EventType` bằng **chuỗi viết tay** trong
`services/account/internal/modules/subscription/domain/entities/subscription_behavior.go`
(dòng 50, 95, 152, 211, 258, 300, 332). Phía nhận lại rẽ nhánh theo chuỗi ở
`services/ai/internal/modules/aiexecutor/application/usecase/grant/grant_uc.go:39-45`.

Hai danh sách **không khớp nhau**:

| Loại | Phía phát thực sự tạo ra | Phía nhận xử lý | Hệ quả |
|---|---|---|---|
| `ACTIVATED` | ✅ có | nhánh mặc định → tạo grant | đúng |
| `EXTENDED` | ✅ có | `return nil` — bỏ qua | **sai**: gia hạn không tạo được mức quyền tính phí kỳ mới |
| `UPGRADED` | ✅ có | nhánh mặc định → tạo grant | đúng |
| `OVERLAID` | ✅ có | nhánh mặc định → tạo grant | đúng |
| `LIFETIME_FALLBACK_RESTORED` | ✅ có | nhánh mặc định → tạo grant | đúng |
| `SUBSCRIPTION_DOWNGRADED_TO_FREE` | ✅ có | hạ cấp → hết hạn grant | đúng |
| `AUTO_RENEW_UPDATED` | ✅ có | nhánh mặc định → **tạo grant** | **sai**: bật/tắt tự gia hạn không đổi gói, nhưng lại hạ cấp rồi cấp lại mức quyền tính phí |
| `RENEWED` | ❌ **không ai tạo ra** | hạ cấp mọi grant | vô nghĩa |
| `EXPIRED` | ❌ **không ai tạo ra** | hạ cấp mọi grant | vô nghĩa |

Đây là **nguyên nhân gốc** khiến tín hiệu hỏng mà không ai phát hiện: hai bên không hề có một nguồn sự thật chung.

**Quyết định**: tạo gói hằng số mới `shared/domain/constants/subscriptioneventtype/`, định nghĩa đủ 7 loại thực tế. Phía phát dùng hằng số thay chuỗi viết tay; phía nhận rẽ nhánh bằng hằng số và **từ chối mọi loại chưa biết** (ghi cảnh báo + đẩy hàng chết) thay vì rơi vào nhánh mặc định.

`design-pattern.md` có mục *"Exception - Domain constants"* và hiến pháp §II cho phép import `domain/constants/` giữa các mô-đun, nên việc đặt hằng số ở `shared/` là hợp lệ. Chú thích ở `wallet.go:22-25` cấm chia sẻ **kiểu DTO** giữa các dịch vụ — đó là vấn đề khác; hằng số rời rạc thì không.

**Phương án đã cân nhắc**:
- *Sao chép hằng số vào cả hai dịch vụ* — chính xác là cái lỗi đang có, loại.
- *Giữ chuỗi viết tay, thêm kiểm thử hợp đồng so khớp hai danh sách* — bắt được hồi quy, nhưng không ngăn được việc thêm loại mới ở một bên và quên bên kia, và không sửa được ba dòng rẽ nhánh đang sai.

---

## R2 — Hàng đỗi thành quorum, và bọc lỗi tạm bằng `RetryableError`

**Vấn đề**: FR-017 đòi tín hiệu "MUST NOT bị mất" khi hệ thống AI tạm không liên lạc được. Chuỗi xử lý lỗi hiện tại:

1. `shared/infrastructure/messaging/rabbitmq.go:114-135` — lỗi từ handler được xử lý như sau:
   `PermanentError` → `Nack(requeue=false)` → hàng chết; `RetryableError` → `Nack(requeue=true)` → giao lại; **mọi lỗi khác → `Nack(requeue=false)` → hàng chết**.
2. `grant_uc.go` trả về lỗi GORM/Postgres thuần — rơi vào nhánh mặc định.
3. `topology.go:63-70` khai báo `subscription_plan_changed_queue` bằng `queueArgs(queue, false)` → **không phải quorum**, nên không có `x-delivery-limit`.

Kết hợp lại: **một lỗi cơ sở dữ liệu tạm thời làm mất vĩnh viễn một tín hiệu, không giao lại lần nào, và không có bộ đếm nào báo**. Không có backoff, không có đường phát lại tự động. Người dùng mất quyền AI và không ai biết.

**Quyết định** (hai thay đổi phải đi cùng nhau):
- `topology.go:63-70` đổi `queueArgs(..., false)` → `true`, tức thêm `x-queue-type: quorum` và `x-delivery-limit: 5`. Sau 5 lần giao lại thì vào `subscription_plan_changed_queue.dlq`.
- Bọc lỗi tạm (lỗi cơ sở dữ liệu, lỗi mạng) trong `event.RetryableError`; giữ `event.PermanentError` cho lỗi không bao giờ thành công khi giao lại (thiếu trường bắt buộc, phiên bản mới hơn mức hiểu).

**Vì sao phải đi cùng nhau**: nếu chỉ bọc `RetryableError` mà giữ hàng đỗi cũ, RabbitMQ sẽ giao lại **vô hạn** một hàng đồ xử lý nặng — vòng lặp nóng, không bao giờ tới hàng chết. Nếu chỉ đổi sang quorum mà không bọc lỗi, lỗi tạm vẫn đi thẳng vào hàng chết ở lần đầu. Chỉ khi có cả hai thì mới đúng: lỗi tạm thử lại có giới hạn, lỗi vĩnh viễn đi hàng chết ngay.

**Chuỗi thử tối đa của một tín hiệu** — hai tầng độc lập, cần biết để không tính nhầm:

| Tầng | Cơ chế | Số lần tối đa | Kết thúc ở đâu |
|---|---|---|---|
| Phía gửi — chưa phát được ra broker | `outbox.retry_max_attempts` | 3 | `outbox_records` chuyển trạng thái thất bại, cảnh báo `OutboxGaugeProbe` |
| Phía nhận — đã nhận được nhưng xử lý lỗi | `x-delivery-limit` (quorum) | 5 | `subscription_plan_changed_queue.dlq` |

Tín hiệu có thể dùng hết **3 × 5 = 15** lần thử trước khi nằm ở hàng chết. Hai con số này ở hai tầng khác nhau nên **không** cộng dồn thành 8 — cần giữ nguyên như vậy khi đọc cảnh báo, nếu không sẽ tưởng tín hiệu chết sớm. Và một tín hiệu nằm ở hàng chết là **mất vĩnh viễn**: không có đường phát lại tự động. Vì vậy FR-086 đòi ghi nhận mỗi lần từ chối, và công cụ khôi phục ở GĐ8 là đường cứu duy nhất.

Ba hàng đỗi khác đã là quorum (`fashion_analyze`, `wardrobe_fashion_analyzed`, `brand_fashion_analyzed`) — có tiền lệ trong chính tệp đó.

**Ảnh hưởng vận hành**: hàng đỗi đang chạy sẽ phải khai báo lại khi khởi động. Hàng chết đã tồn tại nên giữ nguyên.

---

## R3 — Một điểm phát tín hiệu duy nhất, dùng lại bởi mọi đường đổi gói

**Vấn đề**: hiện chỉ **một** nơi phát tín hiệu trong toàn bộ kho mã —
`services/account/internal/modules/subscription/application/usecase/purchase/wallet.go:158`,
và nó dựng payload bằng một struct viết tay cục bộ (`wallet.go:26-33`).

Tám đường đổi gói còn lại — nâng cấp, hạ cấp, quyết toán PayOS, cấp thủ công, thu hồi thủ công, cấp phát gói mặc định, bật/tắt tự gia hạn — **không phát gì cả**. Đây chính là FR-011/FR-012 và là nguyên nhân gốc của Vấn đề 2 trong spec.

**Quyết định**: trích một hàm trợ giúp dùng chung, đặt tên theo quy ước `_helper.go`
(`plan_change_publish_helper.go`), nhận `(txCtx, event, sub, plan)` và làm đúng ba việc trong cùng giao dịch:
1. Tạo bản ghi lịch sử nếu chưa có (dùng `GetByKey` + `event_key` bất biến).
2. Dựng payload đầy đủ 14 trường từ event + sub + plan.
3. `eventPublisher.Publish(ctx, TopicSubscriptionPlanChanged, payload)` — ghi vào outbox.

Mọi đường đổi gói gọi đúng hàm này. Bỏ struct payload viết tay ở `wallet.go`.

**Lý do không gọi thẳng từ tầng repository**: FR-011 nói bản ghi lịch sử là nguồn phát tín hiệu **duy nhất**, và `IUserSubscriptionEventRepository` không được phụ thuộc ngược lại `IEventPublisher` (tầng domain không được biết tầng application). Hàm trợ giúp đặt ở tầng application là điểm thoả mãn cả hai.

---

## R4 — Xoá hẳn `POST /api/v1/admin/ai/grants`

**Vấn đề**: `ai_admin_uc.go:182-209` (`CreateGrant`) **không bao giờ trả về 201**. Nó bỏ trống `policy_snapshot` (cột `NOT NULL`) và `status` (cột `NOT NULL` + `CHECK IN ('active','future','closed')`), nên mọi lần gọi đều vi phạm ràng buộc:

| Cột | Ràng buộc | Giá trị ghi | Kết quả |
|---|---|---|---|
| `policy_snapshot` | `jsonb NOT NULL` | chưa gán → `NULL` | `23502` |
| `status` | `varchar(20) NOT NULL` + `CHECK` | `""` | `23514` |
| `plan_code` | `varchar(100) NOT NULL` | `""` | qua — nhưng rỗng |
| `tier_rank` | `int NOT NULL` | `0` | qua — nhưng sai |

Ngoài ra nó bỏ qua lỗi của `ExpireGrantsByUserID` (`_ =`), nên có thể tạo ra hai mức quyền tính phí chồng nhau mà vẫn giao dịch thành công.

**Quyết định**: FR-035 đã cho phép chính sách "tạo được thì dùng, không làm được thì gỡ". Ta **gỡ**. Lý do không phải "nó hỏng" — mà là nguyên tắc của GĐ5: sau khi gỡ, hệ thống AI là chủ sở hữu duy nhất, và một đường ghi thứ hai chỉ tồn tại vì lịch sử. Giữ nó nghĩa là phải duy trì nó, kiểm thử nó, và nó vẫn cho phép ghi chéo. Thao tác cấp gói của quản trị viên đi qua luồng gói của Account như mọi người dùng, rồi tín hiệu tới AI.

`GET /api/v1/admin/ai/grants` (`ListGrants`) **giữ lại** — cần cho quan sát.

**Thay đổi phá vỡ**: `POST /api/v1/admin/ai/grants` biến mất. Nhật ký Swagger phải cập nhật. Hiện chưa có giao diện quản trị trên web dùng nó (nằm ngoài phạm vi), nên không tồn tại màn hình nào bị hỏng.

---

## R5 — Endpoint hàng loạt đặt ở `/admin/subscription-grants` (bắt buộc về mặt kỹ thuật)

**Vấn đề**: FR-060 đòi **một và chỉ một** giao diện lập trình. Đường cũ `POST /api/v1/admin/subscriptions/:userId/grant` phải bị thay.

Cái bẫy: nhóm `adminSubs` trong `subscription/presentation/router/router.go:86-92` đang có
`POST /:userId/cancel` và `POST /:userId/grant`. Nếu thêm `POST /grants` vào **cùng nhóm đó**, Gin's httprouter sẽ **panic lúc khởi động**: một đoạn tĩnh (`grants`) không thể cùng cấp với một đoạn đại lượng (`:userId`) ở cùng một cấp. Đây là hành vi đã biết của Gin, không phải suy đoán.

Mọi phương án "gộp vào `/admin/subscriptions/...`" đều dính: `/admin/subscriptions/grants` (1 đoạn tĩnh) và `/admin/subscriptions/grants/batch` (vẫn có `/grants` ở vị trí đại lượng).

**Quyết định**: nhóm hành chính mới, tiền tố riêng, không trùng vị trí đại lượng nào:

```text
POST /api/v1/admin/subscription-grants        # cấp gói hàng loạt (FR-060..FR-085)
POST /api/v1/admin/subscription-grants/restore # khôi phục mức quyền tính phí (FR-088)
```

Hai endpoint này dùng chung bộ giới hạn tần suất và cùng nhóm phân quyền `auth.Handle()` + `RolesAuthorize(roleslug.Admin)`.

Ngoài ra `POST /api/v1/admin/subscriptions/:userId/grant` bị **xoá**, đúng FR-060. `POST /api/v1/admin/subscriptions/:userId/cancel` **giữ nguyên** — nó không thuộc phạm vi FR-060 (FR-060 chỉ nói về *thao tác cấp gói*), và xoá nó sẽ mất chức năng thu hồi gói đang chạy.

**Sau khi thêm route**: `make generate svc=account` để sinh lại Swagger và tuyến nginx (`make nginx-routes` đọc từ Swagger).

---

## R6 — Chỉ mục `source_event_id` cho chống xử lý trùng

**Vấn đề**: FR-033 yêu cầu bỏ qua tín hiệu đã có bản ghi cùng `sourceEventId`. Truy vấn kiểm tra sẽ là `WHERE source_event_id = ?`. Chỉ mục hiện có là `ix_user_ai_policy_grants_active (user_id, effective_from, effective_to)` — **không** phục vụ truy vấn đó. Ở quy mô 82 tài khoản thì không sao, nhưng đây là nút thắt chạy trên **mọi** tín hiệu, và số bản ghi sẽ tăng theo số tài khoản × số lần đổi gói.

**Quyết định**: thêm chỉ mục. Đặt tệp Goose ở đâu là câu hỏi kiến trúc thật sự:

| Dịch vụ | Cây migration | DDL của `ai_policy_grants` |
|---|---|---|
| Account | `services/account/database/migrations/` (23 tệp) | ✅ **có** — tạo bảng, đổi tên schema sang `aiexecutor`, cắt khoá ngoại |
| AI | `services/ai/database/migrations/` | ❌ chỉ có `.gitkeep`; `di/providers.go:13-15` trả `nil`, validator là `noopValidator` |

**Chọn Account.** Hiến pháp quy định "mỗi dịch vụ sở hữu và chạy cây migration của chính nó" — nó không quy định mỗi bảng phải nằm trong cây của chủ sở hữu. Vì vậy đặt ở đây **không vi phạm**. Chuyển quyền sở hữu DDL sang dịch vụ AI sẽ là một đợt tái cấu trúc riêng (tạo cây migration, nối `provideMigrationsFS`, bỏ `noopValidator`, rà lại 23 tệp lịch sử) — vượt xa phạm vi 024.

Chỉ mục: `CREATE INDEX CONCURRENTLY` không dùng được trong Goose (nó phải nằm ngoài giao dịch), nên dùng `CREATE INDEX` thường; bảng có 16 dòng nên thời gian khoá không đáng kể.

**Nợ kỹ thuật cần ghi nhận**: `ai_policy_grants` sẽ tiếp tục có DDL do Account quản lý trong khi hệ thống AI là chủ sở hữu duy nhất về mặt ghi. Nên mở một mục nợ kỹ thuật riêng, không gộp vào 024.

---

## R7 — Suy giảm tầng 2 dùng lại đường bỏ qua đã có, không sinh bản ghi

**Vấn đề**: tầng 2 của FR-007 là "chế độ chỉ quan sát, không trần, 0 lượt trả phí mỗi ngày, mọi thao tác trả phí chuyển sang mô hình miễn phí". Để làm được, hệ thống phải trả về một quyết định chi phí mà không cần bản ghi mức quyền tính phí nào.

Hệ thống **đã có sẵn** một đường làm đúng việc đó: nhóm thao tác miễn phí ở `cost/usecase.go:46-72` — tạo `AIUsageEvent` với `ledger_id = NULL`, không `Reserve`, trả `Route: airoute.Free`, `Tracked: true`. Cột `ai_usage_events.ledger_id` là nullable với khoá ngoại `ON DELETE CASCADE`, nên đường này chạy ổn định (đang phục vụ Vision và Embedding).

**Quyết định**: tầng 2 dùng lại chính đường đó. Không tạo bản ghi, không ghi sổ tính phí, không sinh dòng nào trong lúc tra cứu — tức là tự động thoát FR-042 (đường đọc không được thay đổi dữ liệu).

Vẫn còn dữ liệu kiểm chứng: mỗi lần suy giảm đều tạo một dòng `ai_usage_events` với `ledger_id IS NULL`, và FR-010 yêu cầu ghi nhận tầng đã dùng cùng lý do.

**Phương án đã loại**: tạo một bản ghi "khẩn cấp" thật trong lúc đọc → vi phạm FR-042 và làm mất đúng ý nghĩa của GĐ6.

---

## R8 — Suy giảm tầng 1 vẫn dùng sổ tính phí `FOR UPDATE` của §VI

Đây là quyết định nặng nhất của đợt thiết kế này. Bản đầu tiên của tài liệu này chọn bộ đếm Redis và ghi ngoại lệ §VI vào Complexity Tracking; **bản này đã bỏ lựa chọn đó** sau khi kiểm chứng được một cách tuân thủ. Lý do nằm ở cuối mục.

**Vấn đề**: FR-007 tầng 1 nói "dùng chính sách phí của gói mặc định đang hoạt động". Truy vấn cơ sở dữ liệu cho thấy chính sách `free-default` có `normal_route = paid` cho cả `chat` lẫn `outfit`:

| Mã chính sách | Thao tác | `normal_route` | Token vào/ra |
|---|---|---|---|
| `free-default` | chat | **paid** | 3000/500 |
| `free-default` | outfit | **paid** | 5000/300 |
| `free-default` | rewriter | free | 1000/250 |
| `free-default` | summary | free | 3000/250 |

Nghĩa là nếu tài khoản thiếu bản ghi mà ta áp dụng chính sách này mà **không** qua sổ tính phí, thì thao tác sẽ đi thẳng lên nhà cung cấp trả phí mà **không bị giới hạn số lượt mỗi ngày**. Đây là rò rỉ tiền thật, không phải rủi ro lý thuyết.

Vậy có thể đi qua sổ tính phí không? Kiểm tra lược đồ thật:

```
ai_usage_period_ledgers.grant_id  uuid NOT NULL
ai_usage_period_ledgers_grant_id_fkey  FOREIGN KEY (grant_id) REFERENCES aiexecutor.ai_policy_grants(id) ON DELETE CASCADE
ai_usage_period_ledgers_user_id   uuid NOT NULL
ai_usage_period_ledgers_grant_id_period_index_key  UNIQUE (grant_id, period_index)
```

Không thể. `grant_id` bắt buộc, và tài khoản không có bản ghi thì không có `grant_id` hợp lệ. Suy ra các phương án:

| Phương án | Vì sao bị loại |
|---|---|
| Cho `grant_id` nullable rồi dựa vào khoá cũ | `ON CONFLICT (grant_id, period_index)` **không** chặn trùng khi `grant_id` là `NULL` (theo SQL `NULL` không bằng `NULL`) → mỗi lần gọi tạo một sổ mới → mất trọn tác dụng chống vượt hạn mức. |
| Cho phép ghi bản ghi trong đường đọc | Vi phạm FR-042 — đúng thứ mà GĐ6 sinh ra để diệt. |
| Bỏ hẳn số lượt trả phí khi suy giảm | Rò tiền ở mức không kiểm soát. |
| Bộ đếm Redis theo `userId` + ngày + thao tác | Chạy được nhưng tạo **hai** cơ chế chống vượt hạn mức: `Reserve` cho tài khoản có bản ghi, bộ đếm cho tài khoản không có. Hai đường phải giữ đồng bộ với nhau, và một đường nào đó sẽ lệch. |
| `UNIQUE NULLS NOT DISTINCT (grant_id, period_index)` | Hai tài khoản **khác nhau** đều có `grant_id = NULL` cùng `period_index` → đụng khoá chéo. |
| **`grant_id` nullable + chỉ mục duy nhất riêng cho dòng fallback** | **Chọn.** |

**Quyết định**: giữ nguyên §VI — sổ tính phí vẫn là điểm chống vượt hạn mứm **duy nhất**, vẫn khoá dòng `FOR UPDATE`, không cơ chế thứ hai nào.

Cách làm: cho `ai_usage_period_ledgers.grant_id` nhận `NULL`, và thêm một chỉ mục duy nhất **riêng cho các dòng không có bản ghi**:

```sql
CREATE UNIQUE INDEX ux_ledgers_user_period_fallback
  ON aiexecutor.ai_usage_period_ledgers (user_id, period_index)
  WHERE grant_id IS NULL;
```

`Reserve` nhận thêm khả năng `grant == nil` và chèn bằng
`ON CONFLICT (user_id, period_index) WHERE grant_id IS NULL DO NOTHING`,
rồi đọc lại dưới `FOR UPDATE` — giữ nguyên cơ chế hiện có, chỉ đổi vị trí khoá chống trùng theo trường hợp.

**Đã kiểm chứng trên PostgreSQL 17.10** (chạy thật trong giao dịch rồi `ROLLBACK`): `ALTER COLUMN ... DROP NOT NULL` thành công; tạo chỉ mục thành công; lần chèn thứ nhất `INSERT 0 1`; lần chèn thứ hai `INSERT 0 0` — tức **chống trùng có hiệu lực**; và dòng fallback không đụng các dòng sổ thật đang có của cùng tài khoản (khoá duy nhất cũ vẫn tách riêng theo `grant_id`). Tổng số dòng tăng đúng 1.

Bảng đã có sẵn cột `user_id NOT NULL`, nên không cần thêm cột nào.

Lợi ích thêm so với bộ đếm Redis: chỉ **một** nơi quyết định có cho chạy tuyến trả phí hay không; không phải đồng bộ hai nguồn sự thật; và tài khoản thiếu bản ghi vẫn có dữ liệu sử dụng để đối soát. Đổi lại: một thay đổi DDL trên bảng lõi của cơ chế tính phí — cùng cây migration đã dùng cho R6, nên không tăng nợ sở hữu DDL.

**Cách tìm chính sách mặc định**: bảng `ai_cost_policies` có `code`, `version`, `is_active` nhưng **không** có `tier_rank`, nên không tự suy ra được "chính sách mặc định". Chọn theo **mã cấu hình qua biến môi trường** `ai.default_cost_policy_code` (mặc định `free-default`), thêm phương thức `GetPolicyByCode(ctx, code)`.

Lý do không thêm cột `is_default`: cần DDL, mà DDL của bảng này lại nằm ở cây migration của Account (xem R6) — thêm một phụ thuộc lịch sử cho một nhu cầu cấu hình. Lý do không hardcode `free-default` trong mã: vi phạm §VIII và khiến việc đổi chính sách mặc định thành sửa mã + phát hành lại. Lưu ý thêm: `shared/config/loader.go` dùng `UnmarshalExact`, nên khoá cấu hình mới bắt buộc phải thêm vào **cả hai** tệp YAML.

---

## R9 — Nâng `write_timeout_seconds`, và sự cạn kiệt của biến môi trường

**Vấn đề**: FR-076 đòi 500 tài khoản hoàn tất trong 60 giây. Cấu hình hiện tại:

| Khoá | Giá trị | Ý nghĩa |
|---|---|---|
| `server.timeout_seconds` | 300 | hạn chót ngữ cảnh của handler, qua `GlobalTimeoutMiddleware` |
| `server.read_timeout_seconds` | 30 | đọc thân yêu cầu |
| **`server.write_timeout_seconds`** | **60** | **hạn chót ghi đáp lời** |
| `server.body_limit.default_mb` | 2 | 500 phần tử ≈ 40 KB → không cần nới |
| `rate_limit.replenishment_seconds` | 10 | cửa sổ cho `rl:api:<ip>` |

`write_timeout_seconds = 60` cắt đúng ngay ngưỡng mà FR-076 đặt ra. Nếu một yêu cầu 500 phần tử mất 59 giây, đáp lời bị cắt giữa chừng dù mọi phần tử đều đã cấp xong — quản trị viên thấy lỗi mạng rồi gửi lại, dính SC-018.

**Quyết định (đã được người phát triển xác nhận 2026-09-29)**: nâng `write_timeout_seconds` lên **120** ở cả `configs/config.yaml` và `configs/config.production.yaml`. Đây là con số **chốt**, không phải đề xuất — mọi tài liệu thiết kế và `tasks.md` đều lấy theo.

Cạn kiệt cần lưu ý: `shared/config/loader.go:90` cho thứ tự `biến môi trường > YAML > mặc định`, nhưng `applyEnvOverrides` (dòng 93-227) **chỉ** phủ một tập con, và trong đó **không có** `Server.TimeoutSeconds`, `BodyLimit.*`, `RateLimit.*`, `ClaimRateLimit.*`. Đồng thời `v.UnmarshalExact` biến khoá lạ trong YAML thành lỗi khởi động. Kết hợp lại: **mọi thay đổi cấu hình này phải sửa tay cả hai tệp YAML**, không có lối tắt qua biến môi trường.

`read_timeout_seconds = 30` giữ nguyên — đọc 40 KB là tức thì.

---

## R10 — Chống cộng dồn kép bằng `user_subscriptions.version`

**Vấn đề**: FR-069 đòi khi hai quản trị viên cấp gói cho cùng một tài khoản cùng lúc thì chỉ một thao tác thành công và thời hạn không bị cộng dồn kép.

Khoá dòng `FOR UPDATE` mà `ApplySubscriptionPlan` đã dùng (`user_subscription_repo_impl.go`, `GetByUserIDWithLock`) chỉ **xoay tuần tự** hai thao tác — thao tác sau vẫn thành công và vẫn cộng dồn, chỉ là chậm hơn. Khoá giải quyết tình huống tranh chấp, không giải quyết tình huống "hai yêu cầu độc lập cùng đích".

**Quyết định**: dùng cột `version` đã tồn tại trên `user_subscriptions` (được `ApplySubscriptionPlan` tăng mỗi lần) làm chống lạc phiên bản. Thao tác hàng loạt đọc `version` khi lập kế hoạch, kiểm tra lại bên trong giao dịch; nếu đã thay đổi thì đánh dấu phần tử đó lỗi với lý do "tài khoản vừa được đổi gói bởi thao tác khác".

**Phương án đã loại**: khoá theo khoá bất biến của yêu cầu (xem R11) — không có tác dụng ở đây vì hai quản trị viên gửi hai yêu cầu khác nhau. Cột mới `updated_at`-based optimistic lock — `version` đã sẵn có và đã được tăng đúng chỗ, thêm cột mới là thừa.

---

## R11 — Chống gửi lại bằng `event_key` dựa trên khoá bất biến của yêu cầu

**Vấn đề**: FR-068 đòi gửi lại cùng một yêu cầu (do lỗi mạng) không được làm thời hạn cộng dồn thêm lần nữa.

Cơ chế có sẵn: `user_subscription_events.event_key` là `varchar(255) NOT NULL UNIQUE`, và `IUserSubscriptionEventRepository.GetByKey(ctx, key)` đã tồn tại, trả `(nil, nil)` khi không thấy. `webhook/purchase_workflow.go:74-80, 115-121` đã dùng đúng mẫu `GetByKey` → nếu có thì bỏ qua → nếu không thì `Create`. Đây là mẫu chống trùng chuẩn của kho mã.

**Quyết định**: yêu cầu cấp gói nhận `Idempotency-Key` (tiêu đề HTTP, chuẩn đã có sẵn ở mọi hệ thống hiện đại). Bắt buộc — nếu không có thì không có gì để so sánh lần gửi thứ hai với lần thứ nhất. Khoá dựng thành `ADMIN_GRANT:{idempotencyKey}:{index của phần tử}`. Phần tử đã có bản ghi lịch sử với khoá đó → trả về `skipped` với lý do `duplicate_request`.

Cần đặt `ActorAdminID` (cột `*uuid.UUID` hiện có nhưng **chưa nơi nào gán** — `payment_resolution_usecase.go:12` mới chỉ mang nó trong DTO). Đây là thứ đáng lẽ phải có từ lâu và giải quyết luôn câu hỏi "quản trị viên nào đã cấp" ở cấp lịch sử, không chỉ ở nhật ký kiểm toán.

**Phương án đã loại**: cửa sổ khửa trùng theo thời gian (ví dụ "cùng người + cùng gói trong 60 giây thì coi như trùng") — dễ bỏ sót yêu cầu hợp lệ, và FR-069 yêu cầu hành vi *từ chối* chứ không phải *bỏ qua trong im lặng*.

---

## R12 — Thêm mẫu email `admin_grant`

**Vấn đề**: FR-083 đòi thông báo cho người dùng sau khi cấp gói, FR-084 đòi thất bại gửi không được làm thất bại thao tác cấp quyền. Danh sách mẫu hiện có (`notification/domain/constants/template/template.go:6-14`) không có mẫu nào cho cấp gói.

Cơ chế đã có và đúng: `INotificationContract.Send` ghi thẳng một bản ghi outbox, nên lời gọi **bên trong** `uow.Execute` là nguyên tử cùng thao tác cấp gói. `wallet.go:163-179` và `webhook/purchase_workflow.go:131-141` đều theo mẫu: tra `userBriefContract.GetByID`, bỏ qua nếu không có email, nuốt lỗi bằng `_ =`.

**Quyết định**: thêm mẫu `admin_grant`. FR-083 yêu cầu nội dung **giống nhau** bất kể danh sách một hay nhiều phần tử — nên dùng **một** mẫu, không tham số hoá theo vị trí trong danh sách, và dùng lại đúng tham số như luồng mua gói (`PlanName`, `ExpiresAt`).

Thêm một mẫu đòi sửa **4 chỗ** phối hợp, nếu thiếu một chỗ thì gửi email sẽ hỏng lúc chạy:
1. Hằng số trong `template/template.go`.
2. Tệp `AdminGrant.html` trong `application/templates/`.
3. Khai báo `//go:embed` + biến trong `templates/templates.go`.
4. Mục trong bản đồ `tmplCache` (`outbox_email_uc.go:36-44`) **và** một nhánh `case` trong `resolveSubject` (`:103-120`).

Thêm hằng số đại diện trong `notification_contract.go` để mô-đun subscription gọi bằng ký hiệu thay vì chuỗi.

---


## R13 — Ảnh chụp cho quyền lợi, cấu hình sống cho chống lỗ tài chính

**Vì sao phải chốt**: Edge Case của spec hỏi *"chính sách phí được sửa trong lúc bản ghi đang hiệu lực — dùng ảnh chụp cũ hay cấu hình mới?"* và không nơi nào trả lời. Đã kiểm chứng trên mã: `ResolvePolicy` đọc lại dòng `ai_cost_policies` **sống** theo `grant.PolicyID`, còn `PolicySnapshot` chỉ xuất hiện **đúng một lần** trong toàn bộ dịch vụ AI — dòng khai báo trường ở `entities/policy.go:53`. Nó là cột **ghi một chiều**.

Vấn đề gốc: bảng `ai_cost_policies` trộn **hai loại cấu hình khác bản chất**. `ai_cost_policy_operations` còn trộn thêm: `max_paid_attempts_per_day` là quyền lợi, còn `normal_route` / `is_enabled` là quyết định vận hành.

### Ngành làm gì

Hai hướng, và cả hai khớp nhau:

**Phía quyền lợi — ảnh chụp, giữ giá cũ là lựa chọn tường minh.** Chargebee có tính năng Grandfathering: đổi entitlement của một plan price thì **mặc định áp dụng cho tất cả** subscription đang chứa plan đó; muốn giữ giá trị cũ cho khách hiện hữu thì phải bật `apply_grandfathering` tường minh. Stripe tương tự — đổi giá subscription bắt buộc phải thay thẳng subscription item, không bao giờ âm thầm đổi.

**Phía trần chi phí — luôn đọc cấu hình sống.** GCP ra mắt Spend Cap tháng 7/2026: khi chi phí chạm ngưỡng, hệ thống tự chặn, và ngưỡng luôn là giá trị **đang cấu hình** — sửa là có hiệu lực ngay. Tài liệu Firebase nói thẳng: *"You remain responsible for any overage costs accrued due to reporting latency."* Không có khái niệm giữ trần cũ cho khách đã mua.

Công thức ngành dùng: **cái khách mua thì đóng băng, cái chặn rò rỉ thì luôn sống.**

### Vì sao "đóng băng cả hai" là nguy hiểm

- GĐ0 đổi trần 25k → 30k sẽ **không có tác dụng lên tài khoản nào đang có bản ghi**.
- Sau này hạ trần xuống 20k vì giá model tăng, bản ghi cũ vẫn siết 30k.
- Tệ nhất: 5 tài khoản gói miễn phí có bản ghi `effective_to IS NULL`, chỉ được tạo lại khi họ đổi gói — tức là **vĩnh viễn**. Vận hành không bao giờ dừng được một đợt chi phí bằng cách hạ trần.

### Vì sao "sống cả hai" cũng sai

Đọc sống `max_paid_attempts_per_day` nghĩa là đổi cấu hình sẽ **âm thầm cắt** hạn mức của người đã trả tiền. Không dấu vết, không ai biết — đúng loại lỗi mà cả spec này sinh ra để diệt.

### Quyết định (người phát triển đã chốt 29/09/2026)

Ảnh chụp gồm hai phần `envelope` + `entitlements` (FR-029). Ranh giới là **một quy tắc duy nhất** (FR-029b):

| Nhóm trường | Đọc từ đâu |
|---|---|
| `max_paid_attempts_per_day`, `normal_max_input_tokens`, `normal_max_output_tokens`, `reduced_*` | **Ảnh chụp** `entitlements` |
| `normal_route`, `reduced_route`, `free_route` | **Sống** — quyết định định tuyến là vận hành; lượt trả phí đã đóng băng nên đổi tuyến chỉ **giảm** chi phí |
| `hard_cost_micro_vnd`, `enforcement_mode`, `free_route_threshold_bps`, `max_unknown_paid_requests_per_day`, `unknown_hold_minutes`, `period_days`, `is_active`, `is_enabled` | **Sống** — chống lỗ tài chính và chống lạm dụng |

### Đánh đổi được chấp nhận

Đổi trường **quyền lợi** không tự áp dụng cho bản ghi cũ. Muốn đẩu xuống tất cả thì chạy công cụ phát lại tín hiệu ở GĐ8 (FR-029c, FR-088). Đây đúng là mẫu Chargebee: giữ bản cũ mặc định, đẩy mới là hành động **tường minh** — và công cụ đẩy **đã có sẵn trong spec**, không phải thêm máy mới. Trường **chống lỗ** không cần công cụ gì.

**Phương án bị loại**: đóng băng toàn bộ rồi bắt vận hành sửa hàng loạt ảnh chụp bằng tệp kịch bản. Cho quyền kiểm soát tốt nhất nhưng biến mỗi lần hạ trần thành thao tác ghi hàng loạt trên dữ liệu người dùng — rủi ro vận hành cao hơn nhiều so với giá trị mang lại.

### Không cần migration DDL

`policy_snapshot` đã là `jsonb NOT NULL` không có lược đồ, nên thêm hai phần chỉ là ghi nội dung khác vào cùng cột. Phần đọc sống dùng đúng các cột đã có ở `ai_cost_policies` và `ai_cost_policy_operations`. Không cột mới, không chỉ mục mới.

**Nhưng phải dựng lại ảnh chụp cho toàn bộ bản ghi.** Đã đọc 8 ảnh chụp không rỗng trên dữ liệu thật: chúng **thiếu hẳn nhóm quyền lợi** (vì bản ghi gốc chỉ chụp cấp hình ở cấp chính sách, bỏ qua bảng thao tác), **trộn hai kiểu ghi** (25 khoá gồm cả `Code` kiểu Go lẫn `code` camelCase → có hai nơi từng ghi), và **lệch với cấu hình sống** (`maxUnknownPaidRequestsPerDay: 0` trong khi cột thật là `2`; `mode: "OBSERVE_ONLY"` viết hoa trong khi hằng số Go là `observe_only`).

Nếu từng có ai đọc ảnh chụp này để thi hành, chống lạm dụng của tài khoản gói miễn phí sẽ bị kẹt ở 0. Việc mã hiện tại đọc cấu hình sống hóa ra **may mắn**, không phải cố ý.

Phạm vi GĐ3b vì vậy nới từ 8 dòng rỗng lên **ghi đè toàn bộ**. Đây là lần chốt giá **một lần**: ảnh chụp của bản ghi sống lấy theo cấu hình lúc chạy tệp kịch bản, không phải lúc tạo bản ghi, vì dữ liệu tháng 6 không lưu được lịch sử giá — chấp nhận được vì 7 bản ghi còn lại đều là kiểm thử (FR-092b).

---

## R14 — Nhật ký chỉnh cổng 2 phát sinh sau, không có trong nghiên cứu nền

Các nghiên cứu R1–R13 tập trung vào đường cấp quyền và bỏ đường ghi vi phạm. Việc **quản trị viên sửa cấu hình cổng 2** không thuộc phạm vi ban đầu: ban đầu cổng 2 chỉ sửa bằng tệp kịch bản cơ sở dữ liệu, và không có nhật ký.

Ba điều chỉ ra sau khi khảo sát hạ tầng, không phải từ suy luận:

1. **Bảng `shared.admin_audit_logs` không dùng được.** Nó thuộc mô-đun `admin` của Account, có consumer `AuditLogConsumerWorker` đã đăng ký trên `QueueAuditLog`. Dịch vụ AI ghi thẳng vào đó sẽ vi phạm §I — đúng loại vi phạm mà tính năng này đang xoá. Vì vậy nhật ký cổng 2 phải nằm ở **bảng của AI**, xem `data-model.md` §2b.

2. **Hạ tầng phát và nhận đã có sẵn.** Chủ đề `admin.audit.log` và hàng đợi `audit_log_queue` đã được khai báo trong `shared/application/constants/eventconstants` (`TopicAuditLog`, `QueueAuditLog`); binding, hàng chết, `AuditLogConsumerWorker` và `dto.RecordAuditLogInput` cũng đã có. Nên phần mới chỉ là **phía phát** — không phải dựng mới hàng đợi, không viết consumer.

3. **Nhật ký và sự kiện phải chung giao dịch.** `OutboxRepositoryImpl.getDB(ctx)` đọc `db.GetTx(ctx)`, và `GormUnitOfWork.Execute` tiêm transaction vào `ctx`. Nhờ vậy `IEventPublisher.Publish` và repository ghi nhật ký cùng ghi vào một transaction mà không cần truyền `tx` riêng — xem FR-004c.

Điểm dễ bỏ sót: `action` và `targetType` là **chuỗi**, không có kiểu để kiểm tra. Sai quy ước đặt tên sẽ **không báo lỗi** mà ghi thẳng một giá trị lạ vào bảng kiểm toán. Vì vậy FR-004d bắt buộc dùng hằng số trong `shared/domain/constants/adminaudit/`.
---

## Tóm tắt các số liệu đã kiểm chứng trên cơ sở dữ liệu

| Truy vấn | Kết quả | Ảnh hưởng tới kế hoạch |
|---|---|---|
| Phiên bản PostgreSQL | 17.10 | Hỗ trợ chỉ mục riêng có điều kiện và `ON CONFLICT ... WHERE`, đủ cho cách làm của R8 |
| Tổng bản ghi mức quyền tính phí | 16 (8 `active`, 7 `closed`, 1 `future`) | Quy mô GĐ3b |
| Bản ghi có ảnh chụp chính sách rỗng | **8 / 16** | SC-009 hiện **không** đạt → GĐ3b phải điền lại (FR-092) |
| Bản ghi `closed` đã hết hiệu lực | 7 | Sẽ xoá ở GĐ3b |
| Bản ghi `active`/`future` đã hết hiệu lực nhưng sai trạng thái | 2 (cùng một tài khoản Premium) | Sẽ đóng ở GĐ3b (FR-092) |
| Tài khoản có ≥2 mức quyền tính phí **cùng hiệu lực** | **0** | SC-007 hiện **đã đạt** — cần đính chính Bối cảnh của spec |
| Dòng sổ tính phí | 13 | Xác nhận `grant_id` FK đang hoạt động → R8 |
| `ai_usage_period_ledgers.grant_id` | `NOT NULL` + FK | R8 — nguồn gốc vướng, đã xử lý bằng `DROP NOT NULL` |
| `ai_usage_period_ledgers.user_id` | `NOT NULL` (đã có sẵn) | Cho phép khoá chống trùng theo tài khoản mà không cần thêm cột — R8 |
| Chèn hai lần với `grant_id = NULL` + chỉ mục riêng | `INSERT 0 1` rồi `INSERT 0 0` | **R8 — đã kiểm chứng chống trùng có hiệu lực** |
| `ai_usage_events.ledger_id` | nullable, FK `ON DELETE CASCADE` | R7 — đường bỏ qua sẵn có là chạy được |
| `ai_cost_policies` | `code`, `version`, `is_active`; **không** có `tier_rank` | R8 — tìm chính sách mặc định theo mã cấu hình |
| `premium-default.hard_cost_micro_vnd` | 25.000.000.000 | GĐ0 nâng lên 30.000.000.000 |
| `free-default.hard_cost_micro_vnd` | `NULL` (đúng ý đồ: không trần) | Không đổi |
| `free-default.normal_route` (chat, outfit) | **`paid`** | R8 — thiếu chặn lượt thì rò tiền |
| Kết nối tới `users` | `ON DELETE CASCADE` | Xoá tài khoản sẽ xoá luôn bản ghi mức quyền tính phí — hành vi chấp nhận được |
| Kết nối tới `ai_policy_grants(plan_id)` | **đã bị cắt** | `PlanID` không được cơ sở dữ liệu bảo vệ → phía phát phải gửi đúng |
