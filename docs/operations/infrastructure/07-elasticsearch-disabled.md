# 07 — Elasticsearch tắt mặc định

Search engine không được funded nên tắt mặc định (`ELASTICSEARCH_ENABLED=false`) trên
mọi môi trường (FR-001).

## 1. Xác nhận cấu hình tắt

- Default nằm ở `configs/config.yaml` / `configs/config.production.yaml`:
  `elasticsearch.enabled: false`.
- **Không** set biến `ELASTICSEARCH_*` trong GitHub secrets hay `.env.production`.
- Managed workflow tự set `ELASTICSEARCH_ENABLED=false` khi deploy (`00` §5).

## 2. Deploy

Không cần bước riêng: deploy theo `05-compute-cloudrun.md` (managed) hoặc
`deploy-vps-services.yml` (VPS).

## 3. Hành vi khi tắt (US1, SC-001/SC-007)

| Thao tác | Hành vi |
|---|---|
| Tìm kiếm trang phục hệ thống | Tự động fallback **database**, trả kết quả bình thường (FR-002) |
| Thêm/sửa/xoá trang phục | Nghiệp vụ thành công; sync job **no-op**, không sinh event lỗi lặp, không đầy DLQ (FR-003) |
| Health endpoint | Phơi trạng thái `elasticsearch: {enabled, healthy}` |

## 4. Verify (VS1)

- [ ] Tìm kiếm trả 200 + kết quả từ DB khi ES tắt.
- [ ] Thêm/sửa/xoá thành công; `elasticsearch_sync_queue` và `.dlq` **không tăng**:
  ```bash
  # Managed (CloudAMQP) hoặc VPS RabbitMQ management
  curl -sf -u "$RABBITMQ_USER:$RABBITMQ_PASSWORD" \
    "https://$RABBITMQ_HOST/api/queues/<vhost>/elasticsearch_sync_queue" | grep -o '"messages":[0-9]*'
  ```
- [ ] Health trả `elasticsearch.enabled=false`.

## 5. Bật lại (khi cần)

1. Đặt `ELASTICSEARCH_ENABLED=true` + `ELASTICSEARCH_ADDRESSES` (+ user/password nếu có).
2. Redeploy service AI; initial sync chạy, đồng bộ lại **không cần can thiệp tay**.
3. Verify index có dữ liệu và search trả kết quả.
