# Kế Hoạch Triển Khai: Cơ chế chiến dịch tặng gói cho tài khoản đăng ký mới

**Nhánh**: `025-signup-campaign-grant` | **Ngày**: 2026-10-01 | **Đặc tả**: [spec.md](./spec.md)
**Nghiên cứu**: [research.md](./research.md) | **Mô hình dữ liệu**: [data-model.md](./data-model.md)

**Đầu vào**: Đặc tả từ `/speckit-specify` + 6 quyết định làm rõ từ `/speckit-clarify`

---

## Tóm tắt

Xây một **cơ chế chiến dịch dùng lại được** trao quyền lợi gói cước cho tài khoản đăng ký mới.
Chiến dịch đầu tiên phục vụ buổi ra mắt: **50 tài khoản** đủ điều kiện sớm nhất, tổng hạn mức 50
cộng hạn mức dự phòng 5, gói `premium-monthly` 1 tháng, không tự động gia hạn, cửa sổ mở từ
`2026-10-02T00:00:00+07:00`.

**Cách tiếp cận kỹ thuật**: luồng tạo tài khoản phát **vô điều kiện** một sự kiện
`identity.user.created` — một sự thật về chính module tài khoản, không phải tín hiệu cho chiến dịch.
Bộ nhận thông báo của chiến dịch lắng nghe chủ đề đó, áp dụng bộ luật hạn mức trong **một
transaction** có khoá tư vấn, rồi tái dùng đường thay đổi gói sẵn có để cấp gói. Tính đúng đắn
không dựa vào sự kiện mà vào cơ chế quét bù chạy định kỳ.

Kèm trong phạm vi: sửa lỗi race sẵn có ở bộ đếm lượt dùng AI hằng ngày, và giao diện tra cứu
chỉ đọc cho quản trị viên.

---

## Bối Cảnh Kỹ Thuật

**Ngôn ngữ / Phiên bản**: Go 1.26.0 (không gian làm việc đa mô-đun, `go.work`)

**Phụ thuộc chính**:
| Thư viện | Vai trò trong tính năng này |
|---|---|
| `gorm.io/gorm` + `driver/postgres` | Lưu trữ, khoá tư vấn, `ON CONFLICT` chống trùng |
| `github.com/rabbitmq/amqp091-go` | Vận chuyển sự kiện qua hàng đỗi quorum |
| `github.com/google/wire` | Lắp kết phụ thuộc cho worker và use case mới |
| `github.com/robfig/cron/v3` | Cơ chế quét bù chạy định kỳ |
| `github.com/google/uuid` | Định danh tài khoản và mốc ưu tiên |
| `go.uber.org/zap` | Nhật ký có cấu trúc cho worker |
| `github.com/spf13/viper` + `godotenv` | Nạp cấu hình chiến dịch |

**Lưu trữ**: PostgreSQL — lược đồ `identity` (tài khoản), `subscription` (gói cước, lượt cấp),
`shared` (hàng đỗi sự kiện), `aiexecutor` (quyền chi phí AI, **không đụng tới**). Redis chỉ dùng
cho bộ nhớ đệm. RabbitMQ làm hàng đỗi.

**Kiểm thử**: `go test` + `testify`; lệnh `make wire && make lint && make test-race`. Có kiểm thử
tích hợp tấn công cơ sở dữ liệu thật cho đường cấp phát và đường hạn mức.

**Nền tảng đích**: máy chủ Linux qua Docker Compose (3 dịch vụ) và nền tảng quản lý. Chỉ dịch vụ
`account` thay đổi.

**Loại dự án**: phía sau, không giao diện. Go workspace đa mô-đun, 3 dịch vụ triển khai độc lập.

**Mục tiêu hiệu năng**:
| Chỉ số | Mục tiêu | Nguồn |
|---|---|---|
| Đăng ký → có gói (đường bình thường) | 95% trong **5 phút** | SC-015 |
| Đăng ký → có gói (kể cả gián đoạn) | 100% trong **1 giờ** | SC-016 |
| Cấp gói → email | trong **60 giây** | SC-005 |
| Độ trễ thêm cho yêu cầu đăng ký | **không quá 10%** | SC-004 |

