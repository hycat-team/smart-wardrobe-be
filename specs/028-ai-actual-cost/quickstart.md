# Quickstart: Kiểm chứng chi phí AI thật (028)

> Điều kiện tiên quyết: đã chạy `manual-seed.md` trên DB của môi trường verify
> (đặc biệt giá `gemini-3.5-flash-lite`, nếu không vision sẽ rơi vào nhánh
> `pricing_config_missing`). Sau khi seed, xả Redis pricing cache 30 phút
> (restart service verify hoặc chờ hết TTL) rồi mới kiểm tra — nếu không lượt
> mới vẫn đọc giá cũ trong cache và rơi nhánh thiếu giá oan.

## Kịch bản 1 — Tác vụ trả phí không đổi (15 phút)

1. Dùng tài khoản còn hạn mức gọi chat/outfit (route paid).
2. Gọi `usage-by-operation` với khoảng ngày chứa lượt gọi.
3. Kỳ vọng: `requestCount`, `totalCostVnd`, token khớp công thức
   `tokens × đơn giá × tỉ giá` — giống hệt trước tính năng.

## Kịch bản 2 — Vision ghi chi phí thật (20 phút)

1. Gọi phân tích ảnh qua AI orchestrator (route free).
2. Kiểm tra `usage-by-operation`: dòng vision có token > 0 và chi phí > 0.
3. Đối chiếu tay với giá trong `ai_model_configs` trong vòng 5 phút (SC-003).

## Kịch bản 3 — Embedding được đánh dấu, không bịa số (10 phút)

(spike T001 đã chứng minh `countTokens` 400 với cả hai embedding model nên
không có nhánh đếm thành công — chỉ còn nhánh fail-open.)

1. Gọi tạo embedding qua orchestrator.
2. Kiểm tra drill-down: bản ghi `status=Confirmed` kèm
   `error_code='usage_unreported'`, chi phí 0.
3. Kiểm tra error-breakdown KHÔNG tăng (mã này không phải lỗi).

## Kịch bản 4 — Thiếu giá không chặn user (10 phút)

1. Tạm thời đổi tên model trong seed (môi trường test) để pricing lookup hụt.
2. Gọi chat: user vẫn thành công theo chính sách cũ.
3. Drill-down hiện `error_code='pricing_config_missing'` cho lượt đó.

## Kịch bản 5 — Quota user nguyên vẹn (SC-002, 20 phút)

1. Chạy cùng bộ case hạn mức (trong hạn mức / vượt hạn mức / hết lượt paid)
   trước và sau deploy.
2. Kỳ vọng: kết quả cho phép/từ chối và số dư hạn mức giống hệt nhau.

## Kịch bản 6 — Lợi nhuận ròng đúng chi phí thật (FR-004, 15 phút)

1. Sau khi có lượt paid + free (bao gồm vision/embedding) trong ngày verify,
   mở `GET /api/v1/admin/dashboard/overview` (hoặc `kpi-cards`).
2. Kỳ vọng: `netAiMarginVnd` = `subscriptionRevenueVnd` − tổng `actual`
   (quy đổi micro-VND), trong đó tổng actual đã bao gồm cả phần free;
   đối chiếu tay với `SUM(actual_cost_micro_vnd)` của `ai_usage_events`
   trong ngày.

## Lệnh chạy test scoped

```powershell
#AI service: các package chạm tới (usecase cost/orchestrator, persistence, contract transport)
go test -race -count=3 ./services/ai/...
#Shared AI provider (google vision/embedding parse)
go test -race -count=3 ./shared/infrastructure/ai/...
```

Gate một phase: `make lint` + lệnh trên. Full `make check` trước push/tag.
