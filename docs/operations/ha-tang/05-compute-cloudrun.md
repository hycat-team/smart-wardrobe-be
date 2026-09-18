# 05 — Compute Cloud Run (trial + cap)

## Topology (FR-044/FR-045)

| Unit | `APP_MODE` | Scale | CPU | Ghi chú |
|------|-----------|-------|-----|---------|
| account/commerce/ai HTTP | `http` | scale-to-zero, `max-instances=2` | theo request, `concurrency=40` | Chỉ HTTP+gRPC; không consumer/cron |
| account/commerce/ai worker | `worker` | `min=1` + `max=1` (singleton) | always-on | consumer/outbox/cron + health `/api/v1/health` |

Region `asia-southeast1`; trial + **billing cap + alert**; teardown là một bước
trong repatriation (FR-031).

## Deploy (CI `deploy-platform.yml`, FR-042)

- Matrix per service: build Dockerfile riêng → push **Artifact Registry**
  (`asia-southeast1`, tag `<svc>:<git-sha>` immutable).
- HTTP deploy với `--use-http2` (gRPC h2c trên `PORT`); worker deploy CPU always-on.
- Secrets tiêm từ GitHub Actions secrets lúc deploy; rotation = redeploy service đó.
- 1 service fail không chặn 2 còn lại; pin version theo SHA (SC-017).

## Quan trắc tối thiểu (FR-057)

- Log JSON ra stdout (Cloud Logging thu thập); metrics/alert Cloud Monitoring cho
  health/latency/error-rate của HTTP + worker; không OTEL trong 020.

## Verify

- [ ] 6 units chạy đúng mode; health `/api/v1/health` pass (SC-018/SC-023).
- [ ] Không consumer/cron trong `http`; không chạy trùng (worker singleton + advisory lock, SC-019).
- [ ] Billing cap + alert hoạt động; tổng conn trong ngân sách (SC-021).
- [ ] Deploy 1 service không ảnh hưởng 2 service còn lại (SC-017).
