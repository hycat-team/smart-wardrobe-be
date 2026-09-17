# Phase 0 Research: Platform Migration & 3 Services (multi-module workspace)

**Feature**: `020-platform-migration` | **Date**: 2026-09-14

Tài liệu này chốt các quyết định kỹ thuật còn để mở trong `Technical Context` và trong quá trình
lập plan. Mọi mục dùng định dạng Decision / Rationale / Alternatives.

> **Ghi chú cập nhật 2026-09-15 (hiến pháp 3.0.0 + ADR 021 + ADR 023)**: multi-module + `go.work`
> thay thế single-module; monolith bị loại khỏi `main` (tag `monolith-legacy`); **ingress là VPS Nginx
> hiện hữu — không Cloudflare/Worker trong đường API** (R14/R15 superseded bởi R22); DNS ở Vercel;
> compute trial + teardown; Supabase 1 project **1 database dùng chung**; repatriation bắt buộc sau 3
> tháng.
> Các mục R2.1, R9, R12, R14, R15, R20, R21 dưới đây được giữ để truy vết nhưng đã ghi chú supersede.

---

## R1. Giao tiếp liên service: gRPC + buf

**Decision**: Dùng gRPC (`google.golang.org/grpc`) với protobuf làm hợp đồng liên service. Sinh mã
bằng `buf` (`protoc-gen-go`, `protoc-gen-go-grpc`); nguồn chân lý đặt tại `contracts/<owner>/v1/`.

**Rationale**: Hợp đồng cần kiểu mạnh, ít lỗi khi map từ DTO thuần (primitive + UUID) hiện có; hỗ
trợ batch (`GetBatchUserBriefs`) để giảm round-trip; chạy tốt trên Cloud Run (HTTP/2).

**Alternatives considered**:
- REST/JSON nội bộ: đơn giản hơn nhưng thiếu kiểu, dễ lệch contract, nhiều round-trip.
- Không gRPC, dùng chung DB cho mọi truy vấn chéo: vi phạm ranh giới module (§II/§IV).

**Ghi chú**: `buf` cần cài trong CI và local (`make tool`); generated code commit vào repo để
tránh phụ thuộc tool khi build runtime.

---

## R2. Dual adapter cho contract (in-process ⇄ gRPC)

> **Cập nhật 2026-09-15 (mô hình C)**: mọi tham chiếu `cmd/server`/monolith trong mục này là lịch
> sử (truy vết). Hiện tại: in-process = module nội bộ **cùng service**; gRPC = module **khác
> service**. Không còn adapter monolith.

**Decision**: Giữ nguyên interface trong `presentation/contract/`. Đặt adapter theo hướng phụ thuộc
(Ports & Adapters — inbound/driving vs outbound/driven), **không tạo cây `internal/interfaces/` song song**:
- **in-process** (local): `services/<service>/internal/modules/<owner>/presentation/contract/<owner>_contract_impl.go`, implements contract interface.
- **gRPC server** (inbound/driving adapter, entry point): `services/<service>/internal/modules/<owner>/presentation/grpc/server.go` của module sở
  hữu. Server phụ thuộc **contract interface** (KHÔNG inject usecase trực tiếp); Wire bind interface →
  impl. Nhờ vậy in-process và gRPC phục vụ **cùng một API, cùng hành vi**, tái dùng mapping DTO hợp
  đồng thuần (primitive + UUID) thay vì map qua usecase DTO.
- **gRPC client** (outbound/driven adapter, gọi service khác): `contracts/<owner>/v1/client/client.go`,
  implements contract interface của owner qua mạng. Đặt ở module `contracts/` **dùng chung** vì một
  contract có nhiều consumer (identity client dùng bởi 5 module); **không** đặt trong `shared/`
  vì shared không được import module service.

Composition root (Wire) chọn adapter theo service: in-process cho module nội bộ service, client ở
`contracts/<owner>/v1/client` cho module ngoài service. Business logic không đổi.

