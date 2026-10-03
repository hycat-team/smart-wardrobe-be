# Kế Hoạch Triển Khai: Sổ đăng ký chiến dịch trong cơ sở dữ liệu

**Nhánh**: `026-campaign-registry-db` | **Ngày**: 2026-10-03 | **Đặc tả**: [spec.md](./spec.md)
**Nghiên cứu**: [research.md](./research.md) | **Mô hình dữ liệu**: [data-model.md](./data-model.md)

**Đầu vào**: Đặc tả từ `/speckit-specify` + 4 quyết định làm rõ từ `/speckit-clarify`

---

## Tóm tắt

Đưa **việc chiến dịch nào tồn tại** ra khỏi tệp cấu hình (được biên dịch vào image) và vào bảng
`subscription.campaigns`, để bốn thao tác vận hành — mở, sửa hạn mức/thời gian, đóng sớm, và tra
nhật ký ai-đổi-gì — trở thành **thao tác dữ liệu có kiểm toán, không cần deploy**.

**Cách tiếp cận kỹ thuật**:
- `subscription.campaigns` là **nguồn sự thật duy nhất**. Khối `campaigns` bị bỏ khỏi `config.yaml`
  và `config.production.yaml`; kiểu `CampaignConfig` và các hàm kiểm tra lúc nạp bị xoá. Nạp cấu
  hình vẫn **không chạm CSDL**.
- `launch-2026-10` (đang chạy, quota 50, reserve 5) được **seed một lần** bằng migration idempotent
  với đúng giá trị production. `campaign_claims`/`campaign_sweep_runs` **không đổi lược đồ**, không
  thêm khoá ngoại.
- Bộ luật chọn chiến dịch, hạn mức, mốc ưu tiên, chống trùng và quét bù của 025 **giữ nguyên**;
  chỉ đổi nguồn danh sách chiến dịch từ cấu hình sang repository CSDL. Bộ đánh giá điều kiện 1–7
  vẫn **thuần túy** (nhận các hàng chiến dịch đã đọc).
- Bất biến `quota`/`reserve`/`plan_slug` khi đã có lượt cấp được giữ bằng **tầng ứng dụng + trigger
  `BEFORE UPDATE`**; đóng cưỡng bức thêm cột `closed_at` và trạng thái thứ sáu `closed`; ghi có
  **kiểm tra phiên bản** (If-Match).
- Ba endpoint ghi + một endpoint đọc nhật ký dưới `/api/v1/admin/campaigns`, dùng nhật ký kiểm toán
  sẵn có (qua outbox, cùng transaction ghi nghiệp vụ).

---

## Bối Cảnh Kỹ Thuật

**Ngôn ngữ / Phiên bản**: Go 1.26.0 (không gian làm việc đa mô-đun, `go.work`)

**Phụ thuộc chính**:
| Thư viện | Vai trò trong tính năng này |
|---|---|
| `gorm.io/gorm` + `driver/postgres` | Repository sổ chiến dịch; `SELECT ... FOR SHARE` cho FR-045; `UPDATE ... WHERE version = ?` cho If-Match |
| `github.com/google/wire` | Lắp kết repository, use case đăng ký, handler mới |
| `github.com/google/uuid` | Định danh, `ActorUserID` của nhật ký kiểm toán |
| `go.uber.org/zap` | Nhật ký có cấu trúc |
| `github.com/gin-gonic/gin` | Nhóm định tuyến `/api/v1/admin/campaigns` |
| `github.com/stretchr/testify` | Kiểm thử đơn vị + tích hợp |
| Goose (CLI) | Migration tạo bảng, trigger, seed dữ liệu |
| `github.com/spf13/viper` + `godotenv` | Nạp cấu hình — **giảm** phần chiến dịch |

**Lưu trữ**: PostgreSQL — lược đồ `subscription` (bảng mới `campaigns`; `campaign_claims` và
`campaign_sweep_runs` không đổi), `identity` (chỉ đọc), `shared` (`outbox_records`,
`admin_audit_logs`). Redis **không** dùng cho tính năng này (không cache sổ). RabbitMQ không đổi
(không chủ đề mới).

**Kiểm thử**: `go test` + `testify`; lệnh `make wire && make lint && go test -race ./...` scoped vào
`services/account/...` và `shared/config` cho mỗi phase; trước khi push dùng `make check`. Cần kiểm
thử tích hợp CSDL thật cho: trigger bất biến, `FOR SHARE`, If-Match, seed idempotent.

