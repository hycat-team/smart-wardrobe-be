# 07 — Elasticsearch tắt mặc định

Search engine **không được funded** nên tắt mặc định (`ELASTICSEARCH_ENABLED=false`)
trên mọi môi trường (FR-001).

## Hành vi khi tắt (US1, SC-001/SC-007)

| Thao tác | Hành vi |
|----------|---------|
| Tìm kiếm trang phục hệ thống | Tự động fallback **database**, trả kết quả bình thường (FR-002) |
| Thêm/sửa/xoá trang phục | Nghiệp vụ thành công; sync job **no-op**, không sinh event lỗi lặp, không đầy DLQ (FR-003) |
| Health endpoint | Phơi trạng thái `elasticsearch: {enabled, healthy}` |
| Bật lại (`ELASTICSEARCH_ENABLED=true` + endpoint) | Initial sync chạy, đồng bộ lại không cần can thiệp tay |

## Verify (VS1)

- [ ] Tìm kiếm trả 200 + kết quả từ DB khi ES tắt.
- [ ] Thêm/sửa/xoá thành công; `elasticsearch_sync_queue` và `.dlq` **không tăng**.
- [ ] Bật lại → initial sync chạy, dữ liệu được đánh index.