**Rationale**: Là điều kiện để chuyển topology không sửa logic (§II, FR-019); cả 3 service module
build độc lập trên `go.work` (FR-018/FR-020).

**Alternatives considered**:
- Tách module vật lý rồi chỉ dùng gRPC: phá monolith, mất đường quay về VPS.
- Build tag để chọn impl: khó test, dễ lệch; Wire chọn lúc khởi tạo rõ ràng hơn.

**Điểm cần chú ý**: adapter gRPC phải map `apperror` ⇄ gRPC status; timeout/retry; correlation ID
truyền qua metadata để giữ tracing (liên quan worker/outbox).

### R2.1 Service import-boundary enforcement (bù cho việc dùng 1 Go module)

> **SUPERSEDED 2026-09-15 bởi mô hình C**: multi-module + `go.work` khiến import implementation
> xuyên service fail ở compile-time. Không cần manifest `service.yaml` và không cần
> `make service-boundary-check`. Nội dung dưới giữ lại để truy vết.

**Decision**: Vì giữ 1 `go.mod`, Go compiler không chặn `cmd/<service>` import module ngoài tập cho
phép. Bù lại bằng cơ chế **service manifest + CI import-boundary check**:
- Mỗi service có manifest `cmd/<service>/service.yaml` liệt kê chính xác các module được phép
  (ví dụ `account: [identity, subscription, admin, notification]`).
- Một check trong CI (`make service-boundary-check`) chạy `go list -deps ./cmd/<service>` và đối
  chiếu với manifest; **fail build** nếu xuất hiện package thuộc module ngoài danh sách (trừ
  `internal/shared`, `internal/pkg`, `internal/config`, `internal/di`, generated proto).
- `cmd/server` được miễn check (monolith được phép dùng mọi module).

**Rationale**: Mang lại hiệu lực gần như compile-time cho ranh giới service mà không cần tách
nhiều `go.mod`; giữ được monolith và đường rollback VPS. Ngăn import chéo service do vô ý.

**Alternatives considered**:
- Tách nhiều `go.mod` + `go.work`: Go chặn ở compiler nhưng phá yêu cầu monolith 1 binary, thêm
  `replace`/versioning ceremony.
- Chỉ dựa vào code review: không đủ chắc (đúng tinh thần "mọi thứ phải chắc chắn").

**Phạm vi cho phép (allowlist chung)**: mọi binary được phép import `internal/shared/*`,
`shared/**`, `contracts/**` (generated).
Với module ngoài manifest: cho phép `presentation/contract/**`, `application/dto`,
`domain/constants` (bề mặt hợp đồng theo §II); chặn `domain/entities`, `domain/repositories`,
`infrastructure/**`, `application/usecase/**`, `application/interface/**`. Nhờ vậy client adapter
(chỉ import contract+dto) không kéo implementation của service khác vào binary.

---

## R3. Supabase Postgres: pooler mode và `LISTEN`

**Decision**:
- Runtime (GORM/pgx): dùng **transaction pooler** (port `6543`) để giới hạn số connection.
- Migration (Goose) và `OutboxListener` (`LISTEN/NOTIFY`): dùng **session/direct connection**
  (port `5432`) qua cấu hình riêng.
- Đặt `sslmode=require` cho Supabase; VPS giữ `disable`.

**Rationale**: Transaction pooler không giữ session-level advisory lock (Goose) và không hỗ trợ
`LISTEN`. Tách DSN là cách duy nhất ổn định và vẫn giữ runtime gọn.

**Alternatives considered**:
- Toàn bộ dùng session mode: đơn giản nhưng tốn connection, phí pooler.
- Bỏ `LISTEN`, chỉ poll: chấp nhận được nhưng tăng độ trễ phát sự kiện; vẫn dùng làm fallback.

---

## R4. Outbox listener resilient (non-fatal + poll fallback)

