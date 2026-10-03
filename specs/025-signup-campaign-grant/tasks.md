---

description: "Task list for feature 025 — Cơ chế chiến dịch tặng gói cho tài khoản đăng ký mới"
---

# Tasks: Cơ Chế Chiến Dịch Tặng Gói Cho Tài Khoản Đăng Ký Mới

**Input**: Tài liệu thiết kế từ `/speckit-plan` cho `specs/025-signup-campaign-grant/`
(`plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`)

**Tests**: CÓ. Đặc tả định nghĩa 19 tiêu chí thành công và `quickstart.md` định nghĩa 8 kịch bản
bắt buộc — mỗi tiêu chí cần ít nhất một kiểm thử chứng minh được.

## Format: `[ID] [P?] [Story] Mô tả kèm đường dẫn tệp`

- **[P]**: chạy song song được (khác tệp, không phụ thuộc)
- **[Story]**: câu chuyện người dùng mà nhiệm vụ thuộc về
- Mọi nhiệm vụ đều kèm đường dẫn tệp chính xác

## Thứ tự ưu tiên — KHÁC với thứ tự P1/P2/P3 của đặc tả

Hai câu chuyện P1 (US3 rồi US1) nằm trên đường găt cho ngày 02/10. US4 (P2) cũng phải xong
trước ngày đó. Ba câu chuyện còn lại có thể làm sau. Thứ tự dưới đây vì vậy theo **ràng buộc phụ
thuộc và hạn ra mắt**, không theo số thứ tự ưu tiên.

| # | Câu chuyện | Ưu tiên | Có trước 02/10? | Ghi chú |
|---|---|---|---|---|
| 1 | US3 — Hạn mức không bị vượt | P1 | **Bắt buộc** | Bộ luật hạn mức là tiền đề của US1 |
| 2 | US1 — Nhận gói quà tặng | P1 | **Bắt buộc** | MVP của chiến dịch |
| 3 | US4 — Không mất lượt dùng AI | P2 | **Bắt buộc** | Vì lý do nêu ở `spec.md` mục "Cần sửa trước" |
| 4 | US2 — Email thông báo | P2 | **Bắt buộc** | FR-044 |
| 5 | US7 — Giao diện tra cứu | P2 | Không bắt buộc | Khối lượng lớn nhất, **có thể cắt** |
| 6 | US5 — Mở chiến dịch mới bằng cấu hình | P3 | Không bắt buộc | Cơ chế đã có sẵn sau US1 |
| 7 | US6 — Loại trừ tài khoản quản trị viên | P3 | Không bắt buộc | Phần phát đã ở Phase 2 |

---

## Phase 1: Thiết Lập (Hạ Tầng Dùng Chung)

**Mục đích**: Hằng số chủ đề và hàng đỗi, khung cấu hình chiến dịch, và kiểm thử canh ranh giới.

Không có nhiệm vụ nào ở phase này chứa logic nghiệp vụ. Mọi thứ ở đây là hạ tầng mà các câu chuyện
sau dùng chung.

- [x] T001 Thêm hằng số chủ đề `TopicUserCreated = "identity.user.created"` trong `shared/application/constants/eventconstants/eventconstants.go`
- [x] T002 [P] Thêm hằng số `QueueCampaignGrantUserCreated = "campaign_grant_user_created_queue"` trong `shared/application/constants/eventconstants/queues.go`
- [x] T003 Thêm `RoutingKeyUserCreated` tham chiếu **symbol** `eventconstants.TopicUserCreated` trong `shared/infrastructure/messaging/rabbitmq_constants.go` (không viết chuỗi thô — FR-093/R7)
- [x] T004 Thêm `Binding` cho hàng đỗi mới và `Binding` tương ứng cho `.dlq` trong `Topology` và `DLQTopology` ở `shared/infrastructure/messaging/topology.go`; quorum + `x-delivery-limit: 5`, `Prefetch 20`, `Concurrency 1`
- [x] T005 [P] Thêm kiểu `CampaignConfig` và trường `Campaigns []CampaignConfig \`mapstructure:"campaigns"\`` trong `shared/config/config.go`; mọi trường có nhận xét tiếng Anh nêu rõ điều kiện kiểm tra
- [x] T005b [P] Thêm kiểu `CampaignSweepConfig` và trường `CampaignSweep CampaignSweepConfig \`mapstructure:"campaign_sweep"\`` trong cùng tệp, theo `data-model.md` mục 4.1.1: `IntervalSeconds int` (bắt buộc) và `RunOnStart bool`. Nhận xét tiếng Anh giải thích vì sao khối này **nằm ngoài** danh sách `campaigns` — một worker quét cho toàn bộ chiến dịch, đặt chu kỳ vào từng phần tử sẽ tạo N worker (FR-083)
- [x] T006 [P] Thêm `validateCampaigns` và gọi nó từ `validateConfig` trong `shared/config/validation.go`; kiểm tra đủ 7 trường hợp ở `data-model.md` mục 4.1. Thêm `validateCampaignSweep` cho `IntervalSeconds >= 60` và mặc định `RunOnStart = true` nếu bỏ trống (mục 4.1.1). Thiếu phần này thì worker quét bù chạy với chu kỳ 0 ở T098
- [x] T007 Thêm phân tích `startsAt`/`endsAt` theo RFC3339 có múi giờ vào bước sau khi nạp cấu hình trong `shared/config/loader.go`; dùng `timeutils.GetNow(cfg.Database.TimeZone)` ở mọi nơi cần thời gian hiện tại — tuyệt đối không `time.Now()` (R6, hiến phán §VIII)
- [x] T008 [P] Ghi chốt trong `applyEnvOverrides` ở `shared/config/loader.go` rằng chiến dịch **không** có biến môi trường nào, kèm nhận xét tiếng Anh nêu hai lý do: `campaigns` là danh sách (một biến không mang được danh sách), và `campaign_sweep` nằm cùng tệp YAML với chiến dịch nên một biến riêng sẽ tạo **hai đường cấu hình cho cùng một ý nghĩa** (R10, `data-model.md` mục 4.1.1). **Không** thêm biến `CAMPAIGN_*` nào vào `applyEnvOverrides`
- [x] T009 [P] Thêm khối `campaigns:` **và** khối `campaign_sweep:` vào `configs/config.yaml`, theo mẫu ở `data-model.md` mục 4 và 4.1.1
- [x] T010 [P] Thêm hai khối đó giống hệt vào `configs/config.production.yaml` — thiếu ở đây thì khối kia thành cấu hình chết. `interval_seconds` ở production phải khớp môi trường dev
- [x] T011 [P] Ghi chú biến đơn lẻ trong `.env.example` bằng **tiếng Anh**, nói rõ danh sách chiến dịch và khoảng thời gian quét bù nằm trong tệp YAML chứ không có biến môi trường
- [x] T012 [P] Ghi chú biến đơn lẻ trong `.env.production.example` bằng **tiếng Anh**, nội dung giống T011
- [x] T013 [P] Viết kiểm thử canh FR-092 trong `shared/infrastructure/messaging/topology_test.go`: `Topology` phải có **đúng một** liên kết chứa `TopicUserCreated`, và liên kết đó phải là hàng đỗi của chiến dịch
- [x] T014 [P] Viết kiểm thử `validateCampaigns` trong `shared/config/campaign_validation_test.go` cho 7 trường hợp sai ở `data-model.md` mục 4.1, **cộng** 3 trường hợp của `validateCampaignSweep`: `interval_seconds` bằng 0, bằng 59, và bỏ trống phải ra 60 (SC-013, SC-014). Ca "bỏ trống" **phải** kiểm ở tầng loader chứ không ở `validateCampaignSweep`: trong Go, khoá vắng mặt và `0` ghi tay giao nhau ở kiểu `int`, nên chỉ `v.SetDefault` mới phân biệt được — và `SetDefault` chỉ chạy khi khoá **vắng mặt**

> **Ghi chú thứ tự**: T013 và T014 là **ngoại lệ** của nguyên tắc "kiểm thử viết trước, phải ĐỎ trước
> khi cài" ở mục "Trong mỗi câu chuyện" phía dưới. Chúng là kiểm thử **canh ranh giới trên mã đã có**
> (topology hàng đỗi, bộ kiểm tra cấu hình), không phải kiểm thử hành vi mới — nên viết sau vẫn hợp
> lý và đọc dễ hơn. Mọi kiểm thử hành vi ở Phase 3 tới 10 vẫn phải viết trước.

