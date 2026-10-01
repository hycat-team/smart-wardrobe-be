# Nghiên cứu & Quyết định Thiết kế — 025 Cơ chế chiến dịch tặng gói

**Đầu vào**: `spec.md` (98 yêu cầu, 19 tiêu chí thành công) · **Hiến pháp**: `.specify/memory/constitution.md` v3.0.0
**Ngày**: 2026-10-01

Mỗi quyết định dưới đây đều kèm bằng chứng đọc từ mã nguồn. Những mục đánh dấu **BẰNG CHỨNG** là
kết quả kiểm chứng trên mã hiện có, không phải suy luận.

---

## R1 — Bảng lượt cấp: khoá duy nhất là `user_id`, không phải cặp với mã chiến dịch

**Quyết định**: bảng mới `subscription.campaign_claims` với ràng buộc `UNIQUE (user_id)`. Cột
`campaign_code` chỉ ghi lại chiến dịch nào đã cấp, **không** nằm trong khoá.

**Lý do**: FR-012 nói mỗi tài khoản tham gia **nhiều nhất một lần trong toàn bộ cơ chế**, và
FR-093 nói lần cấp của chiến dịch dùng cùng quy ước với lần cấp của quản trị viên. Nếu khoá là
`(campaign_code, user_id)` thì một tài khoản đã nhận quà chiến dịch A vẫn có thể nhận quà chiến
dịch B — vi phạm FR-012. Khoá chỉ trên `user_id` khiến **một câu lệnh insert** chống được cả
trùng thông báo lẫn trùng chiến dịch, không cần logic kiểm tra riêng.

**Phái sinh**: đếm tách phần theo `COUNT(*) WHERE campaign_code = ? AND grant_bucket = ?`. Vẫn cần
cột `campaign_code` cho việc này, chỉ là không dùng làm khoá. Phải đếm **tách phần** chứ không đếm
gộp một khối — xem R2b.

**Phương án đã xét**:
- `UNIQUE (campaign_code, user_id)` — bị loại: cho phép một tài khoản tham gia nhiều chiến dịch.
- Bảng riêng đếm số lượt + bảng riêng chống trùng — bị loại: hai nguồn sự thật phải khớp trong
  cùng transaction, không có lợi ích nào để đổi.

---

## R2 — Mốc ưu tiên lưu ở đâu

**Quyết định**: cột boolean `fills_main_quota` trên chính bảng lượt cấp, kèm chỉ mục
`CREATE UNIQUE INDEX ... ON subscription.campaign_claims (campaign_code) WHERE fills_main_quota`.

**Lý do**: mốc ưu tiên là *thời điểm tạo của tài khoản lấp đầy hết phần hạn mức chính* (FR-030).
Gắn nó vào chính bản ghi đó giữ được tính bất biến "mốc không đổi theo thứ tự xử lý" mà không cần
bảng trạng thái riêng. Chỉ mục một phần bảo đảm **tối đa một** mốc trên mỗi chiến dịch ngay ở
tầng dữ liệu, nên hai worker chạy song song không thể chốt hai mốc khác nhau.

**Cách đọc mốc**:
```sql
SELECT registered_at FROM subscription.campaign_claims
WHERE campaign_code = ? AND fills_main_quota
```
Một câu, có chỉ mục, trả về tối đa 1 dòng.

**Cột này KHÔNG dùng để đếm phần chính** — xem R2b.

---

## R2b - Tách phần chính và phần dự phòng bằng cột riêng, không suy ra từ cột mốc

**Quyết định**: thêm cột `grant_bucket VARCHAR(16) NOT NULL` với
`CHECK (grant_bucket IN ('main', 'reserve'))` trên bảng lượt cấp, cùng chỉ mục
`(campaign_code, grant_bucket)`. Giữ `fills_main_quota` nguyên vẹn cho vai trò mốc ưu tiên.

**Lý do**: bản thiết kế đầu tiên đếm phần chính bằng `COUNT(*) WHERE NOT fills_main_quota`. Cách
đó **sai**, và sai theo cách không thể phát hiện bằng mắt khi đọc lướt:

- Cột `fills_main_quota` chỉ đánh dấu **một** dòng (lượt lấp đầy). Phủ định của nó vì thế bằng
  `tổng số lượt cấp trừ 1`, tức là **cộng luôn cả phần dự phòng vào phần chính**.
