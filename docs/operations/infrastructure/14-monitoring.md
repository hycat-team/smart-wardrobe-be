# 14 — Monitoring Cloud Run (metrics + alert + vận hành, FR-057)

## 1. Log

3 service ghi **JSON ra stdout** trên Cloud Run (console màu chỉ cho local/dev);
Cloud Logging thu thập tự động. Không OTEL trong 020.

Verify log:
```bash
gcloud logging read 'resource.type="cloud_run_revision" AND resource.labels.service_name="smart-wardrobe-account"' \
  --limit=5 --format='value(jsonPayload)'
```

## 2. Tạo metrics + alert tối thiểu

Tạo trong **Cloud Monitoring** (cùng đợt deploy T074):

| Signal | Metric | Ngưỡng gợi ý | Kênh |
|---|---|---|---|
| Health | `run.googleapis.com/container/uptime` / startup probe fail | alert khi restart loop | Pager/Email |
| Latency | `request_latencies` p95 | > 500ms trong 5m (gate FR-041) | Pager/Email |
| Error-rate | `request_count` by `response_code` (tỷ lệ 5xx) | > 1% trong 5m | Pager/Email |
| Worker liveness | `container/uptime` của 3 worker + log-based metric outbox backlog | worker down / backlog > 10000 | Pager |

Tạo log-based metric backlog (điều chỉnh `--log-filter` đúng field log thực tế):
```bash
gcloud logging metrics create outbox_pending \
  --description="Outbox pending gauge" \
  --log-filter='resource.type="cloud_run_revision" AND jsonPayload.message="outbox_backlog"'
```
Tạo alert policy (URL có thể lấy từ console sau khi tạo; hoặc dùng UI):
`Monitoring → Alerting → Create policy → chọn metric → Condition` theo bảng trên.
Áp dụng cho **cả HTTP và worker** (6 units). Dashboard gộp 3 service + filter theo
`service_name`/`APP_MODE`.

## 3. Runbook vận hành khi alert

1. **Alert firing** → mở Cloud Logging (filter `service_name`, `severity>=WARNING`) + trace
   `request_id` (Nginx `main_tid` ↔ app log).
2. **Error-rate cao 1 service** → xem revision + log → rollback revision-level (`05` §8)
   nếu do deploy mới.
3. **Worker down / backlog tăng** → kiểm tra worker singleton (`min=max=1`), advisory lock,
   rồi restart worker deployment.
4. **Latency p95 vượt** → kiểm tra Supabase pooler (prepared statement), AMQP/Redis conn,
   rồi xem có cần tăng `max-instances` (trong ngân sách FR-047) không.
5. Ghi mọi incident vào funding-clock checklist nếu ảnh hưởng gate cutover (FR-041).

## 4. Verify (VS6/VS10 + gate)

- [ ] 3/3 service phát log JSON thấy được trên Cloud Logging (SC-031).
- [ ] Dashboard + alert health/latency/error hoạt động cho HTTP và worker.
- [ ] Gate cutover đo được (smoke/error-rate/p95/SSE 7 ngày) lấy số từ dashboard này.
