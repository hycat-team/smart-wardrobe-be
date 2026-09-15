# ADR 021 - Nền tảng được tài trợ, kiến trúc Cloudflare + Cloud Run và lộ trình repatriation 3 tháng

*   **Status**: Accepted (**đã supersede một phần** bởi **ADR 023**: bỏ Cloudflare/Worker và NS-delegate khỏi đường API; ingress là VPS Nginx; cutover giai đoạn đầu bị hoãn)
*   **Decision Date**: 2026-09-15
*   **Decision Maker**: Duck (+ Muse Spark đối chiếu)
*   **Supersedes**: Mọi giả định trial/LB/GCP-managed-DB trong `specs/020-platform-migration` trước ngày này.
*   **Ghi chú quản trị**: nội dung vòng đời kinh phí/funding-clock thuộc
    `docs/project/funding-and-platform-lifecycle.md` (hiến pháp 3.0.0 chỉ chứa quy định
    code/system/architecture và đã bao gồm mô hình service topology).

## 1. Context

Dự án được đầu tư kinh phí cho đúng 4 hạng mục, duy trì **3 tháng**:

| Hạng mục | Gói | Vai trò |
|---|---|---|
| Cloudflare | Pro Plan | Edge: DNS zone `api-v2...`, WAF, rate-limit, Worker router |
| Supabase | Pro Plan | Postgres + pgvector (1 project, **1 database dùng chung** cho môi trường Cloud) |
| CloudAMQP | Tough Tiger ($19/tháng, shared) | RabbitMQ: ~1000 queue, 100k queued messages, quota 10M msg/tháng, **100 connections** |
| Redis Cloud | Essentials 250MB (~$5–9/tháng) | Cache: 256 connections, 100GB bandwidth/tháng, 1000 ops/s, TLS |

Ràng buộc cứng:
- Zone gốc `hycat.online` ở Vercel **không được chuyển** sang Cloudflare; chỉ được delegate subdomain `api-v2.closy.hycat.online` bằng NS record (Vercel cho phép).
- Môn học yêu cầu workload chạy trên Cloud công cộng; VPS không tính.
- Compute (Cloud Run) **không thuộc diện đầu tư** → chạy bằng trial/free-tier + billing cap + teardown.
- Hết 3 tháng: toàn bộ phải về lại VPS.

## 2. Decision

> **Lưu ý supersede (2026-09-15, ADR 023)**: các quyết định liên quan tới **Cloudflare/Worker và ingress**
> dưới đây **không còn hiệu lực**: §2.1 (Cloudflare zone + Worker), §2.2 (cutover SaaS — hoãn sang phase
> sau), §2.6 (origin-auth `X-Gateway-Secret`), §2.7 (SSE qua Worker), §2.8 (route table single-source với
> Worker). Thay thế: **ingress = VPS Nginx hiện hữu**; auth = **Cloud Run IAM (ID token)**; DNS ở **Vercel**.
> Các quyết định về compute, database, queue/cache, migration, repatriation, funding clock vẫn giữ nguyên.

1.  **Edge = Cloudflare Pro trên subdomain zone `api-v2.closy.hycat.online`** (NS-delegate từ
    Vercel, plan gắn theo zone). Worker edge-router định tuyến longest-prefix (~30 rule) tới 3
    service Cloud Run; WAF Managed Ruleset + OWASP Core bật; 2 rate-limit rule (login/OTP + API
    chung); Worker tiêm `X-Gateway-Secret`, backend verify.
2.  **Cutover `api.closy...` bằng Cloudflare for SaaS custom hostname** trên zone api-v2 (1 CNAME
    trên Vercel). Sau cutover, `api-v2...` redirect 301 về `api.closy...` bằng Redirect Rule.
3.  **Compute = Cloud Run `asia-southeast1`**, request-based billing, scale-to-zero cho HTTP,
    worker deployment riêng `min-instances>=1`; billing cap cứng; teardown trial là **một bước
    trong repatriation** (không tách rời sau demo).
4.  **Database = 1 project Supabase Pro, 1 database dùng chung** cho toàn bộ môi trường Cloud
    (không tách schema staging/prod); runtime qua Supavisor pooler, migration + `LISTEN` qua direct
    connection; drill runbook dùng **schema tạm rồi xoá**; migrate dữ liệu prod VPS → Supabase
    tại thời điểm merge/cutover.
