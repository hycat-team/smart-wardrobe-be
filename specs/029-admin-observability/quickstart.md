# Quickstart: Kiểm chứng khả năng quan sát admin (029)

> Điều kiện tiên quyết: stack 3 service chạy local (`make dev`), có tài khoản admin seed và ít
> nhất 2 tài khoản user đã đăng nhập. Token admin đặt vào biến `$TOKEN` (PowerShell). Port và
> đường dẫn gateway theo `docs/development/local-setup.md`. Hợp đồng chi tiết:
> [`contracts/admin-api-delta.md`](./contracts/admin-api-delta.md) và
> [`data-model.md`](./data-model.md) — quickstart chỉ nêu kịch bản + kỳ vọng.

## Kịch bản 1 — Nhật ký hoạt động người dùng (US1, 15 phút)

1. Đăng nhập + refresh phiên cho user A và user B (giữ 1 phiên đã hết hạn để test bộ lọc ngày).
2. `GET /api/v1/admin/activity-events?limit=5` → sự kiện mới nhất trước, kèm `userEmail` +
   `userFirstName`/`userLastName`; `activityType` là `login`/`refresh`.
3. Lọc `userId` của A + `activityType=login` + `fromDate`/`toDate` = hôm nay → chỉ sự kiện khớp đủ.
4. Kỳ vọng lỗi: `activityType=logout` → 400; `userId=abc` → 400; `fromDate=2026-13-01` → 400;
   token user thường → 403.

## Kịch bản 2 — Hợp đồng giao dịch payOS (US2, 20 phút)

1. `GET /api/v1/admin/subscriptions/transactions?limit=5` → mỗi item `status` là **chuỗi** trong
   tập 11 giá trị; không có trường Go viết hoa.
2. Lấy `id` một giao dịch → `GET /transactions/{id}` → cùng bộ trường camelCase như item trong list.
3. Nếu dữ liệu có giao dịch legacy (status số 2): lọc `status=failed_legacy` trả đúng nó; giá trị
   `failed_legacy` xuất ra dùng lại được làm filter (khứ hồi).
4. Biên kỳ VN: tạo/lọc giao dịch có `createdAt` lúc 23:30 giờ VN hôm nay với `toDate` = hôm nay →
   giao dịch vẫn phải xuất hiện (trước đây bị bỏ sót). `fromDate=2026-10-16&toDate=2026-10-15` → 400.
5. Gọi 1 hành động resolve trên giao dịch `reconciliation_required` → lần gọi detail kế tiếp thấy
   trạng thái mới (nghiệp vụ không đổi).

## Kịch bản 3 — Kỳ dữ liệu dashboard nhất quán (US3, 25 phút)

1. Gọi `revenue-breakdown?fromDate=<mùng 1 tháng này>&toDate=<hôm nay>` → chỉ giao dịch trong kỳ
   (so với gọi không tham số: mặc định 6 tháng — hai kết quả phải khác nhau khi có dữ liệu cũ).
2. Gọi `distribution` với cùng kỳ → số theo gói của user có gói hiệu lực giao với kỳ; gọi không
   tham số → hiện trạng (khác nhau nếu có user hết hạn trong kỳ).
3. Gọi `renewal-stats` với cùng kỳ → chỉ attempt tạo trong kỳ; không tham số → toàn thời gian.
4. Gọi `ai/error-breakdown` với cùng kỳ → Σ `breakdown[].errorCount` = `totalErrors`;
   `errorRate` = `totalErrors/totalRequests×100` và ≤ 100; gọi không tham số → mặc định 30 ngày.
5. Mọi endpoint: `fromDate > toDate` → 400 tiếng Việt; ngày sai định dạng → 400.

## Kịch bản 4 — payOS theo kỳ + đúng giá trị (US3/US4, 20 phút)

1. `GET /admin/dashboard/payos-reconciliation` không tham số → số của hôm nay.
2. Gọi với kỳ 30 ngày → số đếm tăng tương ứng dữ liệu 30 ngày.
3. Đối chiếu `totalSuccessValue` với SQL:
   `SELECT COALESCE(SUM(amount) FILTER (WHERE status = 1), 0) FROM deposit_transactions
    WHERE created_at >= :from AND created_at < :to;` → khớp 0 đồng (SC-004).
4. Nạp ví (wallet_topup) thành công trong kỳ → số đếm payOS tăng (trước đây không tính).

## Kịch bản 5 — Không hồi quy (SC-005, 15 phút)

1. `kpi-cards`, `overview`, `ai-margin`, `usage-by-operation`, `drilldown/ai-events` → shape không
   đổi; riêng `overview.payosReconcile` mang ngữ nghĩa mới và `kpi-cards.payOSSuccessCount` đổi
   giá trị theo phạm vi hai loại (đã ghi trong guide).
2. 4 endpoint resolve/retry giao dịch: đường dẫn + hành vi giữ nguyên (phản hồi `data: null`).
3. Gọi các endpoint kỳ không tham số → mặc định như bảng trong `data-model.md` §3.

## Lệnh chạy test scoped (gate mỗi phase)

```powershell
#Account: subscription (billing, admin stats), identity (activity), contract transport
go test -race -count=3 ./services/account/...
#Commerce: dashboard uc/handler
go test -race -count=3 ./services/commerce/...
#AI: cost repo (error breakdown range)
go test -race -count=3 ./services/ai/...
#Shared: helper kỳ VN
go test -race -count=3 ./shared/pkg/utils/validation/... ./shared/pkg/utils/timeutils/...
```

Trước push/tag: `make lint` → `make check` (regen `make buf-gen`, `make swagger-all` nằm trong
quy trình `make generate`/`generate-check`).
