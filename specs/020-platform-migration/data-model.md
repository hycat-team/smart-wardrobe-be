# Phase 1 Data Model: Platform Migration & 3 Services (multi-module workspace)

**Feature**: `020-platform-migration` | **Date**: 2026-09-14

Feature này không thêm bảng nghiệp vụ mới. "Data model" dưới đây mô tả **mô hình cấu hình, mô
hình service và mô hình triển khai** — là những thực thể mà plan/tasks sẽ thao tác.

---

## 1. Cấu hình môi trường (Environment Configuration)

Bổ sung/đổi trong `internal/config/config.go` (mapstructure key trong ngoặc). Giá trị lấy từ YAML
+ `os.ExpandEnv`, có override boolean qua `os.LookupEnv` khi cần.

### 1.1 Elasticsearch

| Field | Key | Kiểu | Default | Ý nghĩa |
|-------|-----|------|---------|---------|
| Enabled | `elasticsearch.enabled` | bool | `false` | Bật/tắt search engine (FR-001) |
| Addresses | `elasticsearch.addresses` | []string | `[]` | Endpoint; để trống khi tắt |
| User/Password | `elasticsearch.user/password` | string | `""` | Auth tùy chọn |

**Quy tắc**: `Enabled=false` ⇒ không start health goroutine; `Index/Delete` no-op success;
`Search` trả lỗi để fallback DB.

### 1.2 Database

| Field | Key | Kiểu | Default | Ý nghĩa |
|-------|-----|------|---------|---------|
| Host/Port/User/Password/DbName/SslMode | `database.*` | — | hiện có | Runtime (pooler transaction mode) |
| MaxIdleConns/MaxOpenConns/ConnMaxLifetime | `database.*` | — | hiện có | Pool |
| SearchPath | `database.search_path` | string | default 8 schema | Giữ nguyên |
| **MigrateOnStart** | `database.migrate_on_start` | bool | `true` | Per-service; bật ở managed để service tự chạy migration của mình (FR-035) |
| **ListenerHost** | `database.listener_host` | string | = Host | Kênh session cho `LISTEN` |
| **ListenerPort** | `database.listener_port` | int | = Port | Supabase session `5432` |
| **ListenerSslMode** | `database.listener_ssl_mode` | string | = SslMode | TLS cho listener |

### 1.3 Redis

| Field | Key | Kiểu | Default | Ý nghĩa |
|-------|-----|------|---------|---------|
| Host/Port/Password/Db | `redis.*` | — | hiện có | Cache/blacklist/quota lock |
| **TLSEnabled** | `redis.tls_enabled` | bool | `false` | Redis Cloud yêu cầu `true` (FR-004) |

### 1.4 RabbitMQ

| Field | Key | Kiểu | Default | Ý nghĩa |
|-------|-----|------|---------|---------|
| Host/Port/User/Password/Prefetch/DeliveryLimit/ManagementPort | `rabbitmq.*` | — | hiện có | Broker |
| **TLSEnabled** | `rabbitmq.tls_enabled` | bool | `false` | CloudAMQP `amqps` 5671 (FR-004) |
| **ManagementTLSEnabled** | `rabbitmq.management_tls_enabled` | bool | = TLSEnabled | Gauge API HTTPS |

### 1.5 Outbox

| Field | Key | Kiểu | Default | Ý nghĩa |
|-------|-----|------|---------|---------|
| ... | `outbox.*` | — | hiện có | Retry/cleanup/batch |
| **ListenerEnabled** | `outbox.listener_enabled` | bool | `true` | Tắt để chỉ poll (FR-009/FR-010) |

### 1.6 JWT

| Field | Key | Kiểu | Default | Ý nghĩa |
|-------|-----|------|---------|---------|
| Secret | `jwt.secret` | string | bắt buộc (legacy HS256) | Giai đoạn chuyển tiếp |
| **Algorithm** | `jwt.algorithm` | string | `HS256` → `ES256` | Thuật toán ký (FR-022) |
| **PrivateKey** | `jwt.private_key` | string (PEM) | `""` | Chỉ identity service |
| **PublicKey** | `jwt.public_key` | string (PEM) | `""` | Các service còn lại verify |

**Validation**: nếu `Algorithm` bất đối xứng thì `PrivateKey` (identity) hoặc `PublicKey`
(service khác) MUST có; `migrate_on_start=true` yêu cầu pool hợp lệ.

### 1.7 Hồ sơ môi trường (Environment Profile)