- Thuật toán dùng số đó để quyết định "lượt này lấp đầy chưa". Sau 50 lượt chính, lượt thứ 51 tính
  `đãCấpChính = 49`, suy ra `49 + 1 == quota` và lại đặt `fills_main_quota = TRUE` — đụng
  `ux_campaign_claims_watermark`. Phần dự phòng **không bao giờ dùng được**: nó lỗi vi phạm chỉ mục
  thay vì rơi sang nhánh dự phòng. Đây đúng là lúc phần dự phòng cần dùng nhất.
- Số liệu tra cứu cũng lệch: cấp 50 chính + 2 dự phòng thì báo 51 chính, 1 dự phòng, và máy trạng
  thái báo `compensation` sớm hơn thực tế.

**Phương án bị loại — `sequence_no SMALLINT` (1..50 là chính, 51..55 là dự phòng)**: suy ra phần từ
hạn mức trong cấu hình, tức là ý nghĩa của các dòng lịch sử phụ thuộc vào tệp YAML. Nâng `quota` từ
50 lên 60 trên chiến dịch đang chạy sẽ âm thầm phân loại lại mọi lượt cấp cũ, làm số liệu tra cứu
sai và dịch chuyển mốc ưu tiên. `grant_bucket` đóng băng ý nghĩa tại thời điểm ghi nên chịu không
đổi.

**Hệ quả vận hành**: `quota` và `reserve` của chiến dịch đang chạy phải bất biến. Muốn đổi thì kết
thúc chiến dịch cũ và mở mã chiến dịch mới.

**Tách bạch hai cột** — đây là chỗ dễ nhầm nhất, nên nói thẳng:

| | `grant_bucket` | `fills_main_quota` |
|---|---|---|
| Trả lời | Lượt cấp này lấy từ phần nào? | Lượt cấp này có lấp đầy phần chính không? |
| Giá trị | Mọi dòng đều có | Đúng một dòng mỗi chiến dịch |
| Dùng để | Đếm `grantedMainCount` / `grantedCompensationCount` | Tìm mốc ưu tiên FR-030 |

---

## R3 — Chặn truy cập hạn mức: khoá tư vấn theo transaction, KHÔNG dùng `JobLocker`

**Quyết định**: `SELECT pg_advisory_xact_lock(hashtextextended('subscription.campaign_claims', 0))`
gọi qua `r.GetDB(ctx)` ngay trong transaction của use case.

**Lý do — BẰNG CHỨNG**: mẫu có sẵn tại
`services/ai/.../infrastructure/persistence/ai_policy_grant_writer.go:23-45`, dùng cho đúng bài toán
"ghi cho một tài khoản phải được tuần tự hoá". Khoá theo transaction tự giải phóng khi commit hoặc
rollback, nên worker chết giữa chừng không để lại khoá mồ côi.

**Phương án đã xét**:
- `shareddb.JobLocker` (`TryLock`) — bị loại: khoá ở **mức session**, phải giữ một connection riêng
  suốt thời gian chạy. Nó dành cho job chạy theo lịch (`SubscriptionRenewalLockID`,
  `LoyaltyExpiryLockID`), không dành cho một transaction ngắn bên trong request. Giữ session khoá
  trong khi transaction đang chờ khoá khác là nguồn kết nối chết.
- Khoá Redis — bị loại: dự án **không có** helper khoá Redis, và khoá Redis không thể nằm trong
  cùng transaction với lần ghi lượt cấp, nên vẫn còn khe hở.

**Cảnh báo bắt buộc cho người viết code**:
`GormUnitOfWork.Execute` gọi `u.db.Transaction(...)` trên **handle gốc** (`shared/infrastructure/db/uow.go:69`),
không phải handle tx trong ctx. Gọi `uow.Execute` lồng nhau sẽ **mở transaction thứ hai trên
connection thứ hai** — hai transaction không nhìn thấy dữ liệu của nhau, và khoá tư vấn lấy ở
transaction trong sẽ không khoá được transaction ngoài.

