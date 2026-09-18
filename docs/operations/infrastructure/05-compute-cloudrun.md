# 05 — Compute Cloud Run (trial + billing cap)

Điều kiện: GCP project đã bật billing, đã tạo Artifact Registry + WIF + deployer SA
(`00` §2), và đã set đầy đủ secret (`00` §5).

## 1. Topology (FR-044/FR-045)

| Unit | `APP_MODE` | Scale | CPU | Ghi chú |
|---|---|---|---|---|
| account/commerce/ai HTTP | `http` | scale-to-zero, `max-instances=2` | theo request, `concurrency=40` | Chỉ HTTP+gRPC; không consumer/cron |
| account/commerce/ai worker | `worker` | `min=1` + `max=1` (singleton) | always-on | consumer/outbox/cron + health `/api/v1/health` |

Region `asia-southeast1`.

## 2. Bật API + tạo Artifact Registry (1 lần)

```bash
gcloud config set project <PROJECT>
gcloud services enable run.googleapis.com artifactregistry.googleapis.com \
  iam.googleapis.com iamcredentials.googleapis.com cloudresourcemanager.googleapis.com \
  monitoring.googleapis.com billingbudgets.googleapis.com

gcloud artifacts repositories create smart-wardrobe \
  --repository-format=docker --location=asia-southeast1 \
  --description="Closy service images"
gcloud artifacts repositories list --location=asia-southeast1   # verify
```

## 3. Set secret trước khi deploy

Theo `00` §5. Tối thiểu cần: `GCP_*`, `DB_*`, `REDIS_*`, `RABBITMQ_*`, `JWT_*`,
`SWAGGER_ACCESS_CODE`, `CLOUDINARY_*`, `EMAIL_SMTP_*`, `PAYOS_*`.
`INGRESS_*` điền ở bước 5 (sau khi có URL).

## 4. Deploy lần đầu

```bash
gh workflow run deploy-platform.yml -R "<OWNER/REPO>" --ref main -f services=all
RUN_ID=$(gh run list --workflow deploy-platform.yml -R "<OWNER/REPO>" --limit 1 \
  --json databaseId --jq '.[0].databaseId')
gh run watch "$RUN_ID" -R "<OWNER/REPO>"
```
Matrix per service: build Dockerfile riêng → push Artifact Registry (`<svc>:<git-sha>`
immutable) → deploy HTTP (`--use-http2`) + worker (CPU always-on). 1 service fail
không chặn 2 service còn lại (SC-017).

## 5. Lấy URL + cấu hình ingress

```bash
for svc in account commerce ai; do
  echo "$svc = $(gcloud run services describe smart-wardrobe-$svc \
    --region asia-southeast1 --format='value(status.url)')"
done
```
Set `INGRESS_ACCOUNT_URL` / `INGRESS_COMMERCE_URL` / `INGRESS_AI_URL` và
`INGRESS_TOKEN_AUDIENCE` (= URL Account) theo `00` §6, rồi **chạy lại** workflow để
app nhận ingress URL.

## 6. Verify 6 units

```bash
gcloud run services list --region asia-southeast1
# Kiểm tra mode từng service
gcloud run services describe smart-wardrobe-account \
  --region asia-southeast1 --format='value(spec.template.spec.containers[0].env)'
# IAM phải đóng: thiếu ID token -> 403
curl -s -o /dev/null -w '%{http_code}\n' \
  "$(gcloud run services describe smart-wardrobe-account --region asia-southeast1 --format='value(status.url)')/api/v1/health"
```
- [ ] 6 units chạy đúng mode; health `/api/v1/health` pass (SC-018/SC-023).
- [ ] Không consumer/cron trong `http`; worker singleton + advisory lock (SC-019).
- [ ] Tổng conn trong ngân sách (SC-021).
- [ ] Deploy 1 service không ảnh hưởng 2 service còn lại (SC-017).

## 7. Billing cap + alert

Console GCP → Billing → Budgets & alerts: tạo budget cho `<PROJECT>`, cap = ngân sách
trial, alert 50/90/100%. Hoặc:
```bash
gcloud billing budgets create --billing-account=<BILLING_ID> \
  --display-name="closy-trial-cap" --budget-amount=<USD> --threshold-rule=percent=0.9
```

## 8. Rollback revision-level (khi deploy mới lỗi)

```bash
gcloud run revisions list --service smart-wardrobe-account --region asia-southeast1
gcloud run services update-traffic smart-wardrobe-account \
  --region asia-southeast1 --to-revisions <REVISION_CU>=100
```