**Ràng buộc**:
- Tuyệt đối không vượt `quota + reserve` (trần cứng 55), kể cả nhiều phiên bản chạy song song — FR-022
- Sự kiện tạo tài khoản **không chứa dữ liệu nhận diện cá nhân** — FR-087
- **0 đồng** bị trừ — SC-006
- **Không sửa dịch vụ AI** — FR-097 (xem R7)
- Không thêm loại sự kiện đổi gói mới — FR-092
- Không thay đổi hợp đồng giao tiếp gọi từ dịch vụ khác — FR-058

**Quy mô**: chiến dịch đầu 50 suất; cơ chế dùng lại cho các chiến dịch sau bằng khai báo cấu hình;
khoảng **55 tệp** bị chạm tới, trong đó 1 dịch vụ, 2 module, 3 migration dữ liệu.

---

## Kiểm Tra Hiến Pháp

*Cổng: phải qua trước Phase 0. Kiểm tra lại sau Phase 1.*

| Điều khoản | Yêu cầu | Cơ chế đáp ứng | Kết quả |
|---|---|---|---|
| §I Dịch vụ triển khai độc lập | Mỗi dịch vụ build và chạy độc lập | Chỉ `services/account` thay đổi. `services/ai` và `services/commerce` **không đổi dòng code nào**. Thêm 1 kiểm thử ở `shared` (R7) để canh, không phải thay đổi hành vi. | ✅ |
| §II Ranh giới module | Cấm import `domain/entities`, `domain/repositories`, `infrastructure/**` của module khác | Cơ chế chiến dịch nằm trong module `subscription`, lấy dữ liệu tài khoản qua `services/account/contract/identity` (R5) — đúng đường module `purchase.go` đã dùng. Cột `created_by_admin` được đọc qua hợp đồng, không đọc thẳng bảng. | ✅ |
| §II Dùng lại hợp đồng không nhân bản | Không khai báo lại cấu trúc đã có ở module cung cấp | Dùng `UserBriefContractDTO` sẵn có, bổ sung 2 trường — không tạo DTO mới cho cùng một thực thể. | ✅ |
| §III Xử lý sự kiện bất đồng bộ | Mọi phát hành sự kiện dùng mẫu outbox | `IEventPublisher.Publish` trong cùng transaction tạo tài khoản → ghi `shared.outbox_records`. Không có đường phát trực tiếp nào. | ✅ |
| §III Worker là bộ chuyển đổi presentation | Worker ở `presentation/worker/`, mỏng, ủy quyền cho use case | `CampaignGrantWorker` và `CampaignSweepWorker` chỉ gọi use case. `BaseWorker` + `RegisterTypedConsumer` + `workerlog`. | ✅ |
| §IV Phân lớp hướng vào trong | presentation → application → domain ← infrastructure | Worker không import infrastructure (nhận `event.IMessageConsumer` qua ràng buộc Wire). Use case chỉ dùng khoá hợp đồng của chính mình. | ✅ |
| §IV Bỏ qua tầng bị cấm | presentation không gọi thẳng repository | Worker gọi `ICampaignUseCase`; use case gọi khoá. | ✅ |
| §V Lỗi tập trung | `apperror` factory, thông điệp tiếng Việt | Lỗi chiến dịch đặt trong `application/errors/`, thông báo client tiếng Việt. | ✅ |
| §VI Lõi kiểm soát chi phí AI | Không sửa ngoài phạm vi đã nêu | R7: dịch vụ AI **không bị sửa**. Chỉ tái dùng đường thay đổi gói sẵn có. | ✅ |
| §VII Tuân thủ design-pattern.md | Đọc và tuân thủ | Cấu trúc lớp, mẫu worker, mẫu phân trang, mẫu lỗi, mẫu khoá đơn vị công việc, quy ước tên DTO — lấy nguyên văn từ tài liệu. | ✅ |
| §VIII Khả chạy trên mọi hạ tầng | Không viết cứng giá trị phụ thuộc môi trường | Múi giờ lấy từ `cfg.Database.TimeZone` (R6). Toàn bộ chiến dịch khai báo bằng cấu hình, ghi đè được bằng biến môi trường cho giá trị đơn lẻ (R10). | ✅ |
| Hiến phán — Goose migration | Mọi thay đổi lược đồ qua tệp Goose | 3 tệp migration có `-- +goose Up` và `-- +goose Down`, lý do bằng tiếng Anh trích dẫn `025 T### (FR-###)`. | ✅ |
| Hiến phán — Ngôn ngữ | Thông điệp client tiếng Việt, mã và nhật ký tiếng Anh | Mẫu email và thông báo lỗi tiếng Việt; nhận ký và nhận xét mã tiếng Anh. | ✅ |
| code-hygiene H1 | Tệp production ≤ 500 dòng, hàm ≤ 80 dòng | Mọi tệp mới tách theo trách nhiệm; use case chiến dịch tách bước ra hàm nhỏ. | ✅ |
| code-hygiene H2 | Tối đa 2 kiểu có hành vi mỗi tệp | Mỗi tệp mới một struct chính. | ✅ |
| code-hygiene H4 | Không import chéo module, không bỏ qua tầng | Như §II và §IV. | ✅ |

