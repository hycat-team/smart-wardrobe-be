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
4. **Ingress/Edge**: **VPS Nginx hiện hữu là reverse proxy/ingress duy nhất**, terminate TLS cho
   cả hai hostname. `api.closy...` → `http://backend:8080` (monolith — **giữ nguyên**).
   `api-v2.closy...` → `https://<service>.run.app` (3 Cloud Run). DNS do **Vercel** quản lý;
   **KHÔNG** NS-delegate; **KHÔNG** Cloudflare trong đường API. Route tới 3 service nằm trong
   Nginx config (single source cho `api-v2`).
5. **Song song + cutover THỦ CÔNG**: monolith trên VPS phục vụ `api.closy`; 3 service Cloud Run phục
   vụ `api-v2`, chạy **song song**. Việc đưa `api.closy...` sang Cloud Run là **thao tác thủ công**,
   do owner quyết định thời điểm dựa trên độ ổn định của `api-v2`; MUST có **runbook cutover + rollback**
   trong `docs/` (deliverable, FR-041). Trước khi cutover, `api.closy` MUST giữ nguyên routing tới monolith.
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
11. **Ingress auth (Nginx → Cloud Run)**: Nginx xác thực bằng **Google-signed ID token** đặt ở
    header `X-Serverless-Authorization` (Cloud Run IAM; **KHÔNG** bật `allUsers`). Token do
    **token helper chạy trên VPS** sinh và cache, với audience = URL `run.app` của từng service.
    App-level `Authorization: Bearer <Closy JWT>` MUST được giữ nguyên cho backend (Cloud Run
    strip header `X-Serverless-Authorization` trước khi tới container).
12. **Migration**: mỗi service sở hữu migration của mình và **tự chạy khi khởi động** (Goose).
    **Mỗi service một bảng version riêng** — Account giữ `goose_db_version` (đã có 24 lịch sử),
    Commerce/AI dùng `goose_db_version_commerce` / `goose_db_version_ai` — để **không lệch version
    giữa các service**; bật **allow-out-of-order** để chịu được migration merge muộn trong cùng
    service. **Advisory lock dùng chung** (cùng lock id) để serialize. Toàn bộ **24 migration lịch
    sử do Account giữ**; migration mới được tạo trong chính service phát sinh.
13. **Dev tooling**: Makefile MUST được thiết kế lại cho multi-module workspace — target per-service
    (build / wire / swagger / migrate / test) và target toàn workspace; MUST NOT còn phụ thuộc path
    monolith (`./cmd/server`, `./internal/di`, `./api/swagger`).
14. **SSE**: `api-v2` SSE đi qua **Nginx → Cloud Run** (không qua Cloudflare); heartbeat 15s;
    `ai_chat_chunk_deadline_seconds = 30s` thống nhất mọi môi trường (dùng chung `configs/`);
    Nginx tắt response buffering cho path SSE và đặt timeout đọc đủ lớn.
15. **Swagger**: generate per-service, hợp đủ **9 section**, expose sau `swagger_access_code`.
16. **Repatriation**: về VPS bằng 3 service compose + Nginx gateway; **teardown trial là một bước trong
    repatriation**; funding clock T-30/T-14/T-7/T-1; runbook viết song song từ bây giờ.
17. **Vòng đời kinh phí**: 4 hạng mục funded duy trì 3 tháng; chi tiết ở
    `docs/project/funding-and-platform-lifecycle.md` (không thuộc hiến pháp).

## Trạng thái tài liệu (đọc trước khi code)

- **Nguồn chân lý giai đoạn 020**: `specs/020-platform-migration/` (spec.md, plan.md, tasks.md,
  data-model.md, contracts/, quickstart.md) + `.specify/memory/constitution.md` v3.0.0 + ADR 021/022/023.
- **Baseline CŨ (không dùng cho 020)**: `docs/system-design/architecture.md`, `README.md`,
  `docs/development/local-setup.md` vẫn mô tả modular monolith (`cmd/server`, `internal/modules` ở root,
  `internal/shared`). Chúng sẽ được cập nhật ở **T109–T111**. Không lấy các file này làm baseline khi
  implement 020.
