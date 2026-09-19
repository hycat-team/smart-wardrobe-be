# 14 — Monitoring VPS + box ngoài (metrics + alert + vận hành)

Không có Cloud Monitoring trong kiến trúc (không Cloud Run). Monitoring dựa trên:
log JSON ra stdout (đọc qua `docker compose logs`), health endpoint, và billing alarm.

## 1. Log

Cả 3 service ghi **JSON ra stdout** trên mọi môi trường (console màu chỉ cho local/dev).
Không OTEL trong 020. Đọc log:

```bash
# VPS
cd /opt/closy
docker compose --env-file .env.production --env-file .env.deploy -f deployments/docker-compose.prod.yml logs -f --tail=100 account
# Box ngoài (GĐ2)
cd /opt/closy-aws
docker compose --env-file .env.aws --env-file .env.deploy -f deployments/docker-compose.aws.yml logs -f --tail=100 commerce-worker
```

Trace `request_id` (Nginx `main_tid` ↔ app log) để nối request qua 2 lớp.

## 2. Health + alert tối thiểu

| Signal | Cách đo | Ngưỡng gợi ý |
|---|---|---|
| Health | `GET /api/v1/<svc>/health` từng service qua cron/Uptime Kuma (cả `api.closy` + `api-v2`) | alert khi down 2 lần liên tiếp |
| Latency | log `duration_ms` / access log Nginx | p95 > 500ms trong 5m |
| Error-rate | đếm 5xx trong log / access log | > 1% trong 5m |
| Worker liveness | `docker compose ps` + outbox backlog trong log | worker down / backlog > 10000 |
| Bill AWS | billing alarm $20/40/60 + Lightsail transfer metrics | vượt ngưỡng |

## 3. Runbook vận hành khi alert

1. **Health down 1 service** → `ps` + log service đó → redeploy service (`deploy-vps-services`
   / `deploy-aws` với `-f services=<svc>`); rollback image cũ bằng digest trước đó nếu do deploy mới.
2. **Error-rate cao 1 service** → xem log → rollback về digest trước nếu do deploy mới.
3. **Worker down / backlog tăng** → kiểm tra worker singleton, advisory lock, rồi restart container.
4. **Latency p95 vượt** → kiểm tra Supabase pooler, AMQP/Redis conn, rồi xem có cần nâng
   plan box không.
5. **Bill AWS vượt** → kiểm tra transfer overage ($0.12/GB), consider tắt box ngoài.

## 4. Verify

- [ ] 3/3 service phát log JSON đọc được qua `docker compose logs`.
- [ ] Health check ngoài (cron/Uptime) hoạt động cho cả 2 hostname.
- [ ] Billing alarm AWS hoạt động.
