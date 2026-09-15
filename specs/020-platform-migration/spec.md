# Feature Specification: Chuyển đổi hạ tầng sang nền tảng managed và tách 3 service (multi-module workspace)

**Feature Branch**: `020-platform-migration`

**Created**: 2026-09-14

**Status**: Draft

**Input**: User description: "Chuyển đổi hạ tầng hệ thống SmartWardrobe Backend từ VPS self-hosted sang các nền tảng managed được tài trợ trong 3 tháng (Cloudflare Pro, Supabase Pro, CloudAMQP Tough Tiger, Redis Cloud Essentials 250MB; compute Cloud Run dùng trial + teardown), tách mã nguồn thành 3 service triển khai độc lập trên multi-module Go workspace (bỏ monolith, giữ tag legacy), edge là Cloudflare Worker trên subdomain zone api-v2 (NS-delegate, không move zone gốc), cutover api.closy bằng SaaS custom hostname, tài liệu hóa runbook 2 chiều (lên Cloud và repatriation về VPS khi hết kinh phí)."

## Quyết định hiện hành (authoritative)

> **Mục này là nguồn chân lý.** `## Clarifications` bên dưới là **lịch sử hội thoại**; khi mâu thuẫn,
> mục này thắng. Agent implement chỉ cần đọc mục này + `plan.md` + `tasks.md`.

1. **Topology**: 3 service — Account (`identity,subscription,admin,notification`), Commerce
   (`wardrobe,fashion,brand,dashboard`), AI (`aiexecutor`) — trên **multi-module Go workspace**
   (`go.work`), service gom trong `services/`. Monolith `cmd/server` bị bỏ khỏi `main`, giữ tag
   `monolith-legacy` frozen. Contract liên service ở module `contracts/`.
2. **Contract**: chỉ 4 domain vượt ranh giới service lên protobuf (`identity,subscription,admin,ai`);
   8 contract nội service giữ in-process.
3. **DAG**: Commerce → Account, Commerce → AI, AI → Account, Account → (không gọi ra ngoài).
   Commerce không host gRPC.
4. **Edge**: Cloudflare Pro gắn subdomain zone `api-v2.closy.hycat.online` (NS-delegate từ Vercel;
   **không** move zone gốc) + Worker router. Route table **single-source** sinh cả Worker routes và
   Nginx includes, CI kiểm parity.
5. **Cutover**: `api.closy...` thành SaaS custom hostname (1 CNAME trên Vercel); sau cutover `api-v2...`
   redirect 301 về `api.closy...`; cutover hoàn tất **trước ngày chấm bài**.
6. **Compute**: Cloud Run `asia-southeast1`; HTTP scale-to-zero, worker `min-instances>=1`; trial +
   billing cap + teardown.
7. **Data**: 1 project Supabase Pro với **1 database dùng chung**; runtime qua pooler, migration +
   `LISTEN` qua direct connection; drill dùng **schema tạm** rồi xoá; migrate prod VPS → Supabase tại
   thời điểm merge/cutover.
8. **Queue/Cache**: CloudAMQP Tiger (100 connections) + Redis Cloud Essentials 250MB (256 connections),
   đều TLS.
9. **Search**: Elasticsearch tắt mặc định (`ELASTICSEARCH_ENABLED=false`), fallback database.
10. **Auth**: JWT bất đối xứng (ES256) cho user; Cloud Run ID token (`run.invoker`) cho gRPC nội bộ;
    session revocation qua Redis blacklist.
11. **Origin-auth**: edge tiêm `X-Gateway-Secret`; service verify, miễn trừ healthcheck hạ tầng.
12. **Migration**: chạy bằng Cloud Run Job riêng (Goose) trước deploy; service **không** tự migrate khi boot.
13. **SSE**: đi qua Worker dạng passthrough; heartbeat 15s; `ai_chat_chunk_deadline_seconds = 30s`
    **thống nhất mọi môi trường**; fallback gọi trực tiếp `run.app` khi vượt Workers Free.
14. **Swagger**: generate per-service, hợp đủ **9 section**, expose sau `swagger_access_code`.
15. **Repatriation**: về VPS bằng 3 service compose + Nginx gateway; **teardown trial là một bước trong
    repatriation**; funding clock T-30/T-14/T-7/T-1; runbook viết song song từ bây giờ.
16. **Vòng đời kinh phí**: 4 hạng mục funded duy trì 3 tháng; chi tiết ở
    `docs/project/funding-and-platform-lifecycle.md` (không thuộc hiến pháp).

## Trạng thái tài liệu (đọc trước khi code)

- **Nguồn chân lý giai đoạn 020**: `specs/020-platform-migration/` (spec.md, plan.md, tasks.md,
  data-model.md, contracts/, quickstart.md) + `.specify/memory/constitution.md` v3.0.0 + ADR 021.
- **Baseline CŨ (không dùng cho 020)**: `docs/system-design/architecture.md`, `README.md`,
  `docs/development/local-setup.md` vẫn mô tả modular monolith (`cmd/server`, `internal/modules` ở root,
  `internal/shared`). Chúng sẽ được cập nhật ở **T109–T111**. Không lấy các file này làm baseline khi
  implement 020.
