# Runbook vận hành (020-platform-migration)

Tài liệu hướng dẫn **2 chiều** (lên Cloud và repatriation về VPS), viết bằng tiếng Việt.

## Cấu trúc

- `ha-tang/` — runbook hạ tầng managed + song song (US3, T083–T093).
  **Bắt đầu ở `00-first-time-deploy-step-by-step.md`** (đăng ký nền tảng + host lần đầu),
  rồi tới:
  `01-tong-quan.md`, `02-supabase-postgres.md`, `03-redis-cloud.md`,
  `04-cloudamqp-rabbitmq.md`, `05-compute-cloudrun.md`,
  `06-vps-nginx-ingress-cloudrun.md`, `07-elasticsearch-tat-mac-dinh.md`,
  `08-bang-bien-moi-truong.md`, `09-kien-truc-song-song-tong-quan.md`,
  `10-chuyen-du-lieu-vps-supabase.md`, `11-dns-vercel-ci-cd.md`,
  `12-ve-lai-vps.md` (US4), `13-ngung-vps-va-thu-hoi-secret.md`, `14-monitoring.md`.
- `nginx-ingress-cloudrun-auth.md` — route map single-source, token helper, rotate SA key, smoke test IAM (T095).
- `manual-cutover-api-closy-to-cloudrun.md` — runbook cutover thủ công + rollback 2 mức (T096, FR-041).
- `repatriate-to-vps.md` — runbook repatriation đầy đủ 2 chiều, gồm bước teardown trial (T102).
- `gcp-trial-teardown.md` — thứ tự xóa trial + verify bill về 0 (T105).

## Quy ước registry & tag (T007a — FR-042, FR-051)

- Image Cloud Run → **Artifact Registry** `asia-southeast1-docker.pkg.dev/<project>/smart-wardrobe/<svc>:<git-sha>`
  (immutable; `:latest` chỉ trong CI). Tạo repo 1 lần bằng gcloud (xem `05-compute-cloudrun.md` — T087).
- Image VPS (rollback `monolith-legacy` + compose 3-service khi repatriation) → **GHCR**
  `ghcr.io/<owner>/<svc>:<git-sha>` với CI/token sẵn có — không migrate registry.
- Image private KHÔNG public. Biến quy ước nằm trong `Makefile` (`AR_HOST/AR_PROJECT/AR_REPO`,
  `GHCR_OWNER`, `GIT_SHA`).

## Nguyên tắc

- Mọi runbook ghi rõ điều kiện tiên quyết, thứ tự bước, kiểm thử và rollback (FR-015).
- Drill trên môi trường Cloud bằng **schema tạm**, xoá sau khi xong (FR-027).