Đã kiểm tra: `SubscriptionStateSupport.GetOrCreateUserDailyQuota`
(`subscription_state_helper.go:253-285`) **không** mở transaction — nó dùng `CreateIfAbsent` với
`ON CONFLICT DO NOTHING` làm chốt tranh chấp. Vì vậy đường cấp gói chiến dịch không vướng.
Ngược lại `GetOrCreateUserSubscription` và `LoadPlanForSubscription` phải gọi **ngoài** transaction
của use case (xem R9).

---

## R4 — `createdByAdmin` không lấy được từ dữ liệu, phải lưu vào cột

**Quyết định**: thêm cột `created_by_admin boolean` vào `identity.users` (migration Goose), mặc
định `false`. Đường quản trị viên ghi `true`; hai đường tự phục vụ ghi mặc định. Sự kiện mang giá
trị này, và cơ chế quét bù đọc nó từ cơ sở dữ liệu.

**Lý do — BẰNG CHỨNG**: `entities.User` (`identity/domain/entities/user.go:15-35`) **không có trường
nào ghi ai đã tạo**. Hai đường tạo tài khoản dựng hai hàng giống hệt nhau về cấu trúc:

| Trường | `register_uc.go:166-179` (tự đăng ký) | `admin_uc.go:198-210` (quản trị viên) |
|---|---|---|
| `EmailVerified` | `true` | `false` |
| `PasswordHash` | có | có |
| `Provider` | `local` | `local` |
| trường "người tạo" | **không có** | **không có** |

Cờ này chỉ lấy được từ **đường gọi**. Nhưng cơ chế quét bù (FR-079) đọc từ cơ sở dữ liệu, nên nếu
không lưu thì nó **không phân biệt được** tài khoản test của quản trị viên với tài khoản khách
hàng — và sẽ cấp quà cho tài khoản test, vi phạm FR-050 và FR-091.

**Tiền lệ có sẵn**: 024 đã làm đúng việc này cho một quy tắc cùng dạng. Ghi chú trong
`services/account/dto/identity/contract_dto.go:19-24`:

> *"the subscription module owns the rule 'an administrator must not be a grant target', but it
> does not own identity.users. Carrying the role on the contract is what lets it enforce that rule
> without reading another module's table."*

⇒ Cột này là thông tin vĩnh viễn và hữu ích của tài khoản (báo cáo, hỗ trợ, phân tích gian lận),
không phải trường phục vụ chiến dịch tạm thời. Nó được giữ lại cùng với toàn bộ cơ chế.

**Phương án đã xét**:
- Suy ra từ `email_verified = false` — bị loại: là **hai sự thật khác nhau**; quản trị viên hoàn toàn
  có thể tạo tài khoản đã xác minh, và tài khoản tự đăng ký luôn đã xác minh. Suy ra sai là hỏng
  âm thầm, đúng loại lỗi mà FR-090 nêu là không được phép.
- Chỉ truyền qua thông báo, không lưu — bị loại: cơ chế quét bù đọc cơ sở dữ liệu, không đọc thông báo.

---

## R5 — Cơ chế quét bù lấy danh sách tài khoản qua khoảng hẹp trong hợp đồng

**Quyết định**: khai báo một interface hẹp **riêng** trong
`services/account/contract/identity`, không mở rộng `IUserBriefContract`:

```go
type CampaignAccountLister interface {
    ListCreatedInWindow(ctx context.Context, filter CampaignAccountFilter) (*CampaignAccountPage, error)
}

func NewCampaignAccountLister(c *UserBriefContract) CampaignAccountLister
```

`CampaignAccountPage.Items` kiểu `[]*dto.UserBriefContractDTO`, **không** phải
`[]*entities.User`. Bổ sung hai trường vào `UserBriefContractDTO`: `CreatedAt time.Time`,
`CreatedByAdmin bool`. Phân trang bằng khoá tổ hợp `(created_at, id)` để không bỏ sót khi nhiều tài
khoản có cùng mốc thời gian.

**Lý do — BẰNG CHỨNG**: module `subscription` **đã** phụ thuộc hợp đồng này
(`purchase.go:7,40`, `admin_user_resolver.go:8`, `provider.go:4`). `IUserBriefContract` hiện có 4
phương thức (`GetByID`, `GetByIDs`, `GetByUsernamesOrEmails`, `SearchUserBriefs`) và **không có**
phương thức nào liệt kê theo thời điểm tạo.