**Điểm kiểm tra**: `make wire && make lint && make test-race` xanh. Hạ tầng chủ đề và hàng đỗi đã sẵn
sàng, chưa có gì tiêu thụ.

---

## Phase 2: Nền Tảng (Chặn Mọi Câu Chuyện)

**Mục đích**: Lược đồ dữ liệu, thông điệp tạo tài khoản, và hợp đồng liệt kê tài khoản.

**⚠️ CRITICAL**: Không câu chuyện nào bắt đầu được trước khi phase này xong. Riêng việc phát
thông điện tạo tài khoản nằm ở đây vì US1, US3 và US7 đều phụ thuộc vào nó.

### Lược đồ dữ liệu

- [x] T015 Tạo tệp migration bằng `make migrate-create svc=account name=add_created_by_admin`, rồi viết SQL theo `data-model.md` mục 2.2 trong `services/account/database/migrations/<timestamp>_add_created_by_admin.sql`; nhận xét `-- +goose` bằng **tiếng Anh**, dẫn chiếu `025 T### (FR-050)`, có cả phần `Down`
- [x] T016 Tạo tệp migration bằng `make migrate-create svc=account name=add_campaign_claims`, rồi viết SQL theo `data-model.md` mục 3.2 trong `services/account/database/migrations/<timestamp>_add_campaign_claims.sql`; phải đủ **4** chỉ mục và ràng buộc kiểm tra: `ux_campaign_claims_user`, `ux_campaign_claims_watermark` (một phần, `WHERE fills_main_quota`), `ix_campaign_claims_campaign_bucket`, `ix_campaign_claims_campaign_granted`, cộng `CHECK (grant_bucket IN ('main','reserve'))`. Bắt buộc có cột `grant_bucket` (R2b)
- [x] T017 [P] Thêm trường `CreatedByAdmin bool` vào `entities.User` trong `services/account/internal/modules/identity/domain/entities/user.go`
- [x] T018 [P] Thêm trường `CampaignClaim` — gồm `GrantBucket` và `FillsMainQuota` — cùng hàm `TableName()` trả về `campaign_claims` trong `services/account/internal/modules/subscription/domain/entities/campaign_claim.go`; đặt tệp `table_name.go` của module ở tệp riêng theo quy ước H2

### Hợp đồng liệt kê tài khoản

> **⚠️ KHÔNG sửa `IUserBriefContract`.** Hợp đồng đó đi qua gRPC và có adapter client + server riêng.
> Thêm phương thức vào đó bắt buộc phải sửa `identity.proto`, tức đổi hợp đồng **xuyên dịch vụ** —
> trái FR-058. Thay vào đó khai báo interface hẹp riêng `CampaignAccountLister`, theo đúng tiền lệ
> `AdminGrantLookup` của 024. Chi tiết ở `contracts/identity-contract.md` mục 2.

- [x] T019 [P] Thêm `CreatedAt time.Time` và `CreatedByAdmin bool` vào `UserBriefContractDTO` trong `services/account/dto/identity/contract_dto.go`, kèm nhận xét tiếng Anh giải thích vì sao **không** suy ra từ `email_verified`
- [x] T020 [P] Thêm `CampaignAccountFilter` và `CampaignAccountPage` vào `services/account/internal/modules/identity/domain/repositories/readmodels.go`. `CampaignAccountPage.Items` kiểu `[]*dto.UserBriefContractDTO` — **không** phải `[]*entities.User` (hiến phán §II: "using DTOs, never domain entities")
- [x] T021 Thêm `ListCreatedInWindow` vào `IUserRepository` trong `services/account/internal/modules/identity/domain/repositories/user_repo.go`
- [x] T022 Cài `ListCreatedInWindow` trong `services/account/internal/modules/identity/infrastructure/persistence/user_repo_impl.go` với phân trang khoá tổ hợp `(created_at, id)`, lọc `is_deleted = false`, và `Count` trước `Find`
- [x] T023 Khai báo interface hẹp `CampaignAccountLister` cùng hàm `NewCampaignAccountLister(c *UserBriefContract) CampaignAccountLister` trong `services/account/contract/identity/user_brief_contract_impl.go`, rồi cài phương thức `ListCreatedInWindow` trên `*UserBriefContract`. **Không** thêm gì vào `IUserBriefContract`. Nhận xét tiếng Anh nêu lý do tách, tham chiếu `AdminGrantLookup` (hiến phán §II)
- [x] T024 Thêm use case đọc cửa sổ tạo tài khoản trong module `identity`, cài ở tệp hợp đồng để `*UserBriefContract` thoả `CampaignAccountLister`. Trả về `Items` là `[]` rỗng khi không có kết quả, **không** phải `nil`
  - Kèm kiểm thử cho **7 kịch bản** ở `contracts/identity-contract.md` mục 6. Ba kịch bản bắt buộc: (a) **phân trang không bỏ sót khi nhiều tài khoản trùng `created_at`** — bẫy của khoá tổ hợp `(created_at, id)`, tạo 3 tài khoản cùng thời điểm rồi duyệt 3 lô `limit=1` phải ra đủ 3 bản ghi không trùng không thiếu; (b) `created_by_admin` trong cơ sở dữ liệu khớp cờ trong thông điệp cho cả ba đường tạo tài khoản; (c) **biên dịch toàn bộ `services/commerce` phải xanh** — bằng chứng cho quyết định không mở rộng hợp đồng gRPC

### Thông điệp tạo tài khoản

- [x] T025 [P] Thêm `UserCreatedEventPayload` với đúng 4 trường (`UserID`, `Provider`, `RegisteredAt`, `CreatedByAdmin`) trong `services/account/dto/identity/event_payload.go`; **không** có trường nhận diện cá nhân (FR-087)
- [x] T026 Thêm bộ phát sự kiện `IEventPublisher` vào trường và tham số ctor của `RegisterUseCase` trong `services/account/internal/modules/identity/application/usecase/register_uc.go`
- [x] T027 Thêm bộ phát sự kiện vào ctor `GoogleAuthUseCase` trong `services/account/internal/modules/identity/application/usecase/google_auth_uc.go`
- [x] T028 Thêm bộ phát sự kiện vào ctor `UserUseCase` trong `services/account/internal/modules/identity/application/usecase/user_uc.go`
- [x] T029 Cập nhật 3 tệp kiểm thử gọi các ctor trên: `register_uc_test.go`, `google_auth_uc_test.go`, `eager_provision_test.go` trong cùng thư mục
- [x] T029b [P] Kiểm thử ba điều kiện mà trình biên dịch không bắt được, thêm vào tệp kiểm thử đã có ở T029:
  - **FR-064**: thông điệp `identity.user.created` mang `correlationId` khớp với dòng hàng đỗi và với log worker nhận được, ở cả ba đường tạo tài khoản
  - **FR-085**: `modules/identity/` **không** có câu lệnh nào tham chiếu cấu hình `campaigns` — quét văn bản
  - **FR-042**: cố tình làm bộ phát hàng đỗi lỗi rồi gọi đăng ký — phải trả về lỗi và **không** còn dòng tài khoản nào trong cơ sở dữ liệu. Nếu tài khoản vẫn được tạo thì thông điệp đã thất lạc vĩnh viễn mà không ai biết
- [x] T030 Phát `identity.user.created` **bên trong** transaction sẵn có ở `register_uc.go` (sau `EnsureDefaultSubscription`), đặt cờ `CreatedByAdmin: false`
- [x] T031 Phát thông điệp trong `createGoogleUser` ở `google_auth_uc.go`, đặt cờ `CreatedByAdmin: false`; phải nằm trong `createFn` đang chạy trong `uow.Execute`
- [x] T032 Đặt `CreatedByAdmin: true` trên thực thể người dùng và phát thông điệp trong `admin_uc.go`; đây là **vị trí ghi `true` duy nhất** trong toàn mã nguồn (FR-050, R4)

### Kho lượt cấp

