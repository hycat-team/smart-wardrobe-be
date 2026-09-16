# Runbook vận hành (020-platform-migration)

Tài liệu hướng dẫn **2 chiều** (lên Cloud và repatriation về VPS), viết bằng tiếng Việt.

## Cấu trúc

- `ha-tang/` — runbook hạ tầng managed + song song (US3, T083–T093):
  `00-tong-quan.md`, `01-supabase-postgres.md`, `02-redis-cloud.md`,
  `03-cloudamqp-rabbitmq.md`, `04-compute-cloudrun.md`,
  `05-vps-nginx-ingress-cloudrun.md`, `06-elasticsearch-tat-mac-dinh.md`,
  `07-bang-bien-moi-truong.md`, `08-kien-truc-song-song-tong-quan.md`,
  `09-chuyen-du-lieu-vps-supabase.md`, `10-dns-vercel-ci-cd.md`,
  `11-ve-lai-vps.md` (US4).
- `nginx-ingress-cloudrun-auth.md` — route map single-source, token helper, rotate SA key, smoke test IAM (T095).
- `manual-cutover-api-closy-to-cloudrun.md` — runbook cutover thủ công + rollback 2 mức (T096, FR-041).
- `repatriate-to-vps.md` — runbook repatriation đầy đủ 2 chiều, gồm bước teardown trial (T102).
- `gcp-trial-teardown.md` — thứ tự xóa trial + verify bill về 0 (T105).

## Quy ước registry & tag (T007a — FR-042, FR-051)

- Image Cloud Run → **Artifact Registry** `asia-southeast1-docker.pkg.dev/<project>/smart-wardrobe/<svc>:<git-sha>`
  (immutable; `:latest` chỉ trong CI). Tạo repo 1 lần bằng gcloud (xem `04-compute-cloudrun.md` — T087).
- Image VPS (rollback `monolith-legacy` + compose 3-service khi repatriation) → **GHCR**
  `ghcr.io/<owner>/<svc>:<git-sha>` với CI/token sẵn có — không migrate registry.
- Image private KHÔNG public. Biến quy ước nằm trong `Makefile` (`AR_HOST/AR_PROJECT/AR_REPO`,
  `GHCR_OWNER`, `GIT_SHA`).

## Nguyên tắc

- Mọi runbook ghi rõ điều kiện tiên quyết, thứ tự bước, kiểm thử và rollback (FR-015).
- Drill trên môi trường Cloud bằng **schema tạm**, xoá sau khi xong (FR-027).
