# Báo cáo Audit Response DTOs — Module `aiexecutor`

**Branch**: `013-briefres-audit`
**Ngày**: 2026-08-22
**Sub-agent**: audit aiexecutor
**Phạm vi**: `internal/modules/aiexecutor/application/dto/` + `internal/modules/aiexecutor/presentation/handler/`

---

## 1. Tổng quan

Module **aiexecutor** chủ yếu là **service-only**: quản lý orchestration AI, policy cost, quota ledger (cost.go, options.go là các type nội bộ không có JSON tag, không phải response API). Phần **có handler API** chỉ là `AdminHandler` phục vụ admin AI với **5 endpoint** nằm trong router `/api/v1/admin/ai/*`.

Trong 5 endpoint đó chỉ có **4 response struct** thực sự trả ra API:

| Nhóm | Handler | Trạng thái audit |
|---|---|---|
| Pricing (2 endpoint) | `ListPricing`, `UpdatePricing` | Đánh giá over/underdelivery |
| Usage (1 endpoint) | `GetUserUsage` | Đánh giá over/underdelivery |
| Grants (2 endpoint) | `ListGrants`, `CreateGrant` | Đánh giá over/underdelivery |
| Dashboard feed (qua contract `IAICostDashboardReader`) | Dashboard module consume | **Chỉ ghi nhận — không đề xuất refactor** (admin aggregation, đã đồng thuận ở report dashboard) |

**Nhận xét chung**: Toàn bộ response struct đều thuộc **context admin** — theo tiền lệ report dashboard, context quản trị giữ nguyên full fields. Vì vậy **overdelivery gần như không có** dù có 2 struct vượt 8 fields (nhưng mỗi field đều có vai trò hiển thị/quản trị). Vấn đề đáng chú ý duy nhất là **underdelivery**: `AIPolicyGrantDTO` trả `AICostPolicyID` bare UUID mà không kèm thông tin policy (tên/mã/version).

---

## 2. Danh sách struct

### 2.1 Response struct (handler aiexecutor — admin)

| # | Struct | Fields | Ghi chú |
|---|---|---|---|
| 1 | `AIModelConfigDTO` (model_config.go:12) | ID, Provider, Model, InputUSDPerMillion, OutputUSDPerMillion, Currency, USDToVND, Version, EffectiveFrom, EffectiveTo | 10 fields — admin pricing editor |
| 2 | `AIUsageItemDTO` (model_config.go:40) | ID, RequestID, UserID, Operation, Status, PromptTokens, OutputTokens, ThinkingTokens, ActualCostMicroVND, Provider, Model, CreatedAt | 12 fields — admin usage log |
| 3 | `AIUsageListRes` (model_config.go:64) | `PaginationResult[*AIUsageItemDTO]` | wrapper — không audit riêng |
| 4 | `AIPolicyGrantDTO` (model_config.go:55) | ID, UserID, AICostPolicyID, EffectiveFrom, EffectiveTo, CreatedAt | 6 fields — admin grant list |

### 2.2 Response struct qua contract (dashboard module consume — chỉ ghi nhận)

| # | Struct | Fields | Ghi chú |
|---|---|---|---|
| 5 | `AIUsageAdminStatsDTO` (dashboard_dto.go:5) | SnapshotDate, AIRequestsCount, AIErrorCount, AIPaidTokens, AIActualCostMicroVND | admin aggregation |
| 6 | `AIEventDrillDownDTO` (dashboard_dto.go:13) | RequestID, Provider, Model, StatusCode, ErrorMessage, CreatedAt | admin drill-down |
| 7 | `AIUsageByOperationDTO` (dashboard_dto.go:22) | Operation, LogicalRoute, RequestCount, TotalCostMicroVND, TotalPromptTokens, TotalOutputTokens | admin aggregation |
| 8 | `AIErrorBreakdownDTO` (dashboard_dto.go:31) | ErrorCode, Provider, Model, ErrorCount | admin aggregation |

### 2.3 Type nội bộ — KHÔNG phải response API