| Profile | DB | Cache/Broker | Search | Compute | Edge | Ghi chú |
|---------|----|--------------|--------|---------|------|---------|
| `local` | docker postgres | docker, no TLS | tắt | docker compose (3 service) | trực tiếp | `deployments/docker-compose.yml` |
| `vps-legacy` | docker postgres (self-host) | docker, no TLS | tắt | `monolith-legacy` tag (frozen) | nginx + certbot | song song giai đoạn đầu, sau đó stopped |
| `vps` (repatriation) | docker postgres + pgvector | docker, no TLS | tắt | 3 service compose | nginx gateway + certbot | đích sau 3 tháng |
| `managed` | Supabase **1 database dùng chung** | Redis Cloud TLS + CloudAMQP amqps | tắt | Cloud Run trial + caps | VPS Nginx: `api.closy`→monolith, `api-v2`→Cloud Run | gồm cả drill (dùng schema tạm) |
| `managed-drill` | Supabase (cùng DB, **schema tạm**) | Redis Cloud TLS + CloudAMQP amqps | tắt | Cloud Run trial | VPS Nginx (`api-v2`) | chỉ khi chạy drill runbook; xoá schema sau khi xong |

**Ghi chú**: chỉ có **một** môi trường Cloud dùng chung 1 database; `api-v2` và monolith (`api.closy`)
cùng trỏ database đó khi cần. Drill không tách database mà dùng schema tạm. **Ingress là VPS Nginx**
cho cả hai hostname; Cloudflare **không** nằm trong đường API.

### 1.8 Ingress auth, Cloud Run và migration

| Field | Key | Kiểu | Default | Ý nghĩa |
|-------|-----|------|---------|---------|
| AccountBaseURL | `ingress.account_base_url` | string (https `run.app`) | `""` | Nginx → Account Cloud Run |
| CommerceBaseURL | `ingress.commerce_base_url` | string | `""` | Nginx → Commerce Cloud Run |
| AIBaseURL | `ingress.ai_base_url` | string | `""` | Nginx → AI Cloud Run |
| TokenHelperURL | `ingress.token_helper_url` | string (localhost) | `http://127.0.0.1:9000` | Helper sinh/cache ID token dùng bởi `auth_request` |
| TokenAudience | `ingress.token_audience` | string | = service `run.app` URL | Audience của Google ID token (mặc định là run.app URL; custom audience nếu cần) |
| AuthHeader | `ingress.auth_header` | string | `X-Serverless-Authorization` | Header Nginx inject ID token; giữ nguyên `Authorization` cho app |
| DrillSchema | `database.drill_schema` | string | `""` | Schema tạm cho drill runbook; xoá sau khi xong |
| MigrateOnStart | `database.migrate_on_start` | bool | `true` ở managed | Service tự chạy migration của mình khi boot (FR-035) |
| MigrationTable | `database.migration_table` | string | `goose_db_version` (Account) / `goose_db_version_commerce` / `goose_db_version_ai` | **Bảng version riêng mỗi service** — tránh lệch ordering giữa các service |
| AllowOutOfOrder | `database.migration_allow_out_of_order` | bool | `true` | Cho phép migration merge muộn trong cùng service (goose `WithAllowOutofOrder`) |
| MigrationLock | `database.migration_lock_id` | int64 | dùng chung, cố định | Advisory lock serialize migration giữa các instance/service |
| SSEHeartbeatSeconds | `server.sse.heartbeat_seconds` | int | `15` | Giữ stream sống qua Nginx (và dưới mọi idle timeout trung gian) |
| AIChunkDeadline | `server.sse.ai_chat_chunk_deadline_seconds` | int | `30` | Thống nhất mọi môi trường (dùng chung `configs/`) |

---

## 2. Mô hình Service (Service Composition — mô hình C, multi-module workspace)

Ánh xạ module → service module trên `go.work` (không di chuyển logic nghiệp vụ).
Cột "Host gRPC" = service mà binary đó **phơi** ra; "Gọi ra ngoài" = contract của service
khác mà binary **tiêu thụ**. Monolith không còn tồn tại trên `main` (tag `monolith-legacy`
frozen, xem §2.1).