- **Thứ tự đọc đề xuất**: spec.md (mục này) → plan.md (Structure Decisions + Technical Context) →
  data-model.md → contracts/ → tasks.md → quickstart.md (verify).

## Thuật ngữ

| Thuật ngữ | Nghĩa trong feature này |
|-----------|--------------------------|
| Mô hình C | Multi-module Go workspace (`go.work`), mỗi service một Go module, bỏ monolith khỏi `main` |
| `monolith-legacy` | Tag frozen tại commit monolith-green cuối cùng; chỉ tham chiếu khẩn cấp, không build CI |
| NS-delegate | Tạo bản ghi NS ở zone cha (Vercel) trỏ subdomain sang nameserver Cloudflare để Cloudflare quản zone con |
| SaaS custom hostname | Tính năng Cloudflare for SaaS: hostname ngoài zone (vd `api.closy...`) chạy qua edge Cloudflare bằng 1 CNAME |
| Worker | Cloudflare Worker; ở đây là edge-router đọc `routes.yaml` và tiêm `X-Gateway-Secret` |
| Origin-auth | Lớp chặn gọi thẳng `run.app`: edge tiêm `X-Gateway-Secret`, service verify |
| Single-source route table | `edge/worker/routes.yaml` là nguồn duy nhất sinh Worker routes và Nginx includes; CI kiểm parity |
| `run.app` | Hostname mặc định do Cloud Run cấp cho mỗi service (không publish; dùng nội bộ + fallback SSE) |
| Pooler / `LISTEN` | Supavisor transaction pooler cho runtime; `LISTEN/NOTIFY` cần direct connection nên tách DSN |
| Quorum queue | Loại queue RabbitMQ nhân bản; phải verify trên CloudAMQP shared plan, fallback classic durable |
| Outbox | Transactional outbox: ghi event cùng transaction nghiệp vụ, worker phát lên RabbitMQ |
| Repatriation | Đưa toàn bộ hệ thống từ nền tảng managed về VPS (3 service compose + Nginx gateway) |
| Funding clock | Mốc đếm ngược T-30/14/7/1 trước khi hết 3 tháng tài trợ, kèm checkpoint xác nhận |
| 9 section | 9 module tài liệu API: identity, subscription, admin, notification, wardrobe, fashion, brand, dashboard, aiexecutor |
| DAG | Đồ thị gọi liên service không có cycle (mục 3 ở trên) |
| Môi trường Cloud (drill) | Một môi trường Cloud duy nhất dùng chung 1 database; "drill" là chạy thử runbook bằng **schema tạm** rồi xoá. **Không có môi trường staging tách biệt.** |

## Clarifications

### Session 2026-09-14

- Q: Phạm vi tách service trong feature này tới đâu? → A: Tách code đủ cả 3 service (Account, Commerce Core, AI Engine) + tài liệu hướng dẫn.
- Q: Cơ chế xác thực giữa các service? → A: JWT bất đối xứng (identity ký bằng private key, service khác verify bằng public key cục bộ + blacklist cache).
- Q: Topology database cho 3 service? → A: Một database dùng chung, tách schema theo module. *(Đợt 2 làm rõ: mỗi service có connection pool riêng trên cùng database, không dùng chung một connection.)*
- Q: Nền tảng compute cho các service? → A: GCP Cloud Run (HTTP scale-to-zero; worker đặt min-instances>=1 và CPU luôn bật), trên bộ nền tảng managed được tài trợ: Cloudflare Pro (edge), Supabase Pro (Postgres + pgvector), CloudAMQP Tough Tiger (RabbitMQ), Redis Cloud Essentials 250MB (cache).
- Q: Runbook được kiểm chứng ở đâu trước cutover? → A: Có môi trường staging/tiền production để chạy thử toàn bộ runbook trước khi áp dụng production. *(Đợt 2 thay đổi: không còn môi trường tách biệt — dùng chung 1 database Cloud, drill bằng schema tạm rồi xoá.)*

### Session 2026-09-15 (khóa kiến trúc theo hạng mục được tài trợ)

- Q: Edge đặt ở đâu khi zone gốc không được move sang Cloudflare? → A: NS-delegate subdomain `api-v2.closy.hycat.online` thành zone Cloudflare riêng (gắn Pro), Worker edge-router định tuyến tới 3 Cloud Run; cutover `api.closy...` bằng SaaS custom hostname (1 CNAME trên Vercel).
- Q: Compute lấy kinh phí từ đâu? → A: Cloud Run trial/free-tier + billing cap cứng + teardown (compute không thuộc diện đầu tư). *(Đợt 2 làm rõ: teardown gộp vào repatriation.)*
- Q: Supabase dùng mấy project? → A: 1 project Pro. *(Đợt 2 thay đổi: 1 database dùng chung, bỏ tách schema staging/prod).*
- Q: Dữ liệu prod xử lý thế nào khi cutover? → A: Migrate prod VPS → Supabase (dump/restore + backfill + rollback).
- Q: Hết 3 tháng tài trợ thì sao? → A: Repatriation bắt buộc toàn bộ về VPS (3 service chạy compose + Nginx gateway, dữ liệu migrate ngược); runbook repatriation viết song song từ bây giờ, funding clock T-30/14/7/1.
- Q: Monolith giữ hay bỏ? → A: Bỏ khỏi `main` (mô hình C multi-module + `go.work`); giữ tag `monolith-legacy` frozen để tham chiếu khẩn cấp; runtime VPS sau này là 3-service compose, không phải monolith.