**Decision**: `NewOutboxListener` không còn fatal khi không kết nối/`LISTEN` thất bại. Khi disabled
hoặc lỗi → trả **no-op listener** (`SignalChan()` là nil channel) và log cảnh báo; outbox worker
tiếp tục chạy bằng **poll ticker** (`outboxFallbackInterval` hiện có). Thêm cờ
`outbox.listener_enabled` và DSN listener riêng.

**Rationale**: FR-009/FR-010; app phải boot được trên pooler. Poll fallback đã tồn tại trong
`outbox_worker.go`, chỉ cần gỡ chặn ở constructor.

**Alternatives considered**:
- Giữ fatal và yêu cầu session mode: không đáp ứng US2.
- Tự viết cơ chế backoff mới: không cần, đã có fallback.

---

## R5. Migration-on-start có kiểm soát

**Decision**: Mỗi service **tự chạy migration của mình khi khởi động** (Goose provider trên
`services/<service>/database/migrations`), dùng **advisory lock dùng chung** (Goose session locker,
cùng lock id) để serialize giữa các instance/service. **Mỗi service một bảng version riêng**
(Account giữ `goose_db_version` — đã giữ toàn bộ migration lịch sử; Commerce/AI dùng bảng có hậu tố) + bật
`goose.WithAllowOutofOrder(true)`. Cờ `database.migrate_on_start` bật ở managed.

**Rationale**: FR-011/FR-012/FR-035; mỗi service sở hữu schema của mình. Bảng version dùng chung gây
**lệch ordering**: service B apply version cao trước, service A apply version thấp sau → goose mặc
định báo lỗi out-of-order. Tách bảng version theo service biến vấn đề này thành không tồn tại;
`WithAllowOutofOrder` xử lý case còn lại trong cùng một service (2 PR merge lệch thứ tự).

**Alternatives considered**:
- 1 bảng version dùng chung (bug đã phát hiện): gây lỗi out-of-order khi 2 service sinh migration
  lệch timestamp → loại.
- Dùng offset block cho version (account=1xxx, commerce=2xxx): vẫn so sánh toàn cục nên vẫn lỗi →
  loại.
- Migration job tách rời (Cloud Run Job) chạy trước deploy: đã bỏ — thêm hạ tầng và tách quyền sở
  hữu migration khỏi service.
- Chạy qua `make migration-up` thủ công: giữ làm đường vận hành/khẩn cấp, không phải đường chính.

---

## R6. Elasticsearch optional (mặc định tắt)

**Decision**: Thêm `Elasticsearch.Enabled` (default `false`). Khi tắt: không start health check;
`IndexDocument`/`DeleteDocument` → no-op `nil`; `Search` → trả lỗi để kích hoạt fallback DB sẵn có;
bỏ qua initial-sync. Không thay đổi `search_sync` usecase vì no-op trả success (không DLQ).

**Rationale**: FR-001/FR-002/FR-003, SC-001/SC-007; code đã có circuit breaker + fallback DB
(`item_uc_read.go` bọc timeout 300ms).

**Alternatives considered**:
- Ngừng consume queue ES hoàn toàn: event sẽ tích tụ; no-op + ack sạch hơn.
- Xoá ES khỏi codebase: mất khả năng bật lại, không cần thiết.

---

## R7. TLS cho Redis và RabbitMQ

**Decision**: Thêm cờ `redis.tls_enabled` và `rabbitmq.tls_enabled` (default `false`).
- Redis: cấu hình `redis.Options.TLSConfig` khi bật.
- RabbitMQ: dùng `amqps://` + `amqp.DialConfig` với `TLSClientConfig` khi bật; quản trị qua HTTPS.

**Rationale**: FR-004; Redis Cloud và CloudAMQP yêu cầu TLS; VPS không TLS + mặc định false để
tương thích ngược (FR-008).

**Alternatives considered**:
- Hardcode TLS theo môi trường: vi phạm portability.
- Dùng URL string đơn thuần: khó cấu hình CA/insecure-skip; cờ rõ ràng hơn.