- [x] T033 [P] Thêm `ICampaignClaimRepository` với 5 phương thức `LockClaimWindow`, `CountByBucket`, `GetWatermark`, `Create`, `ListByCampaign` trong `services/account/internal/modules/subscription/domain/repositories/campaign_claim_repo.go`. `CountByBucket(campaignCode string, bucket GrantBucket) (int64, error)` trả về **một** số cho **một** phần — dùng nó để lấy cả `grantedMainCount` lẫn `grantedCompensationCount`; KHÔNG cung cấp phương thức nào đếm "phần chính" bằng cách loại trừ điều kiện trên `fills_main_quota` (R2b)
- [x] T034 Cài `CampaignClaimRepository` trong `services/account/internal/modules/subscription/infrastructure/persistence/campaign_claim_repo_impl.go`; `Create` dùng `ON CONFLICT (user_id) DO NOTHING` và trả về `RowsAffected > 0`; `LockClaimWindow` dùng `pg_advisory_xact_lock` qua `r.GetDB(ctx)` (R3); `CountByBucket` dùng chỉ mục `ix_campaign_claims_campaign_bucket`; `GetWatermark` trả về cả `found bool` lẫn `registeredAt` vì `registered_at` không rỗng nhưng **vắng mặt** thì có thật (R3)
- [x] T035 [P] Thêm lỗi chiến dịch bằng tiếng Việt trong `services/account/internal/modules/subscription/application/errors/errors.go`

**Điểm kiểm tra**: `make migrate-up svc=account` chạy được, `make wire && make lint && make test-race`
xanh. Một tài khoản tạo mới **đã** phát ra thông điệp — nhưng chưa có ai tiêu thụ, tức là thông
điệp sẽ nằm trong hàng đỗi. Điều đó đúng và tạm thời chấp nhận được.

---

## Phase 3: US3 — Bộ Luật Hạn Mức (P1)

**Mục đích**: Chọn đúng chiến dịch, chốt hạn mức tuyệt đối, và lưu mốc ưu tiên bền vững.

Phase này **chưa** cấp gói. Nó dựng bộ quyết định "có cấp hay không". US1 sẽ dùng bộ quyết định
này để cấp thật. Nhờ tách vậy, ba yêu cầu khó nhất của đặc tả (FR-022, FR-025, FR-030, FR-031)
được kiểm chứng độc lập, không cần dựng cả đường cấp gói.

**Kiểm thử độc lập**: Cho N lượt cấp đã ghi, đưa vào một tài khoản mới, khẳng định bộ quyết
định đúng — chưa cần gói nào được cấp.

### Kiểm thử (viết trước, phải ĐỎ trước khi cài)

- [x] T036 [P] [US3] Kiểm thử trần cứng: 20 lần gọi với 1 suất còn lại trên 5 luồng song song chỉ cấp đúng 1 lần — `services/account/internal/modules/subscription/application/usecase/campaign/campaign_quota_test.go`. **Bắt buộc kèm ca hồi quy dự phòng** (R2b): cấp đủ `quota` lượt chính, rồi cấp tiếp `reserve` lượt nữa — phải nhận đủ `quota + reserve` lượt với `grant_bucket` lần lượt là `main` rồi `reserve`, **không** lỗi vi phạm `ux_campaign_claims_watermark`. Đây là ca bắt đúng lỗi "phần dự phòng không bao giờ dùng được": nếu cài sai cột đếm, bước này đỏ ngay (SC-001, SC-001b, SC-001c)
- [x] T037 [P] [US3] Kiểm thử giao lại cùng một thông điệp 100 lần chỉ tạo 1 lượt cấp — cùng tệp với T036 (SC-002)
- [x] T038 [P] [US3] Kiểm thử mốc ưu tiên: lượt thứ N đánh dấu `fills_main_quota`, đúng 1 bản ghi mang cờ đó, và tài khoản tạo sau mốc không được cấp — cùng tệp với T036. Khẳng định thêm: đếm `grant_bucket = 'main'` trả về `quota` chứ không phải `tổng - 1`, và `COUNT(*) WHERE NOT fills_main_quota` **không** được dùng ở bất kỳ đâu trong mã nguồn này (R2b, SC-001, SC-001d)
- [x] T039 [P] [US3] Kiểm thử lựa chọn chiến dịch: hai chiến dịch cùng hiệu lực thì chọn chiến dịch bắt đầu muộn nhất, bất kể thứ tự thông điệp tới — `campaign_admission_test.go` (SC-001)
- [x] T040 [P] [US3] Kiểm thử tài khoản do quản trị viên tạo bị bỏ qua với lý do riêng — cùng tệp với T039 (FR-050)

### Cài đặt

- [x] T041 [US3] Khai báo giao diện `ICampaignUseCase` trong `services/account/internal/modules/subscription/application/interface/usecase/campaign_uc.go`
- [x] T042 [US3] Cài `admission_uc.go` trong `services/account/internal/modules/subscription/application/usecase/campaign/`: 9 điều kiện theo đúng thứ tự ở `data-model.md` mục 6, và quy tắc chọn chiến dịch ở mục 6.1
- [x] T043 [US3] Cài `quota_uc.go` cùng thư mục: giữ khoá tư vấn, đếm **riêng** theo `grant_bucket`, chốt mốc ưu tiên, rồi quyết định lấy từ phần chính hay phần dự phòng, theo nguyên văn mã giả ở `data-model.md` mục 6.2. Nhánh dự phòng bắt buộc có hai điều kiện: `reserveCount < reserve` **và** `watermarkAt` có giá trị; thiếu mốc thì bỏ qua với lý do `quota_anomaly` chứ không đoán (R2b)
- [x] T044 [US3] Cài `campaign_uc.go` cùng thư mục: điều phối một lần xử lý thông điệp, ghi lại lý do bỏ qua theo FR-061, trả về `PermanentError` cho thông điệp hỏng và `RetryableError` cho lỗi tạm thời
- [x] T045 [US3] Bảo đảm `quota_uc.go` **không** gọi `uow.Execute` lồng nhau: chỉ một lần `uow.Execute` ở `campaign_uc.go`, các bước bên trong dùng chung `txCtx` (R3 — `uow.go:69` dùng handle gốc)

**Điểm kiểm tra**: `make test-race` xanh. Bộ quyết định cấp chạy đúng, gói chưa được cấp.

---

## Phase 4: US1 — Người Dùng Mới Nhận Gói Quà Tặng (P1) 🎯 MVP

**Mục đích**: Biến quyết định "được cấp" thành gói thật, và cấp qua cả hai đường.

**Kiểm thử độc lập**: Đăng ký một tài khoản mới trong khi chiến dịch đang chạy, đợi độ trễ đã
công bố, kiểm tra tài khoản đó ở gói Premium 1 tháng, cờ tự động gia hạn tắt, đồng thời quyền
AI cũng chuyển sang chính sách Premium.

### Kiểm thử

- [x] T046 [P] [US1] Kiểm thử cấp gói: sau khi bộ quyết định trả về "được cấp", tài khoản ở `premium-monthly`, `expires_at = granted_at + duration_days` (**KHÔNG** phải `registered_at + 30 ngày` — FR-034), `is_auto_renew_enabled = false` — `campaign_grant_test.go` (SC-006)
- [x] T047 [P] [US1] Kiểm thử đọc lại bản ghi gói trước khi phát sự kiện: `effectiveTo` phải khớp thời hạn, **không** rỗng (R13 — rỗng là quyền AI vô thời hạn, SC-008)
- [x] T048 [P] [US1] Kiểm thử quyền AI cập nhật theo: lịch sử đổi gói có đúng 1 dòng cho lần cấp, và dòng đó mang `toPlanId` cùng `aiCostPolicyId` của gói Premium (FR-037, SC-006, SC-008)
- [x] T049 [P] [US1] Kiểm thử giao dịch nguyên tử: cố tình làm `PublishPlanChange` thất bại, khẳng định **không** còn lượt cấp nào (FR-039, SC-003)
- [x] T050 [P] [US1] Kiểm thử chống cộng thêm thời hạn: tài khoản đang ở `premium-monthly` còn hạn thì bộ quyết định trả về "không cấp" và thời hạn giữ nguyên (FR-018, SC-009)

### Cài đặt

- [x] T051 [US1] Cài `grant_uc.go` trong `services/account/internal/modules/subscription/application/usecase/campaign/`: gọi `ApplySubscriptionPlanWithEvent`, **đọc lại** bản ghi gói, rồi `PublishPlanChange` (FR-036, FR-037, R13)
- [x] T052 [US1] Trong `grant_uc.go`, ghi nhật ký kiểm toán với nguồn gốc là mã chiến dịch và **không** có quản trị viên thực hiện (FR-038)
- [x] T053 [US1] Trong `grant_uc.go`, gửi email qua `INotificationContract.SendByUserID` **bên trong cùng transaction**; lỗi gửi thì ghi nhật và tiếp tục, không làm mất gói đã cấp (FR-049)
- [x] T054 [US1] Cài worker mỏng `campaign_grant_worker.go` trong `services/account/internal/modules/subscription/presentation/worker/`: nhúng `*worker.BaseWorker`, `RegisterTypedConsumer` với `ConsumerOpts{}`, `workerlog`, chỉ ủy quyền cho use case
- [x] T055 [US1] Đăng ký `CampaignGrantWorker` trong `providers` của `services/account/internal/modules/subscription/provider.go` và `internal/di/wire.go`
- [x] T056 [US1] Thêm trường `CampaignGrant` vào `Workers` và vào `All()` trong `services/account/internal/bootstrap/worker.go`
- [x] T057 [US1] Chạy `make wire` để sinh lại `wire_gen.go`.
  **⚠️ Tệp dùng chung**: không chạy song song với T067, T083, T099 — dễ mất thay đổi của nhau khi sinh lại cùng một tệp
