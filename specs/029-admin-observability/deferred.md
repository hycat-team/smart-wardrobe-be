# Deferred — Khả năng quan sát cho admin (029)

Work discovered while building 029 that is **not this feature's job**.
Nothing here blocks closing 029; nothing here is a task of 029.

## Entries

### D-01 — Index bổ sung cho các truy vấn kỳ dữ liệu (T002)

- **Vấn đề**: Không có index `created_at` đơn lẻ trên `subscription.deposit_transactions`
  (chỉ có partial index theo trạng thái/reconciliation), `subscription.subscription_renewal_attempts`
  (chỉ có unique key), và `aiexecutor.ai_usage_events` chỉ có composite `(user_id, created_at, status)`
  — không dùng được cho truy vấn lọc theo `created_at` đơn thuần. Các truy vấn mới của 029
  (payOS theo kỳ, renewal theo `created_at`, error-breakdown theo kỳ) và các truy vấn cũ
  (`ListAdmin`, revenue-breakdown, worker snapshot 15 phút/lần) đều bounded theo kỳ nên hiện
  chấp nhận được ở quy mô hiện tại, nhưng sẽ chậm dần khi bảng lớn lên.
- **Vì sao ngoài phạm vi**: 029 tuyên bố 0 migration (R12 — toàn bộ thay đổi là read-side);
  thêm index là một migration riêng cần cân nhắc chi phí ghi và kế hoạch migration của account/AI.
  Đây là quyết định hạ tầng vận hành, không phải yêu cầu chức năng của 029.
- **Gỡ bằng cách nào**: Khi `deposit_transactions` / `subscription_renewal_attempts` /
  `ai_usage_events` đủ lớn để truy vấn kỳ chậm (đo bằng `EXPLAIN ANALYZE` trên môi trường thật),
  thêm migration Goose tại service tương ứng: `(created_at)` hoặc composite theo pattern truy vấn
  phổ biến nhất (ví dụ `(status, created_at)` cho deposit). Ghi vào spec vận hành/performance tiếp theo.
- **Ảnh hưởng tới task đã tick**: không — T002 chính là task ghi nhận phát hiện này; các task
  truy vấn (T030, T033, T035) vẫn đúng vì truy vấn bounded theo kỳ.