**Nền tảng đích**: máy chủ Linux qua Docker Compose (3 dịch vụ) và nền tảng quản lý. **Chỉ dịch vụ
`account` và module `shared/config` thay đổi.** `services/ai` và `services/commerce` **không đổi**.

**Loại dự án**: phía sau, không giao diện. Go workspace đa mô-đun.

**Mục tiêu hiệu năng**:
| Chỉ số | Mục tiêu | Nguồn |
|---|---|---|
| Tạo chiến dịch → có hiệu lực | dưới **1 phút**, **0** deploy | SC-001 |
| Đọc sổ cho mỗi thông điệp đăng ký | **1** truy vấn có chỉ mục | FR-080 |
| Số hàng trong sổ | vài hàng | Giả định |

**Ràng buộc**:
- **Một nguồn sự thật**: `subscription.campaigns`; khối `campaigns` bị bỏ khỏi YAML — FR-001/FR-002
- Nạp cấu hình **không chạm CSDL** — FR-003 (hiến phán §I, §VIII)
- `services/ai` không sửa; **không** thêm loại sự kiện đổi gói — FR-044 (FR-092 của 025)
- Giữ nguyên hình dạng response và **đúng 12 mã `reason`** — FR-050
- `campaign_claims`/`campaign_sweep_runs`: **không đổi lược đồ, không thêm FK** — FR-091
- Không lồng `uow.Execute`
- Không thêm biến môi trường; comment `.env*` tiếng Anh — FR-081/FR-082
- Tài liệu trong `specs/` tiếng Việt; đường dẫn tương đối

**Quy mô**: 4 mục tiêu vận hành; 1 bảng mới + 1 trigger + 1 seed; ~3 endpoint ghi + 1 endpoint đọc +
bổ sung trường đọc; khoảng **45–55 tệp** bị chạm, gọn trong `services/account` và `shared/config`.

---

## Kiểm Tra Hiến Pháp

*Cổng: phải qua trước Phase 0. Kiểm tra lại sau Phase 1.*

| Điều khoản | Yêu cầu | Cơ chế đáp ứng | Kết quả |
|---|---|---|---|
| §I Dịch vụ triển khai độc lập | Mỗi dịch vụ build và chạy độc lập | Chỉ `services/account` + `shared/config` thay đổi. `services/ai` và `services/commerce` không đổi dòng code nào. | ✅ |
| §II Ranh giới module | Cấm import `domain/entities`, `domain/repositories`, `infrastructure/**` của module khác | Sổ nằm trong `subscription/domain/entities`; tài khoản đọc qua hợp đồng; nhật ký chiến dịch đọc qua **hợp đồng hẹp riêng** của module `admin`, không đọc thẳng bảng. | ✅ |
| §II DTO, không thực thể | "using DTOs, never domain entities" xuyên module | Endpoint nhật ký trả `dto/admin.AdminAuditLogDTO` qua hợp đồng hẹp, không trả entity của module khác. | ✅ |
| §III Outbox | Mọi phát sự kiện dùng outbox | Nhật ký kiểm toán gọi `IAuditLogContract.Record` → ghi `shared.outbox_records` trong cùng transaction nghiệp vụ. Không đường phát trực tiếp. | ✅ |
| §IV Phân lớp hướng vào trong | presentation → application → domain ← infrastructure | Handler mới mỏng, gọi `ICampaignRegistryUseCase`; use case gọi repository; không tầng nào nhảy cấp. | ✅ |
| §IV Bỏ qua tầng | presentation không gọi thẳng repository | Handler chỉ gọi use case; use case chỉ gọi khoá. | ✅ |
| §V Lỗi tập trung | `apperror` factory, thông điệp tiếng Việt | Lỗi đăng ký chiến dịch thêm vào `application/errors/`, thông báo client tiếng Việt. | ✅ |
| §V Đơn vị công việc | Giao dịch nhiều ghi qua `IUnitOfWork.Execute`, không lồng | Use case đăng ký mở **đúng một** `uow.Execute`; cấm lồng (R3 của 025). | ✅ |
| §VI Lõi kiểm soát chi phí AI | Không sửa ngoài phạm vi | `services/ai` không bị sửa; không thêm chủ đề; chỉ tái dùng đường đổi gói sẵn có. | ✅ |
| §VII design-pattern.md | Đọc và tuân thủ | Repository/GORM, mapper, phân trang `PaginationQuery`/`PaginationResult`, lỗi `apperror`, handler `WrapHandler`, quy ước đặt tên. | ✅ |
| §VIII Khả chạy mọi hạ tầng | Không viết cứng giá trị phụ thuộc môi trường | Múi giờ từ `cfg.Database.TimeZone`; dữ liệu chiến dịch nằm trong CSDL theo môi trường; **không** thêm env. | ✅ |
| Goose migration | Mọi thay đổi lược đồ qua Goose | Migration tạo bảng + trigger + seed có `-- +goose Up/Down`, lý do tiếng Anh, tham chiếu `026 (FR-###)`. | ✅ |
| Ngôn ngữ | Client tiếng Việt; mã/nhật ký tiếng Anh | Thông báo lỗi tiếng Việt; định danh và comment tiếng Anh. | ✅ |
| Governance — Technical debt | Nợ kỹ thuật không sửa ghi vào `deferred.md` | Các điểm cắt được (ví dụ cache tương lai) ghi vào `deferred.md`, không để task chưa tick. | ✅ |
| Governance — Frontend guide | Tính năng đổi HTTP endpoint/DTO phải ship `frontend-guide.md` | FR-093; guide mới được cập nhật ở phase cuối. | ✅ |
| code-hygiene H1/H2/H4 | Tệp ≤ 500 dòng, hàm ≤ 80 dòng, 2 kiểu hành vi/tệp, không import chéo | Use case đăng ký tách theo trách nhiệm (validate, write, audit); mỗi tệp một kiểu chính. | ✅ |