### Session 2026-09-15 (đợt 2 — chốt vòng đời, dữ liệu, SSE và ưu tiên)

- Q: Teardown trial xảy ra lúc nào so với repatriation? → A: Gộp — teardown trial là một phần của repatriation; hệ thống managed vẫn chạy tới mốc repatriation, không teardown riêng sau demo.
- Q: Ưu tiên US5 và US3? → A: Cả hai nâng lên P1 (US5 là tiền đề của US3/US6/US7; US3 là mốc cứng vì chấm bài cần cả `api.closy` trên Cloud).
- Q: Tách staging/prod trong 1 project Supabase? → A: **1 database dùng chung** cho môi trường Cloud (bỏ phương án tách schema staging/prod); drill runbook dùng schema tạm rồi xoá.
- Q: Vai trò của `api-v2` trong giai đoạn song song? → A: Test nội bộ với dữ liệu dump một lần; người dùng thật vẫn ở VPS; dump lại VPS → Supabase tại thời điểm merge/cutover.
- Q: Bảo vệ origin Cloud Run khỏi gọi thẳng `run.app`? → A: Thêm ràng buộc origin-auth: edge tiêm `X-Gateway-Secret`, service verify (trừ healthcheck hạ tầng).
- Q: SSE/AI streaming đi đường nào? → A: Đi qua Cloudflare Worker (passthrough stream) cùng hostname `api-v2`; dùng Workers Free, fallback gọi trực tiếp `run.app` nếu vượt hạn mức.
- Q: Xử lý idle timeout 100s của Cloudflare proxy? → A: Heartbeat ~15s và hạ `ai_chat_chunk_deadline_seconds` xuống 30s, **thống nhất mọi môi trường** (dùng chung `configs/`, tránh phân nhánh config theo môi trường).
- Q: Xác thực gọi gRPC giữa 3 service? → A: Cloud Run ID token (`run.invoker`) cho tầng hạ tầng + JWT bất đối xứng cho tầng ứng dụng (hai lớp độc lập).
- Q: Ai chạy Goose migration trên DB dùng chung? → A: Một migration job riêng chạy trước khi deploy service; không service nào tự migrate lúc khởi động.
- Q: Tài liệu API phơi ở đâu trên managed? → A: Giữ expose sau `swagger_access_code`, hợp của 3 service phải đủ 9 section (9 module: identity, subscription, admin, notification, wardrobe, fashion, brand, dashboard, aiexecutor).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Tắt search engine mặc định mà hệ thống vẫn hoạt động (Priority: P1)

Đội vận hành/dự án không có kinh phí cho search engine. Họ cần hệ thống chạy đầy đủ các chức năng tìm kiếm và tủ đồ khi search engine bị tắt, mà không phải sửa mã nguồn.

**Why this priority**: Search engine là hạ tầng đắt tiền và không được tài trợ. Nếu tắt mà hệ thống hỏng hoặc sinh lỗi hàng loạt, production không thể chạy.

**Independent Test**: Có thể kiểm chứng bằng cách tắt search engine qua cấu hình, sau đó thực hiện tìm kiếm trang phục hệ thống và các thao tác thêm/sửa/xóa trang phục; kết quả tìm kiếm vẫn trả về và không có sự kiện nào bị kẹt ở trạng thái thất bại hàng loạt.

**Acceptance Scenarios**:

1. **Given** search engine bị tắt bằng cấu hình, **When** người dùng tìm kiếm trang phục hệ thống, **Then** hệ thống trả về kết quả từ nguồn dự phòng (database) và phản hồi bình thường.
2. **Given** search engine bị tắt, **When** một trang phục được thêm/sửa/xóa, **Then** thao tác nghiệp vụ thành công và không phát sinh sự kiện thất bại lặp lại (không dồn hàng đợi lỗi).
3. **Given** search engine được bật trở lại, **When** hệ thống chạy, **Then** dữ liệu tìm kiếm được đồng bộ lại mà không cần can thiệp thủ công.

---

### User Story 2 - Chạy ổn định trên database managed có connection pooler (Priority: P1)

Đội vận hành chuyển database sang dịch vụ managed dùng connection pooler. Họ cần ứng dụng khởi động và vận hành đúng, bao gồm cơ chế phát sự kiện nền và cập nhật cấu trúc dữ liệu.