- **Thứ tự đọc đề xuất**: spec.md (mục này) → plan.md (Structure Decisions + Technical Context) →
  data-model.md → contracts/ → tasks.md → quickstart.md (verify).
- **Cấu trúc thư mục đích**: xem `plan.md` mục "Source Code (repository root — mô hình C, gom trong
  `services/`)" và "Structure Decisions". Spec này **không** liệt kê cây thư mục (giữ ở mức
  requirement, không phải implementation detail).

## Thuật ngữ

| Thuật ngữ | Nghĩa trong feature này |
|-----------|--------------------------|
| Mô hình C | Multi-module Go workspace (`go.work`), mỗi service một Go module, bỏ monolith khỏi `main` |
| `monolith-legacy` | Tag frozen tại commit monolith-green cuối cùng; chỉ tham chiếu khẩn cấp, không build CI |
| Ingress | **VPS Nginx hiện hữu** — reverse proxy/terminate TLS cho cả `api.closy` và `api-v2` |
| Cloud Run IAM | Cơ chế chặn gọi Cloud Run: request phải có Google ID token hợp lệ (`roles/run.invoker`; không `allUsers`) |
| `X-Serverless-Authorization` | Header Nginx gửi ID token; Cloud Run strip trước khi tới container → giữ nguyên `Authorization` cho app |
| Token helper | Tiến trình trên VPS sinh/cache/refresh Google ID token (audience = `run.app` URL) |
| Route map (Nginx) | Cấu hình route `api-v2` → 3 Cloud Run trong Nginx (single source); `api.closy` → monolith |
| `run.app` | Hostname mặc định do Cloud Run cấp cho mỗi service (không publish; dùng nội bộ/vận hành — **fallback debug SSE** khi cần, không phải đường người dùng) |
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

- Q: Edge đặt ở đâu khi zone gốc không được move sang Cloudflare? → A: NS-delegate subdomain `api-v2.closy.hycat.online` thành zone Cloudflare riêng (gắn Pro), Worker edge-router định tuyến tới 3 Cloud Run; cutover `api.closy...` bằng SaaS custom hostname (1 CNAME trên Vercel). *(Đợt 4 thay đổi: **bỏ Cloudflare khỏi đường API** — VPS Nginx là ingress; không NS-delegate; DNS ở Vercel.)*
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
- Q: Bảo vệ origin Cloud Run khỏi gọi thẳng `run.app`? → A: Thêm ràng buộc origin-auth: edge tiêm `X-Gateway-Secret`, service verify (trừ healthcheck hạ tầng). *(Đợt 4 thay đổi: dùng **Cloud Run IAM** + `X-Serverless-Authorization`, bỏ `X-Gateway-Secret`.)*
- Q: SSE/AI streaming đi đường nào? → A: Đi qua Cloudflare Worker (passthrough stream) cùng hostname `api-v2`; dùng Workers Free, fallback gọi trực tiếp `run.app` nếu vượt hạn mức. *(Đợt 4 thay đổi: đi qua **Nginx → Cloud Run**, không qua Cloudflare.)*
- Q: Xử lý idle timeout 100s của Cloudflare proxy? → A: Heartbeat ~15s và hạ `ai_chat_chunk_deadline_seconds` xuống 30s, **thống nhất mọi môi trường** (dùng chung `configs/`, tránh phân nhánh config theo môi trường). *(Đợt 4: giữ heartbeat 15s/30s nhưng không còn Cloudflare trong đường API.)*
- Q: Xác thực gọi gRPC giữa 3 service? → A: Cloud Run ID token (`run.invoker`) cho tầng hạ tầng + JWT bất đối xứng cho tầng ứng dụng (hai lớp độc lập).
- Q: Ai chạy Goose migration trên DB dùng chung? → A: Một migration job riêng chạy trước khi deploy service; không service nào tự migrate lúc khởi động. *(Đợt 3 thay đổi: mỗi service tự chạy migration của mình khi khởi động, dùng chung advisory lock; 24 migration lịch sử do Account giữ.)*
- Q: Tài liệu API phơi ở đâu trên managed? → A: Giữ expose sau `swagger_access_code`, hợp của 3 service phải đủ 9 section (9 module: identity, subscription, admin, notification, wardrobe, fashion, brand, dashboard, aiexecutor).

