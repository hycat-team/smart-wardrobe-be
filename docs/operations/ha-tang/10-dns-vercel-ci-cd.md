# 10 — DNS Vercel + CI/CD platform

## DNS (FR-029)

- `hycat.online` giữ DNS tại **Vercel**; **không** NS-delegate, **không** chuyển
  nameserver, **không** Cloudflare DNS.
- `api-v2.closy.hycat.online`: bản ghi **A (hoặc CNAME)** trỏ về VPS hiện tại.
  Thêm **sau khi** cert + Nginx server block sẵn sàng (T070→T069→T068).

## CI/CD platform (FR-042/FR-052)

| Workflow | Trigger | Nội dung |
|----------|---------|----------|
| `ci.yml` | PR/push `main` | fmt/tidy/generate-check, vet, test(-race), build matrix 3 service, buf lint + proto drift, parity swagger, image-build matrix |
| `deploy-platform.yml` | dispatch | Matrix per service → Artifact Registry (`<svc>:<sha>`) → Cloud Run HTTP + worker; secrets từ GitHub Actions; 1 service fail không chặn 2 còn lại |
| `deploy-vps-services.yml` | dispatch + approval | Matrix 3 service → GHCR (pin digest) → `compose up -d <svc>`; dùng cho repatriation + vận hành VPS |
| `deploy-production.yml` | dispatch + xác nhận | **Chỉ rollback khẩn cấp**: build từ tag `monolith-legacy` → GHCR → compose profile `rollback` |

## Quy ước image/secret

- Tag immutable `<svc>:<git-sha>`; `:latest` chỉ trong CI; không public image private.
- Secrets (DB/Redis/AMQP/JWT/Cloudinary/Brevo/`SWAGGER_ACCESS_CODE`/ingress URLs)
  tiêm lúc deploy từ GitHub secrets; **0 giá trị in ra log** (SC-026); rotation =
  redeploy service đó.