**Kết luận cổng**: **ĐẠT**, không có ngoại lệ nào cần ghi vào Bảng Theo Dõi Độ Phức Tạp.

### Kiểm tra lại sau Phase 1

Sau khi sinh mô hình dữ liệu và hợp đồng, đối chiếu lại ba điều khoản dễ vi phạm nhất:

| Điều khoản | Cần xác nhận lại | Kết quả |
|---|---|---|
| §II | Quét bù lấy danh sách tài khoản mà không kéo theo module khác đọc thẳng bảng `identity.users`, **và** không đụng hợp đồng gRPC | ✅ Khoảng hẹp `CampaignAccountLister` trả **DTO**, không trả `entities.User`; `IUserBriefContract` và `identity.proto` **không đổi một dòng** |
| §VIII | `created_by_admin` là dữ liệu, không phải cấu hình — không viết cứng giá trị `true` ở bất kỳ tệp cấu hình nào | ✅ Chỉ được đặt ở đúng một đường ghi trong mã |
| Hiến phán §II | Mốc ưu tiên lưu trong bảng của `subscription`, không đọc chéo | ✅ |

---

## Cấu Trúc Dự Án

### Tài liệu (tính năng này)

```text
specs/025-signup-campaign-grant/
├── spec.md                     # Đặc tả — nguồn sự thật về phạm vi
├── plan.md                     # Tệp này
├── research.md                 # 14 quyết định thiết kế kèm bằng chứng
├── data-model.md               # Lược đồ, thực thể, chuyển trạng thái
├── quickstart.md               # Cẩu nang kiểm chứng chạy tay
├── contracts/
│   ├── identity-contract.md    # Hợp đồng mở rộng: liệt kê tài khoản theo cửa sổ thời gian
│   ├── account-created-event.md # Hợp đồng sự kiện tạo tài khoản
│   └── campaign-query-api.md   # Hợp đồng giao diện tra cứu cho quản trị viên
├── checklists/
│   └── requirements.md         # Đã đạt 19/19
└── tasks.md                    # Sinh ở /speckit-tasks — đã tạo, 119 nhiệm vụ
```

### Mã nguồn (thư mục gốc dự án)