**Vì sao không mở rộng `IUserBriefContract`** — đây là chỗ thiết kế ban đầu đã sai:

- Hợp đồng đó **được phục vụ qua gRPC**. Adapter client
  `contracts/identity/v1/client/user_brief_client.go:19,24` có khẳng định biên
  dịch `var _ IUserBriefContract = (*UserBriefClient)(nil)`, và có adapter server
  `services/account/internal/modules/identity/presentation/grpc/server.go:31`. Thêm phương thức mới bắt buộc phải sửa `identity.proto` và cả hai adapter.
- Đó là thay đổi hợp đồng **xuyên dịch vụ**, trái FR-058 của chính tính năng này.
- Tiền lệ 024 đã giải đúng bài toán giống hệt: `AdminGrantLookup` là interface hẹp riêng, có
  `NewAdminGrantLookup(c *UserBriefContract)`, khai trong cùng tệp hợp đồng
  (`user_brief_contract_impl.go:42,57`). `provisioning_contract.go:24` ghi rõ đó là cách tách cố ý
  để giữ hợp đồng gRPC nguyên vẹn.

**Vì sao trả DTO chứ không trả `entities.User`**: hiến phán §II — *"using DTOs, never domain
entities"*, và L65-66 cấm struct pointer tới thực thể của module khác. Thiết kế đầu tiên trả
`[]*entities.User` là vi phạm. `UserBriefContractDTO` đã có sẵn `ID` và sau khi bổ sung hai trường
nói trên thì đủ đúng ba giá trị mà quét bù cần, nên không phải khai báo DTO mới.

**Phương án đã xét**:
- Module `subscription` đọc thẳng bảng `identity.users` — bị loại: vi phạm hiến phán §II (cấm import
  `domain/entities` của module khác), và là vi phạm mức `fail` của quy tắc H4.
- Quét bảng hàng đỗi (`shared.outbox_records`) theo chủ đề — bị loại: buộc cơ chế chiến dịch phụ thuộc
  lược đồ bảng của hạ tầng khác và phải phân tích JSON thô để lấy mã tài khoản.
- Ghi một dòng "ứng viên" cho **mọi** tài khoản khi tạo — bị loại: cơ chế hướng publisher cố ý
  không tham gia vào lúc tạo tài khoản (FR-085); thêm bước ghi này là phá vỡ chính quyết định đó.

**Quy mô quét có kiểm soát**: cửa sổ quét bị chặn trong `[startsAt, endsAt]` của chiến dịch và chia
lô. Với chiến dịch ra mắt 50 suất, tổng số tài khoản phải duyệt là con số nhỏ. Giả định này được
ghi trong `spec.md`; nếu sau này một chiến dịch chạy quy mô hàng trăm nghìn tài khoản thì cách
quét này cần thay.

---

## R6 — Múi giờ: dùng cấu hình sẵn có, tuyệt đối không viết cứng

**Quyết định**: mọi phép so sánh thời gian dùng `timeutils.GetNow(cfg.Database.TimeZone)`.
`startsAt` và `endsAt` phân tích theo RFC3339 **có múi giờ** ngay lúc nạp cấu hình.

**Lý do — BẰNG CHỨNG**: `shared/config/config.go:238` đã có `Database.TimeZone`, và cả hai tệp
cấu hình đều đặt `time_zone: Asia/Ho_Chi_Minh` (`configs/config.yaml:70`,
`configs/config.production.yaml:70`). Tiền lệ sử dụng: `subscription_uc.go:115`.

**Vì sao quan trọng**: FR-008 đòi mốc 00:00 ngày 02/10/2026 theo giờ Việt Nam hiểu như nhau ở mọi
môi trường. Nếu dùng `time.Now()` thì máy chủ đặt múi giờ UTC sẽ chạy chiến dịch lệch 7 giờ.
Hiến phán §VIII cấm viết cứng giá trị phụ thuộc môi trường.

---

## R7 — Không thêm loại sự kiện đổi gói mới; dịch vụ AI không bị đụng tới

**Quyết định**: dùng chung tập loại hiện có. Không sửa `services/ai`. Không thêm consumer mới
bên dịch vụ AI.