**Kết luận cổng**: **ĐẠT**, không có ngoại lệ phải ghi vào Bảng Theo Dõi Độ Phức Tạp.

### Kiểm tra lại sau Phase 1

| Điều khoản | Cần xác nhận lại | Kết quả |
|---|---|---|
| §II | Endpoint nhật ký chiến dịch không đọc thẳng `shared.admin_audit_logs` | ✅ Hợp đồng hẹp `ICampaignAuditLookup` trả **DTO**, cài trong module `admin` |
| §III + §V | Ghi kiểm toán nằm trong cùng transaction nhưng qua outbox | ✅ `Record` → outbox row trong cùng `uow.Execute`; bảng audit được ghi bất đồng bộ bởi consumer sẵn có |
| §VIII | Không có giá trị chiến dịch nào còn trong YAML/env | ✅ `campaigns` bị xoá; chỉ `campaign_sweep` (nhịp quét) còn |
| Hiến pháp — an toàn dữ liệu | Migration không đụng `campaign_claims` | ✅ Không FK, không `ALTER` bảng cũ; seed idempotent |

---

## Cấu Trúc Dự Án

### Tài liệu (tính năng này)

```text
specs/026-campaign-registry-db/
├── spec.md                         # Đặc tả — nguồn sự thật về phạm vi
├── plan.md                         # Tệp này
├── research.md                     # Quyết định thiết kế kèm bằng chứng mã nguồn
├── data-model.md                   # Lược đồ, trigger, seed, thực thể, máy trạng thái
├── quickstart.md                   # Cẩu nang kiểm chứng chạy tay (V1–V7)
├── contracts/
│   ├── campaign-registry-api.md    # 3 endpoint ghi + 1 endpoint đọc nhật ký
│   └── campaign-query-api-delta.md # Khác biệt cộng thêm so với hợp đồng đọc của 025
├── checklists/
│   └── requirements.md             # Đã đạt 16/16 (3 mục có ngoại lệ có chủ ý)
├── frontend-guide.md               # Sinh ở phase cuối (FR-093)
└── tasks.md                        # Sinh ở /speckit-tasks
```

### Mã nguồn (thư mục gốc dự án)