**Why this priority**: Nếu ứng dụng không khởi động được hoặc mất sự kiện nền thì toàn bộ luồng bất đồng bộ (AI, thanh toán, đồng bộ) đổ vỡ.

**Independent Test**: Có thể kiểm chứng bằng cách cấu hình database trỏ tới chế độ pooler; khởi động ứng dụng phải thành công và các sự kiện nền vẫn được phát đi (qua cơ chế dự phòng nếu kênh lắng nghe không khả dụng).

**Acceptance Scenarios**:

1. **Given** database không hỗ trợ kênh lắng nghe ở chế độ pooler, **When** ứng dụng khởi động, **Then** ứng dụng vẫn khởi động thành công và cơ chế phát sự kiện nền chuyển sang phương án dự phòng.
2. **Given** cập nhật cấu trúc dữ liệu cần chạy, **When** môi trường production được cấu hình không tự chạy cập nhật lúc khởi động, **Then** cập nhật được chạy có kiểm soát qua kênh phù hợp trước khi triển khai.

---

### User Story 3 - Chuyển production khỏi VPS sang nền tảng managed theo runbook (Priority: P1)

Đội vận hành cần một tài liệu đầy đủ, copy-paste, để chuyển hệ thống đang chạy trên VPS sang nền tảng managed trong một cửa sổ bảo trì. Edge là Cloudflare Pro trên subdomain zone `api-v2.closy.hycat.online` (NS-delegate từ Vercel, không move zone gốc), Worker định tuyến tới 3 service Cloud Run; cutover `api.closy...` bằng SaaS custom hostname.

**Why this priority**: Việc di chuyển hạ tầng là thay đổi lớn, dễ sai sót; thiếu runbook sẽ gây downtime kéo dài hoặc mất dữ liệu. Đây là **mốc cứng** vì yêu cầu môn học chỉ được thoả mãn khi cả `api-v2...` lẫn `api.closy...` chạy trên Cloud trước ngày chấm bài.

**Independent Test**: Có thể kiểm chứng bằng cách thực hiện đúng runbook trên môi trường Cloud và xác nhận hệ thống hoạt động sau cutover.

**Acceptance Scenarios**:

1. **Given** hệ thống đang chạy trên VPS, **When** đội vận hành làm theo runbook, **Then** toàn bộ nền tảng managed (database, cache, message broker, edge, compute) được cấu hình và ứng dụng chạy trên nền tảng đó, truy cập được qua `api-v2...`.
2. **Given** cutover bắt đầu trong cửa sổ bảo trì, **When** hoàn tất các bước migrate dữ liệu (VPS → Supabase) và gắn SaaS custom hostname, **Then** người dùng cuối truy cập bình thường qua `api.closy...` và dữ liệu được chuyển đầy đủ; `api-v2...` redirect 301 về `api.closy...`.
3. **Given** ngày chấm bài đã được ấn định, **When** kiểm tra điều kiện, **Then** cả `api-v2...` và `api.closy...` đã chạy trên Cloud trước mốc đó.
4. **Given** có lỗi trong quá trình cutover, **When** kích hoạt phương pháp khôi phục, **Then** hệ thống trở về VPS trong thời gian cho phép.

---

### User Story 4 - Quay lại VPS bằng đổi cấu hình và DNS, không sửa code (Priority: P2)

Khi hết thời gian tài trợ nền tảng managed (3 tháng), đội vận hành phải đưa hệ thống về chạy trên VPS dưới dạng **3 service chạy Docker Compose + Nginx gateway**, không cần viết lại mã nguồn. Monolith cũ chỉ còn là tag frozen tham chiếu, không phải runtime quay về.

**Why this priority**: Kinh phí managed chỉ duy trì 3 tháng; nếu không có đường quay về, dự án rơi vào tình trạng mất khả năng vận hành.

**Independent Test**: Có thể kiểm chứng bằng cách migrate dữ liệu từ Supabase về Postgres VPS, dựng 3 service + Nginx gateway bằng compose, đổi DNS về VPS, rồi xác nhận hệ thống chạy đầy đủ.

**Acceptance Scenarios**:

1. **Given** hệ thống đang chạy trên nền tảng managed, **When** đội vận hành migrate dữ liệu về VPS và đổi cấu hình, **Then** 3 service chạy được trên VPS mà không sửa mã nguồn nghiệp vụ.
2. **Given** cấu hình môi trường đã đổi, **When** so sánh, **Then** sự khác biệt chỉ nằm ở cấu hình và DNS, không nằm ở mã nguồn nghiệp vụ.
3. **Given** bản sao dữ liệu được tạo trước khi hết hạn tài trợ, **When** cần phục hồi, **Then** có thể phục hồi toàn bộ dữ liệu vào VPS.

---

### User Story 5 - Tách 3 service triển khai độc lập trên multi-module workspace (Priority: P1)