---

## R8. CloudAMQP (Tough Tiger) tương thích topology hiện có

**Decision**: Tạo vhost + user riêng; chỉ bật plugin cần thiết; topology (quorum queue + DLX +
delivery-limit) do code tự khai báo (`topology.go`). Consumer/worker chạy ở Cloud Run worker
(min-instances=1 **và `max-instances=1`** — singleton theo FR-045; advisory lock cho job không idempotent).

**Rationale**: FR-025/FR-026; giữ `x-queue-type: quorum` và DLX tương thích; tránh cấu hình tay
lệch với code. Dùng management API (HTTPS) cho gauge queue depth.

**Alternatives considered**:
- Cấu hình queue thủ công trên CloudAMQP: dễ lệch với `topology.go`.
- Classic queue: mất tính bền khi node restart.

---

## R9. Cloud Run: tách HTTP và Worker

> **Giữ nguyên 2026-09-15**, bổ sung: region `asia-southeast1`, request-based billing,
> billing cap cứng; teardown trial là một bước trong repatriation (compute không funded).

**Decision**: Mỗi service có 2 deployment Cloud Run: một HTTP (scale-to-zero, `max-instances=2`) và một worker
(`min-instances=1` + `max-instances=1`, CPU always allocated — FR-045/FR-047). HTTP service phục vụ API; worker chạy consumer/outbox/
scheduler. VPS legacy chạy monolith frozen; VPS repatriation chạy 3 service compose (mỗi service
gộp worker của nó).

**Rationale**: FR-026; scale-to-zero sẽ ngắt consumer RabbitMQ; tách deployment giữ nguyên code,
chỉ khác entrypoint/config.

**Alternatives considered**:
- Chạy worker trong HTTP instance: bị throttle khi không có request.
- Worker ngoài danh sách funded: vi phạm FR-025.

---

## R10. JWT bất đối xứng

**Decision**: Chuyển từ HS256 sang thuật toán bất đối xứng (khuyến nghị ES256; fallback RS256).
Identity service giữ private key ký; các service khác nhận public key qua secret/env để verify
cục bộ. Blacklist/refresh giữ qua Redis Cloud.

**Rationale**: FR-022/FR-023; verify cục bộ ~0ms, không gọi mạng mỗi request; key public an toàn
hơn khi phân phối.

**Alternatives considered**:
- Giữ HS256 chia sẻ secret: kém an toàn khi nhiều service.
- Introspection: thêm độ trễ và phụ thuộc identity sống.

**Di trú**: hỗ trợ verify cả HS256 (legacy, giai đoạn chuyển tiếp) và thuật toán mới; loại bỏ HS256
sau khi mọi service cập nhật.

---

## R11. Parameter hoá config (giới hạn `os.ExpandEnv`)

**Decision**: `config.production.yaml` thay host/endpoint literal bằng `${VAR}` (ví dụ
`${DB_HOST}`, `${REDIS_HOST}`, `${RABBITMQ_HOST}`, `${ELASTICSEARCH_ADDRESSES}`). Với cờ boolean
mà `os.ExpandEnv` không hỗ trợ default (`${VAR:-false}`), dùng literal mặc định `false` trong YAML
rồi override bằng `os.LookupEnv` trong `loadConfig()`.

**Rationale**: FR-005/FR-006/SC-005; cùng một file config dùng cho nhiều môi trường.

**Alternatives considered**:
- Nhiều file config theo môi trường: khó bảo trì, dễ lệch.
- Viper `AutomaticEnv`: thay đổi cơ chế load hiện tại, rủi ro không cần thiết.

---

## R12. Môi trường Cloud (drill), CI/CD và docs

> **Cập nhật 2026-09-15**: **không có môi trường staging tách biệt** — môi trường Cloud dùng
> **1 database dùng chung** (không tách schema staging/prod); drill runbook dùng schema tạm rồi xoá;
> CI build **3 binary service** (không monolith); giữ workflow VPS cho legacy song song (không disable
> cho tới khi repatriation xong).