```text
shared/
├── config/
│   ├── config.go                       # − CampaignConfig, − Config.Campaigns; giữ CampaignSweepConfig
│   ├── loader.go                       # − phân tích/ghi đè khối campaigns
│   ├── validation.go                   # − validateCampaigns/validateCampaign/campaignCodePattern
│   └── campaign_validation_test.go     # XOÁ — thay bằng kiểm thử ràng buộc CSDL/API
│   └── campaign_config_files_test.go   # ĐẢO CHIỀU — 025 T106 khẳng định cả hai YAML *phải có* `campaigns`
├── domain/constants/adminaudit/
│   ├── targettype.go                   # + Campaign = "campaign"
│   └── action.go                       # + CampaignCreate/CampaignUpdate/CampaignClose
configs/
├── config.yaml                         # − khối campaigns (kể cả sample-2026-11); giữ campaign_sweep
├── config.production.yaml              # − khối campaigns; giữ campaign_sweep
.env.example / .env.production.example  # − ghi chú khối campaigns (comment tiếng Anh)

services/account/
├── database/migrations/
│   ├── <ts>_add_campaigns.sql          # MỚI — bảng + CHECK + chỉ mục + trigger bất biến
│   └── <ts>_seed_launch_campaign.sql   # MỚI — seed launch-2026-10 idempotent
├── dto/subscription/
│   └── campaign_dto.go                 # + ClosedAt, BudgetLocked, Version, CampaignAuditLogRes;
│                                       #   + DTO request create/update/close; + hằng status closed
├── dto/admin/
│   └── audit.go                         # KHÔNG đổi — `AuditLogFilter` mở rộng ở tầng repository, không
│                                       #   sửa DTO chung (xem research.md R8)
├── contract/admin/
│   └── campaign_audit_contract.go       # MỚI — hợp đồng hẹp ICampaignAuditLookup (không đụng gRPC)
├── internal/modules/admin/
│   ├── domain/repositories/admin_audit_log_repo.go   # + ListByCampaignCode
│   └── infrastructure/persistence/admin_audit_log_repo.go
├── internal/modules/subscription/
│   ├── domain/entities/
│   │   └── campaign.go                  # MỚI — entity Campaign
│   ├── domain/repositories/
│   │   └── campaign_repo.go             # MỚI — ICampaignRepository
│   ├── infrastructure/persistence/
│   │   └── campaign_repo_impl.go        # MỚI — GORM, FOR SHARE, If-Match
│   ├── application/
│   │   ├── errors/errors.go             # + lỗi đăng ký (trùng mã, bất biến, phiên bản lệch, …)
│   │   ├── mapper/campaign_mapper.go    # + map entity → DTO (kể cả closedAt/budgetLocked)
│   │   ├── interface/usecase/
│   │   │   └── campaign_registry_uc.go  # MỚI — ICampaignRegistryUseCase, ICampaignAuditLookup dùng lại
│   │   └── usecase/campaign/
│   │       ├── campaign_uc.go           # đổi: nhận campaign repo thay cfg.Campaigns
│   │       ├── grant_uc.go              # đổi KIỂU: campaign sharedconfig.CampaignConfig → *entities.Campaign
│   │       ├── admission_uc.go          # đổi: []*entities.Campaign
│   │       ├── quota_uc.go              # đổi: claimInput.Campaign *entities.Campaign; + re-read FOR SHARE (FR-045)
│   │       ├── campaign_query_uc.go      # đổi: đọc repo; + closedAt/budgetLocked/version/status closed
│   │       ├── campaign_query_claims_uc.go  # đổi: newCampaignEvaluator(uc.cfg.Campaigns, …) → nguồn repo
│   │       ├── sweep_uc.go              # đổi: đọc repo
│   │       ├── registry_uc.go           # MỚI — create/update/close + audit + version
│   │       ├── registry_validate.go     # MỚI — kiểm hình dạng dùng chung
│   │       └── *_test.go                # ĐỔI KIỂU — 13 tệp kiểm thử 025 dựng
│   │                                      #   []sharedconfig.CampaignConfig (danh sách đầy đủ ở
│   │                                      #   tasks.md T021)
│   └── presentation/
│       ├── handler/
│       │   ├── campaign_admin_handler.go      # + GetCampaignAudit
│       │   └── campaign_registry_handler.go   # MỚI — create/update/close
│       ├── router/router.go             # + POST "", PATCH "/:code", POST "/:code/close", GET "/:code/audit"
│       └── worker/campaign_sweep_worker.go    # không đổi cấu trúc; use case tự đọc repo
└── internal/modules/subscription/provider.go / di/app.go / di/wire_gen.go  # + repo/use case/handler mới
```

**Quyết định cấu trúc**: giữ nguyên kiến trúc phân lớp của dự án; **không** tách module mới. Sổ chiến
dịch dùng chung bảng gói cước, chung đường thay đổi gói và chung hợp đồng tài khoản với module
`subscription`, nên một module mới chỉ tạo thêm hợp đồng bọc lại thứ đã có (lập luận của 025 vẫn
đúng). Handler đọc và handler ghi tách riêng để bảo đảm **chỉ đọc** của đường tra cứu vẫn là ràng
buộc kiểu, không phải kỷ luật.