**Lý do — BẰNG CHỨNG**: tập loại là **đóng** (`shared/domain/constants/subscriptioneventtype/subscription_event_type.go`),
và bên nhận **từ chối vĩnh viễn** mọi loại ngoài tập rồi đưa vào hàng đỗi chết
(`grant_payload_parser.go:122`, trả về `PermanentError`). Chú thích ngay trong tệp enum nêu rõ:
*"an unknown type must never be guessed at, because guessing wrong writes a grant that should not
exist."*

Thêm nữa, `docs/system-design/architecture.md:414` khẳng định bảng quyền chi phí AI có **đúng một
nhà ghi**, là dịch vụ AI, và chỉ tới đó bằng cách tiêu thụ chủ đề đổi gói. Sự kiện đổi gói phát
lúc cấp phát gói mặc định là **tín hiệu duy nhất** để bảng đó có bản ghi cho tài khoản mới.

**Hệ quả cần kiểm thử bảo vệ**: một kiểm thử khẳng định bộ nhận thông báo của dịch vụ AI chỉ
tiêu thụ **đúng một** chủ đề. Nếu ai đó thêm một chủ đề nữa, trình biên dịch vẫn build và hệ thống
vẫn chạy — chỉ biến mọi tài khoản mới thành tài khoản không có quyền AI, đi qua đường dự phòng,
không ai báo lỗi.

---

## R8 — Loại sự kiện mà lần cấp gói chiến dịch thực sự ghi: `ACTIVATED`

**Quyết định**: không khẳng định loại sự kiện trong kiểm thử. Chỉ khẳng định **kết quả quan sát
được**: mã gói, thời hạn, cờ tự động gia hạn tắt, và cửa sổ quyền AI khớp thời hạn gói.

**Lý do — BẰNG CHỨNG**: logic chọn loại tại
`subscription/application/usecase/subscription/internal_subscription_apply_workflow.go:147-152`:
```go
eventType := ACTIVATED
switch {
case !isNewSub && fromPlanCode == plan.Slug:                                  → EXTENDED
case !isNewSub && fromTierRank > 0 && plan.TierRank > fromTierRank:           → UPGRADED
}
```
Tài khoản miễn phí có `fromTierRank = 0`, nên `0 > 0` là sai ⇒ rơi về `ACTIVATED`. Lần cấp gói
của quản trị viên từ gói miễn phí cũng cho kết quả giống hệt.

**Vì sao không sao**: dịch vụ AI xử lý `ACTIVATED` đúng như cần — tạo grant mới, đóng grant cũ,
xoá grant chờ. Hành vi đúng; chỉ là tên loại sự kiện gây nhầm lẫn. Viết kiểm thử đoán tên loại sẽ
tạo ra một bài kiểm thử **sai** mà lại xanh.

---

## R9 — Sửa race bộ đếm lượt dùng AI: bọc đúng đoạn, không bọc cả hàm

**Quyết định**: giữ `UserQuotaUseCase` chỉ nhận 4 phụ thuộc như hiện tại **cộng** `uow`; bọc
`uow.Execute` quanh **đúng đoạn đọc–tăng–ghi** trong `UpdateOutfitQuota` và `UpdateAiChatQuota`,
dùng `GetByUserIDWithLock` thay `GetByUserID`. Hai lệnh đọc `GetOrCreateUserSubscription` và
`LoadPlanForSubscription` chạy **ngoài** transaction.

**Lý do — BẰNG CHỨNG**: `quota_uc.go:90-113` hiện là 5 câu lệnh không transaction, và
`GetByUserIDWithLock` (`user_daily_quota_repo_impl.go:66-76`) đã tồn tại nhưng **không nơi nào trong
production gọi**. Chỉ thêm khoá mà không bọc transaction thì không được: khoá ở mức câu lệnh được
giải phóng ngay khi câu lệnh kết thúc, nên hai transaction vẫn đọc cùng một số.

**Bẫy cần tránh**: bọc cả hàm sẽ kéo `GetOrCreateUserSubscription` vào trong transaction, mà
hàm đó có thể tự mở transaction của riêng nó (xem R3 về việc `uow.Execute` lồng nhau mở connection
thứ hai).