- [x] T058 [US1] Vô hiệu hoạt cache tổng quan gói **sau commit** bằng đúng hàm `invalidateOverviewCache` đang có (FR-040), không dùng `_ =` trên ctx của người gọi

**Điểm kiểm tra**: MVP hoàn tất. Một tài khoản đăng ký mới nhận gói trong vòng 5 phút.

**⚠️ MVP giai đoạn 1 CHƯA an toàn để ra mắt**: chưa có cơ chế quét bù (Phase 10), nên thông
điệp bị rơi sẽ mất tài khoản đó vĩnh viễn. Chạy được K2.1–K2.2 để kiểm chứng, nhưng đừng
bật chiến dịch thật cho khách hàng cho tới khi xong phần quét bù ở Phase 10.

---

## Phase 5: US4 — Không Mất Lượt Dùng AI Vì Thao Tác Chồng Nhau (P2)

**Mục đích**: Sửa lỗi đếm thiếu có sẵn, độc lập hoàn toàn với chiến dịch.

Câu chuyện này không phụ thuộc gì và có thể làm **song song** với Phase 3.

**⚠️ Không chạy song song với Phase 4 (US1) và Phase 7 (US7)**: cả ba đều chạy `make wire` sinh lại
`wire_gen.go`, nên sẽ đụng độ tệp. Nếu buộc phải chạy song song thì **một người giữ `wire_gen.go`**,
người còn lại làm mọi việc khác rồi gom lại một lần.

**Kiểm thử độc lập**: Một tài khoản ở 29/30 lượt, chạy 3 thao tác song song, đối chiếu bộ đếm.

### Kiểm thử

- [x] T059 [P] [US4] Kiểm thử chống đếm thiếu: 3 lần tăng bộ đếm đồng thời từ 29/30 phải ra **30**, không phải 29 — `services/account/internal/modules/subscription/application/usecase/quota/quota_uc_test.go` (SC-010)
- [x] T060 [P] [US4] Kiểm thử chặn vượt: lần thứ 31 bị từ chối bằng lỗi nghiệp vụ — cùng tệp với T059 (SC-010)
- [x] T061 [P] [US4] Kiểm thử đường đọc không bị khoá: gọi `GetAndResetDailyQuota` khi đang có thao tác ghi chạy, phải trả về mà không treo (FR-055) — cùng tệp với T059 (SC-010)
- [x] T062 [P] [US4] Kiểm thử về 0 khi sang ngày mới, không cần thao tác thủ công (FR-054) — cùng tệp với T059
- [x] T062b [P] [US4] Kiểm thử canh FR-058: hợp đồng gRPC trong contracts/subscription/v1/ **không đổi một dòng** sau khi sửa bộ đếm — sửa lỗi race là sửa bên trong use case, hợp đồng ra bên ngoài phải giữ nguyên. Thêm vào tệp kiểm thử canh ranh giới đã tạo ở T090

### Cài đặt

- [x] T063 [US4] Thêm trường `uow shared_repos.IUnitOfWork` vào `UserQuotaUseCase` trong `services/account/internal/modules/subscription/application/usecase/quota/quota_uc.go`
- [x] T064 [US4] Bọc **đúng đoạn** đọc–tăng–ghi của `UpdateOutfitQuota` trong `uow.Execute`, dùng `GetByUserIDWithLock` thay `GetByUserID`; hai lệnh `GetOrCreateUserSubscription` và `LoadPlanForSubscription` phải ở **ngoài** transaction (R9)
- [x] T065 [US4] Sửa `UpdateAiChatQuota` giống hệt T064
- [x] T066 [US4] Sửa `checkAndResetDailyQuota`: giữ đường đọc không khoá khi chưa sang ngày; chỉ mở transaction khi thực sự cần về 0 (R9, FR-055)
- [x] T067 [US4] Chạy `make wire` để sinh lại `wire_gen.go`.
  **⚠️ Tệp dùng chung**: không chạy song song với T057, T083, T099 — dễ mất thay đổi của nhau khi sinh lại cùng một tệp

**Điểm kiểm tra**: `make test-race` xanh. Bộ đếm không còn đếm thiếu.

---

## Phase 6: US2 — Người Dùng Được Báo Rõ Quyền Lợi (P2)

**Mục đích**: Gửi email nói rõ gói, thời hạn, và việc không tự động gia hạn.

**Kiểm thử độc lập**: Cấp gói cho một tài khoản, kiểm tra thông điệp hàng đỗi `notification.email`
mà có đúng một bản ghi với nội dung đúng.

### Kiểm thử

- [x] T068 [P] [US2] Kiểm thử nội dung thông điệp: có tên gói và ngày hết hạn, nói rõ không tự động gia hạn, **không** có số suất của chiến dịch — `services/account/internal/modules/notification/application/usecase/email/outbox_email_uc_test.go` (SC-005)
- [x] T069 [P] [US2] Kiểm thử không gửi cho tài khoản không được cấp gói (FR-048) — cùng tệp với T068 (SC-005)

### Cài đặt

- [x] T070 [US2] Thêm hằng số `CampaignGrant Template = "campaign_grant"` trong `services/account/internal/modules/notification/domain/constants/template/template.go`, kèm nhận xét về lý do **không** dùng lại `admin_grant` (R12)
- [x] T071 [US2] Tạo `services/account/internal/modules/notification/application/templates/CampaignGrant.html` bằng tiếng Việt: nêu tên gói, ngày hết hạn, nói rõ không tự động gia hạn, **không** gợi ý đã thanh toán, **không** nhắc số suất
- [x] T072 [US2] Nhúng tệp trong `templates.go` ở `services/account/internal/modules/notification/application/templates/templates.go`
- [x] T073 [US2] Thêm mục đệm vào `tmplCache` và nhánh tiêu đề vào `resolveSubject` trong `services/account/internal/modules/notification/application/usecase/email/outbox_email_uc.go`; tiêu đề bắt đầu bằng `[Closy] `

**Điểm kiểm tra**: `make test-race` xanh. Hộp thư nhận được đúng một email mỗi lần cấp gói.

---

## Phase 7: US7 — Người Vận Hành Trả Lời Được Khách Hàng (P2)

**Mục đích**: Giao diện tra cứu chỉ đọc, phân biệt được ba trạng thái.

**Có thể cắt toàn bộ phase này** mà không ảnh hưởng ngày 02/10. Đánh đổi là mất SC-011 — bộ
phận hỗ trợ phải tra cứu bằng tay.

**Kiểm thử độc lập**: Mở chức năng tra cứu với một chiến dịch đã có số liệu thật.

### Kiểm thử

- [X] T074 [P] [US7] Kiểm thử ba trạng thái trả về **khác nhau** cho ba tài khoản khác nhau; trả giống nhau là không đạt FR-068 — `services/account/internal/modules/subscription/presentation/handler/campaign_admin_handler_test.go` (SC-011, FR-068)
- [X] T075 [P] [US7] Kiểm thử chỉ đọc: gọi 50 lần, số lượt cấp không đổi, không sinh bản ghi hàng đỗi email (FR-069) — cùng tệp với T074 (SC-011)
- [X] T076 [P] [US7] Kiểm thử mã chiến dịch không tồn tại trả 404 kèm thông điệp tiếng Việt, không phải danh sách rỗng (FR-072) — cùng tệp với T074 (SC-011)
- [X] T076b [P] [US7] Phủ ba tình huống còn thiếu của `contracts/campaign-query-api.md` mục 4: người dùng thường gọi vào trả **403** (FR-070); vượt giới hạn tần suất trả **429** (FR-071); `:userId` không tồn tại trả **404** kèm thông điệp tiếng Việt — cùng tệp với T074
- [X] T076c [P] [US7] Kiểm thử bộ lọc `from`/`to` theo thời điểm cấp: lượt cấp ngoài khoảng không xuất hiện, lượt trong khoảng thì có, và `from > to` bị chặn (FR-067) — cùng tệp với T074