Đội phát triển cần tách hệ thống thành 3 service triển khai độc lập (Account, Commerce Core, AI Engine) trên multi-module Go workspace (`go.work`) để đáp ứng yêu cầu môn học và cô lập workload AI. Monolith bị loại khỏi `main` (giữ tag `monolith-legacy` frozen); ranh giới service do compiler enforce, không còn kiểm tra manifest thủ công.

**Why this priority**: Là tiền đề bắt buộc của US3/US6/US7 (không có image 3 service thì không deploy được lên Cloud) và là yêu cầu môn học; đồng thời đảm bảo đường quay về VPS (3-service compose) không bị mất.

**Independent Test**: Có thể kiểm chứng bằng cách build 3 binary service, xác nhận từng service khởi động, giao tiếp được với các service khác, và import implementation xuyên service bị chặn ở compile-time.

**Acceptance Scenarios**:

1. **Given** mã nguồn được tách theo service trên `go.work`, **When** build, **Then** cả 3 binary service đều build thành công và `cmd/server` không còn tồn tại trên `main`.
2. **Given** hệ thống chạy ở topology 3 service, **When** một service cần dữ liệu của service khác, **Then** giao tiếp diễn ra qua hợp đồng liên service mà không vi phạm ranh giới module.
3. **Given** một service import implementation của module thuộc service khác, **When** build, **Then** build thất bại ở compile-time.

---

### User Story 6 - Teardown trial GCP gộp vào repatriation, bill về 0 (Priority: P2)

Compute Cloud Run không thuộc diện đầu tư nên toàn bộ tài nguyên trial GCP phải được xóa **như một phần của repatriation** (không tách rời sau demo) và hóa đơn trial về 0. Hệ thống managed vẫn chạy tới mốc repatriation; 4 hạng mục funded được tắt trước ngày renew.

**Why this priority**: Tránh phát sinh chi phí ngoài ngân sách được duyệt.

**Independent Test**: Có thể kiểm chứng bằng cách chạy runbook teardown trên môi trường Cloud rồi đối chiếu billing trial.

**Acceptance Scenarios**:

1. **Given** repatriation được kích hoạt, **When** chạy runbook (teardown là một bước trong đó), **Then** mọi service Cloud Run scale về 0 và bị xóa, không còn forwarding/resource tính phí nào trên trial.
2. **Given** teardown hoàn tất, **When** đối chiếu billing, **Then** không còn chi phí mới phát sinh trên trial GCP.

---

### User Story 7 - Repatriation có lịch trình khi hết 3 tháng tài trợ (Priority: P2)

Trước khi 4 hạng mục funded hết hạn (3 tháng), đội vận hành thực hiện repatriation toàn bộ hệ thống về VPS theo funding clock T-30/T-14/T-7/T-1: verify backup, drill restore, rehearsal rollback DNS, migrate ngược dữ liệu, tắt funded items trước ngày renew.

**Why this priority**: Hết kinh phí mà chưa về VPS = mất khả năng vận hành + bị charge thêm chu kỳ mới.

**Independent Test**: Có thể kiểm chứng bằng drill repatriation đầy đủ trên môi trường Cloud trước T-30.

**Acceptance Scenarios**:

1. **Given** còn 30 ngày tới hạn tài trợ, **When** bắt đầu funding clock, **Then** mọi checkpoint T-30/T-14/T-7/T-1 đều có người xác nhận và bằng chứng thực thi.
2. **Given** repatriation hoàn tất, **When** kiểm chứng, **Then** hệ thống chạy đầy đủ trên VPS (3 service + Nginx gateway), dữ liệu đầy đủ, và không còn hạng mục funded nào bị charge thêm.

---

### Edge Cases

- Search engine bị tắt nhưng vẫn có sự kiện đồng bộ phát sinh liên tục từ thao tác người dùng.
- Kênh lắng nghe sự kiện nền bị mất kết nối giữa chừng (không chỉ lúc khởi động).
- Môi trường managed bắt buộc kết nối mã hóa (TLS) trong khi VPS không yêu cầu TLS.
- Database pooler không hỗ trợ khóa phiên hoặc kênh lắng nghe.
- Tài trợ nền tảng hết hạn đột ngột trước khi kịp cutover/quay về.
- Dữ liệu phát sinh trên nền tảng sau cutover nhưng chưa được sao lưu.
- Thông tin xác thực cũ trên VPS (khóa, token) chưa được thu hồi sau khi chuyển đi.
- Trial GCP hết credit giữa kỳ demo (billing cap chặn service → phải có runbook khôi phục/thu hẹp phạm vi).
- NS-delegate `api-v2` thất bại (Vercel chặn hoặc Cloudflare chưa cấp nameserver) → fallback SaaS custom hostname CNAME thẳng, cùng Worker phục vụ.
- Quorum queue không được hỗ trợ trên CloudAMQP shared plan → fallback classic durable + ADR bổ sung.
- SSE/AI streaming qua edge: heartbeat 15s và `ai_chat_chunk_deadline_seconds` = 30s (thống nhất mọi môi trường); origin trả `X-Accel-Buffering: no`; test stream thật trước cutover.
- Worker free vượt 100k req/ngày trong đợt cao điểm → chi phí Workers paid chưa funded, phải báo trước; fallback gọi trực tiếp `run.app` cho SSE.
- Drill và môi trường Cloud dùng chung **1 database** → thao tác drill ghi nhầm dữ liệu thật (bắt buộc dùng schema tạm rồi xoá, không đụng schema chính).
- Trong giai đoạn song song, dữ liệu `api-v2` (Supabase) lệch dần so với VPS vì VPS vẫn nhận ghi của người dùng thật → phải dump lại tại cutover và xác nhận không mất ghi nào trong cửa sổ chuyển.
- Cloudflare proxy idle timeout 100s trong khi stream AI có khoảng lặng lâu → Error 524; phải có heartbeat và hạ chunk deadline (thống nhất mọi môi trường).
- Chấm bài diễn ra trước khi cutover hoàn tất → không thoả yêu cầu "chạy trên Cloud" cho `api.closy`; cutover là mốc cứng phải xong trước ngày chấm.