**Decision**:
- Drill: dùng **schema tạm** trong database dùng chung (xoá sau khi drill); CloudAMQP vhost riêng;
  Redis Cloud; Cloud Run (truy cập qua **ingress VPS Nginx**, không Cloudflare — cf. R22), để chạy thử runbook (FR-027, SC-009).
- CI/CD: thêm workflow `deploy-platform.yml` (build **3** service binary → deploy Cloud Run
  HTTP + worker); giữ `deploy-production.yml` (VPS legacy song song, không disable cho tới khi
  repatriation xong); CI build kiểm tra 3 binary (FR-020).
- Docs: `docs/operations/ha-tang/` + `docs/operations/` (ingress/nginx, teardown, repatriate) +
  `docs/system-design/microservices/` + ADR `docs/decisions/021-funded-platforms-and-repatriation.md`;
  cập nhật `README.md`, `local-setup.md`.

**Rationale**: FR-013..FR-017/FR-020/FR-025/FR-027; giữ đường rollback bằng cách không xoá workflow VPS.

**Alternatives considered**:
- Xoá workflow VPS: mất đường rollback nhanh.
- Không drill trước cutover: rủi ro cao cho cutover lớn (chốt phải drill runbook).

---

## R13. Rủi ro & giảm thiểu

| Rủi ro | Giảm thiểu |
|--------|-----------|
| Pooler không hỗ trợ LISTEN/lock | Tách DSN session cho migration/listener; poll fallback |
| TLS managed khác self-hosted | Cờ TLS toggle mặc định false; test cả hai môi trường (Cloud + VPS) |
| Service module hỏng khi tách | CI build 3 service là gate bắt buộc (hiến pháp 3.0.0; monolith chỉ còn tag frozen) |
| Mất dữ liệu phát sinh sau cutover | Backup trước cutover + reverse + hạn tài trợ + funding clock T-minus; ghi rõ trong runbook |
| Worker bị scale-to-zero | Deployment worker riêng với `min-instances=1` + `max-instances=1` (FR-045) |
| Lệch topology queue CloudAMQP | Code tự khai báo topology; không cấu hình tay |
| Gõ/generated code lệch DTO | Contract test mapping DTO ⇄ proto + CI generate-check |

---

## R14. Edge: Cloudflare subdomain zone + Worker router (thay Nginx-VPS-gateway/LB/Firebase)

> **SUPERSEDED 2026-09-15 (đợt 4) bởi R22**: KHÔNG dùng Cloudflare/Worker trong đường API.
> Ingress là **VPS Nginx hiện hữu**; DNS ở Vercel. Nội dung dưới giữ lại để truy vết.

**Decision**: NS-delegate `api-v2.closy.hycat.online` thành zone Cloudflare riêng, gắn Pro Plan
theo zone (zone gốc ở lại Vercel, không move). Worker edge-router định tuyến longest-prefix
(~30 rule, xử lý chồng lấn `/me/*`, `/admin/*`) tới 3 Cloud Run `run.app` nội bộ (không publish).
WAF Managed + OWASP Core bật; 2 rate-limit rule (login/OTP + API chung); Worker tiêm
`X-Gateway-Secret`, backend verify (origin-auth, trừ healthcheck hạ tầng).

**Rationale**: FR-025/FR-028; không tốn LB/Armor; chống bypass `run.app`; rate-limit toàn cục
mà Nginx scale-ngang không làm được; Worker free 100k req/ngày đủ demo.

**Alternatives considered**:
- Nginx VPS gateway cho api-v2: double-hop, origin yếu, VPS thành điểm nghẽn — loại.
- GCP ALB + Armor: ngoài ngân sách — loại.
- Nginx-on-Cloud-Run: hop tính phí thừa, rate-limit theo instance, domain-mapping preview — loại.
- Firebase Hosting rewrites: cần Blaze/billing cho Cloud Run, quota transfer thấp — loại.

