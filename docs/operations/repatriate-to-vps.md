# Repatriate to VPS — runbook 2 chiều đầy đủ

## Chiều về VPS (chính)

1. **Verify backup**: dump Supabase + checksum; snapshot Redis (RDB) nếu cần.
2. **Drill restore**: restore vào Postgres local/prod-like, verify checksum (SC-008).
3. **Deploy 3 service**: `deploy-vps-services.yml` (matrix → GHCR pin digest →
   `compose up -d <svc>`); smoke test `api.closy`.
4. **Migrate ngược dữ liệu**: Supabase → Postgres VPS; drain AMQP (đợi queue rỗng);
   Redis snapshot/warmup.
5. **DNS về VPS**; verify host × route + SSE + auth + upload.
6. **Tắt funded items trước ngày renew** (Supabase/CloudAMQP/Redis Cloud) — xem
   funding clock T-30/T-14/T-7/T-1.
7. **Teardown trial GCP** (một bước trong repatriation, FR-031):
   xoá Cloud Run → artifacts → alerts; đối chiếu billing trial về 0 (SC-010).

## Chiều lên Cloud (tham chiếu)

Xem `ha-tang/00..10` + `manual-cutover-api-closy-to-cloudrun.md`.

## Sau repatriation (FR-055)

- `api.closy` là host canonical; `api-v2` **redirect 301** về `api.closy` trong
  grace period **14–30 ngày** (cả hai host cùng phục vụ 3-service compose).
- Sau grace period: **xoá bản ghi DNS** `api-v2`.
- Xem `ha-tang/12-ve-lai-vps.md` cho migrate ngược chi tiết.

## Mục tiêu

Repatriation ≤ 4h gồm migrate ngược (SC-004); drill đầy đủ trước T-30 (T108).