| Binary (module) | Modules in-process | Host gRPC (server) | Gọi ra ngoài (gRPC client) | Workers |
|--------|--------------------|--------------------|----------------------------|---------|
| `services/account/cmd/account` | identity, subscription, admin, notification | Identity, Subscription, Admin | **không** | renewal, payment reconcile, webhook inbox, email, audit log/cleanup, outbox |
| `services/commerce/cmd/commerce` | wardrobe, fashion, brand, dashboard | không | Identity, Subscription, Admin, AIOrchestrator, + 3 read (Identity/Subscription/AI Dashboard) | ES sync (no-op), fashion init/analyze/cleanup, wardrobe cleanup/bind, dashboard aggregator, loyalty expiry, outbox |
| `services/ai/cmd/ai` | aiexecutor | AIOrchestrator, AIDashboard | Identity, Subscription, Admin | AI usage reconcile, plan-changed, outbox |

**Đồ thị gọi liên service (DAG, không cycle)**:

```
Commerce ──► Account
Commerce ──► AI
AI       ──► Account
Account  ──► (không gọi service nào)
```

**Ràng buộc**: module trong cùng service gọi nhau in-process qua contract; khác service dùng
adapter gRPC. Không module nào truy cập entity/repository của module khác (§II).

### 2.1 Ranh giới service (compiler enforce — thay service manifest)

Mô hình C multi-module + `go.work`: service module **không thấy** implementation package của
module thuộc service khác ở compile-time. Không cần manifest `service.yaml`, không cần
`make service-boundary-check` (research R2.1 superseded).

| Service module | Modules sở hữu |
|----------------|----------------|
| `services/account/` | identity, subscription, admin, notification |
| `services/commerce/` | wardrobe, fashion, brand, dashboard |
| `services/ai/` | aiexecutor |

**Quy tắc import** (khớp Constitution §I/§II):
- Trong service module: import tự do các package của module mình sở hữu.
- Xuyên service: CHỈ được import `contracts/<owner>/v1/client` (gRPC client) và DTO/constants
  hợp đồng; cấm `domain/entities`, `domain/repositories`, `infrastructure/**`,
  `application/usecase/**` của module thuộc service khác (compiler chặn).
- Tag `monolith-legacy` frozen: tham chiếu khẩn cấp, không nhận code mới, không build trong CI.

### 2.2 Vị trí adapter (theo hướng phụ thuộc — Ports & Adapters)

| Adapter | Hướng | Vị trí | Ghi chú |
|---------|-------|--------|---------|
| gRPC server | inbound (driving) | `services/<service>/internal/modules/<owner>/presentation/grpc/server.go` | entry point; phụ thuộc **contract interface**, Wire bind interface → impl |
| gRPC client | outbound (driven) | `contracts/<owner>/v1/client/` | **implements contract interface** của owner, gọi qua mạng |
| In-process impl | local | `services/<service>/internal/modules/<owner>/presentation/contract/<owner>_contract_impl.go` | implements contract interface, gọi usecase in-process |

Lý do server **không** inject usecase trực tiếp: `presentation/contract` là **port công khai** của
module. Cả in-process impl lẫn gRPC server đều bind cùng contract interface → local caller và remote
caller nhận **cùng một API, cùng hành vi**; mapping DTO ⇄ proto tái dùng đúng DTO hợp đồng thuần
(primitive + UUID), không lộ usecase DTO/entity. Contract interface là điểm phụ thuộc duy nhất của
server; impl do Wire quyết định (kể cả khi impl là usecase tự implement, như dashboard reader).

Lý do vị trí: server là **inbound adapter** nên thuộc `presentation/`; client là **outbound adapter**
(gọi service khác) nên để ở module `contracts/` (`contracts/<owner>/v1/client/`) — dùng chung vì một
contract có nhiều consumer (identity client dùng bởi 5 module) và **không** thể đặt trong `shared/`
(shared không được import module service).

### 2.3 Adapter Contract

| Aspect | in-process | gRPC |
|--------|-----------|------|
| Đăng ký | Wire cung cấp impl hiện tại | Wire cung cấp client adapter |
| DTO | `application/dto/*ContractDTO` | map sang proto message |
| Lỗi | `apperror` | map `apperror` ⇄ `google.rpc.Status` |
| Context | `context.Context` | metadata: `x-correlation-id`, token |
| Timeout | theo usecase | deadline ngắn + retry idempotent |

---

## 3. Danh mục Contract: cái nào lên protobuf

Nguyên tắc: **chỉ contract bị module ở service KHÁC gọi mới lên protobuf**. Contract mà mọi caller
nằm cùng service giữ in-process (vẫn giữ nguyên interface, không cần proto).

### 3.1 Cần protobuf (vượt ranh giới service)

