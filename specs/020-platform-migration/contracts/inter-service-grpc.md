# Contracts: Inter-Service gRPC

**Feature**: `020-platform-migration` | **Date**: 2026-09-14

Nguồn chân lý triển khai: `contracts/<owner>/v1/*.proto`. Tài liệu này mô tả hợp đồng thiết kế.

## Nguyên tắc chọn contract lên protobuf

Chỉ **contract bị module ở service khác gọi** mới lên protobuf. Contract mà mọi caller nằm cùng
service giữ nguyên in-process (không tạo proto). Danh sách đầy đủ xem
[data-model.md §3](../data-model.md).

**Đồ thị gọi liên service**: `Commerce → Account`, `Commerce → AI`, `AI → Account`,
`Account → (không gọi ra ngoài)`. Commerce **không phơi** gRPC.

Quy tắc chung:
- Chỉ dùng primitive + UUID + enum string; map 1-1 từ `application/dto/*ContractDTO`.
- Không trả entity; không tham chiếu repository/persistence.
- Lỗi map sang `google.rpc.Status`; giữ `x-correlation-id` và (tùy chọn) token trong metadata.
- Thay đổi proto tương thích ngược (chỉ thêm field số mới).
- Auth: verify JWT cục bộ bằng public key, KHÔNG dùng RPC.

---

## 1. IdentityService — `contracts/identity/v1/identity.proto`

Owner: `identity` (Account). Callers: Commerce, AI.

```proto
syntax = "proto3";
package identity.v1;
option go_package = "smart-wardrobe-be/contracts/identity/v1;identityv1";

service IdentityService {
  rpc GetUserBrief (GetUserBriefRequest) returns (UserBrief);
  rpc GetBatchUserBriefs (BatchUserBriefsRequest) returns (BatchUserBriefsResponse);
  rpc GetUsersByUsernamesOrEmails (IdentifiersRequest) returns (BatchUserBriefsResponse);
}

// Dashboard read (consumer: dashboard/Commerce)
service IdentityDashboardService {
  rpc GetAdminUserStats (SnapshotDateRequest) returns (UserAdminStats);
  rpc GetActiveUserIDs (Empty) returns (IDsResponse);
}

message Empty {}
message GetUserBriefRequest { string user_id = 1; }
message BatchUserBriefsRequest { repeated string user_ids = 1; }
message IdentifiersRequest { repeated string identifiers = 1; }
message SnapshotDateRequest { string snapshot_date = 1; } // RFC3339
message IDsResponse { repeated string ids = 1; }

message UserBrief {
  string id = 1;
  string username = 2;
  string email = 3;
  string first_name = 4;
  string last_name = 5;
  string gender = 6;
  string status = 7;
  string avatar_url = 8;
}
message BatchUserBriefsResponse { repeated UserBrief users = 1; }
message UserAdminStats { string snapshot_date = 1; int64 total_users = 2; int64 active_users = 3; }
```

Ánh xạ: `identity/application/dto.UserBriefContractDTO`, `UserAdminStatsDTO`.

---

## 2. SubscriptionService — `contracts/subscription/v1/subscription.proto`

Owner: `subscription` (Account). Callers: Commerce, AI.