| # | Type | Vai trò |
|---|---|---|
| 9 | `AITokenEstimationMeta`, `ExecutionPlan`, `AICostDecision`, `AIUsage` (cost.go) | Type orchestration/cost nội bộ, không JSON tag |
| 10 | `OrchestratorOptions` (options.go) | Input options cho orchestrator |
| 11 | `PlanChangedPayload` (grant.go) | Event payload (message bus), không phải API response |
| 12 | `UpdatePricingReq`, `CreateAIGrantReq` (model_config.go) | Request body — không audit |

---

## 3. Overdelivery

### 3.1 `AIModelConfigDTO` (10 fields) — Thấp (biên)

- **Context hiển thị**: Trang admin quản lý bảng giá model AI (list + editor).
- **Phân tích**: Cả 10 field đều có vai trò quản trị: `Provider`, `Model` (nhận diện model), `InputUSDPerMillion`, `OutputUSDPerMillion`, `Currency`, `USDToVND` (giá hiển thị + quy đổi), `Version` (đối chiếu version hiệu lực), `EffectiveFrom`/`EffectiveTo` (thời hạn hiệu lực), `ID` (cần cho `PUT /pricing/:id`).
- **Kết luận**: Vượt ngưỡng 8 fields nhưng thuộc context admin CRUD — **KHÔNG đề xuất bỏ field nào**.

### 3.2 `AIUsageItemDTO` (12 fields) — Thấp (biên)

- **Context hiển thị**: Bảng lịch sử sử dụng AI của một user (admin filter theo `userId`).
- **Phân tích**: Token counts, cost, status, provider/model, operation là dữ liệu lõi của usage log. `ID` + `RequestID` phục vụ trace/drill-down. `UserID` hơi thừa (đã filter theo userId query param) nhưng vô hại trong context admin log.
- **Kết luận**: Context admin — **KHÔNG đề xuất refactor**. Nếu tối giản, chỉ có thể bỏ `UserID` (ưu tiên thấp, không đáng).

### 3.3 `AIPolicyGrantDTO` (6 fields)

- Dưới ngưỡng 8 fields → không overdelivery.

---

## 4. Underdelivery

### 4.1 `AIPolicyGrantDTO.AICostPolicyID` (bare UUID) — Trung bình

- **Struct**: `AIPolicyGrantDTO` (model_config.go:55)
- **Vấn đề**: Admin xem danh sách grant của user cần biết **grant đó gắn chính sách chi phí nào** (tên/mã policy) nhưng response chỉ trả `AICostPolicyID` thuần. Entity `AICostPolicy` (policy.go:14) có `Code`, `Name`, `Version` nhưng không được trả → admin phải gọi thêm API pricing để nhận diện policy từng dòng.
- **Giải pháp**: Thêm nested `AICostPolicy *AICostPolicyBriefRes`.
- **Ghi chú thêm (minor)**: Entity `AIPolicyGrant` (policy.go:46) còn có `Status`, `PlanCode`, `TierRank` nhưng DTO đang **bỏ sót** `Status` — trạng thái hạn mức (active/expired) rất hữu ích cho admin list. Đề xuất cân nhắc thêm `Status` bên cạnh brief policy.

### 4.2 `AIPolicyGrantDTO.UserID` (bare UUID) — Thấp

- Endpoint đã filter theo `userId` query param → user context đã biết, không cần user brief. Không phải vấn đề thực sự.
- `AIUsageItemDTO.UserID` cũng tương tự (đã filter theo userId) → không phải vấn đề.

### 4.3 Các struct còn lại

- `AIModelConfigDTO`: không có FK nào → không underdelivery.
- Dashboard feed structs (`dashboard_dto.go`): admin aggregation → **chỉ ghi nhận, không refactor** (theo tiền lệ report dashboard).

---

## 5. Đề xuất BriefRes

### 5.1 `AICostPolicyBriefRes` *(tạo mới)* — Trung bình