**Fallback**: nếu NS-delegate thất bại → SaaS custom hostname CNAME thẳng cho cả `api-v2`,
cùng Worker phục vụ.

---

## R15. Cutover `api.closy` bằng SaaS custom hostname + redirect

> **SUPERSEDED 2026-09-15 (đợt 4) bởi R22**: giai đoạn đầu **không cutover**; chạy song song.
> `api.closy` giữ nguyên monolith; cutover là **thao tác thủ công** có runbook (FR-041).

**Decision**: `api.closy...` thành custom hostname trên zone api-v2 (1 CNAME trên Vercel);
sau cutover `api-v2...` redirect 301 về `api.closy...` bằng Redirect Rule. VPS giữ stopped
làm fallback + reverse-cutover (trỏ DNS về VPS).

**Rationale**: FR-029; hưởng full edge mà không move zone; redirect ở edge nên không cần giữ
VPS chỉ để redirect.

---

## R16. Supabase 1 project, 1 database dùng chung

> **Cập nhật 2026-09-15 (đợt 2)**: bỏ phương án tách schema staging/prod. Chỉ có **1 database dùng
> chung** cho môi trường Cloud; drill runbook dùng schema tạm rồi xoá; `api-v2` (test nội bộ) và
> `api.closy` (sau cutover) cùng trỏ database đó.

**Decision**: 1 project Pro, 1 database dùng chung; credential riêng cho drill/migration; runtime
pooler 6543, migration (job riêng) + `LISTEN` direct 5432, `sslmode=require`.

**Rationale**: Tiết kiệm 1 project Pro; rủi ro nhầm lẫn kiểm soát bằng kỹ thuật + checklist.

---

## R17. Compute trial + teardown bắt buộc

**Decision**: Cloud Run chạy bằng trial/free-tier + billing cap cứng + max-instances cap;
runbook teardown xóa toàn bộ resource trial và verify bill về 0 sau demo (US6).

**Rationale**: compute không funded; không teardown = phát sinh chi phí ngoài ngân sách.

---

## R18. Migrate dữ liệu prod VPS → Supabase lúc cutover

**Decision**: dump/restore + backfill + verify checksum + rollback; backup trước cutover,
trước reverse, trước hạn tài trợ và theo funding clock (thay chiến lược "chỉ dump trước hạn").

---

## R19. Mô hình C multi-module + `go.work`, bỏ monolith

**Decision**: 3 service module (đặt trong `services/`) + module `shared` + module `contracts` trên
`go.work`; `cmd/server` xóa khỏi `main`;
tag `monolith-legacy` frozen; ranh giới service do compiler enforce (thay R2.1 manifest-check);
client gRPC ở `contracts/<owner>/v1/client`; server bind contract interface.

**Rationale**: FR-018/FR-020/FR-021; yêu cầu môn học; ranh giới chắc chắn hơn CI-check.

---

## R20. Repatriation bắt buộc sau 3 tháng + funding clock

**Decision**: Phase repatriation: 3 service image chạy compose trên VPS + Nginx gateway
(route map trong Nginx) + migrate ngược (Supabase→Postgres, CloudAMQP→RMQ
drain, Redis Cloud→Redis) + DNS về VPS + tắt funded items trước renew. Funding clock
T-30/14/7/1 có xác nhận. Runbook repatriation viết song song từ bây giờ.

**Rationale**: US4/US7, FR-032, §VIII time-box rule; hết kinh phí mà chưa về = mất vận hành +
bị charge thêm chu kỳ.

---

## R21. Swagger per-service, giữ 9 section + Nginx route map

> **Cập nhật 2026-09-15 (đợt 4)**: bỏ "Worker routes"; route map nằm trong Nginx (single source);
> CI parity Worker↔Nginx không còn áp dụng.

**Decision**: Mỗi service `swag init` trên module nó chứa; **Nginx** phục vụ docs index + route docs
từng service; CI assert đủ 9 section (fail nếu thiếu).