```proto
syntax = "proto3";
package subscription.v1;
option go_package = "smart-wardrobe-be/contracts/subscription/v1;subscriptionv1";

service SubscriptionService {
  rpc GetUserSubscriptionOverview (UserRequest) returns (UserSubscription);
  rpc GetUserSubscriptionOverviews (BatchUsersRequest) returns (UserSubscriptionMap);
  rpc GetAndResetDailyQuota (UserRequest) returns (UserQuota);
  rpc UpdateOutfitQuota (UpdateQuotaRequest) returns (Empty);
  rpc UpdateAiChatQuota (UpdateQuotaRequest) returns (Empty);
}

service SubscriptionDashboardService {
  rpc GetAdminSubscriptionStats (SnapshotDateRequest) returns (AdminSubscriptionStats);
  rpc GetSubscriptionDistribution (Empty) returns (SubscriptionDistributionList);
  rpc GetRevenueBreakdown (DateRangeRequest) returns (RevenueBreakdownList);
  rpc GetRenewalStats (Empty) returns (RenewalStats);
}

message Empty {}
message UserRequest { string user_id = 1; }
message BatchUsersRequest { repeated string user_ids = 1; }
message SnapshotDateRequest { string snapshot_date = 1; }
message DateRangeRequest { string from_date = 1; string to_date = 2; }
message UpdateQuotaRequest { string user_id = 1; int32 count = 2; }

message UserSubscription {
  string plan_id = 1;
  string plan_name = 2;
  string plan_slug = 3;
  optional string expires_at = 4;
  bool is_auto_renew_enabled = 5;
  int32 max_wardrobe_items = 6;
  int32 max_outfits = 7;
  int32 ai_outfit_daily_quota = 8;
  int32 ai_chat_daily_quota = 9;
}
message UserSubscriptionMap { map<string, UserSubscription> items = 1; }
message UserQuota { int32 outfit_remaining = 1; int32 chat_remaining = 2; string plan_kind = 3; }

message AdminSubscriptionStats { string snapshot_date = 1; int64 total_subscriptions = 2; double mrr = 3; }
message SubscriptionDistribution { string plan_slug = 1; int64 count = 2; }
message SubscriptionDistributionList { repeated SubscriptionDistribution items = 1; }
message RevenueBreakdown { string date = 1; double amount = 2; }
message RevenueBreakdownList { repeated RevenueBreakdown items = 1; }
message RenewalStats { int64 renewed = 1; int64 canceled = 2; }
```

Ánh xạ: `UserSubscriptionContractDTO`, `UserSubscriptionDTO`, các `*DTO` dashboard tương ứng.

---

## 3. AdminService — `contracts/admin/v1/admin.proto`

Owner: `admin` (Account). Callers: Commerce, AI.

```proto
syntax = "proto3";
package admin.v1;
option go_package = "smart-wardrobe-be/contracts/admin/v1;adminv1";

service AdminService {
  rpc RecordAuditLog (RecordAuditLogRequest) returns (RecordAuditLogResponse);
}

message RecordAuditLogRequest {
  string actor_user_id = 1;
  string action = 2;
  string target_type = 3;
  string target_id = 4;
  map<string, string> metadata = 5;
}
message RecordAuditLogResponse { bool accepted = 1; }
```

Ánh xạ: `admin/application/dto.RecordAuditLogInput`. **Ưu tiên async** qua outbox; RPC chỉ dùng
khi cần đồng bộ.

---

## 4. AIOrchestratorService — `contracts/ai/v1/ai.proto`

Owner: `aiexecutor` (AI). Caller: Commerce (fashion, dashboard).

```proto
syntax = "proto3";
package ai.v1;
option go_package = "smart-wardrobe-be/contracts/ai/v1;aiv1";

service AIOrchestratorService {
  rpc AnalyzeImage (AnalyzeImageRequest) returns (TextResponse);
  rpc GenerateEmbeddings (GenerateEmbeddingsRequest) returns (GenerateEmbeddingsResponse);
  rpc GenerateChatText (GenerateTextRequest) returns (TextResponse);
  rpc GenerateRecommendationText (GenerateTextRequest) returns (TextResponse);
  rpc GenerateChatTextStream (GenerateTextRequest) returns (stream TextChunk);
}

service AIDashboardService {
  rpc GetAdminAIUsageStats (SnapshotDateRequest) returns (AIUsageAdminStats);
  rpc GetAIEventsDrillDown (DrillDownRequest) returns (AIEventDrillDownList);
  rpc GetAIUsageByOperation (DateRangeRequest) returns (AIUsageByOperationList);
  rpc GetTotalAIRequestCount (DateRangeRequest) returns (CountResponse);
  rpc GetAIErrorBreakdown (Empty) returns (AIErrorBreakdownList);
}

message Empty {}
message SnapshotDateRequest { string snapshot_date = 1; }
message DateRangeRequest { string from_date = 1; string to_date = 2; }
message DrillDownRequest { string status = 1; int32 page = 2; int32 limit = 3; }

message OrchestratorOptions {
  int32 max_output_tokens = 1;
  double temperature = 2;
  string response_mime_type = 3;
  string user_id = 4;
  string operation = 5;
}
message AnalyzeImageRequest { string image_url = 1; string prompt = 2; }
message GenerateEmbeddingsRequest { repeated string chunks = 1; }
message GenerateTextRequest {
  string system_prompt = 1;
  string user_prompt = 2;
  OrchestratorOptions options = 3;
}
message TextResponse { string content = 1; }
message TextChunk { string chunk = 1; bool done = 2; }
message GenerateEmbeddingsResponse { repeated Embedding embeddings = 1; }
message Embedding { repeated float values = 1; }

message AIUsageAdminStats { string snapshot_date = 1; int64 total_requests = 2; double total_cost = 3; }
message CountResponse { int64 count = 1; }
message AIEventDrillDown { string id = 1; string status = 2; string operation = 3; }
message AIEventDrillDownList { repeated AIEventDrillDown items = 1; int64 total = 2; }
message AIUsageByOperation { string operation = 1; int64 count = 2; }
message AIUsageByOperationList { repeated AIUsageByOperation items = 1; }
message AIErrorBreakdown { string error_type = 1; int64 count = 2; }
message AIErrorBreakdownList { repeated AIErrorBreakdown items = 1; }
```

