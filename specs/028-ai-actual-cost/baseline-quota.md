# Baseline quota (T025) — 028-ai-actual-cost

Ngày chụp: 2026-10-11. Mục đích: số liệu gốc để T017 so sánh "giống hệt" (SC-002). Chỉ số liệu, không code.

## Lệnh đã chạy (module Go `smart-wardrobe-be/services/ai`)

```powershell
# Từ thư mục services/ai:
go test -count=1 ./internal/modules/aiexecutor/application/usecase/grant/ ./internal/modules/aiexecutor/application/usecase/cost/
go test -count=1 -v ./internal/modules/aiexecutor/application/usecase/grant/
go test -count=1 -v ./internal/modules/aiexecutor/application/usecase/cost/
```

## Kết quả

| Package | Kết quả | `=== RUN` | `--- PASS` | `--- FAIL` |
|---|---|---|---|---|
| `.../usecase/grant/` | ok (0.856s) | 150 | 96 | 0 |
| `.../usecase/cost/` | ok (0.434s) | 8 | 8 | 0 |

Tổng: 0 FAIL ở cả hai package.

## Ma trận Kịch bản 5 quickstart.md (hiện trạng)

Ba nhóm case quota (trong hạn mức / vượt hạn mức / hết lượt paid) được bao phủ bởi suite `grant/` ở trên (kết quả cho phép/từ chối và số dư đều xanh tại thời điểm baseline). So sánh sau Phase 3 (T017) phải cho kết quả giống hệt bảng này.