**Đường đọc phải giữ nhẹ**: `GetAndResetDailyQuota` (`quota_uc.go:70-87`) là pre-check được gọi ở
**mọi** lần dùng AI. Chỉ mở transaction khi thực sự sang ngày mới; nếu không thì giữ nguyên đường
đọc không khoá. Không làm vậy thì mỗi lần dùng AI đều phải chờ một hàng khoá, vi phạm FR-055.

---

## R10 — Hình dạng cấu hình chiến dịch: danh sách, không phải bản đồ

**Quyết định**: `campaigns` là một **danh sách** trong cấu hình. Mỗi phần tử có: `code`,
`plan_slug`, `quota`, `reserve`, `starts_at`, `ends_at`. Kiểm tra lúc khởi động theo FR-010.

**Lý do**: FR-011 chọn chiến dịch **bắt đầu muộn nhất** trong số những chiến dịch tài khoản đủ
điều kiện. Với một danh sách, thứ tự ưu tiên là một phép so sánh thuần tuý trên `startsAt` — không
phụ thuộc thứ tự khai báo trong tệp. Với một bản đồ, sẽ phải quy định thêm thứ tự duyệt khoá, và
một cấu hình viết sai thứ tự sẽ cho kết quả khác.

**Múi giờ bắt buộc**: `starts_at` và `ends_at` là chuỗi RFC3339 **có múi giờ**
(`2026-10-02T00:00:00+07:00`). Cấu hình không tự bỏ qua phần múi giờ, vì FR-008 đòi mọi môi trường
hiểu giờ giống nhau.

**Ghi đè biến môi trường**: chỉ áp dụng cho các giá trị đơn lẻ. Danh sách chiến dịch chỉ khai
báo trong tệp cấu hình — nêu rõ trong `.env.example` (tiếng Anh) rằng danh sách nằm ở tệp YAML,
để người đọc không tìm nhầm biến môi trường không tồn tại.

---

## R11 — Bẫy định tuyến Gin: nhóm mới bắt buộc, không được đặt cạnh tham số

**Quyết định**: đăng ký hai đường dẫn tra cứu trong **nhóm riêng** `/admin/campaigns`.

**Lý do — BẰNG CHỨNG**: nhóm `adminSubs` (`/admin/subscriptions`) đang giữ
`POST /:userId/cancel`. Gin **không** cho đăng ký một đoạn tĩnh cạnh một tham số ở cùng cấp; xung
đột này làm **hỏng ngay lúc khởi động**, cả dịch vụ Account sẽ không boot được. Tiền lệ đã bị
dính và đã ghi lại tại `subscription/presentation/router/router.go:96-103` (024 R5).

**Lưu ý thêm**: `router.go` dùng `RolesAuthorize(roleslug.Admin)` cho mọi nhóm quản trị. FR-070
được đáp ứng bằng đúng middleware này, không cần viết kiểm tra vai trò riêng.

**Giới hạn tần suất (FR-071)**: tái dùng đúng `AdminGrantRateLimitMiddleware` đang gắn lên
đường cấp gói hàng loạt. Đây là giới hạn tần suất theo quản trị viên, đúng nghĩa FR-071
("một vòng dò không làm tê liệt hệ thống"). Tái dùng cơ chế có sẵn, không mở thêm cấu hình.

---

## R12 — Tên chủ đề, tên hàng đỗi, và ranh giới hàng đỗi

**Quyết định**:
- Chủ đề: `identity.user.created`
- Hàng đỗi: `campaign_grant_user_created_queue`, ràng buộc với chính chủ đề đó
- Quorum, `x-delivery-limit: 5`, `Prefetch 20`, `Concurrency 1`

**Lý do đặt tên theo tính năng, không theo chiến dịch**: FR-090 nói một tính năng sau này chỉ cần
viết thêm bộ nhận thông báo, không sửa luồng đăng ký. Điều đó chỉ đúng nếu hàng đỗi đặt theo
**bên nhận thông báo**, không theo chiến dịch. Tên hàng đỗi phải là hằng số tĩnh, nên không thể
đặt theo mã chiến dịch — đây là lý do cấu trúc, không phải lựa chọn phong cách.