### Cài đặt

- [X] T077 [P] [US7] Thêm DTO tra cứu trong `services/account/dto/subscription/campaign_dto.go` theo `contracts/campaign-query-api.md` mục 3, nhãn `camelCase`
- [X] T078 [US7] Thêm hàm ánh xạ trong `services/account/internal/modules/subscription/application/mapper/campaign_mapper.go`; dùng `sliceutils.IndexBy` khi ghép dữ liệu tài khoản theo mảng mã (hiến phán §II)
- [X] T079 [US7] **Khai báo giao diện đọc RIÊNG** `ICampaignQueryUseCase` (tra cứu trạng thái chiến dịch, danh sách lượt cấp có bộ lọc `from`/`to`, trạng thái một tài khoản). **KHÔNG** thêm phương thức đọc vào `ICampaignUseCase` — giao diện đó đã đóng ở T041 với đúng một phương thức `ProcessUserCreated`, và nhận xét "one method because there is exactly one trigger" là có chủ ý. Cài truy vấn trong `campaign_query_uc.go`; **mọi số đếm đều dẫn xuất lúc tra cứu**, không lưu (FR-065, FR-066, FR-067)
- [X] T079a [US7] Migration tạo bảng `subscription.campaign_sweep_runs` theo `data-model.md` mục 5.1, khoá chính theo `campaign_code`. Tạo bằng `make migrate-create svc=account name=…` theo T015/T016: có cả `-- +goose Up` và `-- +goose Down`, nhận xét tiếng Anh dạng `025 T### (FR-###)` (hiến phán: Goose Migrations là MUST). Chạy `make migrate-up svc=account` sau đó
- [X] T079b [US7] Cài máy trạng thái chiến dịch (`not_started`, `running`, `compensation`, `exhausted`, `expired`) trong `campaign_query_uc.go`; **KHÔNG** đặt trong `admission_uc.go`. Lý do: `admission_uc.go` là bộ đánh giá thuần theo chính `data-model.md` mục 6 ("pure function of its inputs"), không có repository; mà máy trạng thái cần đếm `grantedMainCount` trên `campaign_claims` để phân biệt `running` với `compensation`. Tính lại lúc tra cứu, không lưu
- [X] T079c [US7] Worker quét bù **ghi trạng thái vào CSDL**, không giữ trong bộ nhớ (FR-059): mỗi lần quét **thành công** cập nhật `last_sweep_at` cho **mọi** chiến dịch; lần quét lỗi thì **không** đẩy mốc `last_sweep_at` (quét hỏng 15 phút/lần trong 6 giờ phải báo `last_sweep_at` cũ 6 giờ — đó mới là cảnh báo hữu ích) nhưng vẫn đặt `degraded = TRUE` + `degraded_reason`; lần quét thành công sau đó thì xoá `degraded_reason`. Truy vấn T079 đọc thẳng bảng này. Bỏ `sweep_uc.go:418 Health()` khỏi đường tra cứu **và khỏi interface**, để không còn nguồn sức khoẻ thứ hai cạnh bảng. **Lý do**: hệ thống chạy nhiều phiên bản mỗi bộ một worker, nên cờ trong bộ nhớ sẽ báo **lành trong khi thật ra hỏng** — hậu quả nguy hiểm nhất của FR-059; với `APP_MODE=http` cờ luôn `false`
- [X] T079d [P] [US7] Kiểm thử **5 trạng thái** của T079b, đặc biệt phân biệt được `compensation` với `running` (đã chốt mốc ưu tiên) và `exhausted` với `expired` (hết suất so với quá `endsAt`) (SC-012)
- [X] T079e [P] [US7] Kiểm thử `degraded` của T079c: một lần quét lỗi **thực sự** đặt `degraded = TRUE` kèm `degraded_reason`, và lần quét thành công sau đó **xoá** `degraded_reason`. Đây là chốt chặn duy nhất chứng minh lý do lưu vào CSDL: cờ trong bộ nhớ sẽ không bắt được lỗi do phiên bản khác gây ra (SC-012, FR-059)
- [X] T080 [US7] Cài truy vấn trạng thái một tài khoản trả về **6** giá trị `eligibility` theo `contracts/campaign-query-api.md` mục 3.4, **bao gồm** `eligible_pending` (đủ điều kiện, còn suất, chưa được cấp — FR-079) với `reason` **rỗng**. Không được gộp `eligible_pending` vào `eligible_but_exhausted`, vì sẽ khiến nhân viên nói với khách "hết suất" khi thật ra chiến dịch còn suất. Trường `reason` chỉ dùng tập **12 mã** ở mục 3.5 (không phải 14 mã) — xem ghi chú loại trừ ở đó
- [X] T081 [US7] Tạo `campaign_admin_handler.go` trong `services/account/internal/modules/subscription/presentation/handler/`; khai báo hằng số thông điệp tiếng Việt ở đầu tệp; dùng `validation.BindQuery` cho tham số truy vấn
- [X] T082 [US7] Đăng ký **nhóm mới** `/admin/campaigns` trong `services/account/internal/modules/subscription/presentation/router/router.go`, dùng `r.auth.Handle()` + `RolesAuthorize(roleslug.Admin)` + `AdminGrantRateLimitMiddleware`. **TUYỆT ĐỐI KHÔNG** đặt dưới nhóm `/admin/subscriptions` — sẽ làm hỏng lúc khởi động (R11)
- [X] T083 [US7] Cập nhật ctor của `Router` và chạy `make wire`.
  **⚠️ Tệp dùng chung**: không chạy song song với T057, T067, T099 — dễ mất thay đổi của nhau khi sinh lại cùng một tệp

**Điểm kiểm tra**: dịch vụ khởi động được (không hỏng định tuyến), ba trạng thái trả về khác nhau.

---

## Phase 8: US5 — Mở Được Chiến Dịch Mới Không Phải Sửa Mã Nguồn (P3)

**Mục đích**: Chứng minh thêm một chiến dịch chỉ cần khai báo, không sửa một dòng mã nguồn nào.

**Kiểm thử độc lập**: Thêm chiến dịch thứ hai vào cấu hình, cho tài khoản mới đăng ký, khẳng định
chiến dịch mới chạy. Không sửa tệp mã nguồn nào.

### Kiểm thử

- [X] T084 [P] [US5] Kiểm thử thêm chiến dịch thứ hai bằng cấu hình: tài khoản mới rơi vào chiến dịch mới, **không** có thay đổi mã nguồn — `campaign_multi_campaign_test.go` (SC-001)
- [X] T085 [P] [US5] Kiểm thử hết hạn theo thời gian: tài khoản tạo sau `ends_at` không được cấp, kể cả khi thông điệp tới muộn (FR-017) — cùng tệp với T084
- [X] T086 [P] [US5] Kiểm thử tài khoản đã tham gia chiến dịch trước không được tham gia chiến dịch sau (FR-012) — cùng tệp với T084. **Không** trích SC-009: SC-009 là 1000 thao tác AI đồng thời, nhiệm vụ này không chứng minh được
- [X] T086b [P] [US5] Kiểm thử chiến dịch **đã quá `ends_at`** không cấp được cho tài khoản mà chiến dịch đang chạy không phủ: chiến dịch A có `ends_at` thật và đã qua, B còn thức, tài khoản tạo trong cửa sổ A nhưng ngoài cửa sổ B — phải bị từ chối chứ không được A cấp (FR-013, `data-model.md` mục 6.1). Lỗi này **không lộ ra khi chỉ có một chiến dịch**, vì điều kiện 2 đã chặn sẵn — cùng tệp với T084

### Cài đặt

- [X] T089 [US5] Bổ sung cấu hình mẫu cho **chiến dịch thứ hai vào `configs/config.yaml` DUY NHẤT**, dùng để kiểm thử. **TUYỆT ĐỐI KHÔNG** thêm chiến dịch mẫu vào `configs/config.production.yaml` (FR-011 chọn chiến dịch bắt đầu muộn nhất → chiến dịch mẫu sẽ cướp lượt đăng ký thật của `launch-2026-10`; xem `quickstart.md` cổng an toàn)

**Điểm kiểm tra**: thêm một khối cấu hình, chiến dịch mới chạy, không sửa mã nguồn.

---