**Rationale**: FR-030, SC-012; annotation giữ nguyên (không gộp tay); repatriation không mất routing.

---

## R22. Ingress = VPS Nginx hiện hữu (bỏ Cloudflare/Worker khỏi đường API)

**Decision**: Giữ **VPS Nginx hiện hữu** làm reverse proxy/ingress. `api.closy...` → `http://backend:8080`
(monolith, giữ nguyên). Thêm server block cho `api-v2.closy.hycat.online` → 3 Cloud Run `run.app`,
route longest-prefix (xử lý `/me/*`, `/admin/*`). DNS ở **Vercel** (bản ghi A/CNAME trỏ VPS); **không**
NS-delegate, **không** chuyển nameserver, **không** Cloudflare/Worker trong đường API. Không thêm
GCP LB / API Gateway chỉ để làm ingress.

**Rationale**: FR-025/FR-028/FR-029; VPS đã được đầu tư và đang có Nginx + TLS + hardening sẵn;
giảm thành phần mới, giữ `api.closy` không đổi; Cloudflare không cần thiết cho API path.

**Alternatives considered**:
- NS-delegate/Cloudflare zone + Worker (R14 cũ): bỏ — không cần và làm phức tạp DNS/path.
- GCP LB + serverless NEG: ngoài ngân sách và thêm hạ tầng không cần.
- Cloud Run domain mapping: preview, không khuyến nghị production.

---

## R23. Cloud Run IAM + token helper trên VPS (giữ `Authorization`)

**Decision**: Cloud Run services **không** bật `allUsers`; grant `roles/run.invoker` cho service
account của token helper. Nginx gửi Google ID token ở **`X-Serverless-Authorization: Bearer <token>`**
(Cloud Run strip header này trước khi tới container) → app-level `Authorization: Bearer <Closy JWT>`
được giữ nguyên. Audience mặc định = URL `run.app` của service (custom audience nếu cần).
**Token helper** chạy trên VPS: sinh/cache/refresh ID token; credential least-privilege + rotation.

**Trade-off chọn cách lấy credential**:
- **Service account key (JSON)** — chọn: đơn giản, chạy được ngoài GCP, hỗ trợ bởi auth libraries;
  nhược: secret dài hạn → giảm thiểu bằng least-privilege (`run.invoker` đúng 3 service), file 0600,
  rotation định kỳ, helper bind localhost.
- **Workload Identity Federation** — an toàn hơn (không key tĩnh) nhưng cần IdP/OIDC issuer cho VPS;
  phức tạp hơn đáng kể so với kiến trúc hiện tại → chưa chọn, ghi nhận làm hướng nâng cấp.
- **Metadata server** — không dùng được ngoài GCP.
- **`gcloud` impersonation** — phụ thuộc user credential, không phù hợp chạy tự động.

**Rationale**: FR-033/FR-039; giữ app Authorization nguyên vẹn, chặn gọi thẳng `run.app`, không
dùng `allUsers`.

---

## R24. TLS (Let's Encrypt) & DNS cho `api-v2`

**Decision**: Nginx VPS tiếp tục terminate TLS cho cả `api.closy...` và `api-v2.closy.hycat.online`.
Mở rộng cert hiện có để gồm hostname mới bằng **`certbot --expand`/`--cert-name`** (liệt kê đủ domain
cũ + mới) hoặc cert riêng; giữ cơ chế renew hiện có (webroot/nginx plugin trong Docker).
DNS: thêm bản ghi `api-v2` trên **Vercel** trỏ VPS.

**Rationale**: FR-029/FR-040; cert hiện tại chưa chắc gồm `api-v2` → phải cấp/mở rộng trước khi bật DNS.

**Alternatives considered**:
- Cloudflare/Google-managed cert: không dùng vì ingress là Nginx.
- Standalone certbot: chọn webroot/nginx plugin để hợp với Nginx đang chạy trong Docker.