```text
shared/
├── config/
│   ├── config.go                       # + CampaignConfig
│   ├── loader.go                       # + phân tích startsAt/endsAt, ghi đè biến môi trường
│   └── validation.go                   # + kiểm tra cấu hình chiến dịch
├── application/constants/eventconstants/
│   ├── eventconstants.go               # + TopicUserCreated
│   └── queues.go                       # + QueueCampaignGrantUserCreated
└── infrastructure/messaging/
    ├── rabbitmq_constants.go           # + định tuyến cho chủ đề mới
    ├── topology.go                     # + 1 liên kết hàng đỗi + 1 liên kết .dlq
    └── topology_test.go                # + canh FR-097: dịch vụ AI chỉ tiêu thụ 1 chủ đề

services/account/
├── database/migrations/
│   ├── <timestamp>_add_created_by_admin.sql
│   ├── <timestamp>_add_campaign_claims.sql
│   └── <timestamp>_add_campaign_sweep_runs.sql   # T079a — FR-059, xem data-model.md mục 5.1
├── dto/                                  # quy ước DTO của dịch vụ account (xem ghi chú bên dưới)
│   ├── identity/
│   │   ├── contract_dto.go              # + CreatedAt, + CreatedByAdmin
│   │   └── event_payload.go             # MỚI — thông điệp identity.user.created
│   └── subscription/
│       └── campaign_dto.go              # MỚI — dữ liệu tra cứu cho quản trị viên
├── contract/identity/                    # + liệt kê tài khoản theo cửa sổ thời gian
├── internal/modules/identity/
│   ├── domain/entities/user.go          # + trường CreatedByAdmin
│   ├── domain/repositories/
│   │   ├── user_repo.go                 # + GetCreatedInWindowPaginated
│   │   └── readmodels.go                # + GetCreatedInWindowFilter
│   ├── infrastructure/persistence/
│   │   └── user_repo_impl.go
│   └── application/usecase/
│       ├── register_uc.go               # + phát thông điện trong transaction
│       ├── google_auth_uc.go            # + phát thông điện trong transaction
│       └── admin_uc.go                  # + đặt cờ + phát thông điện
├── internal/modules/subscription/
│   ├── domain/entities/
│   │   ├── campaign_claim.go            # MỚI
│   │   └── table_name.go                # + tên bảng
│   ├── domain/repositories/
│   │   └── campaign_claim_repo.go       # MỚI
│   ├── infrastructure/persistence/
│   │   └── campaign_claim_repo_impl.go  # MỚI
│   ├── application/
│   │   ├── errors/errors.go             # + lỗi chiến dịch
│   │   ├── mapper/campaign_mapper.go     # MỚI
│   │   ├── interface/usecase/
│   │   │   └── campaign_uc.go           # MỚI — giao diện
│   │   └── usecase/campaign/
│   │       ├── campaign_uc.go           # MỚI — điều phối
│   │       ├── admission_uc.go          # MỚI — chọn chiến dịch + điều kiện
│   │       ├── quota_uc.go              # MỚI — khoá, đếm, chốt mốc
│   │       ├── grant_uc.go              # MỚI — cấp gói + thông báo
│   │       └── sweep_uc.go              # MỚI — quét bù
│   └── presentation/
│       ├── handler/campaign_admin_handler.go   # MỚI
│       ├── router/router.go             # + nhóm /admin/campaigns
│       └── worker/
│           ├── campaign_grant_worker.go # MỚI — bộ nhận thông báo
│           └── campaign_sweep_worker.go # MỚI — quét bù theo lịch
└── internal/modules/notification/
    ├── domain/constants/template/template.go    # + CampaignGrant
    ├── application/templates/
    │   ├── templates.go                 # + nhúng tệp
    │   └── CampaignGrant.html           # MỚI
    └── application/usecase/email/outbox_email_uc.go  # + mục đệm + tiêu đề

configs/
├── config.yaml                          # + khối campaigns
└── config.production.yaml               # + khối campaigns
.env.example                             # + ghi chú biến đơn lẻ (tiếng Anh)
.env.production.example                  # + ghi chú biến đơn lẻ (tiếng Anh)
```

**Quyết định cấu trúc**: giữ nguyên bố cục kiến trúc phân lớp của dự án
(`domain` → `application` → `presentation` → `infrastructure`, mỗi module một cây). Cơ chế chiến
dịch **không** tách thành một module mới, vì nó dùng chung bảng gói cước, chung đường thay đổi gói
và chung hợp đồng tài khoản với module `subscription`. Tách module mới sẽ tạo ra bốn hợp đồng
mới chỉ để bọc lại những thứ đã có sẵn. Phần dùng lại được nhất — `ApplySubscriptionPlanWithEvent`
và `PlanChangePublisher` — nằm trong package `subscription` và **không** xuất khỏi `internal/`, nên
module mới cũng không thể dùng lại chúng.

**Ghi chú vị trí DTO — sai lệch có chủ ý so với tài liệu**: `design-pattern.md` quy định DTO ở
`internal/modules/<module>/application/dto/`. Dịch vụ `account` **không** theo quy ước đó: các
module `identity`, `subscription`, `admin` đặt DTO ở `services/account/dto/<module>/`, còn module
`notification` mới theo tài liệu. Tính năng này **theo quy ước đang dùng của các module nó chạm
tới** (`dto/<module>/`), vì tính nhất quán bên trong một dịch vụ quan trọng hơn là khớp tài liệu khi
dịch vụ đó đang chuyển đổi. Sự lệch này đã có sẵn và không phải do tính năng này tạo ra.

**Về thư mục `usecase/campaign/`**: tách thành năm tệp theo bước nghiệp vụ, thay vì một tệp. Quy tắc
H1 giới hạn 500 dòng mỗi tệp, và một tệp duy nhất chứa chọn chiến dịch, khoá hạn mức, cấp gói và
quét bù sẽ vượt ngưỡng ngay từ đầu.

---

## Bảng Theo Dõi Độ Phức Tạp

**Không có mục nào.** Kiểm tra cổng hiến phán ở trên không phát hiện ngoại lệ nào cần biện minh.