### Session 2026-09-15 (đợt 3 — migration ownership & dev tooling)

- Q: Migration cũ và migration mới thuộc về ai? → A: Toàn bộ 24 migration lịch sử đặt trong **Account service**; migration mới tạo trong chính service phát sinh.
- Q: Ai chạy migration và khi nào? → A: Chính service **tự chạy khi khởi động** (Goose provider trong service), không dùng migration job tách rời.
- Q: Đồng bộ giữa các service khi DB dùng chung? → A: Dùng **advisory lock dùng chung** (Goose session locker, cùng lock id) để serialize; **mỗi service một bảng version riêng** (Account giữ `goose_db_version`; Commerce/AI dùng bảng hậu tố) + `WithAllowOutofOrder` để tránh lệch ordering giữa các service.
- Q: Makefile sau khi tách microservice xử lý thế nào? → A: Viết lại Makefile cho multi-module — target per-service + target workspace; bỏ mọi path monolith (`./cmd/server`, `./internal/di`, `./api/swagger`).

### Session 2026-09-15 (đợt 4 — ingress qua VPS Nginx, bỏ Cloudflare khỏi đường API)

- Q: Cloudflare có đứng trước API không? → A: **Không**. `hycat.online` giữ DNS tại **Vercel**; **không NS-delegate**, **không chuyển nameserver**. Cloudflare Pro vẫn là hạng mục funded nhưng **không nằm trong đường API**.
- Q: Ingress cho 3 Cloud Run là gì? → A: **VPS Nginx hiện hữu** làm reverse proxy/ingress: thêm server block `api-v2.closy.hycat.online` proxy tới 3 Cloud Run; **giữ nguyên** block `api.closy...` → `http://backend:8080`.
- Q: Xác thực Nginx → Cloud Run? → A: **Cloud Run IAM** với Google ID token ở `X-Serverless-Authorization`; token do **token helper trên VPS** sinh/cache (audience = `run.app` URL). Không dùng `allUsers`.
- Q: DNS record cho `api-v2`? → A: Bản ghi **A (hoặc CNAME) trên Vercel** trỏ về VPS hiện tại; giữ nguyên nameserver.
- Q: TLS cho `api-v2`? → A: Nginx VPS terminate TLS; **cấp/mở rộng cert Let's Encrypt** để gồm `api-v2.closy.hycat.online`.
- Q: Có cutover `api.closy` trong giai đoạn này? → A: **Không** thực hiện tự động. Chạy song song. *(Đợt 5 làm rõ: cutover là thao tác **thủ công** có runbook; timing owner quyết định.)*

### Session 2026-09-15 (đợt 5 — cutover `api.closy` là thao tác thủ công có runbook)

- Q: Ai thực hiện cutover `api.closy` sang Cloud Run? → A: Owner thực hiện **thủ công** theo runbook; không tự động hoá, không làm trong phase deploy `api-v2`.
- Q: Khi nào cutover? → A: Tùy độ ổn định của `api-v2`, do owner quyết định; không gắn mốc cứng trong giai đoạn này.
- Q: Outcome docs cần gì? → A: MUST có runbook cutover + rollback (đổi Nginx/DNS, đồng bộ dữ liệu, kiểm thử, quay lui về monolith), và drill được trên môi trường Cloud (FR-041).

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

### User Story 3 - Đưa 3 service lên Cloud Run qua VPS Nginx, chạy song song (Priority: P1)

Đội vận hành cần tài liệu copy-paste để đưa 3 service lên Cloud Run và phục vụ qua **VPS Nginx hiện hữu** dưới `api-v2.closy.hycat.online`, **song song** với monolith đang phục vụ `api.closy...`. DNS ở Vercel; Nginx xác thực tới Cloud Run bằng Cloud Run IAM (ID token).

**Why this priority**: Đây là deployment mới (yêu cầu "chạy trên Cloud"); phải làm song song để không ảnh hưởng API cũ, và có runbook để lặp lại/khôi phục.

