# Runbook vận hành (020-platform-migration)

Tài liệu hướng dẫn **2 chiều** (lên Cloud và repatriation về VPS), viết bằng tiếng Việt.
Mỗi runbook gồm: điều kiện tiên quyết → thứ tự bước (kèm lệnh) → verify → rollback.

## Cấu trúc

- `infrastructure/` — runbook hạ tầng managed + song song (US3, T083–T093).
  **Bắt đầu ở `infrastructure/00-first-time-deploy-step-by-step.md`** (đăng ký nền tảng +
  host lần đầu), rồi tới:
  `infrastructure/01-overview.md`, `infrastructure/02-supabase-postgres.md`,
  `infrastructure/03-redis-cloud.md`, `infrastructure/04-cloudamqp-rabbitmq.md`,
  `infrastructure/05-compute-cloudrun.md`, `infrastructure/06-vps-nginx-ingress-cloudrun.md`,
  `infrastructure/07-elasticsearch-disabled.md`, `infrastructure/08-environment-variables.md`,
  `infrastructure/09-parallel-architecture-overview.md`,
  `infrastructure/10-data-migration-vps-supabase.md`, `infrastructure/11-dns-vercel-ci-cd.md`,
  `infrastructure/12-repatriate-vps.md` (US4), `infrastructure/13-stop-vps-and-revoke-secrets.md`,
  `infrastructure/14-monitoring.md`.
- `nginx-ingress-cloudrun-auth.md` — route map single-source, token helper, rotate SA key,
  smoke test IAM (T095).
- `manual-cutover-api-closy-to-cloudrun.md` — runbook cutover thủ công + rollback 2 mức
  (T096, FR-041).
- `repatriate-to-vps.md` — runbook repatriation đầy đủ 2 chiều, gồm bước teardown trial (T102).
- `gcp-trial-teardown.md` — thứ tự xóa trial + verify bill về 0 (T105).

## Theo tình huống — dùng khi nào

| Runbook | Loại | Dùng khi |
|---|---|---|
| `infrastructure/00-first-time-deploy-step-by-step.md` | **Bắt buộc** | Lần host đầu tiên (managed + VPS song song). Đây là điểm vào; các `01..14` là chi tiết của từng mảng. |
| `infrastructure/01..11` | **Bắt buộc** (theo `00`) | Dựng song song: nền tảng, deploy Cloud Run, ingress, DNS. |
| `nginx-ingress-cloudrun-auth.md` | Tham chiếu trong lúc dựng | Khi làm ingress `api-v2` + token-helper + IAM (chi tiết của `00` §6–§7, `infrastructure/06`). |
| `infrastructure/12..14` | Theo sự kiện | Khi repatriation / tắt funded / cần monitoring. |
| `manual-cutover-api-closy-to-cloudrun.md` | **Theo sự kiện** | Chỉ khi owner quyết định cutover `api.closy` → Cloud Run, sau gate 7 ngày. |
| `repatriate-to-vps.md` | **Theo sự kiện** | Kết thúc cửa sổ funded hoặc muốn đưa hệ thống về VPS. |
| `gcp-trial-teardown.md` | **Theo sự kiện** (bước cuối của repatriation) | Sau khi repatriation chạy ổn định; **không** chạy khi managed còn phục vụ. |

**Luồng tổng quát**: `00` (+ `01..11`) → [song song] → `manual-cutover...` (nếu chọn Cloud Run)
→ `repatriate-to-vps.md` → `gcp-trial-teardown.md` →
`infrastructure/13-stop-vps-and-revoke-secrets.md`.

## Quy ước registry & tag (T007a — FR-042, FR-051)

- Image Cloud Run → **Artifact Registry**
  `asia-southeast1-docker.pkg.dev/<project>/smart-wardrobe/<svc>:<git-sha>`
  (immutable; `:latest` chỉ trong CI). Tạo repo 1 lần bằng gcloud (xem
  `infrastructure/05-compute-cloudrun.md`).
- Image VPS (rollback `monolith-legacy` + compose 3-service khi repatriation) → **GHCR**
  `ghcr.io/<owner>/<svc>:<git-sha>` với CI/token sẵn có — không migrate registry.
- Image private KHÔNG public. Biến quy ước nằm trong `Makefile` (`AR_HOST/AR_PROJECT/AR_REPO`,
  `GHCR_OWNER`, `GIT_SHA`).

## Nguyên tắc

- Mọi runbook ghi rõ điều kiện tiên quyết, thứ tự bước, kiểm thử và rollback (FR-015).
- Drill trên môi trường Cloud bằng **schema tạm**, xoá sau khi xong (FR-027).