- **Nơi định nghĩa**: Module **aiexecutor** sở hữu entity `AICostPolicy` → đặt tại `internal/modules/aiexecutor/application/dto/`, dùng trong `AIPolicyGrantDTO`.
- **Fields đề xuất** (4):
  ```go
  type AICostPolicyBriefRes struct {
      ID      uuid.UUID `json:"id"`
      Code    string    `json:"code"`
      Name    string    `json:"name"`
      Version int64     `json:"version"`
  }
  ```
- **Cách dùng**: Trong `AIPolicyGrantDTO`, thay `AICostPolicyID uuid.UUID` bằng `AICostPolicy *AICostPolicyBriefRes` (hoặc giữ thêm `AICostPolicyID` tùy nhu cầu, nhưng brief đã chứa ID).
- **Data source**: `AICostPolicy` entity (policy.go:14) có đủ `Code`, `Name`, `Version`; admin UC có thể join/query policy theo `PolicyID` khi build response `ListGrants`.

---

## 6. Ưu tiên

| Ưu tiên | Struct / Endpoint | Lý do |
|---|---|---|
| **Trung bình** | `AIPolicyGrantDTO` — thêm `AICostPolicy *AICostPolicyBriefRes` | Admin không nhận diện được policy (tên/mã) gắn với từng grant nếu chỉ có bare UUID → phải gọi thêm API; nhưng là context admin nên có thể lookup thủ công → không phải blocker. |
| **Thấp** | `AIPolicyGrantDTO` — thêm `Status` | Entity đã có trường này; bổ sung giúp admin lọc grant còn hiệu lực. Không bắt buộc. |
| **Thấp** | `AIUsageItemDTO` — bỏ `UserID` | Thừa nhẹ (đã filter theo userId) nhưng vô hại; context admin. Không đáng refactor. |
| **Không** | `AIModelConfigDTO`, dashboard feed structs | Context admin CRUD/aggregation — giữ nguyên full fields. |

---

## 7. API bị ảnh hưởng

| Endpoint | Handler | Response hiện tại | Response đề xuất |
|---|---|---|---|
| `GET /api/v1/admin/ai/grants` | `ListGrants` | `[]AIPolicyGrantDTO` (ID, UserID, AICostPolicyID, EffectiveFrom, EffectiveTo, CreatedAt) | Thay `AICostPolicyID` bare bằng `AICostPolicy *AICostPolicyBriefRes`; cân nhắc thêm `Status` |
| `POST /api/v1/admin/ai/grants` | `CreateGrant` | `AIPolicyGrantDTO` | Đồng bộ với thay đổi struct (nếu áp dụng) |
| `GET /api/v1/admin/ai/pricing` | `ListPricing` | `[]AIModelConfigDTO` | Không thay đổi (admin CRUD) |
| `PUT /api/v1/admin/ai/pricing/:id` | `UpdatePricing` | `AIModelConfigDTO` | Không thay đổi |
| `GET /api/v1/admin/ai/usage` | `GetUserUsage` | `PaginationResult[AIUsageItemDTO]` | Không thay đổi (admin log) |

### Dashboard feed (chỉ ghi nhận, không refactor)

- `AIUsageAdminStatsDTO`, `AIEventDrillDownDTO`, `AIUsageByOperationDTO`, `AIErrorBreakdownDTO` — qua contract `IAICostDashboardReader`, dashboard module consume cho admin dashboard AI → giữ nguyên, theo tiền lệ report dashboard.

---

## Kết luận

- **Overdelivery**: Gần như không có. Module service-only; toàn bộ response struct thuộc **context admin** (CRUD/aggregation) nên full fields đều có vai trò. `AIModelConfigDTO` (10) và `AIUsageItemDTO` (12) vượt ngưỡng 8 fields nhưng biện minh được → không refactor.
- **Underdelivery**: 1 vấn đề đáng xử lý — `AIPolicyGrantDTO` trả `AICostPolicyID` bare UUID thiếu thông tin policy (tên/mã/version) → cần tạo mới `AICostPolicyBriefRes` (ưu tiên trung bình). Kèm gợi ý bổ sung `Status`.
- **Không có vấn đề nghiêm trọng**: Module chủ yếu là service nội bộ; các type cost.go/options.go/grant.go không phải response API nên không thuộc phạm vi briefres audit.