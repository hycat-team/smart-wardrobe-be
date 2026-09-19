# Runbook vận hành (020-platform-migration)

Tài liệu hướng dẫn vận hành VPS 3-service + box ngoài độc lập (AWS, GĐ2), viết bằng tiếng Việt.
Mỗi runbook gồm: điều kiện tiên quyết → thứ tự bước (kèm lệnh) → verify → rollback.

## Cấu trúc

- `infrastructure/` — runbook hạ tầng (US3/US4, T083–T093).
  **Bắt đầu ở `infrastructure/00-first-time-deploy-step-by-step.md`** (VPS lên 3-service),
  rồi tới:
  `infrastructure/01-overview.md`, `infrastructure/02-supabase-postgres.md`,
  `infrastructure/03-redis-cloud.md`, `infrastructure/04-cloudamqp-rabbitmq.md`,
  `infrastructure/06-vps-nginx-ingress-cloudrun.md`,
  `infrastructure/07-elasticsearch-disabled.md`, `infrastructure/08-environment-variables.md`,
  `infrastructure/09-parallel-architecture-overview.md`,
  `infrastructure/10-data-migration-vps-supabase.md`, `infrastructure/11-dns-vercel-ci-cd.md`,
  `infrastructure/12-repatriate-vps.md` (US4), `infrastructure/13-stop-vps-and-revoke-secrets.md`,
  `infrastructure/14-monitoring.md`, `infrastructure/15-aws-compute.md` (GĐ2: box ngoài).
- `repatriate-to-vps.md` — runbook gom hệ thống về VPS đầy đủ 2 chiều (T102).
- `manual-cutover-api-closy-to-cloudrun.md`, `gcp-trial-teardown.md` — tài liệu thời Cloud Run,
  **đã lỗi thời** (Cloud Run chưa từng tồn tại, xem ghi chú đầu mỗi file); cắt box ngoài
  theo `15-aws-compute.md` §10 + `12-repatriate-vps.md`.

## Theo tình huống — dùng khi nào

| Runbook | Loại | Dùng khi |
|---|---|---|
| `infrastructure/00-first-time-deploy-step-by-step.md` | **Bắt buộc** | VPS monolith → 3-service (GĐ1). Đây là điểm vào; các `01..11` là chi tiết từng mảng. |
| `infrastructure/01..11` | **Bắt buộc** (theo `00`) | Dựng VPS services: ingress, DNS, CI/CD. |
| `infrastructure/12..14` | Theo sự kiện | Khi cắt box ngoài / tắt managed / cần monitoring. |
| `repatriate-to-vps.md` | **Theo sự kiện** | Gom toàn bộ traffic về VPS (cắt box ngoài). |
| `infrastructure/15-aws-compute.md` (GĐ2) | Theo sự kiện | Khi bật box ngoài AWS (`api-v2`). |

## Hai trục độc lập (chi tiết `09`)

| Trục VPS (`NGINX_MODE`) | `api.closy.hycat.online` | Monolith | Chi tiết |
|---|---|---|---|
| `services` | 3 service VPS | **dừng** | `00` §7, `infrastructure/06` |
| `services-redirect` | 3 service VPS + `api-v2`→301 | dừng | grace sau khi tắt box ngoài |
| `rollback` | monolith | chạy | `deploy-production.yml` |

| Trục box ngoài | `api-v2.closy.hycat.online` | Chi tiết |
|---|---|---|
| tắt | (chưa có DNS) | trạng thái GĐ1 |
| bật (GĐ2) | box AWS → managed | `15-aws-compute.md` + `deploy-aws.yml` |
| cắt | về VPS (`services-redirect`) → xóa DNS | `12`, `repatriate-to-vps.md` |

**Luồng tổng quát**: `00` (VPS → `services`) → [GĐ2: `15` bật box ngoài + DNS `api-v2`] →
[cắt: `services-redirect` + reunify data → tắt box/managed] → (dự phòng luôn mở: `rollback`).

## Quy ước registry & tag (T007a — FR-042, FR-051)

- Image service (VPS + AWS) → **GHCR** `ghcr.io/<owner>/<svc>:<git-sha>`
  (immutable; `:latest` chỉ trong CI). Deploy VPS/AWS pull từ GHCR — không migrate registry.
- Image rollback `monolith-legacy` → **GHCR** (build 1 lần từ tag, T097a).
- Image private KHÔNG public. Biến quy ước nằm trong `Makefile`
  (`AR_HOST/AR_PROJECT/AR_REPO` giữ cho build registry thủ công, `GHCR_OWNER`, `GIT_SHA`).

## Nguyên tắc

- Mọi runbook ghi rõ điều kiện tiên quyết, thứ tự bước, kiểm thử và rollback (FR-015).
- Chuyển môi trường chỉ đổi config + DNS, **không sửa code**.
- VPS chỉ dùng infra local; box ngoài chỉ dùng managed (luật isolation, `01` §4).
