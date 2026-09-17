# 00 — Tổng quan hạ tầng 020

Feature `020-platform-migration`: 3 service độc lập (Account, Commerce, AI) trên
multi-module Go workspace, chạy **song song** (monolith `api.closy` + Cloud Run
`api-v2`) rồi repatriation về VPS khi hết 3 tháng funded.

## Ma trận nền tảng

| Hạng mục | Managed (funded 3 tháng) | Self-host (VPS) | Ghi chú |
|----------|--------------------------|-----------------|---------|
| Compute | Cloud Run trial + billing cap + teardown | Docker Compose 3 service | `APP_MODE=http/worker/all` |
| Database | Supabase Pro (1 project, 1 DB dùng chung) | Postgres + pgvector (docker) | Pooler 6543 / direct 5432 |
| Cache | Redis Cloud Essentials 250MB (TLS) | Redis docker (no TLS) | `rediss` vs `redis` |
| Queue | CloudAMQP Tough Tiger (amqps) | RabbitMQ docker | 100 conn, prefetch 5 |
| Search | Tắt mặc định (`ELASTICSEARCH_ENABLED=false`) | Tắt mặc định | Fallback database |
| Ingress | **VPS Nginx hiện hữu** (cả 2 hostname) | Nginx gateway + certbot | Cloudflare KHÔNG nằm trong đường API |
| DNS | Vercel (A/CNAME về VPS) | Vercel/DNS về VPS | Không NS-delegate |

## Hybrid trong giai đoạn song song

- `api.closy` → monolith frozen (`monolith-legacy`) trên VPS.
- `api-v2` → VPS Nginx → 3 Cloud Run (IAM `run.invoker`, token helper).
- Dữ liệu `api-v2` dump 1 lần từ VPS; re-dump tại cutover.

## Cảnh báo constitution

- Mọi tham số hạ tầng qua env (FR-005/FR-008); không hardcode host/endpoint.
- Không dùng tính năng độc quyền vendor (Supabase Auth/RLS/Storage) — chỉ Postgres/Redis/RabbitMQ chuẩn (FR-007).
- Chuyển môi trường chỉ đổi config + DNS, không sửa code.

## Runbook trong thư mục này

`01-supabase` · `02-redis` · `03-cloudamqp` · `04-cloudrun` ·
`05-nginx-ingress` · `06-es-off` · `07-bang-bien-moi-truong` ·
`08-song-song` · `09-chuyen-du-lieu` · `10-dns-vercel-ci-cd` ·
`11-ve-lai-vps` · `12-ngung-vps-va-thu-hoi-secret`
