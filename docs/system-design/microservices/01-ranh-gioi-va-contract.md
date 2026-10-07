# 01 — Ranh giới service và contract (bảng proto vs in-process)

Nguyên tắc: **chỉ contract bị module ở service KHÁC gọi mới lên protobuf**.
Contract mà mọi caller nằm cùng service giữ in-process.

## Cần protobuf (vượt ranh giới service)

| Contract (interface) | Owner | Service phơi | Caller khác service | gRPC service |
|----------------------|-------|--------------|---------------------|--------------|
| `IUserBriefContract` | identity | Account | Commerce, AI | `IdentityService` |
| `IUserDashboardReader` | identity | Account | Commerce (dashboard) | `IdentityDashboardService` |
| `IUserSubscriptionContract` | subscription | Account | Commerce | `SubscriptionService` |
| `IUserQuotaContract` | subscription | Account | Commerce | `SubscriptionService` |
| `ISubscriptionDashboardReader` | subscription | Account | Commerce (dashboard) | `SubscriptionDashboardService` |
| `IAuditLogContract` | admin | Account | Commerce, AI | `AdminService` |
| `IAIOrchestrator` | aiexecutor | AI | Commerce | `AIOrchestratorService` |
| `IAICostDashboardReader` | aiexecutor | AI | Commerce (dashboard) | `AIDashboardService` |

**Nhận xét**: Commerce **không phơi** gRPC nào; Account không gọi ra ngoài.

> **AI chỉ là caller của hai contract, và cả hai đều không đọc bảng nghiệp vụ**:
> `IUserBriefContract` (đọc user) và `IAuditLogContract` (ghi log kiểm toán). Nó
> **không** phải caller của `IUserSubscriptionContract` hay `IUserQuotaContract` —
> client subscription đã bị gỡ có chủ đích ở 024 T178, lý do ở
> `00-tong-quan-3-service.md`. Đừng sửa cột `IUserSubscriptionContract` thành
> "Commerce, AI" như một sự đính chính hiển nhiên: đó là hành vi đã quyết định, không
> phải tài liệu lỗi thời.

## Giữ in-process (mọi caller cùng service)

`IBrandContract`, `IFashionContract`, `ICategoryContract`, `IWardrobeContract`,
các `*DashboardReader` nội Commerce, `INotificationContract` (Account nội bộ).

## Quy tắc map (Constitution §II)

- Chỉ primitive + UUID + enum; DTO thuần, không entity.
- Cross-service: qua `contracts/<owner>/v1/client` (gRPC) — xem `03-adapter-va-composition.md`.
- Cross-module cùng service: qua `presentation/contract` in-process.
- Import bị cấm tuyệt đối: `domain/entities`, `domain/repositories`,
  `infrastructure/**`, `application/usecase/**` của module thuộc service khác.
- Import được phép: `presentation/contract/`, `application/dto`, `domain/constants/`
  (luôn tham chiếu symbol, không hardcode giá trị).