5.  **Queue/Cache = CloudAMQP Tiger + Redis Cloud Essentials** qua TLS; service share connection
    trong giới hạn gói; ES giữ disabled.
6.  **Origin-auth**: Worker tiêm `X-Gateway-Secret`, service verify (trừ healthcheck) để chặn gọi
    thẳng `run.app`. gRPC liên service xác thực hai lớp: Cloud Run ID token + JWT bất đối xứng.
    Migration do **chính service chạy khi khởi động**: **mỗi service một bảng version riêng**
    (Account giữ `goose_db_version`; Commerce/AI có bảng hậu tố) + `WithAllowOutofOrder` + advisory
    lock dùng chung; 24 migration lịch sử do Account giữ.
7.  **SSE/AI streaming đi qua Worker** (passthrough stream, Workers Free; fallback gọi trực tiếp
    `run.app` nếu vượt hạn mức). Do Cloudflare proxy có **idle timeout 100s** và có thể buffer SSE:
    bắt buộc heartbeat ~15s, origin trả `X-Accel-Buffering: no`, và hạ
    `ai_chat_chunk_deadline_seconds` từ **180s → 30s thống nhất mọi môi trường** (sửa config dùng
    chung `configs/`, tránh phân nhánh config). Đây là thay đổi kế thừa từ ADR 020 §2.4; theo
    ADR 020 ("mở ADR mới khi trigger kích hoạt"), ghi nhận tại đây thay vì sửa ADR 020.
8.  **Repatriation bắt buộc sau 3 tháng**: 3 service image chạy lại trên VPS bằng
    docker-compose multi-service + Nginx gateway (route table single-source với Worker, CI check
    parity); dữ liệu migrate ngược; DNS về VPS; tắt funded items trước ngày renew. Runbook
    repatriation viết **song song** với forward runbook. Teardown trial nằm trong bước này.
9.  **Funding clock**: hard deadline + checkpoint T-30/T-14/T-7/T-1 (verify backup, drill
    restore, rehearsal DNS rollback) ghi trong plan 020.
10. **Mốc chấm bài (điều chỉnh bởi ADR 023)**: chỉ **`api-v2...` bắt buộc chạy trên Cloud** trước
    ngày chấm; **cutover `api.closy` KHÔNG còn mốc cứng** — do owner quyết định khi `api-v2` đủ ổn
    định, có runbook cutover/rollback (FR-041, SC-015). Ưu tiên US3/US5 là P1.

## 3. Consequences

- **Positive**: Đáp ứng yêu cầu môn học (100% Cloud công cộng) trong đúng ngân sách được duyệt;
  không phát sinh LB/Armor/DB-managed ngoài danh sách; đường về VPS luôn mở.
- **Negative**: Compute sống bằng trial (rủi ro hết credit giữa kỳ → billing cap + teardown drill
  là bắt buộc); Cloud Run → Supabase là egress Internet (chấp nhận cho demo); Worker free giới
  hạn 100k req/ngày (vượt → $5/tháng chưa funded, phải báo trước).
- **Trigger reconsider**: bất kỳ hạng mục nào không renew sau 3 tháng → kích hoạt repatriation
  (đây là kế hoạch, không phải sự cố); có kinh phí compute → thay trial bằng project trả phí mà
  không sửa code.

## 4. Verify list (phải xác nhận khi implement)

- Quorum queue có được hỗ trợ trên Tiger shared plan không (fallback: classic durable + ADR bổ sung).
- Supavisor transaction-mode + `LISTEN` (đã có cách direct connection, phải test thật).
- SSE/AI streaming qua Worker: đo CPU < 10ms (Workers Free), xác nhận heartbeat 15s giữ dưới idle
  timeout 100s, origin trả `X-Accel-Buffering: no`, và test stream thật dài trước cutover.
- NS-delegate `api-v2` làm bước đầu tiên vì cả plan phụ thuộc vào nó.
- Per-service migration on-start (advisory lock dùng chung, không chạy lại 24 migration lịch sử) +
  origin-auth secret: chạy thử trên môi trường Cloud trước khi cutover prod.
- Makefile viết lại cho multi-module (FR-038): CI/dev không còn dùng path monolith.