## Phase 9: US6 — Phía Bên Nhận: Bỏ Qua Tài Khoản Của Quản Trị Viên (P3)

**Mục đích**: Bỏ qua tài khoản do quản trị viên tạo, với lý do ghi nhận riêng.

**Phần lớn câu chuyện này đã nằm ở Phase 2** — cột `created_by_admin` (T015, T017) và cả ba
đường phát thông điệp (T030, T031, T032) — vì US1, US3 và US7 đều phụ thuộc vào chúng. Phase này
chỉ làm phần còn lại: phía bên nhận thông báo, và chứng minh hai nguồn không lệch nhau.

**Kiểm thử độc lập**: Tạo tài khoản bằng đường quản trị viên, khẳng định không tiêu suất nào
bị giảm và có gói miễn phí.

### Kiểm thử

- [X] T090 [P] [US6] Kiểm thử chống trôi lệch: `created_by_admin` trong cơ sở dữ liệu khớp `createdByAdmin` trong thông điệp cho cả ba đường tạo tài khoản — `services/account/internal/modules/identity/application/usecase/user_created_event_test.go` (SC-006). Đường dẫn gốc chỉ `eager_provision_test.go` là **sai**: coverage thật nằm ở đây, do T029b tạo
- [X] T091 [P] [US6] Kiểm thử tìm kiếm văn bản: trong `services/account` chỉ có **đúng một** chỗ ghi `CreatedByAdmin: true` — thêm vào `user_created_event_test.go` cùng T090 (SC-006). Quét **toàn bộ** `services/account`, không chỉ `modules/identity` — bản quét cũ không nhìn thấy writer thứ hai nằm ngoài identity
- [X] T092 [P] [US6] Kiểm thử tài khoản do quản trị viên tạo vẫn được cấp gói miễn phí (FR-051) — cùng tệp với T090 (SC-006)
- [X] T093 [P] [US6] Kiểm thử đường cấp gói thủ công của quản trị viên không bị ảnh hưởng (FR-051) — cùng tệp với T090 (SC-006)

### Cài đặt

- [x] T094 [US6] **Đã xong ở T042** (điều kiện 3 của bảng quyết định): `admission_uc.go:83` gọi `adminSkipReason(createdByAdmin)` trả lý do riêng `SkipCreatedByAdmin` = `created_by_admin`. Giữ lại dòng này như **xác nhận**, không phải việc mới — sửa lại để không dẫn người đọc tới viết lại thứ đã có. Tách bạch với **giá trị `eligibility` `not_eligible`** ở hợp đồng mục 3.4, KHÔNG phải với một mã `reason` (tập `reason` ở mục 3.5 đã **cố ý** không có `not_eligible`) (FR-050, FR-033)
- [X] T095 [US6] Lọc cùng điều kiện đó trong đường quét bù `sweep_uc.go`, dùng cột `created_by_admin` từ hợp đồng tài khoản
- [X] T096 [US6] Bảo đảm cơ chế quét bù **không** gọi lại `admission_uc.go` bằng một đường khác, mà dùng **chung** một hàm bỏ qua, để hai đường không thể lệch nhau (FR-080)

**Điểm kiểm tra**: tài khoản kiểm thử của quản trị viên không ăn suất của khách hàng thật.

---

## Phase 10: Cơ Chế Quét Bù, Đóng Sót Và Hoàn Thiện

**Mục đích**: Cơ chế quét bù, kiểm thử ranh giới còn lại, và chạy cẩu nang kiểm chứng.

**Đây là phase bắt buộc trước ngày 02/10, không phải để hoàn thiện**: cơ chế quét bù (T097–T099) là
lưới an toàn duy nhất cho thông điệp bị rơi. Thiếu nó thì mọi tài khoản có thông điệp thất lạc sẽ
không bao giờ được cấp gói, kể cả khi đã xử lý thủ công. Nó chỉ phụ thuộc Phase 2, 3, 4 — tức là
làm **song song với Phase 6 và 7** được.

- [x] T097 Cài `sweep_uc.go` trong `services/account/internal/modules/subscription/application/usecase/campaign/`: duyệt tài khoản theo lô trong cửa sổ chiến dịch qua hợp đồng, bỏ qua tài khoản đã có lượt cấp, rồi chạy **cùng** bộ luật hạn mức của đường thông điệp (FR-079, FR-080)
- [x] T098 Cài `campaign_sweep_worker.go` trong `services/account/internal/modules/subscription/presentation/worker/` theo mẫu `subscription_renewal_worker.go`: `cron.New(cron.WithSeconds())`, chạy một lần lúc khởi động, khoảng thời gian lấy từ cấu hình chứ không viết cứng (FR-083)
- [x] T099 Đăng ký `CampaignSweepWorker` trong `services/account/internal/modules/subscription/provider.go`, `internal/di/wire.go` và trường `CampaignSweep` trong `services/account/internal/bootstrap/worker.go`; chạy `make wire`.
  **⚠️ Tệp dùng chung**: không chạy song song với T057, T067, T083 — dễ mất thay đổi của nhau khi sinh lại cùng một tệp
- [x] T100 [P] [US3] Kiểm thử tính bất biến của mốc ưu tiên khi hai lần quét chạy song song: mốc **không** đổi giữa các lần (FR-030, SC-001)
- [x] T101 [P] [US1] Kiểm thử bù trễ: thông điệp tới muộn hơn 5 phút vẫn cấp gói, và sự kiện trễ được ghi nhận như sự kiện quan sát được (FR-078, FR-082, SC-015)
- [x] T102 [P] [US1] Kiểm thử quét bù đóng sót: tài khoản không có thông điệp nhưng đủ điều kiện vẫn được cấp (FR-079, FR-089, SC-016)
- [X] T103 [P] Kiểm thử canh FR-092: tập loại sự kiện đổi gói trong `shared/domain/constants/subscriptioneventtype/` vẫn đúng 7 phần tử, không thêm không bớt (FR-092)
- [X] T104 [P] Kiểm thử canh FR-098: module tạo tài khoản không có câu lệnh nào tra cứu gói cước — tìm kiếm văn bản trong `services/account/internal/modules/identity/` (FR-098)
- [X] T105 [P] Kiểm thử canh FR-095: lần cấp gói ghi **đúng một** dòng lịch sử thay đổi gói (SC-008)
- [X] T106 [P] Kiểm thử rằng **cả hai** tệp cấu hình đều có khóa `campaigns` và đều phân tích được khối đó; `config.yaml` chứa **hai** chiến dịch còn `config.production.yaml` chứa **một** (SC-014). Không đòi hai tệp giống nhau: buộc chúng giống nhau chính là cơ chế biến việc thử nghiệm ở dev thành thay đổi hành vi production (T089, xem `quickstart.md` cổng an toàn)
- [X] ~~T107~~ **KHÔNG CHẠY** — `AGENTS.md` cấm `make hygiene-scan` nếu không được dev cho phép rõ ràng; lệnh sinh tệp báo cáo làm rác kho mã nguồn Chạy `make hygiene-scan` **chỉ khi được cho phép rõ ràng** — lệnh này sinh tệp báo cáo trong kho mã nguồn
- [X] T108  ⚠️ **K2.6 KHÔNG kiểm chứng được**: nhánh hạ cấp nằm ở `SubscriptionRenewalWorker` với cron hard-code `0 0 0 * * *`, không chạy lúc khởi động, không có endpoint kích hoạt, không có biến cấu hình. 7/8 kịch bản PASS Chạy **8** kịch bản bắt buộc ở `quickstart.md` mục 2: K2.1 thông điệp ngược thứ tự, K2.2 giao lại 100 lần, K2.3 5 phiên bản song song, K2.4 tắt worker 10 phút, K2.5 thứ tự thuận, K2.6 hết hạn tự hạ về `free`, K2.7 cấp gói không trừ đồng bị, K2.8 đo độ chậm (SC-001d, SC-006, SC-007, SC-004). Không được bỏ K2.5 — đó là kịch bản chứng minh tính đúng đắn của thứ tự
- [X] T109 Chạy 7 kiểm tra ranh giới ở `quickstart.md` mục 6, gồm **7** cấu hình sai phải chặn khởi động (`plan_slug` sai được kiểm ở mục 4 với mã `plan_unavailable`, xem FR-010)
- [X] T110 ✅ `make check` exit 0 (7 bước: fmt-check, tidy-check, generate-check, nginx-check, vet, test-race 5 module, build-all) Chạy `make wire && make lint && make test-race` lần cuối và đối chiếu `quickstart.md` mục 8
- [X] T111 ⚠️ **dev đã xác nhận**: chiến dịch `launch-2026-10` chạy và đã cấp gói cho người đăng ký thật trên production. Trạng thái `running` được xác nhận qua bằng chứng production, không cần chạy lại local (lệnh này cố ý đặt trước 00:00 02/10/2026 — mốc đã qua) Khởi động dịch vụ với cấu hình thật của chiến dịch ra mắt và xác nhận `status` là `running` trước **00:00 ngày 02/10/2026** giờ Việt Nam

