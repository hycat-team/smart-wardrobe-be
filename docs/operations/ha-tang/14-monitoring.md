# 14 — Monitoring Cloud Run (metrics + alert + runbook vận hành, FR-057)

## Log (đã có từ code — T119)

- 3 service ghi **JSON ra stdout** trên Cloud Run (console màu chỉ cho local/dev);
  Cloud Logging thu thập tự động. Không OTEL trong 020.

## Metrics + alert tối thiểu (tạo trên Cloud Monitoring — ops, cùng đợt deploy T074)

| Signal | Metric | Ngưỡng gợi ý | Kênh |
|--------|--------|--------------|------|
| Health | `run.googleapis.com/container/uptime` / startup probe fail | alert khi restart loop | Pager/Email |
| Latency | `request_latencies` p95 | > 500ms trong 5m (tham chiếu gate FR-041) | Pager/Email |
| Error-rate | `request_count` by `response_code` (5xx ratio) | > 1% trong 5m | Pager/Email |
| Worker liveness | `container/uptime` của 3 worker + outbox backlog (log-based metric `outbox_pending`) | worker down / backlog > 10000 | Pager |

Áp dụng cho **cả HTTP và worker** (6 units). Dashboard gộp 3 service + filter theo
`service_name`/`APP_MODE`.

## Runbook vận hành

1. **Alert firing** → mở Cloud Logging (filter `service_name`, `severity>=WARNING`)
   + trace `request_id` (Nginx `main_tid` ↔ app log).
2. **Error-rate cao 1 service** → xem revision + log → rollback revision-level
   (T096 mức 1) nếu do deploy mới.
3. **Worker down / outbox backlog tăng** → kiểm tra worker singleton (`min=max=1`),
   advisory lock, rồi restart worker deployment.
4. **Latency p95 vượt** → kiểm tra Supabase pooler (prepared statement), AMQP/Redis
   conn, rồi xem có cần tăng `max-instances` (trong ngân sách FR-047) không.
5. Mọi incident ghi vào funding-clock checklist nếu ảnh hưởng gate cutover (FR-041).

## Verify (VS6/VS10 + gate)

- [ ] 3/3 service phát log JSON thấy được trên Cloud Logging (SC-031).
- [ ] Dashboard + alert health/latency/error hoạt động cho HTTP và worker.
- [ ] Gate cutover đo được (smoke/error-rate/p95/SSE 7 ngày) lấy số từ chính dashboard này.