**Về ranh giới hàng đỗi**: `topology.go:17` khai báo `RoutingKeys []string`, tức một hàng đỗi có thể
gắn nhiều định tuyến. Sau này khi thêm bên nhận thông báo thứ hai cho cùng chủ đề (email chào mừng,
thưởng giới thiệu), mỗi bên nhận một hàng đỗi **riêng** — đó là mô hình phát/thu, và mỗi bên xử
lý độc lập, không chen nhau. Mẫu có sẵn: `fashion.item.created` tách `.wardrobe` và `.brand`.

**Tên mẫu email**: `campaign_grant`, **không** phải `launch_promo`. Cùng lý do: chiến dịch ra mắt là
chiến dịch đầu tiên, không phải chiến dịch duy nhất. Một mẫu tên theo buổi ra mắt sẽ phải được
dùng lại cho các chiến dịch sau với một cái tên nói dối. `campaign_grant` mô tả đúng việc nó làm:
thông báo cho tài khoản vừa được trao quyền lợi bởi chiến dịch.

Mẫu `admin_grant` đã có sẵn trong hệ thống không dùng lại được: nội dung nó viết cho đường cấp gói
thủ công, và nó phải giữ nguyên vì đường đó vẫn còn người dùng. Một mẫu riêng là cần thiết, và
`admin_grant` không thể trở thành mẫu chung.

---

## R13 — Lần cấp gói phải đọc lại bản ghi gói trước khi phát sự kiện

**Quyết định**: sau `ApplySubscriptionPlanWithEvent`, **đọc lại** bản ghi gói rồi mới gọi
`PublishPlanChange`. Không dùng con trỏ đã bị hàm đó sửa.

**Lý do — BẰNG CHỨNG**: `BuildPlanChangedPayload` lấy `effectiveTo` từ `sub.ExpiresAt`
(`application/mapper/plan_changed_payload.go:102-104`), còn `ApplySubscriptionPlanWithEvent` chỉ
**trả về** bản ghi sự kiện chứ không trả về bản ghi gói đã cập nhật. Đường cấp gói của quản trị viên
đã ghi rõ lý do tại `admin_grant_batch_uc.go:442-447`: truyền nhầm bản ghi cũ làm `effectiveTo`
rỗng, tức quyền AI **vô thời hạn** cho một tài khoản chỉ nên có 1 tháng.

**Phát hiện thêm cùng lúc**: lần cấp gói của chiến dịch mang loại sự kiện `ACTIVATED`, **không
phải** `UPGRADED` (xem R8). Hành vi đúng, nhưng cần biết để không viết kiểm thử đoán sai.

---

## R14 — Phạm vi công việc: tách thành năm nhóm, giữ nguyên ranh giới

**Quyết định**: tách công việc thành năm nhóm theo thứ tự phụ thuộc.

| Nhóm | Nội dung | Vì sao tách |
|---|---|---|
| **A** | Sửa race bộ đếm lượt dùng AI | Độc lập, không liên quan chiến dịch. Phải có trước vì chiến dịch làm lượt dùng AI tăng vọt ngay ngày đầu — sửa sau thì ngày đầu tiên vẫn đếm sai. |
| **B** | Sự kiện tạo tài khoản + cột `created_by_admin` | Nền cho cả chiến dịch lẫn mọi tính năng tương lai. Độc lập với cơ chế chiến dịch. |
| **C** | Cơ chế chiến dịch: bảng lượt cấp, bộ luật hạn mức, worker, quét bù | Lõi của tính năng. |
| **D** | Mẫu email + thông báo | Độc lập về mã, phụ thuộc C về dữ liệu. |
| **E** | Giao diện tra cứu cho quản trị viên | Khối lượng lớn nhất, **cắt bỏ được** mà không ảnh hưởng C, D. Có thể dời ra sau ngày ra mắt nếu cần. |

**Về việc nhóm E có thể cắt**: FR-062 đã đưa giao diện tra cứu ra ngoài phạm vi với khách hàng,
và bộ phận hỗ trợ vẫn trả lời được bằng dữ liệu lượt cấp. Nhưng SC-011 đo thời gian trả lời
dưới 1 phút — cần nhóm E để đạt. Nếu phải chọn giữa đạt đúng hạn ngày 02/10 và đạt SC-011, chọn
hạn ngày 02/10.