---

## Phụ Thuộc & Thứ Tự Thực Hiện

### Phụ thuộc giữa các phase

- **Phase 1 (Thiết Lập)**: không phụ thuộc — bắt đầu ngay
- **Phase 2 (Nền Tảng)**: phụ thuộc Phase 1 — **chặn mọi câu chuyện**
- **Phase 3 (US3)**: phụ thuộc Phase 2
- **Phase 4 (US1)**: phụ thuộc Phase 3 — bộ quyết định cấp phải có trước
- **Phase 5 (US4)**: phụ thuộc **chỉ Phase 1** — chạy song song được với Phase 2 và 3.
  **Không** chạy song song với Phase 4 hay Phase 7 vì cùng sinh lại `wire_gen.go` (xem T067)
- **Phase 6 (US2)**: phụ thuộc Phase 4
- **Phase 7 (US7)**: phụ thuộc Phase 2 và phần quét bù của Phase 10 (T097–T099, đã xong) — chạy song song với Phase 3, 4, 6 nhưng **không** với Phase 5. T079b/T079c **phải** nằm trong Phase 7 vì `contracts/campaign-query-api.md` mục 3.1 bắt buộc endpoint của Phase 7 trả `status`, `degraded`, `lastSweepAt`; đặt chúng ở Phase 8 thì kiểm thử Phase 7 không bao giờ xanh
- **Phase 8 (US5)**: phụ thuộc Phase 3, 4
- **Phase 9 (US6)**: phụ thuộc Phase 3
- **Phase 10 (Cơ Chế Quét Bù, Đóng Sót & Hoàn Thiện)**: phần quét bù (T097–T102) phụ thuộc Phase 2, 3, 4 —
  chạy song song với Phase 6 và 7 được. Các kiểm thử canh ranh giới phụ thuộc tất cả

### Phụ thuộc giữa các câu chuyện

- **US3 (P1)**: sau Phase 2. Không phụ thuộc câu chuyện nào
- **US1 (P1)**: sau US3. Không phụ thuộc câu chuyện nào khác
- **US4 (P2)**: sau Phase 1. **Hoàn toàn độc lập** — có thể giao cho người khác làm song song
- **US2 (P2)**: sau US1
- **US7 (P2)**: sau Phase 2, độc lập với US1
- **US5 (P3)**: sau US1
- **US6 (P3)**: phần lõi đã ở Phase 2; phần còn lại sau US3

### Trong mỗi câu chuyện

- Kiểm thử viết trước và phải **ĐỎ** trước khi cài đặt
- Lớp domain trước lớp persistence trước use case trước worker
- Cốt lõi trước tích hợp

### Cơ hội chạy song song

- Mọi nhiệm vụ `[P]` trong Phase 1 chạy song song
- **T017, T018, T019, T020, T025, T033, T035** trong Phase 2 chạy song song
- Sau Phase 2: **US3 và US4 chạy song song** (an toàn — Phase 3 không sinh lại `wire_gen.go`)
- **US4 không chạy song song với US1 hay US7** (cùng sinh lại `wire_gen.go`)
- Phần quét bù của Phase 10 chạy song song với US2 và US7
- Mọi nhiệm vụ kiểm thử `[P]` của một câu chuyện chạy song song

### Tài nguyên phải tuần tự hoá

`wire_gen.go` là **tài nguyên dùng chung**: 4 nhiệm vụ (T057, T067, T083, T099) đều sinh lại
tệp này bằng `make wire`. Quy tắc: trong mỗi nhóm chạy song song, chỉ **một** người chạy
`make wire` và người đó làm nhiệm vụ cuối cùng của nhóm. Lỗi thường gặp — hai người cùng sinh,
mất thay đổi của nhau, và lỗi biên dịch chỉ lộ ra ở người thứ ba chạy tiếp.

---

## Ví Dụ Chạy Song Song

### Phase 2

```powershell
# Nhóm A — lược đồ (khác tệp, không phụ thuộc)
Task: "T017 Thêm trường CreatedByAdmin vào entities.User"
Task: "T018 Thêm thực thể CampaignClaim"
Task: "T019 Thêm 2 trường vào UserBriefContractDTO"
Task: "T020 Thêm bộ lọc vào readmodels.go"
Task: "T025 Thêm UserCreatedEventPayload"
Task: "T033 Thêm ICampaignClaimRepository"
Task: "T035 Thêm lỗi chiến dịch"
```

### Phase 3 và Phase 5 chạy cùng lúc

```powershell
# Người A — trên đường găt 02/10
Task: "T041 Khai báo ICampaignUseCase"
Task: "T042 Cài admission_uc.go"
Task: "T043 Cài quota_uc.go"
Task: "T044 Cài campaign_uc.go"

# Người B — độc lập hoàn toàn
Task: "T063 Thêm trường uow vào UserQuotaUseCase"
Task: "T064 Sửa UpdateOutfitQuota"
Task: "T065 Sửa UpdateAiChatQuota"
Task: "T066 Sửa checkAndResetDailyQuota"
```

---

## Chiến Lược Thực Hiện

### MVP — giai đoạn 1: có gói quà tặng chạy được

1. Phase 1: Thiết Lập
2. Phase 2: Nền Tảng
3. Phase 3: US3 — bộ luật hạn mức
4. Phase 4: US1 — cấp gói thật
5. **DỪNG và KIỂM CHỨNG**: chạy K2.1 và K2.2 ở `quickstart.md`
6. Một tài khoản đăng ký mới nhận gói trong vòng 5 phút

### MVP — giai đoạn 2: đủ điều kiện ra mắt

7. Phase 5: US4 — sửa bộ đếm lượt
8. Phase 6: US2 — email
9. **Phase 10: cơ chế quét bù trước** (T097–T099), rồi mới tới kiểm thử ranh giới
10. **DỪNG và KIỂM CHỨNG**: chạy hết K2.1 tới K2.8 và 7 kiểm tra ranh giới
11. Sẵn sàng cho 02/10

### Sau ra mắt

12. Phase 7: US7 — giao diện tra cứu
13. Phase 8: US5 — chiến dịch thứ hai bằng cấu hình
14. Phase 9: US6 — phần còn lại

### Nếu phải cắt bớt để kịp hạn

Cắt theo thứ tự này, **đừng** cắt Phase 3, 4, 5, 6:

| Cắt | Mất gì | Vẫn ra mắt được không |
|---|---|---|
| Phase 7 (US7) | SC-011, SC-012 | Có — hỗ trợ tra cứu tay bằng dữ liệu lượt cấp, nhưng mất luôn máy trạng thái chiến dịch và cờ `degraded` |
| Phase 8 (US5) | FR-011 với nhiều chiến dịch | Có — cơ chế vẫn chạy với một chiến dịch |
| Phase 9 (US6) phần bên nhận | FR-050 | **Không** — tài khoản test sẽ ăn suất. Không được cắt |

---

## Ghi Chú

- `[P]` = khác tệp, không phụ thuộc
- `[Story]` ánh xạ tới câu chuyện người dùng trong `spec.md`
- Mỗi câu chuyện phải hoàn thành và kiểm chứng **độc lập**
- Dừng ở mỗi điểm kiểm tra để kiểm chứng câu chuyện
- Tránh: nhiệm vụ mơ hồ, xung đột cùng tệp, phụ thuộc chéo làm hỏng tính độc lập
- **Không** thêm loại sự kiện đổi gói mới (FR-092)
- **Không** sửa dịch vụ AI (FR-097)
- **Không** chứa dữ liệu nhận diện cá nhân trong thông điệp (FR-087)
- Đặt nhóm định tuyến cạnh tham số sẽ **hỏng lúc khởi động** (R11)
- Truyền nhầm bản ghi gói cũ sẽ tạo quyền AI **vô thời hạn** (R13)
---

## Phase 11: Tài liệu Tích Hợp Frontend

**Mục đích**: bộ phận hỗ trợ và vận hành cần trả lời câu hỏi khách hàng *"tôi có được tặng gói không"* mà không cần đọc code backend.

