# Repatriate to VPS — runbook 2 chiều đầy đủ

Mục tiêu: repatriation ≤ 4h gồm migrate ngược (SC-004); drill đầy đủ trước T-30 (T108).

> **Khi nào dùng**: khi kết thúc cửa sổ funded (trước ngày renew) hoặc khi muốn chủ động
> đưa hệ thống về VPS.
> **Điều kiện**: gate song song đạt (`infrastructure/09-parallel-architecture-overview.md`);
> VPS có Docker + `/opt/closy`; image 3 service đã push GHCR.
> **Không dùng khi**: đang ở giai đoạn dựng song song (dùng `infrastructure/00`..`11`).

## 1. Điều kiện tiên quyết

- [ ] Gate song song đạt (`infrastructure/09-parallel-architecture-overview.md`).
- [ ] VPS có Docker + `/opt/closy` (compose + nginx + `.env.production`).
- [ ] Image 3 service sẵn sàng trên GHCR (build bởi `deploy-vps-services.yml`).

## 2. Chiều về VPS — thứ tự bước

1. **Verify backup**: dump Supabase + checksum; snapshot Redis (RDB) nếu cần.
2. **Drill restore**: restore vào Postgres local/prod-like, verify checksum (SC-008).
3. **Deploy 3 service**:
   ```bash
   gh workflow run deploy-vps-services.yml -R "<OWNER/REPO>" --ref main -f services=all
   gh run watch <RUN_ID> -R "<OWNER/REPO>"
   ```
   (matrix → GHCR pin digest → `compose up -d <svc>`); smoke test `api.closy`.
   Sau đó **dừng backend cũ** nếu còn chạy (tránh 2 writer cùng DB):
   `docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml stop backend || true`.
4. **Migrate ngược dữ liệu**: Supabase → Postgres VPS; drain AMQP (đợi queue rỗng);
   Redis snapshot/warmup. Chi tiết `infrastructure/12-repatriate-vps.md`.
5. **DNS về VPS**; verify host × route + SSE + auth + upload.
6. **Tắt funded items trước ngày renew** (Supabase/CloudAMQP/Redis Cloud) theo funding
   clock T-30/T-14/T-7/T-1.
7. **Teardown trial GCP** (FR-031): xoá Cloud Run → artifacts → alerts; đối chiếu billing
   trial về 0 (SC-010) — xem `gcp-trial-teardown.md`.

## 3. Chiều lên Cloud (tham chiếu)

Xem `infrastructure/00-first-time-deploy-step-by-step.md`,
`infrastructure/10-data-migration-vps-supabase.md` và
`manual-cutover-api-closy-to-cloudrun.md`.

## 4. Sau repatriation (FR-055)

- `api.closy` là host canonical; `api-v2` **redirect 301** về `api.closy` trong grace
  period **14–30 ngày** (cả hai host cùng phục vụ 3-service compose).
- Sau grace period: **xoá bản ghi DNS** `api-v2`.
- Thu hồi credential cũ + ngừng VPS: `infrastructure/13-stop-vps-and-revoke-secrets.md`.

## 5. Verify

- [ ] 3 service chạy trên VPS, `api.closy` pass smoke + route + SSE + auth + upload.
- [ ] Checksum dữ liệu migrate ngược khớp (SC-008).
- [ ] AMQP drained; Redis đã snapshot/warmup (nếu cần).
- [ ] Billing managed/trial không còn charge mục đã tắt (SC-010/SC-011).
