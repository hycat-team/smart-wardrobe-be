# 01 — Tổng quan hạ tầng 020

Feature `020-platform-migration`: 3 service độc lập (Account, Commerce, AI) trên Go
multi-module workspace, chạy **song song** (monolith `api.closy` + Cloud Run `api-v2`),
rồi repatriation về VPS khi hết 3 tháng funded.

## 1. Ma trận nền tảng

| Hạng mục | Managed (funded 3 tháng) | Self-host (VPS) | Ghi chú |
|---|---|---|---|
| Compute | Cloud Run trial + billing cap + teardown | Docker Compose 3 service | `APP_MODE=http/worker/all` |
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
| 5 | Cloud Run (6 units, billing cap) | `05-compute-cloudrun.md` |
| 6 | VPS Nginx ingress + token-helper + cert | `06-vps-nginx-ingress-cloudrun.md` |
| 7 | Tắt Elasticsearch (mặc định) | `07-elasticsearch-disabled.md` |
| 8 | Bảng biến môi trường 3 profile | `08-environment-variables.md` |
| 9 | Kiến trúc song song + gate | `09-parallel-architecture-overview.md` |
| 10 | Dump dữ liệu VPS → Supabase | `10-data-migration-vps-supabase.md` |
| 11 | DNS Vercel + CI/CD | `11-dns-vercel-ci-cd.md` |
| 12 | Repatriation về VPS | `12-repatriate-vps.md` + `../repatriate-to-vps.md` |
| 13 | Ngừng VPS cũ + thu hồi credential | `13-stop-vps-and-revoke-secrets.md` |
| 14 | Monitoring + alert | `14-monitoring.md` |

## 3. Điều kiện tiên quyết

- [ ] GCP project trial đã bật billing, có Artifact Registry + WIF + deployer SA (`00` §2).
- [ ] Supabase Pro / Redis Cloud / CloudAMQP đã tạo và có host/port/password (`00` §0.3–0.5).
- [ ] VPS hiện hữu chạy monolith (`api.closy`), có Docker + `/opt/closy` + Nginx.
- [ ] DNS `hycat.online` quản lý ở **Vercel** (không NS-delegate).
- [ ] GitHub repo có 2 environment `gcp-production`/`vps-production` + đầy đủ secrets (`00` §5).
- [ ] Tag `monolith-legacy` đã push (`00` §1).

## 4. Nguyên tắc bắt buộc

- Mọi tham số hạ tầng đi qua env (FR-005/FR-008); không hardcode host/endpoint.
- Chỉ dùng chuẩn Postgres/Redis/RabbitMQ; **không** dùng tính năng độc quyền vendor
  (Supabase Auth/RLS/Storage) (FR-007).
- Chuyển môi trường chỉ đổi config + DNS, **không sửa code**.
- Migration/LISTEN/advisory lock dùng direct 5432; runtime query dùng pooler 6543 (FR-046).

## 5. Verify tổng thể (gate)

- [ ] 3 service HTTP + 3 worker chạy đúng `APP_MODE`, health `/api/v1/health` 200.
- [ ] `api-v2` qua VPS Nginx route đủ 3 service + SSE + auth + upload.
- [ ] `api.closy` không đổi routing trong suốt giai đoạn song song.
- [ ] 0 request `api-v2` đi qua Cloudflare.