## Requirements *(mandatory)*

### Functional Requirements

**Cấu hình linh hoạt theo môi trường**

- **FR-001**: Hệ thống MUST cho phép bật/tắt search engine hoàn toàn bằng cấu hình môi trường, không cần sửa mã nguồn.
- **FR-002**: Khi search engine bị tắt, các chức năng tìm kiếm MUST tự động dùng nguồn dự phòng và trả kết quả cho người dùng.
- **FR-003**: Khi search engine bị tắt, các thao tác đồng bộ chỉ mục MUST không tạo ra sự kiện thất bại lặp lại và không làm đầy hàng đợi lỗi.
- **FR-004**: Hệ thống MUST hỗ trợ cấu hình kết nối mã hóa (TLS) cho cache và message broker, với mặc định tắt để tương thích VPS.
- **FR-005**: Mọi tham số kết nối hạ tầng (host, port, chế độ SSL, TLS, cổng kênh lắng nghe, bật/tắt cập nhật cấu trúc khi khởi động) MUST cấu hình được qua biến môi trường.
- **FR-006**: File cấu hình MUST NOT hardcode host/endpoint theo từng môi trường.
- **FR-007**: Hệ thống MUST NOT phụ thuộc tính năng độc quyền của nhà cung cấp managed; chỉ dùng khả năng chuẩn của database, cache và message broker.
- **FR-008**: Việc chuyển đổi giữa môi trường self-hosted và managed MUST chỉ cần thay đổi cấu hình và DNS.

**Độ bền của hạ tầng nền**

- **FR-009**: Ứng dụng MUST khởi động thành công khi kênh lắng nghe sự kiện nền không khả dụng (ví dụ do connection pooler).
- **FR-010**: Khi kênh lắng nghe không khả dụng, cơ chế phát sự kiện nền MUST tự chuyển sang phương án dự phòng và tiếp tục phát sự kiện.
- **FR-011**: Hệ thống MUST cho phép bật/tắt việc tự động chạy cập nhật cấu trúc dữ liệu lúc khởi động; mặc định an toàn cho môi trường managed.
- **FR-012**: Hệ thống MUST hỗ trợ chạy cập nhật cấu trúc dữ liệu qua kênh kết nối phù hợp (phiên trực tiếp) trước khi triển khai.

**Tài liệu chuyển đổi và khả năng quay về**

- **FR-013**: Dự án MUST cung cấp runbook đầy đủ cho việc chuyển từ VPS sang nền tảng managed (bao gồm tạo tài nguyên, ánh xạ biến môi trường, migrate dữ liệu, cutover DNS, kiểm thử).
- **FR-014**: Dự án MUST cung cấp runbook đối xứng cho việc quay lại VPS (reverse cutover).
- **FR-015**: Runbook MUST ghi rõ điều kiện tiên quyết, thứ tự thực hiện, cách kiểm thử và cách rollback.
- **FR-016**: Runbook MUST bao gồm quy trình thu hồi thông tin xác thực cũ và ngừng VPS sau khi đã xác nhận.
- **FR-017**: Dự án MUST cung cấp hướng dẫn tách 3 service triển khai độc lập, bao gồm ranh giới service, cơ chế giao tiếp và cấu trúc triển khai.

**Service topology (multi-module workspace, không còn monolith)**

- **FR-018**: Mã nguồn MUST tổ chức thành multi-module Go workspace (`go.work`): một module mỗi service; `cmd/server` (monolith) MUST bị xóa khỏi `main`; commit monolith-green cuối cùng MUST được giữ ở tag `monolith-legacy` frozen.
- **FR-019**: Hợp đồng giao tiếp liên module MUST có cả triển khai trong cùng tiến trình (module nội bộ service) và triển khai gọi qua mạng (gRPC, module khác service), để đổi môi trường không cần sửa logic nghiệp vụ.
- **FR-020**: Quy trình CI MUST build và kiểm tra cả 3 binary service; import implementation xuyên service MUST fail ở compile-time (thay thế service manifest + boundary-check thủ công).
- **FR-021**: Tag `monolith-legacy` MUST được giữ nguyên để tham chiếu khẩn cấp và MUST NOT nhận logic mới; runtime quay về VPS là 3-service compose, không phải monolith.

