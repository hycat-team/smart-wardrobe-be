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

### D-02 — Trần `limit` của list giao dịch lệch giữa repo (200) và metadata (100)

- **Vấn đề**: `GET /admin/subscriptions/transactions` truyền `limit` thô xuống repo — repo cap 200 —
  trong khi `PaginationQuery.Normalize` (dùng cho `metadata`) cap 100. Gửi `limit=150` trả 150 item
  nhưng `metadata.limit` ghi 100. Lỗi tồn tại trước 029.
- **Vì sao ngoài phạm vi**: FR-010 giữ nguyên hành vi phân trang của list; sửa trần là thay đổi
  hành vi ngoài phạm vi tính năng (và cần rà các caller khác của repo).
- **Gỡ bằng cách nào**: thống nhất một trần duy nhất (khuyến nghị 100 theo `PaginationQuery`) trong
  handler hoặc repo, kèm test; guide 029 đã khuyến nghị FE gửi ≤ 100.
- **Ảnh hưởng tới task đã tick**: không (T019/T022/T023 giữ nguyên semantics phân trang cũ theo FR-010).

### D-03 — Ví dụ envelope cũ (`"success": true`) trong `docs/api/dashboard/dashboard-admin-api.md`

- **Vấn đề**: ví dụ `kpi-cards` trong doc dashboard còn dùng envelope thời monolith
  (`{"success": true, "data": {...}}`); envelope thật là `{message, data}` (`shared/presentation/response.go`).
  Tồn tại trước 029 (xác nhận bằng `git show cd920cfe`).
- **Vì sao ngoài phạm vi**: T042 chỉ cập nhật before→after của các thay đổi thuộc 029; sửa ví dụ cũ
  là rà soát doc tổng thể, không thuộc hợp đồng delta.
- **Gỡ bằng cách nào**: rà `docs/api/dashboard/*.md` thay mọi ví dụ `"success": true` bằng envelope
  thật; cân nhắc thêm bước lint doc.
- **Ảnh hưởng tới task đã tick**: không.

### D-04 — Message 401 là chuỗi mã nội bộ ("4011")

- **Vấn đề**: lỗi `ErrInvalidAccessToken` trả `message: "4011"` — không thân thiện nếu FE hiển thị thô.
  Tồn tại trước 029.
- **Vì sao ngoài phạm vi**: thuộc tầng auth dùng chung mọi endpoint, không thuộc bề mặt 029; sửa
  message là thay đổi client-facing ngoài phạm vi tính năng.
- **Gỡ bằng cách nào**: đổi message sang tiếng Việt thân thiện trong `apperror` auth (kèm rà FE có
  phụ thuộc chuỗi mã không); guide 029 đã khuyến nghị FE không hiển thị thô `message` của 401.
- **Ảnh hưởng tới task đã tick**: không.

### D-05 — 13 lỗi UX dashboard phía FE (danh sách đầy đủ trong `frontend-guide.md` §5)

- **Vấn đề**: các lỗi UX thuộc repo `smart-wardrobe-fe` (mặc định 30 ngày cuốn, thiếu preset,
  nhãn trùng, filter AI events lỗi giá trị mặc định, `console.log` sót, lệch múi giờ hiển thị,
  granularity hardcode, trộn ngôn ngữ, cột "Loại" sai trường, dữ liệu chết KPI...).
- **Vì sao ngoài phạm vi**: repo BE không chứa code FE; 029 chỉ bàn giao đánh giá qua guide (FR-019).
- **Gỡ bằng cách nào**: FE triển khai theo checklist `frontend-guide.md` §7; theo dõi ở backlog
  `smart-wardrobe-fe`. Các fix BE liên quan (kỳ VN, 400, error-rate, payOS value) đã xong trong 029.
- **Ảnh hưởng tới task đã tick**: không (guide §5 phân loại rõ BE-đã-sửa vs FE-cần-làm).

### D-06 — Distribution không tái dựng được lịch sử đổi gói (R7)

- **Vấn đề**: `user_subscriptions` chỉ lưu trạng thái gói hiện tại; phân bổ theo kỳ đếm theo giao
  cửa sổ gói hiện tại với kỳ lọc — user đã đổi gói chỉ được tính theo gói hiện tại, gói cũ trong
  kỳ không xuất hiện.
- **Vì sao ngoài phạm vi**: đã chốt là giới hạn chấp nhận khi clarify (Q1); tái dựng lịch sử cần
  nguồn dữ liệu mới (audit/renewal không đảm bảo đủ) — thiết kế khác, không phải sửa nhỏ.
- **Gỡ bằng cách nào**: khi có yêu cầu chính xác lịch sử phân bổ, thiết kế bảng snapshot phân bổ
  định kỳ hoặc dựng lại từ `subscription_audit` + `deposit_transactions` trong một spec riêng.
- **Ảnh hưởng tới task đã tick**: không — T027/T030/T035 khớp quyết định clarify và có test khóa.

### D-07 — Activity events không có thiết bị/IP

- **Vấn đề**: bảng `user_activity_events` không lưu `deviceName`/`deviceIP`/`userAgent` (những
  trường này nằm ở bảng sessions). Admin điều tra "ai đăng nhập từ đâu" chỉ thấy user + thời điểm.
- **Vì sao ngoài phạm vi**: scope FR-001 chỉ đọc bảng activity-events hiện có; thêm trường thiết bị
  cần migration + thay đổi write-path của session (và quyết định privacy).
- **Gỡ bằng cách nào**: spec riêng bổ sung cột/nguồn dữ liệu thiết bị cho sự kiện login (hoặc join
  sessions theo thời điểm) kèm cân nhắc PII.
- **Ảnh hưởng tới task đã tick**: không.
