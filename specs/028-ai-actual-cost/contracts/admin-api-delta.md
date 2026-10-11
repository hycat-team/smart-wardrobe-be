# Contracts: Chi phí AI thật phục vụ báo cáo lợi nhuận (028)

> Nguyên tắc: không đổi endpoint, không đổi semantics. Chỉ một field cộng thêm.

## 1. Thay đổi duy nhất: `AIEventDrillDown.error_code`

File: `contracts/ai/v1/ai.proto` (message `AIEventDrillDown`, hiện tại field 1–6).

```proto
message AIEventDrillDown {
  string request_id = 1;
  string provider = 2;
  string model = 3;
  int32 status_code = 4;
  string error_message = 5;
  google.protobuf.Timestamp created_at = 6;
  string error_code = 7;  // MỚI: rỗng khi không có; 'pricing_config_missing' | 'usage_unreported' | ...
}
```

Phạm vi regen + mapper (implement phase):

- Regen stub từ `ai.proto` theo quy trình hiện tại của repo.
- `services/ai/contract/aiexecutor/transport/to_proto.go` — map `ErrorCode` sang field 7.
- `services/ai/contract/aiexecutor/transport/mapper.go` — map ngược.
- `services/ai/.../infrastructure/persistence/ai_cost_repo_impl.go::GetAIEventsDrillDown` — đọc thêm cột `error_code` (hiện tại bỏ qua).
- Mọi consumer phía commerce của drill-down (nếu có) — kiểm tra và map tiếp.

Tương thích ngược: field mới là string, client cũ bỏ qua — an toàn lăn bánh
từng service (đúng constitution: deploy độc lập).

## 2. Các API KHÔNG đổi (chỉ giá trị trả về trở nên đúng)

| API | Vì sao không đổi |
|---|---|
| `GET /api/v1/admin/dashboard/ai/usage-by-operation` | Shape giữ nguyên; `totalCostVnd`/`totalPromptTokens` giờ phản ánh số thật (vision > 0; embedding > 0 khi đếm thành công, 0 + `usage_unreported` chỉ khi đếm thất bại). |
| `GET /api/v1/admin/dashboard/kpi-cards`, `overview` | `NetAIMargin` tự đúng vì nguồn `SUM(actual)` đã đúng; không thêm cột snapshot. |
| gRPC `GetAdminAIUsageStats`, `GetAIUsageByOperation`, `GetAIErrorBreakdown` | Không thêm field. Error-breakdown không đổi (chỉ đếm `Released`). |
| gRPC `AnalyzeImage`, `GenerateEmbeddings` | Không đổi signature. |
| Shared `IAIService.AnalyzeImage/GenerateEmbeddings` | Không đổi — hàm `...WithUsage` mới là hàm cộng thêm, caller commerce cũ giữ nguyên. |

## 3. Frontend guide (implement phase phải ship kèm)

Theo constitution, DTO `AIEventDrillDown` đổi (thêm field) → `tasks.md` phải có
task viết `specs/028-ai-actual-cost/frontend-guide.md` liệt kê field mới là
**changed (additive)**, các endpoint bảng trên là **unchanged về shape**.