**Không tạo bảng mới cho nhật ký**: dùng `shared.admin_audit_logs` sẵn có, thêm `targetType =
"campaign"`; mã chiến dịch nằm trong `payload.campaignCode` (vì `targetId` là UUID). Đọc qua hợp
đồng hẹp `ICampaignAuditLookup` để không sửa hợp đồng gRPC `IAuditLogContract`.

---

## Bảng Theo Dõi Độ Phức Tạp

**Không có mục nào vi phạm hiến pháp.** Ba điểm từng được cân nhắc là ngoại lệ và **không** phải
ngoại lệ:

| Điểm nghi | Vì sao không phải ngoại lệ |
|---|---|
| Thêm **trigger PL/pgSQL** (mẫu mới trong repo) | Là PostgreSQL thuần, không khoá nhà cung cấp (§VIII). Đối xứng với chỉ mục một phần `ux_campaign_claims_watermark` của 025: bất biến sống ở tầng dữ liệu. Không có `CHECK` nào diễn tả được "bất biến khi có lượt cấp", và tầng ứng dụng không ràng buộc được SQL gõ tay. |
| Thêm **cột `version`** vào bảng chiến dịch | Không phục vụ bất biến ngân sách mà phục vụ chống ghi đè ngầm (FR-055). Tiền lệ có sẵn: `user_subscriptions.version`. |
| Mở **hợp đồng hẹp** đọc nhật ký thay vì mở rộng `IAuditLogContract` | Hợp đồng đó được phục vụ qua gRPC và có adapter client/server; thêm phương thức bắt buộc sửa proto — thay đổi hợp đồng xuyên dịch vụ. 024 đã giải đúng bài toán này bằng `AdminGrantLookup`, 025 bằng `CampaignAccountLister` (R5 của 025). |

---

## Lộ Trình Thực Thi

Năm nhóm, thứ tự phụ thuộc. Chi tiết tác vụ ở `tasks.md` (sinh tại `/speckit-tasks`).

| Nhóm | Nội dung | Phụ thuộc |
|---|---|---|
| **A** | Bảng `campaigns` + trigger + seed + entity + repository | Không |
| **B** | Gỡ cấu hình khối `campaigns` (config/loader/validation/tests) | Không |
| **C** | Đọc sổ trong use case: đăng ký, tra cứu, quét bù; FR-045 re-read | A, B |
| **D** | Vòng đời ghi: create/update/close + If-Match + audit; endpoint đọc nhật ký | A, C |
| **E** | Hợp đồng/tài liệu: data-model, contracts, frontend-guide, quickstart | toàn bộ |

**Cut-over**: nhóm A (migration + seed) **phải chạy trước khi binary mới lên**. Khối YAML `campaigns`
được gỡ ở nhóm B; có thể tồn tại tạm trong một cửa sổ triển khai làm bản sao rollback nhưng binary
mới **không đọc**. Xem `data-model.md` mục "Chuyển đổi".

**Có thể cắt nếu cần**: không. Cả bốn mục tiêu đều bắt buộc; nhưng nhóm D (ghi + nhật ký) có thể giao
sau nhóm C miễn là bất biến đã được trigger giữ.

## Bước Kiểm Chứng

Cẩu nang chạy tay: [quickstart.md](./quickstart.md). Bảy kịch bản, kèm cổng an toàn môi trường và
cổng email:

| Kịch bản | Chứng minh điều gì |
|---|---|
| V1 — mở rồi đóng chiến dịch qua API, **không deploy** | SC-001, SC-002, SC-009; bắt buộc |
| V2 — chiến dịch đã đóng không cấp cho tài khoản mới | FR-031, SC-002; bắt buộc |
| V3 — bất biến `quota`/`reserve` (API + trigger, kể cả SQL trực tiếp) | FR-020/FR-021, SC-003 |
| V4 — trung thực của migration/seed (chạy hai lần, rollback) | SC-004, FR-090/FR-092 |
| V5 — hợp đồng đọc không đổi (chỉ thêm trường; đúng 12 mã reason) | SC-005 |
| V6 — quét bù nhận chiến dịch mới trong một chu kỳ | SC-007, FR-070/FR-071; `lastSweepAt` null trước lần quét đầu (FR-072) |
| V7 — đọc nhật ký kiểm toán (tạo → sửa → đóng rồi đọc, chờ consumer) | FR-060..FR-063, SC-006, SC-011; bắt buộc |