| Contract (interface) | Owner | Service phơi | Caller khác service | gRPC service |
|----------------------|-------|--------------|---------------------|--------------|
| `IUserBriefContract` | identity | Account | Commerce, AI | `IdentityService` |
| `IUserDashboardReader` | identity | Account | Commerce (dashboard) | `IdentityDashboardService` |
| `IUserSubscriptionContract` | subscription | Account | Commerce, AI | `SubscriptionService` |
| `IUserQuotaContract` | subscription | Account | Commerce | `SubscriptionService` |
| `ISubscriptionDashboardReader` | subscription | Account | Commerce (dashboard) | `SubscriptionDashboardService` |
| `IAuditLogContract` | admin | Account | Commerce, AI | `AdminService` |
| `IAIOrchestrator` | aiexecutor | AI | Commerce | `AIOrchestratorService` |
| `IAICostDashboardReader` | aiexecutor | AI | Commerce (dashboard) | `AIDashboardService` |

### 3.2 Giữ in-process (mọi caller cùng service)

| Contract (interface) | Owner | Service | Caller |
|----------------------|-------|---------|--------|
| `IBrandContract` | brand | Commerce | fashion, wardrobe, dashboard |
| `IFashionContract` | fashion | Commerce | brand, wardrobe, dashboard |
| `ICategoryContract` | fashion | Commerce | wardrobe |
| `IWardrobeContract` | wardrobe | Commerce | fashion, dashboard |
| `IBrandDashboardReader` | brand | Commerce | dashboard |
| `IFashionDashboardReader` | fashion | Commerce | dashboard |
| `IWardrobeDashboardReader` | wardrobe | Commerce | dashboard |
| `INotificationContract` | notification | Account | identity, subscription |

**Nhận xét**: Commerce **không phơi** gRPC nào (mọi contract nội bộ Commerce); Account không gọi ra
ngoài; AI gọi Account; Commerce gọi Account + AI.

**Quy tắc map**: chỉ primitive + UUID + enum string; DTO thuần hiện có map 1-1 (đã xác nhận trong
`contract_dto.go`). Không đưa entity. Auth giữa service dùng JWT verify cục bộ (public key), không
phải RPC.

---

## 4. Mô hình triển khai (Deployment Topology)

### 4.1 Managed (VPS Nginx ingress + Cloud Run, 3 tháng funded + trial compute)

| Unit | Loại | Scale | CPU | Ghi chú |
|------|------|-------|-----|---------|
| account-http | HTTP (`APP_MODE=http`) | scale-to-zero, `max-instances=2` | theo request, `concurrency=40` | API qua VPS Nginx (IAM ID token) |
| account-worker | worker (`APP_MODE=worker`) | `min=1`, `max=1` (singleton) | always | consumer/outbox/cron |
| commerce-http | HTTP (`APP_MODE=http`) | scale-to-zero, `max-instances=2` | theo request, `concurrency=40` | |
| commerce-worker | worker (`APP_MODE=worker`) | `min=1`, `max=1` (singleton) | always | ES no-op, fashion |
| ai-http | HTTP (`APP_MODE=http`) | scale-to-zero, `max-instances=2` | theo request, `concurrency=40` | SSE stream (Nginx buffering off) |
| ai-worker | worker (`APP_MODE=worker`) | `min=1`, `max=1` (singleton) | always | AI reconcile |

**Cap & budget (FR-045/FR-047)**: HTTP `max-instances=2` + `concurrency=40`; worker `min=max=1` (singleton,
advisory lock cho job không idempotent); `max_open_conns=10`/instance; tổng client conn ≤ Supabase pooler
(đo thực, headroom ≥20%), AMQP ≤100, Redis ≤256.

Biên: **VPS Nginx hiện hữu** terminate TLS cho `api.closy` (→ monolith `http://backend:8080`) và
`api-v2` (→ 3 Cloud Run `run.app`); route map trong Nginx; xác thực Cloud Run IAM qua
`X-Serverless-Authorization` (token helper trên VPS). DNS ở **Vercel**. Cloudflare **không** nằm trong
đường API. Trial GCP: billing cap + teardown (US6).

### 4.2 Self-hosted VPS

| Giai đoạn | Unit | Ghi chú |
|-----------|------|---------|
| Song song (đầu 020) | `monolith-legacy` container + postgres/redis/rabbitmq + nginx + certbot | VPS hiện tại giữ nguyên phục vụ `api.closy`, sau đó stopped |
| Repatriation (sau 3 tháng) | 3 service containers + postgres+pgvector/redis/rabbitmq + **nginx gateway** (route từ single-source) + certbot | `deployments/docker-compose.prod.yml` (viết mới); runtime chính, không phải monolith |

