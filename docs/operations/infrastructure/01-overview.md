# 01 — Tổng quan hạ tầng 020

Feature `020-platform-migration`: 3 service độc lập (Account, Commerce, AI) trên Go
multi-module workspace. GĐ1 đưa VPS từ monolith sang 3 service (`services/`);
GĐ2 bật box ngoài AWS (`api-v2`) độc lập — 2 trục không dính nhau (chi tiết `09`).

## 1. Ma trận nền tảng

| Hạng mục | Managed (funded 3 tháng) | Self-host (VPS) | Ghi chú |
|---|---|---|---|
| Compute | AWS box (GĐ2: Lightsail/EC2 + compose 6 container) | Docker Compose 3 service | `APP_MODE=http/worker/all` |
| Database | Supabase Pro (1 project, 1 DB dùng chung) | Postgres + pgvector (docker) | Pooler 6543 / direct 5432 |
| Cache | Redis Cloud Essentials 250MB (TLS) | Redis docker (no TLS) | `rediss` vs `redis` |
| Queue | CloudAMQP Tough Tiger (amqps) | RabbitMQ docker | 100 conn, prefetch 5 |
| Search | Tắt mặc định (`ELASTICSEARCH_ENABLED=false`) | Tắt mặc định | Fallback database |
| Ingress | **VPS Nginx hiện hữu** (cả 2 hostname) | Nginx gateway + certbot | Cloudflare KHÔNG nằm trong đường API |
| DNS | Vercel (A/CNAME về VPS) | Vercel/DNS về VPS | Không NS-delegate |

## 2. Trình tự thực hiện (đọc theo thứ tự)

| Bước | Việc | Tài liệu |
|---|---|---|
| 0 | Đăng ký nền tảng + lấy credential + set GitHub secrets | `00-first-time-deploy-step-by-step.md` |
| 1 | Hiểu phạm vi, ma trận nền tảng, nguyên tắc | tài liệu này (`01`) |
| 2 | Supabase Postgres (pooler/direct, schema, migration) | `02-supabase-postgres.md` |
| 3 | Redis Cloud (TLS, ngân sách conn) | `03-redis-cloud.md` |
| 4 | CloudAMQP (amqps, quorum, ngân sách conn) | `04-cloudamqp-rabbitmq.md` |
| 5 | Box ngoài AWS (GĐ2: 6 container, managed) | `15-aws-compute.md` |
| 6 | VPS Nginx ingress + cert | `06-vps-nginx-ingress-cloudrun.md` |
| 7 | Tắt Elasticsearch (mặc định) | `07-elasticsearch-disabled.md` |
| 8 | Bảng biến môi trường 3 profile | `08-environment-variables.md` |
| 9 | Kiến trúc song song + gate | `09-parallel-architecture-overview.md` |
| 10 | Dump dữ liệu VPS → Supabase | `10-data-migration-vps-supabase.md` |
| 11 | DNS Vercel + CI/CD | `11-dns-vercel-ci-cd.md` |
| 12 | Repatriation về VPS | `12-repatriate-vps.md` + `../repatriate-to-vps.md` |
| 13 | Ngừng VPS cũ + thu hồi credential | `13-stop-vps-and-revoke-secrets.md` |
| 14 | Monitoring + alert | `14-monitoring.md` |

## 3. Điều kiện tiên quyết

- [ ] Supabase / Redis Cloud / CloudAMQP đã tạo và có host/port/password (`00` §0.3–0.5) — cần khi tới GĐ2.
- [ ] (GĐ2) Billing alarm AWS đã bật.
- [ ] VPS hiện hữu chạy monolith (`api.closy`), có Docker + `/opt/closy` + Nginx.
- [ ] DNS `hycat.online` quản lý ở **Vercel** (không NS-delegate).
- [ ] GitHub repo có 2 environment `aws-production`/`vps-production` + đầy đủ secrets (`00` §5).
- [ ] Tag `monolith-legacy` đã push (`00` §1).

## 4. Nguyên tắc bắt buộc

- Mọi tham số hạ tầng đi qua env (FR-005/FR-008); không hardcode host/endpoint.
- Chỉ dùng chuẩn Postgres/Redis/RabbitMQ; **không** dùng tính năng độc quyền vendor
  (Supabase Auth/RLS/Storage) (FR-007).
- **Isolation theo môi trường (luật cứng): VPS chỉ dùng infra local trong docker
  (`postgres`/`redis`/`rabbitmq`); Cloud chỉ dùng managed (Supabase/Redis Cloud/CloudAMQP).
  Cấm VPS trỏ DB/Redis/RabbitMQ ra hostname managed ở runtime — hai môi trường không bao
  giờ dùng chung infra (ngoại lệ duy nhất: đồng bộ dump/restore offline ở `10`).**
- Chuyển môi trường chỉ đổi config + DNS, **không sửa code**.
- Migration/LISTEN/advisory lock dùng direct 5432; runtime query dùng pooler 6543 (FR-046).

## 5. Verify tổng thể (gate)

- [ ] 3 service HTTP + 3 worker chạy đúng `APP_MODE`, health `/api/v1/<svc>/health` 200 đúng tên service.
- [ ] `api.closy` phục vụ 3 service VPS sau GĐ1; `api-v2` (GĐ2) route đủ 3 container + SSE.
- [ ] 0 request đi qua Cloudflare.