**Independent Test**: Có thể kiểm chứng bằng cách thực hiện đúng runbook trên môi trường Cloud và xác nhận `api-v2` hoạt động qua Nginx↔Cloud Run, trong khi `api.closy` không đổi.

**Acceptance Scenarios**:

1. **Given** hệ thống monolith đang chạy trên VPS, **When** đội vận hành làm theo runbook, **Then** `api-v2...` phục vụ 3 service Cloud Run qua VPS Nginx (DNS Vercel, TLS Let's Encrypt, Cloud Run IAM), và **`api.closy...` giữ nguyên routing tới monolith**.
2. **Given** `api-v2` đang chạy song song, **When** kiểm tra, **Then** không có request `api-v2` nào đi qua Cloudflare; request thiếu ID token bị Cloud Run từ chối.
3. **Given** Cloud Run (api-v2) gặp sự cố, **When** xảy ra lỗi, **Then** `api.closy` (monolith) **không bị ảnh hưởng**.
4. **Given** cần quay lui `api-v2`, **When** tắt/tạm dừng deployment `api-v2`, **Then** API cũ vẫn hoạt động và có thể bật lại sau.

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
- VPS là **single ingress** cho cả hai hostname → VPS/Nginx down thì `api.closy` và `api-v2` đều ảnh hưởng; cần health check + restart policy + runbook.
- Quorum queue không được hỗ trợ trên CloudAMQP shared plan → fallback classic durable + ADR bổ sung.
- SSE/AI streaming qua **Nginx → Cloud Run**: heartbeat 15s và `ai_chat_chunk_deadline_seconds` = 30s (thống nhất mọi môi trường); Nginx tắt buffering; test stream thật trước khi mở traffic.
- Token helper (sinh ID token cho Cloud Run) lỗi/hết hạn → `api-v2` bị Cloud Run trả 401/403; cần health check + retry/backoff và cảnh báo.
- Service account key của token helper bị lộ → có thể invoke Cloud Run; MUST least-privilege (`roles/run.invoker` trên đúng 3 service) + rotation + file permission 0600.
- Cert Let's Encrypt chưa gồm `api-v2.closy.hycat.online` → TLS handshake fail; MUST mở rộng cert trước khi bật DNS cho `api-v2`.
- Cloud Run URL `run.app` không cố định → Nginx MUST resolve động (`resolver`) và proxy theo hostname, không pin IP.
- Drill và môi trường Cloud dùng chung **1 database** → thao tác drill ghi nhầm dữ liệu thật (bắt buộc dùng schema tạm rồi xoá, không đụng schema chính).
- Trong giai đoạn song song, dữ liệu `api-v2` (Supabase) lệch dần so với VPS vì VPS vẫn nhận ghi của người dùng thật → phải dump lại tại cutover và xác nhận không mất ghi nào trong cửa sổ chuyển.
- Cloudflare proxy idle timeout (nếu có proxy trung gian) trong khi stream AI có khoảng lặng lâu → Error 524; mitiga bằng heartbeat 15s. Trong kiến trúc mới Cloudflare **không** nằm trong đường API.
- Yêu cầu "chạy trên Cloud": `api-v2` đã chạy trên Cloud Run; `api.closy` chỉ chuyển sang Cloud Run khi owner thực hiện **cutover thủ công** theo FR-041 (thời điểm owner quyết định).
- Nhiều instance/ba service migrate đồng thời khi khởi động trên cùng DB → advisory lock dùng chung phải serialize; **mỗi service một bảng version riêng** để không lệch ordering, và allow-out-of-order cho case migration merge muộn trong cùng service.
- Makefile cũ còn path monolith → CI/dev gãy sau khi tách; bắt buộc viết lại (FR-038).

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

- **FR-025**: Hệ thống MUST triển khai trên đúng các nền tảng funded: **VPS hiện hữu** (ingress/reverse proxy — đã có kinh phí), **Cloudflare Pro** (funded nhưng **KHÔNG** nằm trong đường API), Supabase Pro 1 project **1 database dùng chung** (Postgres + pgvector), CloudAMQP Tough Tiger (RabbitMQ), Redis Cloud Essentials 250MB (cache); compute GCP Cloud Run (trial + billing cap + teardown). MUST NOT thêm Load Balancer / API Gateway / dịch vụ ngoài danh sách chỉ để làm ingress.
- **FR-026**: Ở topology service chạy trên Cloud Run, các tiến trình nền (worker) MUST được cấu hình để chạy liên tục (không bị scale-to-zero).
- **FR-027**: Toàn bộ runbook chuyển đổi (và quay về) MUST được chạy thử thành công trên môi trường Cloud **trước** khi áp dụng cho production; vì môi trường Cloud dùng chung 1 database (không tách môi trường riêng), drill MUST thao tác trên **schema tạm** và xoá sau khi kết thúc, không đụng schema chính.

**Edge, tài liệu API và vòng đời tài trợ**

- **FR-028**: `api-v2.closy.hycat.online` MUST được phục vụ qua **VPS Nginx hiện hữu** (thêm server block/routing, MUST NOT dựng reverse proxy mới); Nginx route longest-prefix tới 3 Cloud Run. `api.closy...` MUST giữ nguyên proxy tới `http://backend:8080`. Route tới 3 service MUST nằm trong một Nginx route map (single source) và có smoke test.
- **FR-029**: DNS cho `api-v2.closy.hycat.online` MUST là bản ghi trên **Vercel DNS** trỏ về VPS hiện tại (A hoặc CNAME); MUST NOT chuyển nameserver, MUST NOT NS-delegate, MUST NOT dùng Cloudflare DNS. Cutover `api.closy...` sang Cloud Run MUST là **thao tác thủ công có runbook** (FR-041); thời điểm do owner quyết định dựa trên độ ổn định của `api-v2`.
- **FR-030**: Mỗi service MUST tự generate tài liệu API của các module nó chứa; hợp của 3 service MUST giữ đủ **9 section** (identity, subscription, admin, notification, wardrobe, fashion, brand, dashboard, aiexecutor) và MUST tiếp tục được expose sau `swagger_access_code`; CI fail nếu thiếu section (parity check).
- **FR-031**: Teardown trial GCP MUST là một bước trong repatriation (không tách rời sau demo); MUST xoá toàn bộ tài nguyên trial và có bằng chứng bill trial về 0.
- **FR-032**: Funding clock T-30/T-14/T-7/T-1 MUST được thực thi có xác nhận trước khi hết 3 tháng tài trợ; repatriation về VPS MUST hoàn tất trước ngày renew để không bị charge thêm.

**Bảo vệ origin và xác thực liên service**

- **FR-033**: Request từ Nginx tới Cloud Run MUST được xác thực bằng **Cloud Run IAM** (`roles/run.invoker` cho service account của token helper; KHÔNG bật `allUsers`). Nginx MUST gửi Google ID token ở `X-Serverless-Authorization: Bearer <token>` và MUST NOT ghi đè `Authorization` của ứng dụng. Smoke test MUST xác nhận request KHÔNG có ID token hợp lệ bị Cloud Run từ chối.
- **FR-034**: Giao tiếp gRPC giữa các service MUST xác thực hai lớp: Cloud Run ID token (`run.invoker`) ở tầng hạ tầng và JWT bất đối xứng ở tầng ứng dụng.
- **FR-035**: Cập nhật cấu trúc dữ liệu MUST do **chính service sở hữu chạy khi khởi động** (Goose) với **advisory lock dùng chung** trên database chia schema; **MUST dùng bảng version riêng cho mỗi service** (tránh lệch ordering giữa các service) và bật **allow-out-of-order** (chịu migration merge muộn trong cùng service); MUST NOT phụ thuộc migration job tách rời. Toàn bộ migration lịch sử MUST do **Account service** giữ; migration mới MUST được tạo và chạy trong chính service phát sinh.

**SSE/AI streaming sau Nginx**

- **FR-036**: SSE/AI streaming của `api-v2` MUST đi qua **Nginx → Cloud Run** với heartbeat ~15s; Nginx MUST tắt response buffering cho path SSE và đặt timeout đọc đủ lớn; `ai_chat_chunk_deadline_seconds` MUST = 30s thống nhất **mọi môi trường** (dùng chung `configs/`).
- **FR-037**: Nginx MUST proxy tới Cloud Run qua HTTPS với **SNI đúng** (`proxy_ssl_server_name on`) và resolve động hostname `run.app` (không pin IP).

**Dev tooling cho multi-module**

- **FR-038**: Bộ dev tooling (`Makefile`) MUST được thiết kế lại cho multi-module workspace: mọi target liên quan build / generate / wire / swagger / migration / test MUST chạy được **theo từng service** hoặc **toàn workspace**, và MUST NOT còn phụ thuộc path monolith (`./cmd/server`, `./internal/di`, `./api/swagger`).

**Ingress auth & TLS (Nginx → Cloud Run)**

- **FR-039**: MUST có **token helper** trên VPS sinh Google ID token cho service account (audience = URL `run.app` của từng service), cache và refresh trước khi hết hạn; credential MUST least-privilege và có phương án rotation. Workload Identity Federation chỉ xem xét nếu triển khai được **không làm tăng phức tạp quá mức** (trade-off ghi trong ADR).
- **FR-040**: Nginx MUST terminate TLS cho **cả** `api.closy...` và `api-v2.closy.hycat.online`; certificate Let's Encrypt MUST được cấp/mở rộng để gồm hostname mới và gia hạn tự động.
- **FR-041**: MUST có runbook **cutover thủ công** `api.closy...` → Cloud Run: điều kiện tiên quyết, thứ tự bước (đổi Nginx/DNS, xác minh dữ liệu), kiểm thử, và **rollback về monolith**; MUST drill được trên môi trường Cloud. Timing do owner quyết định. Cutover thủ công MUST NOT yêu cầu sửa mã nguồn nghiệp vụ.

### Key Entities *(include if feature involves data)*

- **Hồ sơ cấu hình môi trường**: tập biến môi trường phân biệt môi trường local/VPS/managed (kết nối database, cache, broker, search engine, cờ cập nhật cấu trúc, TLS).
- **Runbook chuyển đổi**: tài liệu hướng dẫn có thể thực thi, gồm các bước, lệnh kiểm thử và rollback cho từng nền tảng — viết 2 chiều (lên Cloud và repatriation về VPS).
- **Binary triển khai**: 3 binary service từ multi-module workspace (`go.work`); không còn monolith trên `main`.
- **Nginx route map (single source)**: định nghĩa route `api-v2` → 3 Cloud Run trong một chỗ; smoke test; `api.closy` giữ nguyên route tới monolith.
- **Funding clock**: deadline 3 tháng + checkpoint T-30/T-14/T-7/T-1 có xác nhận.
- **Bản sao dữ liệu (dump/backup)**: ảnh chụp dữ liệu dùng cho migrate, phục hồi và repatriation (trước cutover, trước reverse, trước hạn tài trợ).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% trường hợp tìm kiếm trang phục hệ thống trả về kết quả khi search engine bị tắt.
- **SC-002**: Ứng dụng khởi động thành công trong 100% lần chạy khi cơ chế lắng nghe sự kiện nền không khả dụng.
- **SC-003**: Có thể đưa `api-v2` lên Cloud Run qua VPS Nginx mà **không ảnh hưởng** `api.closy` (0 downtime cho API cũ).
- **SC-004**: Có thể khôi phục hệ thống về VPS (3 service + Nginx gateway) chỉ bằng thay đổi cấu hình và DNS, trong tối đa 4 giờ (bao gồm migrate ngược dữ liệu).
- **SC-005**: 0 tham số host/endpoint hạ tầng bị hardcode trong file cấu hình theo môi trường.
- **SC-006**: 100% binary (3 service, không còn monolith) build thành công trong CI; import xuyên service fail ở compile-time.
- **SC-007**: 0 sự kiện đồng bộ bị kẹt ở trạng thái thất bại do search engine bị tắt.
- **SC-008**: Toàn bộ dữ liệu production được phục hồi thành công vào VPS từ bản sao dữ liệu trước hạn tài trợ.
- **SC-009**: 100% các bước runbook chuyển đổi và quay về chạy thành công trên môi trường Cloud trước khi thực hiện trên production.
- **SC-010**: Sau teardown, 0 chi phí mới phát sinh trên trial GCP (đối chiếu billing).
- **SC-011**: Repatriation hoàn tất trước ngày renew funded items; 0 hạng mục bị charge thêm chu kỳ.
- **SC-012**: Hợp tài liệu API của 3 service giữ đủ 9 section; route `api-v2` tới đúng 3 service qua Nginx pass 100% smoke test.
- **SC-013**: 100% request tới Cloud Run thiếu ID token hợp lệ bị từ chối (Cloud Run IAM); 0 request `api-v2` bypass được IAM.
- **SC-014**: Stream AI dài giữ được kết nối qua Nginx → Cloud Run với heartbeat 15s; 0 lỗi timeout do buffering.
- **SC-015**: `api-v2` chạy trên Cloud Run qua Nginx; `api.closy` tiếp tục monolith cho tới khi owner thực hiện **cutover thủ công**; runbook cutover + rollback tồn tại và đã drill (FR-041).
- **SC-016**: 0 request của `api-v2` đi qua Cloudflare (Cloudflare nằm ngoài đường API).

## Assumptions

- Kinh phí 4 hạng mục funded duy trì 3 tháng; sau đó hệ thống repatriation về VPS (3 service compose + Nginx gateway).
- Compute Cloud Run dùng trial/free-tier + billing cap; **teardown trial là một bước trong repatriation** (không tách rời sau demo).
- `hycat.online` (gồm `api.closy...`, `api-v2.closy...`) giữ DNS tại **Vercel**; **không NS-delegate**, **không Cloudflare DNS**; Cloudflare Pro không nằm trong đường API.
- Supabase dùng 1 project Pro với **1 database dùng chung** cho môi trường Cloud; drill runbook dùng schema tạm rồi xoá.
- Giai đoạn song song: `api-v2` chỉ phục vụ test nội bộ với dữ liệu dump một lần; người dùng thật vẫn ở VPS; dump lại VPS → Supabase tại thời điểm merge/cutover.
- Tag `monolith-legacy` frozen tại commit monolith-green cuối cùng, chỉ tham chiếu khẩn cấp.
- Search engine không được cấp kinh phí và sẽ tắt mặc định; nguồn dự phòng là database.
- Docker Compose tiếp tục được dùng cho môi trường local và cho VPS sau repatriation.
- Việc sao lưu dữ liệu thực hiện: trước cutover, trước reverse cutover, trước hạn tài trợ, và theo funding clock T-minus (thay thế chiến lược "chỉ dump trước hạn" cũ).
- Thời điểm repatriation theo funding clock, phải hoàn tất trước ngày renew.
- Các cửa sổ bảo trì cho phép dừng hệ thống trong thời gian ngắn để migrate dữ liệu và chuyển DNS.
- SSE/AI streaming đi qua **Nginx → Cloud Run**; Nginx tắt buffering; heartbeat ~15s; `ai_chat_chunk_deadline_seconds` = 30s thống nhất mọi môi trường (dùng chung `configs/`).
- Chạy **song song**: monolith ở `api.closy`, 3 service ở `api-v2`. Cutover `api.closy` là **thao tác thủ công** có runbook (FR-041), timing do owner quyết định khi `api-v2` đủ ổn định.
- Cutover thủ công chỉ đổi Nginx/DNS + đồng bộ dữ liệu; MUST NOT sửa mã nguồn nghiệp vụ.
- **VPS là single ingress** (đã có kinh phí, không phải constraint chi phí); Nginx hiện hữu được giữ, chỉ bổ sung server block cho `api-v2`.
- **Token helper** chạy trên VPS dùng **service account key** (trade-off với WIF ghi ở ADR 023) để sinh ID token cho Cloud Run.
- Vòng đời kinh phí 3 tháng và funding clock không thuộc hiến pháp (hiến pháp chỉ chứa quy định code/system/architecture); xem `docs/project/funding-and-platform-lifecycle.md`.