### 4.3 Trạng thái chuyển đổi (State Transitions)

```
[VPS Legacy Active] --cutover--> [Managed Active] --teardown trial--> [Managed funded-only]
      | stopped fallback                |
      +-- reverse cutover (lỗi)         +-- repatriation (hết 3 tháng) --> [VPS 3-service Active]
```

| Trạng thái | DNS | Compute | DB |
|-----------|-----|---------|-----|
| VPS Legacy Active | `api.closy` → VPS | monolith-legacy | docker postgres |
| Song song | `api-v2` → VPS (A/CNAME Vercel) | monolith + 3 Cloud Run | VPS DB; Supabase cho `api-v2` |
| Managed Active | `api-v2` → VPS Nginx → Cloud Run | 3 service | Supabase (+ Tiger/Redis Cloud) |
| Teardown trial | giữ nguyên | xóa resource trial | giữ funded |
| Repatriation | → VPS | 3 service compose | migrate ngược về docker postgres |
| VPS 3-service Active | → VPS | 3 service + Nginx | docker postgres |

---

## 5. Hiện vật tài liệu (Documentation Artifacts)

| Hiện vật | Đường dẫn | Nội dung |
|----------|-----------|----------|
| Tổng quan hạ tầng | `docs/operations/ha-tang/00-tong-quan.md` | ma trận thay thế, hybrid, cảnh báo constitution |
| Runbook nền tảng | `docs/operations/ha-tang/01..06` | Supabase, Redis Cloud, CloudAMQP, Cloud Run, **VPS Nginx ingress + token helper**, ES-off |
| Bảng biến môi trường | `docs/operations/ha-tang/07-bang-bien-moi-truong.md` | mapping cũ → mới, gồm migrate session-DSN |
| Song song + Cloud Run | `docs/operations/ha-tang/08..10` | tổng quan, dữ liệu VPS→Supabase, DNS Vercel + CI/CD |
| Repatriation | `docs/operations/ha-tang/11-ve-lai-vps.md` | verify/rollback, migrate ngược, funding clock, tắt funded, DNS về VPS |
| Vận hành — ngừng VPS & thu hồi secret | `docs/operations/ha-tang/12-ngung-vps-va-thu-hoi-secret.md` | thu hồi credential cũ + ngừng VPS sau khi xác nhận (FR-016) |
| Quan trắc | `docs/operations/ha-tang/13-monitoring.md` | metrics/alert Cloud Monitoring + runbook vận hành (FR-057, T120) |
| Ingress Nginx + auth | `docs/operations/nginx-ingress-cloudrun-auth.md` | route map single-source, token helper, rotate SA key, smoke test IAM |
| Container & deploy độc lập | `docs/operations/container-images-and-independent-deploy.md` | Dockerfile multi-stage per-service, `.dockerignore`/context root, Artifact Registry + tag SHA, deploy độc lập từng service, image rollback `monolith-legacy` (FR-042/FR-043) |
| Cutover thủ công | `docs/operations/manual-cutover-api-closy-to-cloudrun.md` | runbook cutover `api.closy` → Cloud Run + rollback (FR-041), owner-triggered |
| Teardown | `docs/operations/gcp-trial-teardown.md` | xóa resource trial + verify bill |
| Repatriation | `docs/operations/repatriate-to-vps.md` | runbook về VPS 2 chiều đầy đủ (viết song song từ bây giờ) |
| Services guides | `docs/system-design/microservices/00..06` | ranh giới, gRPC/event, adapter, JWT, repo/CI, lộ trình (mô hình C, không monolith) |
| ADR | `docs/decisions/021-...` + `docs/decisions/023-...` | funded platforms + repatriation; **ingress VPS Nginx + Cloud Run IAM** (supersede phần Cloudflare của ADR 021) |

---

## 6. Không thay đổi (Out of scope)

- Schema nghiệp vụ (45 bảng, 8 schema) — giữ nguyên, migration hiện có.
- API HTTP công khai — giữ nguyên đường dẫn/hợp đồng (`/api/v1/...`).
- Luồng nghiệp vụ AI/thanh toán/outbox — giữ nguyên logic.
- Không GCP LB/Armor/Cloud SQL/Memorystore; không Firebase; không Nginx-on-Cloud-Run; **không Cloudflare/Cloudflare Workers trong đường API**.