**Bảo mật liên service**

- **FR-022**: Hệ thống MUST xác thực yêu cầu giữa các service bằng JWT bất đối xứng: chỉ service identity giữ private key để ký; các service còn lại chỉ giữ public key để xác thực cục bộ mà không cần gọi mạng.
- **FR-023**: Việc thu hồi phiên MUST được tra cứu qua cache dùng chung, không phụ thuộc vào việc gọi lại service identity cho mỗi yêu cầu.
- **FR-024**: Các service MUST dùng chung một database với tách schema theo module; mọi truy vấn/join xuyên module qua tầng dữ liệu MUST bị cấm (chỉ giao tiếp qua hợp đồng).

**Nền tảng managed được tài trợ (3 tháng)**

- **FR-025**: Hệ thống MUST triển khai trên đúng các nền tảng được tài trợ: Cloudflare Pro (edge, gắn vào subdomain zone `api-v2...`), Supabase Pro 1 project với **1 database dùng chung** cho môi trường Cloud (Postgres + pgvector), CloudAMQP Tough Tiger (RabbitMQ), Redis Cloud Essentials 250MB (cache); compute dùng GCP Cloud Run trial + billing cap + teardown; MUST NOT phụ thuộc dịch vụ ngoài danh sách này.
- **FR-026**: Ở topology service chạy trên Cloud Run, các tiến trình nền (worker) MUST được cấu hình để chạy liên tục (không bị scale-to-zero).
- **FR-027**: Toàn bộ runbook chuyển đổi (và quay về) MUST được chạy thử thành công trên môi trường Cloud **trước** khi áp dụng cho production; vì môi trường Cloud dùng chung 1 database (không tách môi trường riêng), drill MUST thao tác trên **schema tạm** và xoá sau khi kết thúc, không đụng schema chính.

**Edge, tài liệu API và vòng đời tài trợ**

- **FR-028**: Edge `api-v2...` MUST là Cloudflare subdomain zone (NS-delegate, không move zone gốc) + Worker router single-source; cùng route table MUST sinh được cả Worker routes và Nginx includes cho repatriation, CI check parity.
- **FR-029**: Cutover `api.closy...` MUST dùng SaaS custom hostname (1 CNAME trên Vercel); sau cutover `api-v2...` MUST redirect 301 về `api.closy...`; cutover MUST hoàn tất trước ngày chấm bài.
- **FR-030**: Mỗi service MUST tự generate tài liệu API của các module nó chứa; hợp của 3 service MUST giữ đủ **9 section** (identity, subscription, admin, notification, wardrobe, fashion, brand, dashboard, aiexecutor) và MUST tiếp tục được expose sau `swagger_access_code`; CI fail nếu thiếu section (parity check).
- **FR-031**: Teardown trial GCP MUST là một bước trong repatriation (không tách rời sau demo); MUST xoá toàn bộ tài nguyên trial và có bằng chứng bill trial về 0.
- **FR-032**: Funding clock T-30/T-14/T-7/T-1 MUST được thực thi có xác nhận trước khi hết 3 tháng tài trợ; repatriation về VPS MUST hoàn tất trước ngày renew để không bị charge thêm.

**Bảo vệ origin và xác thực liên service**

- **FR-033**: Mọi request từ edge vào service MUST mang `X-Gateway-Secret` do edge tiêm; service MUST từ chối request thiếu/sai secret, **trừ** healthcheck hạ tầng. CI MUST có test xác nhận request thiếu secret bị từ chối.
- **FR-034**: Giao tiếp gRPC giữa các service MUST xác thực hai lớp: Cloud Run ID token (`run.invoker`) ở tầng hạ tầng và JWT bất đối xứng ở tầng ứng dụng.
- **FR-035**: Cập nhật cấu trúc dữ liệu MUST được chạy bởi **một migration job riêng** trước khi deploy service; không service nào tự chạy migration lúc khởi động.

**SSE/AI streaming sau edge**

- **FR-036**: SSE/AI streaming MUST đi qua Cloudflare Worker dạng passthrough stream; MUST có heartbeat (~15s) để không vượt idle timeout 100s của Cloudflare; `ai_chat_chunk_deadline_seconds` MUST được đặt thống nhất **30s cho mọi môi trường** (dùng chung `configs/`).
- **FR-037**: Nếu vượt hạn mức Workers Free, hệ thống MUST có đường fallback gọi trực tiếp `run.app` cho SSE mà không cần đổi mã nghiệp vụ.

### Key Entities *(include if feature involves data)*