Ánh xạ: `aiexecutor/application/dto.OrchestratorOptions`, các `*DTO` dashboard. `GenerateChatTextStream`
map sang SSE ở tầng HTTP của Commerce.

---

## 5. Contract giữ in-process (KHÔNG có proto)

Các contract dưới đây mọi caller đều cùng service → giữ nguyên interface, không tạo proto:

| Contract | Owner | Service |
|----------|-------|---------|
| `IBrandContract` | brand | Commerce |
| `IFashionContract` | fashion | Commerce |
| `ICategoryContract` | fashion | Commerce |
| `IWardrobeContract` | wardrobe | Commerce |
| `IBrandDashboardReader` | brand | Commerce |
| `IFashionDashboardReader` | fashion | Commerce |
| `IWardrobeDashboardReader` | wardrobe | Commerce |
| `INotificationContract` | notification | Account |

Nếu sau này ranh giới service thay đổi làm một contract trong đây vượt service, mới bổ sung proto
theo đúng quy tắc trên.

---

## 6. HTTP API công khai

Không thay đổi đường dẫn/schema (`/api/v1/...`). **VPS Nginx hiện hữu** là ingress: `api.closy...`
→ monolith `http://backend:8080` (giữ nguyên); `api-v2.closy...` → 3 Cloud Run `run.app`. DNS do
Vercel quản lý; không NS-delegate, không Cloudflare.

---

## 7. Kiểm thử hợp đồng

- **Contract behavior test**: mỗi adapter (in-process và gRPC client) cho cùng contract phải pass
  cùng bộ test hành vi (DTO ⇄ proto không lệch). Áp dụng cho 8 contract ở §1–§4.
- **Generate-check**: CI chạy `buf lint` + `buf generate`, fail nếu có diff.
- **Idempotency test**: `UpdateOutfitQuota`/`UpdateAiChatQuota`, `RecordAuditLog`, `GetAndResetDailyQuota`
  (chống trừ/ghi trùng khi retry).

---

## 8. Ingress auth: Cloud Run IAM (hợp đồng vận hành)

- **Managed**: request từ VPS Nginx tới Cloud Run MUST mang **Google-signed ID token** ở header
  `X-Serverless-Authorization: Bearer <token>` (Cloud Run IAM, `roles/run.invoker`; **KHÔNG** bật
  `allUsers`). Cloud Run strip header này trước khi tới container, nên **`Authorization` của ứng
  dụng (Closy JWT) được giữ nguyên**. Audience mặc định = URL `run.app` của service (có thể dùng
  custom audience nếu cần).
- **Token**: do **token helper trên VPS** sinh/cache/refresh (xem data-model §1.8). Healthcheck hạ
  tầng (`/api/v1/health`) vẫn phải qua IAM như mọi request khác (không ngoại lệ ở tầng Cloud Run).
- **Repatriation (VPS)**: khi chạy 3 service compose sau Nginx, không còn Cloud Run IAM; auth tại
  ingress lúc đó do Nginx/network nội bộ quyết định (ghi rõ trong runbook repatriation).
