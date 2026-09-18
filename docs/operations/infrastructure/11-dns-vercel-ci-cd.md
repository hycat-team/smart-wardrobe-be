# 11 — DNS Vercel + CI/CD platform

**Điều kiện tiên quyết**: tag `monolith-legacy` đã push lên remote — nếu thiếu,
workflow rollback `deploy-production.yml` sẽ fail.

```bash
git ls-remote --tags origin | grep monolith-legacy
# nếu chưa có: git tag -a monolith-legacy <BASELINE_SHA> -m "frozen monolith-green" && git push origin monolith-legacy
```

## 1. Thêm bản ghi DNS `api-v2` trên Vercel

1. Mở https://vercel.com/dashboard → **Domains** → chọn `hycat.online`.
2. Vào tab **DNS Records** → **Add**:
   - **Type**: `A` (hoặc `CNAME` nếu trỏ tới hostname có sẵn)
   - **Name**: `api-v2`
   - **Value**: IP VPS (hoặc target CNAME hiện tại của `api`)
   - **TTL**: để thấp (vd `60`) trong giai đoạn chuyển.
3. **Không** NS-delegate, **không** đổi nameserver, **không** dùng Cloudflare DNS (FR-029).
4. Chỉ thêm **sau khi** cert + Nginx server block `api-v2` đã sẵn sàng (`06` §4–§5).

## 2. Verify DNS

```bash
dig +short api-v2.closy.hycat.online
dig +short api.closy.hycat.online       # không đổi
# Xác nhận không có Cloudflare
curl -sSI https://api-v2.closy.hycat.online | grep -i cf-ray || echo "0 Cloudflare OK"
```

## 3. CI/CD platform (FR-042/FR-052)

| Workflow | Trigger | Nội dung |
|---|---|---|
| `ci.yml` | PR/push `main` | fmt/tidy/generate-check, vet, test(-race), build matrix 3 service, buf lint + proto drift, parity swagger, image-build matrix |
| `deploy-platform.yml` | dispatch | Matrix per service → Artifact Registry (`<svc>:<sha>`) → Cloud Run HTTP + worker; secrets từ GitHub Actions; 1 service fail không chặn 2 còn lại |
| `deploy-vps-services.yml` | dispatch + approval | Matrix 3 service → GHCR (pin digest) → `compose up -d <svc>`; dùng cho repatriation + vận hành VPS |
| `deploy-production.yml` | dispatch + xác nhận | **Chỉ rollback khẩn cấp**: build từ tag `monolith-legacy` → GHCR → compose profile `rollback` |

Dispatch (xem `00` §6.1):
```bash
gh workflow run deploy-platform.yml   -R "<OWNER/REPO>" --ref main -f services=all
gh workflow run deploy-vps-services.yml -R "<OWNER/REPO>" --ref main -f services=all
gh workflow run deploy-production.yml -R "<OWNER/REPO>" --ref main -f confirm=rollback
```

## 4. Quy ước image/secret

- Tag immutable `<svc>:<git-sha>`; `:latest` chỉ trong CI; không public image private.
- Secrets (DB/Redis/AMQP/JWT/Cloudinary/Brevo/`SWAGGER_ACCESS_CODE`/ingress URLs) tiêm
  lúc deploy từ GitHub secrets; **0 giá trị in ra log** (SC-026); rotation = redeploy
  service đó.