- **Hồ sơ cấu hình môi trường**: tập biến môi trường phân biệt môi trường local/VPS/managed (kết nối database, cache, broker, search engine, cờ cập nhật cấu trúc, TLS).
- **Runbook chuyển đổi**: tài liệu hướng dẫn có thể thực thi, gồm các bước, lệnh kiểm thử và rollback cho từng nền tảng — viết 2 chiều (lên Cloud và repatriation về VPS).
- **Binary triển khai**: 3 binary service từ multi-module workspace (`go.work`); không còn monolith trên `main`.
- **Route table single-source**: định nghĩa tuyến edge một lần, sinh Worker routes (Cloud) và Nginx includes (VPS), CI check parity.
- **Funding clock**: deadline 3 tháng + checkpoint T-30/T-14/T-7/T-1 có xác nhận.
- **Bản sao dữ liệu (dump/backup)**: ảnh chụp dữ liệu dùng cho migrate, phục hồi và repatriation (trước cutover, trước reverse, trước hạn tài trợ).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% trường hợp tìm kiếm trang phục hệ thống trả về kết quả khi search engine bị tắt.
- **SC-002**: Ứng dụng khởi động thành công trong 100% lần chạy khi cơ chế lắng nghe sự kiện nền không khả dụng.
- **SC-003**: Có thể chuyển từ VPS sang nền tảng managed trong cửa sổ bảo trì tối đa 2 giờ.
- **SC-004**: Có thể khôi phục hệ thống về VPS (3 service + Nginx gateway) chỉ bằng thay đổi cấu hình và DNS, trong tối đa 4 giờ (bao gồm migrate ngược dữ liệu).
- **SC-005**: 0 tham số host/endpoint hạ tầng bị hardcode trong file cấu hình theo môi trường.
- **SC-006**: 100% binary (3 service, không còn monolith) build thành công trong CI; import xuyên service fail ở compile-time.
- **SC-007**: 0 sự kiện đồng bộ bị kẹt ở trạng thái thất bại do search engine bị tắt.
- **SC-008**: Toàn bộ dữ liệu production được phục hồi thành công vào VPS từ bản sao dữ liệu trước hạn tài trợ.
- **SC-009**: 100% các bước runbook chuyển đổi và quay về chạy thành công trên môi trường Cloud trước khi thực hiện trên production.
- **SC-010**: Sau teardown, 0 chi phí mới phát sinh trên trial GCP (đối chiếu billing).
- **SC-011**: Repatriation hoàn tất trước ngày renew funded items; 0 hạng mục bị charge thêm chu kỳ.
- **SC-012**: Hợp tài liệu API của 3 service giữ đủ 9 section; route Worker ↔ Nginx parity 100%.
- **SC-013**: 100% request thiếu `X-Gateway-Secret` bị service từ chối (trừ healthcheck).
- **SC-014**: Stream AI dài giữ được kết nối qua edge với heartbeat 15s; 0 lỗi 524 do khoảng lặng.
- **SC-015**: Cả `api-v2` và `api.closy` chạy trên Cloud trước ngày chấm bài.

## Assumptions

- Kinh phí 4 hạng mục funded duy trì 3 tháng; sau đó hệ thống repatriation về VPS (3 service compose + Nginx gateway).
- Compute Cloud Run dùng trial/free-tier + billing cap; **teardown trial là một bước trong repatriation** (không tách rời sau demo).
- Zone gốc `hycat.online` ở lại Vercel; chỉ NS-delegate `api-v2...` sang Cloudflare (fallback: SaaS custom hostname CNAME thẳng).
- Supabase dùng 1 project Pro với **1 database dùng chung** cho môi trường Cloud; drill runbook dùng schema tạm rồi xoá.
- Giai đoạn song song: `api-v2` chỉ phục vụ test nội bộ với dữ liệu dump một lần; người dùng thật vẫn ở VPS; dump lại VPS → Supabase tại thời điểm merge/cutover.
- Tag `monolith-legacy` frozen tại commit monolith-green cuối cùng, chỉ tham chiếu khẩn cấp.
- Search engine không được cấp kinh phí và sẽ tắt mặc định; nguồn dự phòng là database.
- Docker Compose tiếp tục được dùng cho môi trường local và cho VPS sau repatriation.
- Việc sao lưu dữ liệu thực hiện: trước cutover, trước reverse cutover, trước hạn tài trợ, và theo funding clock T-minus (thay thế chiến lược "chỉ dump trước hạn" cũ).
- Thời điểm repatriation theo funding clock, phải hoàn tất trước ngày renew.
- Các cửa sổ bảo trì cho phép dừng hệ thống trong thời gian ngắn để migrate dữ liệu và chuyển DNS.
- SSE/AI streaming đi qua Cloudflare Worker ở dạng passthrough; heartbeat ~15s; `ai_chat_chunk_deadline_seconds` = 30s thống nhất mọi môi trường (dùng chung `configs/`); fallback `run.app` nếu vượt Workers Free.
- Chấm bài cần cả `api.closy` trên Cloud nên cutover là mốc cứng; thời gian song song bị giới hạn bởi mốc này.
- Vòng đời kinh phí 3 tháng và funding clock không thuộc hiến pháp (hiến pháp chỉ chứa quy định code/system/architecture); xem `docs/project/funding-and-platform-lifecycle.md`.