Hai điểm từng được cân nhắc là ngoại lệ và **không** phải ngoại lệ:

| Điểm nghi | Vì sao không phải ngoại lệ |
|---|---|
| Thêm cột `created_by_admin` vào bảng tài khoản cho một tính năng tạm thời | Đây là thông tin vĩnh viễn và hữu ích của tài khoản, không phải trường phục vụ chiến dịch. Nó làm cho FR-091 thực thi được bằng cả hai đường xử lý (thông báo và quét bù), và tiền lệ đã có trong 024 (mang `RoleSlug` trên hợp đồng để module `subscription` thực thi quy tắc mà không đọc bảng của module khác). |
| Mở rộng `IUserBriefContract` | **Bị loại**: hợp đồng đó được phục vụ qua gRPC và có adapter client (`contracts/identity/v1/client/user_brief_client.go:24` khẳng định biên dịch `var _ IUserBriefContract = (*UserBriefClient)(nil)`) lẫn adapter server. Thêm phương thức bắt buộc phải sửa `identity.proto` — tức đổi hợp đồng **xuyên dịch vụ**, trái FR-058. 024 đã có tiền lệ `AdminGrantLookup` làm đúng việc này, xem `research.md` R5. |
| Trả `[]*entities.User` qua hợp đồng | **Bị loại**: hiến phán §II — *"using DTOs, never domain entities"*, và L65-66 cấm struct pointer tới thực thể của module khác. Thiết kế ban đầu vi phạm; `UserBriefContractDTO` đã đủ dùng. |

---

## Lộ Trình Thực Thi

Năm nhóm, thứ tự phụ thuộc cứng. Chi tiết trong `tasks.md` (sinh ở `/speckit-tasks`).

| Nhóm | Nội dung | Phụ thuộc |
|---|---|---|
| **A** | Sửa race bộ đếm lượt dùng AI | Không |
| **B** | Sự kiện tạo tài khoản + cột `created_by_admin` | Không |
| **C** | Cơ chế chiến dịch: bảng lượt cấp, bộ luật hạn mức, worker, quét bù | B |
| **D** | Mẫu email + thông báo | C (về dữ liệu) |
| **E** | Giao diện tra cứu cho quản trị viên | C |

**Có thể cắt nếu cần**: nhóm E. Nó là khối lượng lớn nhất và **không** ảnh hưởng hạn ngày 02/10 —
bộ phận hỗ trợ vẫn trả lời được bằng dữ liệu lượt cấp. Đánh đổi là mất SC-011.

## Bước Kiểm Chứng

Cẩu nang chạy tay: [quickstart.md](./quickstart.md). Tám kịch bản bắt buộc:

| Kịch bản | Chứng minh điều gì |
|---|---|
| K2.1 — 55 tài khoản, thông báo tới **ngược thứ tự** | Trần 55 không bị vượt; **không lượt cấp dự phòng nào** tạo sau mốc ưu tiên; đếm `grant_bucket` tách đúng phần chính và dự phòng (SC-001, SC-001b, SC-001c) |
| K2.2 — 100 lần giao lại **cùng một** thông báo | Cấp đúng một lần; thời hạn vẫn 1 tháng (SC-002) |
| K2.3 — 5 phiên bản chạy song song, 20 thông báo, còn 1 suất | Đúng 1 tài khoản được cấp (SC-003) |
| K2.4 — Tắt worker 10 phút rồi bật lại | Quét bù đóng sót; không tài khoản nào mất quyền lợi (SC-016) |
| K2.5 — Cùng bộ dữ liệu, thông báo tới **thuận thứ tự** | `main_count` = 50 và `watermark_rows` = 1 **giống hệt** K2.1 — đó mới là tiêu chí so sánh; `total` khác (50 so với 55) là hành vi đúng, xem `spec.md` mục 'Giới hạn đã biết' (SC-001d) |
| K2.6 — Cấp gói rồi để hết hạn | Tự hạ về `free`, cờ tự gia hạn vẫn tắt, **0 đồng bị bị trừ** (SC-007, SC-006) |
| K2.7 — Cấp gói lúc đang có số dư | Cấp gói không trừ đồng bị (SC-006) |
| K2.8 — Đo độ chậm trước và sau khi bật | Không đoạn nào chậm hơn 10%; thời gian giao dịch lượt cấp không tăng theo số lượt đã cấp (SC-004) |