- [X] T112 Viết `specs/025-signup-campaign-grant/frontend-guide.md` cho **4 API tra cứu** ở Phase 7, theo khuôn của `specs/023-analyze-status-handling/frontend-guide.md`. Phải có: TL;DR, bảng 6 giá trị `eligibility` **kèm 12 mã `reason`**, bảng 5 trạng thái `status` + `degraded`, ví dụ JSON thật, bảng mã lỗi HTTP, checklist frontend, changelog
- [X] T113 Bổ sung công tắc `email.enabled` cho `shared/config` để **tắt đường gửi email khi chạy kịch bản hàng loạt**. Lý do: hiện `config.yaml` trỏ thẳng `smtp-relay.brevo.com` và không có công tắc nào, nên mọi kịch bản tạo nhiều tài khoản đều gửi email thật và đốt hạn mức Brevo (đã xảy ra: lượt K2.2+K2.5 cấp cho ~125 tài khoản). Khi có công tắc: K2.1/K2.2/K2.3/K2.4/K2.5/K2.7/K2.8 chạy được an toàn; chỉ một kịch bản riêng mới bật lại email để kiểm chứng nội dung, và kịch bản đó dùng **đúng một** tài khoản.. Đây là khoảng trống kiến trúc, **đã được sửa ngay trong 025** vì nó chặn việc chạy an toàn K2.2/K2.5

---

## Phase 12: Convergence

**Mục đích**: hai việc còn lại sau khi 122 nhiệm vụ đã hoàn tất. Cả hai đều là **khoảng trống kiểm chứng**, không phải khoảng trống hành vi — mã nguồn cả hai đường đã có, chỉ là chưa có bằng chứng giữ nó.

- [X] T114 Bổ sung kiểm thử chứng minh `subscription.plan.changed` được phát trên **3 đường còn thiếu** của FR-095 — FR-065 tới FR-072 không liên quan; FR-095 yêu cầu bằng chứng trên **ít nhất sáu** đường, và hiện chỉ có **3** đường được chứng minh. Đã có: cấp phát mặc định lúc tạo tài khoản (`TestDefaultProvisioningWritesEverythingInsideOneTransaction` + `TestEnsureDefaultSubscriptionWritesHistoryInsideTheCallersTransaction` đòi `eventRepo.creates == 1`), cấp gói quản trị viên (`TestAdminGrantBatchWritesEventAndAuditOnSameTransaction` đòi `events.lastCtx != nil`), và cấp gói chiến dịch (`TestCampaignGrantWritesExactlyOneHistoryRow` đòi đúng 1 dòng). **Còn thiếu**: (1) **mua gói** — `purchase/wallet.go:144` và `webhook/purchase_workflow.go:81,120` gọi `PublishPlanChange`, nhưng `purchase/checkout_test.go` không hề chạm nhánh này và `webhook_integration_test.go` chỉ đếm `ProviderPaymentEvent`/`WalletStatement`, **không** đếm dòng `user_subscription_events`; (2) **gia hạn** — `renewal_payment.go:111` gọi `PublishPlanChange`, `TestExecuteScheduledRenewalRenewsPaidPlan` chỉ đòi `statements.creates`/`updates`/`notif`, **không** đòi dòng lịch sử; (3) **hạ về `free` khi hết hạn** — `renewal_fallback.go:73,91` gọi `PublishPlanChange`, `TestExecuteScheduledRenewalDowngradesWhenAutoRenewOff` chỉ đòi `updates`/`notif`. Cả ba harness **đã nối sẵn** `NewPlanChangePublisher` với event repo giả (xem `renewal_test.go:331`, `webhook_integration_test.go:205`), nên nhiệm vụ là **thêm phép đo**, không phải dựng hạ tầng mới. Bổ sung vào `services/account/internal/modules/subscription/application/usecase/subscription/renewal_test.go` và `services/account/internal/modules/subscription/application/usecase/purchase/` (tệp kiểm thử mới nếu cần, theo quy ước H1). **Không** sửa `services/ai` (FR-097)
- [X] T115 Cập nhật **CỔNG AN TOÀN #2** ở `specs/025-signup-campaign-grant/quickstart.md` (mục 2) để nói đúng sự thật sau T113 (Constitution §Governance "Outbound email volume during verification", FR-005). Cổng hiện khẳng định **sai** rằng *"Hiện chưa có công tắc bật/tắt cho `email` trong `shared/config`"* và *"**chưa có cách nào an toàn để chạy K2.2 và K2.5** cho tới khi bổ sung công tắc đó"* — trong khi `email.enabled` đã có, mặc định `true`, và `EMAIL_ENABLED` ghi đè được (hiến phán §VIII). Sửa thành: nêu `email.enabled: false` trong `configs/config.yaml` hoặc `EMAIL_ENABLED=false`, xác nhận K2.2/K2.5 **chạy được an toàn** khi tắt, giữ nguyên quy tắc **một tài khoản cho kịch bản kiểm tra nội dung email**, và giữ cảnh báo "không chạy lại K2.2/K2.5 chỉ để kiểm tra cho chắc". Sửa **riêng** `quickstart.md`; `plan.md` và `data-model.md` **không** cần lặp lại quy tắc — nó đã nằm ở hiến phán
- [X] T116 Phủ nốt nhánh **thanh toán PayOS** của đường "mua gói" trong FR-095 (`webhook/purchase_workflow.go:120`). **Chưa làm vì cản trở thật, không phải vì khó**: `webhook_integration_test.go` `t.Skip` khi thiếu `TEST_DATABASE_URL`, và `cleanupWebhookTables` chạy `DELETE` không giới hạn trên `user_subscriptions`, `subscription_plans`, `user_wallets`, `wallet_statements`, `deposit_transactions` — tức xoá sạch dữ liệu dev. Cần dev cho phép rõ ràng trước khi chạy. FR-095 nêu 6 đường và đường "mua gói" đã được phủ qua nhánh ví (`purchase/wallet.go:144`), nên đây là phủ dư theo cùng nhóm, không phải thiếu sót so với FR. ⚠️ **Bằng chứng này có điều kiện**: nó thu được trên schema do `AutoMigrate` dựng, **không** phải schema do migration tạo, và hai schema đó đã được chứng minh là khác nhau (tên constraint `uni_subscription_plans_slug` so với `subscription_plans_slug_key`). Xem D-01 trong `deferred.md` trước khi tính FR-095 là đủ bằng chứng
- [X] T117 Ghi nhận trung thực **SC-007 chưa có bằng chứng kiểm chứng**: K2.6 (hết hạn tự hạ về `free`) không quan sát được vì `SubscriptionRenewalWorker` chạy cron `0 0 0 * *`, không chạy lúc khởi động và không có endpoint kích hoạt. Hành vi hạ cấp **đã có và đã được kiểm thử** (`TestExecuteScheduledRenewalDowngradesWhenAutoRenewOff`, `TestIntegrationScheduledRenewalDowngradesWithoutFunds`) — đây là khoảng trống **quan sát được**, không phải thiếu hành vi. Có thể quan sát được nếu dời `expires_at` về quá khứ rồi gọi `ProcessScheduledRenewals` trực tiếp (chính là `executeScheduledRenewal`, hàm mà integration test đang gọi). T108 đã tick với ghi chú này; `quickstart.md` vẫn liệt K2.6 trong tám kịch bản bắt buộc — trạng thái kiểm chứng phải nhất quán giữa hai tài liệu

---

## Nợ Kỹ Thuật Phát Hiện Khi Đóng Feature

Hai mục phát hiện lúc đóng feature **không thuộc phạm vi 025** và **không** để nguyên ô `[ ]` ở đây: ô trống nghĩa là *việc của 025 chưa làm*, mà hai mục này không phải việc của 025. Chúng đã chuyển sang `deferred.md` theo hiến phán §Governance "Technical debt found, not fixed" (v3.3.0).

- **D-01** — `AutoMigrate` trong harness webhook không chạy nổi trên schema do migration tạo. **Làm mất hiệu lực bằng chứng của `[X] T116`**: T116 chứng minh được nhánh PayOS của FR-095, nhưng trên schema **không** đúng hình dạng production. Đọc D-01 trước khi tin T116.
- **D-02** — `cleanupWebhookTables` xoá không giới hạn, và bảng `subscription_renewal_attempts` có FK không kèm `ON DELETE` nằm ngoài danh sách dọn.

Chi tiết, lý do ngoài phạm vi, và cách gỡ: `deferred.md`.
