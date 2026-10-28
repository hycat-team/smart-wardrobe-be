# Ma trận phân quyền Brand Portal (Brand Permission Matrix)

**Date**: 2026-08-17 | **Feature**: [spec.md](../spec.md) | **Nguồn chuẩn**: code (lấy code làm chuẩn)

Ma trận này là nguồn duy nhất cho middleware Brand RBAC và tài liệu docs (`docs/system-design/brand-permission-matrix.md`). Hai vai trò: **owner** (chủ sở hữu) và **staff** (nhân viên vận hành). Chỉ 2 vai trò này (các vai trò cũ đã gom về staff).

---

## Nguyên tắc

- **Owner-only**: chỉ owner được thao tác (cấu hình chiến lược, đổi hạng thành viên).
- **Member (owner + staff)**: mọi nghiệp vụ vận hành còn lại.
- **Admin hệ thống**: (1) Portal brand vẫn yêu cầu membership (admin không phải member → 403); (2) Brand dashboard được phép bypass membership (xem mọi brand); (3) Admin có endpoint riêng quản lý brand/member/đổi status.

## Nhóm quyền trong middleware

| Nhóm | Roles | Áp dụng |
|------|-------|---------|
| `member` | owner, staff | Toàn bộ route portal trừ owner-only |
| `owner` | owner | Cấu hình loyalty program + loyalty tier |

## Chi tiết theo nhóm nghiệp vụ

| Nhóm nghiệp vụ | Route (prefix `/brand-portal/brands/:brandId`) | owner | staff |
|----------------|------------------------------------------------|:-----:|:-----:|
| **Brand profile** | `GET` (portal detail), `PATCH /profile-images` | ✅ | ✅ |
| **Members** | `GET /members`, `POST /members` (chỉ thêm role staff) | ✅ | ✅ |
| **Customers & claim** | `GET /customers`, `GET /customers/:id`, `POST /customers/:id/claim-token`, `GET /customers/:id/claim-tokens`, `POST /customers/:id/claim-tokens/:claimId/revoke`, `POST /customers/offline-purchase` | ✅ | ✅ |
| **Loyalty vận hành** | `POST /loyalty/points`, `GET /loyalty/accounts/:accountId/transactions`, `GET /loyalty/accounts/:accountId/lots`, `GET /loyalty/program`, `GET /loyalty/tiers` | ✅ | ✅ |
| **Loyalty cấu hình** | `PUT /loyalty/program`, `POST /loyalty/tiers`, `PUT /loyalty/tiers/:tierId` | ✅ | ❌ |
| **Benefits** | `POST /benefits`, `GET /benefits`, `PATCH /benefits/:benefitId/status` | ✅ | ✅ |
| **Conversations (CS)** | `GET /conversations`, `GET /conversations/:id/messages`, `POST /conversations/:id/read`, `POST /conversations/:id/close`, `POST /conversations/:id/reopen`, `POST /conversations/:id/messages` | ✅ | ✅ |
| **Brand items / Sample Lab** | `POST /items`, `GET /items`, `GET /items/upload-signature`, `GET /items/:itemId`, `PUT /items/:itemId`, `PATCH /items/:itemId/status`, `GET /items/:itemId/feedbacks`, `GET /items/tasks/:taskId/sse` | ✅ | ✅ |
| **Brand dashboard** | `/brand/dashboard/*`, `/brand/dashboard/drilldown/*` | ✅ | ✅ |

## Ghi chú ràng buộc

- API thêm member (`POST /members`) **chỉ tạo role staff**, không tạo owner mới qua API này.
- Mỗi brand chỉ có 1 owner active.
- Admin hệ thống KHÔNG được coi là member; portal yêu cầu membership thực sự (admin không phải member → 403).
- Admin được phép xem `/brand/dashboard/*` của mọi brand (bypass membership) — ngoại lệ có chủ đích.

## Việc triển khai

1. Middleware áp dụng nhóm quyền này (xem [brand-role-middleware.md](./brand-role-middleware.md)).
2. Copy ma trận này thành `docs/system-design/brand-permission-matrix.md` (docs deliverable, Phase D3/D5).
3. Các file docs khác (`brand-api.md`, `loyalty-api.md`, `constants/brand.md`) reference về ma trận để tránh mâu thuẫn.
## Bổ sung (2026-10-28)

| Nhóm nghiệp vụ | Route (prefix `/brand-portal/brands/:brandId`) | owner | staff |
|----------------|------------------------------------------------|:-----:|:-----:|
| **Brand items / Sample Lab** | `POST /items/:itemId/convert-to-product` (chuyển sample → product) | ✔ | ✔ |
| **Brand profile** | `PATCH /profile` (tên + mô tả) | ✔ | ✘ |
| **Brand items / Sample Lab** | `GET /items/:itemId` (chi tiết), `POST /items/:itemId/retry-analysis` | ✔ | ✔ |

Ghi chú: sau migration chuẩn hóa status/role, middleware so sánh không phân biệt hoa/thường để tránh 403 với dữ liệu cũ.

| **Members** | `PATCH /members/:memberId` (status active/disabled) | ✔ | ✘ |
| **Members** | `DELETE /members/:memberId` (không áp dụng cho owner) | ✔ | ✘ |

| **Brand registration** | `DELETE /brand-portal/brands/:brandId/request` (hủy request pending/rejected) | ✔ (người tạo request) | ✘ |
